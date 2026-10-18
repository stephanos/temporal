package temporal
package features.activity
package standalone
package realization

import protocol.ActivityProtocol

import framework.*
import framework.realize.*
import temporal.realize.*
import temporal.server.api.testpilot.v1.ActivityAttemptResponse.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED

object ResetAfterHeartbeatRun
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
        ServerStep(
          deadline.heartbeat,
          CauseKind.timer,
          Some(timerDeadline),
          TimeoutBasis.heartbeat
        ),
        deferredResetSettlement
      )
