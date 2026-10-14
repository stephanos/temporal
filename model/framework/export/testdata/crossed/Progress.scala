// A composition's progress claim reads its composed state and states its bound, which the compiler
// holds it to before anything is lifted.
package fixture.crossed

import framework.dsl.*

enum Level:
  case empty, full

final case class Tank(level: Level)

enum Filled:
  case done

final case class Pair(left: Tank, right: Tank)

val fill = action(Actor("pump"))

object Tanks extends Machine[Tank, Filled, Nothing]:
  val init = Tank(Level.empty)
  def end(t: State) = true
  object rules extends Bindings(fill ~> (t => List(Step(Filled.done, t, Nil))))

object Tanked extends Composition[Pair](_.left -> Tanks, _.right -> Tanks):
  def end(p: State) = true

// A predicate of a member's state, not of the composed state.
val memberPredicate = Tanked.leadsTo("member")((_: Tank) => true, _ => true, 1)

// No bound.
val unbounded = Tanked.leadsTo("unbounded")(_ => true, _ => true)
