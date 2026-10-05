package umpire

import scala.collection.mutable
import umpire.realize.Realization

/**
 * A checked-in IR file, `model/ir/<name>.json`, and the declarations that are its roots, named by
 * value beside the Models it holds:
 *
 * {{{
 * val ordersControlFile =
 *   irFile("orders-control")(forgedCompletion, OrdersRealization.forgedCompletion)
 * }}}
 *
 * The IR generator (model/irgen) reads every such val and writes each file, in one run, from its
 * roots and everything they reach, as it lifts the roots named on its command line. A root that names
 * nothing does not compile. A declaration may be a root of several files and is lifted into each; one no
 * file names stays out of model/ir, so a design can be kept out of the checked files. The file's
 * `source` lists its roots' fully qualified names.
 */
final class IrFile private[umpire] (val name: String, val roots: Seq[IrRoot]):
  /**
   * Constructs every root and what it reaches, as the gate does for every IR file (a Model test
   * beside the Models): each machine's rules, so an overlap of two of them is refused
   * here, the members of each composition, and the machines each Query, capability declaration,
   * progress claim and realization names.
   */
  def construct(): Unit =
    val seen = mutable.Set.empty[AnyRef]
    def model(m: Model): Unit =
      if seen.add(m) then
        m match
          case machine: Machine[?, ?, ?] => machine.bindings: Unit
          case c: Composition[?]         => c.members.foreach(model)
          case _                         => ()
    roots.foreach:
      case m: Machine[?, ?, ?] => model(m)
      case c: Composition[?]   => model(c)
      case q: Query            => Seq(q.scenario.machine, q.property.machine).foreach(model)
      case qs: Seq[?]          =>
        qs.foreach:
          case q: Query => Seq(q.scenario.machine, q.property.machine).foreach(model)
          case _        => ()
      case p: Progress[?]     => model(p.machine)
      case r: Realization     => model(r.machine)
      case c: Capabilities[?] => model(c.model)

object IrFile:
  private[umpire] val made = mutable.ArrayBuffer.empty[IrFile]

  /** Every IR file declared so far: each `irFile` val of an object that has initialized. */
  def declared: Seq[IrFile] = made.synchronized(made.toSeq)

/**
 * What an IR file names as a root: a machine, a composition, a Query, a list of Queries, a progress
 * claim, a realization, or a capability declaration with the laws it brings.
 */
type IrRoot = Machine[?, ?, ?] | Composition[?] | Query | Seq[Query] | Progress[?] | Realization |
  Capabilities[?]

/** Declares the IR file `model/ir/<name>.json` and its roots. */
def irFile(name: String)(roots: IrRoot*): IrFile =
  val file = IrFile(name, roots)
  IrFile.made.synchronized(IrFile.made += file)
  file
