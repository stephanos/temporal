// How a Case runs the standalone activity's three System machines: StandaloneActivity, HeldDispatch
// and LostStartAnswer.
//
// For StandaloneActivity a controller starts one activity with StartActivityExecution, controls it,
// and reads its status back with DescribeActivityExecution. The Case's own worker runs the attempts,
// and each attempt ends with the answer the path gives it. Evidence never needs to see a state the
// activity only passes through: a call's evidence is the Run's record of the call, an attempt's is
// the Run's record of the delivered attempt, and a status is read back only where the activity stays
// in it (paused until the controller unpauses it, or over).
//
// HeldDispatch and LostStartAnswer are races at admission: a controller holds the activity's
// dispatch and then releases it, or loses the answer to it, and reads what admission committed.
//
// The roles, bindings, window and run records are the kit's (temporal/realize). Go lowers a Query's
// witness through these declarations (tools/umpire/lower).
package temporal
package features.activity
package standalone
package system

import umpire.*
import umpire.realize.{Fact as RealizationFact, *}
import temporal.realize.*
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.ActivityExecutionStatus.*
import temporal.server.api.testpilot.v1.{DeliveryAdmissionDecision, InstructionOutcome}
import temporal.server.api.testpilot.v1.DeliveryAdmissionDecision.*

// Every call of the controller is made on the WorkflowService, in the run's namespace, of the
// activity the run started under its own id.
private val calls =
  RequestBase(workflowService, "namespace" -> workerNamespace, "activity_id" -> run)

// The status DescribeActivityExecution reports while each fact holds. Each status is read in a
// source of its own, and only once the activity stays in it.
private val described = DescribedStatus(
  ActivitySystem,
  calls,
  METHOD_DESCRIBE_ACTIVITY_EXECUTION,
  Field(_.getInfo),
  Field(_.activityId),
  Field(_.status)
)(
  system.Fact.statusPaused -> ACTIVITY_EXECUTION_STATUS_PAUSED,
  system.Fact.statusCompleted -> ACTIVITY_EXECUTION_STATUS_COMPLETED,
  system.Fact.statusFailed -> ACTIVITY_EXECUTION_STATUS_FAILED,
  system.Fact.statusCanceled -> ACTIVITY_EXECUTION_STATUS_CANCELED,
  system.Fact.statusTerminated -> ACTIVITY_EXECUTION_STATUS_TERMINATED,
  everyValue(system.Fact.statusTimedOut) -> ACTIVITY_EXECUTION_STATUS_TIMED_OUT
)

// The status table the machine's Describable capability names.
val activityStatus = described.table

// ### The controller

// The worker's process stopping, as a path's own step.
private val stopWorker = fault(taskQueue, FaultKind.workerStop)

// Each Case runs an activity type of its own, so two Cases on one worker never share one.
private val activityType = perCase("activity")

// The start every class of the start action makes, under the run's id; a class adds the deadlines
// it sets.
private val startActivity = rpc(calls, METHOD_START_ACTIVITY_EXECUTION) {
  field(_.getActivityType.name) := Operand.named(activityType)
  field(_.getTaskQueue.name) := taskQueueName
  field(_.requestId) := run
}
// The start of a class that sets no start-to-close deadline. The server refuses a start that sets
// neither a start-to-close nor a schedule-to-close deadline, so it carries a start-to-close
// deadline no Case lives to see.
private val startUnreached = startActivity.withFields {
  field(_.getStartToCloseTimeout) := duration(unreachedDeadlineSeconds)
}

private val pauseActivity = rpc(calls, METHOD_PAUSE_ACTIVITY_EXECUTION) {}
private val unpauseActivity = rpc(calls, METHOD_UNPAUSE_ACTIVITY_EXECUTION) {}
private val requestCancelActivity = rpc(calls, METHOD_REQUEST_CANCEL_ACTIVITY_EXECUTION) {}
private val terminateActivity = rpc(calls, METHOD_TERMINATE_ACTIVITY_EXECUTION) {}

// Paths that require an unstarted activity hold the server's dispatch before it can reach a worker.
// A pause path releases it only after the unpause; a schedule-to-start timeout leaves cleanup to
// cancel it after the deadline fires. Neither depends on SDK-worker shutdown or matching unloads.
private val unstartedDispatch =
  Actuator("unstarted-dispatch", ControlKind.HoldDispatched(worker.poll), taskQueue)
private val holdDispatchBeforePause = hold(unstartedDispatch)
private val holdDispatchBeforeTimeout = hold(unstartedDispatch)
private val releaseDispatchAfterPause = release(unstartedDispatch)

// ### The worker

// The failure an attempt that fails ends with.
private def failed(retryable: Boolean) =
  attemptFailure(applicationFailure("AttemptFailed", "attempt failed", retryable))

private val completeAttempt = finish("done")
private val failAttempt = failed(retryable = true)
private val failActivity = failed(retryable = false)
private val cancelAttempt = attemptCanceled

// The activity's attempts: each delivery to the worker is an attempt start, answered in order.
private val attempts = script(
  "attempts",
  WorkerActivation
    .Activity(activityType, caseWorker, taskQueue, starts = Vector(worker.poll))
)(
  perform(
    worker.respondCompleted -> completeAttempt,
    worker.respondFailed(Failure.retryable) -> failAttempt,
    worker.respondFailed(Failure.fatal) -> failActivity,
    worker.respondCanceled -> cancelAttempt
  )
)

// The kind of the unpause's answer, which confirms that the activity was scheduled again.
private val scheduledAgain = "statusScheduledAgain"

// One standalone activity a controller starts and the Case's own worker runs. An activity is
// scheduled by its start, again by an unpause, and again by a retried failure. Each has evidence
// that stays true: the start's answer, the unpause's answer, and the second attempt's delivery,
// which shows at once that the first attempt failed, that the activity was scheduled again, and
// that a worker took it again. Its server steps are derived: an attempt starts when the server
// delivers it, a retry waits out the backoff timer at the server's first retry interval, after which
// the retried attempt is dispatched (chasm/lib/activity/attempt.go:72-82, statemachine.go:393-420),
// and a timeout class fires at the deadline its start sets.
object Standalone extends Realizes(ActivitySystem):
  // The one order every functional Query's path makes its calls in. A pause is read back only of an
  // activity no worker has taken: a running worker may be delivered the first attempt, and answer
  // it, before the pause lands, and a held attempt's pause is a request whose release schedules
  // nothing, so that release's answer would evidence a scheduling that did not happen. So a path
  // that pauses arms a dispatch hold as the start is sent, waits for it immediately after the
  // start's answer, and releases it only after the unpause.
  object controller
      extends Controller(
        perform(shared.worker.worker.stop -> stopWorker),
        // Every class of the start, each setting the deadlines it expires; a schedule-to-close
        // deadline no start sets, so a class that expires one is unrealizable.
        deadlines[StartActivityExecutionRequest](
          client.start,
          startActivity,
          duration(deadlineSeconds),
          unset = Some(startToClose -> startUnreached)
        )(
          scheduleToStart.sets(_.getScheduleToStartTimeout),
          startToClose.sets(_.getStartToCloseTimeout)
        ),
        onPath(client.pause)(holdDispatchBeforePause),
        onPath(deadline.scheduleToStart)(holdDispatchBeforeTimeout),
        perform(client.pause -> pauseActivity),
        onPath(client.pause)(described.await(system.Fact.statusPaused)),
        perform(client.unpause -> unpauseActivity),
        onPath(client.unpause)(releaseDispatchAfterPause),
        perform(client.requestCancel -> requestCancelActivity),
        perform(client.terminate -> terminateActivity),
        onPath(worker.respondCompleted)(
          described.await(system.Fact.statusCompleted)
        ),
        onPath(worker.respondFailed(Failure.fatal))(
          described.await(system.Fact.statusFailed)
        ),
        onPath(worker.respondCanceled)(described.await(system.Fact.statusCanceled)),
        onPath(client.terminate)(described.await(system.Fact.statusTerminated)),
        onPath(deadline.scheduleToStart, deadline.startToClose)(
          described.await(everyValue(system.Fact.statusTimedOut))
        )
      )
  object workers extends Workers(attempts)
  object evidence
      extends Evidences(
        answered(system.Fact.statusScheduled, startActivity),
        delivered(
          system.Fact.statusStarted,
          attempts,
          attempt = 1,
          after = startActivity,
          Taking(worker.poll, 1)
        ),
        described(system.Fact.statusPaused),
        answered(system.Fact.statusCancelRequested, requestCancelActivity),
        described(system.Fact.statusCompleted),
        described(system.Fact.statusFailed),
        described(system.Fact.statusCanceled),
        described(system.Fact.statusTerminated),
        described(everyValue(system.Fact.statusTimedOut)),
        delivered(
          system.Fact.attemptCount,
          attempts,
          attempt = 2,
          after = startActivity,
          Taking(worker.respondFailed(Failure.retryable), 1),
          Taking(worker.poll, 2)
        ),
        answeredAs(
          kind = scheduledAgain,
          records = system.Fact.statusScheduled,
          call = unpauseActivity,
          Taking(client.unpause, 1)
        )
      )
  object controls extends Controls(unstartedDispatch)

// ### The held race
// A controller starts one activity on a queue no worker polls, holds its dispatch between
// history's validated dispatch task and matching, pauses it, reads the pause back, and releases the
// old message to admission, which records what it committed. A Driver realizes the hold only
// where its environment runs the server, so the canary refuses the Case before any I/O.

private val dispatchHold =
  Actuator("hold-dispatch", ControlKind.HoldDispatched(history.dispatch), taskQueue)
private val holdDispatch = hold(dispatchHold)

// The hold lets no dispatch reach admission before the release, and the release records every
// admission there was of it, so it closes that kind: a Run that records none admitted none.
private val releaseDispatch =
  command(release(dispatchHold), closes = Vector(evidenceId(AdmissionFact.attemptAdmitted)))

// That admission committed `decision`, as the release's record of the delivery names it.
private def decided(decision: DeliveryAdmissionDecision): Condition[InstructionOutcome] =
  Condition.equal(Field(_.getDeliveryAdmission.decision), Operand.enumValue(decision))
private val admitted = decided(DELIVERY_ADMISSION_DECISION_ADMITTED)
private val rejected = decided(DELIVERY_ADMISSION_DECISION_REJECTED)

// The pause HeldDispatch reads back, as DescribeActivityExecution reports it.
private val pauseDescribed = DescribedStatus(
  HeldDispatch,
  calls,
  METHOD_DESCRIBE_ACTIVITY_EXECUTION,
  Field(_.getInfo),
  Field(_.activityId),
  Field(_.status)
)(AdmissionFact.statusPaused -> ACTIVITY_EXECUTION_STATUS_PAUSED)

// What admission committed for the released delivery, from the release's record where it meets
// `guard`: never what a client was told; keyed by the record's activity, stamped with its delivery.
private def committed(
    fact: RealizationFact,
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

// The stale dispatch of one paused activity, held, then delivered to admission. The machine starts
// scheduled, so every Case carries the start; its one deadline, a start-to-close no Case lives to
// see, competes with no delivery.
object HeldDelivery extends Realizes(HeldDispatch):
  object controller
      extends Controller(
        everyCase(startUnreached),
        perform(history.dispatch -> holdDispatch),
        perform(client.pause -> pauseActivity),
        onPath(client.pause)(pauseDescribed.await(AdmissionFact.statusPaused)),
        perform(worker.poll -> releaseDispatch)
      )
  object evidence
      extends Evidences(
        answered(AdmissionFact.dispatchSent, holdDispatch),
        pauseDescribed(AdmissionFact.statusPaused),
        committed(AdmissionFact.admissionRejected, releaseDispatch)(rejected),
        committed(AdmissionFact.attemptAdmitted, releaseDispatch, exhaustive = true)(admitted)
      )
  object controls extends Controls(dispatchHold)

// ### The lost admission answer
// A controller holds the activity's dispatch as in the held race, then loses admission's answer to
// it, and reads the durable decision admission recorded before the answer was replaced.

// The lost answer's release, under the held race's command name, so each race's evidence reads
// one name.
private val loseAdmissionResponse =
  aliasOf(releaseDispatch)(fault(taskQueue, FaultKind.admissionResponseLoss))

// One lost admission answer, with its durable decision observed before the response is replaced.
object LostAdmissionResponse extends Realizes(LostStartAnswer):
  object controller
      extends Controller(
        everyCase(startUnreached),
        perform(history.dispatch -> holdDispatch),
        perform(shared.taskqueue.fault.ackLoss -> loseAdmissionResponse)
      )
  object evidence
      extends Evidences(
        answered(AdmissionResponseFact.dispatchSent, holdDispatch),
        // A lost answer's record also numbers the attempt admission committed, which must be one.
        committed(
          AdmissionResponseFact.attemptAdmitted,
          loseAdmissionResponse,
          fields = Vector(attemptField(Field(_.getDeliveryAdmission.attempt)))
        )(admitted, Condition.greater(Field(_.getDeliveryAdmission.attempt), Operand.integer(0)))
      )
  object controls extends Controls(dispatchHold)
