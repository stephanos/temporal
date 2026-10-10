package umpire.irgen

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.JsonNode
import umpire.check.{Ran, Tools}

class DeadlinePresetsSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/irgen/testdata/deadlinePresets/"
  private val modelJar = build.resolve("model-scala.jar")
  private val scratch = Files.createTempDirectory(
    Files.createDirectories(build.resolve("history")),
    "deadline-presets."
  )
  private val fixtureJar = scratch.resolve("presets.jar")
  private val mapper = new com.fasterxml.jackson.databind.ObjectMapper()

  override def beforeAll(): Unit =
    val stream = Files.newDirectoryStream(tools.directory.resolve(stored), "*.scala")
    try
      for from <- stream.asScala do
        Files.writeString(
          scratch.resolve(from.getFileName),
          Files
            .readString(from)
            .replace("../../../build/model-scala.jar", modelJar.toString)
            .replace("../../../build/api-scalapb.jar", build.resolve("api-scalapb.jar").toString)
        )
    finally stream.close()
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          scratch.toString,
          "-f",
          "-o",
          fixtureJar.toString
        )
      )
      .orFail(): Unit

  private def lift(roots: String*): (Ran, Path) =
    val out = scratch.resolve(roots.head.split('.').last + ".json")
    val ran = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "umpire.irgen.lift",
        s"$fixtureJar=$stored,$modelJar=model/",
        build.resolve("model-scala.classpath").toString,
        out.toString
      ) ++ roots
    )
    (ran, out)

  private def items(root: String): Seq[JsonNode] =
    val (ran, out) = lift(root)
    ran.orFail(): Unit
    mapper
      .readTree(Files.readString(out))
      .at("/realizations/0/scripts/0/items")
      .elements()
      .asScala
      .toSeq

  test("three retry presets retain all 48 distinct start classes and their command identities"):
    val controller = items("temporal.features.activity.standalone.system.Standalone")
    val itemCommands = controller.filter(_.has("command")).map(_.at("/command/id").asText())
    assertEquals(itemCommands.distinct.size, itemCommands.size)
    val starts = controller
      .flatMap(_.path("performs").elements().asScala)
      .filter(_.at("/step/action").asText().endsWith("client.start"))
    assertEquals(starts.size, 48)
    val classes = starts.map(_.path("step"))
    assertEquals(classes.distinct.size, 48)
    val policyCounts =
      starts.groupBy(_.at("/step/inputs/5/enum/case").asText()).view.mapValues(_.size).toMap
    assertEquals(policyCounts, Map("unlimited" -> 16, "one" -> 16, "two" -> 16))
    for step <- starts do
      val assignments = step.at("/command/rpc/assign").elements().asScala.toSeq
      assertEquals(assignments.map(_.path("target").asText()).distinct.size, assignments.size)
      val maximum =
        assignments.find(_.path("target").asText() == "retry_policy.maximum_attempts").get
      val expected = step.at("/step/inputs/5/enum/case").asText() match
        case "unlimited" => "0"
        case "one"       => "1"
        case "two"       => "2"
      assertEquals(maximum.at("/value/literal/number").asText(), expected)
    assertEquals(starts.map(_.at("/command/id").asText()).distinct.size, 1)
    val catalog = items("fixture.deadlinepresets.Catalog")
    val bound = catalog
      .dropRight(1)
      .flatMap(_.path("performs").elements().asScala)
      .map(_.path("step"))
    val selected = catalog.last.path("when").elements().asScala.toSeq
    assertEquals(bound.size, 6)
    assertEquals(selected, bound)
    assertEquals(selected.distinct.size, 6)

  test("a bare all-Timeout action keeps its four bindings and onPath catalog"):
    val controller = items("fixture.deadlinepresets.Legacy")
    val performed = controller.head.path("performs").elements().asScala.toSeq
    assertEquals(performed.size, 4)
    assertEquals(performed.map(_.path("step")).distinct.size, 4)
    assertEquals(controller(1).path("when").size(), 4)
    assertEquals(performed.map(_.at("/command/id").asText()).distinct.size, 1)
    for step <- performed do
      val inputs = step.at("/step/inputs").elements().asScala.toSeq
      val expected = inputs
        .zip(Seq("start_delay.seconds", "start_to_close_timeout.seconds"))
        .collect { case (input, field) if input.at("/enum/case").asText() == "expires" => field }
        .toSet
      val assignments = step.at("/command/rpc/assign").elements().asScala.toSeq
      assertEquals(
        assignments.map(_.path("target").asText()).toSet,
        expected ++ Set("namespace", "activity_id")
      )
      for assignment <- assignments if expected(assignment.path("target").asText()) do
        assertEquals(assignment.at("/value/literal/number").asText(), "2")

  test(
    "non-Timeout presets are explicit, Timeout defaults unambiguous, and duplicate classes refused"
  ):
    for (root, message) <- Seq(
        "Bare" -> "requires an explicit Class preset for non-Timeout inputs",
        "ExpiringPreset" -> "must leave Timeout input startToClose at its default",
        "DuplicatePreset" -> "binds a class twice"
      )
    do
      val (ran, _) = lift("fixture.deadlinepresets." + root)
      assert(ran.failed, ran.output)
      assert(ran.output.contains(message), ran.output)

  test("the timeout-retry machine keeps the base table and has only its own occurrence evidence"):
    val prefix = "temporal.features.activity.standalone.system."
    val (ran, out) = lift(
      prefix + "TimeoutRetry$.queries",
      prefix + "RetryAfterTimeout",
      prefix + "ActivitySystem$.queries",
      prefix + "RetryFailures$.queries",
      prefix + "Standalone"
    )
    ran.orFail(): Unit
    val model = mapper.readTree(Files.readString(out))
    def named(kind: String, name: String) = model
      .path(kind)
      .elements()
      .asScala
      .find(_.path("name").asText() == name)
      .get
    val base = named("machines", "activitySystem")
    val derived = named("machines", "timeoutRetry")
    for field <- Seq(
        "stateType",
        "outcomeType",
        "factType",
        "starts",
        "ends",
        "steps",
        "evidence",
        "unobservable"
      )
    do
      assert(!base.path(field).isMissingNode, field)
      assertEquals(derived.path(field), base.path(field), field)
    val failures = named("queries", "retryExhaustionByFailures")
    assertEquals(failures.path("form").asText(), "FORM_VERIFY")
    assert(!failures.has("expectedRun"))
    val failurePath = named("scenarios", "exhausted").path("actions").elements().asScala.toSeq
    assertEquals(failurePath.size, 6)
    assertEquals(failurePath.count(_.path("action").asText().endsWith("worker.respondFailed")), 2)
    val exhaustion = named("queries", "retryExhaustion")
    assertEquals(exhaustion.at("/scenario/machine").asText(), "timeoutRetry")
    val exhaustionPath = named("scenarios", "timedOutThenFailed").path("actions")
    assertEquals(exhaustionPath.size(), 6)
    assert(exhaustionPath.path(2).path("action").asText().endsWith("deadline.startToClose"))
    assert(exhaustionPath.path(5).path("action").asText().endsWith("worker.respondFailed"))
    val realization = named("realizations", "retryAfterTimeout")
    val commands = realization.at("/scripts/1/items").elements().asScala.toSeq
    assert(commands.head.at("/command/attemptWithheld").isObject)
    assertEquals(commands.head.path("performs").size(), 0)
    assertEquals(commands.head.path("when").size(), 1)
    val timer = commands.head.at("/when/0/action").asText()
    assert(timer.endsWith("deadline.startToClose"))
    val second = realization
      .path("evidence")
      .elements()
      .asScala
      .find(_.at("/runEvent/attempt/number").asText() == "2")
      .get
    val confirms = second.path("confirms").elements().asScala.toSeq
    assertEquals(
      confirms.map(_.at("/step/action").asText()),
      Seq(timer, "temporal.features.activity.worker.poll")
    )
    assertEquals(confirms.map(_.path("occurrence").asText()), Seq("1", "2"))
    assert(!realization.path("evidence").toString.contains("statusTimedOut"))
    val proof = tools.directory.resolve(".flow/tmp/activity-batch/fn1283-focused.json")
    Files.createDirectories(proof.getParent)
    Files.copy(out, proof, java.nio.file.StandardCopyOption.REPLACE_EXISTING): Unit
