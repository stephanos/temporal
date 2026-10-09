// One shared def that builds a Query of one name over two machines of one package, each instance in
// an IR file of its own (fn-126 decision 23). The two records share the def's position, yet derive
// one ID, `fixture.derivedtwins.query.anyLit`, over two machines: the run refuses the second.
package fixture.derivedtwins

import framework.*

final case class LampState(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

val flip = action(Actor("user"))

def flipStep(l: LampState): List[Step[LampState, Outcome, Nothing]] = enter(LampState(!l.lit))

object Lamp extends Machine[LampState, Outcome, Nothing]:
  val init = LampState(false)
  def end(lamp: State) = true

  object rules extends Bindings(flip ~> flipStep)

object PlainLamp extends Machine[LampState, Outcome, Nothing]:
  val init = LampState(true)
  def end(lamp: State) = true

  object rules extends Bindings(flip ~> flipStep)

val two = Limits(steps = 2, actions = 2, search = 64)

// The Query each machine gets from one declaration, named alike for both.
def litQueries(m: Machine[LampState, Outcome, Nothing]): Vector[Query] = Vector(
  query("anyLit") find (m.property(s"${m.name}.lit") holds (after => after.state.lit)) in m
    .scenario("any")
    .free limits two
)

val lampQueries = litQueries(Lamp)
val plainLampQueries = litQueries(PlainLamp)

val lampFile = irFile("lamp")(lampQueries)
val plainLampFile = irFile("plainLamp")(plainLampQueries)
