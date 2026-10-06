// Objects that group a Model's actions, named after where they are declared (fn-126 decision 23):
// a Definition ID is the declaration's fully qualified Scala name, the package and every object it
// sits in, so a top-level action is `fixture.sections.reset`, one in the top-level object `panel`
// is `fixture.sections.panel.flip`, and one in an actor object of a machine's object is
// `fixture.sections.Switch.operator.press`. Two objects may each hold an action of one name, which
// their objects tell apart, as `Switch` and `Clapper` bind them. The lifter's tests lift both,
// compare the IR with expected/sections.json and check those IDs.
package fixture.sections

import umpire.*

final case class Light(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

/** A top-level action beside the objects, named after its package alone. */
val reset = internal

/** An object at the top level: its member's ID is the object's, `fixture.sections.panel.flip`. */
object panel:
  val flip = action(Actor("panel"))

// Two objects, each with an action of one name.
object leftHand:
  val clap = action(Actor("fixture"))

object rightHand:
  val clap = action(Actor("fixture"))

object Switch extends Machine[Light, Outcome, Nothing]:
  val init = Light(false)
  def end(light: State) = true

  /** An actor directly in the machine's object: its action's ID names the machine's object too. */
  object operator extends Actor:
    val press = action(this)

  def flipStep(l: Light): List[Step[Light, Outcome, Nothing]] =
    List(Step(Outcome.accepted, Light(!l.on)))
  def pressStep(l: Light): List[Step[Light, Outcome, Nothing]] =
    List(Step(Outcome.accepted, Light(true)))
  def resetStep(l: Light): List[Step[Light, Outcome, Nothing]] =
    List(Step(Outcome.accepted, Light(false)))

  object rules
      extends Bindings(
        panel.flip ~> flipStep,
        operator.press ~> pressStep,
        reset ~> resetStep,
        leftHand.clap ~> flipStep
      )

/** The other hand's clap, an action of the name of the left hand's, which its object tells apart. */
object Clapper extends Machine[Light, Outcome, Nothing]:
  val init = Light(false)
  def end(light: State) = true
  object rules extends Bindings(rightHand.clap ~> Switch.flipStep)
