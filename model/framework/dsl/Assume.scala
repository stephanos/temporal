package framework.dsl

// A named assumption a check makes, which every result the check reports names: that a machine
// stands for an opaque provider, that it includes a fault only this assumption allows, or the
// fairness a progress claim rests on.
final class Assumption private[framework] (
    val name: String,
    val fairness: List[ActionDecl],
    val composed: List[Composed] = Nil,
    private[framework] val fairSegments: Vector[Assumption.Fairness] = Vector.empty
) extends RetainedDeclarationName:
  private[framework] def nativeIdentity: Option[String] =
    scala.util.Try(declarationName.identity(s"assumption $name")).toOption
  def toIR: io.temporal.server.api.umpire.v1.Assumption = Signature.assumption(this)
  private def retained(value: Assumption): Assumption =
    declarationName.copyTo(value.declarationName)
    value
  // Makes the actions weakly fair under this assumption: one that stays enabled is taken.
  def fair(actions: Action[?]*): Assumption =
    val selected = actions.map(_.decl).toList
    retained(
      Assumption(
        name,
        fairness ++ selected,
        composed,
        fairSegments :+ Assumption.Fairness.Actions(selected)
      )
    )

  // Makes composed steps of one composition weakly fair, for a progress claim of that composition:
  // `assume.fair(c.synced(_.order -> dispatch), c.own(_.queue, queue.persist))`. Each selects an
  // action, every class of it, or one class of it, `c.own(_.queue, enqueue(Priority.high))`, as a
  // Scenario selects them; a synced step is the sync's, whose enabledness is the composition's, never
  // its member's alone. The lifter refuses an assumption that mixes both forms, one selecting steps of
  // another composition than the claim's, and one a machine assumes.
  def fair(first: Composed, more: Composed*): Assumption =
    val selected = first :: more.toList
    retained(
      Assumption(
        name,
        fairness,
        composed ++ selected,
        fairSegments :+ Assumption.Fairness.Composed(selected)
      )
    )

  override def toString: String = s"assumption $name"

object Assumption:
  private[framework] enum Fairness:
    case Actions(values: List[ActionDecl])
    case Composed(values: List[framework.dsl.Composed])

// Declares an assumption.
def assume(name: String): Assumption = Assumption(name, Nil).retainExplicitName(true)

// Declares an assumption named after the `val` that declares it.
inline def assume: Assumption = inferredAssumption(NativeLocation.declaredName)

@scala.annotation.publicInBinary
private[framework] def inferredAssumption(name: Option[String]): Assumption =
  val value = Assumption("", Nil).retainExplicitName(false)
  name.foreach(value.declarationName.spelledAt)
  value

// A declared hole: behavior the Model leaves unknown on purpose. A step that reaches it is neither
// disabled, as an empty list of steps is, nor an error: a result that depends on it is incomplete
// (model/SEMANTICS.md, Holes).
final class Hole private[framework] (val name: String) extends RetainedDeclarationName:
  def toIR: io.temporal.server.api.umpire.v1.Hole = Signature.hole(this)
  // Where a step function reaches the hole. Only the IR interpreter reads holes; a step function
  // run as Scala stops here.
  def reached: Nothing = throw HoleReached(this)

  override def toString: String = s"hole $name"

// Declares a hole named after the `val` that declares it.
def hole: Hole = Hole("").retainExplicitName(false)

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

extension [S, O, F](m: RecordMachine[S, O, F])
  def leadsTo(
      name: String
  )(from: S => Boolean, to: S => Boolean, within: Int, under: Assumption*): Progress[S] =
    Progress(name, m, from, to, within, under.toList)

extension [S <: Product](c: Composition[S])
  def leadsTo(
      name: String
  )(from: S => Boolean, to: S => Boolean, within: Int, under: Assumption*): Progress[S] =
    Progress(name, c, from, to, within, under.toList)
