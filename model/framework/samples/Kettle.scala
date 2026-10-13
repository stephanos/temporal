// Counterpart: quint-co/quint/examples/tutorials/repl/kettle.qnt
package samples

import framework.dsl.*
import framework.dsl.Finite.given

object Kettle:
  enum Phase:
    case off, heating, boiling

  final case class Snapshot(phase: Phase)

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  object user extends Actor:
    val press = action(this)

  object heater:
    val warm = internal
    val cool = internal

  object Model extends Machine[Snapshot, Outcome, Nothing], Phased[Snapshot, Phase](_.phase):
    val init = Snapshot(Phase.off)
    override def end(state: Snapshot) =
      state.phase == Phase.off || state.phase == Phase.boiling

    object effects:
      def start(state: Snapshot) =
        enter[Snapshot, Kettle.Outcome, Nothing](state.copy(phase = Phase.heating))
      def boil(state: Snapshot) =
        enter[Snapshot, Kettle.Outcome, Nothing](state.copy(phase = Phase.boiling))
      def stop(state: Snapshot) =
        enter[Snapshot, Kettle.Outcome, Nothing](state.copy(phase = Phase.off))

    object rules extends Rules:
      on(user.press)(when(Phase.off) ~> effects.start)
      on(heater.warm)(when(Phase.heating) ~> effects.boil)
      on(heater.cool)(when(Phase.heating, Phase.boiling) ~> effects.stop)

  val heatsBeforeBoiling: Property[Snapshot] = Model.property holdsAcross { (before, after) =>
    before.phase != Phase.off || after.state.phase != Phase.boiling
  }
  val anyRun: Scenario[Snapshot] = Model.scenario.starts(Model.init).free
  val short = Limits(steps = 3, actions = 3, search = 64)
  val queries = Vector(query verify heatsBeforeBoiling in anyRun limits short)
