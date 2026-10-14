package fixture.headerclient

import framework.dsl.*
import fixture.constructedheader.{Inner, Mode, Outcome, State}
import fixture.headerlibrary.Generic

object Inherited extends Generic[State, Outcome, Nothing]:
  val init = State(Inner(Mode.ready, false))
  def end(s: State) = true
  object rules extends Bindings()
object DerivedInherited extends Derived(Inherited.unmonitored)
