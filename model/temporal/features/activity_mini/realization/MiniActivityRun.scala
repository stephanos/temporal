package temporal
package features.activity
package mini
package realization

import protocol.ActivityProtocol
import framework.dsl.*
import framework.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest

object MiniActivityRun extends Realizes(
  ActivityProtocol,
  runs = Seq(ActivityProtocol.queries.protocolCompleted, ActivityProtocol.queries.retryCompleted)
):
  object controller extends Controller(
    deadlines[StartActivityExecutionRequest](
      client.start(maxAttempts := MaxAttempts.one), startOne, duration(deadlineSeconds),
      unset = Some(startToClose -> startOneUnreached)
    )(
      scheduleToStart.sets(_.getScheduleToStartTimeout),
      startToClose.sets(_.getStartToCloseTimeout)
    ),
    deadlines[StartActivityExecutionRequest](
      client.start(maxAttempts := MaxAttempts.two), startTwo, duration(deadlineSeconds),
      unset = Some(startToClose -> startTwoUnreached)
    )(
      scheduleToStart.sets(_.getScheduleToStartTimeout),
      startToClose.sets(_.getStartToCloseTimeout)
    ),
    onPath(deadline.scheduleToStart)(holdDispatchBeforeTimeout),
    perform(client.cancel -> requestCancelActivity),
    onPath(activityWorker.respondCompleted)(described.await(protocol.Fact.statusCompleted)),
    onPath(activityWorker.respondFailed(Failure.fatal))(described.await(protocol.Fact.statusFailed)),
    onPath(client.cancel)(described.await(protocol.Fact.statusCanceled)),
    onPath(deadline.scheduleToStart, deadline.startToClose)(described.await(protocol.Fact.statusTimedOut))
  )

  object workers extends Workers(attempts)

  object evidence extends Evidences(
    answered(protocol.Fact.statusScheduled, startActivity),
    delivered(protocol.Fact.statusStarted, attempts, attempt = 1, after = startActivity,
      Taking(activityWorker.poll, 1)),
    answered(protocol.Fact.statusCancelRequested, requestCancelActivity),
    described(protocol.Fact.statusCompleted),
    described(protocol.Fact.statusFailed),
    described(protocol.Fact.statusCanceled),
    described(protocol.Fact.statusTimedOut)
  )

  object controls extends Controls(unstartedDispatch)
