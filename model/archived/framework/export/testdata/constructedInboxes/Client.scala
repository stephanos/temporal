package fixture.constructedinbox

import framework.dsl.*
import Helpers.given

object HelperMachine extends Machine[Helped, Boolean, Nothing]:
  val init = Helped(first.empty)
  def end(s: State) = true
  object rules extends Bindings()
