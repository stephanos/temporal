package framework

// A named assumption a check makes, which every result the check reports names: that a machine
// stands for an opaque provider, that it includes a fault only this assumption allows, or the
// fairness a progress claim rests on.
final class Assumption private[framework] (val name: String, val fairness: List[ActionDecl]):
  // Makes the actions weakly fair under this assumption: one that stays enabled is taken.
  def fair(actions: Action[?]*): Assumption = Assumption(name, actions.map(_.decl).toList)

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

// A bounded progress claim over one machine: from any state `from` accepts, a state `to` accepts is
// reached within `within` steps, under the assumptions named. A finite path that ends before either
// says nothing; the IR interpreter checks the claim (model/SEMANTICS.md, Progress).
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
