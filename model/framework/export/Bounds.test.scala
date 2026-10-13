package framework.`export`

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.Tools
import FixtureBuild.buildFixture

class BoundsTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val modelJar = ModelBuild.jar
  private val scratch = framework.check.Scratch.directory("bounds.")
  private val json = new com.fasterxml.jackson.databind.ObjectMapper()
  private def lift(jars: String, output: java.nio.file.Path, roots: Seq[String]) =
    val result = InProcess.lift(
      tools.directory,
      Seq(jars, ModelBuild.classpath.toString, output.toString) ++ roots
    )
    assertEquals(result.exit, 0, result.diagnostics)
    json.readTree(Files.readString(output))

  test("qualified and imported central references lift all eleven independent receipt budgets"):
    val directory = Files.createDirectory(scratch.resolve("fixture"))
    Files.copy(
      tools.directory.resolve("model/framework/export/testdata/bounds/Bounds.scala"),
      directory.resolve("Bounds.scala")
    ): Unit
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
          directory.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val ir = lift(
      s"$jar=model/framework/export/testdata/bounds/,$modelJar=model/",
      scratch.resolve("fixture.json"),
      Seq("fixture.bounds.Job$.queries", "fixture.bounds.Job$.capabilities")
    )
    val expected = Map(
      "q2" -> ("two", 2, 512),
      "q3" -> ("three", 3, 4096),
      "q4" -> ("four", 4, 32768),
      "q5" -> ("five", 5, 65536),
      "q6" -> ("six", 6, 262144),
      "q7" -> ("seven", 7, 262144),
      "q8" -> ("eight", 8, 262144),
      "q12" -> ("twelve", 12, 262144),
      "q4Wide" -> ("fourWide", 4, 1048576),
      "q5Wide" -> ("fiveWide", 5, 1048576),
      "q12Wide" -> ("twelveWide", 12, 4194304),
      "job.kept" -> ("two", 2, 512),
      "job.absent" -> ("six", 6, 262144)
    )
    val queries = ir.path("queries").elements().asScala.toVector
    assertEquals(queries.map(_.path("name").asText()).toSet, expected.keySet)
    assertEquals(queries.size, 13)
    for q <- queries do
      val (name, depth, search) = expected(q.path("name").asText())
      assertEquals(q.path("limits").path("name").asText(), name)
      assertEquals(q.path("limits").path("steps").asInt(), depth)
      assertEquals(q.path("limits").path("actions").asInt(), depth)
      assertEquals(q.path("limits").path("search").asInt(), search)
      assertEquals(q.path("total").asText(), (2 * depth).toString)

  test(
    "complete actual mini query inventory preserves budgets, overrides, totals and source positions"
  ):
    // The mini lifted once for the suites from the packaged Models (ModelBuild).
    val ir = json.readTree(Files.readString(ModelBuild.mini.resolve("activity-mini.json")))
    val expected = Vector(
      ("activityProduct.terminalStatesAreFinal", 3, "product/Completion.scala", 49, 315),
      (
        "activityProtocol.retryable.attemptCountIsWithinPolicy",
        6,
        "protocol/Completion.scala",
        52,
        18432
      ),
      ("activityProtocol.retryable.failureCancels", 6, "protocol/Completion.scala", 52, 18432),
      ("activityProtocol.retryable.failureEndsFailed", 6, "protocol/Completion.scala", 52, 18432),
      (
        "activityProtocol.retryable.failureReturnsToWaiting",
        6,
        "protocol/Completion.scala",
        52,
        18432
      ),
      (
        "activityProtocol.scheduleDeadline.deadlineTimesOut",
        6,
        "protocol/Completion.scala",
        60,
        18432
      ),
      (
        "activityProtocol.scheduleDeadline.firesInWindow",
        6,
        "protocol/Completion.scala",
        60,
        18432
      ),
      ("answerLoss", 3, "protocol/AnswerDispatch.scala", 63, 225),
      ("cancellationUnknown", 3, "protocol/AnswerDispatch.scala", 61, 225),
      ("completed", 3, "product/Product.scala", 47, 21),
      ("failedAnswer", 3, "protocol/AnswerDispatch.scala", 64, 225),
      ("protocolCompleted", 3, "protocol/Protocol.scala", 78, 576),
      ("queueDelivery", 3, "protocol/Queue.scala", 30, 2304),
      ("retryCompleted", 6, "protocol/Protocol.scala", 73, 1152),
      ("staleAnswerFound", 6, "protocol/AnswerTrusting.scala", 23, 450),
      ("staleAnswerRejected", 6, "protocol/AnswerTrusting.scala", 24, 450),
      ("staleIsFenced", 6, "protocol/AnswerDispatch.scala", 59, 450),
      ("starts", 3, "protocol/DispatchingProgress.scala", 23, 192),
      ("terminalIsFinal", 3, "product/Product.scala", 49, 315),
      ("terminalThroughProduct", 6, "protocol/Protocol.scala", 72, 18432)
    )
    val actual = ir.path("queries").elements().asScala.toVector
    assertEquals(actual.map(_.path("name").asText()), expected.map(_._1))
    for (q, (_, depth, file, line, total)) <- actual.zip(expected) do
      assertEquals(q.path("limits").path("name").asText(), if depth == 3 then "three" else "six")
      assertEquals(q.path("limits").path("steps").asInt(), depth)
      assertEquals(q.path("limits").path("actions").asInt(), depth)
      assertEquals(q.path("limits").path("search").asInt(), if depth == 3 then 4096 else 262144)
      assertEquals(
        q.path("position").path("file").asText(),
        "model/temporal/features/activity_mini/" + file
      )
      assertEquals(q.path("position").path("line").asInt(), line)
      assertEquals(q.path("total").asText(), total.toString)
    for (section, size) <- Vector(
        "types" -> 22,
        "functions" -> 90,
        "actions" -> 16,
        "machines" -> 8,
        "properties" -> 19,
        "scenarios" -> 19,
        "queries" -> 20,
        "compositions" -> 1,
        "realizations" -> 1,
        "monitors" -> 1,
        "assumptions" -> 3,
        "holes" -> 1,
        "channels" -> 1,
        "progress" -> 1
      )
    do assertEquals(ir.path(section).size(), size, section)
