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
 * The roles, bindings, window and run records every Temporal realization shares are the kit's
 * (temporal/realize). Nothing here builds a Case: the lifter emits the declarations into the IR and
 * Go lowers a Query's witness through them (tools/umpire/lower).
 */
package temporal
package standaloneactivity

import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.{AttemptCanceled, AttemptFailure, Fault, Finish, Hold, Release}
import temporal.realize.*
import io.temporal.api.workflowservice.v1.*
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.ActivityExecutionStatus
import io.temporal.api.enums.v1.ActivityExecutionStatus.*
import io.temporal.api.activity.v1.ActivityExecutionInfo
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure}
import temporal.server.api.testpilot.v1.{DeliveryAdmissionDecision, InstructionOutcome}
import temporal.server.api.testpilot.v1.DeliveryAdmissionDecision.*

import ActivityFamily.given
import Timeout.{expires, unset}
import worker.workerStop
import admission.{
  admissionResponseLoss,
  dispatch,
  heldAdmission,
  AdmissionFact,
  AdmissionResponseFact
}

object ActivityRealization:
  // ### Reading a status back

  /**
   * One status as DescribeActivityExecution reports it. Each kind counts in a source of its own,
   * because one poll reads one source.
   */
  private def status(fact: Fact) = Evidence.read(
    id = evidenceId(fact),
    records = fact,
    source = sourceId(fact),
    from = Recorded.single(
      METHOD_DESCRIBE_ACTIVITY_EXECUTION,
      Field[DescribeActivityExecutionResponse, ActivityExecutionInfo](_.getInfo)
    ),
    operation = Field[ActivityExecutionInfo, String](_.activityId),
    commitment = Commitment.reported
  )

  /** The status the activity's description reports while each fact holds. */
  private val activityStatus = statusTable(
    ProtocolFact.statusPaused -> ACTIVITY_EXECUTION_STATUS_PAUSED,
    ProtocolFact.statusCompleted -> ACTIVITY_EXECUTION_STATUS_COMPLETED,
    ProtocolFact.statusFailed -> ACTIVITY_EXECUTION_STATUS_FAILED,
    ProtocolFact.statusCanceled -> ACTIVITY_EXECUTION_STATUS_CANCELED,
    ProtocolFact.statusTerminated -> ACTIVITY_EXECUTION_STATUS_TERMINATED,
    ProtocolFact.statusTimedOut -> ACTIVITY_EXECUTION_STATUS_TIMED_OUT
  )

  /** Polls the activity's description until it reads the status the fact's evidence names. */
  private def awaitStatus(fact: Fact) =
    await(status(fact), workflowService)(
      Condition.equal(
        Field[ActivityExecutionInfo, ActivityExecutionStatus](_.status),
        Operand.enumValue(activityStatus(fact))
      )
    ) {
      field(_.namespace) := workerNamespace
      field(_.activityId) := run
    }

  // ### The controller's calls and controls

  private val stopWorker = Fault(taskQueue, FaultKind.workerStop)
  private val stopWorkerUntilReleased = Fault(taskQueue, FaultKind.workerStop)
  private val resumeWorker = Fault(taskQueue, FaultKind.workerResume)

  /** Each Case runs an activity type of its own, so two Cases on one worker never share one. */
  private val activityType = perCase("activity")

  /**
   * The start every class of the start action makes, under the run's id. A class adds the deadlines
   * it sets. The server refuses a start that sets neither a start-to-close nor a schedule-to-close
   * deadline, so a class that sets none still carries a start-to-close deadline, one no Case lives
   * to see.
   */
  private val startActivity = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {
    field(_.namespace) := workerNamespace
    field(_.activityId) := run
    field(_.getActivityType.name) := Operand.named(activityType)
    field(_.getTaskQueue.name) := taskQueueName
    field(_.requestId) := run
  }
  private val startUnreached = startActivity.setting {
    field(_.getStartToCloseTimeout.seconds) := unreachedDeadline
  }

  private val pauseActivity = rpc(workflowService, METHOD_PAUSE_ACTIVITY_EXECUTION) {
    field(_.namespace) := workerNamespace
    field(_.activityId) := run
  }
  private val unpauseActivity = rpc(workflowService, METHOD_UNPAUSE_ACTIVITY_EXECUTION) {
    field(_.namespace) := workerNamespace
    field(_.activityId) := run
  }
  private val requestCancelActivity =
    rpc(workflowService, METHOD_REQUEST_CANCEL_ACTIVITY_EXECUTION) {
      field(_.namespace) := workerNamespace
      field(_.activityId) := run
    }
  private val terminateActivity = rpc(workflowService, METHOD_TERMINATE_ACTIVITY_EXECUTION) {
    field(_.namespace) := workerNamespace
    field(_.activityId) := run
  }

  private val awaitPaused = awaitStatus(ProtocolFact.statusPaused)
  private val awaitCompleted = awaitStatus(ProtocolFact.statusCompleted)
  private val awaitFailed = awaitStatus(ProtocolFact.statusFailed)
  private val awaitCanceled = awaitStatus(ProtocolFact.statusCanceled)
  private val awaitTerminated = awaitStatus(ProtocolFact.statusTerminated)
  private val awaitTimedOut = awaitStatus(ProtocolFact.statusTimedOut)

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
  private val standaloneController = controller(
    perform(workerStop -> stopWorker),
    onPath(control(Control.pause))(stopWorkerUntilReleased),
    perform(
      start(unset, unset, unset) -> startUnreached,
      start(Inputs.scheduleToStart := expires) -> startActivity.setting {
        field(_.getStartToCloseTimeout.seconds) := unreachedDeadline
        field(_.getScheduleToStartTimeout.seconds) := deadline
      },
      start(Inputs.startToClose := expires) -> startActivity.setting {
        field(_.getStartToCloseTimeout.seconds) := deadline
      }
    ),
    perform(control(Control.pause) -> pauseActivity),
    onPath(control(Control.pause))(awaitPaused),
    perform(control(Control.unpause) -> unpauseActivity),
    onPath(control(Control.unpause))(resumeWorker),
    perform(control(Control.requestCancel) -> requestCancelActivity),
    perform(control(Control.terminate) -> terminateActivity),
    onPath(attemptResult(AttemptResult.completed))(awaitCompleted),
    onPath(attemptResult(AttemptResult.failed(false)))(awaitFailed),
    onPath(attemptResult(AttemptResult.canceled))(awaitCanceled),
    onPath(control(Control.terminate))(awaitTerminated),
    onPath(scheduleToClose, scheduleToStart, startToClose)(awaitTimedOut)
  )

  // ### The worker

  /** The failure an attempt that fails ends with. */
  private def attemptFailure(nonRetryable: Boolean) = AttemptFailure(
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
  )

  private val completeAttempt = Finish(Operand.Literal(ProtoValue.Text("done")))
  private val failAttempt = attemptFailure(nonRetryable = false)
  private val failActivity = attemptFailure(nonRetryable = true)
  private val cancelAttempt = AttemptCanceled

  /**
   * The activity's attempts. Each delivery of an attempt to the worker is the attempt start, and the
   * answers the path gives are the attempts in order.
   */
  private val attempts = script(
    "activity",
    Activation.Activity(activityType, caseWorker, taskQueue, starts = Vector(attemptStart))
  )(
    perform(
      attemptResult(AttemptResult.completed) -> completeAttempt,
      attemptResult(AttemptResult.failed(true)) -> failAttempt,
      attemptResult(AttemptResult.failed(false)) -> failActivity,
      attemptResult(AttemptResult.canceled) -> cancelAttempt
    )
  )

  // An activity is scheduled by its start, again by the release of a pause, and again by a failure
  // the server retries, and a worker is delivered each of its attempts. Each of those steps has
  // evidence of its own that stays true once it is so: the start call's answer; the release's answer;
  // and the second attempt's delivery, which is what shows that the first failed, that the activity
  // was scheduled again and that a worker took it again, and so confirms the three steps at once.

  /** One standalone activity a controller starts and the Case's own worker runs. */
  val standalone: Realization = temporalRealization(
    machine = activityProtocol,
    operation = activity,
    roles = Vector(workflowService, caseWorker, taskQueue),
    scripts = Vector(standaloneController, attempts),
    evidence = Vector(
      answered(ProtocolFact.statusScheduled, startActivity),
      delivered(ProtocolFact.statusStarted, attempts, 1, startActivity, Taking(attemptStart, 1)),
      status(ProtocolFact.statusPaused),
      answered(ProtocolFact.statusCancelRequested, requestCancelActivity),
      status(ProtocolFact.statusCompleted),
      status(ProtocolFact.statusFailed),
      status(ProtocolFact.statusCanceled),
      status(ProtocolFact.statusTerminated),
      status(ProtocolFact.statusTimedOut),
      delivered(
        ProtocolFact.attemptCount,
        attempts,
        2,
        startActivity,
        Taking(attemptResult(AttemptResult.failed(true)), 1),
        Taking(attemptStart, 2)
      ),
      answeredAs(
        "statusScheduledAgain",
        ProtocolFact.statusScheduled,
        unpauseActivity,
        Taking(control(Control.unpause), 1)
      )
    )
  )

  // ### The held race
  //
  // A controller starts one activity on a queue no worker polls, holds its dispatch at the cut
  // between history's validated dispatch task and matching, pauses it, reads the pause back, and
  // releases the old message. The release delivers it to admission, so it performs the attempt
  // start, and records what admission committed for it: the commit observation. A Driver realizes
  // the hold only where its environment runs the server, so the canary refuses the Case before any
  // I/O.

  private val dispatchHold =
    Actuator("hold-dispatch", ControlKind.HoldDispatched(dispatch), taskQueue)
  private val holdDispatch = Hold(dispatchHold)

  // The hold lets no dispatch of the activity reach admission before the release, and the release
  // records an attempt admission commits for any delivery of it: its record is every admission there
  // was, so the release closes that kind, and a Run that records none admitted none.
  private val releaseDispatch =
    command(Release(dispatchHold), closes = Vector(evidenceId(AdmissionFact.attemptAdmitted)))

  /** The lost answer's release: the held race's command, so each race's evidence reads one name. */
  private val loseAdmissionResponse =
    Command("release-dispatch", Fault(taskQueue, FaultKind.admissionResponseLoss))

  /**
   * What admission committed for the released delivery, from the release's own record: the
   * decision the server committed, never what a caller was told. It is keyed by the activity the
   * record names, so a decision of another activity is no evidence of this one; and the delivery is
   * the stamp the message carried, which tells two deliveries apart. A lost answer's record also
   * numbers the attempt admission committed, which must be one.
   */
  private def committed(
      fact: Fact,
      decision: DeliveryAdmissionDecision,
      release: Command | Instruction,
      exhaustive: Boolean,
      attempt: Vector[Condition[InstructionOutcome]],
      fields: Vector[TypedEvidenceField[InstructionOutcome, ?]]
  ) = Evidence.runEvent(
    id = evidenceId(fact),
    records = fact,
    source = runRecord,
    from = Recorded.runEvent[InstructionOutcome](
      EventKind.instructionCompleted,
      controllerScript,
      release,
      key = Operand.path(
        Operand.Projected.as[InstructionOutcome],
        Field[InstructionOutcome, String](_.getDeliveryAdmission.activityId)
      ),
      guard = Some(
        Condition.all(
          succeeded,
          (Vector(
            Condition.equal(
              Field[InstructionOutcome, DeliveryAdmissionDecision](_.getDeliveryAdmission.decision),
              Operand.enumValue(decision)
            )
          ) ++ attempt)*
        )
      )
    ),
    commitment = Commitment.durable,
    fields = Vector(
      EvidenceField.typed(
        "delivery",
        Field[InstructionOutcome, String](_.getDeliveryAdmission.deliveryId),
        role = Some(FieldRole.delivery)
      )
    ) ++ fields ++ Vector(
      EvidenceField.typed(
        "activityRun",
        Field[InstructionOutcome, String](_.getDeliveryAdmission.activityRunId)
      )
    ),
    exhaustive = exhaustive
  )

  // The machine starts scheduled, so no step of a path is the start: every Case carries it. It sets
  // one deadline, a start-to-close no Case lives to see, so no deadline competes with the delivery.

  /** The stale dispatch of one paused activity, held, then delivered to admission. */
  val heldDelivery: Realization = temporalRealization(
    machine = heldAdmission,
    operation = activity,
    roles = Vector(workflowService, taskQueue),
    scripts = Vector(
      controller(
        always(startUnreached),
        perform(dispatch -> holdDispatch),
        perform(control(Control.pause) -> pauseActivity),
        onPath(control(Control.pause))(awaitPaused),
        perform(attemptStart -> releaseDispatch)
      )
    ),
    evidence = Vector(
      answered(AdmissionFact.dispatchSent, holdDispatch),
      status(AdmissionFact.statusPaused),
      committed(
        AdmissionFact.admissionRejected,
        DELIVERY_ADMISSION_DECISION_REJECTED,
        releaseDispatch,
        false,
        Vector.empty,
        Vector.empty
      ),
      committed(
        AdmissionFact.attemptAdmitted,
        DELIVERY_ADMISSION_DECISION_ADMITTED,
        releaseDispatch,
        true,
        Vector.empty,
        Vector.empty
      )
    ),
    controls = Vector(dispatchHold)
  )

  /** One lost admission answer, with its durable decision observed before the response is replaced. */
  val lostAdmissionResponse: Realization = temporalRealization(
    machine = admissionResponseLoss,
    operation = activity,
    roles = Vector(workflowService, taskQueue),
    scripts = Vector(
      controller(
        always(startUnreached),
        perform(dispatch -> holdDispatch),
        perform(taskqueue.ackLoss -> loseAdmissionResponse)
      )
    ),
    evidence = Vector(
      answered(AdmissionResponseFact.dispatchSent, holdDispatch),
      committed(
        AdmissionResponseFact.attemptAdmitted,
        DELIVERY_ADMISSION_DECISION_ADMITTED,
        loseAdmissionResponse,
        false,
        Vector(
          Condition.greater(
            Field[InstructionOutcome, Int](_.getDeliveryAdmission.attempt),
            Operand.integer(0)
          )
        ),
        Vector(
          EvidenceField.typed(
            "attempt",
            Field[InstructionOutcome, Int](_.getDeliveryAdmission.attempt),
            role = Some(FieldRole.attempt)
          )
        )
      )
    ),
    controls = Vector(dispatchHold)
  )
