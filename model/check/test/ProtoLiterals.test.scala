package umpire.check

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

  test("field paths are refused across a declaration newline and without snake case"):
    val source =
      """val pathName =
        |  "namespace"
        |val p = "history.events""".stripMargin
    assertEquals(ProtoLiterals.problems(source).map(_.value), Vector("namespace", "history.events"))

  test("joined method and field path fragments cannot hide a proto name"):
    val source =
      """val method = "/temporal.api.workflowservice.v1.WorkflowService/" + "StartActivityExecution"
        |val path = "task_queue" + ".name""".stripMargin
    assertEquals(ProtoLiterals.problems(source).map(_.category), Vector("method", "field path"))

  test("Model identifiers, payload map data and comments are not proto names"):
    val source =
      """val role = Role("temporal.workflow-service", RoleKind.endpoint)
        |val dottedRole = Role("temporal.workflow_service.v1", RoleKind.endpoint)
        |val evidenceId = "temporal.activity.some_state.evidence.run"
        |val evidence = Evidence.read(evidenceId, records = "scheduled")
        |val metadata = ProtoEntry.typed("encoding", ProtoValue.utf8("json/plain"))
        |val dataKey = ProtoEntry.typed("task_queue.name", ProtoValue.utf8("json/plain"))
        |// val old = "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
        |/* val old = "task_queue.name" */""".stripMargin
    assertEquals(ProtoLiterals.problems(source), Vector.empty)

  test("bound and positional evidence IDs and multiline payload map keys remain data"):
    val source =
      """private val scheduled = "temporal.nexus.caller.evidence.some_state"
        |val bound = Evidence.read(scheduled, records = "scheduled")
        |val read = Evidence.read("temporal.activity.some_state", records = "scheduled")
        |val metadata = ProtoEntry.typed(
        |  "task_queue.name",
        |  ProtoValue.utf8("json/plain")
        |)""".stripMargin
    assertEquals(ProtoLiterals.problems(source), Vector.empty)

  test("ID-shaped variable names and comments do not exempt field paths"):
    val source =
      """val pathId = "history.events"
        |val operation = pathId
        |val id = "history.events"
        |// val pathName =
        |val label = "namespace"
        |// Evidence.read(scheduled, records = "scheduled")
        |val scheduled = "temporal.nexus.caller.evidence.some_state""".stripMargin
    assertEquals(
      ProtoLiterals.problems(source).map(_.value),
      Vector("history.events", "history.events", "temporal.nexus.caller.evidence.some_state")
    )

  test("an ID-shaped binding cannot hide a protobuf message or method name"):
    val source =
      """val scheduledEvidence = "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
        |val methodId = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution""".stripMargin
    assertEquals(
      ProtoLiterals.problems(source).map(_.category),
      Vector("package or message", "method")
    )
