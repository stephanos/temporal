/* The lamp, the template a new feature copies (fn-126 R20; model/README.md, "Starting a new
 * feature"). A feature whose Models include a refinement pair has two levels, each in a folder of
 * its own, because different people read them:
 *
 *   - Lamp.scala, this file, named after the feature's folder: the types, the signature and
 *     `object exports`, and no machine;
 *   - product/Product.scala: LampProduct, what a caller reads, which refines nothing;
 *   - system/System.scala: LampSystem, how the server gets there, which refines LampProduct;
 *   - system/Bulb.scala: Bulb, a zoom-in on how the System keeps its promise. A level folder holds
 *     one file per subject beside the level's own; a zoom-in goes in the folder of its audience,
 *     whatever it refines, and no folder sits below a level's.
 *
 * A feature with one machine, or none that refines another, keeps it in this file and has neither
 * folder. Read top to bottom: the types, the signature (the user and its actions), and last
 * exports, the feature's IR file. To start a feature, copy this folder to
 * model/temporal/features/<feature>/, name the file after the folder and the levels' machines
 * after the feature, and replace the lamp with what the server does.
 */
package fixture.features.lamp

import umpire.*
import product.LampProduct
import system.{Bulb, LampSystem}

given Family = Family("fixture.lamp")

// ### Types
// Every enum, state record and alias the levels share sits here. A subject's own types sit in its
// file, as system/Bulb.scala keeps its Filament.

/** What a caller reads: whether the lamp is lit. */
final case class Lamp(lit: Boolean) derives Finite

/** How the server gets there: whether the circuit through the lamp is closed. */
final case class Circuit(closed: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

type LampStep = Step[Lamp, Outcome, Nothing]
type CircuitStep = Step[Circuit, Outcome, Nothing]

// ### Signature
// Who acts, and the actions each takes, as the objects of their parties. Every level binds them.

/** The user switches the lamp on and off. */
object user extends Actor:
  val switchOn = action(this)
  val switchOff = action(this)

/** Every step answers `accepted`, which `enter` reads. */
given Ok[Outcome] = Ok(Outcome.accepted)

/** One step, one action: the bounds the Queries of every level run within. */
val one = Limits(steps = 1, actions = 1, search = 8)

// ### Exports
// The feature's IR files, one `object exports` per feature, in this file alone; each val is named
// after its file and names its roots, in every level.

object exports:
  val lamp = irFile("lamp")(LampProduct, LampSystem, Bulb)
