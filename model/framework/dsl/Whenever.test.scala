package framework.dsl

import framework.checker.*

import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

// Progress as `whenever(from): eventually(to)`: a bound only where the claim states one, weak
// fairness that only an unbounded claim can use, and `without`, the environment that drops actions.
object WheneverFixture:

  object worker extends Actor:
    val finish = action

  object clock extends Internal:
    val tick = step

  // A worker that may finish while a clock ticks forever: the ticks are a cycle that avoids
  // `done`, which only a fair finish leaves.
  object Model extends Machine[Outcome, Nothing]:
    var done = false
    var odd = false

    object transitions extends Transitions:
      on(worker.finish):
        if !done then effects.finish()
      on(clock.tick):
        effects.tick()

    object effects:
      def finish() =
        done = true
      def tick() =
        odd = !odd

    val fairFinish =
      fair(worker.finish)

    object progress:
      val finishes =
        (whenever(!done):
          eventually(done)
        ) in fairFinish
      val finishesAtOnce =
        (whenever(!done):
          eventually(done, within = 1)
        ) in fairFinish
      val finishesUnfairly =
        whenever(!done):
          eventually(done)

class WheneverSuite extends munit.FunSuite:
  import WheneverFixture.*
  import NativeCheck.Verdict.*

  test("an unbounded claim holds where a fair action leaves every cycle, and a bound does not"):
    val report = NativeCheck.verifyAll(Model)
    assertEquals(
      report.verdicts,
      Map(
        "finishes" -> verified,
        "finishesAtOnce" -> missedDeadline,
        "finishesUnfairly" -> cycle
      )
    )
    // the ticks may go on for any number of steps before the fair finish, so no path is longest
    assertEquals(report.result("finishes").longest, None)
    assertEquals(
      report.result("finishesAtOnce").detail,
      Vector("model.finishesAtOnce: a cycle can avoid a to state for any number of steps")
    )

  test("whenever builds the claim leadsTo does, with the bound eventually states"):
    val finishes = Model.progress.finishes
    assertEquals(finishes.name, "finishes")
    assert(finishes.machine.exists(_ eq Model))
    assertEquals(finishes.within, None)
    assertEquals(finishes.scenario.name, "fairFinish")
    assertEquals(Model.progress.finishesAtOnce.within, Some(1))
    assertEquals(Model.progress.finishesUnfairly.scenario.decl.fair, Vector.empty)

  test("without admits every action but those it names, and not beside only or actions"):
    val dropped = without(worker.finish).decl
    assertEquals(dropped.admits(worker.finish.decl), false)
    assertEquals(dropped.admits(clock.tick.decl), true)
    assertEquals(dropped.modelOnly, true)
    intercept[IllegalArgumentException](only(clock.tick).without(worker.finish)): Unit

  test("without one input class leaves the action's other classes admitted"):
    import ScenarioClassesFixture.{calls, Model as Deciding, worker as decider}
    calls.clear()
    val result = NativeCheck.query(
      verify(Deciding.invariants.unfinished) in without(decider.decide(true)),
      "model.unfinished"
    )
    assertEquals(result.verdict, verified)
    assert(result.complete)
    assert(calls.nonEmpty)
    assert(calls.forall(_ == false), calls.toString)

  test("eventually belongs in a whenever block, and whenever in progress"):
    val machine =
      "object M extends Machine[framework.dsl.outcomes.Outcome, Nothing]:\n  var x = 1\n"
    val stray = compileErrors(machine + "  object progress:\n    val a = eventually(x == 2)\n")
    assert(stray.contains("eventually is the deadline of a whenever block"), stray)
    val misplaced = compileErrors(
      machine + "  object invariants:\n    val a = whenever(x == 1):\n      eventually(x == 2)\n"
    )
    assert(misplaced.contains("whenever, a progress claim, belongs in progress"), misplaced)
    val zero = compileErrors(
      machine + "  object progress:\n    val a = whenever(x == 1):\n      eventually(x == 2, within = 0)\n"
    )
    assert(zero.contains("within is a positive number of steps, written as a literal"), zero)
