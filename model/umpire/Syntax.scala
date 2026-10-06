/* Sugar: definitions whose meaning a core declaration already expresses, kept for readability. Each
 * names the core form it stands for, and the IR generator (model/irgen/Syntax.scala) lowers it to
 * the IR that core form lifts to. No other file of the framework uses them.
 */
package umpire

import scala.annotation.{implicitNotFound, targetName, unused}
import scala.collection.mutable
import scala.compiletime.codeOf

/**
 * The outcome `enter` and `stay` answer for a machine whose outcomes are `O`, declared once beside
 * the outcome type: `given Ok[Outcome] = Ok(Outcome.accepted)`. Core form: the outcome itself,
 * `Outcome.accepted`, written in each `Step`.
 */
final case class Ok[O](outcome: O)

/**
 * One step with the ok outcome into `state`, recording `facts`. Core form:
 * `List(Step(Outcome.accepted, state, List(facts*)))`.
 */
def enter[S, O, F](state: S, facts: F*)(using ok: Ok[O]): List[Step[S, O, F]] =
  List(Step(ok.outcome, state, facts.toList))

/**
 * One step with the ok outcome that keeps the state and records nothing. Core form:
 * `List(Step(Outcome.accepted, s))`.
 */
def stay[S, O, F](s: S)(using ok: Ok[O]): List[Step[S, O, F]] =
  List(Step(ok.outcome, s))

/**
 * One step with the outcome `outcome` that keeps the state and records nothing, such as a request
 * the machine refuses: `reject(Outcome.notFound, s)`. Core form: `List(Step(Outcome.notFound, s))`.
 */
def reject[S, O](outcome: O, s: S): List[Step[S, O, Nothing]] = List(Step(outcome, s))

/** No step: the action is disabled here. Core form: `Nil`. */
val disabled: List[Nothing] = Nil

/**
 * `phase.in(a, b, c)`: whether the value is one of the members listed, of which there is at least
 * one. Written dotted, never infix. Core form: `List(a, b, c).contains(phase)`.
 */
extension [A](value: A) def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)

/**
 * `a implies b`: `b` holds wherever `a` does. `b` is read only where `a` holds, so a hole it reaches
 * is not reached where `a` is false. Core form: `!a || b`.
 */
extension (a: Boolean) infix def implies(b: => Boolean): Boolean = !a || b

/**
 * `after.records(fact)`: whether the step records the fact, which is of the step's own fact type:
 * a step's facts are covariant, so the evidence keeps a fact of another type from widening it.
 * Core form: `after.facts.contains(fact)`.
 */
extension [S, O, F](step: Step[S, O, F])
  def records[G](fact: G)(using G <:< F): Boolean = step.facts.contains(fact)

/**
 * `after.records(_.member, fact)`: whether a composition's step records the fact of the member the
 * selector names, the composition form of `after.records(fact)`. Core form:
 * `after.facts.contains("member_fact")`, the composed key `<field>_<fact>` a composition records.
 */
extension [S, O](step: Step[S, O, String])
  def records[M](@unused member: S => M, fact: Any): Boolean =
    step.facts.exists(_.endsWith(s"_$fact"))

/**
 * `m.property("terminalIsFinal").once(terminal).keeps(_.phase)`: once `over` holds of the state
 * before a step, the step keeps the value the projection selects. Core form:
 * `holdsAcross((before, after) => !over(before) || after.state.phase == before.phase)`.
 */
extension [S, O, F](b: PropertyBuilder[S, O, F])
  def once(over: S => Boolean): Once[S, O, F] = Once(b, over)

  /**
   * `never(to)`: no step is one `to` holds of, the same-step invariant. Core form:
   * `holds(after => !to(after))`. `never(to).from(before)` says it of the steps from a state `before`
   * holds of only.
   */
  def never(to: Step[S, O, F] => Boolean): Never[S, O, F] = Never(b, to)

  /**
   * `stays(p)`: a step from a state `p` holds of keeps it holding. Core form:
   * `holdsAcross((before, after) => !p(before) || p(after.state))`. `stays(p).unless(release)` also
   * lets a step `release` holds of leave it.
   */
  def stays(p: S => Boolean): Stays[S, O, F] = Stays(b, p)

/**
 * What `once(over)` leaves to say: the value it keeps. Core form: the `holdsAcross` lambda `keeps`
 * finishes.
 */
final class Once[S, O, F] private[umpire] (b: PropertyBuilder[S, O, F], over: S => Boolean):
  /**
   * The value a step from a state `over` holds of keeps, a field path such as `_.phase` or
   * `_.order.phase`. Core form:
   * `holdsAcross((before, after) => !over(before) || after.state.phase == before.phase)`.
   */
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

/**
 * `never(to)`, a same-step Property, which `from` turns into a transition one. Core form:
 * `holds(after => !to(after))`.
 */
final class Never[S, O, F] private[umpire] (
    b: PropertyBuilder[S, O, F],
    to: Step[S, O, F] => Boolean
) extends Property[S](
      PropertyDecl(b.name, b.m, b.when, Some((after: Step[S, O, F]) => !to(after)), None)
    ):
  /**
   * No step from a state `before` holds of is one `to` holds of. Core form:
   * `holdsAcross((before, after) => !before(before) || !to(after))`.
   */
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

/**
 * `stays(p)`, a transition Property, which `unless` releases. Core form:
 * `holdsAcross((before, after) => !p(before) || p(after.state))`.
 */
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
  /**
   * A step `release` holds of may leave `p`. Core form:
   * `holdsAcross((before, after) => !p(before) || p(after.state) || release(after))`.
   */
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

/**
 * A named slot with the value it receives, what `slot := value` writes. Core form: the value itself,
 * written at the slot's place, `expires` in `start(unset, expires, unset)`.
 */
final case class Assigned[A] private[umpire] (slot: Slot[A], value: A)

/**
 * `slot := value`: this named slot receives this value, and nothing else. The value has the slot's
 * type. Core form: the value at the slot's place in the positional call, `expires` in
 * `start(unset, expires, unset)`.
 */
extension [A](slot: Slot[A])
  @targetName("set")
  def :=(value: A): Assigned[A] = Assigned(slot, value)

/**
 * `start(scheduleToStart := expires)`: the class of the inputs the call supplies by name, in the
 * order the action declares them, each input it omits at its domain's first value. Each named slot
 * is an input token the action declares, supplied once. Core form: the positional call,
 * `start(unset, expires, unset)`.
 */
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

/**
 * `val retainedOutcome = sticky(outcomePreserved)`: a monitor of a promise about every step that,
 * once a step breaks it, stays broken, named after its `val`. Its verdict is read after every step.
 * Core form:
 * `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(after))(broken => broken)`.
 */
def sticky[S, O, F](promise: Step[S, O, F] => Boolean): Monitor[S, O, F, Boolean] =
  monitor[S, O, F, Boolean](false)((broken, _, after) => broken || !promise(after))(broken =>
    broken
  )

/**
 * `val ownerAcknowledgment = stickyAcross(ackOnlyWhenKept)`: `sticky` of a promise about the state
 * before a step and the step. Core form:
 * `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(before, after))(broken => broken)`.
 */
def stickyAcross[S, O, F](promise: (S, Step[S, O, F]) => Boolean): Monitor[S, O, F, Boolean] =
  monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(before, after))(
    broken => broken
  )

/**
 * A case of an `on` block: where in the state space its action fires, `in(placed)`, `in(open)`,
 * `where(g)`, `always` or `in(open).where(g)`, and through `~>` the effect it has there,
 * `in(placed) ~> effects.ship`. Core form: the arm `if g(s) then e(s, inputs) else ...` of the
 * action's step function.
 */
final class Case[S, O, F] private[umpire] (
    private[umpire] val heading: String,
    private[umpire] val guard: S => Boolean
):
  /**
   * This case where `condition` also holds of the state, beyond its phases:
   * `in(open).where(_.deadline == Timeout.expires)`. Core form: `g(s) && condition(s)`.
   */
  def where(condition: S => Boolean): Case[S, O, F] =
    Case(s"$heading.where", s => guard(s) && condition(s))

  /**
   * The effect of the block's action where this case holds, read of the state alone:
   * `in(placed) ~> effects.ship`. Core form: the arm `if g(s) then ship(s)` of the action's step
   * function.
   */
  infix def ~>(effect: S => List[Step[S, O, F]])(using firing: Firing[S, O, F, ?]): Unit =
    firing.bind(this, (s, _) => effect(s))

  /**
   * The effect of the block's action, of one input, where this case holds, read of the state and
   * the input: `in(open) ~> effects.change`. Core form: the arm `if g(s) then change(s, c)`.
   */
  @targetName("readsOne")
  infix def ~>[A](effect: (S, A) => List[Step[S, O, F]])(using
      firing: Firing[S, O, F, A *: EmptyTuple]
  ): Unit = firing.bind(this, effectOf(firing.decl, effect))

  /**
   * The effect of the block's action, of two inputs, read of the state and its inputs. Core form:
   * the arm `if g(s) then e(s, x, y)`.
   */
  @targetName("readsTwo")
  infix def ~>[A, B](effect: (S, A, B) => List[Step[S, O, F]])(using
      firing: Firing[S, O, F, (A, B)]
  ): Unit = firing.bind(this, effectOf(firing.decl, effect))

  /**
   * The effect of the block's action, of three inputs, read of the state and its inputs:
   * `in(unplaced) ~> effects.open`. Core form: the arm `if g(s) then open(s, x, y, z)`.
   */
  @targetName("readsThree")
  infix def ~>[A, B, C](effect: (S, A, B, C) => List[Step[S, O, F]])(using
      firing: Firing[S, O, F, (A, B, C)]
  ): Unit = firing.bind(this, effectOf(firing.decl, effect))

/**
 * The action, or the one class of it, an `on` block fires, and where its cases go, given to the
 * block's cases. Core form: none of its own; it is the `action` of `action ~> stepFunction`, whose
 * arms the block's cases are.
 */
final class Firing[S, O, F, I <: Tuple] private[umpire] (
    private[umpire] val decl: ActionDecl,
    private[umpire] val values: Option[List[Any]],
    private[umpire] val action: String,
    add: (Case[S, O, F], (S, List[Any]) => List[Step[S, O, F]]) => Unit
):
  private[umpire] def bind(c: Case[S, O, F], effect: (S, List[Any]) => List[Step[S, O, F]]): Unit =
    add(c, effect)

/**
 * A case that holds where `condition` holds of the state: `where(states.held) ~> effects.settle`.
 * Core form: `if condition(s) then e(s) else ...`.
 */
def where[S, O, F](using firing: Firing[S, O, F, ?])(condition: S => Boolean): Case[S, O, F] =
  val _ = firing
  Case("where", condition)

/**
 * A case that holds in every state: `always ~> effects.keep`. Core form:
 * `if true then e(s) else ...`.
 */
def always[S, O, F](using firing: Firing[S, O, F, ?]): Case[S, O, F] =
  val _ = firing
  Case("always", _ => true)

/**
 * When each action of a machine fires, written as the machine object's `object rules extends
 * Rules`, or `Rules(_.phase)` where its cases name phases: one block per action or action class,
 * `on(clerk.ship) { in(placed) ~> effects.send }`, whose cases each say where the action fires,
 * `in(...)` of phases or of a named set of them, `where(g)` of the state, `in(...).where(g)` of
 * both or `always`, and what it does there, `~> effects.x`; and the actions no state enables,
 * `disabled(courier.strike)`. A block fires a whole action, `on(buyer.change)`, or one class of it,
 * `on(buyer.change(Change.hold))`, whose effects then read the state alone; an effect that takes
 * arguments beyond the state binds them in place, `effects.timeOut(_, Deadline.close)`. Where no
 * case holds, the action is disabled: there is no catch-all. The cases of one action class hold in
 * no common state: each is checked against the earlier ones as it is declared, over every state of
 * the `Finite` state type and every class, and an overlap is refused naming the machine, the class,
 * both rules and a state where both hold. Core form: one step function per action, the cases of the
 * action in order,
 * `action ~> ((s, i) => if g1(s) && c1(i) then e1(s, i) else if g2(s) then e2(s, i) else Nil)`, and
 * `action ~> (_ => Nil)` for an action no state enables.
 *
 * `on` is `inline`, with an `inline` action: the one sanctioned exception to "no inline on the
 * author surface" (.plans/DSL_OPERATORS.md, rule 5). An action's runtime name is `""`, since the
 * lifter names it after its `val`, so `scala.compiletime.codeOf` names the action in an overlap
 * message. TASTy is pickled before inlining, so the lifter reads the unexpanded call, and it refuses
 * an expanded one loudly ("not a rule") rather than lift it.
 */
abstract class Rules[S, O, F, P](using owner: Owner[S, O, F])(
    phase: S => P = (_: S) => throw IllegalStateException("these rules declare no projection")
) extends RuleBook[S, O, F]:
  private val written = mutable.ArrayBuffer.empty[Rule[S, O, F]]
  private val never = mutable.ArrayBuffer.empty[ActionDecl]
  private val order = mutable.LinkedHashSet.empty[ActionDecl]
  private val blocks = mutable.ArrayBuffer.empty[(ActionDecl, Option[List[Any]])]
  private var open: Option[String] = None // scalafix:ok DisableSyntax.var

  /** Runs one block's cases, refusing a block in a block and a second block of one target. */
  private[umpire] def block[I <: Tuple](
      decl: ActionDecl,
      values: Option[List[Any]],
      code: String
  )(cases: Firing[S, O, F, I] ?=> Unit): Unit =
    val action = writtenAction(code)
    require(open.isEmpty, s"on($action) sits in on(${open.get}): a block holds cases alone")
    require(!never.contains(decl), s"$action is disabled and fired by a rule")
    require(
      !blocks.contains(decl -> values),
      s"on($action) is written twice: an action, or a class of it, has one block"
    )
    blocks += decl -> values
    open = Some(action)
    val firing =
      Firing[S, O, F, I](decl, values, action, (c, effect) => bind(decl, values, c, effect, code))
    try cases(using firing)
    finally open = None

  /** Registers a rule, refusing one that overlaps an earlier rule of its class. */
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

  /**
   * The cases of a whole action: `on(clerk.ship) { in(placed) ~> effects.send }`. Core form: the
   * action's step function, whose arms are the cases in order.
   */
  inline def on[I <: Tuple](inline a: Action[I])(cases: Firing[S, O, F, I] ?=> Unit): Unit =
    block[I](a.decl, None, codeOf(a))(cases)

  /**
   * The cases of one class of an action, whose effects read the state alone:
   * `on(courier.report(Report.delivered)) { in(sent) ~> effects.deliver }`. Core form: the arms
   * `if g(s) && result == Report.delivered then deliver(s)` of the action's step function.
   */
  inline def on(inline c: Class)(cases: Firing[S, O, F, EmptyTuple] ?=> Unit): Unit =
    block[EmptyTuple](c.decl, Some(c.values), codeOf(c))(cases)

  /**
   * A case that holds in the phases listed, as the projection of `Rules(_.phase)` reads them; rules
   * that declare no projection name no phase. Core form: `List(p1, p2).contains(s.phase)`.
   */
  def in[Q](first: Q, rest: Q*)(using PhasesOf[P, Q]): Case[S, O, F] =
    val phases = first +: rest
    Case(s"in(${phases.mkString(", ")})", s => phases.contains(phase(s)))

  /**
   * A case that holds in a named set of phases, a predicate of the projection the machine's
   * `states` declares: `in(states.terminal)`. Core form: `terminal(s.phase)`.
   */
  inline def in(inline set: P => Boolean): Case[S, O, F] = phases(set, codeOf(set))

  private[umpire] def phases(set: P => Boolean, code: String): Case[S, O, F] =
    Case(s"in(${code.trim})", s => set(phase(s)))

  /**
   * Actions no state enables, which the machine binds all the same, such as a courier's strike it does
   * not feel. Core form: `action ~> (_ => Nil)`.
   */
  def disabled(actions: Action[?]*): Unit =
    for a <- actions do
      require(
        !never.contains(a.decl) && !written.exists(_.decl == a.decl),
        "an action is disabled twice, or disabled and bound by a rule"
      )
      never += a.decl
      order += a.decl

  /**
   * `phase.in(a, b)` in a condition, as outside the rules. Core form: `List(a, b).contains(phase)`.
   */
  extension [A](value: A) def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)

  private[umpire] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    order.toVector.map: decl =>
      decl -> (if never.contains(decl) then Bound.Disabled[S, O, F]()
               else Bound.Ruled(written.filter(_.decl == decl).toVector))

/**
 * Evidence that a case's phases are of the type the rules' projection reads: the rules of
 * `object rules extends Rules(_.phase)` name phases with `in`, and rules that declare no projection
 * name none. Core form: none of its own; `in(p1, p2)` is `List(p1, p2).contains(s.phase)`.
 */
@implicitNotFound(
  "in names phases of ${Q}, and these rules project the state onto ${P}: declare the projection the " +
    "phases are of, `object rules extends Rules(_.phase)`"
)
final class PhasesOf[P, Q] private ()

/**
 * The one projection a case's phases are of: its own. Core form: `List(p1, p2).contains(s.phase)`.
 */
object PhasesOf:
  /**
   * Phases of the projection's own type. Core form: `List(p1, p2).contains(s.phase)`, whose phases
   * are of the type of `s.phase`.
   */
  given [P]: PhasesOf[P, P] = PhasesOf()

/**
 * The rules of one action a derivation binds in its source's place:
 * `rebind(on(clerk.ship) { always ~> OrderRecord.effects.send })`, each case `where(g)` or
 * `always`, since a derivation's rules name no phase. Its action is `inline`, as a rule block's is
 * (`Rules`), so that an overlap names it as written. Core form: the step function
 * `action ~> ((s, i) => if g(s) then e(s, i) else Nil)`.
 */
inline def on[S, O, F, I <: Tuple](using
    owner: Owner[S, O, F]
)(inline a: Action[I])(
    cases: Firing[S, O, F, I] ?=> Unit
): RuleGroup[S, O, F] = derivedRules(owner, a.decl, codeOf(a))(cases)

/** The rules of a derivation's `on`, each named by its action as `written` spells it. */
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
