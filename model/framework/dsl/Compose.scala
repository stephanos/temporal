package framework.dsl

import scala.annotation.unused
import scala.deriving.Mirror

// One Model from machines of different entities, for a claim no one of them can state.
// The composed state `S` is a case class with one field per member, named after
// the member. A `sync` pairs member actions into one step; a member action no `sync` names steps
// its member alone.
//
// A member is named by a selector of its field, `_.order -> recordMember`; `->` pairs the
// member with its value and never means a transition. The lifter reads the selectors from the
// source, so what a composition holds here is what its author wrote.
// A member's helper reads its state directly, `Order.held(s.order)`. A composition's facts
// carry the member's fact in the enum case it declares for that member.
//
// Declared as an object that is the composition, named after the object with its first letter
// lowered, with the states it may end in and its syncs:
//
// {{{
// object OrderOverQueue extends Composition[OverQueue](_.order -> OrderRecord, _.queue -> Queue):
//   def end(s: OverQueue) = OrderRecord.end(s.order)
//   object syncs extends Syncs:
//     sync(_.order -> clerk.dispatch, _.queue -> queue.enqueue)
// }}}
//
// then its `states`, `properties`, `capabilities` and `queries` sections. A composition with one member replaced is an
// object too, `object LateOverQueue extends Composition(OrderOverQueue.withMember(_.order ->
// LateRecord))`, and declares only its own sections. `end` and `syncs` are not members a declared
// composition must implement, since a derived one keeps its source's: the IR generator requires them
// of a declared composition object and refuses them in a derived one. At run time a composition is
// its members, which the gate constructs (IrFile.construct).
abstract class Composition[S <: Product] private (
    private[framework] val shape: Composition.Shape[S],
    private[framework] val mirror: Mirror.ProductOf[S]
)(using private[framework] val fs: Data[S], private[framework] val ss: StateData[S])
    extends Declares[S]:
  type Outcome = String

  override def reach: Limits = shape match
    case Composition.Shape.Members(_)    => super.reach
    case Composition.Shape.Of(source)    => source.reach
    case Composition.Shape.With(base, _) => base.reach

  // A composition of these members, each named by a selector of its field.
  def this(members: (S => MemberSelection)*)(using
      mirror: Mirror.ProductOf[S],
      data: Data[S],
      state: StateData[S]
  ) =
    this(Composition.Shape.Members(members.toVector.map(Composition.retain(_, mirror))), mirror)

  // The composition a derivation of another makes, such as `c.withMember(...)`.
  def this(derivation: Composition[S]) =
    this(Composition.Shape.Of(derivation), derivation.mirror)(using derivation.fs, derivation.ss)

  // The object's name with its first letter lowered.
  def name: String = objectName(this)

  // The owner its `syncs` read the composed state type from.
  protected given compositionOwner: Composer[S] = Composer(this)

  // What its `capabilities` section declares the capabilities of: this composition.
  protected given declaring: Declaring[S, String, Fact] = Declaring(this)

  // The members, each constructed: the machines and compositions it composes.
  private[framework] def members: Vector[Model] = shape match
    case Composition.Shape.Members(selectors) => selectors.map(Composition.selected(_, mirror))
    case Composition.Shape.Of(derivation)     => derivation.members
    case Composition.Shape.With(base, member) =>
      base.members :+ Composition.selected(member, mirror)

  private[framework] lazy val memberFields: Vector[(String, Model)] =
    def selected(member: S => (Any, Model)): (String, Model) =
      val field = MemberSelector
        .retainedField(member)
        .getOrElse(
          throw IllegalArgumentException(s"$name: no retained direct member selector metadata")
        )
      field -> MemberSelector.value(member, mirror.fromProduct(Composition.Empty))._2
    shape match
      case Composition.Shape.Members(selectors) =>
        val fields = selectors.map(selected)
        require(fields.map(_._1).distinct.size == fields.size, s"$name: duplicate member fields")
        fields
      case Composition.Shape.Of(derivation)     => derivation.memberFields
      case Composition.Shape.With(base, member) =>
        val fields = base.memberFields
        val replacement = selected(member)
        require(fields.exists(_._1 == replacement._1), s"$name: no member ${replacement._1}")
        fields.map(value => if value._1 == replacement._1 then replacement else value)

  private[framework] def referenceSyncs: Vector[Syncs.Statement] = shape match
    case Composition.Shape.Members(_)    => Syncs.declaredBy(this).toVector.flatMap(_.recorded)
    case Composition.Shape.Of(source)    => source.referenceSyncs
    case Composition.Shape.With(base, _) => base.referenceSyncs

  // This composition with one member replaced, `OrderOverQueue.withMember(_.order ->
  // LateRecord)`: the same syncs, ends and member order, named after the object that declares it,
  // `object LateOverQueue extends Composition(OrderOverQueue.withMember(...))`. A member that stands
  // in for another machine here stands in
  // for the one its new machine declares it refines; the lifter refuses a new machine of another
  // state type, one that binds no action a sync of the member pairs, and one that refines nothing
  // where the member replaces a machine.
  def withMember(
      member: S => MemberSelection
  ): Composition[S] { type Fact = Composition.this.Fact } & DerivedFrom[this.type] =
    new Composition[S](Composition.Shape.With(this, Composition.retain(member, mirror)), mirror)(
      using
      fs,
      ss
    ) with DerivedFrom[this.type]:
      type Fact = Composition.this.Fact
      private[framework] def source: Composition.this.type = Composition.this

  // The step a sync takes, by one of the member actions it pairs, `c.synced(_.order -> dispatch)`:
  // a class a Scenario of this composition lists, the action `whenAction` names, or the steps an
  // assumption of its progress claims makes fair (framework/dsl/Assume.scala). The lifter
  // refuses a member action no sync pairs, or more than one does.
  def synced(move: S => (Any, Class | Action[?])): Composed = Composed(this, move, "synced")

  // A step a member takes alone, `c.own(_.order, control(Control.pause))`: a class a Scenario of
  // this composition lists, the action `whenAction` names, or the steps an assumption of its progress
  // claims makes fair. The lifter refuses an action a sync of the member pairs, since that action
  // steps only with its pair.
  def own(member: S => Any, action: Class | Action[?]): Composed =
    Composed(this, (member, action), "own")

  // The key of one member outcome in this composition. The member fixes the outcome type, so a
  // capability cannot bind an outcome of another machine or hand-write the composed encoding.
  def composedOutcome[MS, O, F](member: RecordMachine[MS, O, F], outcome: O): String =
    val fields = mirror.fromProduct(Composition.Empty).productElementNames.toVector
    val candidates = fields.zip(members).collect { case (field, m) if m eq member => field }
    require(
      candidates.size == 1,
      s"${member.name} is not one unique member of $name"
    )
    s"${candidates.head}_${valueKey(outcome)}"

// A composition derivation, as `Composition(derivation)` is, declared as an object that is it:
// `object LateOverQueue extends DerivedComposition(OrderOverQueue.withMember(...))`. It keeps its
// source's syncs and end, which the lifter reads from the source.
abstract class DerivedComposition[S <: Product, F](derivation: Composition[S] { type Fact = F })
    extends Composition[S](derivation):
  type Fact = F

object Composition:
  private[framework] def retain[S](
      selector: S => MemberSelection,
      mirror: Mirror.ProductOf[S]
  ): S => (Any, Model) =
    val selected = selector(mirror.fromProduct(Empty))
    MemberSelector.retain(
      state =>
        val member = selector(state)
        member.value -> member.model
      ,
      selected.field
    )

  // How a composition is made: from its members, from another's derivation, or with a member.
  private[framework] enum Shape[S <: Product]:
    case Members(selectors: Vector[S => (Any, Model)])
    case Of(derivation: Composition[S])
    case With(base: Composition[S], member: S => (Any, Model))

  // A selector reads one field of the composed state and names the member that fills it, so the
  // member is read off a composed state of empty fields: a case class's constructor keeps them.
  private[framework] object Empty extends Product:
    def canEqual(that: Any): Boolean = false
    def productArity: Int = 0
    // An empty field is null, as a case class constructor keeps it.
    def productElement(n: Int): Any = null // scalafix:ok DisableSyntax.null

  private[framework] def selected[S](selector: S => (Any, Model), m: Mirror.ProductOf[S]): Model =
    selector(m.fromProduct(Empty))._2

// The owner of a composition's `syncs`, which read its composed state type.
final class Composer[S <: Product] private[framework] (val composition: Composition[S])

// A composition's syncs, `object syncs extends Syncs`: each statement pairs two members' actions
// into one step, `sync(_.order -> clerk.dispatch, _.queue -> queue.enqueue)`, named after the
// first member's action or by the name it is given, `sync("admit", ...)`; `replaces(_.queue,
// OpaqueProduct)` says a member stands in for an opaque machine. The IR generator reads them from the
// source, in order.
abstract class Syncs[S <: Product](using composer: Composer[S]):
  // Retains statement order and endpoint identities for native declaration discovery.
  private[framework] val recorded = scala.collection.mutable.ArrayBuffer.empty[Syncs.Statement]

  private def move(selector: S => (Any, Action[?])): ActionDecl =
    selector(composer.composition.mirror.fromProduct(Composition.Empty))._2.decl

  // Pairs two members' actions into one step named after the first member's action.
  def sync(first: S => (Any, Action[?]), second: S => (Any, Action[?])): Unit =
    val statement = Syncs.Statement.Sync("", move(first), move(second))
    statement.endpoints = Some(
      (MemberSelector.retainedField(first), MemberSelector.retainedField(second), false)
    )
    recorded += statement: Unit

  // Pairs two members' actions into one step named `name`.
  def sync(
      name: String,
      first: S => (Any, Action[?]),
      second: S => (Any, Action[?])
  ): Unit =
    val statement = Syncs.Statement.Sync(name, move(first), move(second))
    statement.endpoints = Some(
      (MemberSelector.retainedField(first), MemberSelector.retainedField(second), true)
    )
    recorded += statement: Unit

  // The member a field selector names, `_.queue`, stands in for `opaque` within this composition:
  // a detailed provider in place of an opaque one, which must declare a refinement of it.
  def replaces(@unused field: S => Any, opaque: Model): Unit =
    recorded += Syncs.Statement.Replaces(opaque): Unit

object Syncs:
  // A statement of a `syncs` object.
  private[framework] enum Statement:
    case Sync(name: String, first: ActionDecl, second: ActionDecl)
    case Replaces(opaque: Model)

    private[framework] var endpoints: Option[(Option[String], Option[String], Boolean)] = None

  // The `syncs` object `composition` declares, where it declares one: a member object a
  // composition need not declare, found as the object's nested module, `<Composition>$syncs$`, as
  // `Refinement.declaredBy` finds a refinement.
  private[framework] def declaredBy(composition: Composition[?]): Option[Syncs[?]] =
    val declaration =
      try Some(java.lang.Class.forName(composition.getClass.getName + "syncs$"))
      catch case _: ClassNotFoundException => None
    declaration
      .map(c => c.getField("MODULE$").get(null)) // scalafix:ok DisableSyntax.null
      .map:
        case syncs: Syncs[?] => syncs
        case other           =>
          throw IllegalStateException(s"${composition.name}.syncs is not Syncs: $other")

// One member's action a sync pairs, by a selector of the member's field.
type Move[S] = S => (Any, Action[?])

// A composed class or action of a composition, as `synced` and `own` select it.
final class Composed private[framework] (
    val composition: Model,
    val selected: Any,
    private[framework] val operation: String = ""
):
  private[framework] lazy val retainedAction: Option[Class | Action[?]] =
    MemberSelector.selected(selected)
  def toIR(declaration: String): io.temporal.server.api.umpire.v1.ComposedActionRef =
    ComposedReference.convert(this, declaration)
