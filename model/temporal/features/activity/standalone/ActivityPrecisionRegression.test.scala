package framework

import framework.Finite.given

import temporal.features.activity.{deadline, timers, Timeout}
import temporal.features.activity.standalone.{
  activityWorker,
  attemptCount,
  client,
  heartbeatDetails,
  protocol,
  MaxAttempts
}
import temporal.features.activity.standalone.protocol.{
  ActivityProtocol,
  ExhaustAfterHeartbeat,
  HeartbeatThenCompletion,
  HeldDelivery,
  LostAdmissionResponse,
  RetryAfterHeartbeat,
  RetryAfterTimeout,
  Standalone
}
import framework.realize.Instruction
import framework.outcomes.{Outcome, Rejection}
import io.temporal.api.activity.v1.ActivityExecutionInfo
import io.temporal.api.common.v1.Payloads
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.METHOD_DESCRIBE_ACTIVITY_EXECUTION

class ActivityPrecisionRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[protocol.Phase, protocol.Phase]]
  private def take(s: protocol.State, c: Class) =
    val binding = ActivityProtocol.bindings.find(_.decl == c.decl).get // scalafix:ok DisableSyntax.==
    effectOf[protocol.State, Outcome, protocol.Fact](binding.decl, binding.function)(s, c.values).head

  test(
    "cancellation precedence exercises a refused pause, permits legal closure and rejects undo"
  ) {
    val holds = ActivityProtocol.properties.cancelIsNotUndone.decl.holds2.get
      .asInstanceOf[
        (protocol.State, Step[protocol.State, Outcome, protocol.Fact]) => Boolean
      ] // scalafix:ok DisableSyntax.asInstanceOf
    val started = take(take(ActivityProtocol.init, client.start()).state, activityWorker.poll()).state
    val requested = take(started, client.requestCancel()).state
    assertEquals(requested.phase, protocol.Phase.cancelRequested)
    val pause = take(requested, client.pause())
    assertEquals(pause.outcome, Outcome.rejected(Rejection.failedPrecondition))
    assertEquals(pause.state, requested)
    assert(holds(requested, pause))
    assert(
      !holds(requested, pause.copy(state = requested.copy(phase = protocol.Phase.pauseRequested)))
    )
    val closures = List(
      activityWorker.respondCompleted(),
      activityWorker.respondFailed(temporal.features.activity.Failure.fatal),
      activityWorker.respondCanceled(),
      client.terminate(),
      deadline.startToClose()
    )
    val armed = requested.copy(startToClose = Timeout.expires)
    for action <- closures do
      val closed = take(armed, action)
      assert(closed.state.phase.in[Closed])
      assert(holds(armed, closed))
    for
      before <- summon[Finite[protocol.State]].values
      if before.phase == protocol.Phase.cancelRequested // scalafix:ok DisableSyntax.==
      binding <- ActivityProtocol.bindings
      inputs <- classesOf(binding.decl)
      after <- effectOf[protocol.State, Outcome, protocol.Fact](binding.decl, binding.function)(
        before,
        inputs
      )
    do assert(holds(before, after), s"${binding.decl.name} in $before")
    val checked = ActivityProtocol.queries.cancellationKeepsPrecedence
    assertEquals(checked.form, QueryForm.verify)
    assert(checked.scenario.free)
    assertEquals(checked.limits.steps, 8)
  }

  test("all ten timer branches explain the nominal versus real time window") {
    val scheduled = ActivityProtocol.init.copy(phase = protocol.Phase.scheduled)
    val held = scheduled.copy(
      phase = protocol.Phase.started,
      attempts = UpTo(1),
      startToClose = Timeout.expires
    )
    val branches = List(
      (scheduled.copy(dispatch = protocol.Dispatch.startDelay), timers.startDelay()),
      (scheduled.copy(dispatch = protocol.Dispatch.backoff), timers.backoff()),
      (scheduled.copy(scheduleToClose = Timeout.expires), deadline.scheduleToClose()),
      (scheduled.copy(scheduleToStart = Timeout.expires), deadline.scheduleToStart()),
      (held, deadline.startToClose()),
      (held.copy(phase = protocol.Phase.pauseRequested), deadline.startToClose()),
      (held.copy(maxAttempts = MaxAttempts.one), deadline.startToClose()),
      (held.copy(heartbeat = Timeout.expires), deadline.heartbeat()),
      (
        held.copy(phase = protocol.Phase.pauseRequested, heartbeat = Timeout.expires),
        deadline.heartbeat()
      ),
      (held.copy(maxAttempts = MaxAttempts.one, heartbeat = Timeout.expires), deadline.heartbeat())
    )
    for (before, action) <- branches do
      val explanation = take(before, action).because
      assert(explanation.contains("nominal"), s"${action.decl.name}: $explanation")
      assert(explanation.contains("real timer"), explanation)
      assert(explanation.contains("model.go:300-310"), explanation)
  }

  test("every realization declares a typed raw attempt observation and ends with a Describe read") {
    val original = List(Standalone, RetryAfterTimeout, HeldDelivery, LostAdmissionResponse)
    val realizations =
      original ++ List(HeartbeatThenCompletion, RetryAfterHeartbeat, ExhaustAfterHeartbeat)
    val controllers = List(
      Standalone.controller,
      RetryAfterTimeout.controller,
      HeldDelivery.controller,
      LostAdmissionResponse.controller,
      HeartbeatThenCompletion.controller,
      RetryAfterHeartbeat.controller,
      ExhaustAfterHeartbeat.controller
    )
    for (realization, controller) <- realizations.zip(controllers) do
      val observed = if original.contains(realization) then
        realization.observations.filter(_.id == attemptCount.name)
      else
        realization.observations.filter(
          _.message == ActivityExecutionInfo.scalaDescriptor.fullName
        ) // scalafix:ok DisableSyntax.==
      assertEquals(observed.size, 1)
      assertEquals(observed.head.id, attemptCount.name)
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
    for realization <- List(HeartbeatThenCompletion, RetryAfterHeartbeat, ExhaustAfterHeartbeat) do
      val details = realization.observations.filter(
        _.message == Payloads.scalaDescriptor.fullName
      ) // scalafix:ok DisableSyntax.==
      assertEquals(details.size, 1)
      assertEquals(details.head.id, heartbeatDetails.name)
      assertEquals(details.head.message, Payloads.scalaDescriptor.fullName)
  }
