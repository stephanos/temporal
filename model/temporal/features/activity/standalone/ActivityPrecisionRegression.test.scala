package umpire

import temporal.features.activity.{deadline, timers, Timeout}
import temporal.features.activity.standalone.{attemptCount, client, system, worker, MaxAttempts}
import temporal.features.activity.standalone.system.{
  ActivitySystem,
  HeldDelivery,
  LostAdmissionResponse,
  RetryAfterTimeout,
  Standalone
}
import umpire.realize.Instruction
import umpire.outcomes.{Outcome, Rejection}
import io.temporal.api.activity.v1.ActivityExecutionInfo
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.METHOD_DESCRIBE_ACTIVITY_EXECUTION

class ActivityPrecisionRegression extends munit.FunSuite:
  private def take(s: system.State, c: Class) =
    val binding = ActivitySystem.bindings.find(_.decl == c.decl).get
    effectOf[system.State, Outcome, system.Fact](binding.decl, binding.function)(s, c.values).head

  test(
    "cancellation precedence exercises a refused pause, permits legal closure and rejects undo"
  ) {
    val holds = ActivitySystem.properties.cancelIsNotUndone.decl.holds2.get
      .asInstanceOf[(system.State, Step[system.State, Outcome, system.Fact]) => Boolean]
    // scalafix:ok DisableSyntax.asInstanceOf
    val started = take(take(ActivitySystem.init, client.start()).state, worker.poll()).state
    val requested = take(started, client.requestCancel()).state
    assertEquals(requested.phase, system.Phase.cancelRequested)
    val pause = take(requested, client.pause())
    assertEquals(pause.outcome, Outcome.rejected(Rejection.failedPrecondition))
    assertEquals(pause.state, requested)
    assert(holds(requested, pause))
    assert(
      !holds(requested, pause.copy(state = requested.copy(phase = system.Phase.pauseRequested)))
    )
    val closures = List(
      worker.respondCompleted(),
      worker.respondFailed(temporal.features.activity.Failure.fatal),
      worker.respondCanceled(),
      client.terminate(),
      deadline.startToClose()
    )
    val armed = requested.copy(startToClose = Timeout.expires)
    for action <- closures do
      val closed = take(armed, action)
      assert(closed.state.phase.in[Closed])
      assert(holds(armed, closed))
    for
      before <- summon[Finite[system.State]].values if before.phase == system.Phase.cancelRequested
      binding <- ActivitySystem.bindings
      inputs <- classesOf(binding.decl)
      after <- effectOf[system.State, Outcome, system.Fact](binding.decl, binding.function)(
        before,
        inputs
      )
    do assert(holds(before, after), s"${binding.decl.name} in $before")
    val checked = ActivitySystem.queries.cancellationKeepsPrecedence
    assertEquals(checked.form, QueryForm.verify)
    assert(checked.scenario.free)
    assertEquals(checked.limits.steps, 8)
  }

  test("all seven timer branches explain the nominal versus real time window") {
    val scheduled = ActivitySystem.init.copy(phase = system.Phase.scheduled)
    val held = scheduled.copy(
      phase = system.Phase.started,
      attempts = UpTo(1),
      startToClose = Timeout.expires
    )
    val branches = List(
      (scheduled.copy(dispatch = system.Dispatch.startDelay), timers.startDelay()),
      (scheduled.copy(dispatch = system.Dispatch.backoff), timers.backoff()),
      (scheduled.copy(scheduleToClose = Timeout.expires), deadline.scheduleToClose()),
      (scheduled.copy(scheduleToStart = Timeout.expires), deadline.scheduleToStart()),
      (held, deadline.startToClose()),
      (held.copy(phase = system.Phase.pauseRequested), deadline.startToClose()),
      (held.copy(maxAttempts = MaxAttempts.one), deadline.startToClose())
    )
    for (before, action) <- branches do
      val explanation = take(before, action).because
      assert(explanation.contains("nominal"), s"${action.decl.name}: $explanation")
      assert(explanation.contains("real timer"), explanation)
      assert(explanation.contains("model.go:300-310"), explanation)
  }

  test("every realization declares a typed raw attempt observation and ends with a Describe read") {
    val realizations = List(Standalone, RetryAfterTimeout, HeldDelivery, LostAdmissionResponse)
    val controllers = List(
      Standalone.controller,
      RetryAfterTimeout.controller,
      HeldDelivery.controller,
      LostAdmissionResponse.controller
    )
    for (realization, controller) <- realizations.zip(controllers) do
      val observed = realization.observations.filter(_.id == attemptCount.name)
      assertEquals(observed.size, 1)
      assertEquals(observed.head.message, ActivityExecutionInfo.scalaDescriptor.fullName)
      val last = controller.items.last
      assert(last.when.isEmpty && last.performs.isEmpty)
      last.command.get.instruction match
        case rpc: Instruction.TypedRpc[?, ?] =>
          assertEquals(
            rpc.method.getFullMethodName,
            METHOD_DESCRIBE_ACTIVITY_EXECUTION.getFullMethodName
          )
        case other => fail(s"final read is $other")
  }
