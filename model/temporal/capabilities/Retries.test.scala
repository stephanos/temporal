package framework

import temporal.capabilities.Retries
import scala.annotation.unused

object RetryFixture:
  enum Phase derives Finite:
    case waiting extends Phase, Waiting
    case failed extends Phase, Failed
    case paused extends Phase, Suspended
    case canceled extends Phase, Canceled

  enum Answer derives Finite:
    case accepted

  final case class Snapshot(
      phase: Phase,
      attempts: UpTo[2],
      pause: Boolean,
      cancel: Boolean
  ) derives Finite

  object worker extends Actor
  val failure = action(worker)
  def count(s: Snapshot): Int = s.attempts
  def remaining(s: Snapshot): Boolean = s.attempts < 1
  def pendingPause(s: Snapshot): Boolean = s.pause
  def pendingCancel(s: Snapshot): Boolean = s.cancel
  def one(@unused s: Snapshot): Option[UpTo[2]] = Some(UpTo[2](1))
  def two(@unused s: Snapshot): Option[UpTo[2]] = Some(UpTo[2](2))
  def unlimited(@unused s: Snapshot): Option[UpTo[2]] = None: Option[UpTo[2]]

  object Record extends Machine[Snapshot, Answer, Nothing], Phased[Snapshot, Phase](_.phase):
    val init = Snapshot(Phase.waiting, UpTo[2](0), false, false)
    object rules extends Rules
    val retries = Retries(
      failure = failure,
      retryable = true,
      attemptCount = count,
      maximumAttempts = one,
      retriesRemaining = remaining,
      pendingPause = Some(pendingPause),
      pendingCancel = Some(pendingCancel)
    )

  enum WaitingPhase derives Finite:
    case waiting extends WaitingPhase, Waiting
    case failed extends WaitingPhase, Failed
  final case class WaitingState(phase: WaitingPhase) derives Finite
  def waitingCount(@unused s: WaitingState): Int = 0
  def waitingMaximum(@unused s: WaitingState): Option[UpTo[2]] = None: Option[UpTo[2]]
  def waitingRemaining(@unused s: WaitingState): Boolean = true

  object WaitingRecord
      extends Machine[WaitingState, Answer, Nothing],
        Phased[WaitingState, WaitingPhase](_.phase):
    val init = WaitingState(WaitingPhase.waiting)
    object rules extends Rules
    val retries = Retries(
      failure = failure,
      retryable = true,
      attemptCount = waitingCount,
      maximumAttempts = waitingMaximum,
      retriesRemaining = waitingRemaining
    )
    val returns = Retries.failureReturnsToWaiting(this)(failure, true, waitingRemaining)
    val fails = Retries.failureEndsFailed(this)(failure, true, waitingRemaining)

class CapabilityRetriesSuite extends munit.FunSuite:
  import RetryFixture.*

  private def transition(
      p: Property[Snapshot]
  ): (Snapshot, Step[Snapshot, Answer, Nothing]) => Boolean =
    p.decl.holds2.get.asInstanceOf[
      (Snapshot, Step[Snapshot, Answer, Nothing]) => Boolean
    ] // scalafix:ok DisableSyntax.asInstanceOf

  test("failure settlement fixes its role from before-state eligibility and control priority"):
    import Record.given
    val cases = Seq(
      (true, 0, false, false, Phase.waiting),
      (true, 0, true, false, Phase.paused),
      (true, 0, false, true, Phase.canceled),
      (true, 0, true, true, Phase.canceled),
      (true, 1, false, false, Phase.failed),
      (true, 1, true, false, Phase.failed),
      (true, 1, false, true, Phase.canceled),
      (true, 1, true, true, Phase.canceled),
      (false, 0, false, false, Phase.failed),
      (false, 0, true, true, Phase.failed),
      (false, 1, false, false, Phase.failed),
      (false, 1, true, true, Phase.failed)
    )
    for (retryable, attempts, pause, cancel, expected) <- cases do
      val properties = Seq(
        Retries.failureReturnsToWaiting(Record)(
          failure,
          retryable,
          remaining,
          pendingPause,
          pendingCancel
        ),
        Retries.failureEndsFailed(Record)(failure, retryable, remaining, pendingCancel),
        Retries.failurePauses(Record)(failure, retryable, remaining, pendingPause, pendingCancel),
        Retries.failureCancels(Record)(failure, retryable, pendingCancel)
      )
      assert(properties.forall(_.decl.when.contains(failure)))
      val before = Snapshot(Phase.waiting, UpTo[2](attempts), pause, cancel)
      for target <- summon[Finite[Phase]].values do
        val after = before.copy(phase = target, attempts = UpTo[2](attempts + 1))
        assertEquals(
          properties.map(transition).forall(_(before, Step(Answer.accepted, after))),
          target == expected,
          s"retryable=$retryable before=$before target=$target expected=$expected"
        )

  test("a finite policy maximum is independent of the counter representation ceiling"):
    val policies: Seq[(Snapshot => Option[UpTo[2]], Boolean)] =
      Seq(one -> false, two -> true, unlimited -> true)
    for (maximum, expected) <- policies do
      val p = Retries.attemptCountIsWithinPolicy(Record)(count, maximum)
      val holds = p.decl.holds.get.asInstanceOf[
        Step[Snapshot, Answer, Nothing] => Boolean
      ] // scalafix:ok DisableSyntax.asInstanceOf
      val atCeiling = Record.init.copy(attempts = UpTo[2](2))
      assertEquals(holds(Step(Answer.accepted, atCeiling)), expected)
      assert(holds(Step(Answer.accepted, Record.init.copy(attempts = UpTo[2](1)))))

  test("absent pending controls require neither Pollable nor control roles"):
    assertEquals(WaitingRecord.retries.pendingPause, None)
    assertEquals(WaitingRecord.retries.pendingCancel, None)
    val holds = WaitingRecord.returns.decl.holds2.get.asInstanceOf[
      (WaitingState, Step[WaitingState, Answer, Nothing]) => Boolean
    ] // scalafix:ok DisableSyntax.asInstanceOf
    assert(holds(WaitingRecord.init, Step(Answer.accepted, WaitingRecord.init)))
