// R20 (c), the closed names of a machine's sections and the one object exports (fn-126). The kiln
// has two levels and no exports in this root feature file, refused at its first declaration; its
// Product holds an object of a name no section has, refused whether it extends Section (timers)
// or not (helpers, which the order lint also reads as vocabulary outside `states`). The pump,
// under shared, may have no exports, but not one outside its root feature file
// (pump/system/System.scala).
package fixture.features.kiln

import umpire.*

given Family = Family("fixture.kiln")

final case class Kiln(hot: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

type KilnStep = Step[Kiln, Outcome, Nothing]

object potter extends Actor:
  val fire = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
