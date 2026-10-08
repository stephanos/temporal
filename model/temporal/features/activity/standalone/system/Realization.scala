// How a Case runs the standalone activity's System machines: StandaloneActivity, HeldDispatch,
// LostStartAnswer and TimeoutRetry.
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
// TimeoutRetry runs the same rules with occurrence evidence for a timed-out first attempt instead
// of a failed one; the worker withholds its first answer until the armed deadline expires.
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
import io.temporal.api.workflowservice.v1.{
  DescribeActivityExecutionResponse,
  StartActivityExecutionRequest
}
import io.temporal.api.activity.v1.ActivityExecutionInfo
import io.temporal.api.common.v1.Payloads
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.ActivityExecutionStatus.*
import io.temporal.api.enums.v1.TimeoutType.TIMEOUT_TYPE_HEARTBEAT
import io.temporal.api.enums.v1.PendingActivityState.*
import io.temporal.api.workflowservice.v1.StartActivityExecutionResponse
import temporal.server.api.testpilot.v1.{
  ActivityAttempt,
  ActivityAttemptResponse,
  DeliveryAdmissionDecision,
  InstructionOutcome
}
import temporal.server.api.testpilot.v1.DeliveryAdmissionDecision.*
import temporal.server.api.testpilot.v1.ActivityAttemptResponse.{
  ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED,
  ACTIVITY_ATTEMPT_RESPONSE_PENDING
}

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

// Raw API data: Describe counts scheduling, while the independent Model counts delivery. Even
// zero-delivery Cases can report public attempt 1. This is no correlated count-equality claim.
private val publicAttemptCount = Observed[ActivityExecutionInfo](attemptCount.name)
private val readAttemptCount = rpc(calls, METHOD_DESCRIBE_ACTIVITY_EXECUTION) {
  read(Field[DescribeActivityExecutionResponse, ActivityExecutionInfo](_.getInfo), Cardinality.one)
    .into(publicAttemptCount)
}

private val rawHeartbeatDetails = Observed[Payloads](heartbeatDetails.name)
private val readHeartbeatDetails = rpc(calls, METHOD_DESCRIBE_ACTIVITY_EXECUTION) {
  field(_.includeHeartbeatDetails) := Operand.flag(true)
  read(
    Field[DescribeActivityExecutionResponse, Payloads](_.getInfo.getHeartbeatDetails),
    Cardinality.one
  )
    .into(rawHeartbeatDetails)
}

private val heartbeatReceipt = Evidence.read(
  id = evidenceId(system.Fact.heartbeatReceived),
  records = system.Fact.heartbeatReceived,
  source = sourceId(system.Fact.heartbeatReceived),
  from = Recorded.single(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Field(_.getInfo)),
  operation = Field[ActivityExecutionInfo, String](_.activityId),
  commitment = Commitment.reported,
  fields = Vector(EvidenceField.typed("activityRun", Field[ActivityExecutionInfo, String](_.runId)))
)
private val receivedHeartbeat = Condition.all(
  Condition.greater(Field[ActivityExecutionInfo, Long](_.totalHeartbeatCount), Operand.number(0)),
  Condition.present(Field[ActivityExecutionInfo, Option[Payloads]](_.heartbeatDetails))
)
private val awaitHeartbeatReceipt = await(heartbeatReceipt, calls)(receivedHeartbeat) {
  field(_.includeHeartbeatDetails) := Operand.flag(true)
}

private val heartbeatFailure = Condition.equal(
  Field[ActivityExecutionInfo, io.temporal.api.enums.v1.TimeoutType](
    _.getLastFailure.getTimeoutFailureInfo.timeoutType
  ),
  Operand.enumValue(TIMEOUT_TYPE_HEARTBEAT)
)
private val heartbeatCompleted = Evidence.read(
  id = evidenceId("heartbeatCompleted"),
  records = system.Fact.statusCompleted,
  source = sourceId("heartbeatCompleted"),
  from = Recorded.single(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Field(_.getInfo)),
  operation = Field[ActivityExecutionInfo, String](_.activityId),
  commitment = Commitment.reported,
  fields = Vector(
    EvidenceField.typed("activityRun", Field[ActivityExecutionInfo, String](_.runId))
  )
)
private val awaitHeartbeatRetryCompletion = await(heartbeatCompleted, calls)(
  Condition.all(
    Condition.equal(
      Field[ActivityExecutionInfo, io.temporal.api.enums.v1.ActivityExecutionStatus](_.status),
      Operand.enumValue(ACTIVITY_EXECUTION_STATUS_COMPLETED)
    ),
    heartbeatFailure
  )
) {
  field(_.includeLastFailure) := Operand.flag(true)
}
private val heartbeatExpired = Evidence.read(
  id = evidenceId("heartbeatExpired"),
  records = system.Fact.heartbeatTimedOut,
  source = sourceId("heartbeatExpired"),
  from = Recorded.single(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Field(_.getInfo)),
  operation = Field[ActivityExecutionInfo, String](_.activityId),
  commitment = Commitment.reported,
  fields = Vector(
    EvidenceField.typed("activityRun", Field[ActivityExecutionInfo, String](_.runId))
  ),
  confirms = Vector(Taking(deadline.heartbeat, 1))
)
private val awaitHeartbeatExpiration = await(heartbeatExpired, calls)(
  Condition.all(
    Condition.equal(
      Field[ActivityExecutionInfo, io.temporal.api.enums.v1.ActivityExecutionStatus](_.status),
      Operand.enumValue(ACTIVITY_EXECUTION_STATUS_TIMED_OUT)
    ),
    heartbeatFailure
  )
) {
  field(_.includeLastFailure) := Operand.flag(true)
}
private val readHeartbeatAttemptCount = readAttemptCount.withFields {
  field(_.includeLastFailure) := Operand.flag(true)
}

private def heartbeatDelivered(attempts: Script, response: ActivityAttemptResponse) =
  Evidence.runEvent(
    id = evidenceId(system.Fact.statusStarted),
    records = system.Fact.statusStarted,
    source = runRecord,
    from = Recorded.runEvent[InstructionOutcome](
      EventKind.diagnostic,
      controllerScript,
      startActivity,
      key = Operand.runKey(),
      guard = Some(
        Condition.all(
          Condition.present(Field[InstructionOutcome, Option[ActivityAttempt]](_.activityAttempt)),
          Condition.not(
            Condition.equal(
              Field[InstructionOutcome, String](_.getActivityAttempt.deliveryId),
              Operand.text("")
            )
          ),
          Condition.equal(
            Field[InstructionOutcome, ActivityAttemptResponse](_.getActivityAttempt.response),
            Operand.enumValue(response)
          ),
          Condition.equal(
            Field[InstructionOutcome, Boolean](_.getActivityAttempt.heartbeatInvoked),
            Operand.flag(true)
          )
        )
      ),
      attempt = Some(AttemptOf(attempts, 1))
    ),
    commitment = Commitment.reported,
    fields = Vector(
      attemptField(Field(_.getActivityAttempt.sdkAttempt)),
      deliveryField(Field(_.getActivityAttempt.deliveryId)),
      activityRunField(Field(_.getActivityAttempt.activityRunId)),
      EvidenceField.typed(
        "heartbeatInvoked",
        Field[InstructionOutcome, Boolean](_.getActivityAttempt.heartbeatInvoked)
      )
    ),
    confirms = Vector(Taking(worker.poll, 1))
  )

private def heartbeatRetryDelivered(
    attempts: Script,
    id: String,
    records: RealizationFact,
    confirms: Taking*
) = Evidence.runEvent(
  id = id,
  records = records,
  source = runRecord,
  from = Recorded.runEvent[InstructionOutcome](
    EventKind.diagnostic,
    controllerScript,
    startActivity,
    key = Operand.runKey(),
    guard = Some(
      Condition.all(
        Condition.present(Field[InstructionOutcome, Option[ActivityAttempt]](_.activityAttempt)),
        Condition.not(
          Condition.equal(
            Field[InstructionOutcome, String](_.getActivityAttempt.deliveryId),
            Operand.text("")
          )
        )
      )
    ),
    attempt = Some(AttemptOf(attempts, 2))
  ),
  commitment = Commitment.reported,
  fields = Vector(
    attemptField(Field(_.getActivityAttempt.sdkAttempt)),
    deliveryField(Field(_.getActivityAttempt.deliveryId)),
    activityRunField(Field(_.getActivityAttempt.activityRunId))
  ),
  confirms = Vector(confirms*)
)

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
private val startUnlimited = startActivity.withFields {
  field(_.getRetryPolicy.maximumAttempts) := Operand.integer(0)
}
// The start of a class that sets no start-to-close deadline. The server refuses a start that sets
// neither a start-to-close nor a schedule-to-close deadline, so it carries a start-to-close
// deadline no Case lives to see.
private val startUnreached = startUnlimited.withFields {
  field(_.getStartToCloseTimeout) := duration(unreachedDeadlineSeconds)
}

private val startOne = startActivity.withFields {
  field(_.getRetryPolicy.maximumAttempts) := Operand.integer(1)
}
private val startOneUnreached = startOne.withFields {
  field(_.getStartToCloseTimeout) := duration(unreachedDeadlineSeconds)
}
private val startTwo = startActivity.withFields {
  field(_.getRetryPolicy.maximumAttempts) := Operand.integer(MaxAttempts.bound)
}
private val startTwoUnreached = startTwo.withFields {
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
private val withholdAttempt = attemptWithheld

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

private val timeoutAttempts = script(
  "timeout-attempts",
  WorkerActivation.Activity(activityType, caseWorker, taskQueue, starts = Vector(worker.poll))
)(
  onPath(deadline.startToClose)(withholdAttempt),
  perform(
    worker.respondCompleted -> completeAttempt,
    worker.respondFailed(Failure.retryable) -> failAttempt
  )
)

private val heartbeatPayloads = Proto[Payloads](
  ProtoField.typed(Field(_.payloads), ProtoValue.messages(jsonPayload("heartbeat")))
)
private val heartbeatAttempt = attemptHeartbeat(heartbeatPayloads)
private val pendingHeartbeatAttempt = attemptPending
private val heartbeatAttempts = script(
  "heartbeat-attempts",
  WorkerActivation.Activity(activityType, caseWorker, taskQueue, starts = Vector(worker.poll))
)(
  perform(worker.heartbeat -> heartbeatAttempt),
  onPath(deadline.heartbeat)(pendingHeartbeatAttempt),
  perform(worker.respondCompleted -> completeAttempt)
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
object Standalone
    extends Realizes(ActivitySystem, observations = Vector(correlated, publicAttemptCount)):
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
          client.start(maxAttempts := MaxAttempts.unlimited),
          startUnlimited,
          duration(deadlineSeconds),
          unset = Some(startToClose -> startUnreached)
        )(
          startDelay.sets(_.getStartDelay),
          scheduleToStart.sets(_.getScheduleToStartTimeout),
          startToClose.sets(_.getStartToCloseTimeout),
          heartbeat.sets(_.getHeartbeatTimeout)
        ),
        deadlines[StartActivityExecutionRequest](
          client.start(maxAttempts := MaxAttempts.one),
          startOne,
          duration(deadlineSeconds),
          unset = Some(startToClose -> startOneUnreached)
        )(
          startDelay.sets(_.getStartDelay),
          scheduleToStart.sets(_.getScheduleToStartTimeout),
          startToClose.sets(_.getStartToCloseTimeout),
          heartbeat.sets(_.getHeartbeatTimeout)
        ),
        deadlines[StartActivityExecutionRequest](
          client.start(maxAttempts := MaxAttempts.two),
          startTwo,
          duration(deadlineSeconds),
          unset = Some(startToClose -> startTwoUnreached)
        )(
          startDelay.sets(_.getStartDelay),
          scheduleToStart.sets(_.getScheduleToStartTimeout),
          startToClose.sets(_.getStartToCloseTimeout),
          heartbeat.sets(_.getHeartbeatTimeout)
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
        onPath(
          deadline.scheduleToStart,
          client.start(startToClose := Timeout.expires, maxAttempts := MaxAttempts.one)
        )(
          described.await(everyValue(system.Fact.statusTimedOut))
        ),
        everyCase(readAttemptCount)
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

// Uses the unchanged System rows, but only the timeout path's second-delivery evidence. The first
// attempt withholds its answer until its armed deadline; the second completes or exhausts retries.
object RetryAfterTimeout
    extends Realizes(TimeoutRetry, observations = Vector(correlated, publicAttemptCount)):
  object controller
      extends Controller(
        deadlines[StartActivityExecutionRequest](
          client.start(maxAttempts := MaxAttempts.two),
          startTwo,
          duration(deadlineSeconds),
          unset = Some(startToClose -> startTwoUnreached)
        )(
          startDelay.sets(_.getStartDelay),
          scheduleToStart.sets(_.getScheduleToStartTimeout),
          startToClose.sets(_.getStartToCloseTimeout),
          heartbeat.sets(_.getHeartbeatTimeout)
        ),
        onPath(worker.respondCompleted)(described.await(system.Fact.statusCompleted)),
        onPath(worker.respondFailed(Failure.retryable))(described.await(system.Fact.statusFailed)),
        everyCase(readAttemptCount)
      )
  object workers extends Workers(timeoutAttempts)
  object evidence
      extends Evidences(
        answered(system.Fact.statusScheduled, startActivity),
        delivered(
          system.Fact.statusStarted,
          timeoutAttempts,
          attempt = 1,
          after = startActivity,
          Taking(worker.poll, 1)
        ),
        delivered(
          system.Fact.attemptCount,
          timeoutAttempts,
          attempt = 2,
          after = startActivity,
          Taking(deadline.startToClose, 1),
          Taking(worker.poll, 2)
        ),
        described(system.Fact.statusCompleted),
        described(system.Fact.statusFailed)
      )

object HeartbeatThenCompletion
    extends Realizes(
      HeartbeatCompletion,
      observations = Vector(correlated, publicAttemptCount, rawHeartbeatDetails)
    ):
  object controller
      extends Controller(
        deadlines[StartActivityExecutionRequest](
          client.start(maxAttempts := MaxAttempts.unlimited),
          startUnlimited,
          duration(deadlineSeconds),
          unset = Some(startToClose -> startUnreached)
        )(
          startDelay.sets(_.getStartDelay),
          scheduleToStart.sets(_.getScheduleToStartTimeout),
          startToClose.sets(_.getStartToCloseTimeout),
          heartbeat.sets(_.getHeartbeatTimeout)
        ),
        onPath(worker.heartbeat)(awaitHeartbeatReceipt),
        onPath(worker.heartbeat)(readHeartbeatDetails),
        onPath(worker.respondCompleted)(described.await(system.Fact.statusCompleted)),
        everyCase(readAttemptCount)
      )
  object workers extends Workers(heartbeatAttempts)
  object evidence
      extends Evidences(
        answered(system.Fact.statusScheduled, startActivity),
        heartbeatDelivered(heartbeatAttempts, ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED),
        heartbeatReceipt,
        described(system.Fact.statusCompleted),
        delivered(system.Fact.attemptCount, heartbeatAttempts, attempt = 2, after = startActivity)
      )

object RetryAfterHeartbeat
    extends Realizes(
      HeartbeatRetry,
      observations = Vector(correlated, publicAttemptCount, rawHeartbeatDetails)
    ):
  object controller
      extends Controller(
        deadlines[StartActivityExecutionRequest](
          client.start(maxAttempts := MaxAttempts.two),
          startTwo,
          duration(deadlineSeconds),
          unset = Some(startToClose -> startTwoUnreached)
        )(
          startDelay.sets(_.getStartDelay),
          scheduleToStart.sets(_.getScheduleToStartTimeout),
          startToClose.sets(_.getStartToCloseTimeout),
          heartbeat.sets(_.getHeartbeatTimeout)
        ),
        onPath(worker.heartbeat)(awaitHeartbeatReceipt),
        onPath(worker.heartbeat)(readHeartbeatDetails),
        onPath(worker.respondCompleted)(awaitHeartbeatRetryCompletion),
        everyCase(readHeartbeatAttemptCount)
      )
  object workers extends Workers(heartbeatAttempts)
  object evidence
      extends Evidences(
        answered(system.Fact.statusScheduled, startActivity),
        heartbeatDelivered(heartbeatAttempts, ACTIVITY_ATTEMPT_RESPONSE_PENDING),
        heartbeatReceipt,
        heartbeatRetryDelivered(
          heartbeatAttempts,
          evidenceId(system.Fact.heartbeatTimedOut),
          system.Fact.attemptCount,
          Taking(deadline.heartbeat, 1),
          Taking(worker.poll, 2)
        ),
        heartbeatCompleted,
        delivered(system.Fact.attemptCount, heartbeatAttempts, attempt = 2, after = startActivity),
        heartbeatRetryDelivered(
          heartbeatAttempts,
          evidenceId("heartbeatTimedOutKind"),
          system.Fact.heartbeatTimedOut
        )
      )

object ExhaustAfterHeartbeat
    extends Realizes(
      HeartbeatExhaustion,
      observations = Vector(correlated, publicAttemptCount, rawHeartbeatDetails)
    ):
  object controller
      extends Controller(
        deadlines[StartActivityExecutionRequest](
          client.start(maxAttempts := MaxAttempts.one),
          startOne,
          duration(deadlineSeconds),
          unset = Some(startToClose -> startOneUnreached)
        )(
          startDelay.sets(_.getStartDelay),
          scheduleToStart.sets(_.getScheduleToStartTimeout),
          startToClose.sets(_.getStartToCloseTimeout),
          heartbeat.sets(_.getHeartbeatTimeout)
        ),
        onPath(worker.heartbeat)(awaitHeartbeatReceipt),
        onPath(worker.heartbeat)(readHeartbeatDetails),
        onPath(deadline.heartbeat)(awaitHeartbeatExpiration),
        everyCase(readHeartbeatAttemptCount)
      )

  object workers extends Workers(heartbeatAttempts)
  object evidence
      extends Evidences(
        answered(system.Fact.statusScheduled, startActivity),
        heartbeatDelivered(heartbeatAttempts, ACTIVITY_ATTEMPT_RESPONSE_PENDING),
        heartbeatReceipt,
        heartbeatExpired,
        delivered(system.Fact.attemptCount, heartbeatAttempts, attempt = 2, after = startActivity),
        described(everyValue(system.Fact.statusTimedOut))
      )

private val activityExecutionRun = Learned("activity-execution-run", LearnedKind.text)
private val executionRun = Operand.learnedValue[String](activityExecutionRun.id)
private val startExternal = startActivity.withFields {
  field(_.getStartToCloseTimeout) := duration(unreachedDeadlineSeconds)
  read(Field[StartActivityExecutionResponse, String](_.runId), Cardinality.one)
    .into(Target.Bind(activityExecutionRun.id))
}
private val byIDCalls = RequestBase(
  workflowService,
  "namespace" -> workerNamespace,
  "activity_id" -> run,
  "run_id" -> executionRun
)
private val answerIdentity = Operand.text("external-answer-controller")
private val respondCompletedById = rpc(byIDCalls, METHOD_RESPOND_ACTIVITY_TASK_COMPLETED_BY_ID) {
  field(_.identity) := answerIdentity
  field(_.getResult) := heartbeatPayloads
}
private val respondFailedById = rpc(byIDCalls, METHOD_RESPOND_ACTIVITY_TASK_FAILED_BY_ID) {
  field(_.identity) := answerIdentity
  field(_.getFailure) := applicationFailure("ExternalFailure", "external fatal failure", false)
}
private val respondCanceledById = rpc(byIDCalls, METHOD_RESPOND_ACTIVITY_TASK_CANCELED_BY_ID) {
  field(_.identity) := answerIdentity
  field(_.getDetails) := heartbeatPayloads
}
private val requestExternalCancellation = requestCancelActivity.extended {
  field(_.runId) := executionRun
  field(_.identity) := Operand.text("external-cancel-controller")
  field(_.requestId) := run
}
private val terminateExternal = terminateActivity.extended {
  field(_.runId) := executionRun
}
private val readExternalAttemptCount = readAttemptCount.extended {
  field(_.runId) := executionRun
}
private val externalCleanup = command(terminateExternal, regardless = true)

private def externalRead(id: String, records: RealizationFact, confirms: Taking*) = Evidence.read(
  id = evidenceId(id),
  records = records,
  source = sourceId(id),
  from = Recorded.single(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Field(_.getInfo)),
  operation = Field[ActivityExecutionInfo, String](_.activityId),
  commitment = Commitment.reported,
  fields = Vector(activityRunFieldForInfo),
  confirms = Vector(confirms*)
)
private val activityRunFieldForInfo =
  EvidenceField.typed("activityRun", Field[ActivityExecutionInfo, String](_.runId))
private val externalStarted = externalRead(
  "externalStarted",
  system.Fact.statusStarted,
  Taking(worker.poll, 1)
)
private val externalCancelRequested = externalRead(
  "externalCancelRequested",
  system.Fact.statusCancelRequested,
  Taking(client.requestCancel, 1)
)
private val externalCompleted = externalRead("externalCompleted", system.Fact.statusCompleted)
private val externalFailed = externalRead("externalFailed", system.Fact.statusFailed)
private val externalCanceled = externalRead("externalCanceled", system.Fact.statusCanceled)
private val sameExecution = Condition.equal(
  Field[ActivityExecutionInfo, String](_.runId),
  executionRun
)
private val running = Condition.equal(
  Field[ActivityExecutionInfo, io.temporal.api.enums.v1.ActivityExecutionStatus](_.status),
  Operand.enumValue(ACTIVITY_EXECUTION_STATUS_RUNNING)
)
private val heldStarted = Condition.all(
  sameExecution,
  running,
  Condition.equal(
    Field[ActivityExecutionInfo, io.temporal.api.enums.v1.PendingActivityState](_.runState),
    Operand.enumValue(PENDING_ACTIVITY_STATE_STARTED)
  )
)
private val heldCancelRequested = Condition.all(
  sameExecution,
  running,
  Condition.equal(
    Field[ActivityExecutionInfo, io.temporal.api.enums.v1.PendingActivityState](_.runState),
    Operand.enumValue(PENDING_ACTIVITY_STATE_CANCEL_REQUESTED)
  )
)
private val awaitExternalStarted = await(externalStarted, calls)(heldStarted) {
  field(_.runId) := executionRun
}
private val awaitExternalCancelRequested =
  await(externalCancelRequested, calls)(heldCancelRequested) {
    field(_.runId) := executionRun
  }
private def terminalExternal(status: io.temporal.api.enums.v1.ActivityExecutionStatus) =
  Condition.all(
    sameExecution,
    Condition.equal(
      Field[ActivityExecutionInfo, io.temporal.api.enums.v1.ActivityExecutionStatus](_.status),
      Operand.enumValue(status)
    ),
    Condition.equal(
      Field[ActivityExecutionInfo, io.temporal.api.enums.v1.PendingActivityState](_.runState),
      Operand.enumValue(PENDING_ACTIVITY_STATE_UNSPECIFIED)
    ),
    Condition.present(
      Field[ActivityExecutionInfo, Option[com.google.protobuf.timestamp.Timestamp]](_.closeTime)
    )
  )
private val awaitExternalCompleted = await(externalCompleted, calls)(
  terminalExternal(ACTIVITY_EXECUTION_STATUS_COMPLETED)
) {
  field(_.runId) := executionRun
  field(_.includeOutcome) := Operand.flag(true)
}
private val awaitExternalFailed = await(externalFailed, calls)(
  terminalExternal(ACTIVITY_EXECUTION_STATUS_FAILED)
) {
  field(_.runId) := executionRun
  field(_.includeOutcome) := Operand.flag(true)
}
private val awaitExternalCanceled = await(externalCanceled, calls)(
  terminalExternal(ACTIVITY_EXECUTION_STATUS_CANCELED)
) {
  field(_.runId) := executionRun
  field(_.includeOutcome) := Operand.flag(true)
}
private val failurePublication = ActivityPublication("external-failure-pending")
private val cancellationPublication = ActivityPublication("external-cancellation-pending")
private val awaitFailurePublication = awaitActivityPublication(failurePublication)
private val awaitCancellationPublication = awaitActivityPublication(cancellationPublication)
private val failurePending = attemptPending(respondFailedById)
private val cancellationPending = attemptPending(respondCanceledById)
private val externalFailureAttempts = script(
  "external-failure-attempts",
  WorkerActivation.Activity(activityType, caseWorker, taskQueue, starts = Vector(worker.poll))
)(onPath(service.respondFailedByID(Failure.fatal))(failurePending))
private val externalCancellationAttempts = script(
  "external-cancellation-attempts",
  WorkerActivation.Activity(activityType, caseWorker, taskQueue, starts = Vector(worker.poll))
)(onPath(service.respondCanceledByID)(cancellationPending))
private val scheduledExternalSettlement = ActivityExternalSettlement.Scheduled(
  carrier = startExternal,
  answer = respondCompletedById,
  settlement = awaitExternalCompleted,
  cleanup = externalCleanup
)
private val fatalExternalSettlement = ActivityExternalSettlement(
  carrier = startExternal,
  activity = externalFailureAttempts,
  attempt = 1,
  pending = failurePublication,
  held = awaitExternalStarted,
  answer = respondFailedById,
  settlement = awaitExternalFailed,
  cleanup = externalCleanup
)
private val canceledExternalSettlement = ActivityExternalSettlement(
  carrier = startExternal,
  activity = externalCancellationAttempts,
  attempt = 1,
  pending = cancellationPublication,
  held = awaitExternalCancelRequested,
  answer = respondCanceledById,
  settlement = awaitExternalCanceled,
  cleanup = externalCleanup,
  requestCancel = Some(requestExternalCancellation)
)

object ScheduledCompletionByID
    extends Realizes(
      ByIDCompletion,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount)
    ):
  object controller
      extends Controller(
        perform(client.start() -> startExternal),
        perform(service.respondCompletedByID -> respondCompletedById),
        everyCase(awaitExternalCompleted),
        everyCase(readExternalAttemptCount)
      )
  object evidence
      extends Evidences(
        answered(system.Fact.statusScheduled, startExternal),
        externalCompleted
      )
  object serverSteps extends ServerSteps(scheduledExternalSettlement)

object HeldFailureByID
    extends Realizes(
      ByIDFailure,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount)
    ):
  object controller
      extends Controller(
        perform(client.start() -> startExternal),
        everyCase(awaitFailurePublication),
        everyCase(awaitExternalStarted),
        perform(service.respondFailedByID(Failure.fatal) -> respondFailedById),
        everyCase(awaitExternalFailed),
        everyCase(readExternalAttemptCount)
      )
  object workers extends Workers(externalFailureAttempts)
  object evidence
      extends Evidences(
        answered(system.Fact.statusScheduled, startExternal),
        externalStarted,
        externalFailed
      )
  object serverSteps extends ServerSteps(fatalExternalSettlement)

object HeldCancellationByID
    extends Realizes(
      ByIDCancellation,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount)
    ):
  object controller
      extends Controller(
        perform(client.start() -> startExternal),
        everyCase(awaitCancellationPublication),
        everyCase(awaitExternalStarted),
        perform(client.requestCancel -> requestExternalCancellation),
        everyCase(awaitExternalCancelRequested),
        perform(service.respondCanceledByID -> respondCanceledById),
        everyCase(awaitExternalCanceled),
        everyCase(readExternalAttemptCount)
      )
  object workers extends Workers(externalCancellationAttempts)
  object evidence
      extends Evidences(
        answered(system.Fact.statusScheduled, startExternal),
        externalStarted,
        externalCancelRequested,
        externalCanceled
      )
  object serverSteps extends ServerSteps(canceledExternalSettlement)

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
object HeldDelivery
    extends Realizes(HeldDispatch, observations = Vector(correlated, publicAttemptCount)):
  object controller
      extends Controller(
        everyCase(startUnreached),
        perform(history.dispatch -> holdDispatch),
        perform(client.pause -> pauseActivity),
        onPath(client.pause)(pauseDescribed.await(AdmissionFact.statusPaused)),
        perform(worker.poll -> releaseDispatch),
        everyCase(readAttemptCount)
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
object LostAdmissionResponse
    extends Realizes(LostStartAnswer, observations = Vector(correlated, publicAttemptCount)):
  object controller
      extends Controller(
        everyCase(startUnreached),
        perform(history.dispatch -> holdDispatch),
        perform(shared.taskqueue.fault.ackLoss -> loseAdmissionResponse),
        everyCase(readAttemptCount)
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
