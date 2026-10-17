package framework.dsl

import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object ScenarioClassesFixture:
  val calls = scala.collection.mutable.ArrayBuffer.empty[Boolean]

  object worker extends Actor:
    val decide = action.input[Boolean](name = "finish")

  object Model extends Machine[Outcome, Nothing]:
    var finished = false

    object transitions extends Transitions:
      on(worker.decide): finish =>
        effects.decide(finish)

    object effects:
      def decide(finish: Boolean) =
        calls += finish: Unit
        finished = finished || finish

    object invariants:
      val unfinished =
        check(!finished)

    object progress:
      val finishes =
        leadsTo(from = !finished, to = finished, within = 3)

    object queries:
      val finishing =
        after(worker.decide):
          check(finished)

class ScenarioClassesSuite extends munit.FunSuite:
  import ScenarioClassesFixture.*
  import NativeCheck.Verdict.*

  test("only one input class excludes sibling classes from safety checks"):
    calls.clear()
    val result = NativeCheck.query(
      verify(Model.invariants.unfinished) in only(worker.decide(false)),
      "model.unfinished"
    )
    assertEquals(result.verdict, verified)
    assert(result.complete)
    assert(calls.nonEmpty)
    assert(calls.forall(_ == false), calls.toString)

  test("a find cannot use an excluded input class"):
    val result = NativeCheck.query(
      find(Model.queries.finishing) in only(worker.decide(false)),
      "model.finishing"
    )
    assertEquals(result.verdict, refuted)
    assert(result.complete)

  test("a whole action still admits every input class"):
    val result = NativeCheck.query(
      find(Model.queries.finishing) in only(worker.decide),
      "model.finishing"
    )
    assertEquals(result.verdict, found)
    assertEquals(result.detail, Vector("witness: decide(true)"))

  test("only the finishing class verifies progress in one step"):
    val result = NativeCheck.query(
      verify(Model.progress.finishes) in only(worker.decide(true)),
      "model.finishes"
    )
    assertEquals(result.verdict, verified)
    assertEquals(result.longest, Some(1))

  test("a fair looping class does not make its finishing sibling fair"):
    val result = NativeCheck.query(
      verify(Model.progress.finishes) in fair(worker.decide(false)),
      "model.finishes"
    )
    assertEquals(result.verdict, cycle)

  test("a fair finishing class excludes the nonprogress cycle but cannot bound its delay"):
    val result = NativeCheck.query(
      verify(Model.progress.finishes) in fair(worker.decide(true)),
      "model.finishes"
    )
    assertEquals(result.verdict, missedDeadline)

  test("fairness of a whole action still applies to each input class"):
    val result = NativeCheck.query(
      verify(Model.progress.finishes) in fair(worker.decide),
      "model.finishes"
    )
    assertEquals(result.verdict, missedDeadline)

  test("only composes with starts and exact-class fairness"):
    val result = NativeCheck.query(
      verify(Model.invariants.unfinished) in
        only(worker.decide(false)).fair(worker.decide(false)).starts(finished = true),
      "model.unfinished"
    )
    assertEquals(result.verdict, refuted)
    assertEquals(result.detail, Vector("counterexample: "))
