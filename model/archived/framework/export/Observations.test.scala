package framework.`export`

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.Tools
import FixtureBuild.buildFixture

class ObservationsTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val modelJar = ModelBuild.jar
  private val scratch = framework.check.Scratch.directory("observations.")
  private val json = new com.fasterxml.jackson.databind.ObjectMapper()
  private val accepted = Files.readString(
    tools.directory.resolve("model/framework/export/testdata/observations/Observations.scala")
  )

  private def compileLift(name: String, source: String) =
    val directory = Files.createDirectory(scratch.resolve(name))
    Files.writeString(directory.resolve("Observations.scala"), source): Unit
    Files.writeString(
      directory.resolve("project.scala"),
      s"//> using scala 3.9.0\n//> using jvm 27\n//> using jar $modelJar\n"
    ): Unit
    val jar = directory.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          directory.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val output = directory.resolve("fixture.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=model/framework/export/testdata/observations/,$modelJar=model/",
        ModelBuild.classpath.toString,
        output.toString,
        "fixture.observations.Observations$package$.run"
      )
    )
    (result, output)

  test(
    "inferred and explicit names, aliases, evidence and typed response metadata lift independently"
  ):
    val (result, output) = compileLift("accepted", accepted)
    assertEquals(result.exit, 0, result.diagnostics)
    val ir = json.readTree(Files.readString(output))
    val realization = ir.path("realizations").get(0)
    val observed = realization.path("observations").elements().asScala.toVector
    assertEquals(observed.map(_.path("id").asText()), Vector("inferred", "explicit-info"))
    assertEquals(
      observed.map(_.path("message").asText()),
      Vector.fill(2)("temporal.api.activity.v1.ActivityExecutionInfo")
    )
    assert(observed.forall(!_.has("read")))
    val reads = realization
      .path("scripts")
      .get(0)
      .path("items")
      .get(0)
      .path("command")
      .path("rpc")
      .path("reads")
    assertEquals(reads.size(), 1)
    assertEquals(reads.get(0).path("path").asText(), "info")
    assertEquals(
      reads.get(0).path("targets").elements().asScala.map(_.path("observe").asText()).toVector,
      Vector("inferred", "explicit-info")
    )
    val evidence =
      ir.path("functions").elements().asScala.find(_.path("name").asText() == "job.evidence").get
    assertEquals(
      evidence.findValues("text").asScala.map(_.asText()).toVector,
      Vector("inferred", "explicit-info", "inferred")
    )
    assert(
      accepted.contains(
        "DescribeActivityExecutionResponse.info: the complete ActivityExecutionInfo message."
      )
    )
    val (wrongName, wrongOutput) =
      compileLift("wrong-name", accepted.replace("\"explicit-info\"", "\"wrong-info\""))
    assertEquals(wrongName.exit, 0, wrongName.diagnostics)
    val wrongObserved =
      json.readTree(Files.readString(wrongOutput)).path("realizations").get(0).path("observations")
    assertEquals(wrongObserved.get(1).path("id").asText(), "wrong-info")
    intercept[munit.ComparisonFailException] {
      assertEquals(wrongObserved.get(1).path("id").asText(), "explicit-info")
    }: Unit
    intercept[munit.ComparisonFailException] {
      assertEquals(reads.get(0).path("path").asText(), "info.attempt")
    }: Unit

  test("a constant non-field response selector is refused at the existing lift boundary"):
    val invalid = accepted.replace("_.getInfo", "_ => ActivityExecutionInfo()")
    assert(invalid != accepted)
    val (result, output) = compileLift("invalid-path", invalid)
    assertEquals(result.exit, 1, result.diagnostics)
    assert(result.diagnostics.contains("field"), result.diagnostics)
    assert(!Files.exists(output))
