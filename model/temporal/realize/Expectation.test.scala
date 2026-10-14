package temporal.realize

import framework.realize.*

class TemporalExpectationTest extends munit.FunSuite:
  test("native satisfied convenience refuses a reason and inconclusive requires one"):
    assert(compileErrors("""
      import temporal.realize.*
      satisfied(because = "already satisfied")
    """).nonEmpty)
    assert(compileErrors("""
      import temporal.realize.*
      import framework.realize.Reason
      inconclusive(Reason.neverEvaluated)
    """).nonEmpty)
    assertEquals(satisfied.property, PropertyOutcome.satisfied)
    assertEquals(satisfied.contract, PropertyOutcome.satisfied)
    assertEquals(satisfied.because, "")
    val expected = inconclusive(Reason.neverEvaluated, "no property evaluation")
    assertEquals(expected.property, PropertyOutcome.inconclusive)
    assertEquals(expected.reason, Some(Reason.neverEvaluated))
    assertEquals(expected.because, "no property evaluation")
    assertEquals(expected.copy(because = "").reason, expected.reason)

  test("incomplete full Run constructors remain compiler refusals"):
    assert(compileErrors("""
      import framework.realize.*
      RunExpectation(Conformance.conformant, PropertyOutcome.satisfied,
        PropertyOutcome.satisfied)
    """).nonEmpty)
