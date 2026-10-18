package temporal
package features.activity
package standalone
package realization

import protocol.ActivityProtocol

import framework.*
import framework.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest
import temporal.server.api.testpilot.v1.ActivityAttemptResponse.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED

object HeartbeatThenCompletionRun
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
