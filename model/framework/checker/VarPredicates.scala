package framework.checker

import framework.dsl.*

private[framework] object VarPredicates:
  // A same-step predicate of this Machine, read with the step's state restored into the vars.
  def sameStep[O, F](
      vars: Vars,
      predicate: Step[State, O, F] => Boolean
  ): Step[State, O, F] => Boolean =
    val restored = (step: Step[State, O, F]) => vars.at(step.state)(predicate(step))
    predicate match
      case bound: PredicateBindings => Bound1(restored, bound)
      case _                        => restored

  // A transition predicate of this Machine, read with the state a step went to restored into the
  // vars; the state it started from is its first argument.
  def transition[O, F](
      vars: Vars,
      predicate: (State, Step[State, O, F]) => Boolean
  ): (State, Step[State, O, F]) => Boolean =
    val restored = (start: State, step: Step[State, O, F]) =>
      vars.at(step.state)(predicate(start, step))
    predicate match
      case bound: PredicateBindings => Bound2(restored, bound)
      case _                        => restored

  final private class Bound1[A](restored: A => Boolean, bound: PredicateBindings)
      extends (A => Boolean),
        PredicateBindings:
    def apply(value: A): Boolean = restored(value)
    def validate(owner: Model): Unit = bound.validate(owner)
    def sourceParameters: Vector[String] = bound.sourceParameters
    override def captured: Option[Captured[AnyRef]] = bound.captured

  final private class Bound2[A, B](restored: (A, B) => Boolean, bound: PredicateBindings)
      extends ((A, B) => Boolean),
        PredicateBindings:
    def apply(first: A, second: B): Boolean = restored(first, second)
    def validate(owner: Model): Unit = bound.validate(owner)
    def sourceParameters: Vector[String] = bound.sourceParameters
    override def captured: Option[Captured[AnyRef]] = bound.captured
