package umpire

/* The framework surface the Models author against.
 *
 * A Model is ordinary Scala 3: enums for domains, case classes for states, plain `def`s for step
 * functions. The DSL is a thin layer of context functions (`Scope ?=> Unit` blocks), `infix`
 * extension methods and a few `inline` declarations that reject a bad declaration while the file
 * compiles. Bodies below are sketched where a comment says so; signatures are what the Models
 * bind to.
 */

import scala.compiletime.{constValue, erasedValue, error, summonFrom, summonInline}
import scala.deriving.Mirror
import scala.quoted.*
import scala.util.TupledFunction

// ---------------------------------------------------------------------------------------------
// Step
// ---------------------------------------------------------------------------------------------

/** One row a step function produces. A step function `S => inputs => List[Step[S, O, F]]` returns
  * `Nil` when the action is not enabled in `state`. */
final case class Step[S, O, F](outcome: O, state: S, facts: List[F]) derives CanEqual

// ---------------------------------------------------------------------------------------------
// Finite: the state table is enumerated, not sampled
// ---------------------------------------------------------------------------------------------

/** A type whose values can all be listed. Derived structurally through `Mirror`: an enum is the
  * concatenation of its cases, a case class (or a parametrised enum case) the cartesian product of
  * its fields, in declaration order, so `Finite[S].values` is the machine's canonical state order. */
trait Finite[T]:
  def values: IndexedSeq[T]

object Finite:
  inline def size[T](using f: Finite[T]): Int = f.values.size

  given Finite[Boolean] with
    def values = IndexedSeq(false, true)

  given Finite[Unit] with
    def values = IndexedSeq(())

  given Finite[EmptyTuple] with
    def values = IndexedSeq(EmptyTuple)

  given [H, T <: Tuple](using h: Finite[H], t: Finite[T]): Finite[H *: T] with
    def values = for x <- h.values; xs <- t.values yield x *: xs

  /** `derives Finite` lands here. A field whose type has no `Finite` fails with the field's name
    * rather than with a bare "no given instance" from inside the derivation. */
  inline def derived[T](using m: Mirror.Of[T]): Finite[T] =
    inline m match
      case s: Mirror.SumOf[T] =>
        val cases = summonAll[s.MirroredElemTypes, s.MirroredLabel]
        new Finite[T]:
          def values = cases.flatMap(_.values).asInstanceOf[IndexedSeq[T]]
      case p: Mirror.ProductOf[T] =>
        val fields = summonInline[Finite[p.MirroredElemTypes]]
        new Finite[T]:
          def values = fields.values.map(p.fromProduct)

  private inline def summonAll[Ts <: Tuple, L]: List[Finite[?]] =
    inline erasedValue[Ts] match
      case _: EmptyTuple => Nil
      case _: (h *: t) =>
        summonFrom {
          case f: Finite[`h`] => f
          // A singleton case or a parametrised case has no given of its own; derive it in place.
          case m: Mirror.Of[`h`] => derived[h](using m)
          case _ => error("case " + constValue[L] + " carries a field that is not Finite")
        } :: summonAll[t, L]

  /** The count of `T`'s values as a compile-time constant, for pins written as `inline` checks.
    * Every branch is an `inline match` over erased types, so the result folds to a literal. */
  transparent inline def sizeOf[T](using m: Mirror.Of[T]): Int =
    inline m match
      case s: Mirror.SumOf[T] => sumSizes[s.MirroredElemTypes]
      case p: Mirror.ProductOf[T] => productSizes[p.MirroredElemTypes]

  private transparent inline def sumSizes[Ts <: Tuple]: Int = inline erasedValue[Ts] match
    case _: EmptyTuple => 0
    case _: (h *: t) => sizeOfOne[h] + sumSizes[t]

  private transparent inline def productSizes[Ts <: Tuple]: Int = inline erasedValue[Ts] match
    case _: EmptyTuple => 1
    case _: (h *: t) => sizeOfOne[h] * productSizes[t]

  private transparent inline def sizeOfOne[T]: Int = inline erasedValue[T] match
    case _: Boolean => 2
    case _: Bounded[n] => constValue[n] + 1
    case _ => sizeOf[T](using summonInline[Mirror.Of[T]])

/** `0..N`, the Scala reading of `Fin (N + 1)`. Values are checked where they are written. */
opaque type Bounded[N <: Int] = Int

object Bounded:
  inline def apply[N <: Int](inline i: Int): Bounded[N] =
    inline if i < 0 || i > constValue[N] then
      error("value out of 0.." + constValue[N])
    else i

  inline def last[N <: Int]: Bounded[N] = constValue[N]

  extension [N <: Int](b: Bounded[N])
    def toInt: Int = b
    /** The successor that stays inside the bound: a retry past the bound stays at it. */
    inline def saturatingSucc: Bounded[N] = if b < constValue[N] then b + 1 else b

  given [N <: Int](using v: ValueOf[N]): Finite[Bounded[N]] with
    def values = 0 to v.value

  given [N <: Int]: CanEqual[Bounded[N], Bounded[N]] = CanEqual.derived

// ---------------------------------------------------------------------------------------------
// Phased: `starts` and `ends` name a state by its phase
// ---------------------------------------------------------------------------------------------

/** A machine state has a field named `phase`; `starts`/`ends` name that field and the table is
  * expanded over every other field. Derived by looking the label up in the `Mirror`, so a state
  * without the field is rejected at the machine declaration, not in a test. */
trait Phased[S]:
  type Phase
  def phaseOf(s: S): Phase

object Phased:
  /** The form every entry point asks for: `P` is bound by the given, so `starts(Phase.unstarted)`
    * types against it. */
  type Aux[S, P] = Phased[S] { type Phase = P }

  transparent inline given derived[S](using m: Mirror.ProductOf[S]): Phased[S] =
    inline indexOf[m.MirroredElemLabels, "phase", 0] match
      case -1 =>
        error("a machine state needs a `phase` field; " + constValue[m.MirroredLabel] + " has none")
      case i =>
        new Phased[S]:
          type Phase = Tuple.Elem[m.MirroredElemTypes, i.type]
          def phaseOf(s: S) = s.asInstanceOf[Product].productElement(i).asInstanceOf[Phase]

  private transparent inline def indexOf[Ls <: Tuple, L, I <: Int]: Int =
    inline erasedValue[Ls] match
      case _: EmptyTuple => -1
      case _: (L *: _) => constValue[I]
      case _: (_ *: t) => indexOf[t, L, I + 1]

// ---------------------------------------------------------------------------------------------
// Entities, parties, actions, observations
// ---------------------------------------------------------------------------------------------

final case class Entity(name: String, keyField: Option[String] = None, refers: Map[String, Entity] = Map.empty):
  infix def key(field: String): Entity = copy(keyField = Some(field))
  infix def refer(links: (String, Entity)*): Entity = copy(refers = refers ++ links)

def entity(name: String): Entity = Entity(name)

/** Who performs an action. `system` is reserved for timers. */
enum Party derives CanEqual:
  case caller, handler, worker, network, operator, system
export Party.{caller, handler, worker, network, operator}

/** An action with its typed, finite inputs. `I` is a tuple, one element per `input` line, so
  * `schedule` is `Action[(Timeout, Timeout, Timeout)]` and `workerStop` is `Action[EmptyTuple]`.
  * A class is one assignment of `I`, which `Finite[I]` enumerates. */
final class Action[I <: Tuple](
    val name: String,
    val party: Party,
    val on: Option[Entity],
    val creates: Option[Entity],
    val schema: Option[String],
    val inputs: List[String],
    val results: Option[Finite[?]],
    val examples: Map[I, String],
)(using val finite: Finite[I]):
  def classes: IndexedSeq[Classed] = finite.values.map(Classed(this, _))
  override def toString = name

  /* The declaration is the value itself: each line below returns a copy with one field set, and
   * each `input` line extends the tuple type by one element, which is what makes the
   * step-function arity a compile-time fact. Bodies are sketched. */
  infix def party(p: Party): Action[I] = ???
  infix def on(e: Entity): Action[I] = ???
  infix def creates(e: Entity): Action[I] = ???
  infix def schema(s: String): Action[I] = ???
  infix def results[R: Finite]: Action[I] = ???
  def input[A: Finite](name: String): Action[Tuple.Append[I, A]] = ???
  infix def examples(xs: (I, String)*): Action[I] = ???

def action(name: String): Action[EmptyTuple] = ???

/** A `system` action a machine owns. */
def timer(name: String): Action[EmptyTuple] = ???

/** An action together with one assignment of its inputs: what a Scenario lists and a `when:` names. */
final case class Classed(action: Action[?], inputs: Tuple)

extension [A](a: Action[A *: EmptyTuple])
  def apply(x: A): Classed = Classed(a, x *: EmptyTuple)
extension [A, B, C](a: Action[(A, B, C)])
  def apply(x: A, y: B, z: C): Classed = Classed(a, (x, y, z))

/** A derived read: evidence with no history event behind it. */
final case class Observation(name: String, on: Entity, read: String)
def observation(name: String): ObservationDecl = ???
final class ObservationDecl:
  infix def on(e: Entity): ObservationDecl = ???
  infix def read(field: String): Observation = ???

/** What an `evidence:` line resolves to: a catalog event name, or a declared observation. */
type Evidence = String | Observation

// ---------------------------------------------------------------------------------------------
// Machine
// ---------------------------------------------------------------------------------------------

/** An action bound to its step function, in the uniform shape `(S, I) => rows`. */
final case class StepBinding[S, O, F](action: Action[?], run: (S, Tuple) => List[Step[S, O, F]])

/** `action ~> stepFunction`. `TupledFunction` lets the author write the step function with the
  * inputs spread as plain parameters, `(state, reply)` or `(state, a, b, c)`, and rejects a
  * function whose arity does not match the action's declared inputs. */
extension [I <: Tuple](a: Action[I])
  infix def ~>[S, O, F, Fn](f: Fn)(using
      tf: TupledFunction[Fn, (S *: I) => List[Step[S, O, F]]]
  ): StepBinding[S, O, F] =
    StepBinding(a, (s, i) => tf.tupled(f)(s *: i.asInstanceOf[I]))

extension [S, O, F](a: Action[EmptyTuple])
  infix def ~>(f: S => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a, (s, _) => f(s))

/** What a Property, a Scenario or a set names: a Machine, or a Compose read as one. */
sealed trait Model[S]:
  def name: String
  def finite: Finite[S]

/** The elaborated machine. Everything the checks read is a table computed once from `Finite[S]`,
  * the bindings and the declarations; the DSL block fills the fields, the constructor checks them. */
final class Machine[S, O, F](
    val name: String,
    val entity: Entity,
    val starts: List[S],
    val ends: List[S],
    val timers: List[Action[EmptyTuple]],
    val unobservable: List[Action[EmptyTuple]],
    val evidence: F => Evidence,
    val steps: List[StepBinding[S, O, F]],
    val refinement: Option[Refinement[S, ?]],
)(using val finite: Finite[S]) extends Model[S]:
  /** Every state, every action class, every row. */
  lazy val table: Table[S, O, F] = Table.build(this)
  def actionKeys: IndexedSeq[String] = table.classes.map(_.key)
  def transitions: List[(S, Classed, Step[S, O, F])] = table.rows
  def reachable: List[S] = Table.reachableFrom(starts, transitions)
  /** A non-end state no row leaves. */
  def stuck: Option[S] = ???
  def stateKeyFor(s: S): String = ???

  /** `from: polling` / `restrict: [workerStop, serve]`: the same machine with fewer actions. */
  infix def restrict(actions: Action[?]*): Machine[S, O, F] = ???

final case class Table[S, O, F](
    states: IndexedSeq[S],
    classes: IndexedSeq[ClassKey],
    rows: List[(S, Classed, Step[S, O, F])],
)
final case class ClassKey(key: String)
object Table:
  def build[S, O, F](m: Machine[S, O, F]): Table[S, O, F] = ??? // states x classes, each step fn once
  def reachableFrom[S, C, R](starts: List[S], rows: List[(S, C, Step[S, ?, ?])]): List[S] = ???

/** The machine-declaration scope. Every entry point below is a top-level `def` that takes it as a
  * context parameter, so inside `machine(...) { ... }` the author writes `starts(...)` bare. */
final class MachineScope[S, O, F]:
  private[umpire] var entity: Option[Entity] = None
  private[umpire] var starts: List[S] = Nil
  private[umpire] var ends: List[S] = Nil
  private[umpire] var timers: List[Action[EmptyTuple]] = Nil
  private[umpire] var unobservable: List[Action[EmptyTuple]] = Nil
  private[umpire] var evidence: Option[F => Evidence] = None
  private[umpire] var steps: List[StepBinding[S, O, F]] = Nil
  private[umpire] var refinement: Option[Refinement[S, ?]] = None

def machine[S: Finite, O, F](name: String)(body: MachineScope[S, O, F] ?=> Unit): Machine[S, O, F] =
  given scope: MachineScope[S, O, F] = MachineScope()
  body
  // Sketched: every timer bound, every unobservable a timer, every action bound at most once,
  // every start and end in the table, refinement checked against the table. Each failure is an
  // exception naming the machine and the line, raised when the enclosing object initialises,
  // which for the Models is the first test that touches them.
  Machine(name, scope.entity.getOrElse(sys.error(s"$name: no `forEntity`")), scope.starts,
    scope.ends, scope.timers, scope.unobservable, scope.evidence.getOrElse(_ => ""),
    scope.steps, scope.refinement)

def forEntity[S, O, F](e: Entity)(using m: MachineScope[S, O, F]): Unit = m.entity = Some(e)

/** `starts: [scheduled]`. The phase names the states; every other field is expanded. `P` is
  * bound by the `Phased.Aux` given, so a value of another machine's phase type is a type error. */
def starts[S, O, F, P](using m: MachineScope[S, O, F], f: Finite[S], p: Phased.Aux[S, P])(phases: P*): Unit =
  m.starts = f.values.filter(s => phases.contains(p.phaseOf(s))).toList
def ends[S, O, F, P](using m: MachineScope[S, O, F], f: Finite[S], p: Phased.Aux[S, P])(phases: P*): Unit =
  m.ends = f.values.filter(s => phases.contains(p.phaseOf(s))).toList
def timers[S, O, F](ts: Action[EmptyTuple]*)(using m: MachineScope[S, O, F]): Unit = m.timers = ts.toList
def unobservable[S, O, F](ts: Action[EmptyTuple]*)(using m: MachineScope[S, O, F]): Unit = m.unobservable = ts.toList

/** `evidence:` as a total function from facts, so a fact without an evidence line is a
  * non-exhaustive match the compiler reports, not a name the runtime fails to resolve. */
def evidence[S, O, F](using m: MachineScope[S, O, F])(lines: F => Evidence): Unit = m.evidence = Some(lines)
def steps[S, O, F](bindings: StepBinding[S, O, F]*)(using m: MachineScope[S, O, F]): Unit = m.steps = bindings.toList

/** The abstraction function as a type-level fact: `Refines[ProtocolState, ProductState]` is a
  * given the Model declares once, next to `productOf`. The machine's `refines` line uses it, and
  * so does a `verify` Query that reads a product Property on a protocol Scenario. The identity
  * instance is what lets a Property be read on its own machine. */
final case class Refines[S, P](map: S => P)
object Refines:
  given identity[S]: Refines[S, S] = Refines(s => s)

/** `refines: m` / `map: f`. Checked when the machine is built, by mapped states: a protocol row
  * `(s, a, s')` is accounted for when `map(s) == map(s')` (a product stutter) or the product has
  * a row from `map(s)` to `map(s')` under any action class. The derived mapping is kept for the
  * pins. */
def refines[S, O, F, S2](product: Machine[S2, ?, ?])(using m: MachineScope[S, O, F], via: Refines[S, S2]): Unit =
  m.refinement = Some(Refinement(product, via.map))

final case class Refinement[S, S2](product: Machine[S2, ?, ?], map: S => S2):
  /** `None` when every row was accounted for; otherwise the first row whose mapped states are
    * neither equal nor joined by any product row. */
  lazy val rejected: Option[String] = ???
  lazy val rows: List[RefinementRow] = ???
enum RefinementRow:
  case Step(productClass: String)
  case Stutter

// ---------------------------------------------------------------------------------------------
// Property, Scenario, Limits, Query
// ---------------------------------------------------------------------------------------------

/** A same-step claim (`when` set) or a transition claim (`when` empty). Indexed by the machine's
  * state type, so a Query pairing a Property with a Scenario of another machine does not type. */
final case class Property[S](
    name: String,
    machine: Model[S],
    when: Option[Classed | Action[?]],
    same: Option[Step[S, ?, ?] => Boolean],
    transition: Option[(Step[S, ?, ?], Step[S, ?, ?]) => Boolean],
)

final class PropertyDecl[S](name: String, m: Model[S], when: Option[Classed | Action[?]]):
  infix def when(c: Classed | Action[?]): PropertyDecl[S] = PropertyDecl(name, m, Some(c))
  infix def holds(p: Step[S, ?, ?] => Boolean): Property[S] = Property(name, m, when, Some(p), None)
  infix def holds(p: (Step[S, ?, ?], Step[S, ?, ?]) => Boolean): Property[S] =
    Property(name, m, when, None, Some(p))

def property[S](name: String)(machine: Model[S]): PropertyDecl[S] = PropertyDecl(name, machine, None)

/** `starts` by phase, `actions` in order. A timer or a no-input action is listed bare, a classed
  * action with its inputs applied; the union type admits both without a conversion. */
final case class Scenario[S](name: String, model: Model[S], starts: List[S], actions: List[Classed])

final class ScenarioDecl[S](name: String, m: Model[S], start: List[S]):
  infix def starts[P](using p: Phased.Aux[S, P])(phase: P): ScenarioDecl[S] =
    ScenarioDecl(name, m, m.finite.values.filter(s => p.phaseOf(s) == phase).toList)
  /** A composition's start, named by one member's phase (`operation at Phase.unscheduled`). */
  infix def starts(phase: MemberPhase): ScenarioDecl[S] = ???
  infix def actions(as: (Classed | Action[EmptyTuple] | MemberAction[EmptyTuple])*): Scenario[S] =
    Scenario(name, m, start, as.toList.map {
      case c: Classed => c
      case a: Action[EmptyTuple] @unchecked => Classed(a, EmptyTuple)
      case ma: MemberAction[EmptyTuple] @unchecked => Classed(ma.action, EmptyTuple)
    })

def scenario[S](name: String)(model: Model[S]): ScenarioDecl[S] = ScenarioDecl(name, model, Nil)

final case class Limits(name: String, steps: Int, actions: Int, search: Int)

/** Checked as written: a search budget that cannot hold the exact sequences the steps admit is a
  * typo, and the message says which field. */
inline def limits(name: String)(inline steps: Int, inline actions: Int, inline search: Int): Limits =
  inline if steps <= 0 then error("limits: steps must be positive")
  else inline if actions < steps then error("limits: actions must be at least steps")
  else Limits(name, steps, actions, search)

enum Result:
  case Found(witness: List[Classed])
  case NotFound(explored: Int)
  case Verified(traces: Int)
  case Counterexample(trace: List[Classed])

/** `find` is realized by a set; `verify` is searched over every trace of the path and never
  * realized. `S` is the Scenario's state type and `P` the Property's; `via` reads a Scenario step
  * as a Property step, and is the identity unless the Property is a product claim read on the
  * protocol machine. The bounded search is `run`, an ordinary function the test module can also
  * call. */
enum Query[S]:
  case Find[S, P](name: String, property: Property[P], scenario: Scenario[S], via: Refines[S, P], limits: Limits) extends Query[S]
  case Verify[S, P](name: String, property: Property[P], scenario: Scenario[S], via: Refines[S, P], limits: Limits) extends Query[S]
  def name: String
  def run: Result = ??? // sketched: BFS over table rows along the scenario, cut at limits.search

final class QueryDecl(name: String):
  infix def find[P](p: Property[P]): QueryIn[P] = QueryIn(name, p, find = true)
  infix def verify[P](p: Property[P]): QueryIn[P] = QueryIn(name, p, find = false)
final class QueryIn[P](name: String, p: Property[P], find: Boolean):
  /** The Scenario's machine must be the Property's, or refine it: `Refines[S, P]` is the identity
    * given or the one the Model declared next to `productOf`. Anything else is a type error. */
  infix def in[S](s: Scenario[S])(using via: Refines[S, P]): QueryLimits[S, P] = QueryLimits(name, p, s, via, find)
final class QueryLimits[S, P](name: String, p: Property[P], s: Scenario[S], via: Refines[S, P], find: Boolean):
  infix def limits(l: Limits): Query[S] =
    if find then Query.Find(name, p, s, via, l) else Query.Verify(name, p, s, via, l)

def query(name: String): QueryDecl = QueryDecl(name)

object Query:
  /** Run the Query while compiling and fail the compilation at the author's expression when the
    * search does not find (or does not verify). The Query must be a stable path into a module that
    * is already compiled: a macro cannot evaluate code from its own compilation run, so the Model
    * lives upstream and the pin downstream (in this sample, the test module). */
  inline def pinned[S](inline q: Query[S]): Query[S] = ${ pinnedImpl('q) }

  private def pinnedImpl[S: Type](q: Expr[Query[S]])(using Quotes): Expr[Query[S]] =
    import quotes.reflect.*
    val term = q.asTerm.underlyingArgument
    val stable = term match
      case s @ Select(_, _) if s.symbol.isValDef || s.symbol.flags.is(Flags.Lazy) => s
      case other =>
        report.errorAndAbort(
          s"Query.pinned needs a stable path to a Query in an upstream module, got `${other.show}`",
          term.pos)
    // Sketched: resolve `stable.symbol.owner` as a top-level object on the macro classloader and
    // read the field. This is the "Model upstream" requirement in one line.
    val query: Query[S] = loadStable[Query[S]](stable.symbol)
    query.run match
      case Result.Found(_) | Result.Verified(_) => q
      case Result.NotFound(explored) =>
        report.errorAndAbort(
          s"""${query.name}: `${propertyName(query)}` is not reached on `${scenarioName(query)}`
             |within ${limitsName(query)} ($explored candidate traces searched)""".stripMargin,
          term.pos)
      case Result.Counterexample(trace) =>
        report.errorAndAbort(
          s"""${query.name}: `${propertyName(query)}` fails on `${scenarioName(query)}`:
             |  ${trace.mkString(" -> ")}""".stripMargin,
          term.pos)

  private def loadStable[T](using Quotes)(sym: quotes.reflect.Symbol): T = ???
  private def propertyName(q: Query[?]): String = q match
    case Query.Find(_, p, _, _, _) => p.name
    case Query.Verify(_, p, _, _, _) => p.name
  private def scenarioName(q: Query[?]): String = q match
    case Query.Find(_, _, s, _, _) => s.name
    case Query.Verify(_, _, s, _, _) => s.name
  private def limitsName(q: Query[?]): String = q match
    case Query.Find(_, _, _, _, l) => l.name
    case Query.Verify(_, _, _, _, l) => l.name

/** A compile-time pin on a state count. `Finite.sizeOf` folds to a literal, so the comparison is
  * decided by the inliner and a wrong count is an error on the pin's own line. */
inline def pinStates[S](inline expected: Int)(using Mirror.Of[S]): Unit =
  inline if Finite.sizeOf[S] != expected then
    error("state count is " + Finite.sizeOf[S] + ", pin says " + expected)

// ---------------------------------------------------------------------------------------------
// Set
// ---------------------------------------------------------------------------------------------

enum Purpose derives CanEqual:
  case functional, canary, exploratory
export Purpose.{functional, canary, exploratory}

enum Binding derives CanEqual:
  case driven, observed
export Binding.{driven, observed}

enum Repeat derives CanEqual:
  case implementation
export Repeat.implementation

/** `cover: rows | results | classMembers`, a flag set with `|`. */
opaque type Cover = Int
object Cover:
  val rows: Cover = 1
  val results: Cover = 2
  val classMembers: Cover = 4
  extension (c: Cover) infix def |(o: Cover): Cover = c | o
export Cover.{rows, results, classMembers}

final case class Set(
    name: String,
    purpose: Purpose,
    bind: Map[Party, Binding],
    repeat: Option[Repeat],
    queries: List[Query[?]],
    machine: Option[Model[?]],
    cover: Option[Cover],
    budget: Option[Limits],
)

final class SetScope:
  private[umpire] var purpose: Option[Purpose] = None
  private[umpire] var bind: Map[Party, Binding] = Map.empty
  private[umpire] var repeat: Option[Repeat] = None
  private[umpire] var queries: List[Query[?]] = Nil
  private[umpire] var machine: Option[Model[?]] = None
  private[umpire] var cover: Option[Cover] = None
  private[umpire] var budget: Option[Limits] = None

def set(name: String)(body: SetScope ?=> Unit): Set =
  given s: SetScope = SetScope()
  body
  // Sketched: a functional set names only `find` Queries; an exploratory set names a machine, a
  // cover and a budget and no Queries; a canary's Queries have no silent step on their path.
  Set(name, s.purpose.get, s.bind, s.repeat, s.queries, s.machine, s.cover, s.budget)

def purpose(p: Purpose)(using s: SetScope): Unit = s.purpose = Some(p)
def bind(pairs: (Party, Binding)*)(using s: SetScope): Unit = s.bind = pairs.toMap
def repeat(r: Repeat)(using s: SetScope): Unit = s.repeat = Some(r)
def queries(qs: Query[?]*)(using s: SetScope): Unit = s.queries = qs.toList
def machine(m: Model[?])(using s: SetScope): Unit = s.machine = Some(m)
def cover(c: Cover)(using s: SetScope): Unit = s.cover = Some(c)
def budget(l: Limits)(using s: SetScope): Unit = s.budget = Some(l)

// ---------------------------------------------------------------------------------------------
// Compose
// ---------------------------------------------------------------------------------------------

/** One member of a composition: a machine of another entity and the projection that reads its
  * state out of the composed state. `member(action)` names the member's copy of an action. */
final class Member[S, MS](val name: String, val machine: Machine[MS, ?, ?], val project: S => MS):
  def apply[I <: Tuple](a: Action[I]): MemberAction[I] = MemberAction(this, a)
  /** A composed start or end, named by the member's phase. */
  infix def at[P](using Phased.Aux[MS, P])(phase: P): MemberPhase = MemberPhase(this, phase)

final case class MemberAction[I <: Tuple](member: Member[?, ?], action: Action[I]):
  /** `a || b`: the two member actions fire as one step of the composition. */
  infix def ||(other: MemberAction[?]): Sync = Sync(this, other)
extension [A](ma: MemberAction[A *: EmptyTuple])
  def apply(x: A): Classed = Classed(ma.action, x *: EmptyTuple)
extension [A, B, C](ma: MemberAction[(A, B, C)])
  def apply(x: A, y: B, z: C): Classed = Classed(ma.action, (x, y, z))

final case class Sync(left: MemberAction[?], right: MemberAction[?])
final case class MemberPhase(member: Member[?, ?], phase: Any)

/** The composed machine. An `object` extending it declares its members as `val`s, so a Scenario
  * outside the object can name `nexusCaller.operation(schedule)(...)`. It is a `Model`, so
  * Property, Scenario and Query take it where they take a Machine. */
abstract class Compose[S](val name: String)(using val finite: Finite[S]) extends Model[S]:
  private val members = List.newBuilder[Member[S, ?]]
  private val syncs = List.newBuilder[(Action[?], Sync)]
  private val startPhases = List.newBuilder[MemberPhase]
  private val endPhases = List.newBuilder[MemberPhase]
  protected def member[MS](name: String, m: Machine[MS, ?, ?])(project: S => MS): Member[S, MS] =
    val mem = Member(name, m, project)
    members += mem
    mem
  protected def sync(as: Action[?], pair: Sync): Unit = syncs += ((as, pair))
  protected def starts(phases: MemberPhase*): Unit = startPhases ++= phases
  protected def ends(phases: MemberPhase*): Unit = endPhases ++= phases
  /** The product machine over member states, with unsynced member actions executable on their
    * own and each `sync` pair as one row. `Machine` so Property, Scenario and Query apply as-is.
    * Sketched: built from the four buffers above on first use. */
  lazy val machine: Machine[S, Any, Any] = ???
