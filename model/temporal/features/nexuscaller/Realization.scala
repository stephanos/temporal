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
 * The roles, bindings, correlation window, correlated record and run records are the Temporal kit's
 * (temporal/realize). Everything below is a declaration the lifter emits into the IR, each referring
 * to the others by value; Go lowers a Query's witness through it into a Testpilot Case
 * (tools/umpire/lower).
 */
package temporal
package features.nexuscaller

import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.{AwaitCommand, AwaitLearned, Finish}
import temporal.realize.WorkerInstruction.{Fault, NexusCompletion, NexusReply, WorkflowCommand}
import temporal.realize.{
  await,
  caseWorker,
  controller,
  correlated,
  deadlineMs,
  deadlineSeconds,
  evidenceId,
  field,
  handlerTaskQueue,
  nexusEndpoint,
  perCase,
  run,
  sourceId,
  taskQueue,
  taskQueueName,
  temporalRealization,
  workerNamespace,
  workflowService,
  CauseKind,
  FaultKind,
  ServerStep,
  WorkerActivation,
  WorkflowHistory
}
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
import com.google.protobuf.duration.Duration
import shared.worker.worker

import Timeout.expires
import system.{NexusSystem, TrustingCaller}

object NexusRealization:
  // ### Evidence
  //
  // The scheduled event is read out of history as soon as it exists, the other events of the
  // operation once the workflow has closed, and the attempt count from the pending operation.

  /** Every event the history command reads, recorded as the protobuf it is. */
  private val historyEvent = Observed[HistoryEvent]("history-event")

  private val historyEvents = Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
    _.getHistory.events.map(event => event)
  )

  private val scheduled = Evidence.read(
    id = evidenceId("scheduled"),
    records = SystemFact.nexusOperationScheduled,
    source = sourceId("scheduled"),
    from = Recorded.read(WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY, historyEvents),
    operation = Field[HistoryEvent, Long](_.eventId),
    commitment = Commitment.reported
  )

  /**
   * One history event kind of the operation, keyed by the scheduled event it answers. The history is
   * read once the workflow has closed, when it holds every event the operation will ever have, so the
   * kind is exhaustive and that read closes it.
   */
  private def historyKind[Attributes](
      kind: String,
      records: Fact,
      attributes: Field[HistoryEvent, Option[Attributes]],
      operation: Field[HistoryEvent, Long]
  ) = Evidence.keyed(
    id = evidenceId(kind),
    records = records,
    source = sourceId("history"),
    from = WorkflowHistory.event(attributes),
    operation = operation,
    commitment = Commitment.reported,
    exhaustive = true
  )

  private val started = historyKind(
    "started",
    SystemFact.nexusOperationStarted,
    Field(_.attributes.nexusOperationStartedEventAttributes),
    Field(_.getNexusOperationStartedEventAttributes.scheduledEventId)
  )
  private val completed = historyKind(
    "completed",
    SystemFact.nexusOperationCompleted,
    Field(_.attributes.nexusOperationCompletedEventAttributes),
    Field(_.getNexusOperationCompletedEventAttributes.scheduledEventId)
  )
  private val failed = historyKind(
    "failed",
    SystemFact.nexusOperationFailed,
    Field(_.attributes.nexusOperationFailedEventAttributes),
    Field(_.getNexusOperationFailedEventAttributes.scheduledEventId)
  )
  private val canceled = historyKind(
    "canceled",
    SystemFact.nexusOperationCanceled,
    Field(_.attributes.nexusOperationCanceledEventAttributes),
    Field(_.getNexusOperationCanceledEventAttributes.scheduledEventId)
  )
  private val timedOut = historyKind(
    "timedOut",
    SystemFact.nexusOperationTimedOut,
    Field(_.attributes.nexusOperationTimedOutEventAttributes),
    Field(_.getNexusOperationTimedOutEventAttributes.scheduledEventId)
  )

  private val pending = Evidence.read(
    id = evidenceId(SystemFact.pendingAttempts),
    records = SystemFact.pendingAttempts,
    source = sourceId("describe"),
    from = Recorded.read(
      WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION,
      Field[DescribeWorkflowExecutionResponse, Seq[PendingNexusOperationInfo]](
        _.pendingNexusOperations
      )
    ),
    operation = Field[PendingNexusOperationInfo, Long](_.scheduledEventId),
    commitment = Commitment.reported
  )

  // ### The controller's commands

  /** The handler's worker stops polling its own queue, so the caller workflow keeps running. */
  private val stopHandlerWorker = Fault(handlerTaskQueue, FaultKind.workerStop)

  /** Each Case starts a workflow type of its own, so two Cases on one worker never share one. */
  private val workflowType = perCase("workflow")

  private val startWorkflow =
    rpc(workflowService, WorkflowServiceGrpc.METHOD_START_WORKFLOW_EXECUTION) {
      field(_.namespace) := workerNamespace
      field(_.workflowId) := run
      field(_.getWorkflowType.name) := Operand.named(workflowType)
      field(_.getTaskQueue.name) := taskQueueName
      field(_.requestId) := run
    }

  /** Polls the history for the scheduled event, run until the event exists. */
  private val awaitScheduled = await(scheduled, workflowService)(
    Condition.present(
      Field[HistoryEvent, Option[NexusOperationScheduledEventAttributes]](
        _.attributes.nexusOperationScheduledEventAttributes
      )
    )
  ) {
    field(_.namespace) := workerNamespace
    field(_.getExecution.workflowId) := run
  }

  /** Polls the pending operation until its first attempt has failed. */
  private val pendingAttempts = await(pending, workflowService)(
    Condition.equal(Field[PendingNexusOperationInfo, Int](_.attempt), Operand.integer(1))
  ) {
    field(_.namespace) := workerNamespace
    field(_.getExecution.workflowId) := run
  }

  /** The handle an asynchronous reply publishes and a completion reads. */
  private val completionAuthority = Learned("completion-authority", LearnedKind.handle)

  private val awaitCompletionAuthority = AwaitLearned(completionAuthority.id)

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

  private val completeNexusOperation =
    NexusCompletion(completionAuthority.id, textPayload("completed"))
  private val failNexusOperation = NexusCompletion(completionAuthority.id, handlerFailure)

  /** Resolves only once the workflow closes, so a read placed after it observes the whole history. */
  private val awaitClose =
    rpc(workflowService, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY) {
      field(_.namespace) := workerNamespace
      field(_.getExecution.workflowId) := run
      field(_.maximumPageSize) := Operand.integer(64)
      field(_.waitNewEvent) := Operand.flag(true)
      field(_.historyEventFilterType) :=
        Operand.enumValue(HistoryEventFilterType.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT)
    }

  /**
   * Lifts the history kinds among the resolved rules; a path that records none lifts nothing,
   * because a lift with no rule is a Case preparation rejects. It runs after the workflow closed, so
   * it is the closing read of every history kind. A call's scope assigns its request and reads
   * nothing back, so this call, which reads its response, is written in the core form.
   */
  private val history = command(
    Instruction.rpc(workflowService, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY)(
      Vector(
        Assignment.typed(
          Field[GetWorkflowExecutionHistoryRequest, String](_.namespace),
          workerNamespace
        ),
        Assignment.typed(
          Field[GetWorkflowExecutionHistoryRequest, String](_.getExecution.workflowId),
          run
        ),
        Assignment.typed(
          Field[GetWorkflowExecutionHistoryRequest, Int](_.maximumPageSize),
          Operand.integer(64)
        ),
        Assignment.typed(
          Field[GetWorkflowExecutionHistoryRequest, Boolean](_.waitNewEvent),
          Operand.flag(true)
        )
      ),
      Vector(
        ResponseRead.typed(
          historyEvents,
          Cardinality.each,
          Vector(Target.Observe(historyEvent.id), Target.Lift(correlated.id))
        )
      )
    ),
    closes = Vector(started, completed, failed, canceled, timedOut)
  )

  /** Only the forged control's path inspects the workflow. */
  private val inspectWorkflow =
    rpc(workflowService, WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION) {
      field(_.namespace) := workerNamespace
      field(_.getExecution.workflowId) := run
    }

  // ### The controller
  //
  // The controller's sequence is the one place the interleaving matters: it stops the handler's
  // worker when the path says so, starts the workflow, reads the scheduled event as soon as it
  // exists, polls the attempt count when a retryable failure is on the path, waits for the authority
  // the handler publishes when a completion is on the path, performs the completion, waits for the
  // workflow to close, and only then reads history.

  private def callerController(steps: Item*) = controller(
    (Vector(
      perform(worker.stop -> stopHandlerWorker),
      everyCase(startWorkflow),
      everyCase(awaitScheduled)
    ) ++ steps ++ Vector(
      onPath(handler.reply(Reply.handlerError(true)))(pendingAttempts),
      onPath(handler.complete(Resolution.succeeded), handler.complete(Resolution.failed))(
        awaitCompletionAuthority
      ),
      perform(
        handler.complete(Resolution.succeeded) -> completeNexusOperation,
        handler.complete(Resolution.failed) -> failNexusOperation
      ),
      everyCase(awaitClose),
      everyCase(history)
    ))*
  )

  // ### The workflow

  /** The service and operation the handler script answers, which the schedule command names. */
  private val service = "umpire.case.service"
  private val operation = "complete"

  private val scheduleAttributes = Vector(
    ProtoField.typed(
      Field[ScheduleNexusOperationCommandAttributes, String](_.endpoint),
      ProtoValue.roleId(nexusEndpoint)
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
   * The schedule command, setting the deadlines one class of the schedule action sets. Every class
   * performs the one command, so the await names one: its id is written out, and each class's
   * deadlines are written where the class binds the command.
   */
  private def scheduling(deadlines: TypedProtoField[ScheduleNexusOperationCommandAttributes, ?]*) =
    Command(
      "start-nexus-operation",
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
              Proto[ScheduleNexusOperationCommandAttributes]((scheduleAttributes ++ deadlines)*)
            )
          )
        )
      )
    )

  private val startNexusOperation = scheduling()

  /** The duration a deadline a path sets realizes as; the backoff is the server's own. */
  private val requestDeadline =
    ProtoValue.message(
      Proto[Duration](
        ProtoField.typed(Field[Duration, Long](_.seconds), ProtoValue.number(deadlineSeconds))
      )
    )

  private val awaitNexusOperation =
    command(AwaitCommand(startNexusOperation.id), regardless = true)

  // The workflow closes on every path: a failed or timed-out operation is the await's recorded
  // outcome, not a reason to leave the workflow open.
  private val finishWorkflow =
    command(Finish(Operand.Literal(ProtoValue.Text("done"))), regardless = true)

  private val workflowScript =
    script("workflow", WorkerActivation.Workflow(workflowType, caseWorker, taskQueue))(
      // The schedule command once per class of deadline a path of the caller Model sets.
      perform(
        caller.schedule() -> startNexusOperation,
        caller.schedule(scheduleToStart := expires) -> scheduling(
          ProtoField.typed(
            Field[ScheduleNexusOperationCommandAttributes, Duration](_.getScheduleToStartTimeout),
            requestDeadline
          )
        ),
        caller.schedule(startToClose := expires) -> scheduling(
          ProtoField.typed(
            Field[ScheduleNexusOperationCommandAttributes, Duration](_.getStartToCloseTimeout),
            requestDeadline
          )
        )
      ),
      onPath(
        caller.schedule(),
        caller.schedule(scheduleToStart := expires),
        caller.schedule(startToClose := expires)
      )(awaitNexusOperation),
      everyCase(finishWorkflow)
    )

  // ### The handler

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

  private val respondAsync = NexusReply(
    response(
      ProtoField.typed(
        Field[StartOperationResponse, StartOperationResponse.Async](_.getAsyncSuccess),
        ProtoValue.message(Proto[StartOperationResponse.Async]())
      )
    ),
    completionAuthority.id
  )
  private val respondSync = NexusReply(
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
    )
  )
  private val respondFailed = NexusReply(
    response(
      ProtoField.typed(
        Field[StartOperationResponse, ApiFailure](_.getFailure),
        ProtoValue.message(handlerFailure)
      )
    )
  )
  private val respondErrorRetryable = NexusReply(
    handlerError(
      "INTERNAL",
      NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE
    )
  )
  private val respondError = NexusReply(
    handlerError(
      "BAD_REQUEST",
      NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE
    )
  )

  private val handlerScript =
    script(
      "handler",
      WorkerActivation.NexusHandler(service, operation, caseWorker, handlerTaskQueue)
    )(
      perform(
        handler.reply(Reply.async) -> respondAsync,
        handler.reply(Reply.syncSuccess) -> respondSync,
        handler.reply(Reply.operationFailed) -> respondFailed,
        handler.reply(Reply.handlerError(true)) -> respondErrorRetryable,
        handler.reply(Reply.handlerError(false)) -> respondError
      )
    )

  /**
   * One Nexus operation scheduled by a controller-started workflow and answered by a handler inside
   * the Case's own worker, named after the val that declares it. `steps` are the forged control's
   * own, after the scheduled event.
   */
  private def realization(
      machine: Machine[SystemState, temporal.features.nexuscaller.Outcome, SystemFact],
      steps: Item*
  ) = temporalRealization(
    machine = machine,
    operation = features.nexuscaller.operation,
    roles = Vector(workflowService, caseWorker, taskQueue, handlerTaskQueue, nexusEndpoint),
    scripts = Vector(callerController(steps*), workflowScript, handlerScript),
    evidence = Vector(scheduled, started, completed, failed, canceled, timedOut, pending),
    learned = Vector(completionAuthority),
    observations = Vector(historyEvent, correlated),
    // A timeout class fires at the deadline its schedule command sets. No command sets a
    // schedule-to-close deadline, so no path waits for that class.
    serverSteps = Vector(
      ServerStep(deadline.scheduleToStart, CauseKind.timer, deadlineMs),
      ServerStep(deadline.startToClose, CauseKind.timer, deadlineMs)
    )
  )

  val asyncNexus = realization(NexusSystem)

  val forgedCompletion =
    realization(TrustingCaller, perform(caller.inspect -> inspectWorkflow))
