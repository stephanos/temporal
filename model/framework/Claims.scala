package framework

// A Property's untyped part. A same-step Property names the action it is about under `when` and
// holds of the step that action produces; a transition Property holds of the state before a step
// and the step after it.
final class PropertyDecl private[framework] (
    val name: String,
    val machine: Model,
    // The action class, or under `whenAction` the action, the Property is about.
    private[framework] val when: Option[ClassRef | Composed | String],
    private[framework] val holds: Option[Nothing => Boolean],
    private[framework] val holds2: Option[(Nothing, Nothing) => Boolean]
)

// A Property over a machine whose state type is `S`. The claim patterns of model/framework/Syntax.scala
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
  infix def when(c: ClassRef): PropertyBuilder[S, O, F] = PropertyBuilder(name, m, Some(c))

  // Restricts the Property to every class of an action, including a composition's synchronized
  // step of that name.
  def whenAction(action: String): PropertyBuilder[S, O, F] = PropertyBuilder(name, m, Some(action))

  // Restricts a composition's Property to every class of one composed action: a sync,
  // `whenAction(c.synced(_.member -> action))`, or a member's own action,
  // `whenAction(c.own(_.member, action))`.
  def whenAction(action: Composed): PropertyBuilder[S, O, F] =
    PropertyBuilder(name, m, Some(action))

  // Finishes a same-step Property.
  infix def holds(f: Step[S, O, F] => Boolean): Property[S] =
    Property(PropertyDecl(name, m, when, Some(f), None))

  // Finishes a transition Property: `f` reads the state before a step and the step after.
  infix def holdsAcross(f: (S, Step[S, O, F]) => Boolean): Property[S] =
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
final class ScenarioBuilder[S] private[framework] (name: String, m: Model, start: Option[S]):
  // The state the Scenario starts in.
  infix def starts(s: S): ScenarioBuilder[S] = ScenarioBuilder(name, m, Some(s))

  // Pins the schedule to exactly these classes, in order: a machine's classes, or a composition's as
  // `c.synced(_.member -> action)` and `c.own(_.member, action)` select them.
  def actions(cs: (ClassRef | Composed)*): Scenario[S] =
    Scenario(ScenarioDecl(name, m, start, cs.toVector, free = false))

  // Admits any action at every step, within the Query's step limit.
  def free: Scenario[S] = Scenario(ScenarioDecl(name, m, start, Vector.empty, free = true))

// Bounds a Query: `steps` is the depth bound, `actions` the schedule length, and `search` the
// number of product states the search may visit before it reports limit-reached. Declared with
// named arguments and no `name`, the bounds are named after the `val` that declares them.
final case class Limits(name: String = "", steps: Int, actions: Int, search: Int)

enum QueryForm:
  // One trace of the Scenario on which the Property holds.
  case find

  // Whether the Property holds on every trace of the Scenario.
  case verify

// A bounded question about a machine: a Property, a Scenario and Limits.
final class Query private[framework] (
    val name: String,
    val form: QueryForm,
    val property: PropertyDecl,
    val scenario: ScenarioDecl,
    val limits: Limits,
    val expectedRun: Option[realize.RunExpectation] = None,
    val exploration: Option[realize.Exploration] = None,
    // The static combination count the author asserts, which `total(n)` sets.
    val total: Option[Long] = None
):
  def expect(expected: realize.RunExpectation): Query =
    Query(name, form, property, scenario, limits, Some(expected), exploration, total)

  def explore(space: realize.Exploration): Query =
    Query(name, form, property, scenario, limits, expectedRun, Some(space), total)

  // Asserts the Query's static combination count, which the lifter requires on every Query and Go
  // recomputes: the Scenario machine's states times the scheduled slots within the step limit when
  // the Scenario is pinned, or states times action classes times steps when it is free (see
  // model/SEMANTICS.md, "Query totals"). Written `... limits four total 48` or `(...).total(48)`.
  infix def total(n: Long): Query =
    Query(name, form, property, scenario, limits, expectedRun, exploration, Some(n))

  override def toString: String = name

// `val completion = query find completes in completed limits three`, named after its `val`, or
// `query("syncCompletion") find syncSucceeds in syncReplied limits two`. A Query neither names, such
// as an item of a list, is named `<machine>.<scenario>.<property>` after its Scenario's machine, its
// Scenario and its Property.
final class QueryDecl private[framework] (name: String):
  infix def find[P](p: Property[P]): QueryOn[P] = QueryOn(name, QueryForm.find, p)
  infix def verify[P](p: Property[P]): QueryOn[P] = QueryOn(name, QueryForm.verify, p)

final class QueryOn[P] private[framework] (name: String, form: QueryForm, p: Property[P]):
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
