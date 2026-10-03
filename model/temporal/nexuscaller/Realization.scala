/* The Nexus caller-side realization: what a Case does to a deployment to take the path a Query
 * found.
 *
 * A controller-started workflow schedules one Nexus operation on the Case's endpoint role; the
 * handler answers it, and the controller completes it when the handler answered asynchronously; the
 * controller reads the history the Case's evidence is lifted from. The realization writes no
 * Program: it declares that scaffolding once and binds each action class of a Model to the
 * instruction that performs it, so the producer puts those instructions where a Query's path took
 * them.
 *
 * Everything below is a declaration the lifter emits into the IR; Go lowers a Query's witness
 * through it into a Testpilot Case (tools/umpire/lower).
 */
package temporal
package nexuscaller

import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.*
import umpire.realize.Operand.*
import umpire.realize.ProtoValue.*
import io.temporal.api.workflowservice.v1.*
import io.temporal.api.history.v1.*
import io.temporal.api.workflow.v1.PendingNexusOperationInfo
import io.temporal.api.common.v1.Payload
import io.temporal.api.command.v1.{Command as ApiCommand, ScheduleNexusOperationCommandAttributes}
import io.temporal.api.enums.v1.{
  CommandType,
  HistoryEventFilterType,
  NexusHandlerErrorRetryBehavior
}
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure as ApiFailure}
import io.temporal.api.nexus.v1.{Failure as NexusFailure, HandlerError, StartOperationResponse}
import temporal.server.api.testpilot.v1.CorrelatedEvidence
import io.grpc.MethodDescriptor
import scalapb.GeneratedMessage

import Timeout.{expires, unset}

object NexusRealization:
  // Roles and methods every realization shares.
  private val workflowServiceRole = "temporal.workflow-service"
  private val workerRole = "temporal.worker"
  private val taskQueueRole = "temporal.task-queue"
  private val handlerTaskQueueRole = "temporal.handler-task-queue"
  private val nexusEndpointRole = "temporal.nexus-endpoint"

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

  private val scheduledEvidence = "temporal.nexus.caller.evidence.scheduled"
  private val startedEvidence = "temporal.nexus.caller.evidence.started"
  private val completedEvidence = "temporal.nexus.caller.evidence.completed"
  private val failedEvidence = "temporal.nexus.caller.evidence.failed"
  private val canceledEvidence = "temporal.nexus.caller.evidence.canceled"
  private val timedOutEvidence = "temporal.nexus.caller.evidence.timedOut"
  private val pendingAttemptsEvidence = "temporal.nexus.caller.evidence.pendingAttempts"

  /** The service and operation the handler script answers. */
  private val service = "umpire.case.service"
  private val operation = "complete"

  /**
   * One history event kind of the operation, keyed by the scheduled event it answers. The history is
   * read once the workflow has closed, when it holds every event the operation will ever have, so the
   * kind is exhaustive and that read closes it.
   */
  private def historySource[Attributes](
      kind: String,
      attributes: Field[HistoryEvent, Option[Attributes]],
      operationKey: Field[HistoryEvent, Long],
      id: String
  ) =
    Evidence.history(
      id = id,
      records = kind,
      source = historySourceID,
      from = Recorded.history(attributes),
      operation = operationKey,
      commitment = Commitment.reported,
      exhaustive = true
    )

  private val historyEvents = Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
    _.getHistory.events.map(event => event)
  )

  /**
   * Every evidence kind the realization admits: the scheduled event read out of history as soon as
   * it exists, the history kinds, and the pending operation's attempt count.
   */
  private val scheduled = Evidence.read(
    id = scheduledEvidence,
    records = "nexusOperationScheduled",
    source = scheduledSource,
    from = Recorded.read(WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY, historyEvents),
    operation = Field[HistoryEvent, Long](_.eventId),
    commitment = Commitment.reported
  )
  private val pending = Evidence.read(
    id = pendingAttemptsEvidence,
    records = "pendingAttempts",
    source = describeSourceID,
    from = Recorded.read(
      WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION,
      Field[DescribeWorkflowExecutionResponse, Seq[PendingNexusOperationInfo]](
        _.pendingNexusOperations
      )
    ),
    operation = Field[PendingNexusOperationInfo, Long](_.scheduledEventId),
    commitment = Commitment.reported
  )
  private val sources = Vector(
    scheduled,
    historySource(
      "nexusOperationStarted",
      Field[HistoryEvent, Option[NexusOperationStartedEventAttributes]](
        _.attributes.nexusOperationStartedEventAttributes
      ),
      Field[HistoryEvent, Long](_.getNexusOperationStartedEventAttributes.scheduledEventId),
      startedEvidence
    ),
    historySource(
      "nexusOperationCompleted",
      Field[HistoryEvent, Option[NexusOperationCompletedEventAttributes]](
        _.attributes.nexusOperationCompletedEventAttributes
      ),
      Field[HistoryEvent, Long](_.getNexusOperationCompletedEventAttributes.scheduledEventId),
      completedEvidence
    ),
    historySource(
      "nexusOperationFailed",
      Field[HistoryEvent, Option[NexusOperationFailedEventAttributes]](
        _.attributes.nexusOperationFailedEventAttributes
      ),
      Field[HistoryEvent, Long](_.getNexusOperationFailedEventAttributes.scheduledEventId),
      failedEvidence
    ),
    historySource(
      "nexusOperationCanceled",
      Field[HistoryEvent, Option[NexusOperationCanceledEventAttributes]](
        _.attributes.nexusOperationCanceledEventAttributes
      ),
      Field[HistoryEvent, Long](_.getNexusOperationCanceledEventAttributes.scheduledEventId),
      canceledEvidence
    ),
    historySource(
      "nexusOperationTimedOut",
      Field[HistoryEvent, Option[NexusOperationTimedOutEventAttributes]](
        _.attributes.nexusOperationTimedOutEventAttributes
      ),
      Field[HistoryEvent, Long](_.getNexusOperationTimedOutEventAttributes.scheduledEventId),
      timedOutEvidence
    ),
    pending
  )

  // ### The scaffolding

  /** Each Case starts a workflow type of its own, so two Cases on one worker never share one. */
  private val workflowType = Name("umpire-", fixture = true, suffix = "-workflow")

  private def rpc[Req <: GeneratedMessage, Rsp <: GeneratedMessage](
      id: String,
      method: MethodDescriptor[Req, Rsp],
      assign: Vector[TypedAssignment[Req, ?]],
      reads: Vector[TypedResponseRead[Rsp, ?]]
  ) =
    Command(id, Instruction.rpc(workflowServiceRole, method)(assign, reads))

  private val historyAssignments = Vector(
    Assignment.typed(
      Field[GetWorkflowExecutionHistoryRequest, String](_.namespace),
      Operand.environment[String](workerNamespaceBinding)
    ),
    Assignment.typed(
      Field[GetWorkflowExecutionHistoryRequest, String](_.getExecution.workflowId),
      Operand.run()
    ),
    Assignment.typed(
      Field[GetWorkflowExecutionHistoryRequest, Int](_.maximumPageSize),
      Operand.integer(64)
    ),
    Assignment.typed(
      Field[GetWorkflowExecutionHistoryRequest, Boolean](_.waitNewEvent),
      Operand.flag(true)
    )
  )

  private val startWorkflow = rpc(
    "start-workflow",
    WorkflowServiceGrpc.METHOD_START_WORKFLOW_EXECUTION,
    Vector(
      Assignment.typed(
        Field[StartWorkflowExecutionRequest, String](_.namespace),
        Operand.environment[String](workerNamespaceBinding)
      ),
      Assignment.typed(Field[StartWorkflowExecutionRequest, String](_.workflowId), Operand.run()),
      Assignment.typed(
        Field[StartWorkflowExecutionRequest, String](_.getWorkflowType.name),
        Operand.named(workflowType)
      ),
      Assignment.typed(
        Field[StartWorkflowExecutionRequest, String](_.getTaskQueue.name),
        Operand.environment[String](taskQueueBinding)
      ),
      Assignment.typed(Field[StartWorkflowExecutionRequest, String](_.requestId), Operand.run())
    ),
    Vector.empty
  )

  /** Resolves only once the workflow closes, so a read placed after it observes the whole history. */
  private val awaitClose = rpc(
    "await-close",
    WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY,
    historyAssignments :+ Assignment.typed(
      Field[GetWorkflowExecutionHistoryRequest, HistoryEventFilterType](_.historyEventFilterType),
      Operand.enumValue(HistoryEventFilterType.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT)
    ),
    Vector.empty
  )

  /**
   * Lifts the history kinds among the resolved rules; a path that records none lifts nothing,
   * because a lift with no rule is a Case preparation rejects. It runs after the workflow closed, so
   * it is the closing read of every history kind.
   */
  private val history = Command(
    "history",
    Instruction.rpc(workflowServiceRole, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY)(
      historyAssignments,
      Vector(
        ResponseRead.typed(
          Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
            _.getHistory.events.map(event => event)
          ),
          Cardinality.each,
          Vector(Target.Observe(historyObservation), Target.Lift(correlatedObservation))
        )
      )
    ),
    closes = Vector(
      startedEvidence,
      completedEvidence,
      failedEvidence,
      canceledEvidence,
      timedOutEvidence
    )
  )

  private val pollAssignments = Vector(
    Assignment.typed(
      Field[GetWorkflowExecutionHistoryRequest, String](_.namespace),
      Operand.environment[String](workerNamespaceBinding)
    ),
    Assignment.typed(
      Field[GetWorkflowExecutionHistoryRequest, String](_.getExecution.workflowId),
      Operand.run()
    )
  )

  private val describeAssignments = Vector(
    Assignment.typed(
      Field[DescribeWorkflowExecutionRequest, String](_.namespace),
      Operand.environment[String](workerNamespaceBinding)
    ),
    Assignment.typed(
      Field[DescribeWorkflowExecutionRequest, String](_.getExecution.workflowId),
      Operand.run()
    )
  )

  /** Polls the pending operation until its first attempt has failed. */
  private val pendingAttempts = Command(
    "pending-attempts",
    Instruction.poll(pending, workflowServiceRole)(
      describeAssignments,
      Condition.equal(Field[PendingNexusOperationInfo, Int](_.attempt), Operand.integer(1)),
      250
    )
  )

  /** Polls the history for the scheduled event, run until the event exists. */
  private val awaitScheduled = Command(
    "await-scheduled",
    Instruction.poll(scheduled, workflowServiceRole)(
      pollAssignments,
      Condition.present(
        Field[HistoryEvent, Option[NexusOperationScheduledEventAttributes]](
          _.attributes.nexusOperationScheduledEventAttributes
        )
      ),
      250
    )
  )

  /** The handle an asynchronous reply publishes and a completion reads. */
  private val completionAuthority = "completion-authority"

  /** A JSON payload of one string, as the SDK's default data converter encodes it. */
  private def textPayload(value: String) =
    Proto[Payload](
      ProtoField.typed(
        Field[Payload, Map[String, com.google.protobuf.ByteString]](_.metadata),
        ProtoValue.mapping(ProtoEntry.typed("encoding", ProtoValue.utf8("json/plain")))
      ),
      ProtoField.typed(
        Field[Payload, com.google.protobuf.ByteString](_.data),
        ProtoValue.utf8("\"" + value + "\"")
      )
    )

  /** The failure a failed reply or completion carries. */
  private val handlerFailure = Proto[ApiFailure](
    ProtoField.typed(Field[ApiFailure, String](_.message), ProtoValue.text("operation failed")),
    ProtoField.typed(
      Field[ApiFailure, ApplicationFailureInfo](_.getApplicationFailureInfo),
      ProtoValue.message(
        Proto[ApplicationFailureInfo](
          ProtoField.typed(
            Field[ApplicationFailureInfo, String](_.`type`),
            ProtoValue.text("OperationFailed")
          ),
          ProtoField.typed(
            Field[ApplicationFailureInfo, Boolean](_.nonRetryable),
            ProtoValue.flag(true)
          )
        )
      )
    )
  )

  /** The durations the deadlines a path sets realize as; the backoff is the server's own. */
  private val deadline =
    ProtoValue.message(
      Proto[com.google.protobuf.duration.Duration](
        ProtoField.typed(
          Field[com.google.protobuf.duration.Duration, Long](_.seconds),
          ProtoValue.number(2L)
        )
      )
    )

  // ### The bindings
  //
  // Each binding names the class it performs and the command that performs it, so the same class
  // performed twice on one path produces two distinct commands.

  private val startNexusOperation = "start-nexus-operation"

  private val scheduleAttributes = Vector(
    ProtoField.typed(
      Field[ScheduleNexusOperationCommandAttributes, String](_.endpoint),
      ProtoValue.roleId(nexusEndpointRole)
    ),
    ProtoField.typed(
      Field[ScheduleNexusOperationCommandAttributes, String](_.service),
      ProtoValue.text(service)
    ),
    ProtoField.typed(
      Field[ScheduleNexusOperationCommandAttributes, String](_.operation),
      ProtoValue.text(operation)
    ),
    ProtoField.typed(
      Field[ScheduleNexusOperationCommandAttributes, Payload](_.getInput),
      ProtoValue.message(textPayload("request"))
    )
  )

  /**
   * The schedule command for one class of the schedule action: the deadlines the class sets, at the
   * realization's durations.
   */
  private def scheduleBinding(
      step: ClassRef,
      deadlines: Vector[TypedProtoField[ScheduleNexusOperationCommandAttributes, ?]]
  ) =
    Performance(
      step,
      Command(
        startNexusOperation,
        WorkflowCommand(
          Proto[ApiCommand](
            ProtoField.typed(
              Field[ApiCommand, CommandType](_.commandType),
              ProtoValue.enumValue(CommandType.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION)
            ),
            ProtoField.typed(
              Field[ApiCommand, ScheduleNexusOperationCommandAttributes](
                _.getScheduleNexusOperationCommandAttributes
              ),
              ProtoValue.message(
                Proto[ScheduleNexusOperationCommandAttributes](
                  (scheduleAttributes ++ deadlines)*
                )
              )
            )
          )
        )
      )
    )

  private def replyBinding(step: ClassRef, id: String, reply: TypedProto[?], binds: String) =
    Performance(step, Command(id, NexusReply(reply, binds), timeoutMs = 5000))

  private def response(variant: TypedProtoField[StartOperationResponse, ?]) =
    Proto[StartOperationResponse](variant)

  private def handlerError(errorType: String, behavior: NexusHandlerErrorRetryBehavior) =
    Proto[HandlerError](
      ProtoField.typed(Field[HandlerError, String](_.errorType), ProtoValue.text(errorType)),
      ProtoField.typed(
        Field[HandlerError, NexusFailure](_.getFailure),
        ProtoValue.message(
          Proto[NexusFailure](
            ProtoField
              .typed(Field[NexusFailure, String](_.message), ProtoValue.text("handler error"))
          )
        )
      ),
      ProtoField.typed(
        Field[HandlerError, NexusHandlerErrorRetryBehavior](_.retryBehavior),
        ProtoValue.enumValue(behavior)
      )
    )

  private def completion(step: ClassRef, id: String, result: TypedProto[?]) =
    Performance(step, Command(id, NexusCompletion(completionAuthority, result)))

  // ### The plan
  //
  // The controller's sequence is the one place the interleaving matters: it stops the handler's
  // worker when the path says so, starts the workflow, reads the scheduled event as soon as it
  // exists, polls the attempt count when a retryable failure is on the path, waits for the authority
  // the handler publishes when a completion is on the path, performs the completion, waits for the
  // workflow to close, and only then reads history.

  private def controller(extra: Vector[Item]) = Script(
    "controller",
    Activation.Controller,
    Vector(
      // The handler's worker stops polling its own queue, so the caller workflow keeps running.
      Item(performs =
        Vector(
          Performance(
            workerStop,
            Command("stop-handler-worker", Fault(handlerTaskQueueRole, FaultKind.workerStop))
          )
        )
      ),
      Item(command = Some(startWorkflow)),
      Item(command = Some(awaitScheduled))
    ) ++ extra ++ Vector(
      Item(command = Some(pendingAttempts), when = Vector(handlerReply(Reply.handlerError(true)))),
      Item(
        command = Some(Command("await-completion-authority", AwaitLearned(completionAuthority))),
        when = Vector(complete(Resolution.succeeded), complete(Resolution.failed))
      ),
      Item(performs =
        Vector(
          completion(
            complete(Resolution.succeeded),
            "complete-nexus-operation",
            textPayload("completed")
          ),
          completion(complete(Resolution.failed), "fail-nexus-operation", handlerFailure)
        )
      ),
      Item(command = Some(awaitClose)),
      Item(command = Some(history))
    )
  )

  private val workflowScript = Script(
    "workflow",
    Activation.Workflow(workflowType, workerRole, taskQueueRole),
    Vector(
      // The schedule command once per class of deadline a path of the caller Model sets.
      Item(performs =
        Vector(
          scheduleBinding(schedule(unset, unset, unset), Vector.empty),
          scheduleBinding(
            schedule(unset, expires, unset),
            Vector(
              ProtoField.typed(
                Field[
                  ScheduleNexusOperationCommandAttributes,
                  com.google.protobuf.duration.Duration
                ](_.getScheduleToStartTimeout),
                deadline
              )
            )
          ),
          scheduleBinding(
            schedule(unset, unset, expires),
            Vector(
              ProtoField.typed(
                Field[
                  ScheduleNexusOperationCommandAttributes,
                  com.google.protobuf.duration.Duration
                ](_.getStartToCloseTimeout),
                deadline
              )
            )
          )
        )
      ),
      Item(
        command = Some(
          Command("await-nexus-operation", AwaitCommand(startNexusOperation), regardless = true)
        ),
        when = Vector(
          schedule(unset, unset, unset),
          schedule(unset, expires, unset),
          schedule(unset, unset, expires)
        )
      ),
      // The workflow closes on every path: a failed or timed-out operation is the await's recorded
      // outcome, not a reason to leave the workflow open.
      Item(command =
        Some(
          Command(
            "finish-workflow",
            Finish(Literal(Text("done"))),
            timeoutMs = 5000,
            regardless = true
          )
        )
      )
    )
  )

  private val handlerScript = Script(
    "handler",
    Activation.NexusHandler(service, operation, workerRole, handlerTaskQueueRole),
    Vector(
      Item(performs =
        Vector(
          replyBinding(
            handlerReply(Reply.async),
            "respond-async",
            response(
              ProtoField.typed(
                Field[StartOperationResponse, StartOperationResponse.Async](_.getAsyncSuccess),
                ProtoValue.message(Proto[StartOperationResponse.Async]())
              )
            ),
            completionAuthority
          ),
          replyBinding(
            handlerReply(Reply.syncSuccess),
            "respond-sync",
            response(
              ProtoField.typed(
                Field[StartOperationResponse, StartOperationResponse.Sync](_.getSyncSuccess),
                ProtoValue.message(
                  Proto[StartOperationResponse.Sync](
                    ProtoField.typed(
                      Field[StartOperationResponse.Sync, Payload](_.getPayload),
                      ProtoValue.message(textPayload("completed"))
                    )
                  )
                )
              )
            ),
            ""
          ),
          replyBinding(
            handlerReply(Reply.operationFailed),
            "respond-failed",
            response(
              ProtoField.typed(
                Field[StartOperationResponse, ApiFailure](_.getFailure),
                ProtoValue.message(handlerFailure)
              )
            ),
            ""
          ),
          replyBinding(
            handlerReply(Reply.handlerError(true)),
            "respond-error-retryable",
            handlerError(
              "INTERNAL",
              NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE
            ),
            ""
          ),
          replyBinding(
            handlerReply(Reply.handlerError(false)),
            "respond-error",
            handlerError(
              "BAD_REQUEST",
              NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE
            ),
            ""
          )
        )
      )
    )
  )

  /**
   * One Nexus operation scheduled by a controller-started workflow and answered by a handler inside
   * the Case's own worker.
   */
  private def realization(
      name: String,
      machine: Machine[ProtocolState, temporal.nexuscaller.Outcome, ProtocolFact],
      extra: Vector[Item]
  ): Realization = Realization(
    name = name,
    machine = machine,
    producer = "temporal.nexus.caller.testpilot",
    producerVersion = "1",
    roles = Vector(
      Role(workflowServiceRole, RoleKind.endpoint),
      Role(workerRole, RoleKind.worker, namespace = workerNamespaceBinding),
      Role(
        taskQueueRole,
        RoleKind.taskQueue,
        namespace = workerNamespaceBinding,
        resource = taskQueueBinding
      ),
      Role(
        handlerTaskQueueRole,
        RoleKind.taskQueue,
        namespace = workerNamespaceBinding,
        resource = handlerTaskQueueBinding
      ),
      Role(nexusEndpointRole, RoleKind.endpoint, resource = nexusEndpointBinding)
    ),
    correlation = Correlation(
      projection = projectionID,
      run = runFieldID,
      operation = operationFieldID,
      observation = correlatedObservation,
      events = 32,
      buffered = 16,
      keys = 8,
      support = 128,
      work = 1000000000,
      eventSize = 512
    ),
    scripts = Vector(controller(extra), workflowScript, handlerScript),
    learned = Vector(Learned(completionAuthority, LearnedKind.handle)),
    observations = Vector(
      Observed[HistoryEvent](historyObservation),
      Observed[CorrelatedEvidence](correlatedObservation)
    ),
    evidence = sources,
    cleanup = "cleanup"
  )

  val asyncNexus: Realization = realization("asyncNexus", nexusProtocol, Vector.empty)

  val forgedCompletion: Realization = realization(
    "forgedCompletion",
    temporal.nexuscaller.Control.forged,
    Vector(
      Item(performs =
        Vector(
          Performance(
            temporal.nexuscaller.Control.inspect,
            rpc(
              "inspect-workflow",
              WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION,
              describeAssignments,
              Vector.empty
            )
          )
        )
      )
    )
  )
