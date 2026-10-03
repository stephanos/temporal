package umpire.gate

class ProtoLiteralsSuite extends munit.FunSuite:
  test("proto names in Model source are refused by category"):
    val examples = Seq(
      "val p = \"temporal.api.workflowservice.v1\"" -> "package or message",
      "val m = \"temporal.api.workflowservice.v1.StartActivityExecutionRequest\"" ->
        "package or message",
      "val m = \"/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution\"" ->
        "method",
      "val f = \"task_queue.name\"" -> "field path",
      "val e = \"ACTIVITY_EXECUTION_STATUS_PAUSED\"" -> "enum value"
    )
    for (source, category) <- examples do
      assertEquals(ProtoLiterals.problems(source).map(_.category), Vector(category))

  test("a single protobuf field in a retired author call is refused"):
    assertEquals(
      ProtoLiterals.problems("Assignment(\"namespace\", Operand.Run)").map(_.category),
      Vector("field path")
    )

  test("joined method and field path fragments cannot hide a proto name"):
    val source =
      """val method = "/temporal.api.workflowservice.v1.WorkflowService/" + "StartActivityExecution"
        |val path = "task_queue" + ".name""".stripMargin
    assertEquals(ProtoLiterals.problems(source).map(_.category), Vector("method", "field path"))

  test("Model identifiers, payload map data and comments are not proto names"):
    val source =
      """val family = Family("temporal.activity.standalone")
        |val nested = Family("temporal.activity.some_state")
        |val role = Role("temporal.workflow-service", RoleKind.endpoint)
        |val dottedRole = Role("temporal.workflow_service.v1", RoleKind.endpoint)
        |val evidenceId = "temporal.activity.some_state.evidence.run"
        |val metadata = ProtoEntry.typed("encoding", ProtoValue.utf8("json/plain"))
        |val dataKey = ProtoEntry.typed("task_queue.name", ProtoValue.utf8("json/plain"))
        |// val old = "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
        |/* val old = "task_queue.name" */""".stripMargin
    assertEquals(ProtoLiterals.problems(source), Vector.empty)
