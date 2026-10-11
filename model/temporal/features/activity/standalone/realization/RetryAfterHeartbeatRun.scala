package temporal
package features.activity
package standalone
package realization

import protocol.ActivityProtocol

import framework.*
import framework.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest
import temporal.server.api.testpilot.v1.ActivityAttemptResponse.ACTIVITY_ATTEMPT_RESPONSE_PENDING

object RetryAfterHeartbeatRun
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
        delivered(
          protocol.Fact.attemptCount,
          heartbeatAttempts,
          attempt = 2,
          after = startActivity
        ),
        attemptRecord(heartbeatAttempts, 2, startActivity)(
          "heartbeatTimedOutKind",
          protocol.Fact.heartbeatTimedOut
        )
      )
