package umpire.gate

import java.nio.file.Files

class SourceMetricsSuite extends munit.FunSuite:
  test("each literal is classified by the code around it, and comments hold none"):
    val dir = Files.createTempDirectory("metrics")
    Files.writeString(
      dir.resolve("Model.scala"),
      """// "not a literal"
        |/* nor "this" */
        |val dispatch = internal("dispatch")
        |val notPaused = m.property("notAdmittedWhilePaused")
        |def evidence(f: Fact): String = f match
        |  case Fact.started => "started"
        |val c = compose[S](Family, "over")("activity" -> record).sync("admit", "activity" -> a)
        |val q = query(s"${m.name}.any")
        |val step = Step(accepted, s, Nil, because = "the worker learns of it")
        |val m = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"
        |val keyed = Vector("activity_control-pause")
        |val again = accepted("dispatch", start)
        |val family = Family("temporal.activity.standalone")
        |""".stripMargin
    )
    Files.writeString(dir.resolve("Model.test.scala"), "val left = \"out\"\n")
    val (lines, literals) = SourceMetrics.measure(dir, Vector(dir.resolve("Model.scala")))
    assertEquals(lines, Vector("Model.scala" -> 13))
    assertEquals(
      literals.map(l => l.line -> l.category),
      Vector(
        3 -> "own name",
        4 -> "declared name",
        6 -> "evidence",
        7 -> "declared name",
        7 -> "composition key",
        7 -> "composition key",
        7 -> "composition key",
        8 -> "computed name",
        9 -> "prose",
        10 -> "Temporal API name",
        11 -> "composition key",
        12 -> "repeated name",
        13 -> "id"
      )
    )
    val report = SourceMetrics.report(dir, Seq(dir))
    assert(report.contains("    13 lines    13 literals  total"), report)
    assert(!report.contains("Model.test.scala"), report)
