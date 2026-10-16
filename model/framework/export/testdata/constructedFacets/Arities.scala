package fixture.facetsshapes

import framework.dsl.*
import framework.dsl.Finite.given

given Ok[Boolean] = Ok(true)

object hand extends Actor:
  val zero = action(this)
  val one = action(this).input[Boolean]("s")
  val two = action(this).inputs[Two]
  val three = action(this).inputs[Three]
  val four =
    action(this).inputs[Four]
  val five = action(this)
    .inputs[Five]
  // Native record inputs (fn-165): each multi-input action declares one record whose fields are
  // its old inputs in order (declared below), and its rules read it as `(s, in) => ...`. The
  // actions keep the lines their input chains started on, so that the frozen positions of the
  // actions, rules and effects hold.
  val six = action(this)
    .inputs[Six]
final case class Two(s: Boolean, state: Boolean)
final case class Three(a: Boolean, b: Boolean, c: Boolean)
final case class Four(a: Boolean, b: Boolean, c: Boolean, d: Boolean)
final case class Five(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean)
final case class Six(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean)

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
      on(hand.two)(always ~> ((s: Boolean, in: Two) => effects.two(s, in.s, in.state)))
      on(hand.three)(always ~> ((s: Boolean, in: Three) => effects.three(s, in.a, in.b, in.c)))
      on(hand.four)(always ~> ((s: Boolean, in: Four) => effects.four(s, in.a, in.b, in.c, in.d)))
      on(hand.five)(always ~> ((s: Boolean, in: Five) => effects.five(s, in.a, in.b, in.c, in.d, in.e)))
      on(hand.six)(always ~> ((s: Boolean, in: Six) => effects.six(s, in.a, in.b, in.c, in.d, in.e, in.f)))

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
