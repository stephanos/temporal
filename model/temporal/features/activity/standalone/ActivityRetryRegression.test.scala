package framework

import framework.Finite.given

import temporal.features.activity.{deadline, timers, Failure, Timeout, TimeoutType}
import temporal.features.activity.standalone.{
  activityWorker,
  client,
  completionClient,
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

  test("private Activity paths are witnesses retaining their full-state endings") {
    val p = ActivityProtocol
    for q <- List(
        p.retry.queries.nonRetryableFailure,
        p.retry.queries.retry,
        p.timeouts.queries.startToCloseTimeout,
        p.timeouts.queries.retryAfterTimeout,
        p.timeouts.queries.retryExhaustion
      )
    do assert(q.isInstanceOf[Witness[?, ?, ?]])
  }

  test("all applicable rows preserve the independent original Activity predicates") {
    type ActivityStep = Step[protocol.State, Outcome, protocol.Fact]
    val p = ActivityProtocol
    val completed = protocol.State(
      protocol.Phase.completed,
      protocol.Dispatch.now,
      UpTo(2),
      Timeout.unset,
      Timeout.unset,
      Timeout.unset,
      Timeout.unset,
      MaxAttempts.unlimited
    )
    val timedCompletion =
      completed.copy(startToClose = Timeout.expires, maxAttempts = MaxAttempts.two)
    val claims: List[(Query, ClassRef, protocol.Fact, protocol.State => Boolean, Int, Int)] = List(
      (
        p.completion.queries.scheduledCompletedByID,
        completionClient.respondCompletedByID,
        protocol.Fact.statusCompleted,
        _.phase == protocol.Phase.completed,
        5616,
        0
      ),
      (
        p.retry.queries.nonRetryableFailure,
        activityWorker.respondFailed(Failure.fatal),
        protocol.Fact.statusFailed,
        _.phase == protocol.Phase.failed,
        2160,
        0
      ),
      (
        p.retry.queries.retry,
        activityWorker.respondCompleted,
        protocol.Fact.statusCompleted,
        _ == completed,
        2160,
        2155
      ),
      (
        p.retry.queries.heldFailedByID,
        completionClient.respondFailedByID(Failure.fatal),
        protocol.Fact.statusFailed,
        _.phase == protocol.Phase.failed,
        5616,
        0
      ),
      (
        p.timeouts.queries.startToCloseTimeout,
        deadline.startToClose,
        protocol.Fact.statusTimedOut(TimeoutType.startToClose),
        _.phase == protocol.Phase.timedOut,
        1080,
        0
      ),
      (
        p.timeouts.queries.retryAfterTimeout,
        activityWorker.respondCompleted,
        protocol.Fact.statusCompleted,
        _ == timedCompletion,
        2160,
        2155
      ),
      (
        p.timeouts.queries.retryExhaustion,
        activityWorker.respondFailed(Failure.retryable),
        protocol.Fact.statusFailed,
        _ == timedCompletion.copy(phase = protocol.Phase.failed),
        2160,
        286
      )
    )
    for (q, tail, fact, ending, expectedEnabled, expectedRecordsOnlyDifferences) <- claims do
      assertEquals(q.property.when, Some(tail))
      val generated = q.property.holds.get
        .asInstanceOf[ActivityStep => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
      var applicable = 0
      var enabled = 0
      var recordsOnlyDifferences = 0
      for
        before <- Finite[protocol.State].values
        binding <- p.bindings
        inputs <- classesOf(binding.decl)
        if tail match
          case c: Class     => c.decl == binding.decl && c.values.equals(inputs)
          case a: Action[?] => a.decl == binding.decl
      do
        applicable += 1
        for after <- effectOf[protocol.State, Outcome, protocol.Fact](
            binding.decl,
            binding.function
          )(before, inputs)
        do
          enabled += 1
          val old = ending(after.state) && after.records(fact)
          assertEquals(generated(after), old, s"$before / $inputs / $after")
          if after.records(fact) != old then recordsOnlyDifferences += 1
          if old then
            assert(!generated(after.copy(facts = Nil)))
            if q eq p.retry.queries.retry then
              val wrongStates = List(
                completed.copy(phase = protocol.Phase.failed),
                completed.copy(dispatch = protocol.Dispatch.backoff),
                completed.copy(attempts = UpTo(1)),
                completed.copy(scheduleToClose = Timeout.expires),
                completed.copy(scheduleToStart = Timeout.expires),
                completed.copy(startToClose = Timeout.expires),
                completed.copy(heartbeat = Timeout.expires),
                completed.copy(maxAttempts = MaxAttempts.two)
              )
              for wrong <- wrongStates do assert(!generated(after.copy(state = wrong)))
      assertEquals(applicable, 5616)
      assertEquals(enabled, expectedEnabled)
      assertEquals(applicable - enabled, 5616 - expectedEnabled)
      assertEquals(recordsOnlyDifferences, expectedRecordsOnlyDifferences)
  }

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
            val satisfies = ActivityProtocol.retry.properties.safety.retryExhausts.decl.holds.get
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
