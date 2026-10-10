package framework

import framework.Finite.given

import temporal.features.activity.{deadline, timers, Failure, Timeout, TimeoutType}
import temporal.features.activity.standalone.{
  client,
  maxAttempts,
  startToClose,
  system,
  worker,
  MaxAttempts
}
import temporal.features.activity.standalone.system.{ActivitySystem, RetryFailures}
import framework.outcomes.Outcome

class ActivityRetryRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[system.Phase, system.Phase]]
  private def take(s: system.State, c: Class): Step[system.State, Outcome, system.Fact] =
    val binding = ActivitySystem.bindings.find(_.decl == c.decl).get // scalafix:ok DisableSyntax.==
    effectOf(binding.decl, binding.function)(s, c.values).head

  test("the start declares a retained retry-policy input") {
    assertEquals(client.start.decl.domains.size, 6)
  }

  test("an unlimited start-to-close timeout retries without terminal timeout evidence") {
    val before = ActivitySystem.init.copy(
      phase = system.Phase.started,
      attempts = UpTo(1),
      startToClose = Timeout.expires
    )
    val after = ActivitySystem.bindings
      // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
      .find(_.decl == deadline.startToClose.decl)
      // scalafix:on DisableSyntax.==
      .get
      .function
      .asInstanceOf[system.State => List[Step[system.State, Outcome, system.Fact]]](before)
      .head // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(
      after.state,
      before.copy(phase = system.Phase.scheduled, dispatch = system.Dispatch.backoff)
    )
    assertEquals(after.facts, List(system.Fact.statusScheduled, system.Fact.attemptCount))
  }

  test("one, two and unlimited retain their policy and exhaust only finite budgets") {
    for policy <- MaxAttempts.values do
      val initial = take(ActivitySystem.init, client.start(maxAttempts := policy)).state
      assertEquals(initial.maxAttempts, policy)
      (1 to 3).foldLeft(initial) { (state, attempt) =>
        if state.phase != system.Phase.scheduled then state // scalafix:ok DisableSyntax.!=
        else
          val polled = take(state, worker.poll()).state
          assertEquals(polled.attempts: Int, attempt.min(MaxAttempts.bound))
          val retryAllowed =
            policy == MaxAttempts.unlimited || (policy == MaxAttempts.two && attempt < 2) // scalafix:ok DisableSyntax.==
          assertEquals(ActivitySystem.states.retriesRemaining(polled), retryAllowed)
          val failed = take(polled, worker.respondFailed(Failure.retryable))
          if policy == MaxAttempts.two then // scalafix:ok DisableSyntax.==
            val satisfies = RetryFailures.properties.retryExhausts.decl.holds.get
              .asInstanceOf[Step[
                system.State,
                Outcome,
                system.Fact
              ] => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
            assert(satisfies(failed))
            val wrong = if retryAllowed then
              failed.copy(state = failed.state.copy(attempts = UpTo(2)))
            else failed.copy(state = failed.state.copy(attempts = UpTo(1)))
            assert(!satisfies(wrong))
          assertEquals(
            failed.state.phase,
            if retryAllowed then system.Phase.scheduled else system.Phase.failed
          )
          assertEquals(failed.state.maxAttempts, policy)
          if retryAllowed then
            assertEquals(failed.state.dispatch, system.Dispatch.backoff)
            take(failed.state, timers.backoff()).state
          else
            assertEquals(failed.facts, List(system.Fact.statusFailed))
            failed.state
      }
  }

  test(
    "timeout settlement preserves pause and cancellation, and never records terminal evidence on retry"
  ) {
    for
      before <- Finite[system.State].values
      if List(system.Phase.started, system.Phase.pauseRequested, system.Phase.cancelRequested)
        .contains(before.phase) &&
        before.startToClose == Timeout.expires // scalafix:ok DisableSyntax.==
    do
      val retryAllowed =
        before.phase != system.Phase.cancelRequested && // scalafix:ok DisableSyntax.!=
          (before.maxAttempts == MaxAttempts.unlimited ||
            (before.maxAttempts == MaxAttempts.two && before.attempts < 2) ||
            (before.maxAttempts == MaxAttempts.one && before.attempts < 1)) // scalafix:ok DisableSyntax.==
      val after = take(before, deadline.startToClose())
      if retryAllowed then
        // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
        val expectedPhase = if before.phase == system.Phase.pauseRequested then
          // scalafix:on DisableSyntax.==
          system.Phase.paused
        else system.Phase.scheduled
        // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
        val status = if before.phase == system.Phase.pauseRequested then
          // scalafix:on DisableSyntax.==
          system.Fact.statusPaused
        else system.Fact.statusScheduled
        assertEquals(
          after.state,
          before.copy(phase = expectedPhase, dispatch = system.Dispatch.backoff)
        )
        assertEquals(after.facts, List(status, system.Fact.attemptCount))
      else
        assertEquals(after.state, before.copy(phase = system.Phase.timedOut))
        assertEquals(after.facts, List(system.Fact.statusTimedOut(TimeoutType.startToClose)))
  }

  test("the timeout retry completes on its second attempt with its full retained state") {
    val path = List(
      client.start(startToClose := Timeout.expires, maxAttempts := MaxAttempts.two),
      worker.poll(),
      deadline.startToClose(),
      timers.backoff(),
      worker.poll(),
      worker.respondCompleted()
    )
    val result = path.foldLeft(ActivitySystem.init)((s, c) => take(s, c).state)
    assertEquals(
      result,
      RetryFailures.states.completedOnRetry
        .copy(startToClose = Timeout.expires, maxAttempts = MaxAttempts.two)
    )
  }
