package framework.dsl

import scala.annotation.implicitNotFound

// The state an `is`, `effect` or `expect` block reads its fields of, by the accessors its state type
// declares, each of one shape: `def phase(using v: View[State]): Phase = v.get(_.phase)`.
// Core form: the state parameter of the predicate or step function, `s` in `s.phase`.
@implicitNotFound(
  "the fields of ${S} are read inside `is`, `effect` or `expect` blocks, which give the state they read: " +
    "outside a block, take the state as a parameter, `def f(s: State) = s.phase`"
)
sealed trait View[S]:
  private[framework] def state: S

  // The value of one field of the state: `v.get(_.phase)`. Core form: `s.phase`.
  def get[A](field: S => A): A = NativeRuntime.installed.invoke(field, state)

// The view an `is:` block reads: the state it is asked of. Core form: the predicate's parameter.
final private[framework] class Fixed[S](private[framework] val state: S) extends View[S]

// The step an expectation checks; bare fields read its resulting state. Core form:
// the parameter `s` of `(s: Step[S, O, F]) => predicate(s)`.
final class Expectation[S, O, F] private[framework] (val step: Step[S, O, F]) extends View[S]:
  private[framework] def state: S = step.state
  private[framework] val failure = new scala.util.control.ControlThrowable {}
  private[framework] def require(condition: Boolean): Unit =
    NativeRuntime.installed.requireExpectation(this, condition)
  private[framework] def evaluate(body: Expectation[S, O, F] ?=> Unit): Boolean =
    NativeRuntime.installed.expectation(this, body)

// The earlier state, available only in a transition expectation. Core form:
// `b` in `(b: S, a: Step[S, O, F]) => predicate(b, a)`.
@implicitNotFound("before is the state a step started from; use it in a postcondition")
final class Before[S] private[framework] (val state: S)

// Explicit alternatives of one requirement. Core form: `first || second || rest`.
final class Alternatives[S, O, F] private[framework] (private[framework] val step: Step[S, O, F]):
  private[framework] val success = new scala.util.control.ControlThrowable {}
  private[framework] def attempt(body: Expectation[S, O, F] ?=> Unit): Unit =
    NativeRuntime.installed.attemptAlternative(this, body)
  private[framework] def evaluate(body: Alternatives[S, O, F] ?=> Unit): Boolean =
    NativeRuntime.installed.alternatives(this, body)

// What an `effect:` block makes of one step while it runs: the state it enters, which each field
// assignment replaces with a copy, by the setters its state type declares, each of one shape:
// `def phase_=(p: Phase)(using d: Draft[State, ?, ?]): Unit = d.set(_.copy(phase = p))`; the facts
// it records; and the outcome it rejects with, if any. Each call of the effect makes a fresh draft,
// which never leaves the block. Core form: the arguments of the step the step function returns,
// `Step(outcome, s.copy(phase = p), List(facts*))`.
@implicitNotFound(
  "this assigns a field of ${S}, records a fact or rejects, which only an `effect:` block does: " +
    "outside one, return the steps, `enter(s.copy(...), fact)` or `reject(outcome, s)`"
)
final class Draft[S, O, F] private[framework] (start: S) extends View[S], Refusing[O], Recording[F]:
  private[framework] var current: S = start // scalafix:ok DisableSyntax.var
  private[framework] var facts: List[F] = Nil // scalafix:ok DisableSyntax.var
  private[framework] var rejection: Option[O] = None // scalafix:ok DisableSyntax.var

  private[framework] def state: S = current

  // The state with one field replaced: `d.set(_.copy(phase = p))`. Core form: `s.copy(phase = p)`.
  def set(update: S => S): Unit = NativeRuntime.installed.setDraft(this, update)

  // The step the block made, from the state `s` it started in: the rejection with `s`, or the ok
  // outcome with the state assigned and the facts in the order recorded.
  private[framework] def step(s: S, accepted: O): Step[S, O, F] =
    NativeRuntime.installed.draftStep(this, s, accepted)

  private[framework] def add(recorded: Seq[F]): Unit =
    NativeRuntime.installed.recordDraft(this, recorded)

  private[framework] def refuse(outcome: O): Unit =
    NativeRuntime.installed.refuseDraft(this, outcome)
