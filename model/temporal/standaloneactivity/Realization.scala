/* The standalone activity realization: a controller starts one activity through
 * StartActivityExecution, controls it, and reads its status back; the Case's own worker runs its
 * attempts, each ending as the path's answer for it says.
 *
 * No kind of evidence depends on seeing a state the activity passes through on its own: with a
 * running worker a scheduled activity is started, and a started one answered, before any read need
 * see either. A call's answer is the Run's record of that call; an attempt start is the Run's record
 * of the attempt the worker was delivered; and a status is read back from DescribeActivityExecution
 * only where the activity stays in it, paused until the controller releases it, or over.
 *
 * The roles, bindings, window and run records are the kit's (temporal/realize); Go lowers a
 * Query's witness through these declarations (tools/umpire/lower).
 */
package temporal
package standaloneactivity

import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.{Finish, Hold, Release}
import temporal.realize.*
import temporal.realize.WorkerInstruction.{AttemptCanceled, AttemptFailure, Fault}
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.ActivityExecutionStatus.*
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure}
import temporal.server.api.testpilot.v1.{DeliveryAdmissionDecision, InstructionOutcome}
import temporal.server.api.testpilot.v1.DeliveryAdmissionDecision.*

import ActivityFamily.given
import Timeout.expires
import worker.workerStop
import admission.{admissionResponseLoss, dispatch, heldAdmission, AdmissionFact}
import admission.AdmissionResponseFact

object ActivityRealization:
  /** A status DescribeActivityExecution reports, each kind in its own source: a poll reads one. */
  private def status(fact: Fact) = Evidence.read(
    id = evidenceId(fact),
    records = fact,
    source = sourceId(fact),
    from = Recorded.single(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Field(_.getInfo)),
    operation = Field(_.activityId),
    commitment = Commitment.reported
  )

  /** The status the activity's description reports while each fact holds. */
  val activityStatus = statusTable(
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
      Condition.equal(Field(_.status), Operand.enumValue(activityStatus(fact)))
    ) {
      field(_.namespace) := workerNamespace
      field(_.activityId) := run
    }

  // ### The controller

  private val stopWorker = Fault(taskQueue, FaultKind.workerStop)
  private val stopWorkerUntilReleased = Fault(taskQueue, FaultKind.workerStop)
  private val resumeWorker = Fault(taskQueue, FaultKind.workerResume)

  /** Each Case runs an activity type of its own, so two Cases on one worker never share one. */
  private val activityType = perCase("activity")

  /**
   * The start every class of the start action makes, under the run's id; a class adds the deadlines
   * it sets. The server refuses a start that sets neither a start-to-close nor a schedule-to-close
   * deadline, so a class that sets none carries a start-to-close deadline no Case lives to see.
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

  // The one order every functional Query's path makes its calls in. A pause is read back only of an
  // activity no worker has taken: a running worker may be delivered the first attempt, and answer
  // it, before the pause lands, and a held attempt's pause is a request whose release schedules
  // nothing, so that release's answer would evidence a scheduling that did not happen. So a path
  // that pauses keeps the worker from polling from before the start until the release.
  private val standaloneController = controller(
    perform(workerStop -> stopWorker),
    onPath(control(Control.pause))(stopWorkerUntilReleased),
    perform(
      start() -> startUnreached,
      start(Inputs.scheduleToStart := expires) -> startUnreached.setting {
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
      ProtoField.typed(Field(_.message), ProtoValue.text("attempt failed")),
      ProtoField.typed(
        Field(_.getApplicationFailureInfo),
        ProtoValue.message(
          Proto[ApplicationFailureInfo](
            ProtoField.typed(Field(_.`type`), ProtoValue.text("AttemptFailed")),
            ProtoField.typed(Field(_.nonRetryable), ProtoValue.flag(nonRetryable))
          )
        )
      )
    )
  )

  private val completeAttempt = Finish(Operand.Literal(ProtoValue.Text("done")))
  private val failAttempt = attemptFailure(nonRetryable = false)
  private val failActivity = attemptFailure(nonRetryable = true)
  private val cancelAttempt = AttemptCanceled

  /** The activity's attempts: each delivery to the worker is an attempt start, answered in order. */
  private val attempts = script(
    "activity",
    WorkerActivation.Activity(activityType, caseWorker, taskQueue, starts = Vector(attemptStart))
  )(
    perform(
      attemptResult(AttemptResult.completed) -> completeAttempt,
      attemptResult(AttemptResult.failed(true)) -> failAttempt,
      attemptResult(AttemptResult.failed(false)) -> failActivity,
      attemptResult(AttemptResult.canceled) -> cancelAttempt
    )
  )

  // An activity is scheduled by its start, again by a pause's release, and again by a retried
  // failure. Each has evidence that stays true: the start's answer, the release's answer, and the
  // second attempt's delivery, which shows the first failed, the activity was scheduled again and a
  // worker took it again, confirming the three at once.

  /** One standalone activity a controller starts and the Case's own worker runs. */
  val standalone = temporalRealization(
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
  // A controller starts one activity on a queue no worker polls, holds its dispatch between
  // history's validated dispatch task and matching, pauses it, reads the pause back, and releases the
  // old message to admission, which records what it committed. A Driver realizes the hold only
  // where its environment runs the server, so the canary refuses the Case before any I/O.

  private val dispatchHold =
    Actuator("hold-dispatch", ControlKind.HoldDispatched(dispatch), taskQueue)
  private val holdDispatch = Hold(dispatchHold)

  // The hold lets no dispatch reach admission before the release, and the release records every
  // admission there was of it, so it closes that kind: a Run that records none admitted none.
  private val releaseDispatch =
    command(Release(dispatchHold), closes = Vector(evidenceId(AdmissionFact.attemptAdmitted)))

  /** The lost answer's release: the held race's command, so each race's evidence reads one name. */
  private val loseAdmissionResponse =
    Command("release-dispatch", Fault(taskQueue, FaultKind.admissionResponseLoss))

  /** That admission committed `decision`, as the release's record of the delivery names it. */
  private def decided(decision: DeliveryAdmissionDecision): Condition[InstructionOutcome] =
    Condition.equal(Field(_.getDeliveryAdmission.decision), Operand.enumValue(decision))
  private val admitted = decided(DELIVERY_ADMISSION_DECISION_ADMITTED)
  private val rejected = decided(DELIVERY_ADMISSION_DECISION_REJECTED)

  /**
   * What admission committed for the released delivery, from the release's record where it meets
   * `guard`: never what a caller was told; keyed by the record's activity, stamped with its delivery.
   */
  private def committed(
      fact: Fact,
      release: Command | Instruction,
      exhaustive: Boolean = false,
      fields: Vector[TypedEvidenceField[InstructionOutcome, ?]] = Vector.empty
  )(guard: Condition[InstructionOutcome]*) = Evidence.runEvent(
    id = evidenceId(fact),
    records = fact,
    source = runRecord,
    from = Recorded.runEvent[InstructionOutcome](
      EventKind.instructionCompleted,
      controllerScript,
      release,
      key = Operand.path(
        Operand.Projected.as[InstructionOutcome],
        Field(_.getDeliveryAdmission.activityId)
      ),
      guard = Some(Condition.all(succeeded, guard*))
    ),
    commitment = Commitment.durable,
    fields = Vector(deliveryField(Field(_.getDeliveryAdmission.deliveryId))) ++ fields :+
      activityRunField(Field(_.getDeliveryAdmission.activityRunId)),
    exhaustive = exhaustive
  )

  // The machine starts scheduled, so every Case carries the start. Its one deadline, a start-to-close
  // no Case lives to see, competes with no delivery.

  /** The stale dispatch of one paused activity, held, then delivered to admission. */
  val heldDelivery = temporalRealization(
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
      committed(AdmissionFact.admissionRejected, releaseDispatch)(rejected),
      committed(AdmissionFact.attemptAdmitted, releaseDispatch, exhaustive = true)(admitted)
    ),
    controls = Vector(dispatchHold)
  )

  /** One lost admission answer, with its durable decision observed before the response is replaced. */
  val lostAdmissionResponse = temporalRealization(
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
      // A lost answer's record also numbers the attempt admission committed, which must be one.
      committed(
        AdmissionResponseFact.attemptAdmitted,
        loseAdmissionResponse,
        fields = Vector(attemptField(Field(_.getDeliveryAdmission.attempt)))
      )(admitted, Condition.greater(Field(_.getDeliveryAdmission.attempt), Operand.integer(0)))
    ),
    controls = Vector(dispatchHold)
  )
