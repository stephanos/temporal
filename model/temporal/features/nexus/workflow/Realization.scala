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
import temporal.realize.*
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

import system.{NexusSystem, TrustingCaller}

// The declarations the Nexus caller realizations share.
private object CallerDeclarations:
  // ### Evidence
  //
  // The scheduled event is read out of history as soon as it exists, the other events of the
  // operation once the workflow has closed, and the attempt count from the pending operation.

  // Every event the history command reads, recorded as the protobuf it is.
  val historyEvent = Observed[HistoryEvent]("history-event")

  val historyEvents = Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
    _.getHistory.events.map(event => event)
  )

  // The kind the scheduled event is, and the source it counts in, of its own.
  val scheduledKind = "scheduled"

  // The source the pending operation's attempt count counts in.
  val describeSource = "describe"

  val scheduled = Evidence.read(
    id = evidenceId(scheduledKind),
    records = system.Fact.nexusOperationScheduled,
    source = sourceId(scheduledKind),
    from = Recorded.read(WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY, historyEvents),
    operation = Field(_.eventId),
    commitment = Commitment.reported
  )

  // The operation's history events, each keyed by the scheduled event it answers.
  val historyKinds =
    HistoryEvidence(key = "scheduled_event_id", factPrefix = "nexusOperation")(
      HistoryKind(
        system.Fact.nexusOperationStarted,
        _.attributes.nexusOperationStartedEventAttributes
      ),
      HistoryKind(
        system.Fact.nexusOperationCompleted,
        _.attributes.nexusOperationCompletedEventAttributes
      ),
      HistoryKind(
        system.Fact.nexusOperationFailed,
        _.attributes.nexusOperationFailedEventAttributes
      ),
      HistoryKind(
        system.Fact.nexusOperationCanceled,
        _.attributes.nexusOperationCanceledEventAttributes
      ),
      HistoryKind(
        system.Fact.nexusOperationTimedOut,
        _.attributes.nexusOperationTimedOutEventAttributes
      )
    )

  val pending = Evidence.read(
    id = evidenceId(system.Fact.pendingAttempts),
    records = system.Fact.pendingAttempts,
    source = sourceId(describeSource),
    from = Recorded.read(
      WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION,
      Field(_.pendingNexusOperations)
    ),
    operation = Field(_.scheduledEventId),
    commitment = Commitment.reported
  )

  // ### The controller's commands

  // Every call of the controller is made on the WorkflowService, in the run's namespace, of the
  // workflow the run started under its own id.
  val calls =
    RequestBase(workflowService, "namespace" -> workerNamespace, "workflow_id" -> run)

  // The handler's worker stops polling its own queue, so the caller workflow keeps running.
  val stopHandlerWorker = fault(handlerTaskQueue, FaultKind.workerStop)

  // Each Case starts a workflow type of its own, so two Cases on one worker never share one.
  val workflowType = perCase("workflow")

  val startWorkflow =
    rpc(calls, WorkflowServiceGrpc.METHOD_START_WORKFLOW_EXECUTION) {
      field(_.getWorkflowType.name) := Operand.named(workflowType)
      field(_.getTaskQueue.name) := taskQueueName
      field(_.requestId) := run
    }

  // Polls the history for the scheduled event, run until the event exists.
  val awaitScheduled = await(scheduled, calls)(
    Condition.present(Field(_.attributes.nexusOperationScheduledEventAttributes))
  ) {}

  // Polls the pending operation until its first attempt has failed.
  val pendingAttempts = await(pending, calls)(
    Condition.equal(Field(_.attempt), Operand.integer(1))
  ) {}

  // The handle an asynchronous reply publishes and a completion reads.
  val completionAuthority = Learned("completion-authority", LearnedKind.handle)

  val awaitCompletionAuthority = awaitLearned(completionAuthority)

  // The failure a failed reply or completion carries.
  val handlerFailure =
    applicationFailure("OperationFailed", "operation failed", retryable = false)

  val completeNexusOperation =
    nexusCompletion(completionAuthority, jsonPayload("completed"))
  val failNexusOperation = nexusCompletion(completionAuthority, handlerFailure)

  // The workflow's history, waiting for new events: the request both history calls extend.
  val historyRead =
    rpc(calls, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY) {
      field(_.maximumPageSize) := Operand.integer(64)
      field(_.waitNewEvent) := Operand.flag(true)
    }

  // Resolves only once the workflow closes, so a read placed after it observes the whole history.
  val awaitClose = historyRead.extended {
    field(_.historyEventFilterType) :=
      Operand.enumValue(HistoryEventFilterType.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT)
  }

  // Lifts the history kinds among the resolved rules; a path that records none lifts nothing,
  // because a lift with no rule is a Case preparation rejects. It runs after the workflow closed, so
  // it is the closing read of every history kind.
  val history = command(
    historyRead.extended {
      read(historyEvents, Cardinality.each).into(historyEvent, Target.Lift(correlated.id))
    },
    closes = historyKinds.evidence
  )

  // Only the forged control's path inspects the workflow.
  val inspectWorkflow =
    rpc(calls, WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION) {}

  // ### The controller
  //
  // The controller's sequence is the one place the interleaving matters: it stops the handler's
  // worker when the path says so, starts the workflow, reads the scheduled event as soon as it
  // exists, polls the attempt count when a retryable failure is on the path, waits for the authority
  // the handler publishes when a completion is on the path, performs the completion, waits for the
  // workflow to close, and only then reads history.

  val callerItems = Vector(
    perform(worker.stop -> stopHandlerWorker),
    everyCase(startWorkflow),
    everyCase(awaitScheduled),
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
  )

  // ### The workflow

  // The service and operation the handler script answers, which the schedule command names.
  val service = "umpire.case.service"
  val operation = "complete"

  // The schedule command, setting the deadlines one class of the schedule action sets.
  def schedule(deadlines: ProtoScope[ScheduleNexusOperationCommandAttributes] ?=> Unit) =
    workflowCommand(proto[ApiCommand] {
      field(_.commandType) := CommandType.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION
      field(_.getScheduleNexusOperationCommandAttributes) {
        field(_.endpoint) := nexusEndpoint
        field(_.service) := service
        field(_.operation) := operation
        field(_.getInput) := jsonPayload("request")
        deadlines
      }
    })

  val startNexusOperation = schedule(())

  // The duration a deadline a path sets realizes as; the backoff is the server's own.
  val requestDeadline = duration(deadlineSeconds)

  val awaitNexusOperation =
    command(awaitCommand(startNexusOperation), regardless = true)

  // The workflow closes on every path: a failed or timed-out operation is the await's recorded
  // outcome, not a reason to leave the workflow open.
  val finishWorkflow = command(finish("done"), regardless = true)

  val workflowScript =
    script("workflow", WorkerActivation.Workflow(workflowType, caseWorker, taskQueue))(
      // The schedule command for every class of deadlines a path of the caller Model sets, each
      // under the one command's name; a schedule-to-close deadline no command sets, so a class that
      // expires one is unrealizable.
      deadlines[ApiCommand](caller.schedule, startNexusOperation, requestDeadline)(
        scheduleToStart.sets(
          _.getScheduleNexusOperationCommandAttributes.getScheduleToStartTimeout
        ),
        startToClose.sets(_.getScheduleNexusOperationCommandAttributes.getStartToCloseTimeout)
      ),
      onPath(caller.schedule)(awaitNexusOperation),
      everyCase(finishWorkflow)
    )

  // ### The handler

  def handlerError(errorType: String, behavior: NexusHandlerErrorRetryBehavior) =
    proto[HandlerError] {
      field(_.errorType) := errorType
      field(_.getFailure)(field(_.message) := "handler error")
      field(_.retryBehavior) := behavior
    }

  val respondAsync = nexusReply(
    proto[StartOperationResponse](field(_.getAsyncSuccess) {}),
    completionAuthority
  )
  val respondSync = nexusReply(proto[StartOperationResponse] {
    field(_.getSyncSuccess)(field(_.getPayload) := jsonPayload("completed"))
  })
  val respondFailed =
    nexusReply(proto[StartOperationResponse](field(_.getFailure) := handlerFailure))
  val respondErrorRetryable = nexusReply(
    handlerError(
      "INTERNAL",
      NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE
    )
  )
  val respondError = nexusReply(
    handlerError(
      "BAD_REQUEST",
      NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE
    )
  )

  val handlerScript =
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

import CallerDeclarations.*

// One Nexus operation scheduled by a controller-started workflow and answered by a handler inside
// the Case's own worker. Its server steps are derived: a timeout class fires at the deadline its
// schedule command sets.
object AsyncNexus
    extends Realizes(
      NexusSystem,
      learned = Vector(completionAuthority),
      observations = Vector(historyEvent, correlated)
    ):
  object controller extends Controller(callerItems*)
  object workers extends Workers(workflowScript, handlerScript)
  object evidence extends Evidences((Vector(scheduled) ++ historyKinds.evidence :+ pending)*)

// The forged control's realization: AsyncNexus, with the workflow inspected after the scheduled
// event.
object ForgedControl extends DerivesFrom(AsyncNexus, TrustingCaller):
  object changes
      extends Changes(inserting(after = awaitScheduled)(perform(caller.inspect -> inspectWorkflow)))
