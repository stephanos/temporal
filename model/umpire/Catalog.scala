/* The catalog of laws: which capability kind, and which unordered pair of kinds, brings which laws.
 *
 * The framework holds the mechanism and names no capability and no law: a feature kit declares its
 * capability kinds (each binding's companion extends `CapabilityKind`), writes its laws as objects
 * that extend `Law`, and provides the one `given Catalog` its Models' declarations read. A machine receives every law of each capability it
 * declares and of each pair of capabilities it declares both of, so a pair's law applies without its
 * author naming the pair.
 *
 * A law joins a catalog only once two instantiating entities adopt it, where an instantiating
 * entity is a machine with its own state type that declares the capability: a composition reading a
 * member's capability through its projection does not count again, and neither does a machine
 * derived from another.
 */
package umpire

/**
 * A kind of capability, by which a catalog keys the laws it brings: the companion object of a
 * capability's binding, named after it.
 */
trait CapabilityKind:
  def name: String = getClass.getSimpleName.stripSuffix("$")

/**
 * A law: an object named after it whose `apply` states it, a def that takes the model and the
 * capability's fields and returns a Property, and as data the lifter reads, the server code it rests
 * on, what it promises and what it leaves to other laws or parameters. `parameters` names the
 * parameters of its `apply` in which entities differ on purpose: each entity's binding of one backs
 * its value with the server code that answers it so, written with `cited`.
 */
abstract class Law(
    val cites: Seq[String],
    val promises: String,
    val doesNotPromise: String,
    val parameters: Seq[String] = Nil
):
  /** The law's name, which generated claims take as `<machine>.<law>`: its object's. */
  def name: String = getClass.getSimpleName.stripSuffix("$")

/** A law and the capability kinds that bring it: one, or an unordered pair. */
final case class Brought(by: Set[CapabilityKind], law: Law)

/** Which laws each capability kind and each unordered pair of kinds brings. */
final case class Catalog(entries: Vector[Brought]):
  infix def ++(other: Catalog): Catalog = Catalog(entries ++ other.entries)

  /** The laws a machine declaring the capabilities `declared` receives, pairs found among them. */
  def laws(declared: Set[CapabilityKind]): Vector[Law] =
    entries.collect { case Brought(by, law) if by.subsetOf(declared) => law }

object Catalog:
  /** The laws the capability kind `c` brings on its own. */
  def single(c: CapabilityKind)(laws: Law*): Catalog =
    Catalog(laws.map(Brought(Set(c), _)).toVector)

  /** The laws the pair of `c` and `d` brings, written in either order. */
  def pair(c: CapabilityKind, d: CapabilityKind)(laws: Law*): Catalog =
    require(c != d, s"a pair is of two capabilities, not ${c.name} twice")
    Catalog(laws.map(Brought(Set(c, d), _)).toVector)
