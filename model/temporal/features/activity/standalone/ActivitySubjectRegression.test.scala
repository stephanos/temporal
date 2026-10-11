package framework

import framework.Finite.given

import temporal.features.activity.{deadline, timers, Failure, Timeout}
import temporal.features.activity.standalone.*
import temporal.features.activity.standalone.system.{
  ActivitySystem,
  ExhaustAfterHeartbeat,
  HeartbeatThenCompletion,
  HeldCancellationByID,
  HeldFailureByID,
  ResetAfterHeartbeat,
  RetryAfterHeartbeat,
  RetryAfterTimeout,
  ScheduledCompletionByID,
  Standalone
}
import temporal.actors.worker.worker as process
import temporal.Bounds.{eight, four, six, three}
import temporal.realize.{inconclusive, satisfied, Realizes}
import framework.outcomes.Outcome
import framework.realize.{Reason, RunExpectation}

// fn-162: the activity Protocol's claims are its facets', each Property declared once, and each
// realization names the Queries it runs (the former claim-container subjects are gone).
class ActivitySubjectRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[system.Phase, system.Phase]]
  private type ActivityStep = Step[system.State, Outcome, system.Fact]
  private type Predicate = (system.State, ActivityStep) => Boolean

  private def predicate(property: Property[system.State]): Predicate =
    property.decl.holds2 match
      case Some(holds) =>
        holds.asInstanceOf[Predicate] // scalafix:ok DisableSyntax.asInstanceOf
      case None =>
        val holds = property.decl.holds.get
          .asInstanceOf[ActivityStep => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
        (_, after) => holds(after)

  // fn-162 R3: no two Properties of the machine with the same `when` agree on every row.
  test("each Property of the activity Protocol is declared once") {
    val p = ActivitySystem
    val properties = List(
      p.completion.properties.completes,
      p.completion.properties.completedByID,
      p.retry.properties.nonRetryableFails,
      p.retry.properties.retryCompletes,
      p.retry.properties.retryExhausts,
      p.retry.properties.fatalFailureByID,
      p.cancellation.properties.cancelRequested,
      p.cancellation.properties.canceledByWorker,
      p.cancellation.properties.cancelIsRequested,
      p.cancellation.properties.canceledByID,
      p.dispatching.properties.dispatchRequiresReady,
      p.dispatching.properties.scheduleToStartRequiresDispatch,
      p.timeouts.properties.scheduleToStartFires,
      p.timeouts.properties.startToCloseFires,
      p.timeouts.properties.scheduleToCloseTimesOut,
      p.timeouts.properties.completesAfterTimeout,
      p.timeouts.properties.failsAfterTimeout,
      p.heartbeat.properties.heartbeatExhausts,
      p.reset.properties.resetOnFatal,
      p.reset.properties.resetOnExhaustion,
      p.reset.properties.resetOnTimeout,
      p.reset.properties.resetKeepsPause,
      p.reset.properties.completionWins,
      p.reset.properties.pauseLeavesResetPending,
      p.reset.properties.resetStaysPending,
      p.reset.properties.resetKeptPaused
    )
    assertEquals(properties.map(_.decl).distinct.size, properties.size)
    for property <- properties do assert(property.decl.machine eq ActivitySystem, property.decl.name)
    val catalog = ActivitySystem.bindings.flatMap { binding =>
      val step = effectOf[system.State, Outcome, system.Fact](binding.decl, binding.function)
      classesOf(binding.decl).map(inputs => inputs -> step)
    }
    val rows = for
      before <- ActivitySystem.fs.values.toVector
      (inputs, step) <- catalog
      after <- step(before, inputs)
    yield before -> after
    for
      (when, group) <- properties.groupBy(_.decl.when)
      pair <- group.combinations(2)
    do
      val (a, b) = (predicate(pair(0)), predicate(pair(1)))
      assert(
        rows.exists((before, after) => a(before, after) != b(before, after)), // scalafix:ok DisableSyntax.!=
        s"${pair(0).decl.name} and ${pair(1).decl.name} state one promise when $when"
      )
  }

  private case class Receipt(
      query: Query,
      owner: Machine[system.State, Outcome, system.Fact],
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
      activityWorker.poll,
      client.requestCancel,
      activityWorker.respondCanceled
    )
    val bothDeadlines = client.start(
      scheduleToClose := Timeout.expires,
      scheduleToStart := Timeout.expires
    )
    val receipts = List(
      Receipt(
        ActivitySystem.completion.queries.completion,
        ActivitySystem,
        ActivitySystem.completion.properties.completes,
        ActivitySystem.completion.queries.completed,
        Vector(client.start(), activityWorker.poll, activityWorker.respondCompleted),
        three,
        16848L,
        Some(satisfied)
      ),
      Receipt(
        ActivitySystem.retry.queries.nonRetryableFailure,
        ActivitySystem,
        ActivitySystem.retry.properties.nonRetryableFails,
        ActivitySystem.retry.queries.nonRetryable,
        Vector(client.start(), activityWorker.poll, activityWorker.respondFailed(Failure.fatal)),
        three,
        16848L,
        Some(satisfied)
      ),
      Receipt(
        ActivitySystem.retry.queries.retry,
        ActivitySystem,
        ActivitySystem.retry.properties.retryCompletes,
        ActivitySystem.retry.queries.retriedThenCompleted,
        Vector(
          client.start(),
          activityWorker.poll,
          activityWorker.respondFailed(Failure.retryable),
          timers.backoff,
          activityWorker.poll,
          activityWorker.respondCompleted
        ),
        six,
        33696L,
        Some(inconclusive(Reason.explanationsDisagree))
      ),
      Receipt(
        ActivitySystem.retry.queries.retryExhaustionByFailures,
        ActivitySystem,
        ActivitySystem.retry.properties.retryExhausts,
        ActivitySystem.retry.queries.exhausted,
        Vector(
          client.start(maxAttempts := MaxAttempts.two),
          activityWorker.poll,
          activityWorker.respondFailed(Failure.retryable),
          timers.backoff,
          activityWorker.poll,
          activityWorker.respondFailed(Failure.retryable)
        ),
        six,
        33696L
      ),
      Receipt(
        ActivitySystem.cancellation.queries.cancel,
        ActivitySystem,
        ActivitySystem.cancellation.properties.canceledByWorker,
        ActivitySystem.cancellation.queries.cancelRequestedThenCanceled,
        canceled,
        four,
        22464L
      ),
      Receipt(
        ActivitySystem.cancellation.queries.cancelRequest,
        ActivitySystem,
        ActivitySystem.cancellation.properties.cancelRequested,
        ActivitySystem.cancellation.queries.cancelRequestedThenCanceled,
        canceled,
        four,
        22464L
      ),
      Receipt(
        ActivitySystem.pausing.queries.pauseResume,
        ActivitySystem,
        ActivitySystem.completion.properties.completes,
        ActivitySystem.pausing.queries.pausedThenCompleted,
        Vector(
          client.start(),
          client.pause,
          client.unpause,
          activityWorker.poll,
          activityWorker.respondCompleted
        ),
        six,
        28080L,
        Some(satisfied)
      ),
      Receipt(
        ActivitySystem.dispatching.queries.startDelayedCompletion,
        ActivitySystem,
        ActivitySystem.completion.properties.completes,
        ActivitySystem.dispatching.queries.delayedThenCompleted,
        Vector(
          client.start(startDelay := Timeout.expires),
          timers.startDelay,
          activityWorker.poll,
          activityWorker.respondCompleted
        ),
        four,
        22464L,
        Some(satisfied)
      ),
      Receipt(
        ActivitySystem.dispatching.queries.delayedAttemptsAreNotDispatched,
        ActivitySystem,
        ActivitySystem.dispatching.properties.dispatchRequiresReady,
        ActivitySystem.queries.any,
        Vector.empty,
        eight,
        5346432L,
        free = true
      ),
      Receipt(
        ActivitySystem.dispatching.queries.scheduleToStartWaitsForDispatch,
        ActivitySystem,
        ActivitySystem.dispatching.properties.scheduleToStartRequiresDispatch,
        ActivitySystem.queries.any,
        Vector.empty,
        eight,
        5346432L,
        free = true
      ),
      Receipt(
        ActivitySystem.timeouts.queries.scheduleToStartTimeout,
        ActivitySystem,
        ActivitySystem.timeouts.properties.scheduleToStartFires,
        ActivitySystem.timeouts.queries.scheduleToStartExpires,
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
        ActivitySystem.timeouts.queries.startToCloseTimeout,
        ActivitySystem,
        ActivitySystem.timeouts.properties.startToCloseFires,
        ActivitySystem.timeouts.queries.startToCloseExpires,
        Vector(
          client.start(startToClose := Timeout.expires, maxAttempts := MaxAttempts.one),
          activityWorker.poll,
          deadline.startToClose
        ),
        three,
        16848L
      ),
      Receipt(
        ActivitySystem.timeouts.queries.competingTimers(0),
        ActivitySystem,
        ActivitySystem.timeouts.properties.scheduleToStartFires,
        ActivitySystem.timeouts.queries.bothDeadlinesStartFirst,
        Vector(bothDeadlines, deadline.scheduleToStart),
        three,
        11232L
      ),
      Receipt(
        ActivitySystem.timeouts.queries.competingTimers(1),
        ActivitySystem,
        ActivitySystem.timeouts.properties.scheduleToCloseTimesOut,
        ActivitySystem.timeouts.queries.bothDeadlinesCloseFirst,
        Vector(bothDeadlines, deadline.scheduleToClose),
        three,
        11232L
      )
    )
    assertEquals(receipts.size, 14)
    assertEquals(ActivitySystem.timeouts.queries.competingTimers.size, 2)
    assertEquals(
      ActivitySystem.timeouts.queries.competingTimers.map(_.name),
      Vector("competingTimers.scheduleToStartFirst", "competingTimers.scheduleToCloseFirst")
    )
    val verifies = Set(
      ActivitySystem.retry.queries.retryExhaustionByFailures,
      ActivitySystem.dispatching.queries.delayedAttemptsAreNotDispatched,
      ActivitySystem.dispatching.queries.scheduleToStartWaitsForDispatch
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

  // fn-162 R8: each realization of the activity Protocol names the find Queries it runs; a find
  // Query none names is model-only (the reset settlement, keep_paused and competing deadlines).
  test("each activity realization names the Queries it runs") {
    val p = ActivitySystem
    val runs: List[(Realizes[?, ?, ?], Seq[AnyRef])] = List(
      Standalone -> Seq(
        p.queries,
        p.capabilities,
        p.completion.queries.completion,
        p.retry.queries.nonRetryableFailure,
        p.retry.queries.retry,
        p.cancellation.queries.cancel,
        p.cancellation.queries.cancelRequest,
        p.pausing.queries.pauseResume,
        p.dispatching.queries.startDelayedCompletion,
        p.timeouts.queries.scheduleToStartTimeout,
        p.timeouts.queries.startToCloseTimeout
      ),
      RetryAfterTimeout -> Seq(p.timeouts.queries.retryAfterTimeout, p.timeouts.queries.retryExhaustion),
      HeartbeatThenCompletion -> Seq(p.heartbeat.queries.heartbeatThenCompletes),
      RetryAfterHeartbeat -> Seq(p.heartbeat.queries.heartbeatTimeoutRetriesThenCompletes),
      ExhaustAfterHeartbeat -> Seq(p.heartbeat.queries.heartbeatTimeoutExhausts),
      ScheduledCompletionByID -> Seq(p.completion.queries.scheduledCompletedByID),
      HeldFailureByID -> Seq(p.retry.queries.heldFailedByID),
      HeldCancellationByID -> Seq(
        p.cancellation.queries.heldCanceledByID,
        p.cancellation.queries.cancelIsRequested
      ),
      ResetAfterHeartbeat -> Seq(p.reset.queries.deferredResetCompletes)
    )
    for (realization, expected) <- runs do
      assert(realization.machine eq ActivitySystem)
      assertEquals(realization.runs.size, expected.size, realization.toString)
      for (named, query) <- realization.runs.zip(expected) do assert(named eq query, realization.toString)
  }
