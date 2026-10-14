package temporal
package features.activity
package mini
package realization

import protocol.ActivityProtocol
import framework.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.ActivityExecutionStatus.*

private val calls =
  RequestBase(workflowService, "namespace" -> workerNamespace, "activity_id" -> run)

private val described = DescribedStatus(
  ActivityProtocol,
  calls,
  METHOD_DESCRIBE_ACTIVITY_EXECUTION,
  Field(_.getInfo),
  Field(_.activityId),
  Field(_.status)
)(
  protocol.Fact.statusCompleted -> ACTIVITY_EXECUTION_STATUS_COMPLETED,
  protocol.Fact.statusFailed -> ACTIVITY_EXECUTION_STATUS_FAILED,
  protocol.Fact.statusCanceled -> ACTIVITY_EXECUTION_STATUS_CANCELED,
  protocol.Fact.statusTimedOut -> ACTIVITY_EXECUTION_STATUS_TIMED_OUT
)

private val activityType = perCase("mini-activity")
private val activation = WorkerActivation.Activity(
  activityType, caseWorker, taskQueue, starts = Vector(activityWorker.poll))

private val startActivity = rpc(calls, METHOD_START_ACTIVITY_EXECUTION) {
  field(_.getActivityType.name) := Operand.named(activityType)
  field(_.getTaskQueue.name) := taskQueueName
  field(_.requestId) := run
}

private val startOne = startActivity.withFields {
  field(_.getRetryPolicy.maximumAttempts) := Operand.integer(1)
}
private val startTwo = startActivity.withFields {
  field(_.getRetryPolicy.maximumAttempts) := Operand.integer(MaxAttempts.bound)
}
// Temporal refuses a start without either a start-to-close or schedule-to-close deadline.
private val startOneUnreached = startOne.withFields {
  field(_.getStartToCloseTimeout) := duration(unreachedDeadlineSeconds)
}
private val startTwoUnreached = startTwo.withFields {
  field(_.getStartToCloseTimeout) := duration(unreachedDeadlineSeconds)
}

private val requestCancelActivity = rpc(calls, METHOD_REQUEST_CANCEL_ACTIVITY_EXECUTION) {}
private val completeAttempt = finish("done")
private val failAttempt =
  attemptFailure(applicationFailure("AttemptFailed", "attempt failed", retryable = true))
private val failActivity =
  attemptFailure(applicationFailure("AttemptFailed", "attempt failed", retryable = false))
private val withholdAttempt = attemptWithheld

private val attempts = script("attempts", activation)(
  onPath(deadline.startToClose)(withholdAttempt),
  perform(
    activityWorker.respondCompleted -> completeAttempt,
    activityWorker.respondFailed(Failure.retryable) -> failAttempt,
    activityWorker.respondFailed(Failure.fatal) -> failActivity
  )
)

private val unstartedDispatch =
  Actuator("unstarted-dispatch", ControlKind.HoldDispatched(activityWorker.poll), taskQueue)
private val holdDispatchBeforeTimeout = hold(unstartedDispatch)
