// A Model folder whose file is not named after it (fn-126 R1, R10): misnamed/ has no feature file,
// misnamed/Misnamed.scala, so its machine object, its top-level Property and its type's companion
// would be held to no reading order. (d) refuses each at its line; the types and the signature are
// no Model declarations and stay.
package fixture.features.misnamed

import framework.*

final case class Lamp(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

val flip = action(Actor("fixture"))

object Switch extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(lamp: State) = true

  object effects:
    def flipped(s: Lamp): List[Step[Lamp, Outcome, Nothing]] =
      List(Step(Outcome.accepted, s.copy(lit = !s.lit)))

  object rules extends Rules:
    on(flip)(always ~> effects.flipped)

def loose: Property[Lamp] = Switch.property holds (after => after.state.lit)

final case class Spare(lit: Boolean)

// A type's companion holds no Model in such a file either.
object Spare:
  object SpareLamp extends Derived(Switch.unmonitored)
