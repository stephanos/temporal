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
 * Everything here is core: it declares what the IR and the sidecar need.
 */
package umpire

import umpire.laws.{Capability, Catalog, Law}
import umpire.realize.StatusTable

/**
 * A capability of a machine whose steps are `Step[S, O, F]`: a predicate of another state type, an
 * outcome of another type or a fact of another type does not compile.
 */
sealed trait CapabilityOf[S, +O, +F]:
  def kind: Capability

/**
 * The entity has a terminal status set: `status` reads a state's status, `terminal` says which
 * statuses close it, and a closed entity answers `rejected`.
 */
final case class Closable[S, P, O](status: S => P, terminal: P => Boolean, rejected: O)
    extends CapabilityOf[S, O, Nothing]:
  def kind: Capability = Capability.Closable

/**
 * The entity can be terminated by `terminate`, which records `settled`; `reach` is the path to a live
 * state the functional laws start from.
 */
final case class Terminable[S, F](terminate: ClassRef, settled: F, reach: Seq[ClassRef])
    extends CapabilityOf[S, Nothing, F]:
  def kind: Capability = Capability.Terminable

/** The entity can be paused by `pause` and unpaused by `unpause`; `paused` says where it is. */
final case class Pausable[S](
    pause: ClassRef | Composed,
    unpause: ClassRef | Composed,
    paused: S => Boolean
) extends CapabilityOf[S, Nothing, Nothing]:
  def kind: Capability = Capability.Pausable

/**
 * A cancel of the entity is requested by `requestCancel`, which records `requested`; `reach` is the
 * path to a live state the functional laws start from.
 */
final case class Cancelable[S, F](requestCancel: ClassRef, requested: F, reach: Seq[ClassRef])
    extends CapabilityOf[S, Nothing, F]:
  def kind: Capability = Capability.Cancelable

/** The entity's work is handed out by polling, `dispatch`; `running` says where a worker holds it. */
final case class Pollable[S](dispatch: ClassRef | Composed, running: S => Boolean)
    extends CapabilityOf[S, Nothing, Nothing]:
  def kind: Capability = Capability.Pollable

/** The entity is described by `status`, the realization's table from each fact to the status read. */
final case class Describable[S, V](status: StatusTable[V])
    extends CapabilityOf[S, Nothing, Nothing]:
  def kind: Capability = Capability.Describable

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
