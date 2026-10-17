package framework.dsl

import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object NativeClaimHolesFixture:

  object worker extends Actor:
    val advance = action
    val pause = action

  object Model extends Machine[Outcome, Nothing]:
    var done = false
    val missing =
      hole

    object transitions extends Transitions:
      on(worker.advance):
        if !done then effects.advance()
      on(worker.pause):
        if done then effects.pause()

    object effects:
      def advance() =
        done = true
      def pause() =
        ()

    object invariants:
      val broken =
        check(false)
      val uncertain =
        check(missing.reached)
      val independent =
        check(true)
      val succeeds =
        oneOf:
          branch:
            check(missing.reached)
          branch:
            check(true)
      val undecided =
        oneOf:
          branch:
            check(false)
          branch:
            check(missing.reached)
      val guardedAlternative =
        oneOf:
          branch:
            if missing.reached then check(true)
          branch:
            check(true)

    object postconditions:
      val uncertainStep =
        after(worker.advance):
          check(missing.reached)
      val knownStep =
        check(done)
      val failsThenMissing =
        if before.done then check(missing.reached)
        else check(false)
      val holdsThenMissing =
        if before.done then check(missing.reached)
        else check(true)

class NativeClaimHolesSuite extends munit.FunSuite:
  import NativeClaimHolesFixture.*
  import NativeCheck.Verdict.*

  test("default checks retain known failures and independent claims around a reached hole"):
    val report = NativeCheck.verifyAll(Model)
    assertEquals(report.result("broken").verdict, refuted)
    assertEquals(report.result("uncertain").verdict, unknown)
    assert(!report.result("uncertain").complete)
    assert(report.result("uncertain").detail.exists(_.contains("missing")))
    assertEquals(report.result("independent").verdict, verified)
    assertEquals(report.result("knownStep").verdict, verified)
    assertEquals(report.result("uncertainStep").verdict, unknown)

  test("a known successful alternative establishes the claim despite another unknown branch"):
    val result = NativeCheck.query(verify(Model.invariants.succeeds), "model.succeeds")
    assertEquals(result.verdict, verified)
    assert(result.complete)

  test("false and unknown alternatives leave a claim unknown"):
    val result = NativeCheck.query(verify(Model.invariants.undecided), "model.undecided")
    assertEquals(result.verdict, unknown)

  test("an unknown branch guard does not hide a known successful alternative"):
    val result =
      NativeCheck.query(verify(Model.invariants.guardedAlternative), "model.guardedAlternative")
    assertEquals(result.verdict, verified)

  test("free verification reports a reached predicate hole as unknown"):
    val result = NativeCheck.query(verify(Model.invariants.uncertain), "model.uncertain")
    assertEquals(result.verdict, unknown)
    assert(!result.complete)

  test("an unknown predicate supplies no free witness"):
    val result = NativeCheck.query(find(Model.invariants.uncertain), "model.uncertain")
    assertEquals(result.verdict, unknown)

  test("a pinned verification reports a reached predicate hole as unknown"):
    val result = NativeCheck.query(
      verify(Model.postconditions.uncertainStep) in actions(worker.advance()),
      "model.uncertainStep"
    )
    assertEquals(result.verdict, unknown)
    assert(!result.complete)

  test("an unknown predicate supplies no pinned witness"):
    val result = NativeCheck.query(
      find(Model.postconditions.uncertainStep) in actions(worker.advance()),
      "model.uncertainStep"
    )
    assertEquals(result.verdict, unknown)

  test("a pinned counterexample survives a later predicate hole"):
    val result = NativeCheck.query(
      verify(Model.postconditions.failsThenMissing) in actions(worker.advance(), worker.pause()),
      "model.failsThenMissing"
    )
    assertEquals(result.verdict, refuted)
    assert(!result.complete)
    assertEquals(result.detail.head, "counterexample at step 1, advance")
    assert(result.detail.exists(_.contains("missing")))

  test("a pinned prefix match followed by a hole cannot complete a witness"):
    val result = NativeCheck.query(
      find(Model.postconditions.holdsThenMissing) in actions(worker.advance(), worker.pause()),
      "model.holdsThenMissing"
    )
    assertEquals(result.verdict, unknown)
    assert(!result.complete)

  test("an ad hoc invariant reports a reached predicate hole as unknown"):
    val result = NativeCheck.verify(
      NativeCheck.explore(Model, Model.initialStates),
      "model.adhoc",
      NativeControls.invariant(check(Model.missing.reached))
    )
    assertEquals(result.verdict, unknown)
    assert(!result.complete)

  test("a known ad hoc failure still refutes the claim and skips a later hole"):
    val result = NativeCheck.verify(
      NativeCheck.explore(Model, Model.initialStates),
      "model.adhoc",
      NativeControls.invariant:
        check(false)
        check(Model.missing.reached)
    )
    assertEquals(result.verdict, refuted)
