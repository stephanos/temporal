package temporal
// Lifecycle classifications are read as roles; states keeps vocabulary with its own meaning.

class RolePredicatesTest extends munit.FunSuite:
  test("states declares no forwarding alias of a phase role outside the named keeps") {
    val retired = Map(
      "temporal.features.activity.standalone.product.ActivityProduct" -> Set("over", "held"),
      "temporal.features.activity.standalone.protocol.ActivityProtocol" -> Set(
        "terminal",
        "live",
        "held",
        "waiting"
      ),
      "temporal.features.nexus.workflow.protocol.NexusProtocol" -> Set(
        "terminalPhase",
        "running",
        "waiting"
      ),
      "temporal.features.nexus.standalone.protocol.NexusProtocol" -> Set("over", "live")
    )
    for
      model <- framework.IrFile.discover("temporal").flatMap(_.subjects()).distinct
      names <- retired.get(framework.IrFile.identity(model))
    do
      val states = model.getClass.getMethod("states").invoke(model)
      assertEquals(
        states.getClass.getDeclaredMethods.map(_.getName).toSet.intersect(names),
        Set.empty[String]
      )
  }
