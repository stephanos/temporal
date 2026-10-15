package fixture.endpointrefused

import framework.dsl.*

object Unsupported extends Machine[Boolean, Boolean, Nothing]:
  def init: Boolean = throw AssertionError("unsupported endpoint")
  def end(s: Boolean) = s
  object rules extends Bindings()
