/* The Scala declarations of the Umpire model layer. A Model is ordinary Scala 3: domains are enums
 * and case classes, actions and machines are vals, and step functions are plain defs with total
 * matches. The IR generator (model/irgen) reads the declarations into the IR, and the Go reader
 * (tools/umpire/interp and tools/umpire/check) builds the tables, checks the refinements and answers the Queries from it;
 * nothing here computes them.
 */
package umpire

import scala.compiletime.{constValue, erasedValue, summonFrom}
import scala.deriving.Mirror

/**
 * A type whose values can all be listed, in catalog order. Derived structurally through `Mirror`:
 * an enum is the concatenation of its cases, a case class (or a parametrised enum case) the
 * product of its fields in declaration order with the last field varying fastest, which is the
 * catalog order the IR's semantics define (SEMANTICS.md) and the Go reader lists.
 */
trait Finite[T]:
  def values: IndexedSeq[T]

object Finite:
  def apply[T](using f: Finite[T]): Finite[T] = f

  def of[T](vs: T*): Finite[T] =
    val all = vs.toIndexedSeq
    new Finite[T]:
      def values = all

  /**
   * `0..hi`, a counter of `hi + 1` values. A model that bounds a counter gives this as a
   * local given next to the state it derives, so the bound sits beside the field it bounds.
   */
  def upTo(hi: Int): Finite[Int] = of((0 to hi)*)

  // False before true, the order the Go reader lists a Boolean's values in.
  given Finite[Boolean] = of(false, true)

  /** A domain with no members: a machine that records no facts has `Nothing` as its fact type. */
  given Finite[Nothing] = of()

  /** An optional value: absent, then present with each member of `A` in its catalog order. */
  given [A](using a: Finite[A]): Finite[Option[A]] = of((None +: a.values.map(Some(_)))*)

  inline def derived[T](using m: Mirror.Of[T]): Finite[T] =
    inline m match
      case s: Mirror.SumOf[T] =>
        val cases = summonCases[s.MirroredElemTypes]
        of(cases.flatMap(_.values).asInstanceOf[List[T]]*) // scalafix:ok DisableSyntax.asInstanceOf
      case p: Mirror.ProductOf[T] =>
        val fields = summonFields[p.MirroredElemTypes, p.MirroredElemLabels]
        of(product(fields).map(vs => p.fromProduct(Tuple.fromArray(vs.toArray))).toList*)

  private inline def summonCases[Ts <: Tuple]: List[Finite[?]] =
    inline erasedValue[Ts] match
      case _: EmptyTuple => Nil
      case _: (h *: t)   =>
        summonFrom {
          case f: Finite[`h`] => f
          // A singleton case or a parametrised case has no given of its own; derive it in place.
          case m: Mirror.Of[`h`] => derived[h](using m)
        } :: summonCases[t]

  private inline def summonFields[Ts <: Tuple, Ls <: Tuple]: List[Finite[?]] =
    inline erasedValue[Ts] match
      case _: EmptyTuple => Nil
      case _: (h *: t)   =>
        inline erasedValue[Ls] match
          case _: (l *: ls) =>
            summonFrom {
              case f: Finite[`h`]    => f
              case m: Mirror.Of[`h`] => derived[h](using m)
              case _                 =>
                scala.compiletime.error(
                  "field " + constValue[l & String] + " has no Finite instance"
                )
            } :: summonFields[t, ls]

  private def product(fields: List[Finite[?]]): List[List[Any]] =
    fields.foldLeft(List(List.empty[Any])) { (prefixes, f) =>
      for prefix <- prefixes; v <- f.values.toList yield prefix :+ v
    }

/**
 * A counter of `0..N`, bounded where its field is declared: `attempts: UpTo[2]` has the values 0, 1
 * and 2, in that order. It reads as an Int; a step writes one with `UpTo(n)`, which the lifter lifts as
 * `n`, and Go refuses a value outside the range as it does for any bounded field.
 */
opaque type UpTo[N <: Int] <: Int = Int

object UpTo:
  /** The counter at `n`, which lies in `0..N`. */
  def apply[N <: Int](n: Int)(using bound: ValueOf[N]): UpTo[N] =
    require(0 <= n && n <= bound.value, s"$n is outside 0..${bound.value}")
    n

  given [N <: Int](using bound: ValueOf[N]): Finite[UpTo[N]] = Finite.of((0 to bound.value)*)
