package umpire

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
// then its `states`, `properties`, `implements` and `queries` sections. A composition with one member replaced is an
// object too, `object LateOverQueue extends Composition(OrderOverQueue.withMember(_.order ->
// LateRecord))`, and declares only its own sections. `end` and `syncs` are not members a declared
// composition must implement, since a derived one keeps its source's: the IR generator requires them
// of a declared composition object and refuses them in a derived one. At run time a composition is
// its members, which the gate constructs (IrFile.construct).
abstract class Composition[S <: Product] private (
    private[umpire] val shape: Composition.Shape[S],
    private[umpire] val mirror: Mirror.ProductOf[S]
) extends Declares[S]:
  type Outcome = String
  type Fact = String

  // A composition of these members, each named by a selector of its field.
  def this(members: (S => (Any, Model))*)(using mirror: Mirror.ProductOf[S]) =
    this(Composition.Shape.Members(members.toVector), mirror)

  // The composition a derivation of another makes, such as `c.withMember(...)`.
  def this(derivation: Composition[S]) =
    this(Composition.Shape.Of(derivation), derivation.mirror)

  // The object's name with its first letter lowered.
  def name: String = objectName(this)

  // The owner its `syncs` read the composed state type from.
  protected given compositionOwner: Composer[S] = Composer(this)

  // What its `implements` declares the capabilities of: this composition.
  protected given declaring: Declaring[S, String, String] = Declaring(this)

  // The members, each constructed: the machines and compositions it composes.
  private[umpire] def members: Vector[Model] = shape match
    case Composition.Shape.Members(selectors) => selectors.map(Composition.selected(_, mirror))
    case Composition.Shape.Of(derivation)     => derivation.members
    case Composition.Shape.With(base, member) =>
      base.members :+ Composition.selected(member, mirror)

  // This composition with one member replaced, `OrderOverQueue.withMember(_.order ->
  // LateRecord)`: the same syncs, ends and member order, named after the object that declares it,
  // `object LateOverQueue extends Composition(OrderOverQueue.withMember(...))`. A member that stands
  // in for another machine here stands in
  // for the one its new machine declares it refines; the lifter refuses a new machine of another
  // state type, one that binds no action a sync of the member pairs, and one that refines nothing
  // where the member replaces a machine.
  def withMember(member: S => (Any, Model)): Composition[S] =
    new Composition[S](Composition.Shape.With(this, member), mirror) {}

  // The step a sync takes, by one of the member actions it pairs, `c.synced(_.order -> dispatch)`:
  // a class a Scenario of this composition lists, or the action `whenAction` names. The lifter
  // refuses a member action no sync pairs, or more than one does.
  def synced(move: S => (Any, Class | Action[?])): Composed = Composed(this, move)

  // A step a member takes alone, `c.own(_.order, control(Control.pause))`: a class a Scenario of
  // this composition lists, or the action `whenAction` names. The lifter refuses an action a sync of
  // the member pairs, since that action steps only with its pair.
  def own(member: S => Any, action: Class | Action[?]): Composed = Composed(this, (member, action))

object Composition:
  // How a composition is made: from its members, from another's derivation, or with a member.
  private[umpire] enum Shape[S <: Product]:
    case Members(selectors: Vector[S => (Any, Model)])
    case Of(derivation: Composition[S])
    case With(base: Composition[S], member: S => (Any, Model))

  // A selector reads one field of the composed state and names the member that fills it, so the
  // member is read off a composed state of empty fields: a case class's constructor keeps them.
  private object Empty extends Product:
    def canEqual(that: Any): Boolean = false
    def productArity: Int = 0
    // An empty field is null, as a case class constructor keeps it.
    def productElement(n: Int): Any = null // scalafix:ok DisableSyntax.null

  private[umpire] def selected[S](selector: S => (Any, Model), m: Mirror.ProductOf[S]): Model =
    selector(m.fromProduct(Empty))._2

// The owner of a composition's `syncs`, which read its composed state type.
final class Composer[S <: Product] private[umpire] (val composition: Composition[S])

// A composition's syncs, `object syncs extends Syncs`: each statement pairs two members' actions
// into one step, `sync(_.order -> clerk.dispatch, _.queue -> queue.enqueue)`, named after the
// first member's action or by the name it is given, `sync("admit", ...)`; `replaces(_.queue,
// OpaqueProduct)` says a member stands in for an opaque machine. The IR generator reads them from the
// source, in order.
abstract class Syncs[S <: Product](using @unused composer: Composer[S]):
  // Pairs two members' actions into one step named after the first member's action.
  def sync(@unused first: S => (Any, Action[?]), @unused second: S => (Any, Action[?])): Unit = ()

  // Pairs two members' actions into one step named `name`.
  def sync(
      @unused name: String,
      @unused first: S => (Any, Action[?]),
      @unused second: S => (Any, Action[?])
  ): Unit = ()

  // The member a field selector names, `_.queue`, stands in for `opaque` within this composition:
  // a detailed provider in place of an opaque one, which must declare a refinement of it.
  def replaces(@unused field: S => Any, @unused opaque: Model): Unit = ()

// One member's action a sync pairs, by a selector of the member's field.
type Move[S] = S => (Any, Action[?])

// A composed class or action of a composition, as `synced` and `own` select it.
final class Composed private[umpire] (val composition: Model, val selected: Any)

// A member's def read from the composed state: `through(_.order, Order.held)` is
// `s => Order.held(s.order)`. A composition's capability field or a declaring function's
// function-valued argument names it where it would name a def, so a law reads a member's status
// set without a def that restates it for the composition. The IR generator lifts it as one function
// of the composed state, named after the state, the path and the def, and refuses a selector that is
// not a field path and a `read` that is not a def of the lifted sources, as it refuses a lambda.
//
// Core: it says which member a law reads, which no def of the lifted sources says. Its two arguments
// share one parameter list so that the composed state is inferred from where it is passed.
def through[S, M, A](select: S => M, read: M => A): S => A = s => read(select(s))
