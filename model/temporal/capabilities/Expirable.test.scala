package framework

import framework.Finite.given

import temporal.capabilities.Expirable

object ExpirableFixture:
  enum Phase derives Finite:
    case active extends Phase, Held
    case waiting extends Phase, Waiting
    case paused extends Phase, Suspended
    case timedOut extends Phase, TimedOut
    case canceled extends Phase, Canceled

  enum Kind derives Finite:
    case first, second
  enum Fact derives Finite:
    case timeout(kind: Kind)
    case ordinary
  enum Answer derives Finite:
    case accepted
  final case class Snapshot(
      phase: Phase,
      attempts: UpTo[1],
      armed: Boolean,
      ready: Boolean,
      pause: Boolean,
      cancel: Boolean
  ) derives Finite

  val expired = timer
  val timeout = Fact.timeout(Kind.first)
  val otherTimeout = Fact.timeout(Kind.second)
  val timeoutFacts = Seq(timeout, otherTimeout)
  def armed(s: Snapshot): Boolean = s.armed && s.ready
  def remaining(s: Snapshot): Boolean = s.attempts < 1
  def pendingPause(s: Snapshot): Boolean = s.pause
  def pendingCancel(s: Snapshot): Boolean = s.cancel

  object Record extends Machine[Snapshot, Answer, Fact], Phased[Snapshot, Phase](_.phase):
    val init = Snapshot(Phase.active, UpTo[1](0), true, true, false, false)

    object rules extends Rules
    val deadline = Expirable[Snapshot, Phase, Held, ExpirableFixture.Fact](
      expired,
      armed,
      timeout,
      timeoutFacts,
      retryable = true,
      retriesRemaining = Some(remaining),
      pendingPause = Some(pendingPause),
      pendingCancel = Some(pendingCancel)
    )

  enum TerminalPhase derives Finite:
    case active extends TerminalPhase, Held
    case timedOut extends TerminalPhase, TimedOut
  final case class TerminalState(phase: TerminalPhase) derives Finite

  object TerminalRecord
      extends Machine[TerminalState, Answer, Fact],
        Phased[TerminalState, TerminalPhase](_.phase):
    val init = TerminalState(TerminalPhase.active)

    object rules extends Rules
    val deadline = Expirable[TerminalState, TerminalPhase, Held, ExpirableFixture.Fact](
      expired,
      _ => true,
      timeout,
      timeoutFacts
    )

  enum RetryPhase derives Finite:
    case active extends RetryPhase, Held
    case waiting extends RetryPhase, Waiting
    case timedOut extends RetryPhase, TimedOut
  final case class RetryState(phase: RetryPhase) derives Finite

  object RetryRecord
      extends Machine[RetryState, Answer, Fact],
        Phased[RetryState, RetryPhase](_.phase):
    val init = RetryState(RetryPhase.active)

    object rules extends Rules
    def remaining(s: RetryState): Boolean =
      s.phase == RetryPhase.active // scalafix:ok DisableSyntax.==
    val deadline = Expirable[RetryState, RetryPhase, Held, ExpirableFixture.Fact](
      expired,
      _ => true,
      timeout,
      timeoutFacts,
      retryable = true,
      retriesRemaining = Some(remaining)
    )
    val returns = Expirable.deadlineReturnsToWaiting(this)(
      expired,
      true,
      remaining,
      timeoutFacts
    )

class CapabilityExpirableSuite extends munit.FunSuite:
  import ExpirableFixture.*

  private def transition(
      p: Property[Snapshot]
  ): (Snapshot, Step[Snapshot, Answer, Fact]) => Boolean =
    p.decl.holds2.get.asInstanceOf[
      (Snapshot, Step[Snapshot, Answer, Fact]) => Boolean
    ] // scalafix:ok DisableSyntax.asInstanceOf

  test("every firing requires the before-state armed predicate and covered role without a fact"):
    import Record.given
    val property = Expirable.firesInWindow[Snapshot, Phase, Held](Record)(expired, armed)
    assert(property.decl.when.contains(expired))
    val holds = transition(property)
    val states = Finite[Snapshot].values
    assertEquals(states.size, 160)
    for before <- states; after <- states do
      assertEquals(
        holds(before, Step(Answer.accepted, after)),
        before.phase == Phase.active && before.armed && before.ready, // scalafix:ok DisableSyntax.==
        s"before=$before after=$after"
      )

  test("deadline settlement reads before-state eligibility and prioritizes cancel then exhaustion"):
    import Record.given
    val states = Finite[Snapshot].values
    assertEquals(states.size, 160)
    for retryable <- Finite[Boolean].values do
      val properties = Seq(
        Expirable.deadlineTimesOut(Record)(expired, retryable, timeout, remaining, pendingCancel),
        Expirable.deadlineReturnsToWaiting(Record)(
          expired,
          retryable,
          remaining,
          timeoutFacts,
          pendingPause,
          pendingCancel
        ),
        Expirable.deadlinePauses(Record)(
          expired,
          retryable,
          remaining,
          pendingPause,
          timeoutFacts,
          pendingCancel
        )
      )
      assert(properties.forall(_.decl.when.contains(expired)))
      for
        before <- states
        after <- states
        facts <- Nil +: Finite[Fact].values.map(List(_))
      do
        val expected =
          // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
          if !retryable || before.cancel || before.attempts == 1 then
            // scalafix:on DisableSyntax.==
            Phase.timedOut
          else if before.pause then Phase.paused
          else Phase.waiting
        val validFacts =
          if expected == Phase.timedOut then facts.contains(timeout) // scalafix:ok DisableSyntax.==
          else !timeoutFacts.exists(facts.contains)
        assertEquals(
          properties.map(transition).forall(_(before, Step(Answer.accepted, after, facts))),
          after.phase == expected && validFacts, // scalafix:ok DisableSyntax.==
          s"retryable=$retryable before=$before after=$after facts=$facts expected=$expected"
        )

  test("optional retry and pause bindings require only their applicable phase roles"):
    assertEquals(TerminalRecord.deadline.retriesRemaining, None)
    assertEquals(RetryRecord.deadline.pendingPause, None)
    assertEquals(RetryRecord.deadline.pendingCancel, None)
    val holds = RetryRecord.returns.decl.holds2.get.asInstanceOf[
      (RetryState, Step[RetryState, Answer, Fact]) => Boolean
    ] // scalafix:ok DisableSyntax.asInstanceOf
    assert(holds(RetryRecord.init, Step(Answer.accepted, RetryState(RetryPhase.waiting))))
    for before <- Finite[RetryState].values; after <- Finite[RetryState].values do
      assertEquals(
        holds(before, Step(Answer.accepted, after)),
        before.phase != RetryPhase.active || after.phase == RetryPhase.waiting // scalafix:ok DisableSyntax.==, DisableSyntax.!=
      )

  test("constructor rejects incomplete duplicate mixed and unbound timeout families and controls"):
    import Record.given
    val malformed = Seq(
      Seq.empty[Fact],
      Seq(timeout),
      Seq(timeout, timeout, otherTimeout),
      Seq(timeout, otherTimeout, Fact.ordinary),
      Seq(otherTimeout)
    )
    for family <- malformed do
      intercept[IllegalArgumentException]:
        Expirable[Snapshot, Phase, Held, Fact](expired, armed, timeout, family)
    intercept[IllegalArgumentException] {
      Expirable[Snapshot, Phase, Held, Fact](
        expired,
        armed,
        timeout,
        timeoutFacts,
        retryable = true
      )
    }: Unit
    intercept[IllegalArgumentException] {
      Expirable[Snapshot, Phase, Held, Fact](
        expired,
        armed,
        timeout,
        timeoutFacts,
        retriesRemaining = Some(remaining)
      )
    }: Unit
    for (pause, cancel) <- Seq((Some(pendingPause), None), (None, Some(pendingCancel))) do
      intercept[IllegalArgumentException]:
        Expirable[Snapshot, Phase, Held, Fact](
          expired,
          armed,
          timeout,
          timeoutFacts,
          pendingPause = pause,
          pendingCancel = cancel
        )
