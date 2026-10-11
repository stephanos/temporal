package temporal
package features.activity
package standalone
package realization

import protocol.ActivityProtocol

import framework.*
import framework.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest

// One standalone activity a controller starts and the Case's own worker runs. An activity is
// scheduled by its start, again by an unpause, and again by a retried failure. Each has evidence
// that stays true: the start's answer, the unpause's answer, and the second attempt's delivery,
// which shows at once that the first attempt failed, that the activity was scheduled again, and
// that a worker took it again. Its server steps are derived: an attempt starts when the server
// delivers it, a retry waits out the backoff timer at the server's first retry interval, after which
// the retried attempt is dispatched (chasm/lib/activity/attempt.go:72-82, statemachine.go:393-420),
// and a timeout class fires at the deadline its start sets.
object StandaloneActivityRun
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
