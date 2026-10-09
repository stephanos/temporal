package fixture.retrycapabilities

import framework.*
import temporal.capabilities.{Closable, Retries}

enum SimplePhase derives Finite:
  case waiting extends SimplePhase, Waiting
  case failed extends SimplePhase, Failed

final case class WaitingState(
    phase: SimplePhase,
    attempts: UpTo[2],
    otherAttempts: UpTo[2]
) derives Finite

enum ControlPhase derives Finite:
  case waiting extends ControlPhase, Waiting
  case failed extends ControlPhase, Failed
  case suspended extends ControlPhase, Suspended
  case canceled extends ControlPhase, Canceled

final case class ControlState(
    phase: ControlPhase,
    attempts: UpTo[2],
    pendingPause: Boolean,
    pendingCancel: Boolean
) derives Finite

enum Answer derives Finite:
  case ok, gone

given Ok[Answer] = Ok(Answer.ok)

object worker extends Actor:
  val fail = action(this)
  val fatalFailure = action(this)
  val attempt = action(this)
  val foreignFailure = action(this)

object network extends Actor:
  val fault = action(this)

object client extends Actor:
  val pause = action(this)
  val cancel = action(this)

object observer extends Actor:
  val keep = action(this)

object WaitingSteps:
  def attempts(s: WaitingState): Int = s.attempts
  def otherAttempts(s: WaitingState): Int = s.otherAttempts
  def maxOne(s: WaitingState): Option[UpTo[2]] = Some(UpTo[2](1))
  def unlimited(s: WaitingState): Option[UpTo[2]] = None: Option[UpTo[2]]
  def remainingOne(s: WaitingState): Boolean = s.attempts < 1
  def remainingUnlimited(s: WaitingState): Boolean = true
  def pending(s: WaitingState): Boolean = true
  def fail(s: WaitingState): List[Step[WaitingState, Answer, Nothing]] =
    if s.phase == SimplePhase.waiting then
      enter(
        s.copy(
          phase = if remainingOne(s) then SimplePhase.waiting else SimplePhase.failed,
          attempts = UpTo((s.attempts + 1).min(1))
        )
      )
    else disabled
  def fault(s: WaitingState): List[Step[WaitingState, Answer, Nothing]] =
    if s.phase == SimplePhase.waiting then
      enter(s.copy(otherAttempts = UpTo((s.otherAttempts + 1).min(2))))
    else disabled
  def keep(s: WaitingState): List[Step[WaitingState, Answer, Nothing]] =
    List(Step(if s.phase == SimplePhase.failed then Answer.gone else Answer.ok, s))

object ControlSteps:
  def attempts(s: ControlState): Int = s.attempts
  def maxOne(s: ControlState): Option[UpTo[2]] = Some(UpTo[2](1))
  def remainingOne(s: ControlState): Boolean = s.attempts < 1
  def pendingPause(s: ControlState): Boolean = s.pendingPause
  def pendingCancel(s: ControlState): Boolean = s.pendingCancel
  def fail(s: ControlState): List[Step[ControlState, Answer, Nothing]] =
    if s.phase == ControlPhase.waiting then
      enter(
        s.copy(
          phase =
            if s.pendingCancel then ControlPhase.canceled
            else if !remainingOne(s) then ControlPhase.failed
            else if s.pendingPause then ControlPhase.suspended
            else ControlPhase.waiting,
          attempts = UpTo((s.attempts + 1).min(1))
        )
      )
    else disabled
  def fatal(s: ControlState): List[Step[ControlState, Answer, Nothing]] =
    if s.phase == ControlPhase.waiting then enter(s.copy(phase = ControlPhase.failed))
    else disabled
  def pause(s: ControlState): List[Step[ControlState, Answer, Nothing]] =
    if s.phase == ControlPhase.waiting then enter(s.copy(pendingPause = true)) else disabled
  def cancel(s: ControlState): List[Step[ControlState, Answer, Nothing]] =
    if s.phase == ControlPhase.waiting then enter(s.copy(pendingCancel = true)) else disabled
  def attempt(s: ControlState): List[Step[ControlState, Answer, Nothing]] =
    if s.phase == ControlPhase.waiting then enter(s.copy(attempts = UpTo((s.attempts + 1).min(1))))
    else disabled

val three = Limits(steps = 3, actions = 3, search = 512)
val two = Limits(steps = 2, actions = 2, search = 256)

object WaitingFailures
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  object rules
      extends Bindings(
        worker.fail ~> WaitingSteps.fail,
        network.fault ~> WaitingSteps.fault,
        observer.keep ~> WaitingSteps.keep
      )
  object capabilities extends Capabilities:
    val eligible: Capability = Retries(
      failure = worker.fail,
      retryable = true,
      attemptCount = WaitingSteps.attempts,
      maximumAttempts = WaitingSteps.maxOne,
      retriesRemaining = WaitingSteps.remainingOne
    )
    val network: Capability = Retries(
      failure = fixture.retrycapabilities.network.fault,
      retryable = true,
      attemptCount = WaitingSteps.otherAttempts,
      maximumAttempts = WaitingSteps.unlimited,
      retriesRemaining = WaitingSteps.remainingUnlimited
    )
    val closable: Capability = Closable(rejected = Answer.gone)
  object queries:
    capabilities.bound(three, Retries.failureReturnsToWaiting[WaitingState, SimplePhase] -> two)

object ControlledFailures
    extends Machine[ControlState, Answer, Nothing],
      Phased[ControlState, ControlPhase](_.phase):
  val init = ControlState(ControlPhase.waiting, UpTo(0), false, false)
  object rules
      extends Bindings(
        worker.fail ~> ControlSteps.fail,
        worker.fatalFailure ~> ControlSteps.fatal,
        worker.attempt ~> ControlSteps.attempt,
        client.pause ~> ControlSteps.pause,
        client.cancel ~> ControlSteps.cancel
      )
  object capabilities extends Capabilities:
    val eligible: Capability = Retries(
      failure = worker.fail,
      retryable = true,
      attemptCount = ControlSteps.attempts,
      maximumAttempts = ControlSteps.maxOne,
      retriesRemaining = ControlSteps.remainingOne,
      pendingPause = Some(ControlSteps.pendingPause),
      pendingCancel = Some(ControlSteps.pendingCancel)
    )
    val fatal: Capability = Retries(
      failure = worker.fatalFailure,
      retryable = false,
      attemptCount = ControlSteps.attempts,
      maximumAttempts = ControlSteps.maxOne,
      retriesRemaining = ControlSteps.remainingOne,
      pendingPause = Some(ControlSteps.pendingPause),
      pendingCancel = Some(ControlSteps.pendingCancel)
    )
  object queries:
    capabilities.bound(three, Retries.failureReturnsToWaiting[ControlState, ControlPhase] -> two)

// The fatal instance's own exhaustion law, under the capability Property's parameters.
def fatalEndsFailed(m: Declares[ControlState])(
    failure: ClassRef,
    retryable: Boolean,
    retriesRemaining: ControlState => Boolean,
    pendingCancel: ControlState => Boolean
): Property[ControlState] =
  m.property.when(failure) holdsAcross ((before, after) =>
    retryable || pendingCancel(before) || after.state.phase == ControlPhase.failed
  )

// ControlledFailures with one of its two instances of failureEndsFailed overridden by name.
object SelectedFailures
    extends Machine[ControlState, Answer, Nothing],
      Phased[ControlState, ControlPhase](_.phase):
  val init = ControlState(ControlPhase.waiting, UpTo(0), false, false)
  object rules
      extends Bindings(
        worker.fail ~> ControlSteps.fail,
        worker.fatalFailure ~> ControlSteps.fatal,
        worker.attempt ~> ControlSteps.attempt,
        client.pause ~> ControlSteps.pause,
        client.cancel ~> ControlSteps.cancel
      )
  object capabilities extends Capabilities:
    val eligible: Capability = Retries(
      failure = worker.fail,
      retryable = true,
      attemptCount = ControlSteps.attempts,
      maximumAttempts = ControlSteps.maxOne,
      retriesRemaining = ControlSteps.remainingOne,
      pendingPause = Some(ControlSteps.pendingPause),
      pendingCancel = Some(ControlSteps.pendingCancel)
    )
    val fatal: Capability = Retries(
      failure = worker.fatalFailure,
      retryable = false,
      attemptCount = ControlSteps.attempts,
      maximumAttempts = ControlSteps.maxOne,
      retriesRemaining = ControlSteps.remainingOne,
      pendingPause = Some(ControlSteps.pendingPause),
      pendingCancel = Some(ControlSteps.pendingCancel)
    )
    overriding(
      Retries.failureEndsFailed[ControlState, ControlPhase] -> fatalEndsFailed,
      because = "a fixture's fatal instance states its own law",
      of = Seq(fatal)
    )
  object queries:
    capabilities.bound(three, Retries.failureReturnsToWaiting[ControlState, ControlPhase] -> two)
