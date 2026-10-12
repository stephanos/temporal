// The Scala declarations of the Umpire model layer. A Model is ordinary Scala 3: domains are enums
// and case classes, actions and machines are vals, and step functions are plain defs with total
// matches. The IR generator (model/irgen) reads the declarations into the IR, and the Go reader
// (tools/umpire/interp and tools/umpire/check) builds the tables, checks the refinements and answers the Queries from it;
// nothing here computes them.
package framework

import scala.deriving.Mirror

// A type whose values can all be listed, in catalog order. Derived structurally through `Mirror`:
// an enum is the concatenation of its cases, a case class (or a parametrised enum case) the
// product of its fields in declaration order with the last field varying fastest, which is the
// catalog order the IR's semantics define (SEMANTICS.md) and the Go reader lists.
@annotation.implicitNotFound(
  "${T} has no finite catalog. A domain is an enum, a case class whose fields are domains, a " +
    "Boolean, an Option of a domain or a counter UpTo[N]; a plain Int or a List is none. The " +
    "explanation below ends at the member that has none."
)
trait Finite[T]:
  def values: IndexedSeq[T]

object Finite:
  given [T](using Finite[T]): CanEqual[T, T] = CanEqual.derived

  def apply[T](using f: Finite[T]): Finite[T] = f

  def of[T](vs: T*): Finite[T] =
    val all = vs.toIndexedSeq
    new Finite[T]:
      def values = all

  // `0..hi`, a scalar catalog also used by named opaque integer types. Model fields and messages
  // declare their bounds with UpTo[N].
  def upTo(hi: Int): Finite[Int] = of((0 to hi)*)

  // False before true, the order the Go reader lists a Boolean's values in.
  given Finite[Boolean] = of(false, true)

  // A domain with no members: a machine that records no facts has `Nothing` as its fact type.
  given Finite[Nothing] = of()

  // An optional value: absent, then present with each member of `A` in its catalog order.
  given [A](using a: Finite[A]): Finite[Option[A]] = of((None +: a.values.map(Some(_)))*)

  // Every enum and case class is a domain without a `derives Finite` clause: its catalog is derived
  // here, from its `Mirror`, where a `Finite` is first needed. A more specific given wins over this
  // one, so `Option`, `UpTo` and a Model's own `given Finite[S]` (a channel's `Inbox`) keep theirs.
  //
  // The whole structure is checked by the implicit search itself, through `Members`: a case class
  // with a field that has no catalog, a plain `Int` or a list, has no `Finite`, and neither has a
  // union of two domains. That is what keeps `==` strict: the `CanEqual` above exists exactly where
  // a catalog does, and the search, not a later inline expansion, decides it.
  given [T](using m: Mirror.Of[T])(using members: Members[m.MirroredElemTypes]): Finite[T] =
    derived[T]

  // The derived catalog of `T`, which a Model's own `given Finite[S]` block names when it binds a
  // field's catalog first (a channel's `Inbox`): `given Finite[S] = { given Finite[Inbox[M]] =
  // wire.contents; Finite.derived }`.
  def derived[T](using m: Mirror.Of[T])(using members: Members[m.MirroredElemTypes]): Finite[T] =
    m match
      case _: Mirror.Sum =>
        val cases = members.catalogs.flatMap(_.values)
        of(cases.asInstanceOf[List[T]]*) // scalafix:ok DisableSyntax.asInstanceOf
      case p: Mirror.Product =>
        val rows = product(members.catalogs).map(vs => p.fromProduct(Tuple.fromArray(vs.toArray)))
        of(rows.asInstanceOf[List[T]]*) // scalafix:ok DisableSyntax.asInstanceOf

  // The catalogs of a Mirror's element types, a sum's cases or a product's fields, in declaration
  // order. A singleton or parametrised enum case has no given of its own; the fallback above derives
  // it in place, as it derives any nested record.
  trait Members[Ts <: Tuple]:
    def catalogs: List[Finite[?]]

  object Members:
    given Members[EmptyTuple] with
      def catalogs: List[Finite[?]] = Nil
    given [H, T <: Tuple](using h: Finite[H], t: Members[T]): Members[H *: T] with
      def catalogs: List[Finite[?]] = h :: t.catalogs

  private def product(fields: List[Finite[?]]): List[List[Any]] =
    fields.foldLeft(List(List.empty[Any])) { (prefixes, f) =>
      for prefix <- prefixes; v <- f.values.toList yield prefix :+ v
    }

// A counter of `0..N`, bounded where its field is declared: `attempts: UpTo[2]` has the values 0, 1
// and 2, in that order. It reads as an Int; a step writes one with `UpTo(n)`, which the lifter lifts as
// `n`, and Go refuses a value outside the range as it does for any bounded field.
opaque type UpTo[N <: Int] <: Int = Int

object UpTo:
  // The counter at `n`, which lies in `0..N`.
  def apply[N <: Int](n: Int)(using bound: ValueOf[N]): UpTo[N] =
    require(0 <= n && n <= bound.value, s"$n is outside 0..${bound.value}")
    n

  given [N <: Int](using bound: ValueOf[N]): Finite[UpTo[N]] = Finite.of((0 to bound.value)*)
