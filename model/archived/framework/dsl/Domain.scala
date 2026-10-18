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
  final def equivalent(left: T, right: T): Boolean =
    NativeRuntime.installed.equivalent(this, left, right)
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

  object Structural extends SelectedStructural:
    @scala.annotation.publicInBinary
    private[framework] def selected[T](finite: Finite[T]): Structural[T] = finite.structural
    // The comparison of a state var whose type has no schema, which its Machine refuses at compile
    // time; it stands in only so the refusal is the one reported.
    @scala.annotation.publicInBinary
    private[framework] def unschematic[T]: Structural[T] = new Structural[T]:
      def compare(left: T, right: T): Int =
        throw IllegalStateException("a state var without a schema is refused at compile time")

    private def ordered[T](ordering: Ordering[T]): Structural[T] =
      Native(Kind.ordered, ordering = Some(ordering))

    given Structural[Boolean] = ordered(Ordering.Boolean)
    given Structural[Int] = ordered(Ordering.Int)
    given [N <: Int]: Structural[UpTo[N]] = ordered(Ordering.Int.on[UpTo[N]](identity))
    given [M](using message: Structural[M]): Structural[Inbox[M]] = Inbox.structural(message)
    given Structural[String] = Native(Kind.string)
    given Structural[Nothing] with
      def compare(left: Nothing, right: Nothing): Int = left
    given [A](using a: Structural[A]): Structural[Option[A]] = Native(Kind.option, fields(a))
    given [A](using a: Structural[A]): Structural[List[A]] = Native(Kind.list, fields(a))
    given [A](using a: Structural[A]): Structural[Set[A]] = Native(Kind.set, fields(a))
    given [K, V](using k: Structural[K], v: Structural[V]): Structural[Map[K, V]] =
      Native(Kind.map, fields(k, v))
    given [T](using m: Mirror.Of[T], members: Members[m.MirroredElemTypes]): Structural[T] =
      derived[T]

  private[framework] enum Kind:
    case ordered, string, option, list, set, map, mirrored

  final private[framework] class Native[T](
      val kind: Kind,
      val fields: List[Data[Any]] = Nil,
      val mirror: Option[Mirror.Of[T]] = None,
      val ordering: Option[Ordering[T]] = None
  ) extends Structural[T]:
    def compare(left: T, right: T): Int = NativeRuntime.installed.compare(this, left, right)

  private def fields(values: Data[?]*): List[Data[Any]] =
    values.toList.asInstanceOf[List[Data[Any]]] // scalafix:ok DisableSyntax.asInstanceOf

  def derived[T](using m: Mirror.Of[T], members: Members[m.MirroredElemTypes]): Structural[T] =
    Native(Kind.mirrored, members.elements, Some(m))

  trait Members[Ts <: Tuple]:
    def elements: List[Data[Any]]

  object Members:
    given Members[EmptyTuple] with
      def elements: List[Data[Any]] = Nil
    given [H, T <: Tuple](using h: Structural[H], t: Members[T]): Members[H *: T] with
      def elements: List[Data[Any]] =
        h.asInstanceOf[Data[Any]] :: t.elements // scalafix:ok DisableSyntax.asInstanceOf

@annotation.implicitNotFound(
  "${T} is not supported native state data; use enums, records, Boolean, Int, UpTo, Option, Set, Map, List or Inbox with supported members"
)
sealed trait StateData[T]

private[dsl] trait SelectedStateData:
  inline given selectedState[T](using selected: Finite[T]): StateData[T] =
    ${ SelectedDomain.state[T]('selected) }

private[dsl] trait SelectedCollectionMember:
  inline given selectedMember[T](using selected: Finite[T]): StateData.CollectionMember[T] =
    ${ SelectedDomain.member[T]('selected) }

object StateData extends SelectedStateData:
  @scala.annotation.publicInBinary
  private[framework] def parameter[T](selected: Finite[T]): StateData[T] =
    nativeMember(selected.structural)
    new StateData[T] {}

  private def nativeMember(data: Data[?]): Unit =
    def supported(data: Data[?]): Boolean = data match
      case empty if empty eq summon[Data.Structural[Nothing]] => true
      case native: Data.Native[?]                             =>
        native.kind != Data.Kind.string && native.fields.forall(supported)
      case _ => false
    require(
      supported(data),
      "a Model domain used in state collections requires supported native members"
    )
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

  object CollectionMember extends SelectedCollectionMember:
    @scala.annotation.publicInBinary
    private[framework] def parameter[T](selected: Finite[T]): CollectionMember[T] =
      nativeMember(selected.structural)
      new CollectionMember[T] {}
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
  final def compare(left: T, right: T): Int =
    NativeRuntime.installed.compareData(structural, left, right)
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
  def upTo(hi: Int): Finite[Int] = new Finite[Int]:
    lazy val values: IndexedSeq[Int] = NativeRuntime.installed.range(hi)

  // False before true, the order the Go reader lists a Boolean's values in.
  given Finite[Boolean] = of(false, true)

  // A domain with no members: a machine that records no facts has `Nothing` as its fact type.
  given Finite[Nothing] = of()

  // A bounded counter's values `0..N`. It lives here rather than in `UpTo`'s companion, which is in
  // the implicit scope of `Data[UpTo[N]]`: there it would compete with Data's structural comparison
  // and make a machine whose whole state is an `UpTo[N]` ambiguous.
  given [N <: Int](using bound: ValueOf[N]): Finite[UpTo[N]] =
    new Finite[UpTo[N]]:
      lazy val values: IndexedSeq[UpTo[N]] =
        NativeRuntime.installed.range(bound.value).map(UpTo[N](_))

  // An optional value: absent, then present with each member of `A` in its catalog order.
  given [A](using a: Finite[A], structural: Data.Structural[A]): Finite[Option[A]] =
    new Finite[Option[A]]:
      lazy val values: IndexedSeq[Option[A]] = NativeRuntime.installed.optional(a)

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
      lazy val values: IndexedSeq[T] = NativeRuntime.installed.catalog(m, members.catalogs)

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

// A counter of `0..N`, bounded where its field is declared: `attempts: UpTo[2]` has the values 0, 1
// and 2, in that order. It reads as an Int; a step writes one with `UpTo(n)`, which the lifter lifts as
// `n`, and Go refuses a value outside the range as it does for any bounded field.
opaque type UpTo[N <: Int] <: Int = Int

object UpTo:
  // The counter at `n`, which lies in `0..N`.
  def apply[N <: Int](n: Int)(using bound: ValueOf[N]): UpTo[N] =
    require(0 <= n && n <= bound.value, s"$n is outside 0..${bound.value}")
    n
