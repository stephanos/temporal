package temporal
// Lifecycle classifications are read as roles; states keeps vocabulary with its own meaning.

class RolePredicatesTest extends munit.FunSuite:
  test("states declares no forwarding alias of a phase role outside the named keeps") {
    import features.activity.standalone.{product as activityProduct, system as activitySystem}
    import features.nexus.standalone.protocol as nexusStandalone
    import features.nexus.workflow.protocol as nexusWorkflow

    val retired = Seq(
      activityProduct.ActivityProduct.states -> Set("over", "held"),
      activitySystem.ActivitySystem.states -> Set("terminal", "live", "held", "waiting"),
      nexusWorkflow.NexusProtocol.states -> Set("terminalPhase", "running", "waiting"),
      nexusStandalone.NexusProtocol.states -> Set("over", "live")
    )
    for (states, names) <- retired do
      assertEquals(
        states.getClass.getDeclaredMethods.map(_.getName).toSet.intersect(names),
        Set.empty[String]
      )
  }
