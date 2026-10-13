// fn-167.2: gaps the lift refuses. Each passes source admission; what it names resolves to nothing
// on its declaration, is declared twice, or names an expiry no Flow spec answers.
package fixture.gapresolution

import framework.dsl.*
import framework.realize.{everyCase, Activation, Correlation, Observed, Realization, Script}
import temporal.realize.{Role, RoleKind}
import io.temporal.api.activity.v1.ActivityExecutionInfo

def unboundMove(s: Job, m: Mode) =
  if s.stage == Stage.idle && m == Mode.fast then stay[Job, Outcome, Fact](s) else disabled

object invalid:
  @gap(neverEnabled(caller.finish))(because = "unbound action")
  @gap(silent(caller.move, in = Mode.fast))(because = "foreign phase")
  @gap(silent[Stage](caller.move))(because = "no phases")
  @gap(uncovered(caller.move))(because = "a realization's kind on a machine")
  @gap(unevidenced(Fact.moved))(because = "no realization of the machine is lifted")
  @gap(unproduced(Mode.fast))(because = "a mode is no outcome or fact")
  @gap(unreachable[Mode](Mode.fast))(because = "no field holds a mode")
  object Unbound
      extends Machine[Job, Outcome, Fact],
        Phased[Job, Stage](_.stage, (s, p) => s.copy(stage = p)):
    val init = Job(Stage.idle, false)
    object rules extends Bindings(caller.move ~> unboundMove)

  @gap(unconstrained(caller.move))(because = "every class")
  @gap(unconstrained(caller.move(Mode.fast)))(because = "one class again")
  object Twice
      extends Machine[Job, Outcome, Fact],
        Phased[Job, Stage](_.stage, (s, p) => s.copy(stage = p)):
    val init = Job(Stage.idle, false)
    object rules extends Bindings(caller.move ~> unboundMove)

  @gap(stuck(Job(Stage.busy, true)))(because = "missing", until = "fn-999999")
  @gap(stuck(Job(Stage.busy, false)))(because = "ambiguous", until = "fn-5")
  @gap(stuck(Job(Stage.idle, true)))(because = "malformed", until = "soon")
  object Expiring
      extends Machine[Job, Outcome, Fact],
        Phased[Job, Stage](_.stage, (s, p) => s.copy(stage = p)):
    val init = Job(Stage.idle, false)
    object rules extends Bindings(caller.move ~> unboundMove)

  object Asking
      extends Machine[Job, Outcome, Fact],
        Phased[Job, Stage](_.stage, (s, p) => s.copy(stage = p)):
    val init = Job(Stage.idle, false)
    object rules extends Bindings(caller.move ~> unboundMove)
    object properties:
      val idle = property.holds(s => s.state.stage != Stage.done)
    object queries:
      val free = scenario.free
      @gap(unrealized)(because = "a verify Query is no find")
      val verified = query verify properties.idle in free limits two
      @gap(witnessOnly)(because = "witnessOnly is a Property's")
      val checked = query verify properties.idle in free limits two

  @gap(unperformed(caller.finish))(because = "first run")
  val first = Realization(
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

  @gap(unperformed(caller.finish))(because = "second run")
  val second = Realization(
    machine = Worker,
    producer = "fixture.gapresolution",
    producerVersion = "1",
    roles = Vector(Role("endpoint", RoleKind.endpoint)),
    correlation = Correlation("projection", "run", "order", "seen", 8, 8, 2, 16, 1000, 512),
    observations = Vector(Observed[ActivityExecutionInfo](declarations.seen.name)),
    scripts = Vector(Script("controller", Activation.Controller, Vector(everyCase(readInfo))))
  )
