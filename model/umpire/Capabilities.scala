/* Capabilities: what a machine declares it can do, binding each protocol's parameters to its own
 * vocabulary, and the laws it receives for them.
 *
 * `capabilities(m, limits)(…)` names the machine, the Limits its generated `verify` Queries run under,
 * and its capabilities; the given `Catalog` brings the laws of each capability and of each unordered
 * pair it declares both of. The lifter (model/lifter/Capabilities.scala) expands the declaration
 * into a Property, a Scenario and a Query per law, each named `<machine>.<law>`, and writes what it
 * expanded into the law sidecar beside the IR file. A function-valued field names a def of the lifted
 * sources, never a lambda, as a shared def's function-valued argument does.
 *
 * The framework names no capability: a feature kit declares its kinds and laws (model/umpire/Catalog.scala).
 * Everything here is core: it declares what the IR and the sidecar need.
 */
package umpire

/**
 * A capability of a machine whose steps are `Step[S, O, F]`: a predicate of another state type, an
 * outcome of another type or a fact of another type does not compile. A feature kit declares each
 * capability as a case class whose companion is its `CapabilityKind`. The lifter reads a field of
 * an action class as an action the machine must bind, a list of them as the path to a live state a
 * functional law's find starts from, and a `RunExpectation` as the Run that find expects.
 */
trait CapabilityOf[S, +O, +F]

/** A law a declaration waives, with the reason: lifted not at all, or as the entity's own def. */
enum Waiver:
  case Except(law: Law, because: String)
  case Overriding(law: Law, by: AnyRef, because: String)

/**
 * What a machine declares: its capabilities, the catalog that brings their laws, the Limits the
 * generated `verify` Queries run under, and the laws it waives.
 */
final class Capabilities[S] private[umpire] (
    val model: Declares[S],
    val limits: Limits,
    val declared: Seq[CapabilityOf[S, ?, ?]],
    val catalog: Catalog,
    val waived: Seq[Waiver]
):
  /**
   * The Property this declaration generates for `law`, `<machine>.<law>`, for a Query of its own
   * to read: a law the declaration waives with `except` has none.
   */
  def claim(law: Law): Property[S] =
    Property(PropertyDecl(s"${model.name}.${law.name}", model, None, None, None))

  /** Lifts no Property and no Query for `law`, for the reason `because` gives. */
  def except(law: Law, because: String): Capabilities[S] =
    Capabilities(model, limits, declared, catalog, waived :+ Waiver.Except(law, because))

  /**
   * Lifts `replaced`'s def, which takes the law's parameters, under the law's name, for the reason
   * `because` gives: `overriding(closedIsRejectedUniformly -> ownDef, because = "…")`.
   */
  def overriding(replaced: (Law, AnyRef), because: String): Capabilities[S] =
    Capabilities(
      model,
      limits,
      declared,
      catalog,
      waived :+ Waiver.Overriding(replaced._1, replaced._2, because)
    )

/**
 * Declares the capabilities of `m`, whose generated `verify` Queries run under `limits`, and receives
 * the laws the given catalog brings for them.
 */
def capabilities[S](m: Declares[S], limits: Limits)(
    declared: CapabilityOf[S, m.Outcome, m.Fact]*
)(using catalog: Catalog): Capabilities[S] =
  Capabilities(m, limits, declared, catalog, Nil)
