package fixture.features.witnesses

import framework.*
import framework.Finite.given

enum Bit:
  case first, second
enum Outcome:
  case ok
enum Fact:
  case done
final case class State(lit: Boolean)
abstract class Reader(@scala.annotation.unused p: Property[State])

object worker extends Actor:
  val finish = action(this).input[Bit]("bit")
  val repeat = action(this).input[Bit]("bit")
  val custom = action(this)
  val stateOnly = action(this)
  val customStart = action(this)
  val fake = action(this)
  val negativeFact = action(this)
  val factory = action(this)
  val core = action(this)
  val unrelated = action(this)
  val statement = action(this)
  val parent = action(this)

given Ok[Outcome] = Ok(Outcome.ok)
val two = Limits(steps = 2, actions = 2, search = 64)

object Witnesses extends Machine[State, Outcome, Fact]:
  val init = State(false)
  def end(s: State) = s.lit || !s.lit
  object states:
    def records(s: Step[State, Outcome, Fact]): Boolean = s.state.lit
    def observe(p: Property[State]): Unit = ()
  object effects:
    def finish(s: State, bit: Bit) =
      enter(s.copy(lit = bit == Bit.first || s.lit), Fact.done)
    def complete(s: State) = enter(s.copy(lit = true), Fact.done)
    def repeat(s: State, bit: Bit) =
      enter(s.copy(lit = bit == Bit.first || s.lit), Fact.done)
  object monitors:
    val litSticks = sticky[State, Outcome, Fact](_.state.lit)
  object rules extends Rules:
    on(worker.finish)(where(_ => true) ~> effects.finish)
    on(worker.repeat)(where(_ => true) ~> effects.repeat)
    on(worker.custom)(where(_ => true) ~> effects.complete)
    on(worker.stateOnly)(where(_ => true) ~> effects.complete)
    on(worker.customStart)(where(_ => true) ~> effects.complete)
    on(worker.fake)(where(_ => true) ~> effects.complete)
    on(worker.negativeFact)(where(_ => true) ~> effects.complete)
    on(worker.factory)(where(_ => true) ~> effects.complete)
    on(worker.core)(where(_ => true) ~> effects.complete)
    on(worker.unrelated)(where(_ => true) ~> effects.complete)
    on(worker.statement)(where(_ => true) ~> effects.complete)
    on(worker.parent)(where(_ => true) ~> effects.complete)
  object properties:
    def manufactured(action: ClassRef) =
      property when action holds (_.records(Fact.done))
    object safety:
      val privateClaim =
        property when worker.finish(Bit.first) holds (s => s.records(Fact.done) && s.state.lit)
      val distinctInputClaim =
        property when worker.finish(Bit.second) holds (_.records(Fact.done))
      val sharedClaim =
        property when worker.finish(Bit.first) holds (s => s.records(Fact.done) && !s.state.lit)
      val sharedScenarioClaim =
        property when worker.custom holds (_.records(Fact.done))
      val sharedScenarioOtherClaim = property.holds(_.state.lit)
      val repeatedClaim =
        property when worker.repeat(Bit.first) holds (_.records(Fact.done))
      val customStartClaim =
        property when worker.customStart holds (_.records(Fact.done))
      val stateOnly = property when worker.stateOnly holds (_.state.lit)
      val monitorPromise = property.holds(_ => true)
      val fakeClaim = property when worker.fake holds (s => states.records(s))
      val negativeClaim =
        property when worker.negativeFact holds (s => !s.records(Fact.done))
      val factoryClaim = manufactured(worker.factory())
      val coreClaim =
        property when worker.core holds (s => s.facts.contains(Fact.done))
      val unrelatedClaim =
        property when worker.unrelated holds (s =>
          List(Fact.done).contains(Fact.done) && s.state.lit
        )
      val statementClaim = property when worker.statement holds (_.records(Fact.done))
      val parentClaim = property when worker.parent holds (_.records(Fact.done))
  object queries extends Reader(properties.safety.parentClaim):
    val privatePath = scenario.actions(worker.finish(Bit.first))
    val distinctInputPath =
      scenario.actions(worker.finish(Bit.first), worker.finish(Bit.second))
    val sharedPath = scenario.actions(worker.finish(Bit.first))
    val sharedScenarioPath = scenario.actions(worker.custom)
    val repeatedPath =
      scenario.actions(worker.repeat(Bit.first), worker.repeat(Bit.first))
    val customPath = scenario.starts(State(true)).actions(worker.customStart)
    val statePath = scenario.actions(worker.stateOnly)
    val fakePath = scenario.actions(worker.fake)
    val negativePath = scenario.actions(worker.negativeFact)
    val factoryPath = scenario.actions(worker.factory)
    val corePath = scenario.actions(worker.core)
    val unrelatedPath = scenario.actions(worker.unrelated)
    val statementPath = scenario.actions(worker.statement)
    val parentPath = scenario.actions(worker.parent)
    states.observe(properties.safety.statementClaim)
    val eligible = query find properties.safety.privateClaim in privatePath limits two
    val distinctInputsEligible =
      query find properties.safety.distinctInputClaim in distinctInputPath limits two
    val shared = query find properties.safety.sharedClaim in sharedPath limits two
    val invariant = query verify properties.safety.sharedClaim in sharedPath limits two
    val monitorOnly =
      query find properties.safety.monitorPromise in sharedPath limits two
    val sharedScenario =
      query find properties.safety.sharedScenarioClaim in sharedScenarioPath limits two
    val sharedScenarioOther =
      query verify properties.safety.sharedScenarioOtherClaim in sharedScenarioPath limits two
    val repeated =
      query find properties.safety.repeatedClaim in repeatedPath limits two
    val customStart =
      query verify properties.safety.customStartClaim in customPath limits two
    val nonRecorded = query find properties.safety.stateOnly in statePath limits two
    val fakeRecords = query find properties.safety.fakeClaim in fakePath limits two
    val negativeRecords =
      query find properties.safety.negativeClaim in negativePath limits two
    val factoryEligible =
      query find properties.safety.factoryClaim in factoryPath limits two
    val coreEligible = query find properties.safety.coreClaim in corePath limits two
    val unrelatedContains =
      query find properties.safety.unrelatedClaim in unrelatedPath limits two
    val statementReader = query find properties.safety.statementClaim in statementPath limits two
    val parentReader = query find properties.safety.parentClaim in parentPath limits two
