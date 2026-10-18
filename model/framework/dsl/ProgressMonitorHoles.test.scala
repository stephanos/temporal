package framework.dsl

import framework.checker.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object ProgressMonitorHolesFixture:
  val calls = scala.collection.mutable.ArrayBuffer.empty[Boolean]

  object worker extends Actor:
    val pulse = action.input[Boolean](name = "breaking")
    val waits = action
    val idle = action

  object Model extends Machine[Outcome, Nothing]:
    var visited = false
    var breaking = false
    val missing =
      hole

    object transitions extends Transitions:
      on(worker.pulse): breaking =>
        effects.pulse(breaking)

    object effects:
      def pulse(next: Boolean) =
        calls += next: Unit
        visited = true
        breaking = next

    object invariants:
      val known =
        check(true)

    object progress:
      val missingFrom =
        leadsTo(from = missing.reached, to = visited, within = 2)
      val missingTo =
        leadsTo(from = !visited, to = missing.reached, within = 2)
      val knownProgress =
        leadsTo(from = !visited, to = visited, within = 2)

    object monitors:
      val uncertain =
        monitor(false)(_ => missing.reached)(identity)
      val knownMonitor =
        monitor(false)(_ => false)(identity)

  object HoleSteps extends Machine[Outcome, Nothing]:
    var done = false
    val missing =
      hole

    object transitions extends Transitions:
      on(worker.waits):
        effects.waits()
      on(worker.idle):
        effects.idle()

    object effects:
      def waits() =
        if missing.reached then done = true
      def idle() =
        ()

    object progress:
      val finishes =
        leadsTo(from = !done, to = done, within = 2)

class ProgressMonitorHolesSuite extends munit.FunSuite:
  import ProgressMonitorHolesFixture.*
  import NativeCheck.Verdict.*

  test("a hole in a progress from predicate produces a named unknown result"):
    calls.clear()
    val result = NativeCheck.query(verify(Model.progress.missingFrom), "model.missingFrom")
    assertEquals(result.verdict, unknown)
    assert(!result.complete)
    assert(result.detail.exists(_.contains("missing")))
    assert(result.detail.exists(_.contains(Model.init.toString)))
    assert(calls.isEmpty, calls.toString)

  test("a hole in a progress to predicate produces unknown"):
    calls.clear()
    val result = NativeCheck.query(verify(Model.progress.missingTo), "model.missingTo")
    assertEquals(result.verdict, unknown)
    assert(!result.complete)
    assert(calls.isEmpty, calls.toString)

  test("a monitor next-state hole produces unknown"):
    val result =
      NativeCheck.monitor(Model, "model.uncertain", Model.monitors.uncertain, Model.initialStates)
    assertEquals(result.verdict, unknown)
    assert(!result.complete)
    assert(result.detail.exists(_.contains("missing")))

  test("a monitor verdict hole produces unknown"):
    val watch = Model.monitor(false)(_ => false)(_ => Model.missing.reached)
    val result = NativeCheck.monitor(Model, "model.watch", watch, Model.initialStates)
    assertEquals(result.verdict, unknown)

  test("a monitor read selector hole produces unknown"):
    val watch = Model.monitors.knownMonitor.readAfter(_ => Model.missing.reached)
    val result = NativeCheck.monitor(Model, "model.watch", watch, Model.initialStates)
    assertEquals(result.verdict, unknown)

  test("a monitor end verdict hole produces unknown from the initial product state"):
    val watch = Model.monitor(false)(_ => false)(_ => Model.missing.reached).readAtEnds
    val result = NativeCheck.monitor(Model, "model.watch", watch, Model.initialStates)
    assertEquals(result.verdict, unknown)
    assertEquals(result.depth, 0)

  test("a monitor violation on another input survives an earlier unknown product branch"):
    val watch =
      Model.monitor(false)(_ => if Model.breaking then true else Model.missing.reached)(identity)
    val result = NativeCheck.monitor(Model, "model.watch", watch, Model.initialStates)
    assertEquals(result.verdict, refuted)
    assert(!result.complete)
    assertEquals(result.detail.head, "counterexample: pulse(true)")

  test("default checking keeps independent safety progress and monitor results"):
    val report = NativeCheck.verifyAll(Model)
    assertEquals(report.result("known").verdict, verified)
    assertEquals(report.result("missingFrom").verdict, unknown)
    assertEquals(report.result("missingTo").verdict, unknown)
    assertEquals(report.result("knownProgress").verdict, verified)
    assertEquals(report.result("uncertain").verdict, unknown)
    assertEquals(report.result("knownMonitor").verdict, verified)

  test("ordinary monitor callback errors still propagate"):
    val watch = Model.monitor(false)(_ => throw IllegalStateException("broken callback"))(identity)
    val error = intercept[IllegalStateException](
      NativeCheck.monitor(Model, "model.watch", watch, Model.initialStates)
    )
    assertEquals(error.getMessage, "broken callback")

  test("ordinary progress predicate errors still propagate"):
    val source = Model.progress.knownProgress
    val claim = LeadsTo.declared(
      source.name,
      source.machine,
      () => throw IllegalStateException("broken predicate"),
      source.to,
      source.within,
      source.scenario
    )
    val error = intercept[IllegalStateException](
      NativeCheck.progress(NativeCheck.explore(Model, Model.initialStates), "model.broken", claim)
    )
    assertEquals(error.getMessage, "broken predicate")

  test("a hole in every outgoing row does not establish a progress deadlock"):
    val result = NativeCheck.query(
      verify(HoleSteps.progress.finishes) in only(worker.waits),
      "holeSteps.finishes"
    )
    assertEquals(result.verdict, unknown)
    assert(!result.complete)

  test("a known nonprogress cycle survives an unrelated transition hole"):
    val result = NativeCheck.query(verify(HoleSteps.progress.finishes), "holeSteps.finishes")
    assertEquals(result.verdict, cycle)
    assert(!result.complete)

  test(
    "unknown enabledness cannot establish fairness of a cycle but a known cycle misses its deadline"
  ):
    val result = NativeCheck.query(
      verify(HoleSteps.progress.finishes) in fair(worker.waits),
      "holeSteps.finishes"
    )
    assertEquals(result.verdict, missedDeadline)
    assert(!result.complete)
    assertEquals(
      result.detail.head,
      "holeSteps.finishes: a cycle can avoid a to state for any number of steps"
    )
