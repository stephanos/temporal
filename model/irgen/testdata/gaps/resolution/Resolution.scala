// fn-167.2: gaps on every supported placement, resolved to the lint's finding keys. `Worker` is a
// faceted machine grouped by its stage; `Plain` refines it and waives a capability Property; `Pair`
// composes two of `Plain`; `runs.one` and `runs.two` realize `Worker`.
package fixture.gapresolution

import framework.*
import framework.realize.*
import temporal.capabilities.product.Closable
import temporal.realize.{Role, RoleKind}
import io.temporal.api.activity.v1.ActivityExecutionInfo
import io.temporal.api.enums.v1.ActivityExecutionStatus
import io.temporal.api.workflowservice.v1.{DescribeActivityExecutionResponse, WorkflowServiceGrpc}

val two = Limits(steps = 2, actions = 2, search = 64)

enum Stage:
  case idle, busy
  case done extends Stage, Terminated

enum Mode:
  case fast, slow

final case class Job(stage: Stage, flag: Boolean)

enum Outcome:
  case accepted, refused

enum Fact:
  case moved, finished

given Ok[Outcome] = Ok(Outcome.accepted)

object declarations:
  val order = Entity(key = "jobId")
  val seen = Observation(on = order)
  val named = Observation("named-info", order)

object caller extends Actor:
  val move = action(this).input[Mode]("mode").on(declarations.order)
  val finish = action(this).on(declarations.order)

@gap(neverEnabled(caller.move(Mode.slow)))(because = "moving facet reason")
trait Moving extends Facet[Job, Outcome, Fact], Phased[Job, Stage]:
  object moving:
    object effects:
      val started = choice
      val held = choice
      def move(s: Job, m: Mode) =
        choose(started -> enters(Stage.busy)(s), held -> stay[Job, Outcome, Fact](s))

    object rules extends Rules:
      on(caller.move) {
        when(Stage.idle) ~> effects.move
      }

    object properties:
      @gap(witnessOnly, unasked)(because = "facet property reason")
      val moves = property.holds(s => s.state.stage != Stage.done)

@gap(neverEnabled(caller.finish))(because = "finishing facet reason")
trait Finishing extends Facet[Job, Outcome, Fact], Phased[Job, Stage]:
  object finishing:
    object effects:
      def finish(s: Job) = enter[Job, Outcome, Fact](s.copy(stage = Stage.done), Fact.finished)

    object rules extends Rules:
      on(caller.finish) {
        when(Stage.busy) ~> effects.finish
      }

@gap(silent(caller.move, in = Stage.done, Stage.busy))(
  because = "silent reason",
  until = "fn-167 R2"
)
@gap(
  byDefault(caller.finish, in = Stage.idle),
  mustNotPinned(caller.move(Mode.fast), in = Stage.busy)
)(
  because = "default reason"
)
@gap(unconstrained(caller.move(Mode.fast)), unproduced(Outcome.refused), unproduced(Fact.moved))(
  because = "table reason",
  until = "fn-167"
)
@gap(
  untaken(Worker.moving.effects.held),
  unreachable[Stage](Stage.done),
  stuck(Job(Stage.busy, true))
)(
  because = "reach reason"
)
@gap(unevidenced(Fact.moved), unreadObservation(declarations.named))(because = "evidence reason")
object Worker
    extends Machine[Job, Outcome, Fact],
      Phased[Job, Stage](_.stage, (s, p) => s.copy(stage = p)),
      Moving,
      Finishing:
  val init = Job(Stage.idle, false)

  object rules extends Facets(moving.rules, finishing.rules)

  object queries:
    val free = scenario.free
    val fastMove = scenario.actions(caller.move(Mode.fast))
    @gap(unfired)(because = "verify reason")
    val movesVerified = query verify moving.properties.moves in free limits two
    @gap(unrealized)(because = "find reason")
    val movesFound = query find moving.properties.moves in fastMove limits two

def plainMove(s: Job, m: Mode) =
  if s.stage == Stage.idle then stay[Job, Outcome, Fact](s) else disabled

object Plain
    extends Machine[Job, Outcome, Fact],
      Phased[Job, Stage](_.stage, (s, p) => s.copy(stage = p)):
  val init = Job(Stage.idle, false)

  @gap(unread)(because = "refinement reason")
  object refinement extends Refinement(Worker):
    def toProduct(s: Job): Job = s

  object monitors:
    val flagged = flaggedOnce
    val steady = calm

  object rules extends Bindings(caller.move ~> plainMove)

  object capabilities extends Capabilities:
    val closable: Capability = Closable(rejected = Outcome.refused)
    except(Closable.terminalStatesAreFinal[Job, Stage], because = "a fixture's waiver reason")

  object queries:
    capabilities.bound(two)

final case class Duo(left: Job, right: Job)

@gap(stuck(Duo(Job(Stage.busy, false), Job(Stage.idle, false))))(because = "composition reason")
object Pair extends Composition[Duo](_.left -> Plain, _.right -> Plain):
  def end(d: State) = d.left.stage == Stage.done
  object syncs extends Syncs

val describe = Instruction.rpc("endpoint", WorkflowServiceGrpc.METHOD_DESCRIBE_ACTIVITY_EXECUTION)(
  Vector.empty,
  Vector(
    ResponseRead.typed(
      Field[DescribeActivityExecutionResponse, ActivityExecutionInfo](_.getInfo),
      Cardinality.one,
      Vector(Target.Observe(declarations.seen.name), Target.Observe(declarations.named.name))
    )
  )
)
val readInfo = command(describe)

object runs:
  @gap(unperformed(caller.finish), uncovered(caller.move))(because = "first run reason")
  @gap(unmapped(ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_RUNNING), explicitWait(readInfo))(
    because = "reference reason"
  )
  val one = Realization(
    machine = Worker,
    producer = "fixture.gapresolution",
    producerVersion = "1",
    roles = Vector(Role("endpoint", RoleKind.endpoint)),
    correlation = Correlation("projection", "run", "order", "seen", 8, 8, 2, 16, 1000, 512),
    observations = Vector(
      Observed[ActivityExecutionInfo](declarations.seen.name),
      Observed[ActivityExecutionInfo](declarations.named.name)
    ),
    scripts = Vector(Script("controller", Activation.Controller, Vector(everyCase(readInfo))))
  )

  @gap(uncovered(caller.move(Mode.slow)), unreachableBinding(caller.move(Mode.fast)))(
    because = "second run reason"
  )
  val two = Realization(
    machine = Worker,
    producer = "fixture.gapresolution",
    producerVersion = "1",
    roles = Vector(Role("endpoint", RoleKind.endpoint)),
    correlation = Correlation("projection", "run", "order", "seen", 8, 8, 2, 16, 1000, 512),
    observations = Vector(Observed[ActivityExecutionInfo](declarations.seen.name)),
    scripts = Vector(Script("controller", Activation.Controller, Vector(everyCase(readInfo))))
  )
