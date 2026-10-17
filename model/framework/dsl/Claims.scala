package framework.dsl

import scala.language.dynamics

// A Property's untyped part. A same-step Property names the action it is about under `after` and
// holds of the step that action produces; a transition Property holds of the state before a step
// and the step after it.
final class PropertyDecl private[framework] (
    val name: String,
    val machine: Model,
    // The exact action class or whole action the Property is about.
    private[framework] val when: Option[ClassRef | Composed | String],
    private[framework] val holds: Option[Nothing => Boolean],
    private[framework] val holds2: Option[(Nothing, Nothing) => Boolean],
    val illustrations: Vector[Illustration[?, ?, ?]] = Vector.empty,
    // A member of a claim section written as a block of assertions, which `verify` and `find`
    // name, and whose native evaluation reruns its section (NativeClaims).
    private[framework] val claim: Option[NativeClaims.Ref] = None,
    // A progress claim, which `verify` names.
    private[framework] val leads: Option[LeadsTo] = None
)

// A Property over a machine or composition, whatever its state. The claim patterns of model/framework/dsl/Syntax.scala
// are Properties of their own kinds.
class Property private[framework] (val decl: PropertyDecl):
  def name: String = decl.name

final class ScopedClaim private[framework] (
    decl: PropertyDecl,
    private[framework] val scenario: Scenario
) extends Property(decl)

object ScopedClaim:
  @scala.annotation.publicInBinary
  private[framework] def apply(
      machine: Model,
      section: String,
      member: String,
      scenario: Scenario
  )(body: => Unit): ScopedClaim =
    if NativeClaims.current.nonEmpty then NativeClaims.boundary(body)
    new ScopedClaim(
      PropertyDecl(
        member,
        machine,
        None,
        None,
        None,
        claim = Some(NativeClaims.Ref(machine, section, member))
      ),
      scenario
    )

// Declares a Property on a machine whose steps are `Step[S, O, F]`.
final class PropertyBuilder[S, O, F] private[framework] (
    private[framework] val name: String,
    private[framework] val m: Declares[S] { type Outcome = O; type Fact = F },
    private[framework] val when: Option[ClassRef | Composed | String]
):
  // Restricts the Property to the step of one action class.
  infix def when(c: ClassRef | Composed): PropertyBuilder[S, O, F] =
    PropertyBuilder(name, m, Some(c))

  // Restricts the Property to every class of an action, including a composition's synchronized
  // step of that name.
  def whenAction(action: String): PropertyBuilder[S, O, F] =
    PropertyBuilder(name, m, Some(action))

  // Restricts a composition's Property to every class of one composed action: a sync,
  // `whenAction(c.synced(_.member -> action))`, or a member's own action,
  // `whenAction(c.own(_.member, action))`.
  def whenAction(action: Composed): PropertyBuilder[S, O, F] =
    PropertyBuilder(name, m, Some(action))

  // Finishes a same-step Property.
  @scala.annotation.publicInBinary
  private[framework] def sameStep(f: Step[S, O, F] => Boolean): SameStepProperty[S, O, F] =
    PredicateBindings.validate(f, m)
    new SameStepProperty[S, O, F](
      PropertyDecl(name, m, when, Some(restored(f)), None),
      () => m.illustrationMachine,
      Vector.empty
    )

  // Finishes a transition Property: `f` reads the state before a step and the step after.
  @scala.annotation.publicInBinary
  private[framework] def transition(f: (S, Step[S, O, F]) => Boolean): TransitionProperty[S, O, F] =
    PredicateBindings.validate(f, m)
    new TransitionProperty[S, O, F](
      PropertyDecl(name, m, when, None, Some(restored(f))),
      () => m.illustrationMachine,
      Vector.empty
    )

  // A predicate of a Machine's vars reads them with the state its step went to restored into them.
  private def restored(f: Step[S, O, F] => Boolean): Step[S, O, F] => Boolean =
    m.stateVars.fold(f)(vars =>
      vars
        .sameStep(
          f.asInstanceOf[Step[State, O, F] => Boolean]
        ) // scalafix:ok DisableSyntax.asInstanceOf
        .asInstanceOf[Step[S, O, F] => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
    )

  private def restored(f: (S, Step[S, O, F]) => Boolean): (S, Step[S, O, F]) => Boolean =
    m.stateVars.fold(f)(vars =>
      vars
        .transition(
          f.asInstanceOf[
            (State, Step[State, O, F]) => Boolean
          ] // scalafix:ok DisableSyntax.asInstanceOf
        )
        .asInstanceOf[(S, Step[S, O, F]) => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
    )

// A Scenario's untyped part: a named action schedule from one start, the path a Query runs. A
// pinned Scenario lists its actions exactly; a free one lists none and admits any action. The
// environment of a check is a Scenario too: `only` the actions it admits, `fair` the admitted
// actions that are weakly fair (one that stays enabled is eventually taken), and `updates` the
// vars it changes from the Machine's initial state, its start.
final class ScenarioDecl private[framework] (
    val name: String,
    val machine: Model,
    val start: Option[Any],
    // The declared classes, or for a composition its composed classes.
    val actions: Vector[ClassRef | Composed],
    val free: Boolean,
    val only: Option[Vector[Action[?] | Class | Composed]] = None,
    val fair: Vector[Action[?] | Class | Composed] = Vector.empty,
    val updates: Vector[(String, Any)] = Vector.empty
):
  require(
    only.isEmpty || actions.isEmpty,
    "a scenario admits actions with only or pins them with actions, not both"
  )

  // Whether a check of this scenario says more of the environment than a realization can drive:
  // that it admits only some actions, or assumes some are fair.
  def modelOnly: Boolean = only.nonEmpty || fair.nonEmpty

  // Whether it admits an action: every action, unless it says `only`.
  private[framework] def admits(decl: ActionDecl): Boolean =
    only.forall(_.exists:
      case a: Action[?] => a.decl eq decl
      case c: Class     => c.decl eq decl
      case _            => false)

  private[framework] def admits(cls: Class): Boolean =
    only.forall(_.exists:
      case a: Action[?] => a.decl eq cls.decl
      case c: Class     => c == cls
      case _            => false)

  // The actions it assumes weakly fair: every class of each one named, or the one class named.
  private[framework] def fairActions: Vector[ActionDecl | Class] = fair.collect:
    case a: Action[?] => a.decl
    case c: Class     => c

  private[framework] def copy(
      name: String = name,
      machine: Model = machine,
      start: Option[Any] = start,
      actions: Vector[ClassRef | Composed] = actions,
      free: Boolean = free,
      only: Option[Vector[Action[?] | Class | Composed]] = only,
      fair: Vector[Action[?] | Class | Composed] = fair,
      updates: Vector[(String, Any)] = updates
  ): ScenarioDecl = ScenarioDecl(name, machine, start, actions, free, only, fair, updates)

// The machine of a Scenario built where no machine is in scope, `only(user.pressButton)` in a test:
// the claim it is a scenario of names it.
private[framework] object Unbound extends Model:
  def name: String = ""

// A Scenario over a machine or composition, whatever its state. Built from `only(...)`,
// `fair(...)`, `starts(x = v)` and `actions(...)`, it composes by chaining them: `only(a, b).fair(a)`,
// `fair(a).starts(heatingOn = true)`. One that says none of them admits every path from the
// Machine's starts and assumes nothing.
final class Scenario private[framework] (val decl: ScenarioDecl):
  def name: String = decl.name

  // Admits these actions and no others.
  def only(actions: (Action[?] | Class | Composed)*): Scenario =
    require(decl.only.isEmpty, s"a scenario says only once: ${decl.name}")
    Scenario(decl.copy(only = Some(actions.toVector)))

  // Makes these admitted actions weakly fair: one that stays enabled is eventually taken.
  def fair(actions: (Action[?] | Class | Composed)*): Scenario =
    Scenario(decl.copy(fair = decl.fair ++ actions))

  // Starts in the Machine's initial state with the vars named changed: `.starts(heatingOn = true)`.
  def starts: Scenario.Starts = Scenario.Starts(this)

  // Pins the schedule to exactly these classes, in order.
  def actions(cs: (ClassRef | Composed)*): Scenario =
    Scenario(decl.copy(actions = cs.toVector, free = false))

  // The scenario of a check of `machine`'s claim: its own machine where it was built in one, and
  // its start the initial state with its updates, checked against the Machine's vars.
  private[framework] def bound(machine: Model, named: String): ScenarioDecl =
    val owner = if decl.machine eq Unbound then machine else decl.machine
    val start =
      if decl.updates.isEmpty then decl.start
      else
        owner match
          case declares: Declares[?] =>
            val vars = declares.stateVars.getOrElse(
              throw IllegalArgumentException(
                s"starts(${decl.updates.map(_._1).mkString(", ")}) names vars, and " +
                  s"${owner.name} declares none"
              )
            )
            Some(vars.updated(vars.initial, decl.updates))
          case other =>
            throw IllegalArgumentException(s"starts names vars, and ${other.name} declares none")
    decl.copy(
      name = if decl.name.isEmpty then named else decl.name,
      machine = owner,
      start = start
    )

object Scenario:
  // The named updates of a scenario's start, `starts(heatingOn = true)`.
  final class Starts private[framework] (scenario: Scenario) extends scala.Dynamic:
    def applyDynamicNamed(method: String)(updates: (String, Any)*): Scenario =
      require(method == "apply", s"starts.$method is not a start")
      Scenario(scenario.decl.copy(updates = scenario.decl.updates ++ updates))

    // `starts()`: the Machine's own starts, unchanged, as the scenario of a check that names
    // nothing else, such as a refined Machine's claim checked on the Machine that refines it.
    def applyDynamic(method: String)(): Scenario =
      require(method == "apply", s"starts.$method is not a start")
      scenario

  // A scenario that says nothing yet, of the machine it was built in, if any.
  @scala.annotation.publicInBinary
  private[framework] def of(machine: Option[Model], name: Option[String]): Scenario =
    Scenario(
      ScenarioDecl(name.getOrElse(""), machine.getOrElse(Unbound), None, Vector.empty, free = true)
    )

// Declares a Scenario.
final class ScenarioBuilder[S] private[framework] (
    name: String,
    m: Model,
    start: Option[S]
):
  // The state the Scenario starts in.
  infix def starts(s: S): ScenarioBuilder[S] = ScenarioBuilder(name, m, Some(s))

  // Pins the schedule to exactly these classes, in order: a machine's classes, or a composition's as
  // `c.synced(_.member -> action)` and `c.own(_.member, action)` select them.
  def actions(cs: (ClassRef | Composed)*): Scenario =
    Scenario(ScenarioDecl(name, m, start, cs.toVector, free = false))

  // Admits any action at every step, within the Query's step limit.
  def free: Scenario = Scenario(
    ScenarioDecl(name, m, start, Vector.empty, free = true)
  )

// Bounds a Query: `steps` is the depth bound, `actions` the schedule length, and `search` the
// number of product states the search may visit before it reports limit-reached. Declared with
// named arguments and no `name`, the bounds are named after the `val` that declares them.
final case class Limits private[framework] (
    name: String = "",
    steps: Int,
    actions: Int,
    search: Int
) extends RetainedDeclarationName:
  def toIR: io.temporal.server.api.umpire.v1.Limits = Signature.limits(this)
  def copy(
      name: String = this.name,
      steps: Int = this.steps,
      actions: Int = this.actions,
      search: Int = this.search
  ): Limits =
    val value = Limits(name, steps, actions, search)
    declarationName.copyTo(value.declarationName)
    value

enum QueryForm:
  // One trace of the Scenario on which the Property holds.
  case find

  // Whether the Property holds on every trace of the Scenario.
  case verify

private[framework] val frameworkReach: Limits =
  Limits("frameworkReach", steps = 256, actions = 0, search = 65536)

enum Checker:
  case umpire, apalache, tlc

final private[framework] case class QueryConfiguration(
    expectedRun: Option[framework.realize.RunExpectation] = None,
    exploration: Option[framework.realize.Exploration] = None,
    openReason: Option[String] = None,
    primary: Option[Checker] = None,
    confirmations: Vector[Checker] = Vector.empty
):
  def checker: Checker = primary.getOrElse(Checker.umpire)

  def opening(because: String): QueryConfiguration =
    require(openReason.isEmpty, "open may be assigned only once")
    require(because.trim.nonEmpty, "open requires a nonblank because reason")
    copy(openReason = Some(because))

  def selecting(checker: Checker): QueryConfiguration =
    require(confirmations.isEmpty, "checkedBy must be selected before confirm")
    require(primary.isEmpty, "checkedBy may be assigned only once")
    copy(primary = Some(checker))

  def confirming(checkers: Seq[Checker]): QueryConfiguration =
    require(confirmations.isEmpty, "confirm may be assigned only once")
    require(checkers.nonEmpty, "confirm requires a nonempty checker list")
    require(checkers.distinct.size == checkers.size, "confirm must not contain duplicate checkers")
    require(!checkers.contains(checker), "confirm must exclude the primary checker")
    copy(confirmations = checkers.toVector)

// A bounded question about a machine: a Property, a Scenario and Limits.
class Query private[framework] (
    val name: String,
    val form: QueryForm,
    val property: PropertyDecl,
    val scenario: ScenarioDecl,
    val limits: Limits,
    private[framework] val configuration: QueryConfiguration = QueryConfiguration()
):
  def expectedRun: Option[framework.realize.RunExpectation] = configuration.expectedRun
  def exploration: Option[framework.realize.Exploration] = configuration.exploration
  def openReason: Option[String] = configuration.openReason
  def checker: Checker = configuration.checker
  def confirmations: Vector[Checker] = configuration.confirmations

  def reach: Limits =
    val bounds = scenario.machine match
      case owner: Declares[?] => owner.reach
      case _ => throw IllegalArgumentException("Query reach requires a Scenario declaration owner")
    require(bounds.steps >= 0, "reach steps must be nonnegative")
    require(bounds.actions >= 0, "reach actions must be nonnegative")
    require(bounds.search > 0, "reach search must be positive")
    bounds

  private[framework] def total: Option[Long] = None

  private[framework] def configured(config: QueryConfiguration): Query =
    Query(name, form, property, scenario, limits, config)

  def open(because: String): Query = configured(configuration.opening(because))

  infix def checkedBy(checker: Checker): Query = configured(configuration.selecting(checker))

  def confirm(checkers: Checker*): Query = configured(configuration.confirming(checkers))

  // Marks a Query as one that generates a live Case, and states what its Run is expected to show
  // (fn-140 R3): a Query without it is checked in the model alone.
  def live(expected: framework.realize.RunExpectation): Query =
    realizable()
    configured(configuration.copy(expectedRun = Some(expected)))

  def explore(space: framework.realize.Exploration): Query =
    realizable()
    configured(configuration.copy(exploration = Some(space)))

  // A realization drives only what a Case can: a scenario that restricts or assumes the
  // environment is checked in the model alone.
  protected def realizable(): Unit =
    require(
      !scenario.modelOnly,
      s"${if name.nonEmpty then name else scenario.name} runs on the model only: its scenario " +
        "restricts or assumes the environment"
    )

  override def toString: String = name

// A check a Model names in `queries`, `verify(invariants.consistency) in anyVote` or
// `find(servedFromL2) in anyRun`, named after its `val`: a claim of the Machine, or a find target
// declared in `queries` itself, checked in a scenario, every path from the Machine's starts unless
// it says one. It names no bound: the checker's budget is the caller's.
final class ClaimQuery private[framework] (
    form: QueryForm,
    property: PropertyDecl,
    scenario: ScenarioDecl,
    config: QueryConfiguration = QueryConfiguration()
) extends Query("", form, property, scenario, frameworkReach, config):
  // The scenario the claim is checked in.
  infix def in(s: Scenario): ClaimQuery =
    ClaimQuery(form, property, s.bound(property.machine, ""), configuration)

  // What is no scenario is refused where it is written: `verify(x) in 3` would otherwise read as
  // membership.
  inline infix def in(@scala.annotation.unused other: Any): Nothing =
    scala.compiletime.error("in takes a scenario: only(...), fair(...), starts(...) or a named one")

  // A Model declares no Limits: a bound is a promise or the checker's budget.
  inline infix def limits(@scala.annotation.unused bounds: Any): Nothing =
    scala.compiletime.error(
      "a bound is a promise (within) or a budget (the checker's configuration); a Model declares " +
        "no Limits"
    )

  override private[framework] def configured(config: QueryConfiguration): Query =
    ClaimQuery(form, property, scenario, config)

object ClaimQuery:
  // A check of a Property, a claim of a Machine written in the retired `property` form.
  @scala.annotation.publicInBinary
  private[framework] def property(form: QueryForm, claim: Property): ClaimQuery =
    val scenario = claim match
      case scoped: ScopedClaim => scoped.scenario
      case _                   => Scenario.of(None, None)
    ClaimQuery(form, claim.decl, scenario.bound(claim.decl.machine, ""))

  // A check of a progress claim, in its own environment unless `in` names another.
  @scala.annotation.publicInBinary
  private[framework] def progress(form: QueryForm, claim: LeadsTo): ClaimQuery =
    require(
      form eq QueryForm.verify,
      s"${claim.name} is a progress claim, which is verified, not found"
    )
    val machine = claim.machine.getOrElse(Unbound)
    val decl = PropertyDecl(claim.name, machine, None, None, None, leads = Some(claim))
    ClaimQuery(form, decl, claim.scenario.bound(machine, claim.name))

  // A check of a member of a claim section written as a block of assertions.
  @scala.annotation.publicInBinary
  private[framework] def block(
      form: QueryForm,
      machine: Model,
      section: String,
      member: String
  ): ClaimQuery =
    val decl = PropertyDecl(
      member,
      machine,
      None,
      None,
      None,
      claim = Some(NativeClaims.Ref(machine, section, member))
    )
    ClaimQuery(form, decl, Scenario.of(None, None).bound(machine, ""))

// `val completion = query find completes in completed limits three`, named after its `val`, or
// `query("syncCompletion") find syncSucceeds in syncReplied limits two`. A Query neither names, such
// as an item of a list, is named `<machine>.<scenario>.<property>` after its Scenario's machine, its
// Scenario and its Property.
final class QueryDecl private[framework] (name: String):
  infix def find(p: Property): QueryOn =
    QueryOn(name, QueryForm.find, p)
  infix def verify(p: Property): QueryOn =
    QueryOn(name, QueryForm.verify, p)

final class QueryOn private[framework] (
    name: String,
    form: QueryForm,
    p: Property
):
  // The Scenario's machine is the Property's, or refines it: a Property of the refined machine is
  // read through the refinement the Scenario's machine declares. `RecordMachine[S, O, F]` does not carry
  // the type it refines, so the lifter, not the compiler, refuses a pair of unrelated machines.
  infix def in(s: Scenario): QueryIn = QueryIn(name, form, p.decl, s.decl)

final class QueryIn private[framework] (
    name: String,
    form: QueryForm,
    p: PropertyDecl,
    s: ScenarioDecl
):
  infix def limits(l: Limits): Query = Query(name, form, p, s, l)

private[framework] def query(name: String): QueryDecl = QueryDecl(name)

private[framework] def query: QueryDecl = QueryDecl("")

// The unfinished path claim: it becomes an ordinary find Query only when a fact is stated.
final class WitnessBuilder[S, O, F] private[framework] (
    private[framework] val owner: Declares[S],
    private[framework] val name: String,
    private[framework] val path: Vector[ClassRef | Composed]
):
  private[framework] def recording(
      recorded: Step[S, O, F] => Boolean
  ): Witness[S, O, F] =
    Witness(
      owner,
      name,
      path,
      recorded,
      Vector.empty,
      None,
      Limits(steps = path.size, actions = path.size, search = 512)
    )

// A completed path claim, usable at every Query root. Its ordinary core declarations carry the
// same name; as with property and scenario, a blank name is captured from the source val by lifting.
final class Witness[S, O, F] private[framework] (
    private[framework] val owner: Declares[S],
    named: String,
    private[framework] val path: Vector[ClassRef | Composed],
    private[framework] val recorded: Step[S, O, F] => Boolean,
    private[framework] val conditional: Vector[Step[S, O, F] => Boolean],
    private[framework] val ending: Option[S => Boolean],
    bounds: Limits,
    config: QueryConfiguration = QueryConfiguration()
) extends Query(
      named,
      QueryForm.find,
      PropertyDecl(
        named,
        owner,
        path.lastOption,
        Some((step: Step[S, O, F]) =>
          recorded(step) && conditional.forall(_(step)) && ending.forall(
            _(step.state)
          )
        ),
        None
      ),
      ScenarioDecl(named, owner, None, path, free = false),
      bounds,
      config
    ):
  private[framework] def recordingWhen(
      condition: S => Boolean,
      fact: Step[S, O, F] => Boolean
  ): Witness[S, O, F] =
    requiring(step => !condition(step.state) || fact(step))

  private[framework] def requiring(clause: Step[S, O, F] => Boolean): Witness[S, O, F] =
    copyWitness(conditional = conditional :+ clause)

  def ends(predicate: S => Boolean): Witness[S, O, F] =
    copyWitness(ending = Some(predicate))

  // The state the path ends in, of a Machine that declares its state as vars: `ends(agreed)`
  // reads the vars, with the state the path ended in restored into them.
  def ends(predicate: => Boolean)(using reads: S <:< State): Witness[S, O, F] =
    copyWitness(ending = Some(s => reads(s).owner.at(s)(predicate)))

  private[framework] def limits(bounds: Limits): Witness[S, O, F] =
    copyWitness(bounds = bounds)

  private def copyWitness(
      conditional: Vector[Step[S, O, F] => Boolean] = this.conditional,
      ending: Option[S => Boolean] = this.ending,
      bounds: Limits = limits,
      config: QueryConfiguration = configuration
  ): Witness[S, O, F] =
    Witness(
      owner,
      name,
      path,
      recorded,
      conditional,
      ending,
      bounds,
      config
    )

  override private[framework] def configured(config: QueryConfiguration): Witness[S, O, F] =
    copyWitness(config = config)

  override def open(because: String): Witness[S, O, F] =
    configured(configuration.opening(because))

  override infix def checkedBy(checker: Checker): Witness[S, O, F] =
    configured(configuration.selecting(checker))

  override def confirm(checkers: Checker*): Witness[S, O, F] =
    configured(configuration.confirming(checkers))

  override def live(expected: framework.realize.RunExpectation): Witness[S, O, F] =
    configured(configuration.copy(expectedRun = Some(expected)))

  override def explore(space: framework.realize.Exploration): Witness[S, O, F] =
    configured(configuration.copy(exploration = Some(space)))
