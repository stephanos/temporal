/* The standalone activity realization: a controller starts one activity through
 * StartActivityExecution, controls it, and reads its status back; the Case's own worker runs its
 * attempts, each ending as the path's answer for it says.
 *
 * No kind of evidence depends on seeing a state the activity passes through on its own: with a
 * running worker a scheduled activity is started, and a started one answered, before any read need
 * see either. What a call the controller makes was answered is the Run's record of that call; that an
 * attempt started, and which, is the Run's record of the attempt the worker was delivered; and a
 * status is read back from DescribeActivityExecution only where the activity stays in it, paused
 * until the controller releases it, or over.
 *
 * Nothing here builds a Case: the lifter emits the declarations into the IR and Go lowers a Query's
 * witness through them (tools/umpire/lower).
 */
package temporal
package standaloneactivity

import umpire.*
// The Model's own `Control`, the caller's four, is the one this file names.
import umpire.realize.{Control as _, *}
import umpire.realize.Instruction.*
import umpire.realize.Operand.*
import umpire.realize.ProtoValue.*
import io.grpc.MethodDescriptor
import scalapb.GeneratedMessage
import io.temporal.api.workflowservice.v1.*
import io.temporal.api.enums.v1.ActivityExecutionStatus
import io.temporal.api.activity.v1.ActivityExecutionInfo
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure}
import temporal.server.api.testpilot.v1.{
  CorrelatedEvidence,
  DeliveryAdmissionDecision,
  InstructionOutcome,
  InstructionOutcomeStatus
}

import Timeout.{expires, unset}
import worker.workerStop
import admission.{admissionResponseLoss, dispatch, heldAdmission}

object ActivityRealization:
  private val workflowServiceRole = "temporal.workflow-service"
  private val workerRole = "temporal.worker"
  private val taskQueueRole = "temporal.task-queue"

  private val workerNamespaceBinding = "temporal.worker.namespace"
  private val taskQueueBinding = "temporal.task-queue.resource"

  private val correlatedObservation = "correlated-evidence"

  // Definition IDs every Case on this realization reads its evidence by.
  private val projectionID = "temporal.activity.standalone.projection"
  private val runFieldID = "temporal.activity.standalone.scope.run"
  private val operationFieldID = "temporal.activity.standalone.scope.activity"

  private def evidenceID(records: String) = "temporal.activity.standalone.evidence." + records

  private def sourceID(records: String) = "temporal.activity.standalone.source." + records

  /**
   * The Run's own record. The Run numbers what it records, so every kind read from it counts in this
   * one source, in the order the Run recorded it.
   */
  private val recordSource = sourceID("record")

  /**
   * One status as DescribeActivityExecution reports it. Each kind counts in a source of its own,
   * because one poll reads one source.
   */
  private def status(records: String) =
    Evidence.read(
      id = evidenceID(records),
      records = records,
      source = sourceID(records),
      from = Recorded.single(
        WorkflowServiceGrpc.METHOD_DESCRIBE_ACTIVITY_EXECUTION,
        Field[DescribeActivityExecutionResponse, ActivityExecutionInfo](_.getInfo)
      ),
      operation = Field[ActivityExecutionInfo, String](_.activityId),
      commitment = Commitment.reported
    )

  private val statusPaused = status("statusPaused")
  private val statusCompleted = status("statusCompleted")
  private val statusFailed = status("statusFailed")
  private val statusCanceled = status("statusCanceled")
  private val statusTerminated = status("statusTerminated")
  private val statusTimedOut = status("statusTimedOut")

  private val activityScriptID = "activity"
  private val startActivity = "start-activity"
  private val unpauseActivity = "unpause-activity"
  private val requestCancelActivity = "request-cancel-activity"

  private val succeeded =
    Condition.equal(
      Field[InstructionOutcome, InstructionOutcomeStatus](_.status),
      Operand.enumValue(InstructionOutcomeStatus.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)
    )

  /**
   * What the server answered a call of the controller: the Run's record of the call's completion,
   * where it succeeded. The run starts one activity, under its own id, so the run is the key.
   */
  private def accepted(
      kind: String,
      records: String,
      command: String,
      confirms: Vector[Taking] = Vector.empty
  ) =
    Evidence.runEvent(
      id = evidenceID(kind),
      records = records,
      source = recordSource,
      from = Recorded.runEvent[InstructionOutcome](
        EventKind.instructionCompleted,
        "controller",
        command,
        key = Operand.runKey(),
        guard = Some(succeeded)
      ),
      commitment = Commitment.reported,
      confirms = confirms
    )

  /**
   * What the worker reports of one attempt it was delivered: the Run's record of an activation the
   * start call carries, declared the record of the attempt of the activity's script that the server
   * numbers `attempt`, counted from 1. The Run records an attempt once it is answered, so this
   * evidence reaches a Run with that answer, and the declaration is what says so and which record it
   * is. A delivered attempt has a delivery; the record of a position no attempt was delivered for has
   * an empty one, which is a value a record holds like any other, so the guard asks for a delivery
   * that is not empty. One record is one piece of evidence, so each attempt has a kind of its own.
   * What the worker then offered the server is not read: an offer is not the server's acceptance.
   */
  private def delivered(kind: String, records: String, attempt: Long, confirms: Vector[Taking]) =
    Evidence.runEvent(
      id = evidenceID(kind),
      records = records,
      source = recordSource,
      from = Recorded.runEvent[InstructionOutcome](
        EventKind.diagnostic,
        "controller",
        startActivity,
        key = Operand.runKey(),
        guard = Some(
          Condition.all(
            Condition.present(
              Field[InstructionOutcome, Option[temporal.server.api.testpilot.v1.ActivityAttempt]](
                _.activityAttempt
              )
            ),
            Condition.not(
              Condition.equal(
                Field[InstructionOutcome, String](_.getActivityAttempt.deliveryId),
                Operand.text("")
              )
            )
          )
        ),
        attempt = Some(AttemptOf(activityScriptID, attempt))
      ),
      commitment = Commitment.reported,
      fields = Vector(
        EvidenceField.typed(
          "attempt",
          Field[InstructionOutcome, Int](_.getActivityAttempt.sdkAttempt),
          role = Some(FieldRole.attempt)
        ),
        EvidenceField.typed(
          "delivery",
          Field[InstructionOutcome, String](_.getActivityAttempt.deliveryId),
          role = Some(FieldRole.delivery)
        ),
        EvidenceField.typed(
          "activityRun",
          Field[InstructionOutcome, String](_.getActivityAttempt.activityRunId)
        )
      ),
      confirms = confirms
    )

  // An activity is scheduled by its start, again by the release of a pause, and again by a failure
  // the server retries, and a worker is delivered each of its attempts. Each of those steps has
  // evidence of its own that stays true once it is so: the start call's answer; the release's answer;
  // and the second attempt's delivery, which is what shows that the first failed, that the activity
  // was scheduled again and that a worker took it again, and so confirms the three steps at once.
  private val sources: Vector[Evidence | EvidenceRef[?, ?] | TypedEvidence[?]] = Vector(
    accepted("statusScheduled", "statusScheduled", startActivity),
    delivered("statusStarted", "statusStarted", attempt = 1, Vector(Taking(attemptStart, 1))),
    statusPaused,
    accepted("statusCancelRequested", "statusCancelRequested", requestCancelActivity),
    statusCompleted,
    statusFailed,
    statusCanceled,
    statusTerminated,
    statusTimedOut,
    delivered(
      "attemptCount",
      "attemptCount",
      attempt = 2,
      Vector(Taking(attemptResult(AttemptResult.failed(true)), 1), Taking(attemptStart, 2))
    ),
    accepted(
      "statusScheduledAgain",
      "statusScheduled",
      unpauseActivity,
      Vector(Taking(control(Control.unpause), 1))
    )
  )

  // ### The controller

  /** Each Case runs an activity type of its own, so two Cases on one worker never share one. */
  private val activityType = Name("umpire-", fixture = true, suffix = "-activity")

  /** The activity a call names: the run's id is the activity's. */
  private def named[Req](namespace: Field[Req, String], activityId: Field[Req, String]) =
    Vector(
      Assignment.typed(namespace, Operand.environment[String](workerNamespaceBinding)),
      Assignment.typed(activityId, Operand.run())
    )

  private val startNamed = named(
    Field[StartActivityExecutionRequest, String](_.namespace),
    Field[StartActivityExecutionRequest, String](_.activityId)
  )
  private val describeNamed = named(
    Field[DescribeActivityExecutionRequest, String](_.namespace),
    Field[DescribeActivityExecutionRequest, String](_.activityId)
  )

  /** The durations the deadlines a path sets realize as. */
  private val deadlineSeconds = Operand.number(2L)

  /**
   * The server refuses a start that sets neither a start-to-close nor a schedule-to-close deadline,
   * so a class that sets none still carries a start-to-close deadline, one no Case lives to see.
   */
  private val longSeconds = Operand.number(300L)

  /** The start request for one class of the start action: the deadlines the class sets. */
  private def startBinding(
      step: ClassRef,
      deadlines: Vector[TypedAssignment[StartActivityExecutionRequest, ?]]
  ) =
    Performance(
      step,
      Command(
        startActivity,
        Instruction.rpc(
          workflowServiceRole,
          WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
        )(
          startNamed ++ Vector(
            Assignment.typed(
              Field[StartActivityExecutionRequest, String](_.getActivityType.name),
              Operand.named(activityType)
            ),
            Assignment.typed(
              Field[StartActivityExecutionRequest, String](_.getTaskQueue.name),
              Operand.environment[String](taskQueueBinding)
            ),
            Assignment.typed(
              Field[StartActivityExecutionRequest, String](_.requestId),
              Operand.run()
            )
          ) ++ deadlines,
          Vector.empty
        )
      )
    )

  private def controlBinding[Req <: GeneratedMessage, Rsp <: GeneratedMessage](
      step: ClassRef,
      id: String,
      method: MethodDescriptor[Req, Rsp],
      namespace: Field[Req, String],
      activityId: Field[Req, String]
  ) =
    Performance(
      step,
      Command(
        id,
        Instruction.rpc(workflowServiceRole, method)(named(namespace, activityId), Vector.empty)
      )
    )

  /** Polls the activity's description until it reads the status one kind of evidence names. */
  private def awaitStatus(
      id: String,
      evidence: EvidenceRef[DescribeActivityExecutionRequest, ActivityExecutionInfo],
      value: ActivityExecutionStatus
  ) =
    Command(
      id,
      Instruction.poll(evidence, workflowServiceRole)(
        describeNamed,
        Condition.equal(
          Field[ActivityExecutionInfo, ActivityExecutionStatus](_.status),
          Operand.enumValue(value)
        ),
        250
      )
    )

  // The controller's sequence is the one order every functional Query's path makes its calls in:
  // the worker stops before the start where the path says so, a pause is read back before its
  // release, a cancel request and a terminate follow, and the status the activity ends in is read
  // last.
  //
  // A pause is read back only of an activity no worker has taken. With a running worker the first
  // attempt is delivered, and may be answered, before the pause lands, and the pause of a held
  // attempt is a request whose release schedules nothing: the release's answer would then be evidence
  // of a scheduling that did not happen. So a path that pauses keeps the Case's worker from polling
  // from before the start until the release.
  private val controller = Script(
    "controller",
    Activation.Controller,
    Vector(
      Item(performs =
        Vector(
          Performance(
            workerStop,
            Command("stop-worker", Fault(taskQueueRole, FaultKind.workerStop))
          )
        )
      ),
      Item(
        command = Some(
          Command("stop-worker-until-released", Fault(taskQueueRole, FaultKind.workerStop))
        ),
        when = Vector(control(Control.pause))
      ),
      Item(performs =
        Vector(
          startBinding(
            start(unset, unset, unset),
            Vector(
              Assignment.typed(
                Field[StartActivityExecutionRequest, Long](_.getStartToCloseTimeout.seconds),
                longSeconds
              )
            )
          ),
          startBinding(
            start(unset, expires, unset),
            Vector(
              Assignment.typed(
                Field[StartActivityExecutionRequest, Long](_.getStartToCloseTimeout.seconds),
                longSeconds
              ),
              Assignment.typed(
                Field[StartActivityExecutionRequest, Long](_.getScheduleToStartTimeout.seconds),
                deadlineSeconds
              )
            )
          ),
          startBinding(
            start(unset, unset, expires),
            Vector(
              Assignment.typed(
                Field[StartActivityExecutionRequest, Long](_.getStartToCloseTimeout.seconds),
                deadlineSeconds
              )
            )
          )
        )
      ),
      Item(performs =
        Vector(
          controlBinding(
            control(Control.pause),
            "pause-activity",
            WorkflowServiceGrpc.METHOD_PAUSE_ACTIVITY_EXECUTION,
            Field[PauseActivityExecutionRequest, String](_.namespace),
            Field[PauseActivityExecutionRequest, String](_.activityId)
          )
        )
      ),
      Item(
        command = Some(
          awaitStatus(
            "await-paused",
            statusPaused,
            ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED
          )
        ),
        when = Vector(control(Control.pause))
      ),
      Item(performs =
        Vector(
          controlBinding(
            control(Control.unpause),
            unpauseActivity,
            WorkflowServiceGrpc.METHOD_UNPAUSE_ACTIVITY_EXECUTION,
            Field[UnpauseActivityExecutionRequest, String](_.namespace),
            Field[UnpauseActivityExecutionRequest, String](_.activityId)
          )
        )
      ),
      Item(
        command = Some(Command("resume-worker", Fault(taskQueueRole, FaultKind.workerResume))),
        when = Vector(control(Control.unpause))
      ),
      Item(performs =
        Vector(
          controlBinding(
            control(Control.requestCancel),
            requestCancelActivity,
            WorkflowServiceGrpc.METHOD_REQUEST_CANCEL_ACTIVITY_EXECUTION,
            Field[RequestCancelActivityExecutionRequest, String](_.namespace),
            Field[RequestCancelActivityExecutionRequest, String](_.activityId)
          )
        )
      ),
      Item(performs =
        Vector(
          controlBinding(
            control(Control.terminate),
            "terminate-activity",
            WorkflowServiceGrpc.METHOD_TERMINATE_ACTIVITY_EXECUTION,
            Field[TerminateActivityExecutionRequest, String](_.namespace),
            Field[TerminateActivityExecutionRequest, String](_.activityId)
          )
        )
      ),
      Item(
        command = Some(
          awaitStatus(
            "await-completed",
            statusCompleted,
            ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_COMPLETED
          )
        ),
        when = Vector(attemptResult(AttemptResult.completed))
      ),
      Item(
        command = Some(
          awaitStatus(
            "await-failed",
            statusFailed,
            ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_FAILED
          )
        ),
        when = Vector(attemptResult(AttemptResult.failed(false)))
      ),
      Item(
        command = Some(
          awaitStatus(
            "await-canceled",
            statusCanceled,
            ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_CANCELED
          )
        ),
        when = Vector(attemptResult(AttemptResult.canceled))
      ),
      Item(
        command = Some(
          awaitStatus(
            "await-terminated",
            statusTerminated,
            ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_TERMINATED
          )
        ),
        when = Vector(control(Control.terminate))
      ),
      Item(
        command = Some(
          awaitStatus(
            "await-timed-out",
            statusTimedOut,
            ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_TIMED_OUT
          )
        ),
        when = Vector(scheduleToClose, scheduleToStart, startToClose)
      )
    )
  )

  // ### The worker

  /** The failure an attempt that fails ends with. */
  private def attemptFailure(nonRetryable: Boolean) =
    Proto[Failure](
      ProtoField.typed(Field[Failure, String](_.message), ProtoValue.text("attempt failed")),
      ProtoField.typed(
        Field[Failure, ApplicationFailureInfo](_.getApplicationFailureInfo),
        ProtoValue.message(
          Proto[ApplicationFailureInfo](
            ProtoField.typed(
              Field[ApplicationFailureInfo, String](_.`type`),
              ProtoValue.text("AttemptFailed")
            ),
            ProtoField.typed(
              Field[ApplicationFailureInfo, Boolean](_.nonRetryable),
              ProtoValue.flag(nonRetryable)
            )
          )
        )
      )
    )

  /**
   * The activity's attempts. Each delivery of an attempt to the worker is the attempt start, and the
   * answers the path gives are the attempts in order.
   */
  private val activityScript = Script(
    activityScriptID,
    Activation.Activity(activityType, workerRole, taskQueueRole, starts = Vector(attemptStart)),
    Vector(
      Item(performs =
        Vector(
          Performance(
            attemptResult(AttemptResult.completed),
            Command("complete-attempt", Finish(Literal(Text("done"))))
          ),
          Performance(
            attemptResult(AttemptResult.failed(true)),
            Command("fail-attempt", AttemptFailure(attemptFailure(nonRetryable = false)))
          ),
          Performance(
            attemptResult(AttemptResult.failed(false)),
            Command("fail-activity", AttemptFailure(attemptFailure(nonRetryable = true)))
          ),
          Performance(
            attemptResult(AttemptResult.canceled),
            Command("cancel-attempt", AttemptCanceled)
          )
        )
      )
    )
  )

  /** One standalone activity a controller starts and the Case's own worker runs. */
  val standalone: Realization = Realization(
    name = "standalone",
    machine = activityProtocol,
    producer = "temporal.activity.standalone.testpilot",
    producerVersion = "1",
    roles = Vector(
      Role(workflowServiceRole, RoleKind.endpoint),
      Role(workerRole, RoleKind.worker, namespace = workerNamespaceBinding),
      Role(
        taskQueueRole,
        RoleKind.taskQueue,
        namespace = workerNamespaceBinding,
        resource = taskQueueBinding
      )
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
    scripts = Vector(controller, activityScript),
    observations = Vector(
      Observed[CorrelatedEvidence](correlatedObservation)
    ),
    evidence = sources,
    cleanup = "cleanup"
  )

  // ### The held race
  //
  // A controller starts one activity on a queue no worker polls, holds its dispatch at the cut
  // between history's validated dispatch task and matching, pauses it, reads the pause back, and
  // releases the old message. The release delivers it to admission, so it performs the attempt
  // start, and records what admission committed for it: the commit observation. A Driver realizes
  // the hold only where its environment runs the server, so the canary refuses the Case before any
  // I/O.

  private val holdDispatch = "hold-dispatch"
  private val holdCommand = "hold-dispatch"
  private val releaseCommand = "release-dispatch"

  /**
   * What admission committed for the released delivery, from the release's own record: the
   * decision the server committed, never what a caller was told. It is keyed by the activity the
   * record names, so a decision of another activity is no evidence of this one; and the delivery is
   * the stamp the message carried, which tells two deliveries apart.
   */
  private def committed(
      records: String,
      decision: DeliveryAdmissionDecision,
      exhaustive: Boolean = false
  ) =
    Evidence.runEvent(
      id = evidenceID(records),
      records = records,
      source = recordSource,
      from = Recorded.runEvent[InstructionOutcome](
        EventKind.instructionCompleted,
        "controller",
        releaseCommand,
        key = Operand.path(
          Operand.Projected.as[InstructionOutcome],
          Field[InstructionOutcome, String](_.getDeliveryAdmission.activityId)
        ),
        guard = Some(
          Condition.all(
            succeeded,
            Condition.equal(
              Field[InstructionOutcome, DeliveryAdmissionDecision](_.getDeliveryAdmission.decision),
              Operand.enumValue(decision)
            )
          )
        )
      ),
      commitment = Commitment.durable,
      fields = Vector(
        EvidenceField.typed(
          "delivery",
          Field[InstructionOutcome, String](_.getDeliveryAdmission.deliveryId),
          role = Some(FieldRole.delivery)
        ),
        EvidenceField.typed(
          "activityRun",
          Field[InstructionOutcome, String](_.getDeliveryAdmission.activityRunId)
        )
      ),
      exhaustive = exhaustive
    )

  // The machine starts scheduled, so no step of a path is the start: every Case carries it. It sets
  // one deadline, a start-to-close no Case lives to see, so no deadline competes with the delivery.
  private val heldStart = Item(command =
    Some(
      Command(
        startActivity,
        Instruction.rpc(
          workflowServiceRole,
          WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
        )(
          startNamed ++ Vector(
            Assignment.typed(
              Field[StartActivityExecutionRequest, String](_.getActivityType.name),
              Operand.named(activityType)
            ),
            Assignment.typed(
              Field[StartActivityExecutionRequest, String](_.getTaskQueue.name),
              Operand.environment[String](taskQueueBinding)
            ),
            Assignment
              .typed(Field[StartActivityExecutionRequest, String](_.requestId), Operand.run()),
            Assignment.typed(
              Field[StartActivityExecutionRequest, Long](_.getStartToCloseTimeout.seconds),
              longSeconds
            )
          ),
          Vector.empty
        )
      )
    )
  )

  private val heldController = Script(
    "controller",
    Activation.Controller,
    Vector(
      heldStart,
      Item(performs = Vector(Performance(dispatch, Command(holdCommand, Hold(holdDispatch))))),
      Item(performs =
        Vector(
          controlBinding(
            control(Control.pause),
            "pause-activity",
            WorkflowServiceGrpc.METHOD_PAUSE_ACTIVITY_EXECUTION,
            Field[PauseActivityExecutionRequest, String](_.namespace),
            Field[PauseActivityExecutionRequest, String](_.activityId)
          )
        )
      ),
      Item(
        command = Some(
          awaitStatus(
            "await-paused",
            statusPaused,
            ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED
          )
        ),
        when = Vector(control(Control.pause))
      ),
      // The hold lets no dispatch of the activity reach admission before the release, and the release
      // records an attempt admission commits for any delivery of it: its record is every admission
      // there was, so the release closes that kind, and a Run that records none admitted none.
      Item(performs =
        Vector(
          Performance(
            attemptStart,
            Command(
              releaseCommand,
              Release(holdDispatch),
              closes = Vector(evidenceID("attemptAdmitted"))
            )
          )
        )
      )
    )
  )

  /** The stale dispatch of one paused activity, held, then delivered to admission. */
  val heldDelivery: Realization = Realization(
    name = "heldDelivery",
    machine = heldAdmission,
    producer = "temporal.activity.standalone.testpilot",
    producerVersion = "1",
    roles = Vector(
      Role(workflowServiceRole, RoleKind.endpoint),
      Role(
        taskQueueRole,
        RoleKind.taskQueue,
        namespace = workerNamespaceBinding,
        resource = taskQueueBinding
      )
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
    scripts = Vector(heldController),
    observations = Vector(
      Observed[CorrelatedEvidence](correlatedObservation)
    ),
    evidence = Vector(
      accepted("dispatchSent", "dispatchSent", holdCommand),
      statusPaused,
      committed(
        "admissionRejected",
        DeliveryAdmissionDecision.DELIVERY_ADMISSION_DECISION_REJECTED
      ),
      committed(
        "attemptAdmitted",
        DeliveryAdmissionDecision.DELIVERY_ADMISSION_DECISION_ADMITTED,
        exhaustive = true
      )
    ),
    controls = Vector(
      umpire.realize
        .Control(holdDispatch, ControlKind.HoldDispatched(dispatch), role = taskQueueRole)
    ),
    cleanup = "cleanup"
  )

  /** One lost admission answer, with its durable decision observed before the response is replaced. */
  val lostAdmissionResponse: Realization = Realization(
    name = "lostAdmissionResponse",
    machine = admissionResponseLoss,
    producer = "temporal.activity.standalone.testpilot",
    producerVersion = "1",
    roles = Vector(
      Role(workflowServiceRole, RoleKind.endpoint),
      Role(
        taskQueueRole,
        RoleKind.taskQueue,
        namespace = workerNamespaceBinding,
        resource = taskQueueBinding
      )
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
    scripts = Vector(
      Script(
        "controller",
        Activation.Controller,
        Vector(
          heldStart,
          Item(performs = Vector(Performance(dispatch, Command(holdCommand, Hold(holdDispatch))))),
          Item(performs =
            Vector(
              Performance(
                taskqueue.ackLoss,
                Command(releaseCommand, Fault(taskQueueRole, FaultKind.admissionResponseLoss))
              )
            )
          )
        )
      )
    ),
    observations = Vector(
      Observed[CorrelatedEvidence](correlatedObservation)
    ),
    evidence = Vector(
      accepted("dispatchSent", "dispatchSent", holdCommand),
      Evidence.runEvent(
        id = evidenceID("attemptAdmitted"),
        records = "attemptAdmitted",
        source = recordSource,
        from = Recorded.runEvent[InstructionOutcome](
          EventKind.instructionCompleted,
          "controller",
          releaseCommand,
          key = Operand.path(
            Operand.Projected.as[InstructionOutcome],
            Field[InstructionOutcome, String](_.getDeliveryAdmission.activityId)
          ),
          guard = Some(
            Condition.all(
              succeeded,
              Condition.equal(
                Field[InstructionOutcome, DeliveryAdmissionDecision](
                  _.getDeliveryAdmission.decision
                ),
                Operand.enumValue(DeliveryAdmissionDecision.DELIVERY_ADMISSION_DECISION_ADMITTED)
              ),
              Condition.greater(
                Field[InstructionOutcome, Int](_.getDeliveryAdmission.attempt),
                Operand.integer(0)
              )
            )
          )
        ),
        commitment = Commitment.durable,
        fields = Vector(
          EvidenceField.typed(
            "delivery",
            Field[InstructionOutcome, String](_.getDeliveryAdmission.deliveryId),
            role = Some(FieldRole.delivery)
          ),
          EvidenceField.typed(
            "attempt",
            Field[InstructionOutcome, Int](_.getDeliveryAdmission.attempt),
            role = Some(FieldRole.attempt)
          ),
          EvidenceField.typed(
            "activityRun",
            Field[InstructionOutcome, String](_.getDeliveryAdmission.activityRunId)
          )
        )
      )
    ),
    controls = Vector(
      umpire.realize
        .Control(holdDispatch, ControlKind.HoldDispatched(dispatch), role = taskQueueRole)
    ),
    cleanup = "cleanup"
  )
