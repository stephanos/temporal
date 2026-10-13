package framework.`export`

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.JsonNode
import framework.check.{Ran, Tools}
import FixtureBuild.buildFixture

class InputDeadlinePresetsTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch =
    Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "input-deadlines.")
  // The packaged Models: the mini and the shared code this fixture reads.
  private val modelJar = ModelBuild.jar
  private def classpath = ModelBuild.classpath
  private val source = """package fixture.inputdeadlines
import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import framework.realize.*
import temporal.realize.*
import temporal.features.activity.Timeout
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest as Request
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
enum Policy:
  case first, second
val close = input[Timeout](default = Timeout.unset)
val policy = input[Policy]
val device = Entity(key = "activityId")
object caller extends Actor:
  val start = action(this).input(close).input(policy).creates(device)
final case class State(value: Boolean)
object Device extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def start(s: State, t: Timeout, p: Policy) = stay(s)
  object rules extends Rules:
    on(caller.start) { always ~> effects.start }
val calls = RequestBase(workflowService, "namespace" -> workerNamespace, "activity_id" -> run)
val launch = rpc(calls, METHOD_START_ACTIVITY_EXECUTION) {}
object Preset extends Realizes(Device):
  object controller extends Controller(
    deadlines[Request](caller.start(policy := Policy.second), launch, duration(2))(
      close.sets(_.getStartToCloseTimeout)),
    onPath(caller.start)(launch))
"""
  private def lift(name: String, written: String, root: String = "fixture.inputdeadlines.Preset") =
    val directory = Files.createDirectory(scratch.resolve(name))
    Files.writeString(directory.resolve("Preset.scala"), written): Unit
    Files.writeString(
      directory.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
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
    val out = directory.resolve("fixture.json")
    val ran = InProcess.lift(
      tools.directory,
      Seq(s"$jar=$directory/,$modelJar=model/", classpath.toString, out.toString, root)
    )
    (ran, out)

  test("explicit required non-Timeout preset values survive both deadline variants"):
    val (ran, out) = lift("required-policy", source)
    assertEquals(ran.exit, 0, ran.diagnostics)
    val json = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(out))
    val items = json.at("/realizations/0/scripts/0/items")
    val performs = items.get(0).path("performs").elements().asScala.toVector
    assertEquals(performs.size, 2)
    assertEquals(
      performs.map(_.at("/step/inputs/1/enum/case").asText()),
      Vector("second", "second")
    )
    assertEquals(
      performs.map(_.at("/step/inputs/0/enum/case").asText()),
      Vector("unset", "expires")
    )
    assertEquals(
      items.get(1).path("when").elements().asScala.toVector,
      performs.map(_.path("step"))
    )

  test("deadline bare, non-default Timeout and required Timeout presets refuse at their contract"):
    for (name, written, message) <- List(
        (
          "bare",
          source.replace("caller.start(policy := Policy.second)", "caller.start"),
          "requires an explicit Class preset for non-Timeout inputs"
        ),
        (
          "expiring",
          source.replace(
            "policy := Policy.second",
            "close := Timeout.expires, policy := Policy.second"
          ),
          "must leave Timeout input close at its default"
        ),
        (
          "required-timeout",
          source
            .replace("(default = Timeout.unset)", "")
            .replace("policy := Policy.second", "close := Timeout.unset, policy := Policy.second"),
          "input close of start has no default"
        )
      )
    do
      val (ran, out) = lift(name, written)
      assertEquals(ran.exit, 1, ran.diagnostics)
      assert(ran.diagnostics.contains(message), ran.diagnostics)
      val line =
        written.linesIterator.toVector.indexWhere(_.trim.startsWith("deadlines[Request]")) + 1
      assert(ran.diagnostics.contains(s"Preset.scala:$line:"), ran.diagnostics)
      assert(!Files.exists(out))

  test("deadline variants use semantic unset/expires even when their enum order is reversed"):
    val reversed = source.replace(
      "import temporal.features.activity.Timeout",
      "enum Timeout:\n  case expires, unset"
    )
    val (ran, out) = lift("reversed", reversed)
    assertEquals(ran.exit, 0, ran.diagnostics)
    val json = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(out))
    val performs = json.at("/realizations/0/scripts/0/items/0/performs").elements().asScala.toVector
    assertEquals(
      performs.map(_.at("/step/inputs/0/enum/case").asText()),
      Vector("unset", "expires")
    )

  test("an expires default is an ordinary input default but refuses the deadline helper contract"):
    val expiring = source.replace("default = Timeout.unset", "default = Timeout.expires")
    val (refused, absent) = lift("expiring-default", expiring)
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(
      refused.diagnostics.contains("Timeout input close must declare default unset"),
      refused.diagnostics
    )
    val line =
      expiring.linesIterator.toVector.indexWhere(_.trim.startsWith("deadlines[Request]")) + 1
    assert(refused.diagnostics.contains(s"Preset.scala:$line:"), refused.diagnostics)
    assert(!Files.exists(absent))
    val ordinary = expiring + """
object Ordinary:
  val holds = Device.property.holds(_ => true)
  val path = Device.scenario.actions(caller.start(policy := Policy.first))
  val queryDefault = query verify holds in path limits Limits("one", steps = 1, actions = 1, search = 64)
"""
    val (accepted, out) =
      lift("ordinary-expiring", ordinary, "fixture.inputdeadlines.Ordinary$.queryDefault")
    assertEquals(accepted.exit, 0, accepted.diagnostics)
    val json = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(out))
    assertEquals(json.at("/scenarios/0/actions/0/inputs/0/enum/case").asText(), "expires")

class DeadlinePresetsSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/framework/export/testdata/deadlinePresets/"
  private val modelJar = build.resolve("model-scala.jar")
  private val scratch = Files.createTempDirectory(
    Files.createDirectories(build.resolve("history")),
    "deadline-presets."
  )
  private val fixtureJar = scratch.resolve("presets.jar")
  private val mapper = new com.fasterxml.jackson.databind.ObjectMapper()

  override def beforeAll(): Unit =
    val stream =
      Files.newDirectoryStream(tools.directory.resolve(stored), "*.scala")
    try
      for from <- stream.asScala do
        Files.writeString(
          scratch.resolve(from.getFileName),
          Files
            .readString(from)
            .replace("../../../../build/model-scala.jar", modelJar.toString)
            .replace(
              "../../../../build/api-scalapb.jar",
              build.resolve("api-scalapb.jar").toString
            )
        )
    finally stream.close()
    tools
      .buildFixture(
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
    val ran = InProcess.lift(
      tools.directory,
      Seq(
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

  test(
    SweepHeld(
      "three retry presets retain all 48 distinct start classes and their command identities",
      "features/activity/standalone"
    )
  ):
    val controller = items(
      "temporal.features.activity.standalone.realization.StandaloneActivityRun"
    )
    val itemCommands =
      controller.filter(_.has("command")).map(_.at("/command/id").asText())
    assertEquals(itemCommands.distinct.size, itemCommands.size)
    val starts = controller
      .flatMap(_.path("performs").elements().asScala)
      .filter(_.at("/step/action").asText().endsWith("client.start"))
    assertEquals(starts.size, 48)
    val classes = starts.map(_.path("step"))
    assertEquals(classes.distinct.size, 48)
    val policyCounts =
      starts
        .groupBy(_.at("/step/inputs/5/enum/case").asText())
        .view
        .mapValues(_.size)
        .toMap
    assertEquals(policyCounts, Map("unlimited" -> 16, "one" -> 16, "two" -> 16))
    for step <- starts do
      val assignments = step.at("/command/rpc/assign").elements().asScala.toSeq
      assertEquals(
        assignments.map(_.path("target").asText()).distinct.size,
        assignments.size
      )
      val maximum =
        assignments
          .find(_.path("target").asText() == "retry_policy.maximum_attempts")
          .get
      val expected = step.at("/step/inputs/5/enum/case").asText() match
        case "unlimited" => "0"
        case "one"       => "1"
        case "two"       => "2"
      assertEquals(maximum.at("/value/literal/number").asText(), expected)
    assertEquals(starts.map(_.at("/command/id").asText()).distinct.size, 1)

  // Over the mini's two attempt presets (fn-166.11; the standalone activity's three, with
  // `unlimited`, bound six).
  test("a preset catalog binds its four distinct classes and selects each once"):
    val catalog = items("fixture.deadlinepresets.Catalog")
    val bound = catalog
      .dropRight(1)
      .flatMap(_.path("performs").elements().asScala)
      .map(_.path("step"))
    val selected = catalog.last.path("when").elements().asScala.toSeq
    assertEquals(bound.size, 4)
    assertEquals(selected, bound)
    assertEquals(selected.distinct.size, 4)

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
        .collect {
          case (input, field) if input.at("/enum/case").asText() == "expires" =>
            field
        }
        .toSet
      val assignments = step.at("/command/rpc/assign").elements().asScala.toSeq
      assertEquals(
        assignments.map(_.path("target").asText()).toSet,
        expected ++ Set("namespace", "activity_id")
      )
      for
        assignment <- assignments
        if expected(assignment.path("target").asText())
      do assertEquals(assignment.at("/value/literal/number").asText(), "2")

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

  // fn-162: the retry after an attempt timeout runs on the activity Protocol itself, through its own
  // realization, which names the Queries it runs.
  test(
    SweepHeld(
      "the timeout retry's realization runs its Queries with only its own occurrence evidence",
      "features/activity/standalone"
    )
  ):
    val prefix = "temporal.features.activity.standalone.protocol."
    val (ran, out) = lift(
      prefix + "Timeouts.timeouts$.queries",
      "temporal.features.activity.standalone.realization.RetryAfterTimeoutRun",
      prefix + "ActivityProtocol$.queries",
      prefix + "Retry.retry$.queries",
      "temporal.features.activity.standalone.realization.StandaloneActivityRun"
    )
    ran.orFail(): Unit
    val model = mapper.readTree(Files.readString(out))
    def named(kind: String, name: String) = model
      .path(kind)
      .elements()
      .asScala
      .find(_.path("name").asText() == name)
      .get
    val failures = named("queries", "retryExhaustionByFailures")
    assertEquals(failures.path("form").asText(), "FORM_VERIFY")
    assert(!failures.has("expectedRun"))
    val failurePath =
      named("scenarios", "exhausted").path("actions").elements().asScala.toSeq
    assertEquals(failurePath.size, 6)
    assertEquals(
      failurePath.count(
        _.path("action").asText().endsWith("worker.respondFailed")
      ),
      2
    )
    val exhaustion = named("queries", "retryExhaustion")
    assertEquals(
      exhaustion.at("/scenario/machine").asText(),
      "activityProtocol"
    )
    val exhaustionPath = named("scenarios", "retryExhaustion").path("actions")
    assertEquals(exhaustionPath.size(), 6)
    assert(
      exhaustionPath
        .path(2)
        .path("action")
        .asText()
        .endsWith("deadline.startToClose")
    )
    assert(
      exhaustionPath
        .path(5)
        .path("action")
        .asText()
        .endsWith("worker.respondFailed")
    )
    val realization = named("realizations", "retryAfterTimeoutRun")
    assertEquals(realization.path("machine").asText(), "activityProtocol")
    assertEquals(
      realization.path("queries").elements().asScala.map(_.asText()).toSeq,
      Seq("retryAfterTimeout", "retryExhaustion")
    )
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
      .find(
        _.at("/runEvent/attempt/number").asText() == "2"
      )
      .get
    val confirms = second.path("confirms").elements().asScala.toSeq
    assertEquals(
      confirms.map(_.at("/step/action").asText()),
      Seq(timer, "temporal.features.activity.worker.poll")
    )
    assertEquals(confirms.map(_.path("occurrence").asText()), Seq("1", "2"))
    assert(!realization.path("evidence").toString.contains("statusTimedOut"))
    val proof =
      tools.directory.resolve(".flow/tmp/activity-batch/fn1283-focused.json")
    Files.createDirectories(proof.getParent)
    Files.copy(
      out,
      proof,
      java.nio.file.StandardCopyOption.REPLACE_EXISTING
    ): Unit
