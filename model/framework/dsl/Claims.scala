package framework.dsl

// A Property's untyped part. A same-step Property names the action it is about under `after` and
// holds of the step that action produces; a transition Property holds of the state before a step
// and the step after it.
final class PropertyDecl private[framework] (
    val name: String,
    val machine: Model,
    // The exact action class or whole action the Property is about.
    private[framework] val when: Option[ClassRef | Composed | String],
    private[framework] val holds: Option[Nothing => Boolean],
    private[framework] val holds2: Option[(Nothing, Nothing) => Boolean]
)

// A Property over a machine whose state type is `S`. The claim patterns of model/framework/dsl/Syntax.scala
// are Properties of their own kinds.
class Property[S] private[framework] (val decl: PropertyDecl):
  def name: String = decl.name

// Declares a Property on a machine whose steps are `Step[S, O, F]`.
final class PropertyBuilder[S, O, F] private[framework] (
    private[framework] val name: String,
    private[framework] val m: Model,
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
  private[framework] def sameStep(f: Step[S, O, F] => Boolean): Property[S] =
    PredicateBindings.validate(f, m)
    Property(PropertyDecl(name, m, when, Some(f), None))

  // Finishes a transition Property: `f` reads the state before a step and the step after.
  @scala.annotation.publicInBinary
  private[framework] def transition(f: (S, Step[S, O, F]) => Boolean): Property[S] =
    PredicateBindings.validate(f, m)
    Property(PropertyDecl(name, m, when, None, Some(f)))

// A Scenario's untyped part: a named action schedule from one start, the path a Query runs. A
// pinned Scenario lists its actions exactly; a free one lists none and admits any action.
final class ScenarioDecl private[framework] (
    val name: String,
    val machine: Model,
    val start: Option[Any],
    // The declared classes, or for a composition its composed classes.
    val actions: Vector[ClassRef | Composed],
    val free: Boolean
)

// A Scenario over a machine whose state type is `S`.
final class Scenario[S] private[framework] (val decl: ScenarioDecl):
  def name: String = decl.name

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
  def actions(cs: (ClassRef | Composed)*): Scenario[S] =
    Scenario(ScenarioDecl(name, m, start, cs.toVector, free = false))

  // Admits any action at every step, within the Query's step limit.
  def free: Scenario[S] = Scenario(
    ScenarioDecl(name, m, start, Vector.empty, free = true)
  )

// Bounds a Query: `steps` is the depth bound, `actions` the schedule length, and `search` the
// number of product states the search may visit before it reports limit-reached. Declared with
// named arguments and no `name`, the bounds are named after the `val` that declares them.
final case class Limits(
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

val frameworkReach: Limits = Limits("frameworkReach", steps = 256, actions = 0, search = 65536)

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
    configured(configuration.copy(expectedRun = Some(expected)))

  def explore(space: framework.realize.Exploration): Query =
    configured(configuration.copy(exploration = Some(space)))

  override def toString: String = name

// `val completion = query find completes in completed limits three`, named after its `val`, or
// `query("syncCompletion") find syncSucceeds in syncReplied limits two`. A Query neither names, such
// as an item of a list, is named `<machine>.<scenario>.<property>` after its Scenario's machine, its
// Scenario and its Property.
final class QueryDecl private[framework] (name: String):
  infix def find[P](p: Property[P]): QueryOn[P] =
    QueryOn(name, QueryForm.find, p)
  infix def verify[P](p: Property[P]): QueryOn[P] =
    QueryOn(name, QueryForm.verify, p)

final class QueryOn[P] private[framework] (
    name: String,
    form: QueryForm,
    p: Property[P]
):
  // The Scenario's machine is the Property's, or refines it: a Property of the refined machine is
  // read through the refinement the Scenario's machine declares. `Machine[S, O, F]` does not carry
  // the type it refines, so the lifter, not the compiler, refuses a pair of unrelated machines.
  infix def in[S](s: Scenario[S]): QueryIn = QueryIn(name, form, p.decl, s.decl)

final class QueryIn private[framework] (
    name: String,
    form: QueryForm,
    p: PropertyDecl,
    s: ScenarioDecl
):
  infix def limits(l: Limits): Query = Query(name, form, p, s, l)

def query(name: String): QueryDecl = QueryDecl(name)

def query: QueryDecl = QueryDecl("")

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

  def limits(bounds: Limits): Witness[S, O, F] =
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
