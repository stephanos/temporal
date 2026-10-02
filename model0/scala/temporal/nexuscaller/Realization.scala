/* The Nexus caller-side realization, ported from model/lean/Temporal/Case/Realization/Nexus.lean and
 * model/go/nexuscaller/realization.go.
 *
 * A controller-started workflow schedules one Nexus operation on the Case's endpoint role; the
 * handler answers it, and the controller completes it when the handler answered asynchronously; the
 * controller reads the history the Case's evidence is lifted from. The realization writes no
 * Program: it declares that scaffolding once and binds each action class of a Model to the
 * instruction that performs it, so the producer puts those instructions where a Query's path took
 * them.
 */
package temporal
package nexuscaller

import umpire.caseproducer.*
import umpire.caseproducer.Build.*
import com.google.protobuf.{ByteString, Duration}
import io.temporal.api.command.v1.{Command, ScheduleNexusOperationCommandAttributes}
import io.temporal.api.common.v1.Payload
import io.temporal.api.enums.v1.{CommandType, NexusHandlerErrorRetryBehavior}
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure}
import io.temporal.api.nexus.v1.{HandlerError as NexusHandlerError, StartOperationResponse}
import io.temporal.api.nexus.v1.Failure as NexusFailure
import temporal.server.api.testpilot.v1.ExpressionOuterClass.{Expression, InstructionReference}
import temporal.server.api.testpilot.v1.InstructionOuterClass.*
import temporal.server.api.testpilot.v1.ProgramOuterClass.*

import scala.jdk.CollectionConverters.*

object NexusRealization:
  // Roles and methods every realization shares (model/lean/Temporal/Case/Support.lean).
  private val workflowServiceRole = "temporal.workflow-service"
  private val workerRole = "temporal.worker"
  private val taskQueueRole = "temporal.task-queue"
  private val handlerTaskQueueRole = "temporal.handler-task-queue"
  private val nexusEndpointRole = "temporal.nexus-endpoint"

  private val startWorkflowMethod = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
  private val getHistoryMethod = "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"
  private val describeMethod = "/temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution"

  private val historyObservation = "history-event"
  private val correlatedObservation = "correlated-evidence"

  private val workerNamespaceBinding = "temporal.worker.namespace"
  private val taskQueueBinding = "temporal.task-queue.resource"
  private val handlerTaskQueueBinding = "temporal.handler-task-queue.resource"
  private val nexusEndpointBinding = "temporal.nexus-endpoint.resource"

  // Definition IDs every Case on this realization reads its evidence by.
  private val projectionID = "temporal.nexus.caller.projection"
  private val historySourceID = "temporal.nexus.caller.source.history"
  private val describeSourceID = "temporal.nexus.caller.source.describe"
  private val scheduledSource = "temporal.nexus.caller.source.scheduled"
  private val runFieldID = "temporal.nexus.caller.scope.run"
  private val operationFieldID = "temporal.nexus.caller.scope.operation"

  private def evidenceKind(name: String) = s"temporal.nexus.caller.evidence.$name"

  /** One history event kind of the operation, keyed by the scheduled event it answers. */
  private def historySource(kind: String, attributes: String, kindName: String) =
    EvidenceSource(kind, Recorded(historyAttributes = attributes),
      makePath(oneofMember("attributes", attributes), field("scheduled_event_id")), evidenceKind(kindName), historySourceID)

  private val historyEvents = makePath(field("history"), repeated("events"))

  /** Every evidence kind the realization admits: the scheduled event read out of history as soon as
    * it exists, the history kinds, and the pending operation's attempt count. */
  val sources: Vector[EvidenceSource] = Vector(
    EvidenceSource("nexusOperationScheduled", Recorded(method = getHistoryMethod, path = historyEvents),
      makePath(field("event_id")), evidenceKind("scheduled"), scheduledSource),
    historySource("nexusOperationStarted", "nexus_operation_started_event_attributes", "started"),
    historySource("nexusOperationCompleted", "nexus_operation_completed_event_attributes", "completed"),
    historySource("nexusOperationFailed", "nexus_operation_failed_event_attributes", "failed"),
    historySource("nexusOperationCanceled", "nexus_operation_canceled_event_attributes", "canceled"),
    historySource("nexusOperationTimedOut", "nexus_operation_timed_out_event_attributes", "timedOut"),
    EvidenceSource("pendingAttempts", Recorded(method = describeMethod, path = makePath(field("pending_nexus_operations"))),
      makePath(field("scheduled_event_id")), evidenceKind("pendingAttempts"), describeSourceID),
  )

  // ### The scaffolding

  private def workflowTypeOf(identity: Identity) = s"umpire-${identity.fixture}-workflow"

  private def rpc(id: String, method: String, assignments: Seq[RequestAssignment], reads: ResponseRead*) =
    node(id, invokeRPC(workflowServiceRole, method, assignments, reads))

  private def historyAssignments = Seq(
    assign("namespace", environment(workerNamespaceBinding)),
    assign("execution.workflow_id", run),
    assign("maximum_page_size", literal(signedInteger(64))),
    assign("wait_new_event", literal(bool(true))),
  )

  private def startWorkflowNode(workflowType: String) = rpc("start-workflow", startWorkflowMethod, Seq(
    assign("namespace", environment(workerNamespaceBinding)),
    assign("workflow_id", run),
    assign("workflow_type.name", literal(text(workflowType))),
    assign("task_queue.name", environment(taskQueueBinding)),
    assign("request_id", run),
  ))

  /** Resolves only once the workflow closes, so a read placed after it observes the whole history. */
  private def awaitCloseNode = rpc("await-close", getHistoryMethod,
    historyAssignments :+ assign("history_event_filter_type", literal(enumValue("HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT"))))

  /** Lifts the history kinds among the resolved rules; a path that records none lifts nothing,
    * because a lift with no rule is a Case preparation rejects. */
  private def historyNode(rules: Vector[EvidenceRule]) =
    val lift = if rules.exists(_.readsHistory) then Seq(evidenceTarget(correlatedObservation, rules)) else Seq.empty
    rpc("history", getHistoryMethod, historyAssignments,
      responseRead(historyEvents, ReadCardinality.READ_CARDINALITY_EMIT_EACH, (observationTarget(historyObservation) +: lift)*))

  private def pollAssignments = Seq(assign("namespace", environment(workerNamespaceBinding)), assign("execution.workflow_id", run))

  /** Polls the pending operation until its first attempt has failed. */
  private def pendingAttemptsNode = node("pending-attempts", readEvidence(evidenceKind("pendingAttempts"), workflowServiceRole,
    pollAssignments, equal(path(projectedValue, "attempt"), literal(signedInteger(1))), 250))

  /** Polls the history for the scheduled event, run until the event exists. */
  private def awaitScheduledNode = node("await-scheduled", readEvidence(evidenceKind("scheduled"), workflowServiceRole,
    pollAssignments, present(path(projectedValue, makePath(oneofMember("attributes", "nexus_operation_scheduled_event_attributes")))), 250))

  // ### Action classes: the IDs a binding is read by where the Model has no member of its key.

  private val scheduleAction = "temporal.nexus.caller.action.schedule"
  private val scheduleToStartAction = "temporal.nexus.caller.action.schedule.scheduleToStart"
  private val startToCloseAction = "temporal.nexus.caller.action.schedule.startToClose"
  private val handlerReplyAction = "temporal.nexus.caller.action.handlerReply"
  private val handlerReplySyncAction = "temporal.nexus.caller.action.handlerReply.syncSuccess"
  private val handlerReplyFailedAction = "temporal.nexus.caller.action.handlerReply.operationFailed"
  private val handlerErrorRetryable = "temporal.nexus.caller.action.handlerReply.handlerError.retryable"
  private val handlerErrorNonRetryable = "temporal.nexus.caller.action.handlerReply.handlerError.nonRetryable"
  private val completeAction = "temporal.nexus.caller.action.complete"
  private val completeFailedAction = "temporal.nexus.caller.action.complete.failed"
  private val workerStopAction = "temporal.nexus.caller.action.workerStop"

  private def completionAuthority(p: Placement) = s"completion-authority${p.suffix}"
  private def operationOf(operation: String, p: Placement) = operation + p.suffix

  /** A JSON payload of one string, as the SDK's default data converter encodes it. */
  private def textPayload(value: String) =
    Payload.newBuilder().putMetadata("encoding", ByteString.copyFromUtf8("json/plain")).setData(ByteString.copyFromUtf8(s"\"$value\"")).build()

  /** The failure a failed reply or completion carries. */
  private def handlerFailure = Failure.newBuilder().setMessage("operation failed")
    .setApplicationFailureInfo(ApplicationFailureInfo.newBuilder().setType("OperationFailed").setNonRetryable(true)).build()

  /** The durations the deadlines a path sets realize as; the backoff is the server's own. */
  private val timers = Map("scheduleToStart" -> 2000L, "startToClose" -> 2000L)

  private def duration(name: String): Option[Duration] =
    timers.get(name).map(ms => Duration.newBuilder().setSeconds(ms / 1000).setNanos((ms % 1000).toInt * 1000000).build())

  private def instruction(f: Instruction.Builder => Instruction.Builder): Instruction = f(Instruction.newBuilder()).build()

  // ### The bindings
  //
  // Each binding builds its node from the id the producer supplies, so the same class performed
  // twice on one path produces two distinct nodes.

  /** The schedule command for one class of the schedule action: the deadlines the class sets, at the
    * realization's durations. */
  private def scheduleBinding(action: String, key: String, service: String, operation: String,
      scheduleToStart: Option[Duration], startToClose: Option[Duration]) =
    ActionBinding(action, key, "start-nexus-operation", (p, id) =>
      val attributes = ScheduleNexusOperationCommandAttributes.newBuilder().setEndpoint(nexusEndpointRole).setService(service)
        .setOperation(operationOf(operation, p)).setInput(textPayload("request"))
      scheduleToStart.foreach(attributes.setScheduleToStartTimeout)
      startToClose.foreach(attributes.setStartToCloseTimeout)
      node(id, instruction(_.setWorkflowCommand(WorkflowCommand.newBuilder().setCommand(Command.newBuilder()
        .setCommandType(CommandType.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION).setScheduleNexusOperationCommandAttributes(attributes))))))

  private def replyBinding(action: String, key: String, id: String, reply: Placement => NexusHandlerReply) =
    ActionBinding(action, key, id, (p, nodeID) => node(nodeID, instruction(_.setNexusHandlerReply(reply(p))), timeoutMilliseconds = Some(5000)))

  private def response(r: StartOperationResponse.Builder, slot: String) =
    NexusHandlerReply.newBuilder().setResponse(r).setHandleSlotId(slot).build()

  private def handlerError(errorType: String, behavior: NexusHandlerErrorRetryBehavior) =
    NexusHandlerReply.newBuilder().setError(NexusHandlerError.newBuilder().setErrorType(errorType)
      .setFailure(NexusFailure.newBuilder().setMessage("handler error")).setRetryBehavior(behavior)).build()

  private def completion(p: Placement, f: NexusOperationCompletion.Builder => NexusOperationCompletion.Builder) =
    instruction(_.setNexusOperationCompletion(f(NexusOperationCompletion.newBuilder().setHandleSlotId(completionAuthority(p)))))

  /** Every binding the realization carries, the schedule command once per class of deadline a path of
    * the caller Model sets. */
  def bindings(service: String, operation: String): Vector[ActionBinding] = Vector(
    scheduleBinding(scheduleAction, "schedule-unset-unset-unset", service, operation, None, None),
    scheduleBinding(scheduleToStartAction, "schedule-unset-expires-unset", service, operation, duration("scheduleToStart"), None),
    scheduleBinding(startToCloseAction, "schedule-unset-unset-expires", service, operation, None, duration("startToClose")),
    replyBinding(handlerReplyAction, "handlerReply-async", "respond-async", p =>
      response(StartOperationResponse.newBuilder().setAsyncSuccess(StartOperationResponse.Async.getDefaultInstance), completionAuthority(p))),
    replyBinding(handlerReplySyncAction, "handlerReply-syncSuccess", "respond-sync", _ =>
      response(StartOperationResponse.newBuilder().setSyncSuccess(StartOperationResponse.Sync.newBuilder().setPayload(textPayload("completed"))), "")),
    replyBinding(handlerReplyFailedAction, "handlerReply-operationFailed", "respond-failed", _ =>
      response(StartOperationResponse.newBuilder().setFailure(handlerFailure), "")),
    replyBinding(handlerErrorRetryable, "handlerReply-handlerError-true", "respond-error-retryable", _ =>
      handlerError("INTERNAL", NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE)),
    replyBinding(handlerErrorNonRetryable, "handlerReply-handlerError-false", "respond-error", _ =>
      handlerError("BAD_REQUEST", NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE)),
    ActionBinding(completeAction, "complete-succeeded", "complete-nexus-operation", (p, id) =>
      node(id, completion(p, _.setPayload(textPayload("completed"))))),
    ActionBinding(completeFailedAction, "complete-failed", "fail-nexus-operation", (p, id) =>
      node(id, completion(p, _.setFailure(handlerFailure)))),
    // The handler's worker stops polling its own queue, so the caller workflow keeps running.
    ActionBinding(workerStopAction, "workerStop", "stop-handler-worker", (_, id) =>
      node(id, instruction(_.setInjectFault(InjectFault.newBuilder().setRoleId(handlerTaskQueueRole).setKind(FaultKind.FAULT_KIND_WORKER_STOP))))),
  )

  // ### The plan
  //
  // The controller's sequence is the one place the interleaving matters: it stops the handler's
  // worker when the path says so, starts the workflow, reads the scheduled event as soon as it
  // exists, polls the attempt count when a retryable failure is on the path, waits for the authority
  // the handler publishes when a completion is on the path, performs the completion, waits for the
  // workflow to close, and only then reads history.

  private val completionKeys = Vector("complete-succeeded", "complete-failed")
  private val scheduleKeys = Vector("schedule-unset-unset-unset", "schedule-unset-expires-unset", "schedule-unset-unset-expires")
  private val retryKeys = Vector("handlerReply-handlerError-true")
  private val alwaysGuard: Option[Expression] = Some(literal(bool(true)))

  private def entrypoint(id: String, nodes: Vector[InstructionNode])(activate: Entrypoint.Builder => Entrypoint.Builder) =
    activate(Entrypoint.newBuilder().setEntrypointId(id).addAllInstructions(nodes.asJava)).build()

  private def asyncPlan(service: String, operation: String) = ProgramPlan(
    roles = Vector(
      role(workflowServiceRole, RoleKind.ROLE_KIND_ENDPOINT),
      role(workerRole, RoleKind.ROLE_KIND_WORKER, workerNamespaceBinding),
      role(taskQueueRole, RoleKind.ROLE_KIND_TASK_QUEUE, workerNamespaceBinding, taskQueueBinding),
      role(handlerTaskQueueRole, RoleKind.ROLE_KIND_TASK_QUEUE, workerNamespaceBinding, handlerTaskQueueBinding),
      role(nexusEndpointRole, RoleKind.ROLE_KIND_ENDPOINT, "", nexusEndpointBinding),
    ),
    instanceSlots = p => Vector(handleSlot(completionAuthority(p))),
    observations = Vector(
      messageObservation(historyObservation, "temporal.api.history.v1.HistoryEvent"),
      messageObservation(correlatedObservation, "temporal.server.api.testpilot.v1.CorrelatedEvidence"),
    ),
    entrypoints = Vector(
      EntrypointPlan((_, nodes) => entrypoint("controller", nodes)(_.setController(ControllerActivation.getDefaultInstance)), Vector(
        Item.Actions(Vector(workerStopAction)),
        Item.Fixed((p, _) => startWorkflowNode(workflowTypeOf(p.identity))),
        Item.Fixed((_, _) => awaitScheduledNode),
        Item.WhenOnPath(retryKeys, (_, _) => pendingAttemptsNode),
        Item.PerInstance(Vector(
          Item.WhenOnPath(completionKeys, (p, _) => node(s"await-completion-authority${p.suffix}",
            instruction(_.setAwaitSlot(AwaitSlot.newBuilder().setSlotId(completionAuthority(p)))))),
          Item.Actions(Vector(completeAction, completeFailedAction)),
        )),
        Item.Fixed((_, _) => awaitCloseNode),
        Item.Fixed((_, rules) => historyNode(rules)),
      )),
      EntrypointPlan((p, nodes) => entrypoint("workflow", nodes)(_.setWorkflow(WorkflowActivation.newBuilder()
        .setWorkflowType(workflowTypeOf(p.identity)).setWorkerRoleId(workerRole).setTaskQueueRoleId(taskQueueRole))), Vector(
        Item.Actions(Vector(scheduleAction, scheduleToStartAction, startToCloseAction)),
        Item.WhenOnPath(scheduleKeys, (p, _) => node(s"await-nexus-operation${p.suffix}",
          instruction(_.setAwaitInstruction(AwaitInstruction.newBuilder().setInstruction(InstructionReference.newBuilder()
            .setEntrypointId("workflow").setInstructionId(s"start-nexus-operation${p.suffix}")))), guard = alwaysGuard)),
        // The workflow closes on every path: a failed or timed-out operation is the await's recorded
        // outcome, not a reason to leave the workflow open.
        Item.Fixed((_, _) => node("finish-workflow", instruction(_.setFinish(Finish.newBuilder().setResult(literal(text("done"))))),
          timeoutMilliseconds = Some(5000), guard = alwaysGuard)),
      )),
      EntrypointPlan((p, nodes) => entrypoint(s"handler${p.suffix}", nodes)(_.setNexusHandler(NexusHandlerActivation.newBuilder()
        .setService(service).setOperation(operationOf(operation, p)).setWorkerRoleId(workerRole).setTaskQueueRoleId(handlerTaskQueueRole))),
        Vector(Item.Actions(Vector(handlerReplyAction, handlerReplySyncAction, handlerReplyFailedAction, handlerErrorRetryable,
          handlerErrorNonRetryable))), perInstance = true),
    ),
    cleanup = Some(Cleanup.newBuilder().setEntrypointId("cleanup").build()),
  )

  /** One Nexus operation scheduled by a controller-started workflow and answered by a handler inside
    * the Case's own worker. */
  def asyncNexus(service: String, operation: String): umpire.caseproducer.Realization = umpire.caseproducer.Realization(
    plan = asyncPlan(service, operation),
    actions = bindings(service, operation),
    producerID = "temporal.nexus.caller.testpilot",
    producerVersion = "1",
    projectionID = projectionID,
    scopeField = runFieldID,
    operationKey = operationFieldID,
    historyObservation = historyObservation,
    correlatedObservation = correlatedObservation,
    sources = sources,
    projectionLimits = ProjectionLimits(events = 32, buffered = 16, keys = 8, support = 128, work = 1000000000, eventSize = 512),
  )

  /** Where the Model's Lean counterpart is declared, which Case provenance names; the Scala Model
    * reuses it so the Case bytes can be compared with the checked-in fixtures. */
  val modelSource: Source = Source("Temporal/Feature/Nexus/Caller/Model.lean", "lean-model")
