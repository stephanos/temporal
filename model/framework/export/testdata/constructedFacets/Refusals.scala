package fixture.facetsrefusals

import framework.dsl.*

given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)
object clock:
  val tick = timer
object fault extends Actor:
  val stop = action(this)

trait Left extends Facet[Boolean, Boolean, Nothing]:
  object left:
    object rules extends Rules:
      on(hand.go)(where(s => !s) ~> ((s: Boolean) => stay[Boolean, Boolean, Nothing](s)))
      on(clock.tick)(where(s => !s) ~> ((s: Boolean) => stay[Boolean, Boolean, Nothing](s)))
      disabled(fault.stop)

trait Right extends Facet[Boolean, Boolean, Nothing]:
  object right:
    object rules extends Rules:
      on(hand.go)(where(s => s) ~> ((s: Boolean) => stay[Boolean, Boolean, Nothing](s)))
      on(clock.tick)(where(s => s) ~> ((s: Boolean) => stay[Boolean, Boolean, Nothing](s)))
      disabled(fault.stop)

object Duplicate extends Machine[Boolean, Boolean, Nothing], Left, Right:
  val init = false
  def end(s: Boolean) = s
  object rules extends Facets(left.rules, right.rules)

object Repeated extends Machine[Boolean, Boolean, Nothing], Left:
  val init = false
  def end(s: Boolean) = s
  object rules extends Facets(left.rules, left.rules)

trait EmptyEffects extends Facet[Boolean, Boolean, Nothing]:
  object emptyEffects:
    object effects:
      def unused(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = if s then Nil else Nil
    object rules extends Rules:
      on(hand.go)(always ~> ((s: Boolean) => stay[Boolean, Boolean, Nothing](s)))

object EmptyEffect extends Machine[Boolean, Boolean, Nothing], EmptyEffects:
  val init = false
  def end(s: Boolean) = s
  object rules extends Facets(emptyEffects.rules)

object Unowned extends Machine[Boolean, Boolean, Nothing], Left:
  val init = false
  def end(s: Boolean) = s
  object loose extends Rules:
    disabled(hand.go)
  object rules extends Facets(loose)
