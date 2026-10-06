package fixture.binding

import umpire.*

object firstCycle:
  val job = Entity()
  lazy val take: Action[(Boolean, Resolution)] = secondCycle.take.on(job)
object secondCycle:
  val job = Entity()
  lazy val take: Action[(Boolean, Resolution)] = firstCycle.take.on(job)

object CyclicForm extends Machine[State, Outcome, Fact]:
  val init = State(false)
  def end(s: State) = s.done
  object rules extends Bindings(
    firstCycle.take ~> ((s, _, _) => List(Step(Outcome.accepted, s)))
  )
