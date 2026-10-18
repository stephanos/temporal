package samples

import framework.dsl.*
import framework.dsl.Finite.given
import framework.checker.{
  Checker,
  Expectation,
  ReplayExpectation,
  ReplayScript,
  ReplayStep,
  Verdict
}

object PublicQuestionsClient:
  enum Answer:
    case ok
  enum Holder:
    case one, two
  given Ok[Answer] = Ok(Answer.ok)

  object actor extends Actor:
    val advance = action

  class Counter[D: Finite](holder: D) extends Machine[Answer, Nothing]:
    var value = 0
    var owners = Set(holder)

    object transitions extends Transitions:
      on(actor.advance):
        if value < 2 then effects.advance()

    object effects:
      def advance(): Unit = value += 1

    object invariants:
      val belowOne =
        check(value < 1)

    object postconditions:
      def firstStep: Boolean = before.value == 0 && value == 1 && outcome == Answer.ok

    object monitors:
      val first = monitor(false)(_ => value == 1)(identity)

  object Original extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        if value < 2 then effects.advance()

    object effects:
      def advance(): Unit = value += 1

  object Other extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        if value < 2 then effects.advance()

    object effects:
      def advance(): Unit = value += 1

  object Changed extends Derived(Original)

  object Together extends Composition[Nothing]:
    val left = Original
    val right = Other

    object syncs:
      val advance = sync(left(actor.advance), right(actor.advance))

  def run(): Unit =
    val first = new Counter(Holder.one)
    val space = Checker.space(first, steps = 2)()
    val found = Checker.find(
      space,
      horizon = 1
    )(Expectation(101, "first step", _ => first.postconditions.firstStep))
    assert(found.verdict == Verdict.found)
    assert(found.subject == "first.find")
    assert(found.path.steps.size == 1)
    val alias = first
    val replayed = Checker.replay(
      alias,
      ReplayScript[State](Vector(
        ReplayStep(actor.advance()),
        ReplayExpectation(Expectation(102, "first step", _ => alias.postconditions.firstStep))
      ))
    )
    assert(replayed.verdict == Verdict.verified)
    assert(replayed.subject == "first.replay")
    assert(replayed.path.steps.forall(step => java.util.Objects.equals(step.outcome, Answer.ok)))
    val replayFirst = new Counter(Holder.two)
    val replayFirstResult =
      Checker.replay(replayFirst, ReplayScript[State](Vector(ReplayStep(actor.advance()))))
    assert(replayFirstResult.subject == "replayFirst.replay")
    val control = new Counter(Holder.one)
    val broken = Checker.refuted(control, control.invariants.belowOne)
    assert(broken.verdict == Verdict.refuted)
    assert(broken.subject == "control.belowOne")
    val watched = new Counter(Holder.one)
    val monitorResult = Checker.refuted(watched, watched.monitors.first)
    assert(monitorResult.verdict == Verdict.refuted)
    assert(monitorResult.subject == "watched.first")
    val objectResult =
      Checker.replay(Original, ReplayScript[State](Vector(ReplayStep(actor.advance()))))
    assert(objectResult.verdict == Verdict.verified)
    val derivedResult =
      Checker.replay(Changed, ReplayScript[State](Vector(ReplayStep(actor.advance()))))
    assert(derivedResult.verdict == Verdict.verified)
    val composed = Checker.find(Checker.space(Together, steps = 1)(), horizon = 1)(
      Expectation(103, "both advance", _ => Original.value == 1 && Other.value == 1)
    )
    assert(composed.verdict == Verdict.found)
    assert(composed.path.steps.size == 1)
    val outsider = new Counter(Holder.two)
    val refused =
      try
        val _ = Checker.refuted(control, outsider.invariants.belowOne)
        false
      catch case _: IllegalArgumentException => true
    assert(refused)

  def main(arguments: Array[String]): Unit =
    require(arguments.isEmpty)
    run()
