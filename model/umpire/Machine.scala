package umpire

import scala.collection.mutable

/**
 * One result of an action: the outcome, the next state and the facts it records. `because` is an
 * optional explanation a generated table view shows beside the row; no fingerprint reads it.
 */
final case class Step[S, O, F](outcome: O, state: S, facts: List[F] = Nil, because: String = "")

/**
 * `steps.because("...")`: the explanation of each step written here, as `Step`'s `because` gives it.
 * The lifter reads it only on a list of steps the step function writes out.
 */
extension [S, O, F](steps: List[Step[S, O, F]])
  def because(reason: String): List[Step[S, O, F]] = steps.map(_.copy(because = reason))

/**
 * The name of one alternative of a step that can go more than one way, declared once by the val that
 * names it, `val committed = choice`; the lifter names it after that val. A token is compared by
 * identity, so two tokens are two names, as two input tokens are two inputs.
 */
final class Choice private[umpire] ()

/** A new choice token: `val committed = choice`. */
def choice: Choice = Choice()

/**
 * `choose(committed -> accept(...), redelivered -> stay(s))`: the results of a step that can go more
 * than one way, in the order written, each named by its token. The names are metadata no check reads
 * (model/SEMANTICS.md, Named choices), so the results are the ones the same steps give written as an
 * unnamed list. Each alternative is one step written out, and each token names one of them; a choose
 * of one alternative does not compile, since one result is no choice.
 */
def choose[S, O, F](
    first: (Choice, List[Step[S, O, F]]),
    second: (Choice, List[Step[S, O, F]]),
    rest: (Choice, List[Step[S, O, F]])*
): List[Step[S, O, F]] =
  val alternatives = first +: second +: rest
  for (token, steps) <- alternatives do
    require(alternatives.count(_._1 eq token) == 1, "a choice names one alternative of a choose")
    require(steps.sizeIs == 1, "each alternative of a choose is one step")
  alternatives.flatMap(_._2).toList

/** A declared, derived or composed machine. */
trait Model:
  def name: String

/**
 * What a claim is declared on, by its state type `S`: a machine or a composition. A claim written once
 * over `Declares[S]`, such as a function over `m: Declares[S]` and the predicates it reads, declares
 * the same Property on either. `Outcome` and `Fact` are the types a step of it answers and records:
 * the machine's own, or for a composition the strings of its composed keys.
 */
trait Declares[S] extends Model:
  type Outcome
  type Fact

  /**
   * A Property or a Scenario is named after the `val` that declares it (`val completes =
   * activityProduct.property holds ...`), or by the name it is given where it is declared without
   * one, such as in a list or inside a function over a machine. A Scenario that names no start starts
   * in its machine's one declared start, or for a composition in its members' starts.
   */
  def property: PropertyBuilder[S, Outcome, Fact] = PropertyBuilder("", this, None)
  def property(name: String): PropertyBuilder[S, Outcome, Fact] = PropertyBuilder(name, this, None)
  def scenario: ScenarioBuilder[S] = ScenarioBuilder("", this, None)
  def scenario(name: String): ScenarioBuilder[S] = ScenarioBuilder(name, this, None)

/** An action bound to its step function. */
final case class StepBinding[S, O, F](decl: ActionDecl, function: AnyRef)

/**
 * `action ~> stepFunction`. One extension per arity, each typed by the action's inputs, so a step
 * function written for another action's inputs does not compile.
 */
extension (a: Action[EmptyTuple])
  infix def ~>[S, O, F](f: S => List[Step[S, O, F]]): StepBinding[S, O, F] = StepBinding(a.decl, f)

extension [A](a: Action[A *: EmptyTuple])
  infix def ~>[S, O, F](f: (S, A) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

extension [A, B](a: Action[(A, B)])
  infix def ~>[S, O, F](f: (S, A, B) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

extension [A, B, C](a: Action[(A, B, C)])
  infix def ~>[S, O, F](f: (S, A, B, C) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

/**
 * The machine-declaration scope. Inside `machine(...) { ... }` the entry points below are bare
 * calls that resolve against the scope in context: there is no builder to thread.
 */
final class MachineScope[S, O, F] private[umpire] ():
  private[umpire] var entity: Option[Entity] = None // scalafix:ok DisableSyntax.var
  private[umpire] var starts: List[S] = Nil // scalafix:ok DisableSyntax.var
  private[umpire] var ends: S => Boolean = _ => false // scalafix:ok DisableSyntax.var
  private[umpire] var evidence: Option[F => String] = None // scalafix:ok DisableSyntax.var
  private[umpire] val unobservable = mutable.LinkedHashSet.empty[String]
  private[umpire] val bindings = mutable.ArrayBuffer.empty[StepBinding[S, O, F]]
  private[umpire] var refinement: Option[RefinementDecl[S]] = None // scalafix:ok DisableSyntax.var
  private[umpire] var visible: Option[F => Boolean] = None // scalafix:ok DisableSyntax.var
  private[umpire] var visibleOutcomes: Option[O => Boolean] = None // scalafix:ok DisableSyntax.var
  private[umpire] val monitors = mutable.ArrayBuffer.empty[Monitor[S, O, F, ?]]
  private[umpire] val assumptions = mutable.ArrayBuffer.empty[Assumption]

/**
 * Declares a machine: a transition relation over the finite state type `S` with outcomes `O` and
 * facts `F`, one step function per action, the states it starts in, the states it may end in and
 * the evidence that confirms each fact.
 */
def machine[S, O, F](family: Family, name: String)(body: MachineScope[S, O, F] ?=> Unit)(using
    Finite[S],
    Finite[O],
    Finite[F]
): Machine[S, O, F] = declare(family, name, body)

/**
 * Declares a machine named after the `val` that declares it, in the `given Family`. Its three types
 * are stated once, here or as the `val`'s type: `val m = machine[S, O, F] { ... }`.
 */
def machine[S, O, F](body: MachineScope[S, O, F] ?=> Unit)(using
    family: Family,
    fs: Finite[S],
    fo: Finite[O],
    ff: Finite[F]
): Machine[S, O, F] = declare(family, "", body)

private def declare[S, O, F](family: Family, name: String, body: MachineScope[S, O, F] ?=> Unit)(
    using
    Finite[S],
    Finite[O],
    Finite[F]
): Machine[S, O, F] =
  val scope = MachineScope[S, O, F]()
  body(using scope)
  Machine(
    family,
    name,
    scope.entity,
    scope.starts,
    scope.ends,
    scope.evidence,
    scope.unobservable.toSet,
    scope.bindings.toList,
    scope.refinement,
    scope.visible,
    scope.visibleOutcomes,
    scope.monitors.toList,
    scope.assumptions.toList
  )

/** Names the entity the machine keeps state for. */
def forEntity(e: Entity)(using m: MachineScope[?, ?, ?]): Unit = m.entity = Some(e)

/** The states the machine starts in. */
def starts[S](using m: MachineScope[S, ?, ?])(states: S*): Unit = m.starts = states.toList

/** Which states the machine may end in. */
def ends[S](using m: MachineScope[S, ?, ?])(end: S => Boolean): Unit = m.ends = end

/** Timers whose step records nothing a Run can read. */
def unobservable(timers: Action[EmptyTuple]*)(using m: MachineScope[?, ?, ?]): Unit =
  m.unobservable ++= timers.map(_.name)

/**
 * `evidence:` as a total function from facts to the recorded event or observation that confirms
 * them, such as a named function over every fact.
 */
def evidence[F](using m: MachineScope[?, ?, F])(lines: F => String): Unit = m.evidence = Some(lines)

/**
 * `evidence { case ... }` lists only the facts confirmed by something other than evidence of their
 * own name. A fact no line names, and every fact of a machine that declares no evidence, is
 * confirmed by the evidence named after it; a fact with fields needs a line of its own, which the
 * lifter refuses to leave out.
 */
def evidence[F](using m: MachineScope[?, ?, F])(exceptions: PartialFunction[F, String]): Unit =
  m.evidence = Some(exceptions)

/** The step functions, one per action. */
def steps[S, O, F](using m: MachineScope[S, O, F])(bindings: StepBinding[S, O, F]*): Unit =
  m.bindings ++= bindings

/**
 * The facts the refined machine sees. A step that reads as a stutter of it records none of them,
 * and a step it carries records only the ones its carrying step records.
 */
def visible[F](using m: MachineScope[?, ?, F])(sees: F => Boolean): Unit = m.visible = Some(sees)

/** The outcomes the refined machine sees. A step that reads as a stutter of it answers none of them. */
def visibleOutcomes[O](using m: MachineScope[?, O, ?])(sees: O => Boolean): Unit =
  m.visibleOutcomes = Some(sees)

/** A machine: its declaration, as the scope above recorded it. */
final class Machine[S, O, F] private[umpire] (
    val family: Family,
    val name: String,
    val entity: Option[Entity],
    private[umpire] val startStates: List[S],
    private[umpire] val isEnd: S => Boolean,
    private[umpire] val evidenceOf: Option[F => String],
    private[umpire] val unobservableNames: Set[String],
    private[umpire] val bindings: List[StepBinding[S, O, F]],
    private[umpire] val refinement: Option[RefinementDecl[S]],
    private[umpire] val visibleFacts: Option[F => Boolean],
    private[umpire] val visibleOutcomeSet: Option[O => Boolean],
    private[umpire] val monitorList: List[Monitor[S, O, F, ?]],
    private[umpire] val assumptions: List[Assumption]
)(using
    private[umpire] val fs: Finite[S],
    private[umpire] val fo: Finite[O],
    private[umpire] val ff: Finite[F]
) extends Declares[S]:
  type Outcome = O
  type Fact = F

  /**
   * A machine that keeps the rows of the named actions and drops the rest, named after the `val`
   * that declares it, in the `given Family`. It keeps the state type, starts and ends, owns its own
   * name and Definition IDs, and does not inherit a refinement.
   */
  def restrict(keep: Action[?]*)(using family: Family): Machine[S, O, F] =
    val decls = keep.map(_.decl).toSet
    // It keeps its source's monitors and assumptions, which are about the state and the machine.
    derived(
      family,
      "",
      unobservableNames = Set.empty,
      bindings = bindings.filter(b => decls(b.decl)),
      refinement = None,
      visibleFacts = None,
      visibleOutcomeSet = None
    )

  /**
   * A machine that binds other step functions to actions this one binds, each in its place. It keeps
   * everything else this machine declares, and is named after the `val` that declares it, in the
   * `given Family`. The lifter refuses an action this machine does not bind.
   */
  def rebind(replaced: StepBinding[S, O, F]*)(using family: Family): Machine[S, O, F] =
    val by = replaced.map(b => b.decl -> b).toMap
    derived(family, bindings = bindings.map(b => by.getOrElse(b.decl, b)))

  /**
   * A machine that also binds actions this one does not, after its own bindings. The lifter refuses
   * an action this machine binds already.
   */
  def extend(added: StepBinding[S, O, F]*)(using family: Family): Machine[S, O, F] =
    derived(family, bindings = bindings ++ added)

  /**
   * A machine that refines `product` through `map` in place of the refinement this one declares,
   * and keeps the facts and outcomes that refinement lets the refined machine see.
   */
  def refining[PS, PO, PF](product: Machine[PS, PO, PF])(map: S => PS)(using
      family: Family
  ): Machine[S, O, F] =
    derived(family, refinement = Some(RefinementDecl[S](product, map)))

  /** A machine whose checks also make the assumptions named, each once, after this one's. */
  def assuming(added: Assumption*)(using family: Family): Machine[S, O, F] =
    derived(family, assumptions = assumptions ++ added)

  /**
   * A machine without this one's monitors and refinement: the same transitions, for a composition
   * a member's monitors may not watch (model/SEMANTICS.md leaves that undefined).
   */
  def unmonitored(using family: Family): Machine[S, O, F] =
    derived(
      family,
      refinement = None,
      visibleFacts = None,
      visibleOutcomeSet = None,
      monitorList = Nil
    )

  // A derived machine takes its name from its val unless it is given one, as `machine { ... }` does.
  private def derived(
      family: Family,
      name: String = "",
      unobservableNames: Set[String] = unobservableNames,
      bindings: List[StepBinding[S, O, F]] = bindings,
      refinement: Option[RefinementDecl[S]] = refinement,
      visibleFacts: Option[F => Boolean] = visibleFacts,
      visibleOutcomeSet: Option[O => Boolean] = visibleOutcomeSet,
      monitorList: List[Monitor[S, O, F, ?]] = monitorList,
      assumptions: List[Assumption] = assumptions
  ): Machine[S, O, F] =
    Machine(
      family,
      name,
      entity,
      startStates,
      isEnd,
      evidenceOf,
      unobservableNames,
      bindings,
      refinement,
      visibleFacts,
      visibleOutcomeSet,
      monitorList,
      assumptions
    )
