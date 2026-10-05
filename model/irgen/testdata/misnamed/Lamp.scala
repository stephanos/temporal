// A Model folder whose file is not named after it (fn-126 R1, R10): misnamed/ has no feature file,
// misnamed/Misnamed.scala, so its machine object and its top-level Property would be held to no
// reading order. (d) refuses both, each at its line; the types and the signature are no Model
// declarations and stay.
package fixture.features.misnamed

import umpire.*

given Family = Family("fixture.misnamed")

final case class Lamp(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

val flip = action(Party("fixture"))

object Switch:
  object effects:
    def flipped(s: Lamp): List[Step[Lamp, Outcome, Nothing]] =
      List(Step(Outcome.accepted, s.copy(lit = !s.lit)))

  val switch = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(false))
    ends(_ => true)
    steps(flip ~> effects.flipped)
  }

def loose: Property[Lamp] = Switch.switch.property holds (after => after.state.lit)
