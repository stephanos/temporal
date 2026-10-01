/* The Nexus caller-side realization, ported from model/lean/Temporal/Case/Realization/Nexus.lean and
 * model/go/nexuscaller/realization.go.
 *
 * A controller-started workflow schedules one Nexus operation on the Case's endpoint role; the
 * handler answers it, and the controller completes it when the handler answered asynchronously; the
 * controller reads the history the Case's evidence is lifted from. The realization writes no
 * Program: it declares that scaffolding once and binds each action class of a Model to the
 * instruction that performs it, so the producer puts those instructions where a Query's path took
 * them.
 *
 * Everything below is a declaration the lifter emits into the IR; Go lowers a Query's witness
 * through it into a Testpilot Case (model/scalav2/goir/testpilot).
 */
package temporal
package nexuscaller

import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.*
import umpire.realize.Operand.*
import umpire.realize.ProtoValue.*

import Timeout.{expires, unset}

object NexusRealization:
  // Roles and methods every realization shares (model/lean/Temporal/Case/Support.lean).
  private val workflowServiceRole = "temporal.workflow-service"
  private val workerRole = "temporal.worker"
  private val taskQueueRole = "temporal.task-queue"
  private val handlerTaskQueueRole = "temporal.handler-task-queue"
  private val nexusEndpointRole = "temporal.nexus-endpoint"

  private val startWorkflowMethod =
    "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
  private val getHistoryMethod =
    "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"
  private val describeMethod =
    "/temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution"

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
  private val pendingAttemptsEvidence = "temporal.nexus.caller.evidence.pendingAttempts"

  /** The service and operation the handler script answers. */
  private val service = "umpire.case.service"
  private val operation = "complete"

  /** One history event kind of the operation, keyed by the scheduled event it answers. */
  private def historySource(kind: String, attributes: String, id: String) =
    Evidence(
      id = id,
      records = kind,
      source = historySourceID,
      from = Recorded.History(attributes),
      operation = s"attributes<$attributes>.scheduled_event_id",
      commitment = Commitment.reported
    )

  private val historyEvents = "history.events[*]"

  /**
   * Every evidence kind the realization admits: the scheduled event read out of history as soon as
   * it exists, the history kinds, and the pending operation's attempt count.
   */
  private val sources: Vector[Evidence] = Vector(
    Evidence(
      id = scheduledEvidence,
      records = "nexusOperationScheduled",
      source = scheduledSource,
      from = Recorded.Read(getHistoryMethod, historyEvents),
      operation = "event_id",
      commitment = Commitment.reported
    ),
    historySource(
      "nexusOperationStarted",
      "nexus_operation_started_event_attributes",
      "temporal.nexus.caller.evidence.started"
    ),
    historySource(
      "nexusOperationCompleted",
      "nexus_operation_completed_event_attributes",
      "temporal.nexus.caller.evidence.completed"
    ),
    historySource(
      "nexusOperationFailed",
      "nexus_operation_failed_event_attributes",
      "temporal.nexus.caller.evidence.failed"
    ),
    historySource(
      "nexusOperationCanceled",
      "nexus_operation_canceled_event_attributes",
      "temporal.nexus.caller.evidence.canceled"
    ),
    historySource(
      "nexusOperationTimedOut",
      "nexus_operation_timed_out_event_attributes",
      "temporal.nexus.caller.evidence.timedOut"
    ),
    Evidence(
      id = pendingAttemptsEvidence,
      records = "pendingAttempts",
      source = describeSourceID,
      from = Recorded.Read(describeMethod, "pending_nexus_operations"),
      operation = "scheduled_event_id",
      commitment = Commitment.reported
    )
  )

  // ### The scaffolding

  /** Each Case starts a workflow type of its own, so two Cases on one worker never share one. */
  private val workflowType = Name("umpire-", fixture = true, suffix = "-workflow")

  private def rpc(
      id: String,
      method: String,
      assign: Vector[Assignment],
      reads: Vector[ResponseRead]
  ) =
    Command(id, Rpc(workflowServiceRole, method, assign, reads))

  private val historyAssignments = Vector(
    Assignment("namespace", Environment(workerNamespaceBinding)),
    Assignment("execution.workflow_id", Run),
    Assignment("maximum_page_size", Literal(Number(64))),
    Assignment("wait_new_event", Literal(Flag(true)))
  )

  private val startWorkflow = rpc(
    "start-workflow",
    startWorkflowMethod,
    Vector(
      Assignment("namespace", Environment(workerNamespaceBinding)),
      Assignment("workflow_id", Run),
      Assignment("workflow_type.name", Literal(Named(workflowType))),
      Assignment("task_queue.name", Environment(taskQueueBinding)),
      Assignment("request_id", Run)
    ),
    Vector.empty
  )

  /** Resolves only once the workflow closes, so a read placed after it observes the whole history. */
  private val awaitClose = rpc(
    "await-close",
    getHistoryMethod,
    historyAssignments :+ Assignment(
      "history_event_filter_type",
      Literal(EnumName("HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT"))
    ),
    Vector.empty
  )

  /**
   * Lifts the history kinds among the resolved rules; a path that records none lifts nothing,
   * because a lift with no rule is a Case preparation rejects.
   */
  private val history = rpc(
    "history",
    getHistoryMethod,
    historyAssignments,
    Vector(
      ResponseRead(
        historyEvents,
        Cardinality.each,
        Vector(Target.Observe(historyObservation), Target.Lift(correlatedObservation))
      )
    )
  )

  private val pollAssignments = Vector(
    Assignment("namespace", Environment(workerNamespaceBinding)),
    Assignment("execution.workflow_id", Run)
  )

  /** Polls the pending operation until its first attempt has failed. */
  private val pendingAttempts = Command(
    "pending-attempts",
    Poll(
      pendingAttemptsEvidence,
      workflowServiceRole,
      pollAssignments,
      Equal(Path(Projected, "attempt"), Literal(Number(1))),
      250
    )
  )

  /** Polls the history for the scheduled event, run until the event exists. */
  private val awaitScheduled = Command(
    "await-scheduled",
    Poll(
      scheduledEvidence,
      workflowServiceRole,
      pollAssignments,
      Present(Path(Projected, "attributes<nexus_operation_scheduled_event_attributes>")),
      250
    )
  )

  /** The handle an asynchronous reply publishes and a completion reads. */
  private val completionAuthority = "completion-authority"

  /** A JSON payload of one string, as the SDK's default data converter encodes it. */
  private def textPayload(value: String) =
    Proto(
      "temporal.api.common.v1.Payload",
      ProtoField("metadata", Mapping(ProtoEntry("encoding", Utf8("json/plain")))),
      ProtoField("data", Utf8("\"" + value + "\""))
    )

  /** The failure a failed reply or completion carries. */
  private val handlerFailure = Proto(
    "temporal.api.failure.v1.Failure",
    ProtoField("message", Text("operation failed")),
    ProtoField(
      "application_failure_info",
      Message(
        Proto(
          "temporal.api.failure.v1.ApplicationFailureInfo",
          ProtoField("type", Text("OperationFailed")),
          ProtoField("non_retryable", Flag(true))
        )
      )
    )
  )

  /** The durations the deadlines a path sets realize as; the backoff is the server's own. */
  private val deadline =
    Message(Proto("google.protobuf.Duration", ProtoField("seconds", Number(2))))

  // ### The bindings
  //
  // Each binding names the class it performs and the command that performs it, so the same class
  // performed twice on one path produces two distinct commands.

  private val startNexusOperation = "start-nexus-operation"

  private val scheduleAttributes = Vector(
    ProtoField("endpoint", RoleId(nexusEndpointRole)),
    ProtoField("service", Text(service)),
    ProtoField("operation", Text(operation)),
    ProtoField("input", Message(textPayload("request")))
  )

  /**
   * The schedule command for one class of the schedule action: the deadlines the class sets, at the
   * realization's durations.
   */
  private def scheduleBinding(step: ClassRef, deadlines: Vector[ProtoField]) =
    Performance(
      step,
      Command(
        startNexusOperation,
        WorkflowCommand(
          Proto(
            "temporal.api.command.v1.Command",
            ProtoField("command_type", EnumName("COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION")),
            ProtoField(
              "schedule_nexus_operation_command_attributes",
              Message(
                Proto(
                  "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes",
                  (scheduleAttributes ++ deadlines)*
                )
              )
            )
          )
        )
      )
    )

  private def replyBinding(step: ClassRef, id: String, reply: Proto, binds: String) =
    Performance(step, Command(id, NexusReply(reply, binds), timeoutMs = 5000))

  private def response(variant: ProtoField) =
    Proto("temporal.api.nexus.v1.StartOperationResponse", variant)

  private def handlerError(errorType: String, behavior: String) =
    Proto(
      "temporal.api.nexus.v1.HandlerError",
      ProtoField("error_type", Text(errorType)),
      ProtoField(
        "failure",
        Message(
          Proto("temporal.api.nexus.v1.Failure", ProtoField("message", Text("handler error")))
        )
      ),
      ProtoField("retry_behavior", EnumName(behavior))
    )

  private def completion(step: ClassRef, id: String, result: Proto) =
    Performance(step, Command(id, NexusCompletion(completionAuthority, result)))

  // ### The plan
  //
  // The controller's sequence is the one place the interleaving matters: it stops the handler's
  // worker when the path says so, starts the workflow, reads the scheduled event as soon as it
  // exists, polls the attempt count when a retryable failure is on the path, waits for the authority
  // the handler publishes when a completion is on the path, performs the completion, waits for the
  // workflow to close, and only then reads history.

  private val controller = Script(
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
      Item(command = Some(awaitScheduled)),
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
            Vector(ProtoField("schedule_to_start_timeout", deadline))
          ),
          scheduleBinding(
            schedule(unset, unset, expires),
            Vector(ProtoField("start_to_close_timeout", deadline))
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
              ProtoField(
                "async_success",
                Message(Proto("temporal.api.nexus.v1.StartOperationResponse.Async"))
              )
            ),
            completionAuthority
          ),
          replyBinding(
            handlerReply(Reply.syncSuccess),
            "respond-sync",
            response(
              ProtoField(
                "sync_success",
                Message(
                  Proto(
                    "temporal.api.nexus.v1.StartOperationResponse.Sync",
                    ProtoField("payload", Message(textPayload("completed")))
                  )
                )
              )
            ),
            ""
          ),
          replyBinding(
            handlerReply(Reply.operationFailed),
            "respond-failed",
            response(ProtoField("failure", Message(handlerFailure))),
            ""
          ),
          replyBinding(
            handlerReply(Reply.handlerError(true)),
            "respond-error-retryable",
            handlerError("INTERNAL", "NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE"),
            ""
          ),
          replyBinding(
            handlerReply(Reply.handlerError(false)),
            "respond-error",
            handlerError("BAD_REQUEST", "NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE"),
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
  val asyncNexus: Realization = Realization(
    name = "asyncNexus",
    machine = nexusProtocol,
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
    scripts = Vector(controller, workflowScript, handlerScript),
    learned = Vector(Learned(completionAuthority, LearnedKind.handle)),
    observations = Vector(
      Observed(historyObservation, "temporal.api.history.v1.HistoryEvent"),
      Observed(correlatedObservation, "temporal.server.api.testpilot.v1.CorrelatedEvidence")
    ),
    evidence = sources,
    cleanup = "cleanup"
  )
