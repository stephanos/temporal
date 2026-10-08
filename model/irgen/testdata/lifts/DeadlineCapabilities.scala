package fixture.deadlinecapabilities

import umpire.*
import temporal.capabilities.Deadline

enum Phase derives Finite:
  case waiting extends Phase, Waiting
  case held extends Phase, Held
  case suspended extends Phase, Suspended
  case timedOut extends Phase, TimedOut

final case class DeadlineState(
    phase: Phase,
    armed: Boolean,
    dispatchNow: Boolean,
    eligible: Boolean,
    pause: Boolean,
    cancel: Boolean
) derives Finite

enum TimeoutType derives Finite:
  case schedule, start

enum Fact derives Finite:
  case timeout(kind: TimeoutType)
  case unrelated

enum Answer derives Finite:
  case ok

given Ok[Answer] = Ok(Answer.ok)

enum Mode derives Finite:
  case ordinaryHeld, pauseHeld, cancelHeld, exhaustedHeld, pauseCancelHeld
  case exhaustedCancelHeld, exhaustedPauseHeld, unarmedWaiting, backoffWaiting, unarmedHeld
  case otherRole

val tick = input[Boolean]
val mode = input[Mode]
val scheduleExpired = timer.input(tick)
val startExpired = timer
val unboundExpired = timer

object foreign:
  val expired = timer

object observer extends Actor:
  val keep = action(this)

object worker extends Actor:
  val prepare = action(this).input(mode)

val timeoutFacts = Seq(Fact.timeout(TimeoutType.schedule), Fact.timeout(TimeoutType.start))

object Steps:
  def dispatchArmed(s: DeadlineState): Boolean = s.armed && s.dispatchNow
  def armed(s: DeadlineState): Boolean = s.armed
  def remaining(s: DeadlineState): Boolean = s.eligible
  def pendingPause(s: DeadlineState): Boolean = s.pause
  def pendingCancel(s: DeadlineState): Boolean = s.cancel

  def schedule(s: DeadlineState, tick: Boolean): List[Step[DeadlineState, Answer, Fact]] =
    if tick && dispatchArmed(s) && s.phase == Phase.waiting then
      enter(s.copy(phase = Phase.timedOut), Fact.timeout(TimeoutType.schedule))
    else disabled

  def start(s: DeadlineState): List[Step[DeadlineState, Answer, Fact]] =
    if s.armed && s.phase == Phase.held then
      if s.cancel || !s.eligible then
        enter(s.copy(phase = Phase.timedOut), Fact.timeout(TimeoutType.start))
      else
        enter(s.copy(phase = if s.pause then Phase.suspended else Phase.waiting, eligible = false))
    else disabled

  def keep(s: DeadlineState): List[Step[DeadlineState, Answer, Fact]] =
    enter(s, Fact.unrelated)

  def prepare(s: DeadlineState, mode: Mode): List[Step[DeadlineState, Answer, Fact]] =
    mode match
      case Mode.ordinaryHeld =>
        enter(
          s.copy(
            phase = Phase.held,
            armed = true,
            dispatchNow = true,
            eligible = true,
            pause = false,
            cancel = false
          )
        )
      case Mode.pauseHeld =>
        enter(
          s.copy(phase = Phase.held, armed = true, eligible = true, pause = true, cancel = false)
        )
      case Mode.cancelHeld =>
        enter(
          s.copy(phase = Phase.held, armed = true, eligible = true, pause = false, cancel = true)
        )
      case Mode.exhaustedHeld =>
        enter(
          s.copy(phase = Phase.held, armed = true, eligible = false, pause = false, cancel = false)
        )
      case Mode.pauseCancelHeld =>
        enter(
          s.copy(phase = Phase.held, armed = true, eligible = true, pause = true, cancel = true)
        )
      case Mode.exhaustedCancelHeld =>
        enter(
          s.copy(phase = Phase.held, armed = true, eligible = false, pause = false, cancel = true)
        )
      case Mode.exhaustedPauseHeld =>
        enter(
          s.copy(phase = Phase.held, armed = true, eligible = false, pause = true, cancel = false)
        )
      case Mode.unarmedWaiting => enter(s.copy(phase = Phase.waiting, armed = false))
      case Mode.backoffWaiting =>
        enter(s.copy(phase = Phase.waiting, armed = true, dispatchNow = false))
      case Mode.unarmedHeld => enter(s.copy(phase = Phase.held, armed = false))
      case Mode.otherRole   => enter(s.copy(phase = Phase.suspended, armed = true))

val three = Limits(steps = 3, actions = 3, search = 100000)
val two = Limits(steps = 2, actions = 2, search = 100000)

object DeadlineTimers
    extends Machine[DeadlineState, Answer, Fact],
      Phased[DeadlineState, Phase](_.phase):
  val init = DeadlineState(Phase.waiting, true, true, true, false, false)
  val evidence: PartialFunction[Fact, String] = { case Fact.timeout(_) => "timeout" }
  object rules
      extends Bindings(
        scheduleExpired ~> Steps.schedule,
        startExpired ~> Steps.start,
        observer.keep ~> Steps.keep,
        worker.prepare ~> Steps.prepare
      )
  object capabilities extends Capabilities:
    val schedule: Capability = Deadline[DeadlineState, Phase, Waiting, Fact](
      timer = scheduleExpired(true),
      armed = Steps.dispatchArmed,
      timeout = Fact.timeout(TimeoutType.schedule),
      timeoutFacts = timeoutFacts
    )
    val start: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      timer = startExpired,
      armed = Steps.armed,
      timeout = Fact.timeout(TimeoutType.start),
      timeoutFacts = timeoutFacts,
      retryable = true,
      retriesRemaining = Some(Steps.remaining),
      pendingPause = Some(Steps.pendingPause),
      pendingCancel = Some(Steps.pendingCancel)
    )
  object queries:
    capabilities.bound(three, Deadline.firesInWindow[DeadlineState, Phase, Waiting] -> two)

enum SimplePhase derives Finite:
  case waiting extends SimplePhase, Waiting
  case timedOut extends SimplePhase, TimedOut

object SimpleDeadlines
    extends Machine[SimplePhase, Answer, Fact],
      Phased[SimplePhase, SimplePhase](p => p):
  val init = SimplePhase.waiting
  val evidence: PartialFunction[Fact, String] = { case Fact.timeout(_) => "timeout" }
  def armed(s: SimplePhase): Boolean = s == SimplePhase.waiting
  def remaining(s: SimplePhase): Boolean = s == SimplePhase.waiting
  def expire(s: SimplePhase): List[Step[SimplePhase, Answer, Fact]] =
    if armed(s) then enter(SimplePhase.waiting) else disabled
  object rules extends Bindings(startExpired ~> expire)
  object capabilities extends Capabilities:
    val retry: Capability = Deadline[SimplePhase, SimplePhase, Waiting, Fact](
      startExpired,
      armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      retryable = true,
      retriesRemaining = Some(remaining)
    )
  object queries:
    capabilities.bound(three)
