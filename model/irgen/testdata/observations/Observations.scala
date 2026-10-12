package fixture.observations

import framework.*
import framework.realize.*
import temporal.realize.{Role, RoleKind}
import io.temporal.api.activity.v1.ActivityExecutionInfo
import io.temporal.api.workflowservice.v1.{DescribeActivityExecutionResponse, WorkflowServiceGrpc}

object declarations:
  val order = Entity(key = "activityId")
  val inferred = Observation(on = order)
  val explicit = Observation("explicit-info", order)
  val alias = inferred

enum State:
  case idle
enum Outcome:
  case accepted
enum Fact:
  case inferred, explicit, aliased

object client extends Actor:
  val inspect = action(this).on(declarations.order)

object Job extends Machine[State, Outcome, Fact]:
  val init = State.idle
  def end(s: State) = true
  val evidence: PartialFunction[Fact, String] = {
    case Fact.inferred => declarations.inferred.name
    case Fact.explicit => declarations.explicit.name
    case Fact.aliased  => declarations.alias.name
  }
  object effects:
    def inspect(s: State) = List(Step(Outcome.accepted, s, List(Fact.inferred)))
  object rules extends Bindings(client.inspect ~> effects.inspect)

// DescribeActivityExecutionResponse.info: the complete ActivityExecutionInfo message.
val describe = Instruction.rpc("endpoint", WorkflowServiceGrpc.METHOD_DESCRIBE_ACTIVITY_EXECUTION)(
  Vector.empty,
  Vector(
    ResponseRead.typed(
      Field[DescribeActivityExecutionResponse, ActivityExecutionInfo](_.getInfo),
      Cardinality.one,
      Vector(Target.Observe(declarations.alias.name), Target.Observe(declarations.explicit.name))
    )
  )
)
val readInfo = command(describe)

val run = Realization(
  machine = Job,
  producer = "fixture.observations",
  producerVersion = "1",
  roles = Vector(Role("endpoint", RoleKind.endpoint)),
  correlation = Correlation("projection", "run", "order", "inferred", 8, 8, 2, 16, 1000, 512),
  observations = Vector(
    Observed[ActivityExecutionInfo](declarations.inferred.name),
    Observed[ActivityExecutionInfo](declarations.explicit.name)
  ),
  scripts = Vector(Script("controller", Activation.Controller, Vector(everyCase(readInfo))))
)
