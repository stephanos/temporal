package framework

import framework.Finite.given

import temporal.features.activity.{deadline, timers, Failure, Timeout, TimeoutType}
import temporal.features.activity.standalone.{
  activityWorker,
  client,
  maxAttempts,
  protocol,
  startToClose,
  MaxAttempts
}
import temporal.features.activity.standalone.protocol.ActivityProtocol
import framework.outcomes.Outcome

class ActivityRetryRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[protocol.Phase, protocol.Phase]]
  private def take(s: protocol.State, c: Class): Step[protocol.State, Outcome, protocol.Fact] =
    val binding = ActivityProtocol.bindings.find(_.decl == c.decl).get
    effectOf(binding.decl, binding.function)(s, c.values).head

  test("the start declares a retained retry-policy input") {
    assertEquals(client.start.decl.domains.size, 6)
  }

  test("an unlimited start-to-close timeout retries without terminal timeout evidence") {
    val before = ActivityProtocol.init.copy(
      phase = protocol.Phase.started,
      attempts = UpTo(1),
      startToClose = Timeout.expires
    )
    val after = ActivityProtocol.bindings
      .find(_.decl == deadline.startToClose.decl)
      .get
      .function
      .asInstanceOf[protocol.State => List[Step[protocol.State, Outcome, protocol.Fact]]](before)
      .head // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(
      after.state,
      before.copy(phase = protocol.Phase.scheduled, dispatch = protocol.Dispatch.backoff)
    )
    assertEquals(after.facts, List(protocol.Fact.statusScheduled, protocol.Fact.attemptCount))
  }

  test("one, two and unlimited retain their policy and exhaust only finite budgets") {
    for policy <- MaxAttempts.values do
      val initial = take(ActivityProtocol.init, client.start(maxAttempts := policy)).state
      assertEquals(initial.maxAttempts, policy)
      (1 to 3).foldLeft(initial) { (state, attempt) =>
        if state.phase != protocol.Phase.scheduled then state
        else
          val polled = take(state, activityWorker.poll()).state
          assertEquals(polled.attempts: Int, attempt.min(MaxAttempts.bound))
          val retryAllowed =
            policy == MaxAttempts.unlimited || (policy == MaxAttempts.two && attempt < 2)
          assertEquals(ActivityProtocol.states.retriesRemaining(polled), retryAllowed)
          val failed = take(polled, activityWorker.respondFailed(Failure.retryable))
          if policy == MaxAttempts.two then
            val satisfies = ActivityProtocol.retry.properties.retryExhausts.decl.holds.get
              .asInstanceOf[Step[
                protocol.State,
                Outcome,
                protocol.Fact
              ] => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
            assert(satisfies(failed))
            val wrong = if retryAllowed then
              failed.copy(state = failed.state.copy(attempts = UpTo(2)))
            else failed.copy(state = failed.state.copy(attempts = UpTo(1)))
            assert(!satisfies(wrong))
          assertEquals(
            failed.state.phase,
            if retryAllowed then protocol.Phase.scheduled else protocol.Phase.failed
          )
          assertEquals(failed.state.maxAttempts, policy)
          if retryAllowed then
            assertEquals(failed.state.dispatch, protocol.Dispatch.backoff)
            take(failed.state, timers.backoff()).state
          else
            assertEquals(failed.facts, List(protocol.Fact.statusFailed))
            failed.state
      }
  }

  test(
    "timeout settlement preserves pause and cancellation, and never records terminal evidence on retry"
  ) {
    for
      before <- Finite[protocol.State].values
      if List(protocol.Phase.started, protocol.Phase.pauseRequested, protocol.Phase.cancelRequested)
        .contains(before.phase) &&
        before.startToClose == Timeout.expires
    do
      val retryAllowed =
        before.phase != protocol.Phase.cancelRequested &&
          (before.maxAttempts == MaxAttempts.unlimited ||
            (before.maxAttempts == MaxAttempts.two && before.attempts < 2) ||
            (before.maxAttempts == MaxAttempts.one && before.attempts < 1))
      val after = take(before, deadline.startToClose())
      if retryAllowed then
        val expectedPhase = if before.phase == protocol.Phase.pauseRequested then
          protocol.Phase.paused
        else protocol.Phase.scheduled
        val status = if before.phase == protocol.Phase.pauseRequested then
          protocol.Fact.statusPaused
        else protocol.Fact.statusScheduled
        assertEquals(
          after.state,
          before.copy(phase = expectedPhase, dispatch = protocol.Dispatch.backoff)
        )
        assertEquals(after.facts, List(status, protocol.Fact.attemptCount))
      else
        assertEquals(after.state, before.copy(phase = protocol.Phase.timedOut))
        assertEquals(after.facts, List(protocol.Fact.statusTimedOut(TimeoutType.startToClose)))
  }

  test("the timeout retry completes on its second attempt with its full retained state") {
    val path = List(
      client.start(startToClose := Timeout.expires, maxAttempts := MaxAttempts.two),
      activityWorker.poll(),
      deadline.startToClose(),
      timers.backoff(),
      activityWorker.poll(),
      activityWorker.respondCompleted()
    )
    val result = path.foldLeft(ActivityProtocol.init)((s, c) => take(s, c).state)
    assertEquals(
      result,
      ActivityProtocol.states.completedOnRetry
        .copy(startToClose = Timeout.expires, maxAttempts = MaxAttempts.two)
    )
  }
