package fixture.actorhelpers

import framework.dsl.*

object Helpers:
  var calls = 0
  var getters = 0
  class Base extends Actor("base")
  object `Named-helper` extends Base:
    override def name: String =
      getters += 1
      "public-helper"
  def actor(name: String): Actor =
    calls += 1
    Actor(name)
