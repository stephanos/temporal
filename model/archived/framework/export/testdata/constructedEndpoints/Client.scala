package fixture.endpointclient

import framework.dsl.*
import fixture.endpointlibrary.Helpers

object HelperJar extends Machine[Boolean, Boolean, Nothing]:
  def init = Helpers.forward(false)
  def end(authored: Boolean) = Helpers.forward(authored)
  object rules extends Bindings()
