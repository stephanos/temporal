// A capability declaration where no Catalog is given, which brings it no law: refused at its line.
package fixture.crossed.nocatalog

import umpire.*
import fixture.crossed.{flick, here, hereOn, ticks}

val uncatalogued = capabilities(here, ticks)(Pollable(dispatch = flick, running = hereOn))
