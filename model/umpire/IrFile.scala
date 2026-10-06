package umpire

import scala.collection.mutable
import umpire.realize.Realization

/**
 * A checked-in IR file, `model/ir/<name>.json`, and the declarations that are its roots, named by
 * value beside the Models it holds:
 *
 * {{{
 * val ordersControl =
 *   irFile("orders-control")(ForgedCompletion, ForgedCompletion.queries, OrdersRealization.forged)
 * }}}
 *
 * The IR generator (model/irgen) reads every such val and writes each file, in one run, from its
 * roots and everything they reach, as it lifts the roots named on its command line. A root is a
 * machine or composition object, a Query, a list of Queries, a progress claim, a realization, an
 * `implements` section, which declares its object's capabilities, or a `queries` section, which
 * roots every Query it declares; the generator refuses anything else. A declaration may be a root of
 * several files and is lifted into each; one no file names stays out of model/ir, so a design can be
 * kept out of the checked files. The file's `source` lists its roots' fully qualified names.
 */
final class IrFile private[umpire] (val name: String, val roots: Seq[IrRoot]):
  /**
   * Constructs every root and what it reaches, as the gate does for every IR file (a Model test
   * beside the Models): each machine's rules, so an overlap of two of them is refused here, the
   * members of each composition, the machine each machine refines, the source of each derivation,
   * and the machines each Query, capability declaration, progress claim and realization names. It
   * gives the names of the machines and compositions constructed, which the gate's test holds to
   * every machine the file's IR lifts, so no lifted machine escapes the check of its rules.
   */
  def construct(): Set[String] =
    val seen = mutable.LinkedHashSet.empty[Model]
    def model(m: Model): Unit =
      if seen.add(m) then
        m match
          case machine: Machine[?, ?, ?] =>
            machine.bindings: Unit
            machine.reaches.foreach(model)
          case c: Composition[?] => c.members.foreach(model)
          case _                 => ()
    def query(q: Query): Unit = Seq(q.scenario.machine, q.property.machine).foreach(model)
    def root(r: Any): Unit = r match
      case m: Machine[?, ?, ?]    => model(m)
      case c: Composition[?]      => model(c)
      case q: Query               => query(q)
      case qs: Seq[?]             => qs.foreach { case q: Query => query(q); case _ => () }
      case p: Progress[?]         => model(p.machine)
      case r: Realization         => model(r.machine)
      case c: Capabilities[?]     => model(c.model)
      case i: Implements[?, ?, ?] => model(i.capabilities.model)
      case section: AnyRef        => IrFile.queriesOf(section).foreach(root)
    roots.foreach(root)
    seen.map(_.name).toSet

object IrFile:
  private[umpire] val made = mutable.ArrayBuffer.empty[IrFile]

  /** Every IR file declared so far: each `irFile` val of an object that has initialized. */
  def declared: Seq[IrFile] = made.synchronized(made.toSeq)

  /**
   * The Queries and lists of Queries a `queries` section declares, read as the members it declares
   * with no parameter: the section is no class of the framework, so its members are found as the
   * object's own, as `Refinement.declaredBy` finds a machine's refinement.
   */
  private def queriesOf(section: AnyRef): Seq[Any] =
    section.getClass.getDeclaredMethods.toSeq
      .filter(m => m.getParameterCount == 0 && java.lang.reflect.Modifier.isPublic(m.getModifiers))
      .filter(m =>
        classOf[Query].isAssignableFrom(m.getReturnType) ||
          classOf[scala.collection.Seq[?]].isAssignableFrom(m.getReturnType)
      )
      .sortBy(_.getName)
      .map(_.invoke(section))

/**
 * What an IR file names as a root: a machine, a composition, a Query, a list of Queries, a progress
 * claim, a realization, a capability declaration with the laws it brings, an `implements` section,
 * or a `queries` section. A section is no class of the framework, so the type admits any object, and
 * the IR generator refuses one that is none of these.
 */
type IrRoot = AnyRef

/** Declares the IR file `model/ir/<name>.json` and its roots. */
def irFile(name: String)(roots: IrRoot*): IrFile =
  val file = IrFile(name, roots)
  IrFile.made.synchronized(IrFile.made += file)
  file
