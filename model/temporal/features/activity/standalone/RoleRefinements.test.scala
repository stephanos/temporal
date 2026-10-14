package framework
// The standalone activity's refinements whose System and Product phases both take roles keep
// closedness: a System state is Closed exactly when its Product image is, over every Finite state.
// Export discovery retains the Scala phase and refinement evidence for every instance.

class RoleRefinements extends munit.FunSuite:
  test("every discovered activity role refinement keeps closedness"):
    val prefix = "temporal.features.activity.standalone.protocol."
    val results =
      ExportedLaws.results.filter(r => r.subject.startsWith(prefix) && r.law.equals("closedness"))
    val passed = results.filter(_.disposition.equals("passed")).map(_.subject).toSet
    assert(
      Set("ActivityProtocol", "ActivityRecord", "HeldDispatch").map(prefix + _).subsetOf(passed)
    )
    for result <- results do assert(!result.disposition.equals("failed"), result.toString)
