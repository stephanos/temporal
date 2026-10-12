package fixture.facetcontract

import framework.*

enum FacetName:
  case pressing, breaking, stopping
enum Phase:
  case off, on
  case broken extends Phase, Terminated
enum Fact:
  case changed
enum Outcome:
  case ok
given Ok[Outcome] = Ok(Outcome.ok)
final case class State(phase: Phase)
object hand extends Actor:
  val press = action(this)
  val smash = action(this)
  val unused = action(this)
object clock:
  val tick = timer
object fault extends Actor:
  val stop = action(this)
val two = Limits(steps = 2, actions = 2, search = 64)

final case class Holdable[S, O, F](rejected: O) extends CapabilityOf[S, O, F]
object Holdable extends CapabilityKind:
  def keeps[S](m: Declares[S])(rejected: m.Outcome): Property[S] =
    m.property("keeps").holds(_.outcome == rejected)

trait Pressing extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object pressing:
    object effects:
      def settle(s: State) = enter[State, Outcome, Fact](s.copy(phase = Phase.on), Fact.changed)
    object rules extends Rules:
      on(hand.press) {
        when(Phase.off) ~> effects.settle
      }
    object capabilities extends Capabilities:
      val holdable: Capability = Holdable(rejected = Outcome.ok)

trait Breaking extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Pressing =>
  object breaking:
    object effects:
      def settle(s: State) = pressing.effects.settle(s)
    object rules extends Rules:
      on(hand.smash) {
        when(Phase.on) ~> effects.settle
      }

object Faceted extends Machine[State, Outcome, Fact],
    Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)), Pressing, Breaking:
  val init = State(Phase.off)
  object rules extends Facets(pressing.rules, breaking.rules)
  object capabilities extends FacetCapabilities(pressing.capabilities)
  object queries:
    capabilities.bound(two)

object Whole extends Machine[State, Outcome, Fact],
    Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)):
  val init = State(Phase.off)
  object effects:
    def press(s: State) = enter[State, Outcome, Fact](s.copy(phase = Phase.on), Fact.changed)
    def smash(s: State) = press(s)
  object rules extends Rules:
    on(hand.press) {
      when(Phase.off) ~> effects.press
    }
    on(hand.smash) {
      when(Phase.on) ~> effects.smash
    }
  object capabilities extends Capabilities:
    val holdable: Capability = Holdable(rejected = Outcome.ok)
  object queries:
    capabilities.bound(two)

trait Stopping extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object stopping:
    object rules extends Rules:
      disabled(fault.stop)
object Disabled extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase), Stopping:
  val init = State(Phase.off)
  object rules extends Facets(stopping.rules)

object Core extends Machine[State, Outcome, Fact]:
  val init = State(Phase.off)
  def end(s: State) = s.phase.in(Phase.broken)
  object rules extends Bindings(hand.press ~> Whole.effects.press)

object Unowned extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase), Pressing:
  val init = State(Phase.off)
  object loose extends Rules:
    disabled(hand.smash)
  object rules extends Facets(pressing.rules, loose)

object DerivedControl extends Derived(Faceted.restrict(hand.press))
final case class Pair(left: State, right: State)
object ComposedControl extends Composition[Pair](_.left -> Faceted, _.right -> Disabled):
  def end(s: State) = true
  object syncs extends Syncs
