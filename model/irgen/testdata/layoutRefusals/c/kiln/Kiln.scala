// R20 (c), the closed names of a machine's sections and the one object exports (fn-126). The kiln
// has two levels and no exports in this root feature file, refused at its first declaration; its
// Product holds objects of names no section has, each refused once, whatever it holds (timers holds
// timers, helpers vocabulary): its name is what makes an object a section. The pump, under
// shared, may have no exports, but not one outside its root feature file (pump/system/System.scala,
// whose type after its machine the order lint refuses: a level's file reads as a feature file).
package fixture.features.kiln

import framework.*

final case class Kiln(hot: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

type KilnStep = Step[Kiln, Outcome, Nothing]

object potter extends Actor:
  val fire = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
