// The Scala declarations of the Umpire model layer. A Model is ordinary Scala 3: domains are enums
// and case classes, actions and machines are vals, and step functions are plain defs with total
// matches. The IR generator (model/framework/export) reads the declarations into the IR, and the Go reader
// (tools/umpire/interp and tools/umpire/check) builds the tables, checks the refinements and answers the Queries from it;
// nothing here computes them.
package framework.dsl

import scala.deriving.Mirror

@annotation.implicitNotFound(
  "${T} has no structural data comparison; its members must also support Data."
)
trait Data[T]:
  def compare(left: T, right: T): Int
  final def equivalent(left: T, right: T): Boolean = compare(left, right) == 0
  @volatile private var retained = Map.empty[String, FiniteSchema]
  @volatile private var inboxes = Map.empty[String, Map[InboxOwnership.Field, InboxOwnership.Entry]]
  final def schema: Option[FiniteSchema] =
    val snapshot = retained
    if snapshot.size == 1 then snapshot.values.headOption else None
  @scala.annotation.publicInBinary
  final private[framework] def schemaFor(identity: String): Option[FiniteSchema] =
    retained.get(identity)
  final private[framework] def inboxesFor(
      identity: String
  ): Map[InboxOwnership.Field, InboxOwnership.Entry] =
    inboxes.getOrElse(identity, Map.empty)
  @scala.annotation.publicInBinary
  final private[framework] def retainInbox(
      identity: String,
      owner: String,
      caseName: String,
      field: String,
      message: String,
      declaration: String,
      supplier: () => Channel[?]
  ): this.type = synchronized {
    val coordinate = InboxOwnership.Field(owner, caseName, field)
    val next = new InboxOwnership.Entry(coordinate, message, declaration, supplier)
    val current = inboxes.getOrElse(identity, Map.empty)
    require(
      current.get(coordinate).forall(_.agrees(next)),
      s"finite schema $identity field ${coordinate.context} has conflicting Inbox ownership"
    )
    if !current.contains(coordinate) then
      inboxes = inboxes.updated(identity, current.updated(coordinate, next))
    this
  }
  @scala.annotation.publicInBinary
  final private[framework] def retainSchema(
      identity: String,
      root: Seq[String],
      closure: Seq[String],
      inboxes: Seq[String] = Nil
  ): this.type = synchronized {
    val next = FiniteSchema.decode(identity, root, closure, inboxes)
    require(
      retained.get(identity).forall(_ == next),
      s"finite schema $identity has conflicting definitions"
    )
    retained = retained.updated(identity, next)
    this
  }

object Data:
  given [T](using Data[T]): CanEqual[T, T] = CanEqual.derived
  given [T](using data: Structural[T]): Data[T] = data

  def apply[T](using data: Data[T]): Data[T] = data

  // A distinct evidence type prevents an authored Finite from selecting itself while its
  // structural comparison is being constructed.
  trait Structural[T] extends Data[T]

  object Structural:
    // The comparison of a state var whose type has no schema, which its Machine refuses at compile
    // time; it stands in only so the refusal is the one reported.
    @scala.annotation.publicInBinary
    private[framework] def unschematic[T]: Structural[T] = new Structural[T]:
      def compare(left: T, right: T): Int =
        throw IllegalStateException("a state var without a schema is refused at compile time")

    private def ordered[T](ordering: Ordering[T]): Structural[T] = new Structural[T]:
      def compare(left: T, right: T): Int = ordering.compare(left, right)

    given Structural[Boolean] = ordered(Ordering.Boolean)
    given Structural[Int] = ordered(Ordering.Int)
    given [N <: Int]: Structural[UpTo[N]] with
      def compare(left: UpTo[N], right: UpTo[N]): Int = Integer.compare(left, right)
    given [M](using message: Structural[M]): Structural[Inbox[M]] = Inbox.structural(message)
    given Structural[String] with
      def compare(left: String, right: String): Int =
        val bytes = java.util.Arrays.compareUnsigned(
          left.getBytes(java.nio.charset.StandardCharsets.UTF_8),
          right.getBytes(java.nio.charset.StandardCharsets.UTF_8)
        )
        if bytes != 0 then bytes else left.compareTo(right)
    given Structural[Nothing] with
      def compare(left: Nothing, right: Nothing): Int = left

    given [A](using a: Structural[A]): Structural[Option[A]] with
      def compare(left: Option[A], right: Option[A]): Int = (left, right) match
        case (None, None)       => 0
        case (None, Some(_))    => -1
        case (Some(_), None)    => 1
        case (Some(x), Some(y)) => a.compare(x, y)

    given [A](using a: Structural[A]): Structural[List[A]] with
      def compare(left: List[A], right: List[A]): Int = sequence(left, right, a.compare)

    given [A](using a: Structural[A]): Structural[Set[A]] with
      private val ordering = Ordering.fromLessThan[A]((x, y) => a.compare(x, y) < 0)
      def compare(left: Set[A], right: Set[A]): Int =
        sequence(left.toList.sorted(using ordering), right.toList.sorted(using ordering), a.compare)

    given [K, V](using k: Structural[K], v: Structural[V]): Structural[Map[K, V]] with
      private val ordering = Ordering.fromLessThan[(K, V)]((x, y) => k.compare(x._1, y._1) < 0)
      def compare(left: Map[K, V], right: Map[K, V]): Int =
        sequence(
          left.toList.sorted(using ordering),
          right.toList.sorted(using ordering),
          (x, y) =>
            val key = k.compare(x._1, y._1)
            if key != 0 then key else v.compare(x._2, y._2)
        )

    given [T](using m: Mirror.Of[T], members: Members[m.MirroredElemTypes]): Structural[T] =
      derived[T]

  def derived[T](using m: Mirror.Of[T], members: Members[m.MirroredElemTypes]): Structural[T] =
    new Structural[T]:
      def compare(left: T, right: T): Int = m match
        case sum: Mirror.SumOf[T] =>
          val index = sum.ordinal(left)
          val tag = Integer.compare(index, sum.ordinal(right))
          if tag != 0 then tag else members.elements(index).compare(left, right)
        case _: Mirror.ProductOf[T] =>
          val x = left.asInstanceOf[Product] // scalafix:ok DisableSyntax.asInstanceOf
          val y = right.asInstanceOf[Product] // scalafix:ok DisableSyntax.asInstanceOf
          members.elements.iterator.zipWithIndex
            .map((field, index) => field.compare(x.productElement(index), y.productElement(index)))
            .find(_ != 0)
            .getOrElse(0)

  trait Members[Ts <: Tuple]:
    def elements: List[Data[Any]]

  object Members:
    given Members[EmptyTuple] with
      def elements: List[Data[Any]] = Nil
    given [H, T <: Tuple](using h: Structural[H], t: Members[T]): Members[H *: T] with
      def elements: List[Data[Any]] =
        h.asInstanceOf[Data[Any]] :: t.elements // scalafix:ok DisableSyntax.asInstanceOf

  private def sequence[A](left: List[A], right: List[A], compare: (A, A) => Int): Int =
    val x = left.iterator
    val y = right.iterator
    var result = 0
    while result == 0 && x.hasNext && y.hasNext do result = compare(x.next(), y.next())
    if result != 0 then result else java.lang.Boolean.compare(x.hasNext, y.hasNext)

@annotation.implicitNotFound(
  "${T} is not supported native state data; use enums, records, Boolean, Int, UpTo, Option, Set, Map, List or Inbox with supported members"
)
sealed trait StateData[T]

object StateData:
  given StateData[Boolean] = new StateData[Boolean] {}
  given StateData[Int] = new StateData[Int] {}
  given StateData[Nothing] = new StateData[Nothing] {}
  given [A](using StateData[A]): StateData[Option[A]] = new StateData[Option[A]] {}
  given [A](using CollectionMember[A]): StateData[List[A]] = new StateData[List[A]] {}
  given [A](using CollectionMember[A]): StateData[Set[A]] = new StateData[Set[A]] {}
  given [K, V](using CollectionMember[K], CollectionMember[V]): StateData[Map[K, V]] =
    new StateData[Map[K, V]] {}
  given [M](using StateData[M]): StateData[Inbox[M]] = new StateData[Inbox[M]] {}
  given [N <: Int]: StateData[UpTo[N]] = new StateData[UpTo[N]] {}
  // A Machine's State: each of its vars has been checked to be state data where it is declared.
  given StateData[State] = new StateData[State] {}
  given [T](using m: Mirror.Of[T], members: Members[m.MirroredElemTypes]): StateData[T] =
    new StateData[T] {}

  sealed trait Members[Ts <: Tuple]

  object Members:
    given Members[EmptyTuple] = new Members[EmptyTuple] {}
    given [H, T <: Tuple](using StateData[H], Members[T]): Members[H *: T] = new Members[H *: T] {}

  sealed trait CollectionMember[T]

  object CollectionMember:
    given CollectionMember[Boolean] = new CollectionMember[Boolean] {}
    given CollectionMember[Int] = new CollectionMember[Int] {}
    given CollectionMember[Nothing] = new CollectionMember[Nothing] {}
    given [N <: Int]: CollectionMember[UpTo[N]] = new CollectionMember[UpTo[N]] {}
    given [A](using CollectionMember[A]): CollectionMember[Option[A]] =
      new CollectionMember[Option[A]] {}
    given [A](using CollectionMember[A]): CollectionMember[List[A]] =
      new CollectionMember[List[A]] {}
    given [A](using CollectionMember[A]): CollectionMember[Set[A]] = new CollectionMember[Set[A]] {}
    given [K, V](using CollectionMember[K], CollectionMember[V]): CollectionMember[Map[K, V]] =
      new CollectionMember[Map[K, V]] {}
    given [T](using
        m: Mirror.Of[T],
        members: MemberFields[m.MirroredElemTypes]
    ): CollectionMember[T] =
      new CollectionMember[T] {}

  sealed trait MemberFields[Ts <: Tuple]

  object MemberFields:
    given MemberFields[EmptyTuple] = new MemberFields[EmptyTuple] {}
    given [H, T <: Tuple](using CollectionMember[H], MemberFields[T]): MemberFields[H *: T] =
      new MemberFields[H *: T] {}

@annotation.implicitNotFound(
  "${T} has no finite catalog. A finite domain is an enum, a case class whose fields are " +
    "finite domains, Boolean, Option of a finite domain or UpTo[N]."
)
trait Finite[T](using private[framework] val structural: Data.Structural[T]) extends Data[T]:
  final def compare(left: T, right: T): Int = structural.compare(left, right)
  def values: IndexedSeq[T]

object Finite:
  given [T](using Data[T]): CanEqual[T, T] = CanEqual.derived

  def apply[T](using f: Finite[T]): Finite[T] = f

  def of[T](vs: T*)(using Data.Structural[T]): Finite[T] =
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

  // A bounded counter's values `0..N`. It lives here rather than in `UpTo`'s companion, which is in
  // the implicit scope of `Data[UpTo[N]]`: there it would compete with Data's structural comparison
  // and make a machine whose whole state is an `UpTo[N]` ambiguous.
  given [N <: Int](using bound: ValueOf[N]): Finite[UpTo[N]] =
    of((0 to bound.value).map(UpTo[N](_))*)

  // An optional value: absent, then present with each member of `A` in its catalog order.
  given [A](using a: Finite[A], structural: Data.Structural[A]): Finite[Option[A]] =
    new Finite[Option[A]]:
      lazy val values: IndexedSeq[Option[A]] = None +: a.values.map(Some(_))

  // Every enum and case class is a domain without a `derives Finite` clause: its catalog is derived
  // here, from its `Mirror`, where a `Finite` is first needed. A more specific given wins over this
  // one, so `Option`, `UpTo` and a Model's own `given Finite[S]` (a channel's `Inbox`) keep theirs.
  //
  // The whole structure is checked by the implicit search itself, through `Members`: a case class
  // with a field that has no catalog, a plain `Int` or a list, has no `Finite`, and neither has a
  // union of two domains. Data comparison is derived independently of the catalogs.
  given [T](using
      m: Mirror.Of[T]
  )(using members: Members[m.MirroredElemTypes], structural: Data.Structural[T]): Finite[T] =
    derived[T]

  // The derived catalog of `T`, which a Model's own `given Finite[S]` block names when it binds a
  // field's catalog first (a channel's `Inbox`): `given Finite[S] = { given Finite[Inbox[M]] =
  // wire.contents; Finite.derived }`.
  def derived[T](using
      m: Mirror.Of[T]
  )(using members: Members[m.MirroredElemTypes], structural: Data.Structural[T]): Finite[T] =
    new Finite[T]:
      lazy val values: IndexedSeq[T] = m match
        case _: Mirror.Sum =>
          val cases = members.catalogs.flatMap(_.values)
          cases.asInstanceOf[List[T]].toIndexedSeq // scalafix:ok DisableSyntax.asInstanceOf
        case p: Mirror.Product =>
          val rows = product(members.catalogs).map(vs => p.fromProduct(Tuple.fromArray(vs.toArray)))
          rows.asInstanceOf[List[T]].toIndexedSeq // scalafix:ok DisableSyntax.asInstanceOf

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
