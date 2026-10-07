// Sugar: definitions whose meaning a core declaration already expresses, kept for readability. Each
// names the core form it stands for, and the IR generator (model/irgen/Syntax.scala) lowers it to
// the IR that core form lifts to. No other file of the framework uses them. Among them are the
// block forms a machine object's section `val`s are written in, `is { }` for a predicate of the
// state and `effect { }` for a step function of it, with an effect block's `record` and
// `reject(outcome)` statements, the `View` and `Draft` the field accessors of the state read and
// assign through, and `Recorded`, the status fact a phase enum's case declares.
package umpire

import scala.annotation.{implicitNotFound, targetName, unused}
import scala.collection.mutable
import scala.compiletime.codeOf
import scala.reflect.{ClassTag, TypeTest}
import scala.util.NotGiven

// The outcome `enter` and `stay` answer for a machine whose outcomes are `O`, declared once beside
// the outcome type: `given Ok[Outcome] = Ok(Outcome.accepted)`. Core form: the outcome itself,
// `Outcome.accepted`, written in each `Step`.
final case class Ok[O](outcome: O)

// The ok outcomes the framework declares itself, in the implicit scope of every `Ok`: a Model's own
// `given Ok` outranks them, and a machine on the shared outcomes declares none. Core form: the
// given beside the outcome type, `given Ok[Outcome] = Ok(Outcome.accepted)`.
object Ok:
  // `accepted`, the ok outcome of the shared outcomes (`outcomes`). Core form:
  // `given Ok[Outcome] = Ok(Outcome.accepted)`.
  given Ok[outcomes.Outcome] = Ok(outcomes.Outcome.accepted)

// One step with the ok outcome into `state`, recording `facts`. Core form:
// `List(Step(Outcome.accepted, state, List(facts*)))`.
def enter[S, O, F](state: S, facts: F*)(using ok: Ok[O]): List[Step[S, O, F]] =
  List(Step(ok.outcome, state, facts.toList))

// One step with the ok outcome that keeps the state and records nothing. Core form:
// `List(Step(Outcome.accepted, s))`.
def stay[S, O, F](s: S)(using ok: Ok[O]): List[Step[S, O, F]] =
  List(Step(ok.outcome, s))

// One step with the outcome `outcome` that keeps the state and records nothing, such as a request
// the machine refuses: `reject(Outcome.notFound, s)`. Core form: `List(Step(Outcome.notFound, s))`.
def reject[S, O](outcome: O, s: S): List[Step[S, O, Nothing]] = List(Step(outcome, s))

// No step: the action is disabled here. Core form: `Nil`.
val disabled: List[Nothing] = Nil

// The outcomes shared by every machine whose steps answer a request, accepted or rejected for a
// reason, which a machine adopts by name, `import umpire.outcomes.{Outcome, Rejection}`, in place of
// its own. `import umpire.*` does not open it, so a Model's own `Outcome` stays its own. Core form: a
// machine's own outcome enum, `enum Outcome derives Finite: case accepted, notFound`.
object outcomes:
  // Why a request is rejected, loosely after the gRPC status codes a realization maps it to: the
  // entity is not there (`notFound`), a create collides with one that is (`alreadyExists`), its
  // state forbids the request (`failedPrecondition`), or the request is bad in every state
  // (`invalidArgument`). Core form: a case of a machine's own outcome enum, `case notFound`.
  enum Rejection derives Finite:
    case notFound, alreadyExists, failedPrecondition, invalidArgument

  // What a step answers: the request is accepted, or rejected for the reason it names. Core form: a
  // machine's own outcome enum, `enum Outcome derives Finite: case accepted, notFound`.
  enum Outcome derives Finite:
    case accepted
    case rejected(why: Rejection)

// `when(...) ~> rejects(Rejection.notFound)`, in a rule of a machine on the shared outcomes: the
// effect that keeps the state, records nothing and answers `rejected(why)`, and
// `rejects(why).because("...")` the same effect with the server's explanation. Core form:
// `reject(Outcome.rejected(Rejection.notFound), s)`.
def rejects[S](why: outcomes.Rejection)(using
    @implicitNotFound(
      "rejects answers the shared umpire.outcomes.Outcome, so it is an effect of a rule, in an `on` " +
        "block, of a machine on the shared outcomes: `import umpire.outcomes.{Outcome, Rejection}`"
    ) @unused firing: Firing[S, outcomes.Outcome, ?, ?]
): Rejects[S] = Rejects(why)

// The effect `rejects(why)` writes: a function of the state alone, as every effect a rule names is.
// Core form: `(s: S) => reject(Outcome.rejected(why), s)`.
final class Rejects[S] private[umpire] (why: outcomes.Rejection)
    extends (S => List[Step[S, outcomes.Outcome, Nothing]]):
  def apply(s: S): List[Step[S, outcomes.Outcome, Nothing]] =
    List(Step(outcomes.Outcome.rejected(why), s))

  // The same effect, its step explained by `reason`, as the IR row's `because` carries it. It
  // explains once: what it returns has no `because`. Core form:
  // `reject(Outcome.rejected(why), s).because(reason)`.
  def because(reason: String): S => List[Step[S, outcomes.Outcome, Nothing]] =
    s => List(Step(outcomes.Outcome.rejected(why), s, Nil, reason))

// The state an `is { }` or `effect { }` block reads its fields of, by the accessors its state type
// declares, each of one shape: `def phase(using v: View[State]): Phase = v.get(_.phase)`.
// Core form: the state parameter of the predicate or step function, `s` in `s.phase`.
@implicitNotFound(
  "the fields of ${S} are read inside `is { }` or `effect { }`, which give the state they read: " +
    "outside a block, take the state as a parameter, `def f(s: State) = s.phase`"
)
sealed trait View[S]:
  private[umpire] def state: S

  // The value of one field of the state: `v.get(_.phase)`. Core form: `s.phase`.
  def get[A](field: S => A): A = field(state)

// The view an `is { }` block reads: the state it is asked of. Core form: the predicate's parameter.
final private class Fixed[S](private[umpire] val state: S) extends View[S]

// The status fact a phase enum's case is recorded as, declared on each case through the enum's
// parameter, which has no default, so a case without one does not compile:
// `enum Phase(val status: Fact) extends Recorded[Fact]`, `case started extends Phase(statusStarted)`.
// An `effect { }` block that assigns such a value records its status after the facts the block
// records. Core form: the status fact each step writes by hand,
// `enter(s.copy(phase = started), statusStarted)`.
trait Recorded[F]:
  def status: F

// What an `effect { }` block makes of one step while it runs: the state it enters, which each field
// assignment replaces with a copy, by the setters its state type declares, each of one shape:
// `def phase_=(p: Phase)(using d: Draft[State, ?, ?]): Unit = d.set(_.copy(phase = p))`, or for a
// field whose values are `Recorded`, `def phase_=(p: Phase)(using d: Draft[State, ?, Fact]): Unit =
// d.set(p)(_.copy(phase = p))`; the facts it records, then the status of the value such a setter
// assigned; and the outcome it rejects with, if any. Each call of the effect makes a fresh draft,
// which never leaves the block. Core form: the arguments of the step the step function returns,
// `Step(outcome, s.copy(phase = p), List(facts*))`.
@implicitNotFound(
  "this assigns a field of ${S}, records a fact or rejects, which only an `effect { }` block does: " +
    "outside one, return the steps, `enter(s.copy(...), fact)` or `reject(outcome, s)`"
)
final class Draft[S, O, F] private[umpire] (start: S) extends View[S]:
  private var current: S = start // scalafix:ok DisableSyntax.var
  private var facts: List[F] = Nil // scalafix:ok DisableSyntax.var
  private var rejection: Option[O] = None // scalafix:ok DisableSyntax.var
  private var statuses: List[F] = Nil // scalafix:ok DisableSyntax.var

  private[umpire] def state: S = current

  // The state with one field replaced: `d.set(_.copy(phase = p))`. Core form: `s.copy(phase = p)`.
  def set(update: S => S): Unit = current = update(current)

  // The state with the field whose values declare their status replaced by `value`, which the step
  // records the status of: `d.set(p)(_.copy(phase = p))`. Core form: `s.copy(phase = p)`, and
  // `p.status` among the step's facts.
  def set(value: Recorded[F])(update: S => S): Unit =
    current = update(current)
    statuses = statuses :+ value.status

  // The step the block made, from the state `s` it started in: the rejection with `s`, or the ok
  // outcome with the state assigned and the facts in the order recorded, then the status of each
  // value assigned, which the block does not record itself.
  private[umpire] def step(s: S, ok: Ok[O]): Step[S, O, F] = rejection match
    case Some(outcome) => Step(outcome, s)
    case None          =>
      for f <- statuses do
        require(
          !facts.contains(f),
          s"an effect block records $f, which its assignment of a status already records: " +
            s"drop the record($f)"
        )
      Step(ok.outcome, current, facts ++ statuses)

  private[umpire] def add(recorded: Seq[F]): Unit = facts = facts ++ recorded

  private[umpire] def refuse(outcome: O): Unit = rejection = Some(outcome)

// `val held = is { phase == Phase.held }`: a predicate of the machine's state, its fields read by
// name. Core form: `def held(s: State) = s.phase == Phase.held`.
def is[S, O, F](using @unused owner: Owner[S, O, F])(body: View[S] ?=> Boolean): S => Boolean =
  s => body(using Fixed(s))

// `val pause = effect { phase = Phase.paused; record(statusPaused) }`: a step function of the
// machine's state alone, its fields assigned by name, and its facts recorded, or its outcome
// rejected, by statements. A block that neither records nor rejects enters the assigned state with
// no fact. Core form: `def pause(s: State) = enter(s.copy(phase = Phase.paused), statusPaused)`, or
// `reject(outcome, s)` for a block that rejects.
def effect[S, O, F](using
    @unused owner: Owner[S, O, F],
    ok: Ok[O]
)(
    body: Draft[S, O, F] ?=> Unit
): S => List[Step[S, O, F]] =
  s =>
    val draft = Draft[S, O, F](s)
    body(using draft)
    List(draft.step(s, ok))

// `record(statusPaused)`, in an `effect { }` block: the step records these facts, after those an
// earlier `record` of the block recorded. Core form: the facts of `enter(s.copy(...), facts*)`.
def record[S, O, F](using draft: Draft[S, O, F])(first: F, rest: F*): Unit =
  draft.add(first +: rest)

// `reject(Outcome.notFound)`, in an `effect { }` block: the step answers `outcome` and keeps the
// state, recording nothing. Core form: `reject(Outcome.notFound, s)`.
def reject[S, O, F](using draft: Draft[S, O, F])(outcome: O): Unit = draft.refuse(outcome)

// `phase.in(a, b, c)`: whether the value is one of the members listed, of which there is at least
// one. Written dotted, never infix. Core form: `List(a, b, c).contains(phase)`.
extension [A](value: A) def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)

// `phase.in[Closed]`: whether the phase has the role `Closed` (model/umpire/Roles.scala), the roles
// its case declares and the broader ones they extend. Written dotted, never infix. Core form:
// `List(<the cases of Closed, in declaration order>).contains(phase)`.
extension [A](value: A)
  def in[R](using role: TypeTest[A, R]): Boolean = role.unapply(value).nonEmpty

// `a implies b`: `b` holds wherever `a` does. `b` is read only where `a` holds, so a hole it reaches
// is not reached where `a` is false. Core form: `!a || b`.
extension (a: Boolean) infix def implies(b: => Boolean): Boolean = !a || b

// `after.records(fact)`: whether the step records the fact, which is of the step's own fact type:
// a step's facts are covariant, so the evidence keeps a fact of another type from widening it.
// Core form: `after.facts.contains(fact)`.
extension [S, O, F](step: Step[S, O, F])
  def records[G](fact: G)(using G <:< F): Boolean = step.facts.contains(fact)

// `after.records(_.member, fact)`: whether a composition's step records the fact of the member the
// selector names, the composition form of `after.records(fact)`. Core form:
// `after.facts.contains("member_fact")`, the composed key `<field>_<fact>` a composition records.
extension [S, O](step: Step[S, O, String])
  def records[M](@unused member: S => M, fact: Any): Boolean =
    step.facts.exists(_.endsWith(s"_$fact"))

// `m.property("terminalIsFinal").once(terminal).keeps(_.phase)`: once `over` holds of the state
// before a step, the step keeps the value the projection selects. Core form:
// `holdsAcross((before, after) => !over(before) || after.state.phase == before.phase)`.
extension [S, O, F](b: PropertyBuilder[S, O, F])
  def once(over: S => Boolean): Once[S, O, F] = Once(b, over)

  // `never(to)`: no step is one `to` holds of, the same-step invariant. Core form:
  // `holds(after => !to(after))`. `never(to).from(before)` says it of the steps from a state `before`
  // holds of only.
  def never(to: Step[S, O, F] => Boolean): Never[S, O, F] = Never(b, to)

  // `stays(p)`: a step from a state `p` holds of keeps it holding. Core form:
  // `holdsAcross((before, after) => !p(before) || p(after.state))`. `stays(p).unless(release)` also
  // lets a step `release` holds of leave it.
  def stays(p: S => Boolean): Stays[S, O, F] = Stays(b, p)

// What `once(over)` leaves to say: the value it keeps. Core form: the `holdsAcross` lambda `keeps`
// finishes.
final class Once[S, O, F] private[umpire] (b: PropertyBuilder[S, O, F], over: S => Boolean):
  // The value a step from a state `over` holds of keeps, a field path such as `_.phase` or
  // `_.order.phase`. Core form:
  // `holdsAcross((before, after) => !over(before) || after.state.phase == before.phase)`.
  def keeps[V](projection: S => V): Property[S] =
    Property(
      PropertyDecl(
        b.name,
        b.m,
        b.when,
        None,
        Some((before: S, after: Step[S, O, F]) =>
          !over(before) || projection(after.state) == projection(before)
        )
      )
    )

// `never(to)`, a same-step Property, which `from` turns into a transition one. Core form:
// `holds(after => !to(after))`.
final class Never[S, O, F] private[umpire] (
    b: PropertyBuilder[S, O, F],
    to: Step[S, O, F] => Boolean
) extends Property[S](
      PropertyDecl(b.name, b.m, b.when, Some((after: Step[S, O, F]) => !to(after)), None)
    ):
  // No step from a state `before` holds of is one `to` holds of. Core form:
  // `holdsAcross((before, after) => !before(before) || !to(after))`.
  def from(before: S => Boolean): Property[S] =
    Property(
      PropertyDecl(
        b.name,
        b.m,
        b.when,
        None,
        Some((s: S, after: Step[S, O, F]) => !before(s) || !to(after))
      )
    )

// `stays(p)`, a transition Property, which `unless` releases. Core form:
// `holdsAcross((before, after) => !p(before) || p(after.state))`.
final class Stays[S, O, F] private[umpire] (b: PropertyBuilder[S, O, F], p: S => Boolean)
    extends Property[S](
      PropertyDecl(
        b.name,
        b.m,
        b.when,
        None,
        Some((before: S, after: Step[S, O, F]) => !p(before) || p(after.state))
      )
    ):
  // A step `release` holds of may leave `p`. Core form:
  // `holdsAcross((before, after) => !p(before) || p(after.state) || release(after))`.
  def unless(release: Step[S, O, F] => Boolean): Property[S] =
    Property(
      PropertyDecl(
        b.name,
        b.m,
        b.when,
        None,
        Some((before: S, after: Step[S, O, F]) => !p(before) || p(after.state) || release(after))
      )
    )

// A named slot with the value it receives, what `slot := value` writes. Core form: the value itself,
// written at the slot's place, `expires` in `start(unset, expires, unset)`.
final case class Assigned[A] private[umpire] (slot: Slot[A], value: A)

// `slot := value`: this named slot receives this value, and nothing else. The value has the slot's
// type. Core form: the value at the slot's place in the positional call, `expires` in
// `start(unset, expires, unset)`.
extension [A](slot: Slot[A])
  @targetName("set")
  def :=(value: A): Assigned[A] = Assigned(slot, value)

// `start(scheduleToStart := expires)`: the class of the inputs the call supplies by name, in the
// order the action declares them, each input it omits at its domain's first value. Each named slot
// is an input token the action declares, supplied once. Core form: the positional call,
// `start(unset, expires, unset)`.
extension [I <: NonEmptyTuple](a: Action[I])
  def apply(first: Assigned[?], rest: Assigned[?]*): Class =
    val supplied = first +: rest
    for s <- supplied do
      require(
        a.decl.tokens.contains(Some(s.slot)),
        s"an input supplied to ${a.name} is not its own"
      )
      require(supplied.count(_.slot == s.slot) == 1, s"an input of ${a.name} is supplied twice")
    val values = a.decl.tokens
      .zip(a.decl.domains)
      .map: (token, domain) =>
        supplied.find(s => token.contains(s.slot)).fold(domain.values.head)(_.value)
    Class(a.decl, values)

// `val retainedOutcome = sticky(outcomePreserved)`: a monitor of a promise about every step that,
// once a step breaks it, stays broken, named after its `val`. Its verdict is read after every step.
// Core form:
// `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(after))(broken => broken)`.
def sticky[S, O, F](promise: Step[S, O, F] => Boolean): Monitor[S, O, F, Boolean] =
  monitor[S, O, F, Boolean](false)((broken, _, after) => broken || !promise(after))(broken =>
    broken
  )

// `val ownerAcknowledgment = stickyAcross(ackOnlyWhenKept)`: `sticky` of a promise about the state
// before a step and the step. Core form:
// `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(before, after))(broken => broken)`.
def stickyAcross[S, O, F](promise: (S, Step[S, O, F]) => Boolean): Monitor[S, O, F, Boolean] =
  monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(before, after))(
    broken => broken
  )

// A case of an `on` block: where in the state space its action fires, `in(placed)`, `in(open)`,
// `where(g)`, `always` or `in(open).where(g)`, and through `~>` the effect it has there,
// `in(placed) ~> effects.ship`. Core form: the arm `if g(s) then e(s, inputs) else ...` of the
// action's step function.
final class Case[S, O, F] private[umpire] (
    private[umpire] val heading: String,
    private[umpire] val guard: S => Boolean
):
  // This case where `condition` also holds of the state, beyond its phases:
  // `in(open).where(_.deadline == Timeout.expires)`. Core form: `g(s) && condition(s)`.
  def where(condition: S => Boolean): Case[S, O, F] =
    Case(s"$heading.where", s => guard(s) && condition(s))

  // The effect of the block's action where this case holds, read of the state alone:
  // `in(placed) ~> effects.ship`. Core form: the arm `if g(s) then ship(s)` of the action's step
  // function.
  infix def ~>(effect: S => List[Step[S, O, F]])(using firing: Firing[S, O, F, ?]): Unit =
    firing.bind(this, (s, _) => effect(s))

  // The effect of the block's action, of one input, where this case holds, read of the state and
  // the input: `in(open) ~> effects.change`. Core form: the arm `if g(s) then change(s, c)`.
  @targetName("readsOne")
  infix def ~>[A](effect: (S, A) => List[Step[S, O, F]])(using
      firing: Firing[S, O, F, A *: EmptyTuple]
  ): Unit = firing.bind(this, effectOf(firing.decl, effect))

  // The effect of the block's action, of two inputs, read of the state and its inputs. Core form:
  // the arm `if g(s) then e(s, x, y)`.
  @targetName("readsTwo")
  infix def ~>[A, B](effect: (S, A, B) => List[Step[S, O, F]])(using
      firing: Firing[S, O, F, (A, B)]
  ): Unit = firing.bind(this, effectOf(firing.decl, effect))

  // The effect of the block's action, of three inputs, read of the state and its inputs:
  // `in(unplaced) ~> effects.open`. Core form: the arm `if g(s) then open(s, x, y, z)`.
  @targetName("readsThree")
  infix def ~>[A, B, C](effect: (S, A, B, C) => List[Step[S, O, F]])(using
      firing: Firing[S, O, F, (A, B, C)]
  ): Unit = firing.bind(this, effectOf(firing.decl, effect))

// The action, or the one class of it, an `on` block fires, and where its cases go, given to the
// block's cases. Core form: none of its own; it is the `action` of `action ~> stepFunction`, whose
// arms the block's cases are.
final class Firing[S, O, F, I <: Tuple] private[umpire] (
    private[umpire] val decl: ActionDecl,
    private[umpire] val values: Option[List[Any]],
    private[umpire] val action: String,
    add: (Case[S, O, F], (S, List[Any]) => List[Step[S, O, F]]) => Unit
):
  private[umpire] def bind(c: Case[S, O, F], effect: (S, List[Any]) => List[Step[S, O, F]]): Unit =
    add(c, effect)

// A case that holds where `condition` holds of the state: `where(states.held) ~> effects.settle`.
// Core form: `if condition(s) then e(s) else ...`.
def where[S, O, F](using firing: Firing[S, O, F, ?])(condition: S => Boolean): Case[S, O, F] =
  val _ = firing
  Case("where", condition)

// A case that holds in every state: `always ~> effects.keep`. Core form:
// `if true then e(s) else ...`.
def always[S, O, F](using firing: Firing[S, O, F, ?]): Case[S, O, F] =
  val _ = firing
  Case("always", _ => true)

// The projection of a machine's or composition's state onto its phase, declared once by mixing it
// into the object: `object OrderProduct extends Machine[OrderState, Outcome, OrderFact],
// Phased[OrderState, Phase](_.phase)`, or for a composition a nested phase,
// `Phased[OverQueue, Phase](_.order.phase)`. Both types are written: a trait parent's lambda takes
// no parameter type from the other parents. Its sections read it as a given: `in(placed)` and
// `in(states.terminal)` in `object rules extends Rules:` test it. It is optional, as a machine that
// names no phase is a plain `Machine`. A derived machine or derived composition mixes in none of
// its own: a derived machine reads its source's, with its phase type. Its default end is the Closed
// role, witnessed at the concrete parent declaration; an explicit end overrides it. The default's
// first evaluation refuses a phase with no Closed case, naming the object and phase type. A derived
// machine's final end prevents mixing it in, and a derived composition refuses its own projection
// as it initializes. Core form: the given `Phasing(_.phase)` the object's sections read.
trait Phased[S, P](private[umpire] val projection: S => P)(using
    Finite[P],
    TypeTest[P, Closed],
    ClassTag[P]
) extends Declares[S]:
  // The projection its sections read.
  protected given phased: Phasing[S, P] = Phasing(projection)

  private lazy val closedCases = phased.roleCases[Closed](name)

  // The default stopping point is closedness. A machine whose stopping point differs overrides it.
  // Core form: `projection(s).in(<the Closed cases of P>)`.
  def end(s: S): Boolean = closedCases.contains(projection(s))

  final override private[umpire] def declaresPhase: Boolean = true

// The phase a derived machine reads through its source. Its evidence is preferred to the evidence
// of a source that declares no phase, which it extends. Core form: none of its own; it is the
// source's `Phasing(_.phase)`.
object Phased extends Inherited.Unphased:
  // A source that mixes in `Phased[S, P]` projects its state onto `P` by its own projection.
  // Core form: the source's `Phasing(_.phase)`.
  given source[S, P, T <: Phased[S, P]]: Inherited[T, S, P] =
    Inherited(_.projection, _.end)

// When each action of a machine fires, written as the machine object's `object rules extends
// Rules`, whose cases may name phases where the machine mixes in `Phased[State, Phase](_.phase)`
// through its given: blocks of an action or action class,
// `on(clerk.ship) { in(placed) ~> effects.send }`, whose cases each say where the action fires,
// `in(...)` or `when(...)` of phases or of a named set of them, `where(g)` of the state,
// `when(...).where(g)` of both or `always`, and what it does there, `~> effects.x`; and the actions
// no state enables, `disabled(courier.strike)`. A block fires a whole action, `on(buyer.change)`,
// one class of it, `on(buyer.change(Change.hold))`, whose effects then read the state alone, or
// several of them alike, `on(clerk.ship, clerk.cancel)`; an effect that takes arguments beyond the
// state binds them in place, `effects.timeOut(_, Deadline.close)`. The blocks of the actions one
// object declares may be grouped in `from(clerk) { import clerk.*; on(ship) { ... } }`. Where no
// case holds, the action is disabled: there is no catch-all. The cases of one action class hold in
// no common state, whichever blocks they sit in: each is checked against the earlier ones as it is
// declared, over every state of the `Finite` state type and every class, and an overlap is refused
// naming the machine, the class, both rules and a state where both hold. Core form: one step
// function per action, the cases of the action in order,
// `action ~> ((s, i) => if g1(s) && c1(i) then e1(s, i) else if g2(s) then e2(s, i) else Nil)`, and
// `action ~> (_ => Nil)` for an action no state enables.
//
// `on` is `inline`, with an `inline` action: the one sanctioned exception to "no inline on the
// author surface" (.plans/DSL_OPERATORS.md, rule 5). An action's runtime name is `""`, since the
// lifter names it after its `val`, so `scala.compiletime.codeOf` names the action in an overlap
// message. TASTy is pickled before inlining, so the lifter reads the unexpanded call, and it refuses
// an expanded one loudly ("not a rule") rather than lift it.
abstract class Rules[S, O, F, P](using
    owner: Owner[S, O, F],
    phasing: Phasing[S, ? <: P]
) extends RuleBook[S, O, F]:
  private val phase: S => P = phasing.projection
  private val written = mutable.ArrayBuffer.empty[Rule[S, O, F]]
  private val never = mutable.ArrayBuffer.empty[ActionDecl]
  private val order = mutable.LinkedHashSet.empty[ActionDecl]
  private var open: Option[String] = None // scalafix:ok DisableSyntax.var
  // The `from` the rules are in, if any: its declarer's name and the actions it declares.
  private var within: Option[(String, Set[ActionDecl])] = None // scalafix:ok DisableSyntax.var

  // Runs one block's cases, refusing a block in a block, and in a `from` an action its declarer
  // does not declare.
  private[umpire] def block[I <: Tuple](
      decl: ActionDecl,
      values: Option[List[Any]],
      code: String
  )(cases: Firing[S, O, F, I] ?=> Unit): Unit =
    val action = writtenAction(code)
    require(open.isEmpty, s"on($action) sits in on(${open.get}): a block holds cases alone")
    require(!never.contains(decl), s"$action is disabled and fired by a rule")
    for (declarer, declared) <- within do
      require(
        declared.contains(decl),
        s"on($action) sits in from($declarer), and $declarer declares no $action: a from holds " +
          "the blocks of the actions its declarer declares"
      )
    open = Some(action)
    val firing =
      Firing[S, O, F, I](decl, values, action, (c, effect) => bind(decl, values, c, effect, code))
    try cases(using firing)
    finally open = None

  // Registers a rule, refusing one that overlaps an earlier rule of its class.
  private def bind(
      decl: ActionDecl,
      values: Option[List[Any]],
      c: Case[S, O, F],
      effect: (S, List[Any]) => List[Step[S, O, F]],
      code: String
  ): Unit =
    val rule = Rule(written.size + 1, c.heading, writtenAction(code), decl, values, c.guard, effect)
    for refused <- overlap(owner.machine.name, owner.machine.fs, written.toSeq, rule) do
      throw IllegalArgumentException(refused)
    written += rule
    order += decl

  // The cases of a whole action: `on(clerk.ship) { in(placed) ~> effects.send }`. Core form: the
  // action's step function, whose arms are the cases in order.
  inline def on[I <: Tuple](inline a: Action[I])(cases: Firing[S, O, F, I] ?=> Unit): Unit =
    block[I](a.decl, None, codeOf(a))(cases)

  // The cases of one class of an action, whose effects read the state alone:
  // `on(courier.report(Report.delivered)) { in(sent) ~> effects.deliver }`. Core form: the arms
  // `if g(s) && result == Report.delivered then deliver(s)` of the action's step function.
  inline def on(inline c: Class)(cases: Firing[S, O, F, EmptyTuple] ?=> Unit): Unit =
    block[EmptyTuple](c.decl, Some(c.values), codeOf(c))(cases)

  // The same cases for two actions, or classes of them, whose effects read the state alone:
  // `on(clerk.hold, clerk.cancel) { when(shipped) ~> rejects(Rejection.failedPrecondition) }`.
  // Fixed arities of two to four, since `codeOf` names no element of an `inline` varargs list.
  // Core form: one block per target with the same cases, `on(a) { cases }` and `on(b) { cases }`.
  inline def on(inline a: Action[?] | Class, inline b: Action[?] | Class)(
      cases: Firing[S, O, F, EmptyTuple] ?=> Unit
  ): Unit = each(List(a -> codeOf(a), b -> codeOf(b)))(cases)

  // The same cases for three actions, or classes of them. Core form: one block per target with
  // the same cases.
  inline def on(
      inline a: Action[?] | Class,
      inline b: Action[?] | Class,
      inline c: Action[?] | Class
  )(cases: Firing[S, O, F, EmptyTuple] ?=> Unit): Unit =
    each(List(a -> codeOf(a), b -> codeOf(b), c -> codeOf(c)))(cases)

  // The same cases for four actions, or classes of them. Core form: one block per target with the
  // same cases.
  inline def on(
      inline a: Action[?] | Class,
      inline b: Action[?] | Class,
      inline c: Action[?] | Class,
      inline d: Action[?] | Class
  )(cases: Firing[S, O, F, EmptyTuple] ?=> Unit): Unit =
    each(List(a -> codeOf(a), b -> codeOf(b), c -> codeOf(c), d -> codeOf(d)))(cases)

  // Runs the cases once for each target, as its own block, refusing a target named twice. The
  // cases run once per target, so each binds its own rules.
  private[umpire] def each(
      targets: List[(Action[?] | Class, String)]
  )(cases: Firing[S, O, F, EmptyTuple] ?=> Unit): Unit =
    val named = targets.map:
      case (a: Action[?], code) => (a.decl, None, code)
      case (c: Class, code)     => (c.decl, Some(c.values), code)
    for ((decl, values, code), i) <- named.zipWithIndex do
      require(
        !named.take(i).exists((d, v, _) => d == decl && v == values),
        s"${writtenAction(code)} is named twice in one on: name each action, or class of it, once"
      )
    for (decl, values, code) <- named do block[EmptyTuple](decl, values, code)(cases)

  // The blocks of the actions `declarer` declares, each named by its bare name after the block's
  // first statement imports them: `from(clerk) { import clerk.*; on(ship) { ... } }`. It only
  // groups: a block means in a `from` what it means outside one. Core form: the blocks it holds,
  // each naming its action through its declarer, `on(clerk.ship) { ... }`.
  def from(declarer: AnyRef)(body: => Unit): Unit =
    val name = declarer match
      case a: Actor => a.name
      case other    => objectName(other)
    require(open.isEmpty, s"from($name) sits in on(${open.get}): a block holds cases alone")
    require(
      within.isEmpty,
      s"from($name) sits in from(${within.get._1}): a from holds on blocks alone"
    )
    within = Some(name -> declaredBy(declarer))
    try body
    finally within = None

  // The actions an object declares: the `Action` values among its public vals. An action records
  // no object that holds it, so the object's members are read.
  private def declaredBy(declarer: AnyRef): Set[ActionDecl] =
    declarer.getClass.getMethods.toSet
      .filter(m => m.getParameterCount == 0 && classOf[Action[?]].isAssignableFrom(m.getReturnType))
      .map(_.invoke(declarer))
      .collect { case a: Action[?] => a.decl }

  // A case that holds in the phases listed, as the machine's `Phased[State, Phase](_.phase)` reads
  // them; the rules of a machine that is not `Phased` name no phase. Core form:
  // `List(p1, p2).contains(s.phase)`.
  def in[Q](first: Q, rest: Q*)(using PhasesOf[P, Q]): Case[S, O, F] =
    val phases = first +: rest
    Case(s"in(${phases.mkString(", ")})", s => phases.contains(phase(s)))

  // A case that holds in a named set of phases, a predicate of the projection the machine's
  // `states` declares: `in(states.terminal)`. Like the phases listed, it needs a machine that is
  // `Phased`, since any predicate is one of `Nothing`. Core form: `terminal(s.phase)`.
  inline def in(inline set: P => Boolean)(using PhasesOf[P, P]): Case[S, O, F] =
    phases(set, codeOf(set))

  private[umpire] def phases(set: P => Boolean, code: String, word: String = "in"): Case[S, O, F] =
    Case(s"$word(${code.trim})", s => set(phase(s)))

  // A case that holds in the phases listed, as `in(...)` does: `when(placed, open)`. Core form:
  // `List(p1, p2).contains(s.phase)`.
  def when[Q](first: Q, rest: Q*)(using PhasesOf[P, Q]): Case[S, O, F] =
    val phases = first +: rest
    Case(s"when(${phases.mkString(", ")})", s => phases.contains(phase(s)))

  // A case that holds in a named set of phases, as `in(set)` does: `when(states.terminal)`. Core
  // form: `terminal(s.phase)`.
  inline def when(inline set: P => Boolean)(using PhasesOf[P, P]): Case[S, O, F] =
    phases(set, codeOf(set), "when")

  // A case that holds in the phases with the role `R` (model/umpire/Roles.scala), as the projection
  // of the machine's `Phased` reads them: `when[Closed] ~> effects.notFound`; a machine that is
  // not `Phased` names no role. Core form:
  // `List(<the cases of R, in declaration order>).contains(s.phase)`.
  def when[R](using
      role: TypeTest[P, R],
      named: ClassTag[R],
      @unused projected: PhasesOf[P, P]
  ): Case[S, O, F] =
    Case(s"when[${named.runtimeClass.getSimpleName}]", s => role.unapply(phase(s)).nonEmpty)

  // Actions no state enables, which the machine binds all the same, such as a courier's strike it does
  // not feel. Core form: `action ~> (_ => Nil)`.
  def disabled(actions: Action[?]*): Unit =
    for a <- actions do
      require(
        !never.contains(a.decl) && !written.exists(_.decl == a.decl),
        "an action is disabled twice, or disabled and bound by a rule"
      )
      never += a.decl
      order += a.decl

  // `phase.in(a, b)` in a condition, as outside the rules. Core form: `List(a, b).contains(phase)`.
  extension [A](value: A) def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)

  // `phase.in[Closed]` in a condition, as outside the rules. Core form:
  // `List(<the cases of Closed, in declaration order>).contains(phase)`.
  extension [A](value: A)
    def in[R](using role: TypeTest[A, R]): Boolean = role.unapply(value).nonEmpty

  private[umpire] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    order.toVector.map: decl =>
      decl -> (if never.contains(decl) then Bound.Disabled[S, O, F]()
               else Bound.Ruled(written.filter(_.decl == decl).toVector))

// Evidence that a case's phases are of the type the rules' projection reads: the rules of a machine
// that mixes in `Phased[State, Phase](_.phase)` name phases with `in` and `when`, and the rules of
// one that is not `Phased`, whose phase type is `Nothing`, name none. Core form: none of its own;
// `in(p1, p2)` is `List(p1, p2).contains(s.phase)`.
@implicitNotFound(
  "in and when name phases of ${Q}, and these rules read phases of ${P}: mix the projection the " +
    "phases are of into the machine, `Phased[State, Phase](_.phase)`"
)
final class PhasesOf[P, Q] private ()

// The one projection a case's phases are of: its own. Core form: `List(p1, p2).contains(s.phase)`.
object PhasesOf:
  // Phases of the projection's own type, other than `Nothing`, or of a narrower one: a case that
  // takes a role, `case done extends Phase, Succeeded`, is a `Phase & Succeeded`. Core form:
  // `List(p1, p2).contains(s.phase)`, whose phases are of the type of `s.phase`.
  given [P, Q <: P](using NotGiven[P =:= Nothing]): PhasesOf[P, Q] = PhasesOf()

// The rules of one action a derivation binds in its source's place:
// `rebind(on(clerk.ship) { always ~> OrderRecord.effects.send })`, each case `where(g)` or
// `always`, since a derivation's rules name no phase. Its action is `inline`, as a rule block's is
// (`Rules`), so that an overlap names it as written. Core form: the step function
// `action ~> ((s, i) => if g(s) then e(s, i) else Nil)`.
inline def on[S, O, F, I <: Tuple](using
    owner: Owner[S, O, F]
)(inline a: Action[I])(
    cases: Firing[S, O, F, I] ?=> Unit
): RuleGroup[S, O, F] = derivedRules(owner, a.decl, codeOf(a))(cases)

// The rules of a derivation's `on`, each named by its action as `written` spells it.
private[umpire] def derivedRules[S, O, F, I <: Tuple](
    owner: Owner[S, O, F],
    decl: ActionDecl,
    written: String
)(cases: Firing[S, O, F, I] ?=> Unit): RuleGroup[S, O, F] =
  val action = writtenAction(written)
  val rules = mutable.ArrayBuffer.empty[Rule[S, O, F]]
  // The derived machine is not made yet, so the overlap names the machine it derives from.
  val machine = owner.machine.name match
    case ""     => "a derivation"
    case source => s"a derivation of $source"
  val firing = Firing[S, O, F, I](
    decl,
    None,
    action,
    (c, effect) =>
      val rule = Rule(rules.size + 1, c.heading, action, decl, None, c.guard, effect)
      for refused <- overlap(machine, owner.machine.fs, rules.toSeq, rule) do
        throw IllegalArgumentException(refused)
      rules += rule
  )
  cases(using firing)
  RuleGroup(rules.toVector)
