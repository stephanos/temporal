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

/** `after.records(fact)`: whether the step records the fact. Core form: `after.facts.contains(fact)`. */
extension [S, O, F](step: Step[S, O, F]) def records(fact: F): Boolean = step.facts.contains(fact)

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
 * When each action of a machine fires, written as the machine object's `object rules extends
 * Rules`, or `Rules(_.phase)` where its rules name phases: one rule per line under a heading,
 * `in(placed) { clerk.ship ~> effects.send }`, and the actions no state enables,
 * `disabled(courier.strike)`. A rule fires a whole action, `buyer.change ~> effects.notFound`,
 * or one class of it, `buyer.change(Change.hold) ~> effects.hold`, whose effect then reads the
 * state alone. The rules of one action class hold in no common state: each rule is checked against
 * the earlier ones as it is declared, over every state of the `Finite` state type and every class,
 * and an overlap is refused naming the machine, the class, both rules and a state where both hold.
 * Core form: one step function per action, the rules of the action in order,
 * `action ~> ((s, i) => if g1(s) && c1(i) then e1(s, i) else if g2(s) then e2(s, i) else Nil)`, and
 * `action ~> (_ => Nil)` for an action no state enables.
 */
abstract class Rules[S, O, F, P](using owner: Owner[S, O, F])(
    phase: S => P = (_: S) => throw IllegalStateException("these rules declare no projection")
) extends RuleBook[S, O, F]:
  private val written = mutable.ArrayBuffer.empty[Rule[S, O, F]]
  private val never = mutable.ArrayBuffer.empty[ActionDecl]
  private val order = mutable.LinkedHashSet.empty[ActionDecl]
  private var heading: Option[(String, S => Boolean)] = None // scalafix:ok DisableSyntax.var

  private def under(name: String, guard: S => Boolean)(rules: => Unit): Unit =
    require(heading.isEmpty, s"$name sits under ${heading.get._1}: a rule has one heading")
    heading = Some(name -> guard)
    try rules
    finally heading = None

  /** Registers a rule, refusing one that overlaps an earlier rule of its class. */
  private[umpire] def bind(
      decl: ActionDecl,
      values: Option[List[Any]],
      effect: (S, List[Any]) => List[Step[S, O, F]],
      code: String
  ): Unit =
    val (name, guard) = heading.getOrElse(
      throw IllegalArgumentException(
        s"${writtenAction(code)} is bound with no heading: write it under `when(...)` or `in(...)`"
      )
    )
    require(!never.contains(decl), s"${writtenAction(code)} is disabled and bound by a rule")
    val rule = Rule(written.size + 1, name, writtenAction(code), decl, values, guard, effect)
    for refused <- overlap(owner.machine.name, owner.machine.fs, written.toSeq, rule) do
      throw IllegalArgumentException(refused)
    written += rule
    order += decl

  /**
   * The rules that fire while `guard` holds of the state. Core form: each rule's guard, `g(s)`, in
   * the condition of its arm of the lowered step function, `if g(s) then e(s) else ...`.
   */
  def when(guard: S => Boolean)(rules: => Unit): Unit = under("when", guard)(rules)

  /**
   * The rules that fire in the phases listed, as the projection of `Rules(_.phase)` reads them; rules
   * that declare no projection name no phase. Core form: `if List(p1, p2).contains(s.phase) then
   * e(s) else ...`.
   */
  def in[Q](first: Q, rest: Q*)(rules: => Unit)(using PhasesOf[P, Q]): Unit =
    val phases = first +: rest
    under(s"in(${phases.mkString(", ")})", s => phases.contains(phase(s)))(rules)

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
   * `phase.in(a, b)` in a guard, as outside the rules. Core form: `List(a, b).contains(phase)`.
   */
  extension [A](value: A) def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)

  /**
   * A rule of an action with no input: `timers.backoff ~> effects.retry`. Core form: the arm
   * `if g(s) then retry(s)` of the action's step function.
   */
  extension (inline a: Action[EmptyTuple])
    inline infix def ~>(effect: S => List[Step[S, O, F]]): Unit =
      bind(a.decl, None, (s, _) => effect(s), codeOf(a))

  /**
   * A rule of every class of an action with one input, whose effect reads it, or reads the state
   * alone: `buyer.change ~> effects.notFound`. Core form: the arm `if g(s) then e(s, c)` of the
   * action's step function.
   */
  extension [A](inline a: Action[A *: EmptyTuple])
    inline infix def ~>(effect: (S, A) => List[Step[S, O, F]]): Unit =
      bind(a.decl, None, effectOf(a.decl, effect), codeOf(a))

    /**
     * A rule of every class of an action with one input whose effect reads the state alone.
     * Core form: the arm `if g(s) then e(s)` of the action's step function.
     */
    @targetName("bindsOneState")
    inline infix def ~>(effect: S => List[Step[S, O, F]]): Unit =
      bind(a.decl, None, (s, _) => effect(s), codeOf(a))

  /**
   * A rule of every class of an action with two inputs. Core form: the arm `if g(s) then e(s, x, y)`
   * of the action's step function.
   */
  extension [A, B](inline a: Action[(A, B)])
    inline infix def ~>(effect: (S, A, B) => List[Step[S, O, F]]): Unit =
      bind(a.decl, None, effectOf(a.decl, effect), codeOf(a))

    /**
     * A rule of every class of an action with two inputs whose effect reads the state alone.
     * Core form: the arm `if g(s) then e(s)` of the action's step function.
     */
    @targetName("bindsTwoState")
    inline infix def ~>(effect: S => List[Step[S, O, F]]): Unit =
      bind(a.decl, None, (s, _) => effect(s), codeOf(a))

  /**
   * A rule of every class of an action with three inputs: `buyer.place ~> effects.open`.
   * Core form: the arm `if g(s) then e(s, x, y, z)` of the action's step function.
   */
  extension [A, B, C](inline a: Action[(A, B, C)])
    inline infix def ~>(effect: (S, A, B, C) => List[Step[S, O, F]]): Unit =
      bind(a.decl, None, effectOf(a.decl, effect), codeOf(a))

    /**
     * A rule of every class of an action with three inputs whose effect reads the state alone.
     * Core form: the arm `if g(s) then e(s)` of the action's step function.
     */
    @targetName("bindsThreeState")
    inline infix def ~>(effect: S => List[Step[S, O, F]]): Unit =
      bind(a.decl, None, (s, _) => effect(s), codeOf(a))

  /**
   * A rule of one class of an action, whose effect reads the state alone:
   * `courier.report(Report.delivered) ~> effects.deliver`. Core form: the arm
   * `if g(s) && result == AttemptResult.completed then complete(s)` of the action's step function.
   */
  extension (inline c: Class)
    inline infix def ~>(effect: S => List[Step[S, O, F]]): Unit =
      bind(c.decl, Some(c.values), (s, _) => effect(s), codeOf(c))

  private[umpire] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    order.toVector.map: decl =>
      decl -> (if never.contains(decl) then Bound.Disabled[S, O, F]()
               else Bound.Ruled(written.filter(_.decl == decl).toVector))

/**
 * Evidence that a heading's phases are of the type the rules' projection reads: the rules of
 * `object rules extends Rules(_.phase)` name phases with `in`, and rules that declare no projection
 * name none. Core form: none of its own; `in(p1, p2)` is `List(p1, p2).contains(s.phase)`.
 */
@implicitNotFound(
  "in names phases of ${Q}, and these rules project the state onto ${P}: declare the projection the " +
    "phases are of, `object rules extends Rules(_.phase)`"
)
final class PhasesOf[P, Q] private ()

/**
 * The one projection a heading's phases are of: its own. Core form: `List(p1, p2).contains(s.phase)`.
 */
object PhasesOf:
  /**
   * Phases of the projection's own type. Core form: `List(p1, p2).contains(s.phase)`, whose phases
   * are of the type of `s.phase`.
   */
  given [P]: PhasesOf[P, P] = PhasesOf()

/**
 * Rules a derivation binds in its source's place, under one heading:
 * `rebind(when(_ => true) { clerk.ship ~> OrderRecord.effects.send })`, each a whole
 * action's. Core form: the step function `action ~> ((s, i) => if g(s) then e(s, i) else Nil)`.
 */
def when[S, O, F](using
    owner: Owner[S, O, F]
)(guard: S => Boolean)(
    bindings: StepBinding[S, O, F]*
): RuleGroup[S, O, F] =
  val rules = bindings.zipWithIndex.toVector.map: (b, i) =>
    Rule(i + 1, "when", b.decl.name, b.decl, None, guard, effectOf[S, O, F](b.decl, b.function))
  for
    (r, i) <- rules.zipWithIndex;
    refused <- overlap(
      owner.machine.name,
      owner.machine.fs,
      rules.take(i),
      r
    )
  do throw IllegalArgumentException(refused)
  RuleGroup(rules)
