// Sections in a file that pins nothing (fn-126 R14): a section is transparent to Definition IDs, so
// each member takes the ID it would take as a direct member of the section's owner. At the file's
// top level that owner is the file's package object, which the lifter finds among the file's own
// definitions rather than spelling it from the file's name; directly in a machine's object, the
// owner is that object. The lifter's tests lift `Switch.switch`, compare the IR with
// expected/sections.json and check those IDs.
package fixture.sections

import umpire.*

given Family = Family("fixture.sections")

final case class Light(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

type LightStep = Step[Light, Outcome, Nothing]

/** A top-level action beside the sections, which takes the file's package object's ID. */
val reset = internal

/** A section at the top level: its member's ID is the package object's, as `reset`'s is. */
object panel extends Section:
  val flip = action(Party("panel"))

object Switch:
  /** An actor directly in the machine's object: its action's ID is `fixture.sections.Switch$`'s. */
  object operator extends Actor:
    val press = action(this)

  def flipStep(l: Light): List[LightStep] = List(Step(Outcome.accepted, Light(!l.on)))
  def pressStep(l: Light): List[LightStep] = List(Step(Outcome.accepted, Light(true)))
  def resetStep(l: Light): List[LightStep] = List(Step(Outcome.accepted, Light(false)))

  val switch = machine[Light, Outcome, Nothing] {
    starts(Light(false))
    ends(_ => true)
    steps(panel.flip ~> flipStep, operator.press ~> pressStep, reset ~> resetStep)
  }
