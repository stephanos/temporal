// The standalone Nexus operation realization: a controller starts one operation through
// StartNexusOperationExecution on the Case's endpoint, controls it, and reads its status back
// through DescribeNexusOperationExecution. No handler answers the operation, so it stays running
// until a control settles it. A handler's answer to a standalone operation is not realized: the
// Driver reserves a Nexus handler only through a workflow's or an activity's start
// (.plans/SEMANTIC_PROTOCOLS.md).
package temporal
package features.nexus
package standalone

import system.{Fact as OperationFact, NexusSystem}
import umpire.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.NexusOperationExecutionStatus.*

// Every call of the controller is made on the WorkflowService, in the run's namespace, of the
// operation the run started under its own id.
private val calls =
  RequestBase(workflowService, "namespace" -> workerNamespace, "operation_id" -> run)

// The status DescribeNexusOperationExecution reports while each fact holds
// (operationExecutionStatus), read once the operation stays in it.
private val described = DescribedStatus(
  NexusSystem,
  calls,
  METHOD_DESCRIBE_NEXUS_OPERATION_EXECUTION,
  Field(_.getInfo),
  Field(_.operationId),
  Field(_.status)
)(
  OperationFact.nexusOperationCompleted -> NEXUS_OPERATION_EXECUTION_STATUS_COMPLETED,
  OperationFact.nexusOperationFailed -> NEXUS_OPERATION_EXECUTION_STATUS_FAILED,
  OperationFact.nexusOperationCanceled -> NEXUS_OPERATION_EXECUTION_STATUS_CANCELED,
  OperationFact.nexusOperationTerminated -> NEXUS_OPERATION_EXECUTION_STATUS_TERMINATED
)

// The status table the machine's Describable capability names.
val operationStatus = described.table

// The service and operation the start names, which no handler of the Case answers.
private val service = "umpire-case-service"

// nexusoperation.Enabled's key (chasm/lib/nexusoperation/config.go).
private val settingKey = "nexusoperation.enableStandalone"
private val operationName = "complete"

// The start, under the run's id, of an operation the Case's endpoint names and no handler answers.
private val startOperation = rpc(calls, METHOD_START_NEXUS_OPERATION_EXECUTION) {
  field(_.endpoint) := nexusEndpointName
  field(_.service) := Operand.text(service)
  field(_.operation) := Operand.text(operationName)
  field(_.requestId) := run
}
private val requestCancelOperation =
  rpc(calls, METHOD_REQUEST_CANCEL_NEXUS_OPERATION_EXECUTION) {
    field(_.requestId) := run
  }
private val terminateOperation =
  rpc(calls, METHOD_TERMINATE_NEXUS_OPERATION_EXECUTION) {
    field(_.requestId) := run
  }

// The frontend serves the standalone operation only with its flag on
// (chasm/lib/nexusoperation/frontend.go isStandaloneNexusOperationEnabled); a Case is refused at
// preparation in an environment that leaves it off.
private val standaloneEnabled = RequiredSetting(key = settingKey, value = "true")

// One standalone operation a controller starts on the Case's endpoint.
object Standalone extends Realizes(NexusSystem, requiredSettings = Vector(standaloneEnabled)):
  object controller
      extends Controller(
        perform(client.start -> startOperation),
        perform(client.requestCancel -> requestCancelOperation),
        perform(client.terminate -> terminateOperation),
        onPath(client.terminate)(described.await(OperationFact.nexusOperationTerminated))
      )
  object evidence
      extends Evidences(
        answered(OperationFact.statusScheduled, startOperation),
        answered(OperationFact.statusCancelRequested, requestCancelOperation),
        described(OperationFact.nexusOperationTerminated)
      )
