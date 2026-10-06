// A capabilities section whose capabilities cross the machine's types, which the section's
// `Capability` refuses before anything is lifted (fn-134.2): a predicate of another state type, an
// outcome of another type and a fact of another type, each a type error at its line.
package fixture.crossed

import umpire.*

final case class Watching[S](on: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]
object Watching extends CapabilityKind

final case class Refusing[S, O](refused: O) extends CapabilityOf[S, O, Nothing]
object Refusing extends CapabilityKind

final case class Noting[S, F](noted: F) extends CapabilityOf[S, Nothing, F]
object Noting extends CapabilityKind

object SectionMachine extends Machine[Here, Outcome, Nothing]:
  val init = Here(false)
  def end(here: State) = true
  object rules extends Bindings()
  object capabilities extends Capabilities:
    val watching: Capability = Watching(on = elsewhereOn)
    val refusing: Capability = Refusing(refused = Signal.up)
    val noting: Capability = Noting(noted = Note.ping)
