package temporal
package features.activity
package standalone
package realization

import protocol.ActivityProtocol

import framework.*
import framework.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest

// Uses the unchanged System rows, but only the timeout path's second-delivery evidence. The first
// attempt withholds its answer until its armed deadline; the second completes or exhausts retries.
object RetryAfterTimeoutRun
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
