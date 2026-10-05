/* Temporal's capabilities: what an entity can do, as the laws of this folder read it. Each is a
 * binding of a protocol's parameters to the entity's own vocabulary, and its companion is the kind
 * the catalog (Catalog.scala) keys the laws it brings by. An entity declares the ones it has with
 * `capabilities(m, limits)(…)` in the `laws` section of its machine's object.
 */
package temporal.capabilities

import umpire.*
import umpire.realize.{RunExpectation, StatusTable}

/**
 * The entity has a terminal status set: `status` reads a state's status, `terminal` says which
 * statuses close it, and a closed entity answers `rejected`.
 */
final case class Closable[S, P, O](status: S => P, terminal: P => Boolean, rejected: O)
    extends CapabilityOf[S, O, Nothing]

object Closable extends CapabilityKind

/**
 * The entity can be terminated by `terminate`, which records `settled`; `reach` is the path to a live
 * state the functional laws start from, and `expect` the Run their find Queries expect of a server.
 */
final case class Terminable[S, F](
    terminate: ClassRef,
    settled: F,
    reach: Seq[ClassRef],
    expect: RunExpectation
) extends CapabilityOf[S, Nothing, F]

object Terminable extends CapabilityKind

/** The entity can be paused by `pause` and unpaused by `unpause`; `paused` says where it is. */
final case class Pausable[S](
    pause: ClassRef | Composed,
    unpause: ClassRef | Composed,
    paused: S => Boolean
) extends CapabilityOf[S, Nothing, Nothing]

object Pausable extends CapabilityKind

/**
 * A cancel of the entity is requested by `requestCancel`, which records `requested`; `reach` is the
 * path to a live state the functional laws start from, and `expect` the Run their find Queries
 * expect of a server.
 */
final case class Cancelable[S, F](
    requestCancel: ClassRef,
    requested: F,
    reach: Seq[ClassRef],
    expect: RunExpectation
) extends CapabilityOf[S, Nothing, F]

object Cancelable extends CapabilityKind

/** The entity's work is handed out by polling, `dispatch`; `running` says where a worker holds it. */
final case class Pollable[S](dispatch: ClassRef | Composed, running: S => Boolean)
    extends CapabilityOf[S, Nothing, Nothing]

object Pollable extends CapabilityKind

/** The entity is described by `status`, the realization's table from each fact to the status read. */
final case class Describable[S, V](status: StatusTable[V]) extends CapabilityOf[S, Nothing, Nothing]

object Describable extends CapabilityKind
