package umpire

import scala.collection.mutable

/**
 * The Property and Scenario names declared on one machine. Two declarations of a kind under one
 * name share a Definition ID, so every Case, fingerprint and Contract naming one would silently
 * name the other; `Check` reports the second.
 */
final private[umpire] class ClaimNames:
  private val seen = mutable.Set.empty[String]
  private val twice = mutable.ArrayBuffer.empty[String]
  def declare(kind: String, name: String): Unit = synchronized {
    if !seen.add(s"$kind $name") then twice += s"$kind $name"
  }
  def duplicates(machine: String): List[ModelError] = synchronized {
    twice.toList.map(k =>
      ModelError(
        s"machine $machine",
        s"$k is declared twice, and both declarations would share one Definition ID; rename one"
      )
    )
  }

/**
 * A Property's untyped part, which Queries and the search read. A same-step Property names the
 * action it is about under `when` and holds of the step that action produces; a transition
 * Property holds of the state before a step and the step after it.
 */
final class PropertyDecl private[umpire] (
    val name: String,
    val machine: Model,
    private[umpire] val when: Option[String => Boolean],
    private[umpire] val whenLabel: String,
    private[umpire] val holds: Option[Any => Boolean],
    private[umpire] val holds2: Option[(Any, Any) => Boolean]
):
  def isTransition: Boolean = holds2.isDefined

  /** Whether a same-step Property is about the step of this action class. */
  def triggers(action: String): Boolean = when.forall(_(action))

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
    when: Option[String => Boolean],
    label: String
):
  /** Restricts the Property to the step of one action class. */
  infix def when(c: ClassRef): PropertyBuilder[S, O, F] =
    val key = ClassRef.resolve(c).key
    PropertyBuilder(name, m, Some(_ == key), key)

  /**
   * Restricts the Property to every class of an action, including a composition's synchronized
   * step of that name.
   */
  def whenAction(action: String): PropertyBuilder[S, O, F] =
    PropertyBuilder(name, m, Some(a => Keys.actionName(a) == action), action)

  /** Finishes a same-step Property. */
  infix def holds(f: Step[S, O, F] => Boolean): Property[S] =
    Property(
      PropertyDecl(name, m, when, label, Some(s => f(s.asInstanceOf[Step[S, O, F]])), None)
    ) // scalafix:ok DisableSyntax.asInstanceOf

  /** Finishes a transition Property: `f` reads the state before a step and the step after. */
  infix def holdsAcross(f: (S, Step[S, O, F]) => Boolean): Property[S] =
    Property(
      PropertyDecl(
        name,
        m,
        when,
        label,
        None,
        Some((b, a) => f(b.asInstanceOf[S], a.asInstanceOf[Step[S, O, F]]))
      )
    ) // scalafix:ok DisableSyntax.asInstanceOf

/**
 * A Scenario's untyped part: a named action schedule from one start, the path a Query runs. A
 * pinned Scenario lists its actions exactly; a free one lists none and admits any action.
 */
final class ScenarioDecl private[umpire] (
    val name: String,
    val machine: Model,
    val start: String,
    val actions: Vector[String],
    /** The declared classes, for a realization that places them. */
    val classes: Vector[Class],
    val free: Boolean
)

/** A Scenario over a machine whose state type is `S`. */
final class Scenario[S] private[umpire] (val decl: ScenarioDecl):
  def name: String = decl.name
  def start: String = decl.start
  def actions: Vector[String] = decl.actions

/** Declares a Scenario. */
final class ScenarioBuilder[S] private[umpire] (
    name: String,
    m: Model,
    key: S => String,
    start: String
):
  /** The state the Scenario starts in. */
  infix def starts(s: S): ScenarioBuilder[S] = ScenarioBuilder(name, m, key, key(s))

  /** Pins the schedule to exactly these classes, in order. */
  def actions(cs: ClassRef*): Scenario[S] =
    val classes = cs.toVector.map(ClassRef.resolve)
    Scenario(ScenarioDecl(name, m, start, classes.map(_.key), classes, free = false))

  /** Pins the schedule to these keys, for a composition whose keys name members. */
  def actionKeys(keys: String*): Scenario[S] = Scenario(
    ScenarioDecl(name, m, start, keys.toVector, Vector.empty, free = false)
  )

  /** Admits any action at every step, within the Query's step limit. */
  def free: Scenario[S] = Scenario(
    ScenarioDecl(name, m, start, Vector.empty, Vector.empty, free = true)
  )

extension [S, O, F](m: Machine[S, O, F])
  def property(name: String): PropertyBuilder[S, O, F] =
    m.names.declare("property", name)
    PropertyBuilder(name, m, None, "")
  def scenario(name: String): ScenarioBuilder[S] =
    m.names.declare("scenario", name)
    ScenarioBuilder(name, m, Keys.of, "")

extension [S <: Product](c: Composition[S])
  def property(name: String): PropertyBuilder[S, String, String] =
    c.names.declare("property", name)
    PropertyBuilder(name, c, None, "")
  def scenario(name: String): ScenarioBuilder[S] =
    c.names.declare("scenario", name)
    ScenarioBuilder(name, c, c.stateKey, "")

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
final class Reads[S, P] private[umpire] (
    private[umpire] val refinement: Option[() => Checked[Refinement]]
)

object Reads:
  given identity[S]: Reads[S, S] = Reads(None)

  /**
   * The reading of a refining machine's steps as the refined machine's. That the machine declares
   * this refinement is checked when a Query runs.
   */
  def through[S, PS](
      m: Machine[S, ?, ?],
      @scala.annotation.unused product: Machine[PS, ?, ?]
  ): Reads[S, PS] =
    Reads(Some(() => m.refinementCheck))

/** A bounded question about a machine: a Property, a Scenario and Limits. */
final class Query private[umpire] (
    val name: String,
    val form: QueryForm,
    val property: PropertyDecl,
    val scenario: ScenarioDecl,
    val limits: Limits,
    private[umpire] val refinement: Option[() => Checked[Refinement]],
    val expectedRun: Option[realize.RunExpectation] = None,
    val exploration: Option[realize.Exploration] = None
):
  def expect(expected: realize.RunExpectation): Query =
    Query(name, form, property, scenario, limits, refinement, Some(expected), exploration)

  def explore(space: realize.Exploration): Query =
    Query(name, form, property, scenario, limits, refinement, expectedRun, Some(space))

  private[umpire] def decl: String = s"query $name"
  lazy val answer: Checked[Answer] = Search.answer(this)
  override def toString: String = name

/** `query("syncCompletion") find syncSucceeds in syncReplied limits two`. */
final class QueryDecl private[umpire] (name: String):
  infix def find[P](p: Property[P]): QueryOn[P] = QueryOn(name, QueryForm.find, p)
  infix def verify[P](p: Property[P]): QueryOn[P] = QueryOn(name, QueryForm.verify, p)

final class QueryOn[P] private[umpire] (name: String, form: QueryForm, p: Property[P]):
  /** The Scenario's machine is the Property's, or refines it through a declared `Reads`. */
  infix def in[S](s: Scenario[S])(using via: Reads[S, P]): QueryIn =
    QueryIn(name, form, p.decl, s.decl, via.refinement)

final class QueryIn private[umpire] (
    name: String,
    form: QueryForm,
    p: PropertyDecl,
    s: ScenarioDecl,
    refinement: Option[() => Checked[Refinement]]
):
  infix def limits(l: Limits): Query = Query(name, form, p, s, l, refinement)

def query(name: String): QueryDecl = QueryDecl(name)

/** A Query's answer, spelled as Lean spells `PlanningOutcome.name`. */
enum Verdict(val spelling: String):
  case found extends Verdict("found")
  case notFound extends Verdict("not-found")
  case verifiedWithinLimits extends Verdict("verified-within-limits")
  case counterexampleFound extends Verdict("counterexample-found")
  case limitReached extends Verdict("limit-reached")

/** One step of a witness. */
final case class TraceStep(action: Atom, outcome: Atom, state: Atom, facts: Vector[Atom])

/** A witness: the start and the steps taken. */
final case class Trace(initial: Atom, steps: Vector[TraceStep])

/**
 * A Query's outcome, its witness or counterexample, and how much the search explored. `exercised`
 * reports that the Property's clause fired on some explored step, so a verified answer was earned
 * rather than vacuous.
 */
final case class Answer(
    outcome: Verdict,
    witness: Option[Trace] = None,
    explored: Int = 0,
    explanation: String = "",
    /** The row keys the witness takes, in order. */
    rows: Vector[String] = Vector.empty,
    exercised: Boolean = false
):
  override def toString: String =
    s"${outcome.spelling} after $explored product states${
        if explanation.nonEmpty then s": $explanation" else ""
      }"
