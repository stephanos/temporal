// A capability declaration where no Catalog is given, which brings it no law: refused at its line.
package fixture.crossed.nocatalog
import umpire.*
import fixture.crossed.{flick, hereOn, ticks, HereMachine}
import temporal.capabilities.Pollable

val uncatalogued = capabilities(HereMachine, ticks)(Pollable(dispatch = flick, running = hereOn))
