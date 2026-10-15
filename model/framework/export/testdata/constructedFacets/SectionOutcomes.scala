package fixture.facetsoutcomes

import framework.dsl.*

given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)

trait Unused extends Facet[Boolean, Boolean, Nothing]:
  object unused:
    object states:
      val ready = is(java.lang.Boolean.parseBoolean("true"))
    object effects:
      def keep(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
    object rules extends Rules:
      on(hand.go)(always ~> effects.keep)

trait Reached extends Facet[Boolean, Boolean, Nothing]:
  object reached:
    object states:
      val ready = is(java.lang.Boolean.parseBoolean("true"))
    object effects:
      def keep(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
    object rules extends Rules:
      on(hand.go)(where(states.ready) ~> effects.keep)

object Safe extends Machine[Boolean, Boolean, Nothing], Unused:
  val init = false
  def end(s: Boolean) = s
  object rules extends Facets(unused.rules)

object Bad extends Machine[Boolean, Boolean, Nothing], Reached:
  val init = false
  def end(s: Boolean) = s
  object rules extends Facets(reached.rules)
