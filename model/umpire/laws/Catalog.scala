/* The catalog of laws: which capability, and which unordered pair of capabilities, brings which
 * laws.
 *
 * The catalog is a value. This file holds its type and the entity-neutral entries; the one
 * `given Catalog` of model/temporal/laws adds the Temporal laws, so the framework names no law of
 * Temporal's. A machine receives every law of each capability it declares and of each pair of
 * capabilities it declares both of, so a pair's law applies without its author naming the pair.
 *
 * A law joins the catalog only once two instantiating entities adopt it, where an instantiating
 * entity is a machine with its own state type that declares the capability: a composition reading a
 * member's capability through its projection does not count again, and neither does a machine
 * derived from another. The catalog test of model/temporal/laws counts them that way.
 */
package umpire.laws

/** A capability a machine declares, by which the catalog keys the laws it brings. */
enum Capability:
  case Closable, Terminable, Pausable, Cancelable, Pollable, Describable

/**
 * A law as data the lifter reads, beside the def that states it: its name, that def by reference,
 * the server code it rests on, what it promises and what it leaves to other laws or parameters.
 */
final case class Law(
    name: String,
    statement: AnyRef,
    cites: Seq[String],
    promises: String,
    doesNotPromise: String
)

/** A law and the capabilities that bring it: one, or an unordered pair. */
final case class Brought(by: Set[Capability], law: Law)

/** Which laws each capability and each unordered pair of capabilities brings. */
final case class Catalog(entries: Vector[Brought]):
  infix def ++(other: Catalog): Catalog = Catalog(entries ++ other.entries)

  /** The laws a machine declaring the capabilities `declared` receives, pairs found among them. */
  def laws(declared: Set[Capability]): Vector[Law] =
    entries.collect { case Brought(by, law) if by.subsetOf(declared) => law }

object Catalog:
  /** The laws the capability `c` brings on its own. */
  def single(c: Capability)(laws: Law*): Catalog = Catalog(laws.map(Brought(Set(c), _)).toVector)

  /** The laws the pair of `c` and `d` brings, written in either order. */
  def pair(c: Capability, d: Capability)(laws: Law*): Catalog =
    require(c != d, s"a pair is of two capabilities, not $c twice")
    Catalog(laws.map(Brought(Set(c, d), _)).toVector)

/** The entity-neutral entries: what any entity with a terminal status set is held to. */
val entityNeutral: Catalog =
  Catalog.single(Capability.Closable)(terminalStatesAreFinalLaw, closedIsRejectedUniformlyLaw)
