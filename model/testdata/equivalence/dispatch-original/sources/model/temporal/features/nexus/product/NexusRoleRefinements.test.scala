package framework

// The Nexus refinements whose System and Product phases both take roles preserve closedness: a
// System state is Closed exactly when its Product image is, over every Finite System state. A
// role-carrying refinement added later is discovered through its export.
class NexusRoleRefinements extends munit.FunSuite:
  test("every discovered Nexus role refinement keeps closedness"):
    val results = ExportedLaws.results.filter(r =>
      r.subject.startsWith("temporal.features.nexus.") && r.law.equals("closedness")
    )
    val passed = results.filter(_.disposition.equals("passed")).map(_.subject).toSet
    assert(
      Set(
        "temporal.features.nexus.workflow.system.NexusSystem",
        "temporal.features.nexus.standalone.system.NexusSystem"
      ).subsetOf(passed)
    )
    for result <- results do assert(!result.disposition.equals("failed"), result.toString)
