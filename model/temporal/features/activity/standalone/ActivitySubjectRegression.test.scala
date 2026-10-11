package framework

import temporal.features.activity.standalone.realization.*
import framework.Finite.given

import temporal.features.activity.{deadline, timers, Failure, Timeout}
import temporal.features.activity.standalone.*
import temporal.features.activity.standalone.protocol.ActivityProtocol
import temporal.actors.worker.worker as process
import temporal.Bounds.{eight, four, six, three}
import temporal.realize.{inconclusive, satisfied, Realizes}
import framework.outcomes.Outcome
import framework.realize.{Reason, RunExpectation}

// fn-162: the activity Protocol's claims are its facets', each Property declared once, and each
// realization names the Queries it runs (the former claim-container subjects are gone).
class ActivitySubjectRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[protocol.Phase, protocol.Phase]]
  private type ActivityStep = Step[protocol.State, Outcome, protocol.Fact]
  private type Predicate = (protocol.State, ActivityStep) => Boolean

  private def predicate(property: Property[protocol.State]): Predicate =
    property.decl.holds2 match
      case Some(holds) =>
        holds.asInstanceOf[Predicate] // scalafix:ok DisableSyntax.asInstanceOf
      case None =>
        val holds = property.decl.holds.get
          .asInstanceOf[ActivityStep => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
        (_, after) => holds(after)

  // fn-162 R3: no two Properties of the machine with the same `when` agree on every row.
  test("each Property of the activity Protocol is declared once") {
    val p = ActivityProtocol
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
    for property <- properties do
      assert(property.decl.machine eq ActivityProtocol, property.decl.name)
    val catalog = ActivityProtocol.bindings.flatMap { binding =>
      val step = effectOf[protocol.State, Outcome, protocol.Fact](binding.decl, binding.function)
      classesOf(binding.decl).map(inputs => inputs -> step)
    }
    val rows = for
      before <- ActivityProtocol.fs.values.toVector
      (inputs, step) <- catalog
      after <- step(before, inputs)
    yield before -> after
    for
      (when, group) <- properties.groupBy(_.decl.when)
      pair <- group.combinations(2)
    do
      val (a, b) = (predicate(pair(0)), predicate(pair(1)))
      assert(
        rows.exists((before, after) => a(before, after) != b(before, after)),
        s"${pair(0).decl.name} and ${pair(1).decl.name} state one promise when $when"
      )
  }

  private case class Receipt(
      query: Query,
      owner: Machine[protocol.State, Outcome, protocol.Fact],
      property: Property[protocol.State],
      scenario: Scenario[protocol.State],
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
        ActivityProtocol.completion.queries.completion,
        ActivityProtocol,
        ActivityProtocol.completion.properties.completes,
        ActivityProtocol.completion.queries.completed,
        Vector(client.start(), activityWorker.poll, activityWorker.respondCompleted),
        three,
        16848L,
        Some(satisfied)
      ),
      Receipt(
        ActivityProtocol.retry.queries.nonRetryableFailure,
        ActivityProtocol,
        ActivityProtocol.retry.properties.nonRetryableFails,
        ActivityProtocol.retry.queries.nonRetryable,
        Vector(client.start(), activityWorker.poll, activityWorker.respondFailed(Failure.fatal)),
        three,
        16848L,
        Some(satisfied)
      ),
      Receipt(
        ActivityProtocol.retry.queries.retry,
        ActivityProtocol,
        ActivityProtocol.retry.properties.retryCompletes,
        ActivityProtocol.retry.queries.retriedThenCompleted,
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
        Some(inconclusive(Reason.explanationsDisagree, because = ""))
      ),
      Receipt(
        ActivityProtocol.retry.queries.retryExhaustionByFailures,
        ActivityProtocol,
        ActivityProtocol.retry.properties.retryExhausts,
        ActivityProtocol.retry.queries.exhausted,
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
        ActivityProtocol.cancellation.queries.cancel,
        ActivityProtocol,
        ActivityProtocol.cancellation.properties.canceledByWorker,
        ActivityProtocol.cancellation.queries.cancelRequestedThenCanceled,
        canceled,
        four,
        22464L
      ),
      Receipt(
        ActivityProtocol.cancellation.queries.cancelRequest,
        ActivityProtocol,
        ActivityProtocol.cancellation.properties.cancelRequested,
        ActivityProtocol.cancellation.queries.cancelRequestedThenCanceled,
        canceled,
        four,
        22464L
      ),
      Receipt(
        ActivityProtocol.pausing.queries.pauseResume,
        ActivityProtocol,
        ActivityProtocol.completion.properties.completes,
        ActivityProtocol.pausing.queries.pausedThenCompleted,
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
        ActivityProtocol.dispatching.queries.startDelayedCompletion,
        ActivityProtocol,
        ActivityProtocol.completion.properties.completes,
        ActivityProtocol.dispatching.queries.delayedThenCompleted,
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
        ActivityProtocol.dispatching.queries.delayedAttemptsAreNotDispatched,
        ActivityProtocol,
        ActivityProtocol.dispatching.properties.dispatchRequiresReady,
        ActivityProtocol.queries.any,
        Vector.empty,
        eight,
        5346432L,
        free = true
      ),
      Receipt(
        ActivityProtocol.dispatching.queries.scheduleToStartWaitsForDispatch,
        ActivityProtocol,
        ActivityProtocol.dispatching.properties.scheduleToStartRequiresDispatch,
        ActivityProtocol.queries.any,
        Vector.empty,
        eight,
        5346432L,
        free = true
      ),
      Receipt(
        ActivityProtocol.timeouts.queries.scheduleToStartTimeout,
        ActivityProtocol,
        ActivityProtocol.timeouts.properties.scheduleToStartFires,
        ActivityProtocol.timeouts.queries.scheduleToStartExpires,
        Vector(
          client.start(scheduleToStart := Timeout.expires),
          process.stop,
          deadline.scheduleToStart
        ),
        three,
        16848L,
        Some(inconclusive(Reason.neverEvaluated, because = ""))
      ),
      Receipt(
        ActivityProtocol.timeouts.queries.startToCloseTimeout,
        ActivityProtocol,
        ActivityProtocol.timeouts.properties.startToCloseFires,
        ActivityProtocol.timeouts.queries.startToCloseExpires,
        Vector(
          client.start(startToClose := Timeout.expires, maxAttempts := MaxAttempts.one),
          activityWorker.poll,
          deadline.startToClose
        ),
        three,
        16848L
      ),
      Receipt(
        ActivityProtocol.timeouts.queries.competingTimers(0),
        ActivityProtocol,
        ActivityProtocol.timeouts.properties.scheduleToStartFires,
        ActivityProtocol.timeouts.queries.bothDeadlinesStartFirst,
        Vector(bothDeadlines, deadline.scheduleToStart),
        three,
        11232L
      ),
      Receipt(
        ActivityProtocol.timeouts.queries.competingTimers(1),
        ActivityProtocol,
        ActivityProtocol.timeouts.properties.scheduleToCloseTimesOut,
        ActivityProtocol.timeouts.queries.bothDeadlinesCloseFirst,
        Vector(bothDeadlines, deadline.scheduleToClose),
        three,
        11232L
      )
    )
    assertEquals(receipts.size, 14)
    assertEquals(ActivityProtocol.timeouts.queries.competingTimers.size, 2)
    assertEquals(
      ActivityProtocol.timeouts.queries.competingTimers.map(_.name),
      Vector("competingTimers.scheduleToStartFirst", "competingTimers.scheduleToCloseFirst")
    )
    val verifies = Set(
      ActivityProtocol.retry.queries.retryExhaustionByFailures,
      ActivityProtocol.dispatching.queries.delayedAttemptsAreNotDispatched,
      ActivityProtocol.dispatching.queries.scheduleToStartWaitsForDispatch
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
      // The reason is source text (fn-140 R4); the expectation is compared without it.
      assertEquals(
        q.expectedRun.map(_.copy(because = "")),
        receipt.expectedRun.map(_.copy(because = ""))
      )
      assertEquals(q.exploration, None)
      assertEquals(q.total, None)
      val slots = if receipt.free then 119L * q.limits.steps
      else receipt.actions.size.min(q.limits.steps).toLong
      assertEquals(receipt.owner.fs.values.size.toLong * slots, receipt.total)
  }

  // fn-162 R8: each realization of the activity Protocol names the find Queries it runs; a find
  // Query none names is model-only (the reset settlement, keep_paused and competing deadlines).
  test("each activity realization names the Queries it runs") {
    val p = ActivityProtocol
    val runs: List[(Realizes[?, ?, ?], Seq[AnyRef])] = List(
      StandaloneActivityRun -> Seq(
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
      RetryAfterTimeoutRun -> Seq(
        p.timeouts.queries.retryAfterTimeout,
        p.timeouts.queries.retryExhaustion
      ),
      HeartbeatThenCompletionRun -> Seq(p.heartbeat.queries.heartbeatThenCompletes),
      RetryAfterHeartbeatRun -> Seq(p.heartbeat.queries.heartbeatTimeoutRetriesThenCompletes),
      ExhaustAfterHeartbeatRun -> Seq(p.heartbeat.queries.heartbeatTimeoutExhausts),
      ScheduledCompletionByIDRun -> Seq(p.completion.queries.scheduledCompletedByID),
      HeldFailureByIDRun -> Seq(p.retry.queries.heldFailedByID),
      HeldCancellationByIDRun -> Seq(
        p.cancellation.queries.heldCanceledByID,
        p.cancellation.queries.cancelIsRequested
      ),
      ResetAfterHeartbeatRun -> Seq(p.reset.queries.deferredResetCompletes)
    )
    for (realization, expected) <- runs do
      assert(realization.machine eq ActivityProtocol)
      assertEquals(realization.runs.size, expected.size, realization.toString)
      for (named, query) <- realization.runs.zip(expected) do
        assert(named eq query, realization.toString)
  }
