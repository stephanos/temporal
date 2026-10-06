// The Nexus caller-side realization: what a Case does to a deployment to take the path a Query
// found.
//
// A controller-started workflow schedules one Nexus operation on the Case's endpoint role; the
// handler answers it, and the controller completes it when the handler answered asynchronously; the
// controller reads the history the Case's evidence is lifted from. The realization writes no
// Program: it declares that scaffolding once and binds each action class of a Model to the
// instruction that performs it, so the producer puts those instructions where a Query's path took
// them.
//
// The roles, bindings, correlation window, correlated record and run records are the Temporal kit's
// (temporal/realize). Everything below is a declaration the lifter emits into the IR, each referring
// to the others by value; Go lowers a Query's witness through it into a Testpilot Case
// (tools/umpire/lower).
package temporal
package features.nexus
package workflow

import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.{AwaitCommand, AwaitLearned}
import temporal.realize.WorkerInstruction.{Fault, NexusCompletion, NexusReply, WorkflowCommand}
import temporal.realize.{
  applicationFailure,
  await,
  caseWorker,
  controller,
  correlated,
  deadlineMs,
  deadlineSeconds,
  duration,
  evidenceId,
  field,
  finish,
  handlerTaskQueue,
  jsonPayload,
  nexusEndpoint,
  perCase,
  proto,
  read,
  run,
  sourceId,
  taskQueue,
  taskQueueName,
  temporalRealization,
  workerNamespace,
  workflowService,
  CauseKind,
  FaultKind,
  ProtoScope,
  ServerStep,
  WorkerActivation,
  WorkflowHistory
}
import io.temporal.api.workflowservice.v1.*
import io.temporal.api.history.v1.*
import io.temporal.api.command.v1.{Command as ApiCommand, ScheduleNexusOperationCommandAttributes}
import io.temporal.api.enums.v1.{
  CommandType,
  HistoryEventFilterType,
  NexusHandlerErrorRetryBehavior
}
import io.temporal.api.nexus.v1.{HandlerError, StartOperationResponse}
import temporal.shared.worker.worker

import Timeout.expires
import system.{NexusSystem, TrustingCaller}

object NexusRealization:
  // ### Evidence
  //
  // The scheduled event is read out of history as soon as it exists, the other events of the
  // operation once the workflow has closed, and the attempt count from the pending operation.

  // Every event the history command reads, recorded as the protobuf it is.
  private val historyEvent = Observed[HistoryEvent]("history-event")

  private val historyEvents = Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
    _.getHistory.events.map(event => event)
  )

  private val scheduled = Evidence.read(
    id = evidenceId("scheduled"),
    records = system.Fact.nexusOperationScheduled,
    source = sourceId("scheduled"),
    from = Recorded.read(WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY, historyEvents),
    operation = Field(_.eventId),
    commitment = Commitment.reported
  )

  // One history event kind of the operation, keyed by the scheduled event it answers. The history is
  // read once the workflow has closed, when it holds every event the operation will ever have, so the
  // kind is exhaustive and that read closes it.
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
    system.Fact.nexusOperationStarted,
    Field(_.attributes.nexusOperationStartedEventAttributes),
    Field(_.getNexusOperationStartedEventAttributes.scheduledEventId)
  )
  private val completed = historyKind(
    "completed",
    system.Fact.nexusOperationCompleted,
    Field(_.attributes.nexusOperationCompletedEventAttributes),
    Field(_.getNexusOperationCompletedEventAttributes.scheduledEventId)
  )
  private val failed = historyKind(
    "failed",
    system.Fact.nexusOperationFailed,
    Field(_.attributes.nexusOperationFailedEventAttributes),
    Field(_.getNexusOperationFailedEventAttributes.scheduledEventId)
  )
  private val canceled = historyKind(
    "canceled",
    system.Fact.nexusOperationCanceled,
    Field(_.attributes.nexusOperationCanceledEventAttributes),
    Field(_.getNexusOperationCanceledEventAttributes.scheduledEventId)
  )
  private val timedOut = historyKind(
    "timedOut",
    system.Fact.nexusOperationTimedOut,
    Field(_.attributes.nexusOperationTimedOutEventAttributes),
    Field(_.getNexusOperationTimedOutEventAttributes.scheduledEventId)
  )

  private val pending = Evidence.read(
    id = evidenceId(system.Fact.pendingAttempts),
    records = system.Fact.pendingAttempts,
    source = sourceId("describe"),
    from = Recorded.read(
      WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION,
      Field(_.pendingNexusOperations)
    ),
    operation = Field(_.scheduledEventId),
    commitment = Commitment.reported
  )

  // ### The controller's commands

  // The handler's worker stops polling its own queue, so the caller workflow keeps running.
  private val stopHandlerWorker = Fault(handlerTaskQueue, FaultKind.workerStop)

  // Each Case starts a workflow type of its own, so two Cases on one worker never share one.
  private val workflowType = perCase("workflow")

  private val startWorkflow =
    rpc(workflowService, WorkflowServiceGrpc.METHOD_START_WORKFLOW_EXECUTION) {
      field(_.namespace) := workerNamespace
      field(_.workflowId) := run
      field(_.getWorkflowType.name) := Operand.named(workflowType)
      field(_.getTaskQueue.name) := taskQueueName
      field(_.requestId) := run
    }

  // Polls the history for the scheduled event, run until the event exists.
  private val awaitScheduled = await(scheduled, workflowService)(
    Condition.present(Field(_.attributes.nexusOperationScheduledEventAttributes))
  ) {
    field(_.namespace) := workerNamespace
    field(_.getExecution.workflowId) := run
  }

  // Polls the pending operation until its first attempt has failed.
  private val pendingAttempts = await(pending, workflowService)(
    Condition.equal(Field(_.attempt), Operand.integer(1))
  ) {
    field(_.namespace) := workerNamespace
    field(_.getExecution.workflowId) := run
  }

  // The handle an asynchronous reply publishes and a completion reads.
  private val completionAuthority = Learned("completion-authority", LearnedKind.handle)

  private val awaitCompletionAuthority = AwaitLearned(completionAuthority.id)

  // The failure a failed reply or completion carries.
  private val handlerFailure =
    applicationFailure("OperationFailed", "operation failed", retryable = false)

  private val completeNexusOperation =
    NexusCompletion(completionAuthority.id, jsonPayload("completed"))
  private val failNexusOperation = NexusCompletion(completionAuthority.id, handlerFailure)

  // The workflow's history, waiting for new events: the request both history calls extend.
  private val historyRead =
    rpc(workflowService, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY) {
      field(_.namespace) := workerNamespace
      field(_.getExecution.workflowId) := run
      field(_.maximumPageSize) := Operand.integer(64)
      field(_.waitNewEvent) := Operand.flag(true)
    }

  // Resolves only once the workflow closes, so a read placed after it observes the whole history.
  private val awaitClose = historyRead.extended {
    field(_.historyEventFilterType) :=
      Operand.enumValue(HistoryEventFilterType.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT)
  }

  // Lifts the history kinds among the resolved rules; a path that records none lifts nothing,
  // because a lift with no rule is a Case preparation rejects. It runs after the workflow closed, so
  // it is the closing read of every history kind.
  private val history = command(
    historyRead.extended {
      read(historyEvents, Cardinality.each).into(historyEvent, Target.Lift(correlated.id))
    },
    closes = Vector(started, completed, failed, canceled, timedOut)
  )

  // Only the forged control's path inspects the workflow.
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

  // The service and operation the handler script answers, which the schedule command names.
  private val service = "umpire.case.service"
  private val operation = "complete"

  // The schedule command, setting the deadlines one class of the schedule action sets. Every class
  // performs the one command, so the await names one: its id is written out, and each class's
  // deadlines are written where the class binds the command.
  private def scheduling(deadlines: ProtoScope[ScheduleNexusOperationCommandAttributes] ?=> Unit) =
    Command(
      "start-nexus-operation",
      WorkflowCommand(proto[ApiCommand] {
        field(_.commandType) := CommandType.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION
        field(_.getScheduleNexusOperationCommandAttributes) {
          field(_.endpoint) := nexusEndpoint
          field(_.service) := service
          field(_.operation) := operation
          field(_.getInput) := jsonPayload("request")
          deadlines
        }
      })
    )

  private val startNexusOperation = scheduling(())

  // The duration a deadline a path sets realizes as; the backoff is the server's own.
  private val requestDeadline = duration(deadlineSeconds)

  private val awaitNexusOperation =
    command(AwaitCommand(startNexusOperation.id), regardless = true)

  // The workflow closes on every path: a failed or timed-out operation is the await's recorded
  // outcome, not a reason to leave the workflow open.
  private val finishWorkflow = command(finish("done"), regardless = true)

  private val workflowScript =
    script("workflow", WorkerActivation.Workflow(workflowType, caseWorker, taskQueue))(
      // The schedule command once per class of deadline a path of the caller Model sets.
      perform(
        caller.schedule() -> startNexusOperation,
        caller.schedule(scheduleToStart := expires) -> scheduling(
          field(_.getScheduleToStartTimeout) := requestDeadline
        ),
        caller.schedule(startToClose := expires) -> scheduling(
          field(_.getStartToCloseTimeout) := requestDeadline
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

  private def handlerError(errorType: String, behavior: NexusHandlerErrorRetryBehavior) =
    proto[HandlerError] {
      field(_.errorType) := errorType
      field(_.getFailure)(field(_.message) := "handler error")
      field(_.retryBehavior) := behavior
    }

  private val respondAsync = NexusReply(
    proto[StartOperationResponse](field(_.getAsyncSuccess) {}),
    completionAuthority.id
  )
  private val respondSync = NexusReply(proto[StartOperationResponse] {
    field(_.getSyncSuccess)(field(_.getPayload) := jsonPayload("completed"))
  })
  private val respondFailed =
    NexusReply(proto[StartOperationResponse](field(_.getFailure) := handlerFailure))
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

  // One Nexus operation scheduled by a controller-started workflow and answered by a handler inside
  // the Case's own worker, named after the val that declares it. `steps` are the forged control's
  // own, after the scheduled event.
  private def realization(
      machine: Machine[
        system.State,
        temporal.features.nexus.Outcome,
        system.Fact
      ],
      steps: Item*
  ) = temporalRealization(
    machine = machine,
    operation = features.nexus.workflow.operation,
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
