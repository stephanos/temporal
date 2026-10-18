package framework.realize

class ExpectationsTest extends munit.FunSuite:
  test("realize exposes the path, sample and Property-only expectation forms") {
    assert(
      scala.compiletime.testing.typeChecks("""
        import framework.realize.*
        val path: RunExpectation = satisfied
        val sample: RunExpectation = conformant
        val property: RunExpectation = inconclusive(Reason.unreadable, because = "not readable")
        (path, sample, property)
      """),
      "framework.realize must expose the three accepted expectation forms"
    )
  }
