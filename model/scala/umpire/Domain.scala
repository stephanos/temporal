/* The Scala implementation of the Umpire model layer, built to be compared with the Lean one under
 * model/lean/ and the Go one under model/go/. A Model is ordinary Scala 3: domains are enums and
 * case classes, actions and machines are vals, step functions are plain defs with total matches,
 * and the finite table, the refinement and the Queries are computed and checked when a test asks.
 *
 * The orders and key spellings follow the Lean implementation exactly, because Definition IDs,
 * witnesses and exploration targets are compared against Lean's output byte for byte. Where a rule
 * is non-obvious, the comment names the Lean source it mirrors; the Go port is the executable
 * reference each rule was checked against first.
 */
package umpire

import scala.compiletime.{constValue, erasedValue, summonFrom}
import scala.deriving.Mirror

/** A type whose values can all be listed, in catalog order. Derived structurally through `Mirror`:
  * an enum is the concatenation of its cases, a case class (or a parametrised enum case) the
  * product of its fields in declaration order with the last field varying fastest, which is the
  * order the Lean `Finite` derivation produces. */
trait Finite[T]:
  def values: IndexedSeq[T]

object Finite:
  def apply[T](using f: Finite[T]): Finite[T] = f

  def of[T](vs: T*): Finite[T] =
    val all = vs.toIndexedSeq
    new Finite[T]:
      def values = all

  /** `0..hi`, the reading of Lean's `Fin (hi + 1)`. A model that bounds a counter gives this as a
    * local given next to the state it derives, so the bound sits beside the field it bounds. */
  def upTo(hi: Int): Finite[Int] = of((0 to hi)*)

  // Lean's `Bool` instance lists false before true.
  given Finite[Boolean] = of(false, true)

  /** A domain with no members: a machine that records no facts has `Nothing` as its fact type. */
  given Finite[Nothing] = of()

  inline def derived[T](using m: Mirror.Of[T]): Finite[T] =
    inline m match
      case s: Mirror.SumOf[T] =>
        val cases = summonCases[s.MirroredElemTypes]
        of(cases.flatMap(_.values).asInstanceOf[List[T]]*)
      case p: Mirror.ProductOf[T] =>
        val fields = summonFields[p.MirroredElemTypes, p.MirroredElemLabels]
        of(product(fields).map(vs => p.fromProduct(Tuple.fromArray(vs.toArray))).toList*)

  private inline def summonCases[Ts <: Tuple]: List[Finite[?]] =
    inline erasedValue[Ts] match
      case _: EmptyTuple => Nil
      case _: (h *: t) =>
        summonFrom {
          case f: Finite[`h`] => f
          // A singleton case or a parametrised case has no given of its own; derive it in place.
          case m: Mirror.Of[`h`] => derived[h](using m)
        } :: summonCases[t]

  private inline def summonFields[Ts <: Tuple, Ls <: Tuple]: List[Finite[?]] =
    inline erasedValue[Ts] match
      case _: EmptyTuple => Nil
      case _: (h *: t) =>
        inline erasedValue[Ls] match
          case _: (l *: ls) =>
            summonFrom {
              case f: Finite[`h`] => f
              case m: Mirror.Of[`h`] => derived[h](using m)
              case _ => scala.compiletime.error("field " + constValue[l & String] + " has no Finite instance")
            } :: summonFields[t, ls]

  private def product(fields: List[Finite[?]]): List[List[Any]] =
    fields.foldLeft(List(List.empty[Any])) { (prefixes, f) =>
      for prefix <- prefixes; v <- f.values.toList yield prefix :+ v
    }

/** Overrides the key a value is spelled with in state, action, row and Definition keys. */
trait Keyed:
  def key: String

/** Keys as Lean spells them: an enum case by its name, a parametrised case by its name followed by
  * its fields, a Boolean as true or false, a counter in decimal, and a state case class by its fields
  * in declaration order, all joined by "-". */
object Keys:
  def of(v: Any): String = v match
    case k: Keyed               => k.key
    case b: Boolean             => b.toString
    case i: Int                 => i.toString
    case s: String              => s
    case e: scala.reflect.Enum  =>
      if e.productArity == 0 then e.toString
      else (e.productPrefix :: e.productIterator.map(of).toList).mkString("-")
    case p: Product             => p.productIterator.map(of).mkString("-")
    case other                  => other.toString

  /** The field names a state has in Definition IDs, in declaration order. */
  def fieldNames(v: Any): List[String] = v match
    case p: Product if !v.isInstanceOf[scala.reflect.Enum] => p.productElementNames.toList
    case _                                                 => Nil

  /** A state's fields as keys, beside their names. */
  def fields(v: Any): List[(String, String)] = v match
    case p: Product if !v.isInstanceOf[scala.reflect.Enum] =>
      p.productElementNames.zip(p.productIterator.map(of)).toList
    case _ => Nil

  /** A class spelled the way a Lean `examples:` line spells it: a parametrised case by its
    * constructor and named fields, `handlerError (retryable := true)`, anything else by its key. */
  def spelling(v: Any): String = v match
    case e: scala.reflect.Enum if e.productArity > 0 =>
      val named = e.productElementNames.zip(e.productIterator.map(of)).map((n, k) => s"$n := $k")
      s"${e.productPrefix} (${named.mkString(", ")})"
    case other => of(other)

  /** The action a class key belongs to: the key before its first "-". */
  def actionName(key: String): String = key.takeWhile(_ != '-')
