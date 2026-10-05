// What a machine is for, as its markers say (fn-126 decision 20), refused at the line of the object
// that declares it: a negative control nothing can refute, one a machine refines and one that
// declares a refinement of its own; a failure model that binds no fault, and one whose every Query
// expects its Run violated; a machine that binds a fault and is marked neither; one marked both; a
// composition whose member binds a fault, unmarked, and one marked a failure model that binds
// none; and a derived negative control nothing refutes. The lifter's tests lift them with the
// other rejected declarations and compare the diagnostics with expected/rejects.txt.
package fixture.markers

import umpire.*
import umpire.realize.*

given Family = Family("fixture.markers")

final case class Lamp(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

val markedLimits = Limits(steps = 1, actions = 1, search = 16)

object hand extends Actor:
  val flip = action(this)

val fault = Party()

object faults extends Section:
  val blowout = action(fault)

/** The steps the lamps share. */
object Steps:
  def toggle(s: Lamp) = enter[Lamp, Outcome, Nothing](Lamp(!s.lit))
  def dark(s: Lamp) = enter[Lamp, Outcome, Nothing](Lamp(false))

/** A lamp under a blowout, and no marker. */
object FaultUnmarked extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(hand.flip ~> Steps.toggle, faults.blowout ~> Steps.dark)

/** A failure model with no fault. */
object FaultlessFailure extends Machine[Lamp, Outcome, Nothing], FailureModel:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(hand.flip ~> Steps.toggle)

/** A failure model every Query of which expects its Run to violate the promise. */
object HopelessFailure extends Machine[Lamp, Outcome, Nothing], FailureModel:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(hand.flip ~> Steps.toggle, faults.blowout ~> Steps.dark)
  object properties extends Section:
    val lit = property when hand.flip holds (_.state.lit)
  object queries extends Section:
    val brokenHopelessFailure =
      (query find properties.lit in scenario("flipped").actions(
        hand.flip
      ) limits markedLimits total 2)
        .expect(
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.violated,
            contract = PropertyOutcome.violated,
            disposition = Disposition.stoppedByMonitor,
            cleanup = Cleanup.succeeded,
            reason = Some(Reason.everyExplanationViolates)
          )
        )

/** A negative control no Query refutes: its one Query is a find whose Run holds. */
object UnrefutedControl extends Machine[Lamp, Outcome, Nothing], NegativeControl:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(hand.flip ~> Steps.toggle)
  object properties extends Section:
    val lit = property when hand.flip holds (_.state.lit)
  object queries extends Section:
    val foundUnrefutedControl =
      query find properties.lit in scenario("flipped").actions(
        hand.flip
      ) limits markedLimits total 2

/** A negative control a machine refines. */
object RefinedControl extends Machine[Lamp, Outcome, Nothing], NegativeControl:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(hand.flip ~> Steps.toggle)
  object properties extends Section:
    val lit = property when hand.flip holds (_.state.lit)
  object queries extends Section:
    val askedRefinedControl =
      query verify properties.lit in scenario("flipped").actions(
        hand.flip
      ) limits markedLimits total 2

object ControlRefiner extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object refinement extends Refinement(RefinedControl):
    def toProduct(s: Lamp) = s
  object rules extends Bindings(hand.flip ~> Steps.toggle)

/** A negative control that declares a refinement of its own, as a System does. */
object SystemicControl extends Machine[Lamp, Outcome, Nothing], NegativeControl:
  val init = Lamp(false)
  def end(s: State) = true
  object refinement extends Refinement(PlainLamp):
    def toProduct(s: Lamp) = s
  object rules extends Bindings(hand.flip ~> Steps.toggle)
  object properties extends Section:
    val lit = property when hand.flip holds (_.state.lit)
  object queries extends Section:
    val askedSystemicControl =
      query verify properties.lit in scenario("flipped").actions(
        hand.flip
      ) limits markedLimits total 2

object PlainLamp extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(hand.flip ~> Steps.toggle)

/** A machine marked both. */
object TornMarkers extends Machine[Lamp, Outcome, Nothing], FailureModel, NegativeControl:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(hand.flip ~> Steps.toggle, faults.blowout ~> Steps.dark)
  object properties extends Section:
    val lit = property when hand.flip holds (_.state.lit)
  object queries extends Section:
    val askedTornMarkers =
      query verify properties.lit in scenario("flipped").actions(
        hand.flip
      ) limits markedLimits total 2

final case class Lamps(left: Lamp, right: Lamp)

/** A failure model a composition puts beside a lamp. */
object Blowing extends Machine[Lamp, Outcome, Nothing], FailureModel:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(hand.flip ~> Steps.toggle, faults.blowout ~> Steps.dark)

/** A composition whose member binds a fault, and no marker. */
object UnmarkedPair extends Composition[Lamps](_.left -> Blowing, _.right -> PlainLamp):
  def end(s: State) = true
  object syncs extends Syncs:
    sync(_.left -> hand.flip, _.right -> hand.flip)

/** A composition marked a failure model whose members bind no fault. */
object FaultlessPair
    extends Composition[Lamps](_.left -> PlainLamp, _.right -> PlainLamp),
      FailureModel:
  def end(s: State) = true
  object syncs extends Syncs:
    sync(_.left -> hand.flip, _.right -> hand.flip)

/** A derived negative control that nothing refutes: no verify, and no refinement it keeps. */
object DerivedControl extends Derived(PlainLamp.unmonitored), NegativeControl:
  object properties extends Section:
    val lit = property when hand.flip holds (_.state.lit)
  object queries extends Section:
    val foundDerivedControl =
      query find properties.lit in scenario("flipped").actions(
        hand.flip
      ) limits markedLimits total 2
