// (b): the realization reads the machine's `laws` back while it initializes, which a realization
// must not do: `laws` reads the realization first.
package fixture.features.initorder

object SwitchRealization:
  val reported = Switch.lit(Lamp(true))
  val checked = Switch.laws.reported
