package fixture.facetsshapes

import framework.dsl.*
import framework.dsl.Finite.given

given Ok[Boolean] = Ok(true)

object hand extends Actor:
  val zero = action(this)
  val one = action(this).input[Boolean]("s")
  val two = action(this).input[Boolean]("s").input[Boolean]("state")
  val three = action(this).input[Boolean]("a").input[Boolean]("b").input[Boolean]("c")
  val four =
    action(this).input[Boolean]("a").input[Boolean]("b").input[Boolean]("c").input[Boolean]("d")
  val five = action(this)
    .input[Boolean]("a")
    .input[Boolean]("b")
    .input[Boolean]("c")
    .input[Boolean]("d")
    .input[Boolean]("e")
  val six = action(this)
    .input[Boolean]("a")
    .input[Boolean]("b")
    .input[Boolean]("c")
    .input[Boolean]("d")
    .input[Boolean]("e")
    .input[Boolean]("f")

trait AritiesFacet extends Facet[Boolean, Boolean, Nothing]:
  object aritiesFacet:
    object effects:
      def zero(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
      def one(s: Boolean, a: Boolean) = stay[Boolean, Boolean, Nothing](s && a)
      def two(s: Boolean, a: Boolean, b: Boolean) = stay[Boolean, Boolean, Nothing](s && a && b)
      def three(s: Boolean, a: Boolean, b: Boolean, c: Boolean) =
        stay[Boolean, Boolean, Nothing](s && a && b && c)
      def four(s: Boolean, a: Boolean, b: Boolean, c: Boolean, d: Boolean) =
        stay[Boolean, Boolean, Nothing](s && a && b && c && d)
      def five(s: Boolean, a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean) =
        stay[Boolean, Boolean, Nothing](s && a && b && c && d && e)
      def six(s: Boolean, a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean) =
        stay[Boolean, Boolean, Nothing](s && a && b && c && d && e && f)
    object rules extends Rules:
      on(hand.zero)(where(s => !s).where(s => s == false) ~> effects.zero)
      on(hand.one)(always ~> effects.one)
      on(hand.two)(always ~> effects.two)
      on(hand.three)(always ~> effects.three)
      on(hand.four)(always ~> effects.four)
      on(hand.five)(always ~> effects.five)
      on(hand.six)(always ~> effects.six)

object Arities extends Machine[Boolean, Boolean, Nothing], AritiesFacet:
  val init = false
  def end(s: Boolean) = s
  object rules extends Facets(aritiesFacet.rules)

enum Phase:
  case open
  case done extends Phase, Closed
final case class Snapshot(phase: Phase)
trait PhaseRulesFacet extends Facet[Snapshot, Boolean, Nothing], Phased[Snapshot, Phase]:
  object phaseRulesFacet:
    object states:
      def closed(p: Phase) = p == Phase.done
    object effects:
      def keep(s: Snapshot) = stay[Snapshot, Boolean, Nothing](s)
    object rules extends Rules:
      on(hand.zero):
        when(Phase.open) ~> effects.keep
        when(states.closed) ~> effects.keep
      on(hand.one)(when[Closed] ~> effects.keep)

object PhaseRules
    extends Machine[Snapshot, Boolean, Nothing],
      Phased[Snapshot, Phase](_.phase),
      PhaseRulesFacet:
  val init = Snapshot(Phase.open)
  object rules extends Facets(phaseRulesFacet.rules)
