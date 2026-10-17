package framework.dsl

import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object ClaimShortCircuitFixture:
  object Guarded extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None
    object transitions extends Transitions

    object invariants:
      val requiresValue =
        check(value.nonEmpty)
        check(value.get > 0)
      val independent = check(false)

  object Alternatives extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None
    object transitions extends Transitions

    object invariants:
      val succeeds = oneOf:
        branch:
          check(false)
          check(value.get > 0)
        branch:
          check(true)
        branch:
          check(value.get > 0)
      val fails =
        oneOf:
          branch:
            check(false)
          branch:
            check(false)
        check(value.get > 0)
      val skipped =
        check(false)
        oneOf:
          branch:
            check(value.get > 0)
          branch:
            check(value.get > 0)
      val nested = oneOf:
        branch:
          oneOf:
            branch:
              check(false)
            branch:
              check(true)
          check(false)
          check(value.get > 0)
        branch:
          check(true)
      val independent = check(false)

  object Missing extends Machine[Outcome, Nothing]:
    var value = false
    val missing = hole
    object transitions extends Transitions

    object invariants:
      val reached = oneOf:
        branch:
          check(false)
        branch:
          check(missing.reached)

  object Recorded extends Machine[Outcome, Boolean]:
    var value: Option[Boolean] = None
    object transitions extends Transitions

    object invariants:
      val guarded =
        check(value.nonEmpty)
        records(value.get)

class ClaimShortCircuitSuite extends munit.FunSuite:
  import ClaimShortCircuitFixture.*
  import NativeCheck.Verdict.*

  test("a failed requirement skips later assertion conditions in its member"):
    val report = NativeCheck.verifyAll(Guarded)
    assertEquals(report.verdicts, Map("requiresValue" -> refuted, "independent" -> refuted))
    assert(report.result("requiresValue").detail.last.endsWith(": value.nonEmpty"))

  test("oneOf skips failed branch assertions and alternatives after its first success"):
    assertEquals(
      NativeCheck.verifyAll(Alternatives).verdicts,
      Map(
        "succeeds" -> verified,
        "fails" -> refuted,
        "skipped" -> refuted,
        "nested" -> verified,
        "independent" -> refuted
      )
    )

  test("an actually evaluated hole is propagated"):
    assertEquals(intercept[HoleReached](NativeCheck.verifyAll(Missing)).hole, Missing.missing)

  test("a failed requirement skips the argument of a later fact assertion"):
    assertEquals(NativeCheck.verifyAll(Recorded).verdicts, Map("guarded" -> refuted))
