package framework

import framework.Finite.given

import temporal.features.activity.{deadline, timers, Failure, Timeout, TimeoutType}
import temporal.features.activity.standalone.{
  activityWorker,
  client,
  maxAttempts,
  startToClose,
  protocol,
  MaxAttempts
}
import temporal.features.activity.standalone.protocol.ActivityProtocol
import framework.outcomes.Outcome

class ActivityRetryRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[protocol.Phase, protocol.Phase]]
  private def take(s: protocol.State, c: Class): Step[protocol.State, Outcome, protocol.Fact] =
    val binding = ActivityProtocol.bindings.find(_.decl == c.decl).get // scalafix:ok DisableSyntax.==
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
      // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
      .find(_.decl == deadline.startToClose.decl)
      // scalafix:on DisableSyntax.==
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
        if state.phase != protocol.Phase.scheduled then state // scalafix:ok DisableSyntax.!=
        else
          val polled = take(state, activityWorker.poll()).state
          assertEquals(polled.attempts: Int, attempt.min(MaxAttempts.bound))
          val retryAllowed =
            policy == MaxAttempts.unlimited || (policy == MaxAttempts.two && attempt < 2) // scalafix:ok DisableSyntax.==
          assertEquals(ActivityProtocol.states.retriesRemaining(polled), retryAllowed)
          val failed = take(polled, activityWorker.respondFailed(Failure.retryable))
          if policy == MaxAttempts.two then // scalafix:ok DisableSyntax.==
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
        before.startToClose == Timeout.expires // scalafix:ok DisableSyntax.==
    do
      val retryAllowed =
        before.phase != protocol.Phase.cancelRequested && // scalafix:ok DisableSyntax.!=
          (before.maxAttempts == MaxAttempts.unlimited ||
            (before.maxAttempts == MaxAttempts.two && before.attempts < 2) ||
            (before.maxAttempts == MaxAttempts.one && before.attempts < 1)) // scalafix:ok DisableSyntax.==
      val after = take(before, deadline.startToClose())
      if retryAllowed then
        // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
        val expectedPhase = if before.phase == protocol.Phase.pauseRequested then
          // scalafix:on DisableSyntax.==
          protocol.Phase.paused
        else protocol.Phase.scheduled
        // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
        val status = if before.phase == protocol.Phase.pauseRequested then
          // scalafix:on DisableSyntax.==
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
