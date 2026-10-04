package umpire.lift

import java.nio.file.{Files, Path}
import java.util.concurrent.Semaphore
import scala.collection.mutable
import scala.concurrent.{blocking, Await, Future}
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.gate.{Ran, Tools}

/**
 * The lifter's fixtures under testdata: the Models it must lift, compared with the IR in
 * testdata/lifts/expected, and the declarations the build or the lifter must refuse, at their lines.
 *
 * The fixtures build against the framework and the Temporal Models the gate packaged into
 * model/gen, so the gate runs these tests after it packaged them. With UMPIRE_LIFTER_UPDATE set, as
 * the gate's --update sets it, the expected files are rewritten instead of compared.
 */
class Fixtures extends munit.FunSuite:
  // A test builds a fixture with scala-cli and lifts it in a JVM of its own.
  override val munitTimeout: Duration = 20.minutes

  private val tools = Tools.here
  private val root = tools.directory
  private val gen = root.resolve("model/gen")
  private val testdata = root.resolve("model/lifter/testdata")
  private val expected = testdata.resolve("lifts/expected")
  private val modelJar = gen.resolve("model-scala.jar")
  private val modelClasspath = gen.resolve("model-scala.classpath")
  private val update = sys.env.get("UMPIRE_LIFTER_UPDATE").exists(_.nonEmpty)

  // Each run builds in a directory of its own under the ignored model/gen/history, which is kept
  // for inspection: nothing is deleted, so another process's build state is never removed.
  private lazy val scratch =
    Files.createTempDirectory(Files.createDirectories(gen.resolve("history")), "lifter.")
  private def lifted(name: String) = scratch.resolve(s"$name.json")

  // Every build and lifter JVM holds one of these while it runs, so the tests run side by side
  // within the memory of the machine. A test awaiting a lift holds none, so none waits on itself.
  private val slots = Semaphore(6, true)
  private def bounded[A](work: => A): A =
    slots.acquire()
    try work
    finally slots.release()

  // A test that builds or lifts is started in beforeAll, after the fixtures' lifts, and runs beside
  // the others; the test awaits it, so its assertions and its failure stay its own.
  private val started = mutable.ArrayBuffer.empty[() => Unit]
  private def concurrently(name: String)(body: => Unit)(using munit.Location): Unit =
    lazy val running = Future(blocking(body))
    started += (() => running: Unit)
    test(name)(Await.result(running, munitTimeout))

  // A fixture's sources are stored under testdata, which the lifter's project.scala excludes from
  // its build, since some of them must not compile. `materialize` copies them into the scratch
  // directory, with its project's jar path resolved. The lifter maps the copies' positions back to
  // the stored files, and the build's diagnostics are mapped back in `refusals`.
  private def materialize(fixture: String): Path =
    val to = Files.createDirectories(scratch.resolve(fixture))
    val stream = Files.newDirectoryStream(testdata.resolve(fixture), "*.scala")
    try
      for from <- stream.asScala do
        Files.writeString(
          to.resolve(from.getFileName.toString),
          Files
            .readString(from)
            .replace("../../../gen/model-scala.jar", modelJar.toString)
            .replace("../../../gen/api-scalapb.jar", gen.resolve("api-scalapb.jar").toString)
        )
    finally stream.close()
    to

  // The lifter's positions of a materialized fixture: its stored files.
  private def stored(fixture: String) = s"model/lifter/testdata/$fixture/"

  /** The lines of the stored files at which the fixture's build fails, as `<file>:<line>:<column>`. */
  private def refusals(fixture: String): Seq[String] =
    val built = bounded(tools.scalaCli(Seq("compile", materialize(fixture).toString)))
    assert(built.failed, s"model/lifter/testdata/$fixture built")
    val at = """\[error\] \S*/([^/\s:]+\.scala):(\d+):(\d+)""".r
    built.errors.collect { case at(file, line, column) =>
      assert(Files.isRegularFile(testdata.resolve(fixture).resolve(file)), built.output)
      s"$file:$line:$column"
    }

  private def packaged(name: String, sources: Path): Path =
    val jar = scratch.resolve(s"$name.jar")
    bounded(
      tools.scalaCli(
        Seq("--power", "package", "--library", sources.toString, "-f", "-o", jar.toString)
      )
    ).orFail()
    jar

  // The lifter is run as `lift` runs it, in a JVM of its own, since it ends the JVM on a refusal:
  // this build's classes, and the arguments of its command line.
  private def lift(arguments: String*): Ran =
    val java = Path.of(System.getProperty("java.home"), "bin", "java").toString
    bounded(
      tools.run(
        java,
        Seq("-cp", System.getProperty("java.class.path"), "umpire.lift.lift") ++ arguments
      )
    )

  private def refused(ran: Ran): Seq[String] =
    ran.output.linesIterator.filter(_.startsWith("lift:")).toSeq

  private val fixtures: Seq[(String, Seq[String])] = Seq(
    "presence" -> Seq("fixture.presence.Presence$package$.presence"),
    "channels" -> Seq(
      "fixture.channels.Channels$package$.relay",
      "fixture.channels.Channels$package$.tallying"
    ),
    "declarations" -> Seq(
      "fixture.declarations.Declarations$package$.queries",
      "fixture.declarations.Declarations$package$.durableEventually"
    ),
    "admission" -> Seq(
      "fixture.specimens.admission.Admission$package$.currentQueries",
      "fixture.specimens.admission.Admission$package$.staleQueries"
    ),
    "closereset" -> Seq(
      "fixture.specimens.closereset.CloseReset$package$.rejectAfterCloseQueries",
      "fixture.specimens.closereset.CloseReset$package$.ackByOriginalQueries",
      "fixture.specimens.closereset.CloseReset$package$.retainAndRouteQueries"
    ),
    "realizations" -> Seq(
      "fixture.realizations.Realizations$package$.learnedRun",
      "temporal.nexuscaller.Claims$package$.syncCompletion",
      "fixture.realizations.Realizations$package$.pauseRace",
      "fixture.realizations.Realizations$package$.pauseRaceQuery",
      "fixture.realizations.Realizations$package$.doorRealization",
      "fixture.realizations.Realizations$package$.doorOpens",
      "fixture.realizations.Realizations$package$.errandRealization",
      "fixture.realizations.Realizations$package$.errandRetry",
      "fixture.realizations.Realizations$package$.errandWithdrawn",
      "fixture.realizations.Realizations$package$.tallyRealization",
      "fixture.realizations.Realizations$package$.tallyOpens"
    )
  )
  // The refusals of fn-112.4's typed composition selectors, in lifts/Rejects.scala.
  private val selectorRejects: Seq[String] = Seq(
    "memberNoField",
    "syncNoMember",
    "memberIncompatible",
    "syncUnbound",
    "withNoMember",
    "withIncompatible",
    "withUnrefined",
    "withUnsynced",
    "syncedNone",
    "syncedTwice",
    "ownSynced",
    "ownSpelledAlike",
    "mixedSchedule",
    "bareInputs",
    "whenClassInputs"
  )
  // The refusals of fn-112.4's claim patterns, records over a member and function-valued arguments.
  private val patternRejects: Seq[String] = Seq(
    "whenStays",
    "keptComputed",
    "fromVal",
    "sharedLambda",
    "recordsNoMember",
    "recordsForeignFact",
    "paramCalled"
  )

  private val rejected = Seq(
    "unbounded",
    "waiting",
    "doubled",
    "listening",
    "counter",
    "crossedRead",
    "negative",
    "watched",
    "unrefined",
    "misplaced",
    "noSuchRoot",
    "unrefinedOutcomes",
    "counting",
    "batching",
    "shuffling",
    "guessing",
    "sharedIds",
    "unnamedInList",
    "unnamedProperty",
    "twins",
    "askedTwice",
    "boundTwice",
    "tapped",
    "unstarted",
    "unstartedPair",
    "undeclared",
    "unlisted",
    "partial",
    "unrelatedRead",
    "assumedTwice",
    "gapTwice",
    "rebindUnbound",
    "extendBound",
    "reboundTwice",
    "assumedAgain",
    "assumingTwice",
    "refinedNothing",
    "refinedOtherwise",
    "loopFirst",
    "aliased",
    "splatted",
    "explained"
  ).map("fixture.rejects.Rejects$package$." + _) ++ (selectorRejects ++ patternRejects).map(
    "fixture.rejects.Rejects$package$." + _
  ) ++ Seq(
    // DefinitionScope pins, a name the compiler made up and a computed accepted outcome, refused in
    // objects of their own.
    "PinnedTwice$.pinnedTwice",
    "PinsOuter$.pinnedNested",
    "Self$.pinnedSelf",
    "Computed$.pinnedComputed",
    "Anonymous$.anonymous",
    "ComputedAccepted$.computedAccept"
  ).map("fixture.rejects." + _)

  private lazy val liftsJar = packaged("lifts", materialize("lifts"))
  private lazy val liftsJars = s"$liftsJar=${stored("lifts")},$modelJar=model/"

  // Each lift is its own JVM, so they run side by side.
  private lazy val lifts: Map[String, Future[Ran]] =
    val jars = liftsJars
    (fixtures :+ ("rejects" -> rejected)).map { (name, roots) =>
      name -> Future(
        blocking(lift((Seq(jars, modelClasspath.toString, lifted(name).toString) ++ roots)*))
      )
    }.toMap
  private def ran(name: String): Ran = Await.result(lifts(name), munitTimeout)

  /** The IR a fixture lifted to. */
  private def ir(name: String): String =
    val lift = ran(name)
    assert(!lift.failed, s"the $name fixture did not lift:\n${lift.diagnostics}")
    Files.readString(lifted(name))

  /** The declarations the lifter refused, one line each, and no IR. */
  private def rejections(): String =
    val lift = ran("rejects")
    assert(
      lift.exit != 0 && !Files.exists(lifted("rejects")),
      "the lifter wrote the rejected declarations' IR"
    )
    refused(lift).map(_ + "\n").mkString

  /** The files of the expected directory that no fixture lifts to. */
  private def leftOver(): Seq[String] =
    val stream = Files.list(expected)
    val held =
      try stream.iterator.asScala.map(_.getFileName.toString).toList.sorted
      finally stream.close()
    held.diff(fixtures.map(_._1 + ".json") :+ "rejects.txt")

  // An update rewrites the expected files only once every fixture lifted, every rejected
  // declaration was refused and no expected file is left over, so the tree it leaves is one whole
  // run's. The gate holds model/ir to the same rule.
  private lazy val everyLift: Unit =
    fixtures.foreach((name, _) => ir(name))
    rejections(): Unit
    assertEquals(leftOver(), Nil, s"${root.relativize(expected)} holds files no fixture lifts to")

  private def expect(file: String, lifted: => String): Unit =
    val path = expected.resolve(file)
    if update then
      everyLift
      if !Files.exists(path) || Files.readString(path) != lifted then
        Files.writeString(path, lifted): Unit
    else
      assert(
        Files.isRegularFile(path) && Files.readString(path) == lifted,
        s"${root.relativize(path)} is stale; make umpire-gen-model rewrites it. Lifted: $scratch"
      )

  override def beforeAll(): Unit =
    for input <- Seq(modelJar, modelClasspath) do
      assert(
        Files.isRegularFile(input),
        s"$input is missing: the gate packages the Models before it runs these tests (make umpire-check-model)"
      )
    lifts: Unit
    started.foreach(_())

  concurrently("the build refuses a warning -Werror makes an error, at its line"):
    assertEquals(refusals("werror"), Seq("Steps.scala:21:58"))

  concurrently("the build refuses crossed types, at their lines"):
    assertEquals(
      refusals("crossed").sorted,
      Seq(
        "ActionInput.scala:15:28",
        "Crossed.scala:35:14",
        "Crossed.scala:45:28",
        "Crossed.scala:49:24",
        "Crossed.scala:59:29"
      )
    )

  concurrently("the build refuses a non-finite state field, at its line"):
    assertEquals(refusals("nonfinite"), Seq("NonFinite.scala:5:47"))

  concurrently("typed API declarations and direct constructors refuse mismatched roots"):
    assertEquals(
      refusals("typedInvalid").sorted,
      Seq(
        "Invalid.scala:107:7",
        "Invalid.scala:118:7",
        "Invalid.scala:128:3",
        "Invalid.scala:132:3",
        "Invalid.scala:136:68",
        "Invalid.scala:137:69",
        "Invalid.scala:140:3",
        "Invalid.scala:143:49",
        "Invalid.scala:144:3",
        "Invalid.scala:146:46",
        "Invalid.scala:155:3",
        "Invalid.scala:157:31",
        "Invalid.scala:160:3",
        "Invalid.scala:165:49",
        "Invalid.scala:22:63",
        "Invalid.scala:29:7",
        "Invalid.scala:43:9",
        "Invalid.scala:66:7",
        "Invalid.scala:81:21",
        "Invalid.scala:84:3",
        "Invalid.scala:89:5",
        "Invalid.scala:92:3",
        "Invalid.scala:93:5",
        "Invalid.scala:97:7"
      ).sorted
    )

  concurrently("only the unknown projected origin admits a dynamic message root"):
    assertEquals(refusals("dynamicInvalid"), Seq("Invalid.scala:10:16"))

  concurrently("retired string proto constructors refuse direct and helper-built names"):
    val positions = refusals("retiredInvalid")
    assert(positions.size >= 12, positions.mkString(", "))
    assert(positions.exists(_.startsWith("Invalid.scala:15:")), positions.mkString(", "))
    assert(positions.exists(_.startsWith("Invalid.scala:16:")), positions.mkString(", "))
    assert(positions.exists(_.startsWith("Invalid.scala:29:")), positions.mkString(", "))

  concurrently("typed protobuf constants refuse mismatched fields, values and forged carriers"):
    assertEquals(
      refusals("typedProtoInvalid").sorted,
      Seq(
        "Invalid.scala:11:20",
        "Invalid.scala:14:55",
        "Invalid.scala:19:26",
        "Invalid.scala:25:24",
        "Invalid.scala:28:36",
        "Invalid.scala:29:50",
        "Invalid.scala:30:19",
        "Invalid.scala:31:75",
        "Invalid.scala:32:75",
        "Invalid.scala:36:56",
        "Invalid.scala:39:45",
        "Invalid.scala:40:23",
        "Invalid.scala:41:25"
      ).sorted
    )

  concurrently("typed schemas, methods, paths, enums and constants lift to their protobuf names"):
    val out = lifted("typed")
    val roots = Seq("typedMachine", "typedRealization")
      .map("fixture.typed.Typed$package$." + _)
    val result = lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
    assert(!result.failed, result.diagnostics)
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val model = mapper.readTree(Files.readString(out))
    val actions = model.path("actions")
    assertEquals(actions.size(), 2)
    assertEquals(
      actions.get(0).path("schemas").get(0).asText(),
      "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    )
    assertEquals(
      actions.get(1).path("schemas").elements().asScala.map(_.asText()).toList,
      List(
        "temporal.api.workflowservice.v1.StartActivityExecutionResponse",
        "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
      )
    )
    val realizations = model.path("realizations")
    assertEquals(realizations.size(), 1)
    val typed = realizations.get(0)
    assertEquals(typed.path("evidence").get(0).path("read").path("path").asText(), "executions[*]")
    assertEquals(
      typed.at("/scripts/0/items/0/command/rpc/method").asText(),
      "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"
    )
    assertEquals(
      typed.at("/scripts/0/items/0/command/rpc/assign/1/target").asText(),
      "task_queue.name"
    )
    assertEquals(typed.at("/scripts/0/items/0/command/rpc/reads/0/path").asText(), "run_id")
    assertEquals(
      typed.at("/scripts/0/items/1/command/poll/evidence").asText(),
      "fixture.typed.evidence.listed"
    )
    assertEquals(
      typed.at("/scripts/0/items/2/command/rpc/reads/0/path").asText(),
      "history.events[*].event_id"
    )
    assertEquals(
      typed.at("/evidence/2/history").asText(),
      "nexus_operation_scheduled_event_attributes"
    )
    assertEquals(
      typed.at("/evidence/3/runEvent/guard/all/operands/0/equal/right/literal/enumName").asText(),
      "DELIVERY_ADMISSION_DECISION_ADMITTED"
    )
    assertEquals(
      typed.at("/scripts/0/items/3/command/nexusReply/reply/message").asText(),
      "temporal.api.common.v1.Payload"
    )
    assertEquals(
      typed.at("/scripts/0/items/3/command/nexusReply/reply/fields/0/name").asText(),
      "metadata"
    )
    assertEquals(
      typed
        .at("/scripts/0/items/3/command/nexusReply/reply/fields/0/value/mapping/entries/0/key")
        .asText(),
      "encoding"
    )
    assertEquals(
      typed
        .at(
          "/scripts/0/items/3/command/nexusReply/reply/fields/0/value/mapping/entries/0/value/utf8"
        )
        .asText(),
      "json/plain"
    )

  concurrently(
    "typed Long operands lift as protobuf integer values, including bound helper values"
  ):
    val out = lifted("typedLong")
    val roots = Seq("fixture.typed.Typed$package$.typedLongRealization")
    val result = lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
    assert(!result.failed, result.diagnostics)
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val realizations = mapper.readTree(Files.readString(out)).path("realizations")
    assertEquals(realizations.size(), 1)
    val rpc = realizations.get(0).at("/scripts/0/items/0/command/rpc")
    assertEquals(
      rpc.path("assign").get(0).path("target").asText(),
      "start_to_close_timeout.seconds"
    )
    assertEquals(rpc.at("/assign/0/value/literal/number").asText(), "300")
    assertEquals(
      rpc.path("assign").get(1).path("target").asText(),
      "schedule_to_start_timeout.seconds"
    )
    assertEquals(rpc.at("/assign/1/value/literal/number").asText(), "2")

  test("typed repeated reads preserve bare paths"):
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val bare = mapper
      .readTree(ir("realizations"))
      .path("realizations")
      .get(1)
      .path("evidence")
      .get(0)
      .path("read")
      .path("path")
      .asText()
    assertEquals(bare, "executions")

  concurrently("the lifter refuses a mapped read ending at a singular message"):
    val out = lifted("typedMapped")
    val result = lift(
      liftsJars,
      modelClasspath.toString,
      out.toString,
      "fixture.typed.Typed$package$.invalidMappedRealization"
    )
    assertNotEquals(result.exit, 0)
    assert(!Files.exists(out), "the lifter wrote an invalid mapped read")
    assert(
      refused(result).exists(_.contains("Recorded.read must end at a repeated message field")),
      result.diagnostics
    )

  concurrently("a generated enum helper refuses an unknown value before writing IR"):
    val out = lifted("typedUnknownEnum")
    val result = lift(
      liftsJars,
      modelClasspath.toString,
      out.toString,
      "fixture.typed.Typed$package$.unknownEnumRealization"
    )
    assertNotEquals(result.exit, 0)
    assert(!Files.exists(out), "the lifter wrote an unknown generated enum")
    assert(
      refused(result).exists(_.contains("expected a generated enum case")),
      result.diagnostics
    )

  // The same Model spelled out (Spelled.scala) and with names taken from vals, a given family, defaulted
  // starts and evidence and a refinement read with no given (Captured.scala): one IR, but for positions
  // and the owner of the types and functions each file declares. Captured.scala pins Spelled.scala's
  // owner, so its symbol-based Definition IDs are Spelled.scala's before anything is substituted.
  concurrently("captured names, a given family and defaults lift to the IR spelled-out forms do"):
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    def spelledAs(fixture: String): String =
      val pkg = s"fixture.$fixture.${fixture.capitalize}$$package$$"
      val out = lifted(fixture)
      val roots = Seq(
        "queries",
        "diskQueries",
        "localDiskQueries",
        "relay",
        "ledger",
        "putOnly",
        "durableEventually"
      ).map(s"$pkg." + _)
      val result = lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
      assert(!result.failed, result.diagnostics)
      Files.readString(out)
    def strip(n: com.fasterxml.jackson.databind.JsonNode): Unit =
      n match
        case o: com.fasterxml.jackson.databind.node.ObjectNode =>
          o.remove(java.util.List.of("position", "source")): Unit
        case _ => ()
      n.elements().asScala.foreach(strip)
    val captured = spelledAs("captured")
    val ids = mapper.readTree(captured)
    for
      kind <- Seq("actions", "monitors", "assumptions", "holes", "channels", "realizations")
      id = ids.path(kind)
    do
      assert(id.size() > 0, s"Captured.scala declares no $kind")
      for d <- id.elements().asScala do
        assert(
          d.path("id").asText().startsWith("fixture.spelled.Spelled$package$."),
          s"${d.path("id").asText()} is not pinned to Spelled.scala's owner"
        )
    val spelled = mapper.readTree(spelledAs("spelled"))
    val same = mapper.readTree(
      captured
        .replace("fixture.captured.Captured$package$", "fixture.spelled.Spelled$package$")
        .replace("fixture.captured.", "fixture.spelled.")
    )
    strip(spelled)
    strip(same)
    assertEquals(same.toPrettyString, spelled.toPrettyString)

  /**
   * The machines and Properties of one lift of `roots` of the lifts fixture `fixture`, by name, each
   * as a pair compares it: without its name, its machine and positions, and with each function it
   * refers to in place of the function's name, so two spellings may name their functions apart. A
   * root of the Temporal Models is named in full.
   */
  private def declarations(
      fixture: String,
      roots: Seq[String]
  ): (com.fasterxml.jackson.databind.JsonNode, String => String, String => String) =
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val pkg = s"fixture.$fixture.${fixture.capitalize}$$package$$"
    val out = lifted(s"$fixture-pairs")
    val named = roots.map(r => if r.startsWith("temporal.") then r else s"$pkg.$r")
    val result = lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ named)*)
    assert(!result.failed, result.diagnostics)
    val model = mapper.readTree(Files.readString(out))
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode => o.remove(java.util.List.of("position", "source")): Unit
        case _             => ()
      n.elements().asScala.foreach(strip)
    val functions = model
      .path("functions")
      .elements()
      .asScala
      .map { f =>
        val body = f.deepCopy[ObjectNode]()
        body.remove("name")
        strip(body)
        f.path("name").asText() -> body
      }
      .toMap
    def deref(o: JsonNode, fields: String*): Unit = o match
      case o: ObjectNode =>
        for field <- fields if o.path(field).asText().nonEmpty do
          o.set(field, functions(o.path(field).asText())): Unit
      case _ => ()
    def find(kind: String, name: String): ObjectNode =
      model
        .path(kind)
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"the $fixture fixture lifted no $kind named $name"))
        .deepCopy[ObjectNode]()
    def machine(name: String): String =
      val m = find("machines", name)
      m.remove(java.util.List.of("name"))
      deref(m, "evidence")
      m.path("steps").elements().asScala.foreach(deref(_, "function"))
      deref(m.path("refines"), "map", "visible", "visibleOutcomes")
      strip(m)
      m.toPrettyString
    def property(name: String): String =
      val p = find("properties", name)
      p.remove(java.util.List.of("name", "machine"))
      deref(p, "holds")
      strip(p)
      p.toPrettyString
    (model, machine, property)

  // Each derived machine beside the machine it stands for, spelled out (lifts/Derived.scala).
  concurrently(
    "rebind, extend, refining, assuming and unmonitored lift to the machines they stand for"
  ):
    val pairs = Seq(
      "stiffLamp" -> "stiffLampSpelled",
      "faultyLamp" -> "faultyLampSpelled",
      "plainLamp" -> "plainLampSpelled",
      "plainStiffLamp" -> "plainStiffLampSpelled",
      "stiffPressOnly" -> "stiffPressOnlySpelled"
    )
    val (model, machine, _) = declarations("derived", pairs.flatMap((a, b) => Seq(a, b)))
    for (derived, spelled) <- pairs do assertEquals(machine(derived), machine(spelled), derived)
    def named(name: String) =
      model.path("machines").elements().asScala.find(_.path("name").asText() == name).get
    def actions(name: String) = named(name)
      .path("steps")
      .elements()
      .asScala
      .map(s => s.path("action").asText().split('.').last)
      .toList
    assertEquals(actions("stiffLamp"), List("press", "wear"), "a rebound action keeps its place")
    assertEquals(actions("faultyLamp"), List("press", "wear", "burnOut"))
    assertEquals(named("faultyLamp").path("family").asText(), "fixture.derived")
    assertEquals(
      named("faultyLamp").path("assumes").elements().asScala.map(_.asText().split('.').last).toList,
      List("lampOpaque", "burnOutAssumed")
    )
    assertEquals(named("faultyLamp").path("refines").path("product").asText(), "viewUnderFaults")
    assert(!named("plainLamp").has("monitors") && !named("plainLamp").has("refines"))
    assertEquals(
      named("plainStiffLamp").path("steps").get(1).path("function").asText(),
      "plainStiffLamp.wear",
      "a lambda a derivation binds is named after the machine it declares"
    )

  // Each sugar form beside its core spelling (lifts/Sugar.scala).
  concurrently(
    "accept, stay, disabled, because, in, implies and records lift as their core forms do"
  ):
    val (model, machine, property) = declarations("sugar", Seq("sugared", "cored", "claims"))
    assertEquals(machine("sugared"), machine("cored"))
    for form <- Seq("records", "implies", "paused") do
      assertEquals(property(s"${form}Sugar"), property(s"${form}Core"), form)
    val functions = model.path("functions").elements().asScala.map(_.path("name").asText()).toList
    assert(
      functions.forall(!_.startsWith("umpire.")),
      s"a framework definition was lifted as a function: ${functions.mkString(", ")}"
    )
    val resume = model
      .path("functions")
      .elements()
      .asScala
      .find(_.path("name").asText().endsWith(".resumeSugar"))
      .get
    assertEquals(
      resume.at("/body/match/cases/1/pattern").toString,
      """{"wildcard":{}}""",
      "a wildcard arm lifts as a wildcard"
    )

  // fn-112.4: typed composition selectors (lifts/Members.scala).
  // Each composition of temporal/standaloneactivity beside its typed twin, the later designs derived
  // by `withMember`; the Scenarios and the `whenAction` whose keys `synced` and `own` select; and the
  // keys of names that carry separators, over two members that bind actions spelled alike.
  concurrently("typed members, syncs, replaces, withMember, synced and own lift as strings do"):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val system = "temporal.standaloneactivity.System$package$."
    val queried = Seq(
      "currentOverQueue",
      "staleOverQueue",
      "currentOverMatching",
      "staleOverMatching",
      "currentOverLossyMatching"
    )
    val unqueried = Seq("currentOverForgetful", "currentOverVolatile")
    val roots = queried.flatMap(d => Seq(s"${d}TypedQueries", s"$system${d}Queries")) ++
      unqueried.flatMap(d => Seq(s"${d}Typed", s"$system$d")) ++ Seq(
        "stoppedWorkerStartsNothingTyped",
        "temporal.standaloneactivity.Claims$package$.stoppedWorkerStartsNothing",
        "switchQueries"
      )
    val (model, _, property) = declarations("members", roots)
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode => o.remove(java.util.List.of("position", "source")): Unit
        case _             => ()
      n.elements().asScala.foreach(strip)
    def all(kind: String) = model.path(kind).elements().asScala.toList
    def of(kind: String, matches: JsonNode => Boolean, without: String*): String =
      val found = all(kind)
        .find(matches)
        .getOrElse(fail(s"the members fixture lifted no such $kind"))
        .deepCopy[ObjectNode]()
      found.remove(without.asJava)
      strip(found)
      found.toPrettyString
    def named(name: String)(n: JsonNode) = n.path("name").asText() == name
    def composition(name: String) = of("compositions", named(name), "name")
    for d <- queried ++ unqueried do assertEquals(composition(s"${d}Typed"), composition(d), d)
    def replaces(name: String) =
      all("compositions").find(named(name)).get.path("members").get(1).path("replaces").asText()
    assertEquals(replaces("currentOverForgetfulTyped"), "dispatchQueue")
    assertEquals(replaces("currentOverLossyMatchingTyped"), "dispatchQueueUnderStorageLoss")
    def scenario(machine: String, name: String) =
      of("scenarios", s => s.path("machine").asText() == machine && named(name)(s), "machine")
    val typed = all("scenarios")
      .map(s => s.path("machine").asText() -> s.path("name").asText())
      .filter(_._1.endsWith("Typed"))
    assertEquals(typed.size, 2 * 3 + 3 * 4, "a typed twin of every keyed Scenario")
    for (machine, name) <- typed do
      assertEquals(scenario(machine, name), scenario(machine.stripSuffix("Typed"), name), name)
    assertEquals(property("startedByPollingWorkerTyped"), property("startedByPollingWorker"))
    def keys(name: String) =
      all("scenarios").find(named(name)).get.path("keys").elements().asScala.map(_.asText()).toList
    assertEquals(
      keys("switchSchedule"),
      List("tap_both-ways", "left_side_turn-on-high", "left_side_flick", "right_side_flick")
    )
    def whenAction(name: String) =
      all("properties").find(named(name)).get.path("whenAction").asText()
    assertEquals(whenAction("turnedOn"), "left_side_turn-on")
    assertEquals(whenAction("tappedBoth"), "tap_both-ways")
    def bound(machine: String) = all("machines")
      .find(named(machine))
      .get
      .path("steps")
      .elements()
      .asScala
      .map(_.path("action").asText().stripPrefix("fixture.members."))
      .toList
    assertEquals(bound("leftSwitch"), List("Left$.tap", "Left$.flick", "Members$package$.turnOn"))
    assertEquals(bound("rightSwitch"), List("Right$.tap", "Right$.flick"))

  // fn-112.4: claim patterns and shared claims (lifts/Patterns.scala).
  // Each claim pattern beside its lambda spelling, on the machine and on the composition, and each
  // claim written once over `Declares[S]` beside the lambda it stands for on either.
  concurrently(
    "once/keeps, never, never/from, stays, stays/unless and records over a member lift as their lambdas do"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.{ArrayNode, ObjectNode}
    val (model, _, _) = declarations("patterns", Seq("claims"))
    val functions =
      model.path("functions").elements().asScala.map(f => f.path("name").asText() -> f).toMap
    // Each child of a tree rewritten, the tree itself kept unless `f` replaces it.
    def rewrite(n: JsonNode)(f: PartialFunction[JsonNode, JsonNode]): JsonNode =
      f.applyOrElse(
        n,
        {
          case o: ObjectNode =>
            for k <- o.fieldNames().asScala.toList do o.set(k, rewrite(o.get(k))(f)): Unit
            o
          case a: ArrayNode =>
            for i <- 0 until a.size do a.set(i, rewrite(a.get(i))(f)): Unit
            a
          case other => other
        }
      )
    // A pattern lifts a lambda of its own as a function named after the Property's and the word that
    // takes it, `<holds>.never`, as `holds` lifts one; such a call is read as the lambda's body, so a
    // claim written with one compares with the lambda that calls the def inside it.
    def inlined(n: JsonNode, holds: String): JsonNode = rewrite(n) {
      case o: ObjectNode if o.path("call").path("function").asText().startsWith(s"$holds.") =>
        val f = functions(o.path("call").path("function").asText())
        val param = f.path("params").get(0).path("name").asText()
        val arg = inlined(o.path("call").path("args").get(0), holds)
        rewrite(f.path("body").deepCopy[JsonNode]()) {
          case v: ObjectNode if v.path("var").asText() == param => arg.deepCopy[JsonNode]()
        }
    }
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode => o.remove(java.util.List.of("position", "source")): Unit
        case _             => ()
      n.elements().asScala.foreach(strip)
    def found(machine: String, name: String): JsonNode =
      model
        .path("properties")
        .elements()
        .asScala
        .find(p => p.path("machine").asText() == machine && p.path("name").asText() == name)
        .getOrElse(fail(s"the patterns fixture lifted no Property $name of $machine"))
    def property(machine: String, name: String): String =
      val p = found(machine, name).deepCopy[ObjectNode]()
      val holds = p.path("holds").asText()
      val f = functions(holds).deepCopy[ObjectNode]()
      f.remove("name")
      p.set("holds", inlined(f, holds)): Unit
      p.remove(java.util.List.of("name", "machine"))
      strip(p)
      p.toPrettyString
    // Each form by its Property on the job and on the pair, and whether it is a transition one.
    val forms = Seq(
      ("doneKeeps", "pairDoneKeeps", true),
      ("noTwo", "pairNoTwo", false),
      ("notStartedWhilePaused", "pairNotStartedWhilePaused", true),
      ("doneStays", "pairDoneStays", true),
      ("activeStays", "pairActiveStays", true)
    )
    val shared =
      Seq("notAdmittedWhilePaused" -> true, "atMostOneActive" -> false, "terminalStays" -> true)
    for
      (onJob, onPair, transition) <- forms
      (machine, name) <- Seq("job" -> onJob, "pair" -> onPair)
    do
      assertEquals(property(machine, name), property(machine, s"${name}Core"), name)
      assertEquals(found(machine, name).path("transition").asBoolean, transition, name)
    for name <- Seq("pairStarted", "lampFlipped") do
      assertEquals(property("pair", name), property("pair", s"${name}Core"), name)
    for (name, transition) <- shared; machine <- Seq("job", "pair") do
      assertEquals(property(machine, name), property(machine, s"${name}Core"), s"$machine.$name")
      assertEquals(found(machine, name).path("transition").asBoolean, transition, name)
    for (name, transition) <- Seq(
        "doneKeepsInline" -> true,
        "noTwoInline" -> false,
        "notStartedInline" -> true,
        "doneStaysInline" -> true,
        "activeStaysInline" -> true
      )
    do assertEquals(found("job", name).path("transition").asBoolean, transition, name)
    val names = functions.keys.toList.sorted
    assert(
      names.forall(!_.startsWith("umpire.")),
      s"a framework definition was lifted as a function: ${names.mkString(", ")}"
    )

  concurrently("the lifter refuses unrelated machines with one state type, at the Query line"):
    val jar = packaged("samestate", materialize("samestate"))
    val lift = this.lift(
      s"$jar=${stored("samestate")}",
      modelClasspath.toString,
      "/dev/null",
      "fixture.samestate.SameState$package$.wrongPair"
    )
    assertNotEquals(lift.exit, 0)
    assertEquals(
      refused(lift),
      Seq(
        "lift: model/lifter/testdata/samestate/SameState.scala:24: wrongPair pairs property, a Property of first, with scenario, a Scenario of second, and reads it through no refinement"
      )
    )

  concurrently("the lifter refuses a construct outside the subset, at its line"):
    val jar = packaged("unsupported", materialize("unsupported"))
    val lift = this.lift(
      s"$jar=${stored("unsupported")}",
      modelClasspath.toString,
      "/dev/null",
      "temporal.fixture.Unsupported$package$.unsupported"
    )
    assertNotEquals(lift.exit, 0)
    val line =
      "lift: model/lifter/testdata/unsupported/Unsupported.scala:18: `var out` has no IR form"
    refused(lift) match
      case Seq(refusal) => assert(refusal.startsWith(line), refusal)
      case refusals     => fail(s"one refusal, not $refusals")

  concurrently("the realization emitter refuses unknown constructors and fields at their lines"):
    val jar = packaged("realizationRefusals", materialize("realizationRefusals"))
    val cases = Seq(
      (
        "unknownConstructor",
        "lift: model/lifter/testdata/realizationRefusals/Refusals.scala:7: Unknown is no activation of Script in the IR"
      ),
      (
        "unknownField",
        "lift: model/lifter/testdata/realizationRefusals/Refusals.scala:10: Realization has no invented in the IR"
      )
    )
    for (name, expected) <- cases do
      val out = lifted(name)
      val lift = this.lift(
        s"$jar=${stored("realizationRefusals")}",
        modelClasspath.toString,
        out.toString,
        s"fixture.realizationRefusals.Refusals$$package$$.$name"
      )
      assertNotEquals(lift.exit, 0)
      assertEquals(refused(lift), Seq(expected))
      assert(!Files.exists(out), s"the lifter wrote the $name realization's IR")

  for (name, _) <- fixtures do
    test(s"the $name fixture lifts to its expected IR"):
      expect(s"$name.json", ir(name))

  test("every rejected declaration is refused at its line, and no IR is written"):
    expect("rejects.txt", rejections())

  test("the expected files are the ones the fixtures lift to"):
    assertEquals(leftOver(), Nil)

  concurrently("a declaration's identity does not move with its line"):
    val was = ir("declarations")
    val shifted = Files.createDirectories(scratch.resolve("shifted"))
    Files.writeString(
      shifted.resolve("Declarations.scala"),
      "// A line the declarations move down by.\n" * 3
        + Files.readString(scratch.resolve("lifts/Declarations.scala"))
    )
    Files.copy(scratch.resolve("lifts/project.scala"), shifted.resolve("project.scala"))
    val jar = packaged("shifted", shifted)
    val roots = fixtures.toMap.apply("declarations")
    val arguments =
      Seq(
        s"$jar=${stored("lifts")},$modelJar=model/",
        modelClasspath.toString,
        lifted("shifted").toString
      )
    val lift = this.lift((arguments ++ roots)*)
    assert(!lift.failed, lift.diagnostics)
    val now = Files.readString(lifted("shifted"))
    def lines(ir: String) = ir.replaceAll("\"line\": [0-9]+", "\"line\": _")
    assertNotEquals(now, was, "moving the declarations did not move their lines")
    assert(lines(now) == lines(was), "moving the declarations down changed more than their lines")

  concurrently("a jar's prefix may be an argument of its own"):
    val out = lifted("presence-prefix-argument")
    val lift = this.lift(
      (Seq(liftsJar.toString, modelClasspath.toString, out.toString, stored("lifts"))
        ++ fixtures.toMap.apply("presence"))*
    )
    assert(!lift.failed, lift.diagnostics)
    assertEquals(Files.readString(out), ir("presence"))

  concurrently("a lift without roots, or without its arguments, is refused"):
    val none = lift(s"$modelJar=model/", modelClasspath.toString, lifted("no-roots").toString)
    assertEquals(none.exit, 1)
    assertEquals(refused(none), Seq("lift: no roots: name the declarations to lift"))
    val usage = lift("only-one")
    assertEquals(usage.exit, 2)
    assert(
      usage.output.contains("usage: lift <jar=prefix>,... <classpath file> <out.json> <root>..."),
      usage.output
    )
