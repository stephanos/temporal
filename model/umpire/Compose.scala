package umpire

import scala.annotation.publicInBinary
import scala.compiletime.constValueTuple
import scala.deriving.Mirror

/**
 * One Model from machines of different entities, for a claim no one of them can state.
 * The composed state `S` is a case class with one field per member, named after
 * the member. A `sync` pairs member actions into one step; a member action no `sync` names steps
 * its member alone.
 */
final class Composition[S <: Product] @publicInBinary private[umpire] (
    val family: Family,
    val name: String,
    private[umpire] val fieldNames: Vector[String],
    private[umpire] val members: Vector[(String, Model)],
    private[umpire] val syncs: Vector[(String, (String, String), (String, String))],
    private[umpire] val isEnd: S => Boolean,
    private[umpire] val replaced: Vector[(String, Model)]
) extends Model:
  /** Pairs two members' actions into one step named `name`. */
  def sync(name: String, first: (String, Action[?]), second: (String, Action[?])): Composition[S] =
    Composition(
      family,
      this.name,
      fieldNames,
      members,
      syncs :+ (name, first._1 -> first._2.name, second._1 -> second._2.name),
      isEnd,
      replaced
    )

  /** Which composed states the composition may end in. */
  def ends(end: S => Boolean): Composition[S] =
    Composition(family, name, fieldNames, members, syncs, end, replaced)

  /**
   * Says the member `field` stands in for `opaque` within this composition: a detailed provider in
   * place of an opaque one. The member must declare a refinement of `opaque` that holds, which is
   * what makes the replacement scoped to this composition.
   */
  def replaces(field: String, opaque: Model): Composition[S] =
    Composition(family, name, fieldNames, members, syncs, isEnd, replaced :+ (field -> opaque))

/** Starts a composition over the members, each named after the composed state's field it fills. */
inline def compose[S <: Product](family: Family, name: String)(members: (String, Model)*)(using
    m: Mirror.ProductOf[S]
): Composition[S] =
  val labels = constValueTuple[m.MirroredElemLabels].toList.map(_.toString).toVector
  Composition[S](
    family,
    name,
    labels,
    members.toVector,
    Vector.empty,
    _ => false,
    Vector.empty
  )

/** Starts a composition named after the `val` that declares it, in the `given Family`. */
inline def compose[S <: Product](members: (String, Model)*)(using
    m: Mirror.ProductOf[S],
    family: Family
): Composition[S] =
  val labels = constValueTuple[m.MirroredElemLabels].toList.map(_.toString).toVector
  Composition[S](family, "", labels, members.toVector, Vector.empty, _ => false, Vector.empty)
