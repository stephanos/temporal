// How a Case runs the standalone activity's System machines: StandaloneActivity, HeldDispatch and
// LostStartAnswer. Each realization names the find Queries it runs (fn-162); a find Query none
// names is model-only.
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
// RetryAfterTimeout runs the same rules with occurrence evidence for a timed-out first attempt
// instead of a failed one; the worker withholds its first answer until the armed deadline expires.
//
// The roles, bindings, window and run records are the kit's (temporal/realize). Go lowers a Query's
// witness through these declarations (tools/umpire/lower).
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.realize.{Fact as RealizationFact, *}
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
  ActivityProtocol,
  calls,
  METHOD_DESCRIBE_ACTIVITY_EXECUTION,
  Field(_.getInfo),
  Field(_.activityId),
  Field(_.status)
)(
  protocol.Fact.statusPaused -> ACTIVITY_EXECUTION_STATUS_PAUSED,
  protocol.Fact.statusCompleted -> ACTIVITY_EXECUTION_STATUS_COMPLETED,
  protocol.Fact.statusFailed -> ACTIVITY_EXECUTION_STATUS_FAILED,
  protocol.Fact.statusCanceled -> ACTIVITY_EXECUTION_STATUS_CANCELED,
  protocol.Fact.statusTerminated -> ACTIVITY_EXECUTION_STATUS_TERMINATED,
  everyValue(protocol.Fact.statusTimedOut) -> ACTIVITY_EXECUTION_STATUS_TIMED_OUT
)

// The status table the machine's Describable capability names.
val activityStatus = described.table

// Raw API data: Describe counts scheduling, while the independent Model counts delivery. Even
// zero-delivery Cases can report public attempt 1. This is no correlated count-equality claim.
// `attemptCount` stands for DescribeActivityExecution's `info.attempt`.
private val publicAttemptCount = Observed[ActivityExecutionInfo](attemptCount.name)

private val readAttemptCount = rpc(calls, METHOD_DESCRIBE_ACTIVITY_EXECUTION) {
  read(Field[DescribeActivityExecutionResponse, ActivityExecutionInfo](_.getInfo), Cardinality.one)
    .into(publicAttemptCount)
}

// `heartbeatDetails` stands for DescribeActivityExecution's `info.heartbeatDetails`.
private val rawHeartbeatDetails = Observed[Payloads](heartbeatDetails.name)

private val readHeartbeatDetails = rpc(calls, METHOD_DESCRIBE_ACTIVITY_EXECUTION) {
  field(_.includeHeartbeatDetails) := Operand.flag(true)
  read(
    Field[DescribeActivityExecutionResponse, Payloads](_.getInfo.getHeartbeatDetails),
    Cardinality.one
  )
    .into(rawHeartbeatDetails)
}

private val activityRunFieldForInfo =
  EvidenceField.typed("activityRun", Field[ActivityExecutionInfo, String](_.runId))

private def describeRead(id: String, records: RealizationFact, confirms: Taking*) = Evidence.read(
  id = evidenceId(id),
  records = records,
  source = sourceId(id),
  from = Recorded.single(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Field(_.getInfo)),
  operation = Field[ActivityExecutionInfo, String](_.activityId),
  commitment = Commitment.reported,
  fields = Vector(activityRunFieldForInfo),
  confirms = Vector(confirms*)
)

private def statusIs(status: io.temporal.api.enums.v1.ActivityExecutionStatus) =
  Condition.equal(
    Field[ActivityExecutionInfo, io.temporal.api.enums.v1.ActivityExecutionStatus](_.status),
    Operand.enumValue(status)
  )

private def runStateIs(state: io.temporal.api.enums.v1.PendingActivityState) =
  Condition.equal(
    Field[ActivityExecutionInfo, io.temporal.api.enums.v1.PendingActivityState](_.runState),
    Operand.enumValue(state)
  )

private val heartbeatReceipt = describeRead("heartbeatReceived", protocol.Fact.heartbeatReceived)

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

private val heartbeatCompleted = describeRead("heartbeatCompleted", protocol.Fact.statusCompleted)

private val awaitHeartbeatRetryCompletion = await(heartbeatCompleted, calls)(
  Condition.all(
    statusIs(ACTIVITY_EXECUTION_STATUS_COMPLETED),
    heartbeatFailure
  )
) {
  field(_.includeLastFailure) := Operand.flag(true)
}

private val heartbeatExpired = describeRead(
  "heartbeatExpired",
  protocol.Fact.heartbeatTimedOut,
  Taking(deadline.heartbeat, 1)
)

private val awaitHeartbeatExpiration = await(heartbeatExpired, calls)(
  Condition.all(
    statusIs(ACTIVITY_EXECUTION_STATUS_TIMED_OUT),
    heartbeatFailure
  )
) {
  field(_.includeLastFailure) := Operand.flag(true)
}

private val readHeartbeatAttemptCount = readAttemptCount.withFields {
  field(_.includeLastFailure) := Operand.flag(true)
}

private def responseIs(response: ActivityAttemptResponse) = Condition.equal(
  Field[InstructionOutcome, ActivityAttemptResponse](_.getActivityAttempt.response),
  Operand.enumValue(response)
)

private val heartbeatInvoked = Condition.equal(
  Field[InstructionOutcome, Boolean](_.getActivityAttempt.heartbeatInvoked),
  Operand.flag(true)
)

private val heartbeatInvokedField = EvidenceField.typed(
  "heartbeatInvoked",
  Field[InstructionOutcome, Boolean](_.getActivityAttempt.heartbeatInvoked)
)

private def attemptRecord(
    script: Script,
    number: Long,
    carrier: Instruction,
    extraGuard: Condition[InstructionOutcome]*
)(
    id: String,
    records: RealizationFact,
    extraFields: Vector[TypedEvidenceField[InstructionOutcome, ?]] = Vector.empty,
    confirms: Vector[Taking] = Vector.empty
) = Evidence.runEvent(
  id = evidenceId(id),
  records = records,
  source = runRecord,
  from = Recorded.runEvent[InstructionOutcome](
    EventKind.diagnostic,
    controllerScript,
    carrier,
    key = Operand.runKey(),
    guard = Some(
      Condition.all(
        Condition.present(Field[InstructionOutcome, Option[ActivityAttempt]](_.activityAttempt)),
        (Vector(
          Condition.not(
            Condition.equal(
              Field[InstructionOutcome, String](_.getActivityAttempt.deliveryId),
              Operand.text("")
            )
          )
        ) ++ extraGuard)*
      )
    ),
    attempt = Some(AttemptOf(script, number))
  ),
  commitment = Commitment.reported,
  fields = Vector[TypedEvidenceField[InstructionOutcome, ?]](
    attemptField(Field(_.getActivityAttempt.sdkAttempt)),
    deliveryField(Field(_.getActivityAttempt.deliveryId)),
    activityRunField(Field(_.getActivityAttempt.activityRunId))
  ) ++ extraFields,
  confirms = confirms
)

// ### The controller

// The worker's process stopping, as a path's own step.
private val stopWorker = fault(taskQueue, FaultKind.workerStop)

// Each Case runs an activity type of its own, so two Cases on one worker never share one.
private val activityType = perCase("activity")

private val activation =
  WorkerActivation.Activity(
    activityType,
    caseWorker,
    taskQueue,
    starts = Vector(activityWorker.poll)
  )

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
  Actuator("unstarted-dispatch", ControlKind.HoldDispatched(activityWorker.poll), taskQueue)

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
  activation
)(
  perform(
    activityWorker.respondCompleted -> completeAttempt,
    activityWorker.respondFailed(Failure.retryable) -> failAttempt,
    activityWorker.respondFailed(Failure.fatal) -> failActivity,
    activityWorker.respondCanceled -> cancelAttempt
  )
)

private val timeoutAttempts = script(
  "timeout-attempts",
  activation
)(
  onPath(deadline.startToClose)(withholdAttempt),
  perform(
    activityWorker.respondCompleted -> completeAttempt,
    activityWorker.respondFailed(Failure.retryable) -> failAttempt
  )
)

private val heartbeatPayloads = Proto[Payloads](
  ProtoField.typed(Field(_.payloads), ProtoValue.messages(jsonPayload("heartbeat")))
)

private val heartbeatAttempt = attemptHeartbeat(heartbeatPayloads)
private val pendingHeartbeatAttempt = attemptPending

private val heartbeatAttempts = script(
  "heartbeat-attempts",
  activation
)(
  perform(activityWorker.heartbeat -> heartbeatAttempt),
  onPath(deadline.heartbeat)(pendingHeartbeatAttempt),
  perform(activityWorker.respondCompleted -> completeAttempt)
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
    extends Realizes(
      ActivityProtocol,
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(
        ActivityProtocol.queries,
        ActivityProtocol.capabilities,
        ActivityProtocol.completion.queries.completion,
        ActivityProtocol.retry.queries.nonRetryableFailure,
        ActivityProtocol.retry.queries.retry,
        ActivityProtocol.cancellation.queries.cancel,
        ActivityProtocol.cancellation.queries.cancelRequest,
        ActivityProtocol.pausing.queries.pauseResume,
        ActivityProtocol.dispatching.queries.startDelayedCompletion,
        ActivityProtocol.timeouts.queries.scheduleToStartTimeout,
        ActivityProtocol.timeouts.queries.startToCloseTimeout
      )
    ):

  // The one order every functional Query's path makes its calls in. A pause is read back only of an
  // activity no worker has taken: a running worker may be delivered the first attempt, and answer
  // it, before the pause lands, and a held attempt's pause is a request whose release schedules
  // nothing, so that release's answer would evidence a scheduling that did not happen. So a path
  // that pauses arms a dispatch hold as the start is sent, waits for it immediately after the
  // start's answer, and releases it only after the unpause.
  object controller
      extends Controller(
        perform(actors.worker.worker.stop -> stopWorker),
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
        onPath(client.pause)(described.await(protocol.Fact.statusPaused)),
        perform(client.unpause -> unpauseActivity),
        onPath(client.unpause)(releaseDispatchAfterPause),
        perform(client.requestCancel -> requestCancelActivity),
        perform(client.terminate -> terminateActivity),
        onPath(activityWorker.respondCompleted)(
          described.await(protocol.Fact.statusCompleted)
        ),
        onPath(activityWorker.respondFailed(Failure.fatal))(
          described.await(protocol.Fact.statusFailed)
        ),
        onPath(activityWorker.respondCanceled)(described.await(protocol.Fact.statusCanceled)),
        onPath(client.terminate)(described.await(protocol.Fact.statusTerminated)),
        onPath(
          deadline.scheduleToStart,
          client.start(startToClose := Timeout.expires, maxAttempts := MaxAttempts.one)
        )(
          described.await(everyValue(protocol.Fact.statusTimedOut))
        ),
        everyCase(readAttemptCount)
      )

  object workers extends Workers(attempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startActivity),
        delivered(
          protocol.Fact.statusStarted,
          attempts,
          attempt = 1,
          after = startActivity,
          Taking(activityWorker.poll, 1)
        ),
        described(protocol.Fact.statusPaused),
        answered(protocol.Fact.statusCancelRequested, requestCancelActivity),
        described(protocol.Fact.statusCompleted),
        described(protocol.Fact.statusFailed),
        described(protocol.Fact.statusCanceled),
        described(protocol.Fact.statusTerminated),
        described(everyValue(protocol.Fact.statusTimedOut)),
        delivered(
          protocol.Fact.attemptCount,
          attempts,
          attempt = 2,
          after = startActivity,
          Taking(activityWorker.respondFailed(Failure.retryable), 1),
          Taking(activityWorker.poll, 2)
        ),
        answeredAs(
          kind = scheduledAgain,
          records = protocol.Fact.statusScheduled,
          call = unpauseActivity,
          Taking(client.unpause, 1)
        )
      )

  object controls extends Controls(unstartedDispatch)

// Uses the unchanged System rows, but only the timeout path's second-delivery evidence. The first
// attempt withholds its answer until its armed deadline; the second completes or exhausts retries.
object RetryAfterTimeout
    extends Realizes(
      ActivityProtocol,
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(
        ActivityProtocol.timeouts.queries.retryAfterTimeout,
        ActivityProtocol.timeouts.queries.retryExhaustion
      )
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
        onPath(activityWorker.respondCompleted)(described.await(protocol.Fact.statusCompleted)),
        onPath(activityWorker.respondFailed(Failure.retryable))(
          described.await(protocol.Fact.statusFailed)
        ),
        everyCase(readAttemptCount)
      )

  object workers extends Workers(timeoutAttempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startActivity),
        delivered(
          protocol.Fact.statusStarted,
          timeoutAttempts,
          attempt = 1,
          after = startActivity,
          Taking(activityWorker.poll, 1)
        ),
        delivered(
          protocol.Fact.attemptCount,
          timeoutAttempts,
          attempt = 2,
          after = startActivity,
          Taking(deadline.startToClose, 1),
          Taking(activityWorker.poll, 2)
        ),
        described(protocol.Fact.statusCompleted),
        described(protocol.Fact.statusFailed)
      )

object HeartbeatThenCompletion
    extends Realizes(
      ActivityProtocol,
      observations = Vector(correlated, publicAttemptCount, rawHeartbeatDetails),
      runs = Seq(ActivityProtocol.heartbeat.queries.heartbeatThenCompletes)
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
        onPath(activityWorker.heartbeat)(awaitHeartbeatReceipt),
        onPath(activityWorker.heartbeat)(readHeartbeatDetails),
        onPath(activityWorker.respondCompleted)(described.await(protocol.Fact.statusCompleted)),
        everyCase(readAttemptCount)
      )

  object workers extends Workers(heartbeatAttempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startActivity),
        attemptRecord(
          heartbeatAttempts,
          1,
          startActivity,
          responseIs(ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED),
          heartbeatInvoked
        )(
          "statusStarted",
          protocol.Fact.statusStarted,
          extraFields = Vector(heartbeatInvokedField),
          confirms = Vector(Taking(activityWorker.poll, 1))
        ),
        heartbeatReceipt,
        described(protocol.Fact.statusCompleted),
        delivered(protocol.Fact.attemptCount, heartbeatAttempts, attempt = 2, after = startActivity)
      )

object RetryAfterHeartbeat
    extends Realizes(
      ActivityProtocol,
      observations = Vector(correlated, publicAttemptCount, rawHeartbeatDetails),
      runs = Seq(ActivityProtocol.heartbeat.queries.heartbeatTimeoutRetriesThenCompletes)
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
        onPath(activityWorker.heartbeat)(awaitHeartbeatReceipt),
        onPath(activityWorker.heartbeat)(readHeartbeatDetails),
        onPath(activityWorker.respondCompleted)(awaitHeartbeatRetryCompletion),
        everyCase(readHeartbeatAttemptCount)
      )

  object workers extends Workers(heartbeatAttempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startActivity),
        attemptRecord(
          heartbeatAttempts,
          1,
          startActivity,
          responseIs(ACTIVITY_ATTEMPT_RESPONSE_PENDING),
          heartbeatInvoked
        )(
          "statusStarted",
          protocol.Fact.statusStarted,
          extraFields = Vector(heartbeatInvokedField),
          confirms = Vector(Taking(activityWorker.poll, 1))
        ),
        heartbeatReceipt,
        attemptRecord(heartbeatAttempts, 2, startActivity)(
          "heartbeatTimedOut",
          protocol.Fact.attemptCount,
          confirms = Vector(Taking(deadline.heartbeat, 1), Taking(activityWorker.poll, 2))
        ),
        heartbeatCompleted,
        delivered(protocol.Fact.attemptCount, heartbeatAttempts, attempt = 2, after = startActivity),
        attemptRecord(heartbeatAttempts, 2, startActivity)(
          "heartbeatTimedOutKind",
          protocol.Fact.heartbeatTimedOut
        )
      )

object ExhaustAfterHeartbeat
    extends Realizes(
      ActivityProtocol,
      observations = Vector(correlated, publicAttemptCount, rawHeartbeatDetails),
      runs = Seq(ActivityProtocol.heartbeat.queries.heartbeatTimeoutExhausts)
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
        onPath(activityWorker.heartbeat)(awaitHeartbeatReceipt),
        onPath(activityWorker.heartbeat)(readHeartbeatDetails),
        onPath(deadline.heartbeat)(awaitHeartbeatExpiration),
        everyCase(readHeartbeatAttemptCount)
      )

  object workers extends Workers(heartbeatAttempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startActivity),
        attemptRecord(
          heartbeatAttempts,
          1,
          startActivity,
          responseIs(ACTIVITY_ATTEMPT_RESPONSE_PENDING),
          heartbeatInvoked
        )(
          "statusStarted",
          protocol.Fact.statusStarted,
          extraFields = Vector(heartbeatInvokedField),
          confirms = Vector(Taking(activityWorker.poll, 1))
        ),
        heartbeatReceipt,
        heartbeatExpired,
        delivered(protocol.Fact.attemptCount, heartbeatAttempts, attempt = 2, after = startActivity),
        described(everyValue(protocol.Fact.statusTimedOut))
      )

private val activityExecutionRun = Learned("activity-execution-run", LearnedKind.text)
private val executionRun = Operand.learnedValue[String](activityExecutionRun.id)

private val runCalls = RequestBase(
  workflowService,
  "namespace" -> workerNamespace,
  "activity_id" -> run,
  "run_id" -> executionRun
)

private val startExternal = startActivity.withFields {
  field(_.getStartToCloseTimeout) := duration(unreachedDeadlineSeconds)
  read(Field[StartActivityExecutionResponse, String](_.runId), Cardinality.one)
    .into(Target.Bind(activityExecutionRun.id))
}

private val answerIdentity = Operand.text("external-answer-controller")

private val respondCompletedById = rpc(runCalls, METHOD_RESPOND_ACTIVITY_TASK_COMPLETED_BY_ID) {
  field(_.identity) := answerIdentity
}

private val respondFailedById = rpc(runCalls, METHOD_RESPOND_ACTIVITY_TASK_FAILED_BY_ID) {
  field(_.identity) := answerIdentity
  field(_.getFailure) := applicationFailure("ExternalFailure", "external fatal failure", false)
}

private val respondCanceledById = rpc(runCalls, METHOD_RESPOND_ACTIVITY_TASK_CANCELED_BY_ID) {
  field(_.identity) := answerIdentity
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

private val externalStarted = describeRead(
  "externalStarted",
  protocol.Fact.statusStarted,
  Taking(activityWorker.poll, 1)
)

private val externalCancelRequested = describeRead(
  "externalCancelRequested",
  protocol.Fact.statusCancelRequested,
  Taking(client.requestCancel, 1)
)

private val externalCompleted = describeRead(
  "externalCompleted",
  protocol.Fact.statusCompleted,
  Taking(completionClient.respondCompletedByID, 1)
)

private val externalFailed = describeRead(
  "externalFailed",
  protocol.Fact.statusFailed,
  Taking(completionClient.respondFailedByID(Failure.fatal), 1)
)

private val externalCanceled = describeRead(
  "externalCanceled",
  protocol.Fact.statusCanceled,
  Taking(completionClient.respondCanceledByID, 1)
)

private val running = statusIs(ACTIVITY_EXECUTION_STATUS_RUNNING)

private val heldStarted = Condition.all(
  running,
  runStateIs(PENDING_ACTIVITY_STATE_STARTED)
)

private val heldCancelRequested = Condition.all(
  running,
  runStateIs(PENDING_ACTIVITY_STATE_CANCEL_REQUESTED)
)

private val awaitExternalStarted = await(externalStarted, runCalls)(heldStarted) {}

private val awaitExternalCancelRequested =
  await(externalCancelRequested, runCalls)(heldCancelRequested) {}

private def terminalExternal(status: io.temporal.api.enums.v1.ActivityExecutionStatus) =
  Condition.all(
    statusIs(status),
    runStateIs(PENDING_ACTIVITY_STATE_UNSPECIFIED),
    Condition.present(
      Field[ActivityExecutionInfo, Option[com.google.protobuf.timestamp.Timestamp]](_.closeTime)
    )
  )

private val awaitExternalCompleted = await(externalCompleted, runCalls)(
  terminalExternal(ACTIVITY_EXECUTION_STATUS_COMPLETED)
) {
  field(_.includeOutcome) := Operand.flag(true)
}

private val awaitExternalFailed = await(externalFailed, runCalls)(
  terminalExternal(ACTIVITY_EXECUTION_STATUS_FAILED)
) {
  field(_.includeOutcome) := Operand.flag(true)
}

private val awaitExternalCanceled = await(externalCanceled, runCalls)(
  terminalExternal(ACTIVITY_EXECUTION_STATUS_CANCELED)
) {
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
  activation
)(onPath(completionClient.respondFailedByID(Failure.fatal))(failurePending))

private val externalCancellationAttempts = script(
  "external-cancellation-attempts",
  activation
)(onPath(completionClient.respondCanceledByID)(cancellationPending))

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
      ActivityProtocol,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(
        ActivityProtocol.completion.queries.scheduledCompletedByID
      )
    ):

  object controller
      extends Controller(
        perform(client.start() -> startExternal),
        perform(completionClient.respondCompletedByID -> respondCompletedById),
        everyCase(awaitExternalCompleted),
        everyCase(readExternalAttemptCount)
      )

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startExternal),
        externalCompleted
      )

  object serverSteps extends ServerSteps(scheduledExternalSettlement)

object HeldFailureByID
    extends Realizes(
      ActivityProtocol,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(
        ActivityProtocol.retry.queries.heldFailedByID
      )
    ):

  object controller
      extends Controller(
        perform(client.start() -> startExternal),
        everyCase(awaitFailurePublication),
        everyCase(awaitExternalStarted),
        perform(completionClient.respondFailedByID(Failure.fatal) -> respondFailedById),
        everyCase(awaitExternalFailed),
        everyCase(readExternalAttemptCount)
      )

  object workers extends Workers(externalFailureAttempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startExternal),
        externalStarted,
        delivered(
          protocol.Fact.attemptCount,
          externalFailureAttempts,
          attempt = 2,
          after = startExternal
        ),
        externalFailed
      )

  object serverSteps extends ServerSteps(fatalExternalSettlement)

object HeldCancellationByID
    extends Realizes(
      ActivityProtocol,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(
        ActivityProtocol.cancellation.queries.heldCanceledByID,
        ActivityProtocol.cancellation.queries.cancelIsRequested
      )
    ):

  object controller
      extends Controller(
        perform(client.start() -> startExternal),
        everyCase(awaitCancellationPublication),
        everyCase(awaitExternalStarted),
        perform(client.requestCancel -> requestExternalCancellation),
        everyCase(awaitExternalCancelRequested),
        perform(completionClient.respondCanceledByID -> respondCanceledById),
        everyCase(awaitExternalCanceled),
        everyCase(readExternalAttemptCount)
      )

  object workers extends Workers(externalCancellationAttempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startExternal),
        externalStarted,
        delivered(
          protocol.Fact.attemptCount,
          externalCancellationAttempts,
          attempt = 2,
          after = startExternal
        ),
        externalCancelRequested,
        externalCanceled
      )

  object serverSteps extends ServerSteps(canceledExternalSettlement)

// ### Reset
// A reset of a held attempt, deferred until its heartbeat deadline ends it. The first attempt
// publishes the SDK's pending record before the controller reads it held and resets it; the
// server's next delivery is a first attempt again, which completes under a one-attempt policy.
// The one start the path takes, with its heartbeat deadline, learning the execution run once.
private val startResetOne = startOne.withFields {
  field(_.getHeartbeatTimeout) := duration(deadlineSeconds)
  field(_.getStartToCloseTimeout) := duration(unreachedDeadlineSeconds)
  read(Field[StartActivityExecutionResponse, String](_.runId), Cardinality.one)
    .into(Target.Bind(activityExecutionRun.id))
}

private val resetHeldActivity = rpc(runCalls, METHOD_RESET_ACTIVITY_EXECUTION) {
  field(_.identity) := Operand.text("reset-controller")
  field(_.keepPaused) := Operand.flag(false)
  field(_.requestId) := run
}

private val resetPublication = ActivityPublication("reset-pending")
private val awaitResetPublication = awaitActivityPublication(resetPublication)

private val resetStarted =
  describeRead("resetStarted", protocol.Fact.statusStarted, Taking(activityWorker.poll, 1))

private val awaitResetHeld = await(resetStarted, runCalls)(heldStarted) {}

private val resetCompleted = describeRead(
  "resetCompleted",
  protocol.Fact.statusCompleted,
  Taking(activityWorker.respondCompleted, 1)
)

private val awaitResetCompleted = await(resetCompleted, runCalls)(
  terminalExternal(ACTIVITY_EXECUTION_STATUS_COMPLETED)
) {}

private val pendingResetAttempt = attemptPending

private val resetAttempts = script(
  "reset-attempts",
  activation
)(
  onPath(deadline.heartbeat)(pendingResetAttempt),
  perform(activityWorker.respondCompleted -> completeAttempt)
)

private val deferredResetSettlement = ActivityResetSettlement(
  carrier = startResetOne,
  activity = resetAttempts,
  attempt = 1,
  pending = resetPublication,
  held = awaitResetHeld,
  resetRequest = resetHeldActivity,
  timer = deadline.heartbeat,
  freshAttempt = 2,
  settlement = awaitResetCompleted,
  cleanup = externalCleanup
)

object ResetAfterHeartbeat
    extends Realizes(
      ActivityProtocol,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(
        ActivityProtocol.reset.queries.deferredResetCompletes
      )
    ):

  object controller
      extends Controller(
        perform(
          client
            .start(heartbeat := Timeout.expires, maxAttempts := MaxAttempts.one) -> startResetOne
        ),
        everyCase(awaitResetPublication),
        everyCase(awaitResetHeld),
        perform(client.reset(ResetPause.resume) -> resetHeldActivity),
        everyCase(awaitResetCompleted),
        everyCase(readExternalAttemptCount)
      )

  object workers extends Workers(resetAttempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startResetOne),
        resetStarted,
        answeredAs(
          kind = "resetRequested",
          records = protocol.Fact.statusStarted,
          call = resetHeldActivity,
          Taking(client.reset(ResetPause.resume), 1)
        ),
        attemptRecord(
          resetAttempts,
          2,
          startResetOne,
          responseIs(ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)
        )(
          "attemptCount",
          protocol.Fact.attemptCount,
          confirms = Vector(Taking(deadline.heartbeat, 1), Taking(activityWorker.poll, 2))
        ),
        attemptRecord(
          resetAttempts,
          2,
          startResetOne,
          responseIs(ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)
        )(
          "resetHeartbeatTimedOut",
          protocol.Fact.heartbeatTimedOut
        ),
        resetCompleted
      )

  object serverSteps
      extends ServerSteps(
        ServerStep(deadline.heartbeat, CauseKind.timer, deadlineMs, TimeoutBasis.heartbeat),
        deferredResetSettlement
      )

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
    extends Realizes(
      HeldDispatch,
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(HeldDispatch.queries)
    ):

  object controller
      extends Controller(
        everyCase(startUnreached),
        perform(history.dispatch -> holdDispatch),
        perform(client.pause -> pauseActivity),
        onPath(client.pause)(pauseDescribed.await(AdmissionFact.statusPaused)),
        perform(activityWorker.poll -> releaseDispatch),
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
    extends Realizes(
      LostStartAnswer,
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(LostStartAnswer.queries)
    ):

  object controller
      extends Controller(
        everyCase(startUnreached),
        perform(history.dispatch -> holdDispatch),
        perform(foundations.taskqueue.fault.ackLoss -> loseAdmissionResponse),
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
