// R20 (a), the folders of a feature with two levels (fn-126): KettleSystem refines KettleProduct,
// so the kettle keeps its levels in product/Product.scala and system/System.scala. It has no
// system/System.scala, refused at the refinement (system/Heater.scala); a machine object in this
// root feature file, which holds the types, the signature and exports alone; and one in a folder
// below system/ (system/element/Element.scala). The tap has one level and still a product/ folder
// and a valve/ one, each refused at its file's first declaration, and a source whose package does
// not mirror its folder (tap/fittings/Washer.scala). The urn misses its root feature file and its
// product/Product.scala (urn/system/System.scala).
package fixture.features.kettle

import umpire.*
import product.KettleProduct
import system.KettleSystem

given Family = Family("fixture.kettle")

final case class Kettle(hot: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

type KettleStep = Step[Kettle, Outcome, Nothing]

object cook extends Actor:
  val boil = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

// A machine in the root feature file.
object Stray extends Machine[Kettle, Outcome, Nothing]:
  val init = Kettle(hot = false)
  def end(s: State) = true

  object effects extends Section:
    def boiled(s: State): List[KettleStep] = enter(s.copy(hot = true))

  object rules extends Rules:
    when(!_.hot)(cook.boil ~> effects.boiled)

object exports:
  val kettle = irFile("kettle")(KettleProduct, KettleSystem)
