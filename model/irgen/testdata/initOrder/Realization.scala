// (b): the realization reads the machine's `implements` back while it initializes, which a
// realization must not do: `implements` reads the realization first.
package fixture.features.initorder

object SwitchRealization:
  val reported = Switch.states.lit(Lamp(true))
  val checked = Switch.implements.reported
