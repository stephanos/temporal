package framework.dsl

import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object SafetyEnvironmentsFixture:
  val calls = scala.collection.mutable.ArrayBuffer.empty[Boolean]

  object worker extends Actor:
    val set = action.input[Boolean](name = "active")
    val explode = action

  object Model extends Machine[Outcome, Nothing]:
    var active = false
    val missing =
      hole
    val off =
      only(worker.set(false)).fair(worker.set(false))

    object transitions extends Transitions:
      on(worker.set): value =>
        effects.set(value)
      on(worker.explode):
        effects.explode()

    object effects:
      def set(value: Boolean) =
        calls += value: Unit
        active = value
      def explode() =
        throw IllegalStateException("an excluded action executed")

    object invariants:
      val inactive =
        check(!active) in off
      val activeFromStart =
        check(active) in only(worker.set(true)).starts(active = true)
      val losesActive =
        check(active) in only(worker.set).starts(active = true)
      val both =
        check(false)
        check(true) in off
      val uncertain =
        check(missing.reached) in off

    object postconditions:
      val staysInactive =
        (after(worker.set):
          check(!active)
        ) in off
      val neverTriggered =
        (after(worker.set(true)):
          check(active)
        ) in off
      val noSteps =
        (after(worker.set(true)):
          check(active)
        ) in only()

    object queries:
      val becomesActive =
        check(active) in off

  object Pinned extends Machine[Outcome, Nothing]:
    var active = false

    object transitions extends Transitions:
      on(worker.set): value =>
        effects.set(value)
      on(worker.explode):
        ()

    object effects:
      def set(value: Boolean) =
        active = value

    object postconditions:
      val inactivePath =
        check(!active) in actions(worker.set(false))
      val activePath =
        check(!active) in actions(worker.set(true))
      val neverTriggered =
        (after(worker.set(true)):
          check(active)
        ) in actions(worker.set(false))
      val emptyPost =
        (after(worker.set(true)):
          check(active)
        ) in actions()
      val disabledPost =
        (after(worker.set(true)):
          check(active)
        ) in actions(worker.explode())

    object invariants:
      val initiallyBroken =
        check(active) in actions(worker.set(true))
      val emptyPath =
        check(false) in actions()

  object MultipleStarts extends Machine[Outcome, Nothing]:
    var active = oneOf(List(false, true))

    object transitions extends Transitions

    object invariants:
      val inactive =
        check(!active) in starts()

  object Mixed extends Machine[Outcome, Nothing]:
    var active = false

    object transitions extends Transitions:
      on(worker.set): value =>
        effects.set(value)

    object effects:
      def set(value: Boolean) =
        active = value

    object invariants:
      val scoped =
        check(!active) in only(worker.set(false))
      val unscoped =
        check(!active)

  class Instance extends Machine[Outcome, Nothing]:
    var active = false

    object transitions extends Transitions

    object invariants:
      val broken =
        check(active) in starts()

class SafetyEnvironmentsSuite extends munit.FunSuite:
  import SafetyEnvironmentsFixture.*
  import NativeCheck.Verdict.*

  test("a safety claim can declare its own environment"):
    val errors = compileErrors("""
      import framework.dsl.*
      import framework.dsl.outcomes.Outcome
      object Model extends Machine[Outcome, Nothing]:
        var active = false
        object transitions extends Transitions
        object invariants:
          val inactive: Property =
            check(!active) in starts(active = false)
    """)
    assertEquals(errors, "")

  test("default checking uses each safety claim's environment without an unrestricted search"):
    calls.clear()
    val report = NativeCheck.verifyAll(Model)
    assertEquals(
      report.verdicts,
      Map(
        "inactive" -> verified,
        "activeFromStart" -> verified,
        "losesActive" -> refuted,
        "both" -> refuted,
        "uncertain" -> unknown,
        "staysInactive" -> verified,
        "neverTriggered" -> verified,
        "noSteps" -> verified
      )
    )
    assertEquals(report.results.size, 8)
    assert(calls.nonEmpty)

  test("explicit verification inherits exact-class selection from the claim"):
    calls.clear()
    val result = NativeCheck.query(verify(Model.invariants.inactive), "model.inactive")
    assertEquals(result.verdict, verified)
    assert(result.complete)
    assert(calls.nonEmpty)
    assert(calls.forall(_ == false), calls.toString)

  test("explicit verification can override the declared environment"):
    val result = NativeCheck.query(
      verify(Model.invariants.inactive) in only(worker.set(true)),
      "model.inactive"
    )
    assertEquals(result.verdict, refuted)
    assertEquals(result.detail, Vector("counterexample: set(true)"))

  test("find inherits the target's environment and allows an explicit override"):
    val inherited = NativeCheck.query(find(Model.queries.becomesActive), "model.becomesActive")
    assertEquals(inherited.verdict, refuted)
    val overridden = NativeCheck.query(
      find(Model.queries.becomesActive) in only(worker.set(true)),
      "model.becomesActive"
    )
    assertEquals(overridden.verdict, found)
    assertEquals(overridden.detail, Vector("witness: set(true)"))

  test("starts updates apply before the first assertion but subsequent states are still checked"):
    val held = NativeCheck.query(verify(Model.invariants.activeFromStart), "model.activeFromStart")
    assertEquals(held.verdict, verified)
    val broken = NativeCheck.query(verify(Model.invariants.losesActive), "model.losesActive")
    assertEquals(broken.verdict, refuted)
    assertEquals(broken.detail, Vector("counterexample: set(false)"))

  test("a scoped tail preserves earlier assertions of the same member"):
    val result = NativeCheck.query(verify(Model.invariants.both), "model.both")
    assertEquals(result.verdict, refuted)
    assertEquals(result.detail, Vector("counterexample: "))

  test("scoped postconditions report targets excluded by their environment as vacuous"):
    val report = NativeCheck.verifyAll(Model)
    assert(
      report.warnings.exists(_.contains("neverTriggered: set(true) never fires")),
      report.toString
    )
    assert(!report.warnings.exists(_.contains("staysInactive:")), report.toString)

  test("scoped and unscoped claims keep distinct default environments"):
    val report = NativeCheck.verifyAll(Mixed)
    assertEquals(report.verdicts, Map("scoped" -> verified, "unscoped" -> refuted))
    assertEquals(report.results.size, 2)

  test("a budget cut cannot verify a claim under its declared environment"):
    val result =
      NativeCheck.query(verify(Model.invariants.activeFromStart), "model.activeFromStart", 1)
    assertEquals(result.verdict, verified)
    val cut = NativeCheck.query(
      verify(Model.invariants.inactive) in only(worker.set),
      "model.inactive",
      1
    )
    assertEquals(cut.verdict, unknown)
    assert(!cut.complete)

  test("default checking an inherited scoped claim uses the derived bindings"):
    val derived = Mixed.rebind(
      on(worker.set): _ =>
        Mixed.effects.set(true)
    )
    val report = NativeCheck.verifyAll(derived, sections = Some(Mixed))
    assertEquals(report.result("scoped").verdict, refuted)

  test("instance claim sections are refused rather than silently verified"):
    val error = intercept[IllegalArgumentException](NativeCheck.verifyAll(new Instance))
    assert(error.getMessage.contains("claim section of an instance"), error.getMessage)

  test("default safety checking follows a declared pinned schedule"):
    val report = NativeCheck.verifyAll(Pinned)
    assertEquals(report.result("inactivePath").verdict, verified)
    assertEquals(report.result("activePath").verdict, refuted)
    assertEquals(report.result("activePath").detail, Vector("counterexample at step 1, set(true)"))

  test("a pinned invariant checks its initial state even when its first step repairs it"):
    val result =
      NativeCheck.query(verify(Pinned.invariants.initiallyBroken), "pinned.initiallyBroken")
    assertEquals(result.verdict, refuted)
    assert(result.detail.exists(_.contains("initial state")), result.toString)

  test("an empty pinned schedule still checks its initial invariant"):
    val report = NativeCheck.verifyAll(Pinned)
    assertEquals(report.result("emptyPath").verdict, refuted)

  test("a pinned postcondition reports an untriggered target as vacuous"):
    val report = NativeCheck.verifyAll(Pinned)
    assert(
      report.warnings.exists(_.contains("neverTriggered: set(true) never fires")),
      report.toString
    )

  test("a free environment with no steps reports its postcondition as vacuous"):
    val report = NativeCheck.verifyAll(Model)
    assert(report.warnings.exists(_.contains("noSteps: no steps fire")), report.toString)

  test("an empty pinned schedule reports its postcondition as vacuous"):
    val report = NativeCheck.verifyAll(Pinned)
    assert(report.warnings.exists(_.contains("emptyPost: no steps fire")), report.toString)

  test("a disabled pinned schedule reports its postcondition as vacuous"):
    val report = NativeCheck.verifyAll(Pinned)
    assert(report.warnings.exists(_.contains("disabledPost: no steps fire")), report.toString)

  test("an environment without start updates retains every declared initial state"):
    val report = NativeCheck.verifyAll(MultipleStarts)
    assertEquals(report.result("inactive").verdict, refuted)
    val result = NativeCheck.query(verify(MultipleStarts.invariants.inactive), "starts.inactive")
    assertEquals(result.verdict, refuted)

  test("a foreign environment cannot verify a claim through an unsupported refinement"):
    val result = NativeCheck.query(
      verify(Model.invariants.inactive) in starts().only(worker.set(false)),
      "model.inactive"
    )
    assertEquals(result.verdict, verified)
    val foreign = NativeCheck.query(
      verify(Model.invariants.inactive) in Mixed.invariants.scoped.scenario,
      "model.foreign"
    )
    assertEquals(foreign.verdict, unknown)
    assert(!foreign.complete)

  test("a scoped requirement cannot be empty"):
    val errors = compileErrors("""
      import framework.dsl.*
      import framework.dsl.outcomes.Outcome
      object Invalid extends Machine[Outcome, Nothing]:
        var active = false
        object transitions extends Transitions
        object invariants:
          val inactive =
            () in starts()
    """)
    assert(errors.contains("a requirement block must not be empty or contain only helper vals"))

  test("an invalid claim environment is refused with the scenario diagnostic"):
    val errors = compileErrors("""
      import framework.dsl.*
      import framework.dsl.outcomes.Outcome
      object Invalid extends Machine[Outcome, Nothing]:
        var active = false
        object transitions extends Transitions
        object invariants:
          val inactive =
            check(!active) in 3
    """)
    assert(errors.contains("in takes a scenario: only(...), fair(...), starts(...) or a named one"))
