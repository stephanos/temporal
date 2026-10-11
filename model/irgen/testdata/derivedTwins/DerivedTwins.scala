// One shared def that builds a Query of one name over two machines of one package, each instance in
// an IR file of its own (fn-126 decision 23). The two records share the def's position, yet derive
// one ID, `fixture.derivedtwins.query.anyLit`, over two machines: the run refuses the second.
package fixture.features.derivedtwins

import framework.*

final case class LampState(lit: Boolean)

enum Outcome:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

object user extends Actor:
  val flip = action(this)

val two = Limits(steps = 2, actions = 2, search = 64)

object Lamp extends Machine[LampState, Outcome, Nothing]:
  val init = LampState(false)
  def end(lamp: State) = true
  object effects:
    def flipStep(l: LampState): List[Step[LampState, Outcome, Nothing]] = enter(
      LampState(!l.lit)
    )
  object rules extends Rules:
    on(user.flip)(always ~> effects.flipStep)
  object queries:
    def litQueries(m: Machine[LampState, Outcome, Nothing]): Vector[Query] =
      Vector(
        query("anyLit") find (m.property(s"${m.name}.lit") holds (after => after.state.lit)) in m
          .scenario("any")
          .free limits two
      )
    val lampQueries = litQueries(Lamp)

object PlainLamp extends Machine[LampState, Outcome, Nothing]:
  val init = LampState(true)
  def end(lamp: State) = true

  object effects:
    def flipStep(l: LampState): List[Step[LampState, Outcome, Nothing]] = enter(
      LampState(!l.lit)
    )
  object rules extends Rules:
    on(user.flip)(always ~> effects.flipStep)
  object queries:
    val plainLampQueries = Lamp.queries.litQueries(PlainLamp)
