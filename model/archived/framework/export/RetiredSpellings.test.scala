package framework.`export`

import java.nio.file.Files
import scala.concurrent.duration.*
import framework.check.{RetainedScratch, Tools}
import FixtureBuild.buildFixture

class RetiredSpellingsTest extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch = retained(tools.directory.resolve(".flow/tmp"), "retired-spellings.")
  private val imports = """package fixture.authoring
import framework.dsl.*
import framework.realize.*
import com.google.protobuf.wrappers.StringValue
import io.grpc.MethodDescriptor
"""

  private def compile(name: String, body: String) =
    val directory = Files.createDirectory(scratch.resolve(name))
    Files.writeString(directory.resolve("Source.scala"), imports + body): Unit
    Files.writeString(
      directory.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    tools.buildFixture(
      Seq(
        "compile",
        "--server=false",
        "--classpath",
        ModelBuild.classpathEntries,
        directory.toString
      )
    )

  test("canonical factories keep their typed representation"):
    val built = compile(
      "canonical",
      """val method: MethodDescriptor[StringValue, StringValue] = null
val field = Field[StringValue, String](_.value)
val assignment: TypedAssignment[StringValue, String] = Assignment.typed(field, Operand.text("value"))
val response: TypedResponseRead[StringValue, String] = ResponseRead.typed(field, Cardinality.one, Vector.empty)
val evidenceField: TypedEvidenceField[StringValue, String] = EvidenceField.typed("value", field)
val single = Recorded.single(method, Field[StringValue, StringValue](identity))
val repeated = Recorded.read(method, Field[StringValue, Seq[StringValue]](_ => Seq.empty))
val evidence = Evidence.read("evidence", "records", "source", single, field, Commitment.reported)
val poll = Instruction.readUntil(evidence, "endpoint")(Vector(assignment), Condition.present(field))
val scopedPoll = readUntil(evidence, "endpoint", Condition.present(field), None) {
  framework.realize.field(_.value) := Operand.text("value")
}
val call = Instruction.rpc("endpoint", method)(Vector(assignment), Vector(response))
val scopedCall: Instruction.TypedRpc[StringValue, StringValue] = rpc("endpoint", method) {
  framework.realize.field(_.value) := Operand.text("value")
}
val event = Recorded.runEvent[StringValue](EventKind.instructionCompleted, "script", "call", Operand.runKey())
val history = temporal.realize.WorkflowHistory.event(field)
"""
    )
    assertEquals(built.exit, 0, built.diagnostics)

  private def constructors = Seq(
    "rpc" -> "Instruction.TypedRpc[StringValue, StringValue](\"endpoint\", null, Vector.empty, Vector.empty)",
    "poll" -> "Instruction.TypedPoll[StringValue, StringValue](null, \"endpoint\", Vector.empty, null, None)",
    "read" -> "Recorded.TypedRead[StringValue, StringValue, StringValue](null, null)",
    "single" -> "Recorded.TypedSingle[StringValue, StringValue, StringValue](null, null)",
    "event" -> "Recorded.TypedRunEvent[StringValue](EventKind.instructionCompleted, \"script\", \"call\", Operand.runKey(), None, None)",
    "assignment" -> "TypedAssignment[StringValue, String](null, Operand.text(\"value\"))",
    "response" -> "TypedResponseRead[StringValue, String](null, Cardinality.one, Vector.empty)",
    "field" -> "TypedEvidenceField[StringValue, String](\"field\", null, None, false)",
    "history" -> "temporal.realize.WorkflowHistory.TypedHistory[StringValue, String](null)"
  )
  for (name, constructor) <- constructors do
    test(s"direct $name constructors are unavailable to an author"):
      val tpe = constructor.takeWhile(_ != '(')
      val arguments = constructor.drop(tpe.length)
      val built = compile(
        name,
        s"val applied = $constructor\nval constructed = new $constructor\nval copied = (null: $tpe).copy$arguments\n"
      )
      assert(built.failed, s"$name remains public: ${built.output}")
      assert(built.diagnostics.contains("Source.scala:6:"), built.diagnostics)
      assert(built.diagnostics.contains("Source.scala:7:"), built.diagnostics)
      assert(built.diagnostics.contains("Source.scala:8:"), built.diagnostics)
      assert(
        built.diagnostics.contains("constructor") && built.diagnostics.contains(
          "cannot be accessed"
        ),
        built.diagnostics
      )

  test("receiverless capabilities and chained overriding remain retired"):
    for (name, body, diagnostic) <- Seq(
        (
          "receiverless",
          "val retired = capabilities(Limits(steps = 1))()\n",
          "Not found: capabilities"
        ),
        (
          "chained",
          "val retired = (null: Query).overriding(\"old\")\n",
          "overriding is not a member"
        )
      )
    do
      val built = compile(name, body)
      assert(built.failed, built.output)
      assert(built.diagnostics.contains(diagnostic), built.diagnostics)
