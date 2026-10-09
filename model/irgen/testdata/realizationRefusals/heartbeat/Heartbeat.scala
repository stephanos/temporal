package fixture.heartbeat

import framework.*
import framework.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION

enum Timeout derives Finite:
  case unset, expires
enum Fact derives Finite:
  case started
final case class State(active: Boolean) derives Finite

val heartbeat = input[Timeout]
val job = Entity("job", key = "job_id")
object client extends Actor:
  val start = action(this).creates(job).input(heartbeat)
object timers:
  val heartbeat = timer.on(job)

object Job extends Machine[State, Boolean, Fact]:
  val init = State(false)
  def end(s: State) = s.active
  object rules
      extends Bindings(
        client.start ~> ((s: State, _: Timeout) =>
          List(Step(true, s.copy(active = true), List(Fact.started)))
        ),
        timers.heartbeat ~> ((s: State) => List(Step(true, s)))
      )

val start = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {}
object FieldBasis extends Realizes(Job):
  object controller
      extends Controller(
        deadlines[StartActivityExecutionRequest](client.start, start, duration(deadlineSeconds))(
          heartbeat.sets(_.getStartToCloseTimeout)
        )
      )

object HeartbeatBasis extends Realizes(Job):
  object controller
      extends Controller(
        deadlines[StartActivityExecutionRequest](client.start, start, duration(deadlineSeconds))(
          heartbeat.sets(_.getHeartbeatTimeout)
        )
      )

object CloseBasis extends Realizes(Job):
  object controller
      extends Controller(
        deadlines[StartActivityExecutionRequest](client.start, start, duration(deadlineSeconds))(
          heartbeat.sets(_.getScheduleToCloseTimeout)
        )
      )

object UnsupportedBasis extends Realizes(Job):
  object controller
      extends Controller(
        deadlines[StartActivityExecutionRequest](client.start, start, duration(deadlineSeconds))(
          heartbeat.sets(_.getScheduleToStartTimeout)
        )
      )

object MissingBasis extends Realizes(Job):
  object controller
      extends Controller(
        perform(client.start(Timeout.expires) -> start)
      )

val heartbeatCommand = attemptHeartbeat(
  Proto[io.temporal.api.common.v1.Payloads](
    ProtoField.typed(
      Field(_.payloads),
      ProtoValue.messages(jsonPayload("first"), jsonPayload("second"))
    )
  )
)
val withheld = attemptWithheld
val pending = attemptPending
object Commands extends Realizes(Job):
  object controller
      extends Controller(
        everyCase(heartbeatCommand),
        everyCase(withheld),
        everyCase(pending)
      )
