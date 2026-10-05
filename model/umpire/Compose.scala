package umpire

import scala.annotation.publicInBinary
import scala.compiletime.constValueTuple
import scala.deriving.Mirror

/**
 * One Model from machines of different entities, for a claim no one of them can state.
 * The composed state `S` is a case class with one field per member, named after
 * the member. A `sync` pairs member actions into one step; a member action no `sync` names steps
 * its member alone.
 *
 * A member is named by a selector of its field, `_.order -> currentRecord`; `->` pairs the
 * member with its value and never means a transition. The lifter reads the selectors from the
 * source, so what a composition holds here is what its author wrote.
 */
final class Composition[S <: Product] @publicInBinary private[umpire] (
    val family: Family,
    val name: String,
    private[umpire] val fieldNames: Vector[String],
    private[umpire] val members: Vector[S => (Any, Model)],
    private[umpire] val syncs: Vector[(String, Move[S], Move[S])],
    private[umpire] val isEnd: S => Boolean,
    private[umpire] val replaced: Vector[(S => Any, Model)],
    private[umpire] val withMembers: Vector[S => (Any, Model)] = Vector.empty
) extends Declares[S]:
  type Outcome = String
  type Fact = String

  private def copy(
      family: Family = family,
      name: String = name,
      syncs: Vector[(String, Move[S], Move[S])] = syncs,
      isEnd: S => Boolean = isEnd,
      replaced: Vector[(S => Any, Model)] = replaced,
      withMembers: Vector[S => (Any, Model)] = withMembers
  ): Composition[S] =
    Composition(family, name, fieldNames, members, syncs, isEnd, replaced, withMembers)

  /**
   * Pairs two members' actions into one step named `name`, each member by its field:
   * `.sync("dispatch", _.order -> dispatch, _.queue -> enqueue)`.
   */
  def sync(
      name: String,
      first: S => (Any, Action[?]),
      second: S => (Any, Action[?])
  ): Composition[S] =
    copy(syncs = syncs :+ (name, first, second))

  /**
   * Pairs two members' actions into one step named after the first member's action, each member by
   * its field: `.sync(_.order -> dispatch, _.queue -> enqueue)` is the sync `dispatch`. The
   * lifter reads the name from the action's declaration, so it is empty here.
   */
  def sync(first: S => (Any, Action[?]), second: S => (Any, Action[?])): Composition[S] =
    copy(syncs = syncs :+ ("", first, second))

  /** Which composed states the composition may end in. */
  def ends(end: S => Boolean): Composition[S] = copy(isEnd = end)

  /**
   * Says the member a field selector names, `_.queue`, stands in for `opaque` within this
   * composition: a detailed provider in place of an opaque one. The member must declare a refinement
   * of `opaque` that holds, which is what makes the replacement scoped to this composition.
   */
  def replaces(field: S => Any, opaque: Model): Composition[S] =
    copy(replaced = replaced :+ (field -> opaque))

  /**
   * This composition with one member replaced, `currentOverQueue.withMember(_.order ->
   * staleRecord)`: the same syncs, ends and member order, named after the `val` that declares it in
   * the `given Family`. A member that stands in for another machine here stands in for the one its
   * new machine declares it refines; the lifter refuses a new machine of another state type, one that
   * binds no action a sync of the member pairs, and one that refines nothing where the member
   * replaces a machine.
   */
  def withMember(member: S => (Any, Model))(using family: Family): Composition[S] =
    copy(family = family, name = "", withMembers = withMembers :+ member)

  /**
   * The step a sync takes, by one of the member actions it pairs, `c.synced(_.order -> dispatch)`:
   * a class a Scenario of this composition lists, or the action `whenAction` names. The lifter
   * refuses a member action no sync pairs, or more than one does.
   */
  def synced(move: S => (Any, Class | Action[?])): Composed = Composed(this, move)

  /**
   * A step a member takes alone, `c.own(_.order, control(Control.pause))`: a class a Scenario of
   * this composition lists, or the action `whenAction` names. The lifter refuses an action a sync of
   * the member pairs, since that action steps only with its pair.
   */
  def own(member: S => Any, action: Class | Action[?]): Composed = Composed(this, (member, action))

/** One member's action a sync pairs, by a selector of the member's field. */
type Move[S] = S => (Any, Action[?])

/** A composed class or action of a composition, as `synced` and `own` select it. */
final class Composed private[umpire] (val composition: Model, val selected: Any)

/**
 * A member's def read from the composed state: `through(_.order, Order.held)` is
 * `s => Order.held(s.order)`. A composition's capability field or a declaring function's
 * function-valued argument names it where it would name a def, so a law reads a member's status
 * set without a def that restates it for the composition. The IR generator lifts it as one function
 * of the composed state, named after the state, the path and the def, and refuses a selector that is
 * not a field path and a `read` that is not a def of the lifted sources, as it refuses a lambda.
 *
 * Core: it says which member a law reads, which no def of the lifted sources says. Its two arguments
 * share one parameter list so that the composed state is inferred from where it is passed.
 */
def through[S, M, A](select: S => M, read: M => A): S => A = s => read(select(s))

/**
 * Starts a composition named after the `val` that declares it, in the `given Family`, over members
 * each named by a selector of the field it fills: `compose[OverQueue](_.order -> currentRecord,
 * _.queue -> dispatchQueue)`.
 */
inline def compose[S <: Product](members: (S => (Any, Model))*)(using
    m: Mirror.ProductOf[S],
    family: Family
): Composition[S] =
  val labels = constValueTuple[m.MirroredElemLabels].toList.map(_.toString).toVector
  Composition[S](family, "", labels, members.toVector, Vector.empty, _ => false, Vector.empty)
