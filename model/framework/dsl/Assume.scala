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

// A progress claim of a Machine that declares its state as vars: `from` and `to` read them, with
// the state they are asked of restored into them.
extension [O, F](m: RecordMachine[State, O, F])
  @scala.annotation.targetName("leadsToVars")
  def leadsTo(
      name: String
  )(from: => Boolean, to: => Boolean, within: Int, under: Assumption*): Progress[State] =
    val vars = Vars.of(m)
    Progress(name, m, s => vars.at(s)(from), s => vars.at(s)(to), within, under.toList)

extension [S <: Product](c: Composition[S])
  def leadsTo(
      name: String
  )(from: S => Boolean, to: S => Boolean, within: Int, under: Assumption*): Progress[S] =
    Progress(name, c, from, to, within, under.toList)

// A progress claim of a Machine that declares its state as vars, `leadsTo(from = heatingOn, to =
// beeping, within = 81)` in its `progress` section, named after its `val`: from every state its
// environment reaches where `from` holds, every admitted path reaches a state where `to` holds
// within `within` steps. Its environment is a Scenario, every path from the Machine's starts unless
// it says one, `in only(heater.heat, heater.failover)`: nothing is fair unless the scenario says so.
// `from` and `to` read the vars, with the state they are asked of restored into them.
final class LeadsTo private[framework] (
    val name: String,
    val machine: Option[Model],
    val from: () => Boolean,
    val to: () => Boolean,
    val within: Int,
    val scenario: Scenario
):
  // The claim checked in the environment `s`.
  infix def in(s: Scenario): LeadsTo = LeadsTo(name, machine, from, to, within, s)

  // What is no scenario is refused where it is written: `boils in 3` would otherwise read as
  // membership.
  inline infix def in(@scala.annotation.unused other: Any): Nothing =
    scala.compiletime.error("in takes a scenario: only(...), fair(...), starts(...) or a named one")

  override def toString: String = s"progress $name"

object LeadsTo:
  @scala.annotation.publicInBinary
  private[framework] def declared(
      name: String,
      machine: Option[Model],
      from: () => Boolean,
      to: () => Boolean,
      within: Int,
      scenario: Scenario
  ): LeadsTo = LeadsTo(name, machine, from, to, within, scenario)

// `leadsTo(from = heatingOn, to = beeping, within = 81)`, in `progress`: from every state its
// environment reaches where `from` holds, every admitted path reaches one where `to` holds within
// `within` steps, a positive literal; named after its `val`. Its environment is every path from the
// Machine's starts unless it says one, `in only(heater.heat, heater.failover)`, and nothing is fair
// unless the scenario says so.
// It sits beside the named `Machine.leadsTo` it replaces, as overloads of one top-level name share a
// file.
inline def leadsTo(inline from: Boolean, inline to: Boolean, inline within: Int): LeadsTo =
  ${ QueryMacros.leadsTo('from, 'to, 'within) }
