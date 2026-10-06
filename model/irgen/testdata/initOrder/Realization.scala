// (b): the realization reads the machine's `implements` back while it initializes, which a
// realization must not do: `implements` reads the realization first.
package fixture.features.initorder

import umpire.{Capabilities, Declaring, Machine}

object SwitchRealization:
  val reported = Switch.states.lit(Lamp(true))
  val checked = Switch.implements.reported

// (b) through a base class: Dimmer's `capabilities` runs the body of Dimmed, the class it extends,
// which reads Dimmer's realization while the section initializes, and the realization reads the
// section back.
abstract class Dimmed(m: Machine[Lamp, Outcome, Nothing])(using Declaring[Lamp, Outcome, Nothing])
    extends Capabilities:
  val reported = DimmerRealization.reported

object DimmerRealization:
  val reported = Switch.states.lit(Lamp(true))
  val checked = Dimmer.capabilities.reported
