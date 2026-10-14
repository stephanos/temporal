package framework.dsl

// A named assumption a check makes, which every result the check reports names: that a machine
// stands for an opaque provider, that it includes a fault only this assumption allows, or the
// fairness a progress claim rests on.
final class Assumption private[framework] (
    val name: String,
    val fairness: List[ActionDecl],
    val composed: List[Composed] = Nil
):
  // Makes the actions weakly fair under this assumption: one that stays enabled is taken.
  def fair(actions: Action[?]*): Assumption = Assumption(name, actions.map(_.decl).toList)

  // Makes composed steps of one composition weakly fair, for a progress claim of that composition:
  // `assume.fair(c.synced(_.order -> dispatch), c.own(_.queue, queue.persist))`. Each selects an
  // action, every class of it, or one class of it, `c.own(_.queue, enqueue(Priority.high))`, as a
  // Scenario selects them; a synced step is the sync's, whose enabledness is the composition's, never
  // its member's alone. The lifter refuses an assumption that mixes both forms, one selecting steps of
  // another composition than the claim's, and one a machine assumes.
  def fair(first: Composed, more: Composed*): Assumption =
    Assumption(name, Nil, first :: more.toList)

  override def toString: String = s"assumption $name"

// Declares an assumption.
def assume(name: String): Assumption = Assumption(name, Nil)

// Declares an assumption named after the `val` that declares it.
def assume: Assumption = assume("")

// A declared hole: behavior the Model leaves unknown on purpose. A step that reaches it is neither
// disabled, as an empty list of steps is, nor an error: a result that depends on it is incomplete
// (model/SEMANTICS.md, Holes).
final class Hole private[framework] (val name: String):
  // Where a step function reaches the hole. Only the IR interpreter reads holes; a step function
  // run as Scala stops here.
  def reached: Nothing = throw HoleReached(this)

  override def toString: String = s"hole $name"

// Declares a hole named after the `val` that declares it.
def hole: Hole = Hole("")

final private[framework] case class HoleReached(hole: Hole)
    extends Exception(s"${hole.name} reached")

// A bounded progress claim over one machine or composition: from any state `from` accepts, a state
// `to` accepts is reached within `within` steps, under the assumptions named. A finite path that ends
// before either says nothing; the IR interpreter checks the claim (model/SEMANTICS.md, Progress). A
// composition's claim reads its composed state, counts every composed step, a sync's once, and rests on
// its members' assumptions and on the composed fairness its own assumptions select.
final class Progress[S] private[framework] (
    val name: String,
    val machine: Model,
    val from: S => Boolean,
    val to: S => Boolean,
    val within: Int,
    val assumptions: List[Assumption]
):
  override def toString: String = s"progress $name"

extension [S, O, F](m: Machine[S, O, F])
  def leadsTo(
      name: String
  )(from: S => Boolean, to: S => Boolean, within: Int, under: Assumption*): Progress[S] =
    Progress(name, m, from, to, within, under.toList)

extension [S <: Product](c: Composition[S])
  def leadsTo(
      name: String
  )(from: S => Boolean, to: S => Boolean, within: Int, under: Assumption*): Progress[S] =
    Progress(name, c, from, to, within, under.toList)
