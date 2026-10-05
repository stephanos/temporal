// A file moved out of Rejects.scala into the subpackage `moved`, its former owner pinned: its
// top-level types keep the names they had in fixture.rejects, so its Gauge is named
// fixture.rejects.Gauge, which Rejects.scala's Gauge still has. The lifter refuses a lift that reads
// both (Rejects.scala's movedNameTaken).
package fixture.rejects
package moved

import umpire.*

given DefinitionScope = DefinitionScope("fixture.rejects.Rejects$package$")

/** A gauge moved out of fixture.rejects, whose name there another Gauge took. */
enum Gauge derives Finite:
  case empty, full
