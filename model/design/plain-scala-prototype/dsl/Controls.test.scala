package plain

object Controls:
  // A negative control is a test of the checker, not part of the Model: the claim must be refuted
  // somewhere in the Model's reachable graph.
  inline def assertRefuted[M <: Facet](machine: M, claim: Claim): Unit =
    Runtime(machine).verify(claim) match
      case _: Verdict.Refuted => ()
      case other => munit.Assertions.fail(s"${claim.name}: expected refuted, got $other")
