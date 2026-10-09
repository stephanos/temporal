package framework

import temporal.features.activity.{deadline, timers, Failure, Timeout, TimeoutType}
import temporal.features.activity.standalone.*
import temporal.features.activity.standalone.system.{
  ActivitySystem,
  Cancellation,
  CancellationExecution,
  CompetingTimeouts,
  Completion,
  CompletionExecution,
  DispatchEligibility,
  DispatchExecution,
  Pausing,
  PausingExecution,
  RetryFailures,
  RetryFailuresExecution,
  Standalone,
  Timeouts,
  TimeoutsExecution
}
import temporal.actors.worker.worker as process
import temporal.Bounds.{four, three}
import temporal.realize.{inconclusive, satisfied, DerivesFrom, Realizes}
import framework.outcomes.Outcome
import framework.realize.{Reason, RunExpectation}

class ActivitySubjectRegression extends munit.FunSuite:
  private type Subject = Derived[system.State, Outcome, system.Fact, system.Phase]
  private type ActivityStep = Step[system.State, Outcome, system.Fact]
  private type Predicate = (system.State, ActivityStep) => Boolean
  private val subjects: List[Subject] = List(
    Completion,
    RetryFailures,
    Cancellation,
    Pausing,
    DispatchEligibility,
    Timeouts,
    CompetingTimeouts
  )

  test("ordinary zero rebind preserves every lifecycle state and ordered transition result") {
    assertEquals(subjects.size, 7)
    for subject <- subjects do
      assertEquals(subject.fs.values, ActivitySystem.fs.values)
      assertEquals(subject.fo.values, ActivitySystem.fo.values)
      assertEquals(subject.ff.values, ActivitySystem.ff.values)
      assertEquals(subject.init, ActivitySystem.init)
      assertEquals(subject.table, ActivitySystem.table)
      val original = ActivitySystem.bindings
      val rebound = subject.bindings
      assertEquals(original.size, 21)
      assertEquals(rebound.map(_.decl), original.map(_.decl))
      val catalog = original.zip(rebound).flatMap { (before, after) =>
        val inputs = classesOf(before.decl)
        assertEquals(classesOf(after.decl), inputs)
        val oldStep = effectOf[system.State, Outcome, system.Fact](before.decl, before.function)
        val newStep = effectOf[system.State, Outcome, system.Fact](after.decl, after.function)
        inputs.map(values => (values, oldStep, newStep))
      }
      assertEquals(catalog.size, 119)
      assertEquals(subject.fs.values.size, 5616)
      val (visited, enabled) = subject.fs.values.foldLeft(0 -> 0) { case (counts, state) =>
        assertEquals(subject.end(state), ActivitySystem.end(state))
        assertEquals(subject.sourcePhasing.phase(state), ActivitySystem.phased.phase(state))
        catalog.foldLeft(counts) { case ((seen, enabled), (inputs, oldStep, newStep)) =>
          val expected = oldStep(state, inputs)
          assertEquals(newStep(state, inputs), expected)
          (seen + 1) -> (enabled + (if expected.nonEmpty then 1 else 0))
        }
      }
      assertEquals(visited, 668304)
      assert(enabled > 0 && enabled < visited)
  }

  private case class Claim(
      owner: Subject,
      property: Property[system.State],
      when: Option[ClassRef],
      reference: Predicate
  )

  private def predicate(property: Property[system.State]): Predicate =
    property.decl.holds2 match
      case Some(holds) =>
        holds.asInstanceOf[Predicate] // scalafix:ok DisableSyntax.asInstanceOf
      case None =>
        val holds = property.decl.holds.get
          .asInstanceOf[ActivityStep => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
        (_, after) => holds(after)

  test("focused subjects independently own the exact complete property truth sets") {
    val completedOnRetry = system.State(
      phase = system.Phase.completed,
      dispatch = system.Dispatch.now,
      attempts = UpTo(MaxAttempts.bound),
      scheduleToClose = Timeout.unset,
      scheduleToStart = Timeout.unset,
      startToClose = Timeout.unset,
      heartbeat = Timeout.unset,
      maxAttempts = MaxAttempts.unlimited
    )
    assertEquals(RetryFailures.states.completedOnRetry, completedOnRetry)
    val completes: Predicate = (_, after) =>
      after.state.phase == system.Phase.completed && after.records(system.Fact.statusCompleted)
    def timesOut(kind: TimeoutType): Predicate = (_, after) =>
      after.state.phase == system.Phase.timedOut && after.records(system.Fact.statusTimedOut(kind))
    val claims = List(
      Claim(Completion, Completion.properties.completes, Some(worker.respondCompleted), completes),
      Claim(Pausing, Pausing.properties.completes, Some(worker.respondCompleted), completes),
      Claim(
        DispatchEligibility,
        DispatchEligibility.properties.completes,
        Some(worker.respondCompleted),
        completes
      ),
      Claim(
        RetryFailures,
        RetryFailures.properties.nonRetryableFails,
        Some(worker.respondFailed(Failure.fatal)),
        (_, after) =>
          after.state.phase == system.Phase.failed && after.records(system.Fact.statusFailed)
      ),
      Claim(
        RetryFailures,
        RetryFailures.properties.retryCompletes,
        Some(worker.respondCompleted),
        (_, after) => after.state == completedOnRetry && after.records(system.Fact.statusCompleted)
      ),
      Claim(
        RetryFailures,
        RetryFailures.properties.retryExhausts,
        Some(worker.respondFailed(Failure.retryable)),
        (_, after) =>
          (after.state == completedOnRetry.copy(
            phase = system.Phase.scheduled,
            dispatch = system.Dispatch.backoff,
            attempts = UpTo(1),
            maxAttempts = MaxAttempts.two
          ) && after.records(system.Fact.statusScheduled) && after.records(
            system.Fact.attemptCount
          )) ||
            (after.state == completedOnRetry.copy(
              phase = system.Phase.failed,
              maxAttempts = MaxAttempts.two
            ) && after.records(system.Fact.statusFailed))
      ),
      Claim(
        Cancellation,
        Cancellation.properties.cancelRequestedWhileStarted,
        Some(client.requestCancel),
        (_, after) =>
          after.state.phase == system.Phase.cancelRequested &&
            after.records(system.Fact.statusCancelRequested)
      ),
      Claim(
        Cancellation,
        Cancellation.properties.canceledByWorker,
        Some(worker.respondCanceled),
        (_, after) =>
          after.state.phase == system.Phase.canceled && after.records(system.Fact.statusCanceled)
      ),
      Claim(
        DispatchEligibility,
        DispatchEligibility.properties.dispatchRequiresReady,
        None,
        (before, after) =>
          !(before.phase == system.Phase.scheduled && after.records(system.Fact.statusStarted)) ||
            before.dispatch == system.Dispatch.now
      ),
      Claim(
        DispatchEligibility,
        DispatchEligibility.properties.scheduleToStartRequiresDispatch,
        None,
        (before, after) =>
          !after.records(system.Fact.statusTimedOut(TimeoutType.scheduleToStart)) ||
            (before.phase == system.Phase.scheduled && before.dispatch == system.Dispatch.now)
      ),
      Claim(
        Timeouts,
        Timeouts.properties.scheduleToStartFires,
        Some(deadline.scheduleToStart),
        timesOut(TimeoutType.scheduleToStart)
      ),
      Claim(
        Timeouts,
        Timeouts.properties.startToCloseFires,
        Some(deadline.startToClose),
        timesOut(TimeoutType.startToClose)
      ),
      Claim(
        CompetingTimeouts,
        CompetingTimeouts.properties.scheduleToStartFires,
        Some(deadline.scheduleToStart),
        timesOut(TimeoutType.scheduleToStart)
      ),
      Claim(
        CompetingTimeouts,
        CompetingTimeouts.properties.scheduleToCloseFires,
        Some(deadline.scheduleToClose),
        timesOut(TimeoutType.scheduleToClose)
      )
    )
    assertEquals(claims.size, 14)
    assertEquals(claims.map(_.property.decl).distinct.size, 14)
    for claim <- claims do
      assert(claim.property.decl.machine eq claim.owner)
      assertEquals(claim.property.decl.when, claim.when)
      assertEquals(claim.property.decl.holds.isDefined, claim.when.isDefined)
      assertEquals(claim.property.decl.holds2.isDefined, claim.when.isEmpty)
    val predicates = claims.map(claim => predicate(claim.property) -> claim.reference)
    val catalog = ActivitySystem.bindings.flatMap { binding =>
      val step = effectOf[system.State, Outcome, system.Fact](binding.decl, binding.function)
      classesOf(binding.decl).map(inputs => inputs -> step)
    }
    val (rows, results) = ActivitySystem.fs.values.foldLeft(0 -> 0) { case (counts, before) =>
      catalog.foldLeft(counts) { case ((rows, results), (inputs, step)) =>
        val after = step(before, inputs)
        for result <- after; (actual, reference) <- predicates do
          assertEquals(actual(before, result), reference(before, result))
        (rows + 1) -> (results + after.size)
      }
    }
    assertEquals(rows, 668304)
    assert(results > 0)
  }

  private case class Receipt(
      query: Query,
      owner: Subject,
      property: Property[system.State],
      scenario: Scenario[system.State],
      actions: Vector[ClassRef],
      limits: Limits,
      total: Long,
      expectedRun: Option[RunExpectation] = None,
      free: Boolean = false
  )

  test("focused query receipts retain exact scenarios bounds and live expectations") {
    val canceled = Vector[ClassRef](
      client.start(),
      worker.poll,
      client.requestCancel,
      worker.respondCanceled
    )
    val bothDeadlines = client.start(
      scheduleToClose := Timeout.expires,
      scheduleToStart := Timeout.expires
    )
    val receipts = List(
      Receipt(
        Completion.queries.completion,
        Completion,
        Completion.properties.completes,
        Completion.queries.completed,
        Vector(client.start(), worker.poll, worker.respondCompleted),
        three,
        16848L,
        Some(satisfied)
      ),
      Receipt(
        RetryFailures.queries.nonRetryableFailure,
        RetryFailures,
        RetryFailures.properties.nonRetryableFails,
        RetryFailures.queries.nonRetryable,
        Vector(client.start(), worker.poll, worker.respondFailed(Failure.fatal)),
        three,
        16848L,
        Some(satisfied)
      ),
      Receipt(
        RetryFailures.queries.retry,
        RetryFailures,
        RetryFailures.properties.retryCompletes,
        RetryFailures.queries.retriedThenCompleted,
        Vector(
          client.start(),
          worker.poll,
          worker.respondFailed(Failure.retryable),
          timers.backoff,
          worker.poll,
          worker.respondCompleted
        ),
        six,
        33696L,
        Some(inconclusive(Reason.explanationsDisagree))
      ),
      Receipt(
        RetryFailures.queries.retryExhaustionByFailures,
        RetryFailures,
        RetryFailures.properties.retryExhausts,
        RetryFailures.queries.exhausted,
        Vector(
          client.start(maxAttempts := MaxAttempts.two),
          worker.poll,
          worker.respondFailed(Failure.retryable),
          timers.backoff,
          worker.poll,
          worker.respondFailed(Failure.retryable)
        ),
        six,
        33696L
      ),
      Receipt(
        Cancellation.queries.cancel,
        Cancellation,
        Cancellation.properties.canceledByWorker,
        Cancellation.queries.cancelRequestedThenCanceled,
        canceled,
        four,
        22464L
      ),
      Receipt(
        Cancellation.queries.cancelRequest,
        Cancellation,
        Cancellation.properties.cancelRequestedWhileStarted,
        Cancellation.queries.cancelRequestedThenCanceled,
        canceled,
        four,
        22464L
      ),
      Receipt(
        Pausing.queries.pauseResume,
        Pausing,
        Pausing.properties.completes,
        Pausing.queries.pausedThenCompleted,
        Vector(client.start(), client.pause, client.unpause, worker.poll, worker.respondCompleted),
        six,
        28080L,
        Some(satisfied)
      ),
      Receipt(
        DispatchEligibility.queries.startDelayedCompletion,
        DispatchEligibility,
        DispatchEligibility.properties.completes,
        DispatchEligibility.queries.delayedThenCompleted,
        Vector(
          client.start(startDelay := Timeout.expires),
          timers.startDelay,
          worker.poll,
          worker.respondCompleted
        ),
        four,
        22464L,
        Some(satisfied)
      ),
      Receipt(
        DispatchEligibility.queries.delayedAttemptsAreNotDispatched,
        DispatchEligibility,
        DispatchEligibility.properties.dispatchRequiresReady,
        DispatchEligibility.queries.any,
        Vector.empty,
        eight,
        5346432L,
        free = true
      ),
      Receipt(
        DispatchEligibility.queries.scheduleToStartWaitsForDispatch,
        DispatchEligibility,
        DispatchEligibility.properties.scheduleToStartRequiresDispatch,
        DispatchEligibility.queries.any,
        Vector.empty,
        eight,
        5346432L,
        free = true
      ),
      Receipt(
        Timeouts.queries.scheduleToStartTimeout,
        Timeouts,
        Timeouts.properties.scheduleToStartFires,
        Timeouts.queries.scheduleToStartExpires,
        Vector(
          client.start(scheduleToStart := Timeout.expires),
          process.stop,
          deadline.scheduleToStart
        ),
        three,
        16848L,
        Some(inconclusive(Reason.neverEvaluated))
      ),
      Receipt(
        Timeouts.queries.startToCloseTimeout,
        Timeouts,
        Timeouts.properties.startToCloseFires,
        Timeouts.queries.startToCloseExpires,
        Vector(
          client.start(startToClose := Timeout.expires, maxAttempts := MaxAttempts.one),
          worker.poll,
          deadline.startToClose
        ),
        three,
        16848L
      ),
      Receipt(
        CompetingTimeouts.queries.competingTimers(0),
        CompetingTimeouts,
        CompetingTimeouts.properties.scheduleToStartFires,
        CompetingTimeouts.queries.bothDeadlinesStartFirst,
        Vector(bothDeadlines, deadline.scheduleToStart),
        three,
        11232L
      ),
      Receipt(
        CompetingTimeouts.queries.competingTimers(1),
        CompetingTimeouts,
        CompetingTimeouts.properties.scheduleToCloseFires,
        CompetingTimeouts.queries.bothDeadlinesCloseFirst,
        Vector(bothDeadlines, deadline.scheduleToClose),
        three,
        11232L
      )
    )
    assertEquals(receipts.size, 14)
    assertEquals(CompetingTimeouts.queries.competingTimers.size, 2)
    assertEquals(
      CompetingTimeouts.queries.competingTimers.map(_.name),
      Vector("competingTimers.scheduleToStartFirst", "competingTimers.scheduleToCloseFirst")
    )
    val verifies = Set(
      RetryFailures.queries.retryExhaustionByFailures,
      DispatchEligibility.queries.delayedAttemptsAreNotDispatched,
      DispatchEligibility.queries.scheduleToStartWaitsForDispatch
    )
    for receipt <- receipts do
      val q = receipt.query
      assertEquals(q.form, if verifies(q) then QueryForm.verify else QueryForm.find)
      assertEquals(q.property, receipt.property.decl)
      assertEquals(q.scenario, receipt.scenario.decl)
      assert(q.property.machine eq receipt.owner)
      assert(q.scenario.machine eq receipt.owner)
      assertEquals(q.scenario.start, None)
      assertEquals(q.scenario.actions, receipt.actions)
      assertEquals(q.scenario.free, receipt.free)
      assertEquals(q.limits, receipt.limits)
      assertEquals(q.expectedRun, receipt.expectedRun)
      assertEquals(q.exploration, None)
      assertEquals(q.total, None)
      val slots = if receipt.free then 119L * q.limits.steps
      else receipt.actions.size.min(q.limits.steps).toLong
      assertEquals(receipt.owner.fs.values.size.toLong * slots, receipt.total)
  }

  test("execution roots attach six subjects and leave competing deadlines model only") {
    val executions: List[(DerivesFrom[system.State, Outcome, system.Fact], Subject)] = List(
      CompletionExecution -> Completion,
      RetryFailuresExecution -> RetryFailures,
      CancellationExecution -> Cancellation,
      PausingExecution -> Pausing,
      DispatchExecution -> DispatchEligibility,
      TimeoutsExecution -> Timeouts
    )
    val standalone = exports.activityStandalone.roots
    val record = exports.activityStandaloneRecord.roots
    assertEquals(executions.size, 6)
    assert(Standalone.machine eq ActivitySystem)
    for (execution, subject) <- executions do
      assert(execution.base eq Standalone)
      assert(execution.machine eq subject)
      assertEquals(standalone.count(_ eq execution), 1)
      assertEquals(record.count(_ eq execution), 0)
    val queries = List(
      Completion.queries,
      RetryFailures.queries,
      Cancellation.queries,
      Pausing.queries,
      DispatchEligibility.queries,
      Timeouts.queries
    )
    for section <- queries do
      assertEquals(standalone.count(_ eq section), 1)
      assertEquals(record.count(_ eq section), 0)
    assertEquals(record.count(_ eq CompetingTimeouts.queries), 1)
    assertEquals(standalone.count(_ eq CompetingTimeouts.queries), 0)
    for root <- standalone ++ record do
      root match
        case execution: Realizes[?, ?, ?] =>
          assert(!(execution.machine eq CompetingTimeouts))
        case _ => ()
  }
