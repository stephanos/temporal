/* The standalone Nexus operation realization: a controller starts one operation through
 * StartNexusOperationExecution on the Case's endpoint, controls it, and reads its status back
 * through DescribeNexusOperationExecution. No handler answers the operation, so it stays running
 * until a control settles it. A handler's answer to a standalone operation is not realized: the
 * Driver reserves a Nexus handler only through a workflow's or an activity's start
 * (.plans/SEMANTIC_PROTOCOLS.md).
 */
package temporal
package features.nexusoperation

import umpire.*
import umpire.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.NexusOperationExecutionStatus.*

import OperationFamily.given

object OperationRealization:
  // Moved from temporal.nexusoperation; the pin keeps its Definition IDs.
  given DefinitionScope = DefinitionScope("temporal.nexusoperation.OperationRealization$")

  /** The status DescribeNexusOperationExecution reports, read once the operation stays in it. */
  private def status(fact: Fact) = Evidence.read(
    id = evidenceId(fact),
    records = fact,
    source = sourceId(fact),
    from = Recorded.single(METHOD_DESCRIBE_NEXUS_OPERATION_EXECUTION, Field(_.getInfo)),
    operation = Field(_.operationId),
    commitment = Commitment.reported
  )

  /** The status the operation's description reports while each fact holds (operationExecutionStatus). */
  val operationStatus = statusTable(
    OperationFact.statusSucceeded -> NEXUS_OPERATION_EXECUTION_STATUS_COMPLETED,
    OperationFact.statusFailed -> NEXUS_OPERATION_EXECUTION_STATUS_FAILED,
    OperationFact.statusCanceled -> NEXUS_OPERATION_EXECUTION_STATUS_CANCELED,
    OperationFact.statusTerminated -> NEXUS_OPERATION_EXECUTION_STATUS_TERMINATED
  )

  /** Polls the operation's description until it reads the status the fact's evidence names. */
  private def awaitStatus(fact: Fact) =
    await(status(fact), workflowService)(
      Condition.equal(Field(_.status), Operand.enumValue(operationStatus(fact)))
    ) {
      field(_.namespace) := workerNamespace
      field(_.operationId) := run
    }

  /** The service and operation the start names, which no handler of the Case answers. */
  private val service = "umpire-case-service"

  /** nexusoperation.Enabled's key (chasm/lib/nexusoperation/config.go). */
  private val settingKey = "nexusoperation.enableStandalone"
  private val operationName = "complete"

  /** The start, under the run's id, of an operation the Case's endpoint names and no handler answers. */
  private val startOperation = rpc(workflowService, METHOD_START_NEXUS_OPERATION_EXECUTION) {
    field(_.namespace) := workerNamespace
    field(_.operationId) := run
    field(_.endpoint) := nexusEndpointName
    field(_.service) := Operand.text(service)
    field(_.operation) := Operand.text(operationName)
    field(_.requestId) := run
  }
  private val requestCancelOperation =
    rpc(workflowService, METHOD_REQUEST_CANCEL_NEXUS_OPERATION_EXECUTION) {
      field(_.namespace) := workerNamespace
      field(_.operationId) := run
      field(_.requestId) := run
    }
  private val terminateOperation =
    rpc(workflowService, METHOD_TERMINATE_NEXUS_OPERATION_EXECUTION) {
      field(_.namespace) := workerNamespace
      field(_.operationId) := run
      field(_.requestId) := run
    }

  private val awaitTerminated = awaitStatus(OperationFact.statusTerminated)

  private val operationController = controller(
    perform(start -> startOperation),
    perform(requestCancel -> requestCancelOperation),
    perform(terminate -> terminateOperation),
    onPath(terminate)(awaitTerminated)
  )

  /**
   * The frontend serves the standalone operation only with its flag on
   * (chasm/lib/nexusoperation/frontend.go isStandaloneNexusOperationEnabled); a Case is refused at
   * preparation in an environment that leaves it off.
   */
  private val standaloneEnabled = RequiredSetting(key = settingKey, value = "true")

  /** One standalone operation a controller starts on the Case's endpoint. */
  val standalone: Realization = temporalRealization(
    machine = Operation.nexusOperation,
    operation = operation,
    roles = Vector(workflowService, taskQueue, nexusEndpoint),
    scripts = Vector(operationController),
    evidence = Vector(
      answered(OperationFact.statusScheduled, startOperation),
      answered(OperationFact.statusCancelRequested, requestCancelOperation),
      status(OperationFact.statusTerminated)
    ),
    requiredSettings = Vector(standaloneEnabled)
  )
