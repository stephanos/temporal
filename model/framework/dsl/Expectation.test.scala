package framework.dsl

import framework.realize.*

class ExpectationTest extends munit.FunSuite:
  test("satisfied full Run and monitor expectations refuse source explanations") {
    assert(compileErrors("""
      import framework.realize.*
      RunExpectation(Conformance.conformant, PropertyOutcome.satisfied,
        PropertyOutcome.satisfied, Disposition.completed, Cleanup.succeeded,
        because = "already satisfied")
    """).nonEmpty)
    assert(compileErrors("""
      import framework.realize.*
      MonitorExpectation("watch", PropertyOutcome.satisfied, because = "already satisfied")
    """).nonEmpty)
  }

  test("new and copy cannot bypass the expectation reason contract") {
    assert(compileErrors("""
      import framework.realize.*
      new RunExpectation(Conformance.conformant, PropertyOutcome.satisfied,
        PropertyOutcome.satisfied, Disposition.completed, Cleanup.succeeded,
        None, None, Vector.empty, "already satisfied")
    """).nonEmpty)
    assert(compileErrors("""
      import framework.realize.*
      new MonitorExpectation("watch", PropertyOutcome.satisfied, None, "already satisfied")
    """).nonEmpty)
    assert(compileErrors("""
      import framework.realize.*
      RunExpectation(Conformance.conformant, PropertyOutcome.satisfied,
        PropertyOutcome.satisfied, Disposition.completed, Cleanup.succeeded)
        .copy(because = "already satisfied")
    """).nonEmpty)
    assert(compileErrors("""
      import framework.realize.*
      MonitorExpectation("watch", PropertyOutcome.satisfied).copy(because = "already satisfied")
    """).nonEmpty)
    assert(compileErrors("""
      import framework.realize.*
      RunExpectation(Conformance.conformant, PropertyOutcome.inconclusive,
        PropertyOutcome.satisfied, Disposition.completed, Cleanup.succeeded,
        because = "no evaluation").copy(property = PropertyOutcome.satisfied)
    """).nonEmpty)
    assert(compileErrors("""
      import framework.realize.*
      val outcome: PropertyOutcome = PropertyOutcome.satisfied
      RunExpectation(Conformance.conformant, outcome, outcome,
        Disposition.completed, Cleanup.succeeded, because = "already satisfied")
    """).nonEmpty)
    assert(compileErrors("""
      import framework.realize.*
      def explained(outcome: PropertyOutcome, because: String) =
        RunExpectation(Conformance.conformant, outcome, outcome,
          Disposition.completed, Cleanup.succeeded, because = because)
      explained(PropertyOutcome.satisfied, "already satisfied")
    """).nonEmpty)
    assert(compileErrors("""
      import framework.realize.*
      def explained(outcome: PropertyOutcome, because: String) =
        MonitorExpectation("watch", outcome, because = because)
      explained(PropertyOutcome.satisfied, "already satisfied")
    """).nonEmpty)
    assert(compileErrors("""
      import framework.realize.*
      MonitorExpectation("watch", PropertyOutcome.inconclusive, because = "no evaluation")
        .copy(outcome = PropertyOutcome.satisfied)
    """).nonEmpty)
  }

  test("generic non-satisfied helpers retain every core expectation field") {
    def run(property: PropertyOutcome.violated.type | PropertyOutcome.inconclusive.type) =
      RunExpectation(
        Conformance.inconclusive,
        property,
        PropertyOutcome.violated,
        Disposition.stoppedByMonitor,
        Cleanup.failed,
        reason = Some(Reason.incomplete),
        conformanceReason = Some(Reason.noEvidence),
        monitors =
          Vector(MonitorExpectation("watch", property, Some(Reason.neverEvaluated), "no read")),
        because = "the run did not close"
      )
    val actual = run(PropertyOutcome.inconclusive)
    assertEquals(actual.conformance, Conformance.inconclusive)
    assertEquals(actual.property, PropertyOutcome.inconclusive)
    assertEquals(actual.contract, PropertyOutcome.violated)
    assertEquals(actual.disposition, Disposition.stoppedByMonitor)
    assertEquals(actual.cleanup, Cleanup.failed)
    assertEquals(actual.reason, Some(Reason.incomplete))
    assertEquals(actual.conformanceReason, Some(Reason.noEvidence))
    assertEquals(actual.monitors.head.reason, Some(Reason.neverEvaluated))
    assertEquals(actual.monitors.head.because, "no read")
    assertEquals(actual.because, "the run did not close")
    assertEquals(actual.copy(because = "").because, "")
    assertEquals(actual, run(PropertyOutcome.inconclusive))
    assertEquals(actual.hashCode(), run(PropertyOutcome.inconclusive).hashCode())
    assertNotEquals(actual, actual.copy(because = ""))
  }
