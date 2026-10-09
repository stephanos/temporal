package fixture.heartbeat

import framework.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*

val externalStart = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {}
val externalAnswer = rpc(workflowService, METHOD_RESPOND_ACTIVITY_TASK_FAILED_BY_ID) {}
val scheduledAnswer = rpc(workflowService, METHOD_RESPOND_ACTIVITY_TASK_COMPLETED_BY_ID) {}
val externalHeld = rpc(workflowService, METHOD_DESCRIBE_ACTIVITY_EXECUTION) {}
val externalTerminal = rpc(workflowService, METHOD_DESCRIBE_ACTIVITY_EXECUTION) {}
val externalTerminate = rpc(workflowService, METHOD_TERMINATE_ACTIVITY_EXECUTION) {}
val externalCleanup = command(externalTerminate, regardless = true)
val externalPublication = ActivityPublication("pending-activity")
val awaitExternalPublication = awaitActivityPublication(externalPublication)
val externalPending = attemptPending(externalAnswer)
val externalAttempts = script(
  "external-attempts",
  WorkerActivation.Activity(perCase("external-activity"), caseWorker, taskQueue)
)(everyCase(externalPending))

object ExternalBindings extends Realizes(Job):
  object controller
      extends Controller(
        everyCase(externalStart),
        everyCase(awaitExternalPublication),
        everyCase(externalHeld),
        everyCase(externalAnswer),
        everyCase(externalTerminal)
      )
  object workers extends Workers(externalAttempts)
  object serverSteps
      extends ServerSteps(
        ActivityExternalSettlement(
          externalStart,
          externalAttempts,
          1,
          externalPublication,
          externalHeld,
          externalAnswer,
          externalTerminal,
          externalCleanup
        )
      )

object ScheduledBindings extends Realizes(Job):
  object controller
      extends Controller(
        everyCase(externalStart),
        everyCase(scheduledAnswer),
        everyCase(externalTerminal)
      )
  object serverSteps
      extends ServerSteps(
        ActivityExternalSettlement.Scheduled(
          externalStart,
          scheduledAnswer,
          externalTerminal,
          externalCleanup
        )
      )
