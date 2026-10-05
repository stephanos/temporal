/* Sugar: definitions whose meaning a core declaration already expresses, kept for readability. Each
 * names the core form it stands for, and the IR generator (model/irgen/Syntax.scala) lowers it to
 * the IR that core form lifts to. No other file of the framework uses them.
 */
package umpire

import scala.annotation.{targetName, unused}

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
