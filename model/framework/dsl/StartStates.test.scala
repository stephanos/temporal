package framework.dsl

import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object StartStatesFixture:

  object switches extends Actor:
    val finish = action

  object Model extends Machine[Outcome, Nothing]:
    var count = oneOf(List(0, 2, 2))
    var ready = oneOf(List(false, true))
    var finished = false

    object transitions extends Transitions:
      on(switches.finish):
        if ready && !finished then effects.finish()

    object effects:
      def finish() =
        finished = true

    object invariants:
      val belowTwo = check(count < 2)

  class Empty extends Machine[Outcome, Nothing]:
    var count = oneOf(List.empty[Int])

    object transitions extends Transitions

class StartStatesTest extends munit.FunSuite:
  import StartStatesFixture.*

  test("declared alternatives form distinct starts in declaration and domain order"):
    assertEquals(
      Model.initialStates,
      Seq(
        Model.init.copy(count = 0, ready = false),
        Model.init.copy(count = 0, ready = true),
        Model.init.copy(count = 2, ready = false),
        Model.init.copy(count = 2, ready = true)
      )
    )
    assertEquals(Model.at(Model.init)(Model.count), 0)
    assertEquals(Model.at(Model.init)(Model.ready), false)

  test("default checking visits every start and finds failures absent from init"):
    val report = NativeCheck.verifyAll(Model)
    assertEquals(report.result("belowTwo").verdict, NativeCheck.Verdict.refuted)
    val graph = NativeCheck.explore(Model, Model.initialStates)
    graph.drain()
    assertEquals(graph.starts.size, 4)
    assertEquals(graph.states.size, 6)

  test("a derived machine keeps its source's alternatives"):
    val derived = Model.restrict(switches.finish)
    assertEquals(derived.initialStates, Model.initialStates)
    assertEquals(
      NativeCheck.verifyAll(derived, sections = Some(Model)).result("belowTwo").verdict,
      NativeCheck.Verdict.refuted
    )

  test("an empty initial domain fails before supplying a start"):
    val error = intercept[IllegalArgumentException](new Empty)
    assert(error.getMessage.contains("count: oneOf takes a non-empty domain"))

  test("oneOf domains are state initializers, never helper declarations or effects"):
    val helper = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      import framework.dsl.outcomes.Outcome
      object Invalid extends Machine[Outcome, Nothing]:
        var count = 0
        val alternatives = oneOf(List(1, 2))
    """)
    assert(helper.nonEmpty)
