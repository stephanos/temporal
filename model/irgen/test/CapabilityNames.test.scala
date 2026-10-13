package umpire.irgen

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.Tools

class CapabilityNamesTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch = umpire.check.Scratch.directory("capability-names.")
  private val mapper = new com.fasterxml.jackson.databind.ObjectMapper()

  test("isolated actual kinds emit owned origins, applicable laws and exact Query budgets"):
    val source = tools.directory.resolve("model/irgen/testdata/capabilityNames/Names.scala")
    Files.copy(source, scratch.resolve("Names.scala")): Unit
    Files.writeString(
      scratch.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = scratch.resolve("names.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          scratch.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val out = scratch.resolve("names.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=model/irgen/testdata/capabilityNames/,${ModelBuild.jar}=model/",
        ModelBuild.classpath.toString,
        out.toString,
        "fixture.capabilitynames.Record$.capabilities",
        "fixture.capabilitynames.Record$.queries"
      )
    )
    assertEquals(result.exit, 0, result.diagnostics)
    val ir = mapper.readTree(Files.readString(out))
    val expected = Map(
      "record.retryable.attemptCountIsWithinPolicy" -> "Retryable.attemptCountIsWithinPolicy",
      "record.retryable.failureEndsFailed" -> "Retryable.failureEndsFailed",
      "record.retryable.failureReturnsToWaiting" -> "Retryable.failureReturnsToWaiting",
      "record.scheduleDeadline.firesInWindow" -> "Expirable.firesInWindow",
      "record.scheduleDeadline.deadlineTimesOut" -> "Expirable.deadlineTimesOut"
    )
    val properties = ir.path("properties").elements().asScala.toList
    assertEquals(properties.map(_.path("name").asText()).toSet, expected.keySet)
    for p <- properties do
      val origin = expected(p.path("name").asText())
      assertEquals(p.path("origin").path("name").asText(), s"temporal.capabilities.system.$origin")
      assertEquals(
        p.path("origin").path("position").path("file").asText(),
        s"model/temporal/capabilities/system/${origin.takeWhile(_ != '.')}.scala"
      )
      assertEquals(
        p.path("position").path("file").asText(),
        "model/irgen/testdata/capabilityNames/Names.scala"
      )
      if origin.startsWith("Expirable") then
        assertEquals(
          p.path("whenClass").path("action").asText(),
          "fixture.capabilitynames.deadline"
        )
      else if !origin.endsWith("attemptCountIsWithinPolicy") then
        assertEquals(
          p.path("whenClass").path("action").asText(),
          "fixture.capabilitynames.worker.fail"
        )
    val queries = ir.path("queries").elements().asScala.toList
    assertEquals(queries.map(_.path("name").asText()).toSet, expected.keySet)
    for q <- queries do
      assertEquals(q.path("form").asText(), "FORM_VERIFY")
      assertEquals(q.path("limits").path("steps").asInt(), 3)
      assertEquals(q.path("limits").path("actions").asInt(), 3)
      assertEquals(q.path("limits").path("search").asInt(), 4096)
      assertEquals(q.path("total").asText(), "72")
      assert(!q.has("expectedRun"))
    assertEquals(ir.path("machines").get(0).path("name").asText(), "record")

  test(
    "actual mini capability instances preserve full bodies, positions, waiver and Query metadata"
  ):
    // The mini lifted from the packaged Models, rather than a stored lift of an earlier mini.
    val bytes = Files.readAllBytes(ModelBuild.mini.resolve("activity-mini.json"))
    val ir = mapper.readTree(bytes)
    val expected = Map(
      "activityProtocol.retryable.attemptCountIsWithinPolicy" -> "Retryable.attemptCountIsWithinPolicy",
      "activityProtocol.retryable.failureCancels" -> "Retryable.failureCancels",
      "activityProtocol.retryable.failureEndsFailed" -> "Retryable.failureEndsFailed",
      "activityProtocol.retryable.failureReturnsToWaiting" -> "Retryable.failureReturnsToWaiting",
      "activityProtocol.scheduleDeadline.deadlineTimesOut" -> "Expirable.deadlineTimesOut",
      "activityProtocol.scheduleDeadline.firesInWindow" -> "Expirable.firesInWindow"
    )
    val selected = ir
      .path("properties")
      .elements()
      .asScala
      .filter(p =>
        p.path("name").asText().startsWith("activityProtocol.retryable.") ||
          p.path("name").asText().startsWith("activityProtocol.scheduleDeadline.")
      )
      .toList
    assertEquals(selected.map(_.path("name").asText()).toSet, expected.keySet)
    for p <- selected do
      val origin = expected(p.path("name").asText())
      assertEquals(p.path("origin").path("name").asText(), s"temporal.capabilities.system.$origin")
      assertEquals(
        p.path("position").path("line").asInt(),
        if origin.startsWith("Retryable") then 52 else 60
      )
      assertEquals(
        p.path("origin").path("position").path("file").asText(),
        s"model/temporal/capabilities/system/${origin.takeWhile(_ != '.')}.scala"
      )
    val queries = ir
      .path("queries")
      .elements()
      .asScala
      .filter(q => expected.contains(q.path("name").asText()))
      .toList
    assertEquals(queries.size, 6)
    for q <- queries do
      assertEquals(q.path("form").asText(), "FORM_VERIFY")
      assertEquals(q.path("limits").path("steps").asInt(), 6)
      assertEquals(q.path("limits").path("actions").asInt(), 6)
      assertEquals(q.path("limits").path("search").asInt(), 262144)
      assertEquals(q.path("total").asText(), "18432")
    val waiver = mapper
      .readTree(Files.readString(ModelBuild.mini.resolve("activity-mini.waivers.json")))
      .path("waivers")
      .get(1)
    assertEquals(
      waiver.path("subject").asText(),
      "activityProtocol.retryable.failureReturnsToWaiting"
    )
    assertEquals(
      waiver.path("because").asText(),
      "The mini protocol distinguishes retry backoff from dispatch-ready waiting while reporting scheduled for both."
    )
    val overrideBody = ir
      .path("functions")
      .elements()
      .asScala
      .find(
        _.path("name").asText() ==
          "activityProtocol.property.activityProtocol.retryable.failureReturnsToWaiting"
      )
      .get
      .path("body")
    assertEquals(
      overrideBody
        .path("binary")
        .path("right")
        .path("binary")
        .path("right")
        .path("literal")
        .path("enum")
        .path("case")
        .asText(),
      "backingOff"
    )
    assertEquals(ir.path("queries").size(), 20)
