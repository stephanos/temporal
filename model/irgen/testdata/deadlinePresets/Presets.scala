package fixture.deadlinepresets

import framework.*
import framework.outcomes.Outcome
import framework.realize.*
import temporal.realize.*
import temporal.features.activity.standalone.{client, maxAttempts, startToClose, MaxAttempts}
import temporal.features.activity.standalone.system.ActivitySystem
import temporal.features.activity.Timeout as ActivityTimeout
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest as StartRequest
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*

enum Timeout derives Finite:
  case unset, expires

val delay = input[Timeout]
val close = input[Timeout]
val device = Entity(key = "activityId")
object caller extends Actor:
  val start = action(this).input(delay).input(close).creates(device)

final case class DeviceState(done: Boolean) derives Finite
object Device extends Machine[DeviceState, Outcome, Nothing]:
  val init = DeviceState(false)
  def end(s: DeviceState) = s.done
  object effects:
    def start(s: DeviceState, a: Timeout, b: Timeout) = enter(s.copy(done = a == b))
  object rules extends Rules:
    on(caller.start) {
      always ~> effects.start
    }

private val calls =
  RequestBase(workflowService, "namespace" -> workerNamespace, "activity_id" -> run)
private val launch = rpc(calls, METHOD_START_ACTIVITY_EXECUTION) {}
private val value = duration(2)

object Legacy extends Realizes(Device):
  object controller
      extends Controller(
        deadlines[StartRequest](caller.start, launch, value)(
          delay.sets(_.getStartDelay),
          close.sets(_.getStartToCloseTimeout)
        ),
        onPath(caller.start)(launch)
      )

object Bare extends Realizes(ActivitySystem):
  object controller
      extends Controller(
        deadlines[StartRequest](client.start, launch, value)(
          startToClose.sets(_.getStartToCloseTimeout)
        )
      )

object Catalog extends Realizes(ActivitySystem):
  object controller
      extends Controller(
        deadlines[StartRequest](
          client.start(maxAttempts := MaxAttempts.unlimited),
          launch,
          value
        )(startToClose.sets(_.getStartToCloseTimeout)),
        deadlines[StartRequest](
          client.start(maxAttempts := MaxAttempts.one),
          launch,
          value
        )(startToClose.sets(_.getStartToCloseTimeout)),
        deadlines[StartRequest](
          client.start(maxAttempts := MaxAttempts.two),
          launch,
          value
        )(startToClose.sets(_.getStartToCloseTimeout)),
        onPath(client.start)(launch)
      )

object ExpiringPreset extends Realizes(ActivitySystem):
  object controller
      extends Controller(
        deadlines[StartRequest](
          client.start(startToClose := ActivityTimeout.expires, maxAttempts := MaxAttempts.one),
          launch,
          value
        )(startToClose.sets(_.getStartToCloseTimeout))
      )

object DuplicatePreset extends Realizes(ActivitySystem):
  object controller
      extends Controller(
        deadlines[StartRequest](
          client.start(maxAttempts := MaxAttempts.two),
          launch,
          value
        )(
          startToClose.sets(_.getStartToCloseTimeout)
        ),
        deadlines[StartRequest](
          client.start(maxAttempts := MaxAttempts.two),
          launch,
          value
        )(
          startToClose.sets(_.getStartToCloseTimeout)
        )
      )
