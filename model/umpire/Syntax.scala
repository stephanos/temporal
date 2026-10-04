/* Sugar: definitions whose meaning a core declaration already expresses, kept for readability. Each
 * names the core form it stands for, and the lifter (model/lifter/Syntax.scala) lowers it to the IR
 * that core form lifts to. No other file of the framework uses them.
 */
package umpire

/**
 * The outcome `accept` and `stay` answer for a machine whose outcomes are `O`, declared once beside
 * the outcome type: `given Accepted[Outcome] = Accepted(Outcome.accepted)`. Core form: the outcome
 * itself, `Outcome.accepted`, written in each `Step`.
 */
final case class Accepted[O](outcome: O)

/**
 * One step with the accepted outcome into `state`, recording `facts`. Core form:
 * `List(Step(Outcome.accepted, state, List(facts*)))`.
 */
def accept[S, O, F](state: S, facts: F*)(using accepted: Accepted[O]): List[Step[S, O, F]] =
  List(Step(accepted.outcome, state, facts.toList))

/**
 * One step with the accepted outcome that keeps the state and records nothing. Core form:
 * `List(Step(Outcome.accepted, s))`.
 */
def stay[S, O, F](s: S)(using accepted: Accepted[O]): List[Step[S, O, F]] =
  List(Step(accepted.outcome, s))

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
