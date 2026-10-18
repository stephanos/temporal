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

object ExhaustAfterHeartbeatRun
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
        delivered(
          protocol.Fact.attemptCount,
          heartbeatAttempts,
          attempt = 2,
          after = startActivity
        ),
        described(everyValue(protocol.Fact.statusTimedOut))
      )
