package umpire

/**
 * A Property's untyped part. A same-step Property names the action it is about under `when` and
 * holds of the step that action produces; a transition Property holds of the state before a step
 * and the step after it.
 */
final class PropertyDecl private[umpire] (
    val name: String,
    val machine: Model,
    /** The action class, or under `whenAction` the action, the Property is about. */
    private[umpire] val when: Option[ClassRef | String],
    private[umpire] val holds: Option[Nothing => Boolean],
    private[umpire] val holds2: Option[(Nothing, Nothing) => Boolean]
)

/**
 * A Property over a machine whose state type is `S`. The type parameter is what lets the compiler
 * reject a Query pairing a Property with a Scenario over another machine's states.
 */
final class Property[S] private[umpire] (val decl: PropertyDecl):
  def name: String = decl.name

/** Declares a Property on a machine whose steps are `Step[S, O, F]`. */
final class PropertyBuilder[S, O, F] private[umpire] (
    name: String,
    m: Model,
    when: Option[ClassRef | String]
):
  /** Restricts the Property to the step of one action class. */
  infix def when(c: ClassRef): PropertyBuilder[S, O, F] = PropertyBuilder(name, m, Some(c))

  /**
   * Restricts the Property to every class of an action, including a composition's synchronized
   * step of that name.
   */
  def whenAction(action: String): PropertyBuilder[S, O, F] = PropertyBuilder(name, m, Some(action))

  /** Finishes a same-step Property. */
  infix def holds(f: Step[S, O, F] => Boolean): Property[S] =
    Property(PropertyDecl(name, m, when, Some(f), None))

  /** Finishes a transition Property: `f` reads the state before a step and the step after. */
  infix def holdsAcross(f: (S, Step[S, O, F]) => Boolean): Property[S] =
    Property(PropertyDecl(name, m, when, None, Some(f)))

/**
 * A Scenario's untyped part: a named action schedule from one start, the path a Query runs. A
 * pinned Scenario lists its actions exactly; a free one lists none and admits any action.
 */
final class ScenarioDecl private[umpire] (
    val name: String,
    val machine: Model,
    val start: Option[Any],
    /** The declared classes, or for a composition the keys that name its members' classes. */
    val actions: Vector[ClassRef | String],
    val free: Boolean
)

/** A Scenario over a machine whose state type is `S`. */
final class Scenario[S] private[umpire] (val decl: ScenarioDecl):
  def name: String = decl.name

/** Declares a Scenario. */
final class ScenarioBuilder[S] private[umpire] (name: String, m: Model, start: Option[S]):
  /** The state the Scenario starts in. */
  infix def starts(s: S): ScenarioBuilder[S] = ScenarioBuilder(name, m, Some(s))

  /** Pins the schedule to exactly these classes, in order. */
  def actions(cs: ClassRef*): Scenario[S] =
    Scenario(ScenarioDecl(name, m, start, cs.toVector, free = false))

  /** Pins the schedule to these keys, for a composition whose keys name members. */
  def actionKeys(keys: String*): Scenario[S] =
    Scenario(ScenarioDecl(name, m, start, keys.toVector, free = false))

  /** Admits any action at every step, within the Query's step limit. */
  def free: Scenario[S] = Scenario(ScenarioDecl(name, m, start, Vector.empty, free = true))

extension [S, O, F](m: Machine[S, O, F])
  def property(name: String): PropertyBuilder[S, O, F] = PropertyBuilder(name, m, None)
  def scenario(name: String): ScenarioBuilder[S] = ScenarioBuilder(name, m, None)

extension [S <: Product](c: Composition[S])
  def property(name: String): PropertyBuilder[S, String, String] = PropertyBuilder(name, c, None)
  def scenario(name: String): ScenarioBuilder[S] = ScenarioBuilder(name, c, None)

/**
 * Bounds a Query: `steps` is the depth bound, `actions` the schedule length, and `search` the
 * number of product states the search may visit before it reports limit-reached.
 */
final case class Limits(name: String, steps: Int, actions: Int, search: Int)

enum QueryForm:
  /** One trace of the Scenario on which the Property holds. */
  case find

  /** Whether the Property holds on every trace of the Scenario. */
  case verify

/**
 * How a Scenario over `S` reads a Property over `P`: the identity when they are one machine, or a
 * declared refinement. A Query asks for one as a given, so a Property of an unrelated machine does
 * not type-check, and the refinement a Model declares is a value it names once.
 */
final class Reads[S, P] private[umpire] (private[umpire] val through: Option[Model])

object Reads:
  given identity[S]: Reads[S, S] = Reads(None)

  /**
   * The reading of a refining machine's steps as the refined machine's. That the machine declares
   * this refinement is checked over the IR, where the Query is answered.
   */
  def through[S, PS](
      m: Machine[S, ?, ?],
      @scala.annotation.unused product: Machine[PS, ?, ?]
  ): Reads[S, PS] =
    Reads(Some(m))

/** A bounded question about a machine: a Property, a Scenario and Limits. */
final class Query private[umpire] (
    val name: String,
    val form: QueryForm,
    val property: PropertyDecl,
    val scenario: ScenarioDecl,
    val limits: Limits,
    private[umpire] val reads: Option[Model],
    val expectedRun: Option[realize.RunExpectation] = None,
    val exploration: Option[realize.Exploration] = None
):
  def expect(expected: realize.RunExpectation): Query =
    Query(name, form, property, scenario, limits, reads, Some(expected), exploration)

  def explore(space: realize.Exploration): Query =
    Query(name, form, property, scenario, limits, reads, expectedRun, Some(space))

  override def toString: String = name

/** `query("syncCompletion") find syncSucceeds in syncReplied limits two`. */
final class QueryDecl private[umpire] (name: String):
  infix def find[P](p: Property[P]): QueryOn[P] = QueryOn(name, QueryForm.find, p)
  infix def verify[P](p: Property[P]): QueryOn[P] = QueryOn(name, QueryForm.verify, p)

final class QueryOn[P] private[umpire] (name: String, form: QueryForm, p: Property[P]):
  /** The Scenario's machine is the Property's, or refines it through a declared `Reads`. */
  infix def in[S](s: Scenario[S])(using via: Reads[S, P]): QueryIn =
    QueryIn(name, form, p.decl, s.decl, via.through)

final class QueryIn private[umpire] (
    name: String,
    form: QueryForm,
    p: PropertyDecl,
    s: ScenarioDecl,
    reads: Option[Model]
):
  infix def limits(l: Limits): Query = Query(name, form, p, s, l, reads)

def query(name: String): QueryDecl = QueryDecl(name)
