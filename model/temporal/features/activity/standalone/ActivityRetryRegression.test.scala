package umpire

import temporal.features.activity.{deadline, timers, Failure, Timeout, TimeoutType}
import temporal.features.activity.standalone.{
  client,
  maxAttempts,
  startToClose,
  system,
  worker,
  MaxAttempts
}
import temporal.features.activity.standalone.system.ActivitySystem
import umpire.outcomes.Outcome

class ActivityRetryRegression extends munit.FunSuite:
  private def take(s: system.State, c: Class): Step[system.State, Outcome, system.Fact] =
    val binding = ActivitySystem.bindings.find(_.decl == c.decl).get
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
      .find(_.decl == deadline.startToClose.decl)
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
      var state = take(ActivitySystem.init, client.start(maxAttempts := policy)).state
      assertEquals(state.maxAttempts, policy)
      for attempt <- 1 to 3 if state.phase == system.Phase.scheduled do
        state = take(state, worker.poll()).state
        assertEquals(state.attempts: Int, attempt.min(MaxAttempts.bound))
        val retryAllowed =
          policy == MaxAttempts.unlimited || (policy == MaxAttempts.two && attempt < 2)
        assertEquals(ActivitySystem.states.retriesRemaining(state), retryAllowed)
        val failed = take(state, worker.respondFailed(Failure.retryable))
        if policy == MaxAttempts.two then
          val satisfies = ActivitySystem.properties.retryExhausts.decl.holds.get
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
          state = take(failed.state, timers.backoff()).state
        else
          assertEquals(failed.facts, List(system.Fact.statusFailed))
          state = failed.state
  }

  test(
    "timeout settlement preserves pause and cancellation, and never records terminal evidence on retry"
  ) {
    for
      policy <- MaxAttempts.values
      count <- 1 to MaxAttempts.bound
      phase <- List(system.Phase.started, system.Phase.pauseRequested, system.Phase.cancelRequested)
    do
      val before = ActivitySystem.init.copy(
        phase = phase,
        attempts = UpTo(count),
        startToClose = Timeout.expires,
        maxAttempts = policy
      )
      val retryAllowed = phase != system.Phase.cancelRequested &&
        (policy == MaxAttempts.unlimited || (policy == MaxAttempts.two && count < 2))
      val after = take(before, deadline.startToClose())
      if retryAllowed then
        val expectedPhase = if phase == system.Phase.pauseRequested then system.Phase.paused
        else system.Phase.scheduled
        val status = if phase == system.Phase.pauseRequested then system.Fact.statusPaused
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
      ActivitySystem.properties.completedOnRetry
        .copy(startToClose = Timeout.expires, maxAttempts = MaxAttempts.two)
    )
  }
