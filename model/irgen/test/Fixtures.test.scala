package umpire.irgen

import java.nio.file.{Files, Path}
import java.util.concurrent.Semaphore
import scala.collection.mutable
import scala.concurrent.{blocking, Await, Future}
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.{Ran, Tools}

/**
 * The lifter's fixtures under testdata: the Models it must lift, compared with the IR in
 * testdata/lifts/expected, and the declarations the build or the lifter must refuse, at their lines.
 *
 * The fixtures build against the framework and the Temporal Models the gate packaged into
 * model/build, so the gate runs these tests after it packaged them. With UMPIRE_LIFTER_UPDATE set,
 * as the gate's --update sets it, the expected files are rewritten instead of compared.
 */
class Fixtures extends munit.FunSuite:
  // A test builds a fixture with scala-cli and lifts it in a JVM of its own.
  override val munitTimeout: Duration = 20.minutes

  private val tools = Tools.here
  private val root = tools.directory
  private val build = root.resolve("model/build")
  private val testdata = root.resolve("model/irgen/testdata")
  private val expected = testdata.resolve("lifts/expected")
  private val modelJar = build.resolve("model-scala.jar")
  private val modelClasspath = build.resolve("model-scala.classpath")
  private val update = sys.env.get("UMPIRE_LIFTER_UPDATE").exists(_.nonEmpty)

  // Each run builds in a directory of its own under the ignored model/build/history, which is kept
  // for inspection: nothing is deleted, so another process's build state is never removed.
  private lazy val scratch =
    Files.createTempDirectory(Files.createDirectories(build.resolve("history")), "lifter.")
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
            .replace("../../../build/model-scala.jar", modelJar.toString)
            .replace("../../../build/api-scalapb.jar", build.resolve("api-scalapb.jar").toString)
        )
    finally stream.close()
    to

  // The lifter's positions of a materialized fixture: its stored files.
  private def stored(fixture: String) = s"model/irgen/testdata/$fixture/"

  /** The lines of the stored files at which the fixture's build fails, as `<file>:<line>:<column>`. */
  private def refusals(fixture: String): Seq[String] =
    val built = bounded(tools.scalaCli(Seq("compile", materialize(fixture).toString)))
    assert(built.failed, s"model/irgen/testdata/$fixture built")
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
        Seq("-cp", System.getProperty("java.class.path"), "umpire.irgen.lift") ++ arguments
      )
    )

  private def refused(ran: Ran): Seq[String] =
    ran.output.linesIterator.filter(_.startsWith("lift:")).toSeq

  private val fixtures: Seq[(String, Seq[String])] = Seq(
    "presence" -> Seq("fixture.presence.Presence"),
    "channels" -> Seq("fixture.channels.Relay", "fixture.channels.Tallying"),
    "declarations" -> Seq(
      "fixture.declarations.Declarations$package$.queries",
      "fixture.declarations.Declarations$package$.durableEventually"
    ),
    "admission" -> Seq(
      "fixture.specimens.admission.Admission$package$.currentQueries",
      "fixture.specimens.admission.Admission$package$.staleQueries"
    ),
    "realizations" -> Seq(
      "fixture.realizations.Realizations$package$.learnedRun",
      "fixture.realizations.Realizations$package$.runOpens",
      "fixture.realizations.Realizations$package$.pauseRace",
      "fixture.realizations.Realizations$package$.pauseRaceQuery",
      "fixture.realizations.Realizations$package$.doorRealization",
      "fixture.realizations.Realizations$package$.doorOpens",
      "fixture.realizations.Realizations$package$.errandRealization",
      "fixture.realizations.Realizations$package$.errandRetry",
      "fixture.realizations.Realizations$package$.errandWithdrawn",
      "fixture.realizations.Realizations$package$.tallyRealization",
      "fixture.realizations.Realizations$package$.tallyOpens"
    ),
    "taskqueue" -> Seq("queueQueries", "matchingQueries", "forgetfulQueries")
      .map("fixture.taskqueue.TaskQueue$package$." + _),
    "capabilities" -> Seq(
      "jobCapabilities",
      "legacyCapabilities",
      "pairCapabilities",
      "keptCapabilities",
      "killedWhileQueued",
      "rogueCapabilities",
      "rightNeverHeld"
    )
      .map("fixture.capabilities.Capabilities$package$." + _),
    "captured" -> Seq(
      "queries",
      "diskQueries",
      "localDiskQueries",
      "Relay",
      "ledger",
      "PutOnly",
      "durableEventually"
    ).map(r =>
      if r.head.isUpper then s"fixture.captured.$r" else s"fixture.captured.Captured$$package$$.$r"
    ),
    // fn-118.2: API behavior hints and server steps (lifts/Hints.scala). The reader admits the
    // first and refuses each realization of the second at its line (tools/umpire/model).
    "hints" -> Seq("keptBehavior", "ownBehavior").map("fixture.hints.Hints$package$." + _),
    // fn-126 decision 23: the objects that group actions, named after where they are declared
    // (lifts/Sections.scala).
    "sections" -> Seq("fixture.sections.Switch", "fixture.sections.Clapper"),
    // fn-126 R15, R16: a machine object, its core twin, derivations and a composition object
    // (lifts/Rules.scala); tools/umpire/model holds the twins' tables equal.
    "rules" -> (Seq(
      "Switch",
      "Mirror",
      "Steady",
      "Loose",
      "Dimming",
      "Twins",
      "Unequal",
      "Dial",
      "Turned"
    )
      .map("fixture.rules." + _) ++ Seq(
      "fixture.rules.CoreSwitch",
      "fixture.rules.Switch$.queries$.pressing",
      "fixture.rules.Switch$.queries$.wornOut"
    )),
    "hintsRefused" -> Seq(
      "zeroInterval",
      "nonPositiveBound",
      "intervalOverBound",
      "unboundedStep",
      "timerNoDeadline",
      "deliveryDeadline"
    ).map("fixture.hints.Hints$package$." + _)
  )

  // fn-122.2: the fixtures whose capability declarations write a law sidecar beside their IR.
  private val sidecars: Seq[String] = Seq("capabilities")
  // The refusals of fn-112.4's typed composition selectors, in lifts/Rejects.scala.
  private val selectorRejects: Seq[String] = Seq(
    "MemberNoField",
    "SyncNoMember",
    "MemberIncompatible",
    "SyncUnbound",
    "WithNoMember",
    "WithIncompatible",
    "WithUnrefined",
    "WithUnsynced",
    "syncedNone",
    "syncedTwice",
    "ownSynced",
    "ownSpelledAlike",
    "mixedSchedule",
    "bareInputs",
    "whenClassInputs",
    "ReplacesSpare",
    "LoopPair",
    "syncedNoMember",
    "ownNoMember",
    "syncedNoField"
  )
  // The refusals of fn-112.4's claim patterns, records over a member and function-valued arguments.
  private val patternRejects: Seq[String] = Seq(
    "whenStays",
    "keptComputed",
    "fromVal",
    "sharedLambda",
    "recordsNoMember",
    "recordsForeignFact",
    "recordsUnfilled",
    "paramCalled"
  )

  // The refusals of fn-112.5's input tokens, inputs supplied by name and bounded counters.
  private val inputRejects: Seq[String] = Seq(
    "foreignToken",
    "suppliedTwice",
    "supplyKept",
    "namedNoTokens",
    "inputTwice",
    "tokenUnnamed",
    "upToNegative"
  )

  // The refusals of fn-112.11's Query totals, which an author may write and the lifter otherwise
  // computes (fn-126 decision 28).
  private val totalRejects: Seq[String] = Seq(
    "totaledTwice",
    "totalComputed",
    "totalKept",
    "totalNegative",
    "sharedComputed"
  )

  // The refusals of fn-120.1's named choices.
  private val choiceRejects: Seq[String] = Seq(
    "ChosenTwice",
    "SpelledTwice",
    "ChoiceHelper",
    "ChoiceDisabled",
    "ChoiceTwoSteps",
    "ChoiceIf",
    "ChoiceUnnamed",
    "ChoiceKept",
    "ChoiceKeptHelper"
  )

  // The refusals of fn-120.2's unnamed branching: several results written without a choose.
  private val unnamedRejects: Seq[String] = Seq("UnnamedList", "UnnamedJoin", "UnnamedInHelper")

  // The refusals of fn-120.5's levels: a step made where no step function is, one per declaration
  // kind whose Scala type admits one (model/SEMANTICS.md, Levels).
  private val levelRejects: Seq[String] = Seq(
    "LevelStart",
    "LevelEnds",
    "LevelEvidence",
    "LevelRefinement",
    "LevelMonitor",
    "LevelRequire",
    "levelProperty",
    "levelTransition",
    "levelPattern",
    "levelProgress",
    "LevelComposition",
    "levelScenario"
  )

  // The refusals of fn-112.10's names taken by default.
  private val defaultRejects: Seq[String] =
    Seq(
      "unnamedTwice",
      "SyncNamedTwice",
      "omittedEmpty",
      "watchUnnamed",
      "watchUnwatched",
      "watchElsewhere"
    )

  // The refusals of fn-112.12's claim bundles.
  private val bundleRejects: Seq[String] = Seq("mixedBundle")

  // The refusals of fn-112.9's script helpers, status tables and request scopes
  // (lifts/ScriptRejects.scala).
  private val scriptRejects: Seq[String] = Seq(
    "unnamedCommand",
    "performsNothing",
    "onNoPath",
    "unlistedStatus",
    "statusTwice",
    "notAField",
    "notAPolledField",
    "foreignFact"
  ).map("fixture.scriptrejects.ScriptRejects$package$." + _)

  // The refusals of fn-122.2's capability declarations, fn-122.5's citations and fn-127.2's
  // `through` (lifts/CapabilityRejects.scala).
  private val capabilityRejects: Seq[String] = Seq(
    "unboundAction",
    "lambdaField",
    "declaredTwice",
    "againFirst",
    "againSecond",
    "notBrought",
    "noReason",
    "otherSignature",
    "waivedClaim",
    "unkinded",
    "sameName",
    "lambdaQuery",
    "uncited",
    "computedCitation",
    "unknownParameter",
    "throughComputed",
    "throughLambda",
    "overridingThrough"
  ).map("fixture.capabilityrejects.CapabilityRejects$package$." + _)

  // The refusals of fn-118.2's API behavior hints (lifts/HintRejects.scala).
  private val hintRejects: Seq[String] =
    Seq("builtWrite", "builtRead").map("fixture.hintrejects.HintRejects$package$." + _)

  // fn-126 decision 20: what a machine is for, as its markers say (lifts/MarkerRejects.scala).
  private val markerRejects: Seq[String] =
    Seq(
      "FaultUnmarked",
      "FaultlessFailure",
      "HopelessFailure$.queries$.brokenHopelessFailure",
      "UnrefutedControl$.queries$.foundUnrefutedControl",
      "ControlRefiner",
      "RefinedControl$.queries$.askedRefinedControl",
      "SystemicControl$.queries$.askedSystemicControl",
      "TornMarkers$.queries$.askedTornMarkers",
      "UnmarkedPair",
      "FaultlessPair",
      "DerivedControl$.queries$.foundDerivedControl"
    ).map("fixture.markers." + _)

  // A root of lifts/Rejects.scala: a machine or composition object, or an object nested in one of its
  // objects, by its object (`Hoarding`, `Anonymous$.anonymous`); a top-level val by its name.
  private def rejectsRoot(name: String): String =
    if name.head.isUpper then s"fixture.rejects.$name"
    else s"fixture.rejects.Rejects$$package$$.$name"

  private val rejected = Seq(
    "Hoarding",
    "closed",
    "Waiting",
    "Doubled",
    "Listening",
    "Counter",
    "crossedRead",
    "negative",
    "Watched",
    "Unrefined",
    "Misplaced",
    "noSuchRoot",
    "UnrefinedOutcomes",
    "Counting",
    "Batching",
    "Shuffling",
    "Guessing",
    "unnamedProperty",
    "unnamedScenario",
    "Twins",
    "askedTwice",
    "boundTwice",
    "Tapped",
    "Undeclared",
    "Unlisted",
    "Partial",
    "unrelatedRead",
    "AssumedTwice",
    "GapTwice",
    "RebindPush",
    "ExtendBound",
    "ReboundTwice",
    "AssumedAgain",
    "AssumingTwice",
    "RefinedNothing",
    "RefinedOtherwise",
    "LoopFirst",
    "Aliased",
    "Splatted",
    "Explained"
  ).map(rejectsRoot) ++ (selectorRejects ++ patternRejects ++ inputRejects ++ totalRejects ++
    choiceRejects ++ unnamedRejects ++ bundleRejects ++ defaultRejects ++ levelRejects).map(
    rejectsRoot
  ) ++ Seq(
    // A name the compiler made up and a computed ok outcome, refused in objects of their own.
    "Anonymous$.anonymous",
    "ComputedOk$.ComputedOk"
  ).map(rejectsRoot) ++ Seq(
    // fn-126 R15, R16 and decision 27: machine objects, rules and effects.
    "EffectOutside",
    "EmptyEffect",
    "Unheaded",
    "BlockTwice",
    "BlockRepeated",
    "PhaseLambda",
    "DisabledFired",
    "ExtendedBare",
    "RebindSeveral",
    "RebindUnbound",
    "Endless",
    "EndedTwice",
    "LooseRefinement",
    "LookalikePair",
    "NilEffect",
    "WatchesElsewhere",
    "RebindOneClass",
    "valMachine",
    "valComposition",
    "UnobservedRefiner",
    "IndexedQueries$.queries"
  ).map(rejectsRoot) ++ scriptRejects ++ capabilityRejects ++ hintRejects ++
    markerRejects

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

  /** The law sidecar a fixture's capability declarations wrote beside its IR. */
  private def laws(name: String): String =
    ir(name): Unit
    Files.readString(scratch.resolve(s"$name.laws.json"))

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
    held.diff(fixtures.map(_._1 + ".json") ++ sidecars.map(_ + ".laws.json") :+ "rejects.txt")

  // An update rewrites the expected files only once every fixture lifted, every rejected
  // declaration was refused and no expected file is left over, so the tree it leaves is one whole
  // run's. The gate holds model/ir to the same rule.
  private lazy val everyLift: Unit =
    fixtures.foreach((name, _) => ir(name))
    sidecars.foreach(laws)
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
        "Capabilities.scala:19:85",
        "Capabilities.scala:22:16",
        "Capabilities.scala:25:17",
        "Capabilities.scala:30:87",
        "Crossed.scala:38:41",
        "Crossed.scala:41:24",
        "Crossed.scala:49:82",
        "IrFile.scala:8:37",
        "NamedInput.scala:21:50",
        "NamedInput.scala:24:27",
        "NamedInput.scala:27:49",
        "NamedInput.scala:30:31",
        "NoCatalog.scala:7:98",
        "OneChoice.scala:18:62",
        "Rules.scala:16:30",
        "Sugar.scala:10:27",
        "Sugar.scala:13:86",
        "Sugar.scala:16:71",
        "Sugar.scala:19:68",
        "Sugar.scala:23:69"
      )
    )

  // fn-126 R15: a machine object declares its init, its end and its rules, and a derived machine
  // has none of its own.
  concurrently("the build refuses a machine object without init, end or rules, at its line"):
    assertEquals(
      refusals("objectForms").sorted,
      Seq(
        "Invalid.scala:31:8", // no init
        "Invalid.scala:37:8", // no end
        "Invalid.scala:43:8", // no rules
        "Invalid.scala:49:3" // a derived machine's own rules
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
        "Invalid.scala:169:26",
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

  concurrently("a realization's reference by value to a missing declaration fails to compile"):
    assertEquals(
      refusals("referenceInvalid").sorted,
      Seq(
        "Invalid.scala:36:32",
        "Invalid.scala:38:44",
        "Invalid.scala:40:38",
        "Invalid.scala:42:41",
        "Invalid.scala:44:63",
        "Invalid.scala:46:38",
        "Invalid.scala:48:28"
      )
    )

  // fn-114 R3: each string-named or string-keyed form the DSL retired, written as it was.
  concurrently("the retired string-named and string-keyed declaration forms do not compile"):
    assertEquals(
      refusals("retiredNames").sorted,
      Seq(
        "Invalid.scala:31:27", // timer("expire")
        "Invalid.scala:34:29", // internal("flush")
        "Invalid.scala:37:20", // hole("crash")
        "Invalid.scala:40:33", // channel[Note]("wire", ...)
        "Invalid.scala:40:41",
        "Invalid.scala:43:38", // Lamp.restrict("pressOnly")(...)
        "Invalid.scala:50:28", // Composition[PairState]("pair")
        "Invalid.scala:51:49", // Composition[PairState]("left" -> ..., ...)
        "Invalid.scala:51:65",
        "Invalid.scala:55:23", // sync("pressBoth", "left" -> ..., ...)
        "Invalid.scala:55:40",
        "Invalid.scala:56:14", // replaces("left", Lamp)
        "Invalid.scala:59:20" // Lamp.scenario.actionKeys("press")
      ).sorted
    )

  concurrently("only the unknown projected origin admits a dynamic message root"):
    assertEquals(refusals("dynamicInvalid"), Seq("Invalid.scala:10:16"))

  // fn-118.2: a hint of a method the generated API lacks, or whose write, read, `when` or bound has
  // the wrong type, a hint built by its constructor, and a server step of no class.
  concurrently("API behavior hints refuse unknown methods and mistyped arguments, at their lines"):
    assertEquals(
      refusals("hintsInvalid").sorted,
      Seq(
        "Invalid.scala:22:20", // WorkflowServiceGrpc.METHOD_DESCRIBE_NOTHING
        "Invalid.scala:23:21", // "StartActivityExecution".visibleTo(...)
        "Invalid.scala:24:36", // start.visibleTo(DescribeActivityExecutionRequest, ...)
        "Invalid.scala:27:66", // visibleTo(describe, WaitBound(100, 1000))
        "Invalid.scala:28:19", // Visibility(...)
        "Invalid.scala:31:48", // boundedBy(Visible.atOnce)
        "Invalid.scala:34:30" // ServerStep(CauseKind.delivery, ...)
      ).sorted
    )

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

  // The typed selectors no live Model writes (lifts/Typed.scala); model/ir pins the rest.
  concurrently("typed schemas, paths and bound constants lift to their protobuf names"):
    val out = lifted("typed")
    val roots = Seq("fixture.typed.Typed", "fixture.typed.Typed$package$.typedRealization")
    val result = lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
    assert(!result.failed, result.diagnostics)
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val model = mapper.readTree(Files.readString(out))
    val actions = model.path("actions")
    assertEquals(actions.size(), 1)
    assertEquals(
      actions.get(0).path("schemas").get(0).asText(),
      "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    )
    val realizations = model.path("realizations")
    assertEquals(realizations.size(), 1)
    val typed = realizations.get(0)
    val call = typed.at("/scripts/0/items/0/command/rpc")
    assertEquals(call.at("/assign/0/target").asText(), "heartbeat_timeout.seconds")
    assertEquals(call.at("/assign/0/value/literal/number").asText(), "5")
    assertEquals(call.at("/reads/0/path").asText(), "run_id")
    assertEquals(
      typed.at("/scripts/0/items/1/command/rpc/reads/0/path").asText(),
      "history.events[*].event_id"
    )
    // Through `Option.map` the optional message is selected, not indexed as repeated.
    assertEquals(typed.at("/evidence/0/fields/0/path").asText(), "delivery_admission.delivery_id")

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

  // fn-112.10: a monitor of a Query's expected Run named by value, as by its name. Lifted apart
  // from the realizations fixture: no realization runs the watched door.
  concurrently("a monitor expectation names its monitor by value as by its name"):
    val out = lifted("monitorExpectations")
    val roots =
      Seq("heldByValue", "heldByName").map("fixture.realizations.Realizations$package$." + _)
    val result = lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
    assert(!result.failed, result.diagnostics)
    val queries = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out))
      .path("queries")
    def expected(name: String) = queries
      .elements()
      .asScala
      .find(_.path("name").asText() == name)
      .getOrElse(fail(s"the realizations fixture lifted no Query $name"))
      .path("expectedRun")
    val byValue = expected("heldByValue")
    assertEquals(
      byValue.path("monitors").elements().asScala.map(_.path("name").asText()).toList,
      List("opensOnce", "staysOpen")
    )
    assertEquals(byValue.toPrettyString, expected("heldByName").toPrettyString)

  // fn-114.1: the IR files the sources declare with `irFile`, lifted by `lift --ir` in one run.
  private def liftIr(out: Path, jars: String, names: String*): Ran =
    lift((Seq("--ir", jars, modelClasspath.toString, out.toString) ++ names)*)

  private def listed(directory: Path): Seq[String] =
    if !Files.isDirectory(directory) then Nil
    else
      val stream = Files.list(directory)
      try stream.iterator.asScala.map(_.getFileName.toString).toList.sorted
      finally stream.close()

  concurrently("one run lifts each IR file apart, and a root of two files into both"):
    val out = scratch.resolve("irFiles")
    val result = liftIr(out, liftsJars, "shared-admission", "shared-presence")
    assert(!result.failed, result.diagnostics)
    assertEquals(listed(out), Seq("shared-admission.json", "shared-presence.json"))
    // Lifted after the wider file, with state of its own: the presence fixture's IR, byte for byte.
    assertEquals(Files.readString(out.resolve("shared-presence.json")), ir("presence"))
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    def machines(json: String) =
      mapper.readTree(json).path("machines").elements().asScala.map(_.path("name").asText()).toSet
    val shared = Files.readString(out.resolve("shared-admission.json"))
    assertEquals(machines(shared), machines(ir("presence")) ++ machines(ir("admission")))
    assertEquals(
      mapper.readTree(shared).path("source").asText(),
      "model: fixture.presence.Presence, " +
        "fixture.specimens.admission.Admission$package$.currentQueries, " +
        "fixture.specimens.admission.Admission$package$.staleQueries"
    )

  concurrently("a refusal names the IR file it was lifting, and the run writes no file"):
    val out = scratch.resolve("irFilesRefused")
    val result = liftIr(out, liftsJars, "refused", "shared-presence", "nowhere")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil, "the lifter wrote an IR file of a run that failed")
    val rejected = Files.readString(expected.resolve("rejects.txt")).linesIterator.toSet
    refused(result) match
      case Seq(nowhere, header, refusal) =>
        assertEquals(
          nowhere,
          "lift: IR file nowhere: no irFile of the lifted sources declares it"
        )
        assertEquals(header, "lift: the roots of refused.json did not lift:")
        assert(rejected(refusal), s"$refusal is not the rejected declaration's refusal")
      case other => fail(s"one refusal under refused.json and an unknown file, not $other")

  concurrently("the lifter refuses an IR file declaration it cannot read, at its line"):
    val jar = packaged("irFileRefusals", materialize("irFileRefusals"))
    val out = scratch.resolve("irFileRefusals-out")
    val at = stored("irFileRefusals") + "IrFiles.scala"
    val result = liftIr(out, s"$jar=${stored("irFileRefusals")},$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assertEquals(
      refused(result),
      Seq(
        s"lift: $at:14: an IR file is named by a nonempty string literal without a `/`",
        s"lift: $at:17: an IR file is named by a nonempty string literal without a `/`",
        s"lift: $at:22: splatted names its roots one by one, each a val that declares one",
        s"lift: $at:9: twice is declared twice, at $at:6 and here: an IR file is declared once"
      )
    )

  // fn-126 decision 23: an ID derived from a name is unique in its package across the run, even
  // when one shared def declares both records at one position.
  concurrently(
    "the lifter refuses one shared def's Query of one name over two machines of a package"
  ):
    val jar = packaged("derivedTwins", materialize("derivedTwins"))
    val out = scratch.resolve("derivedTwins-out")
    val at = stored("derivedTwins") + "DerivedTwins.scala"
    val result = liftIr(out, s"$jar=${stored("derivedTwins")},$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assertEquals(
      refused(result),
      Seq(
        s"lift: $at:35: this declaration over plainLamp and the one over lamp derive the ID " +
          "fixture.derivedtwins.query.anyLit: name them apart, since an ID derived from a name is " +
          "unique in its package"
      )
    )

  // fn-126 R4: the declaration-order lint, before anything is lifted. Each kind at its line, and
  // none of the reads beside them that initialize nothing yet (initOrder/Forward.scala). Every other
  // lift of these tests, and the gate's of the Models, runs it too and is refused nothing.
  concurrently(
    "the declaration-order lint refuses a read before its declaration, a cycle, an order and a place"
  ):
    val jar = packaged("initOrder", materialize("initOrder"))
    val out = scratch.resolve("initOrder-out")
    val at = stored("initOrder")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val forward = (line: Int) =>
      s"lift: ${at}Forward.scala:$line: late is read while Forward initializes, before it is " +
        s"declared at ${at}Forward.scala:18, so it is still null here: declare it before the " +
        "declaration that reads it"
    val feature = s"${at}InitOrder.scala"
    assertEquals(
      refused(result),
      Seq(
        // (a)
        forward(7),
        forward(17),
        // (d): a Model declaration beside the feature file
        s"lift: ${at}Forward.scala:23: stray is a Scenario, which a feature declares in its " +
          s"feature file, $feature: in the `queries` object of its machine's object there",
        s"lift: ${at}Forward.scala:29: strayInCompanion is a Scenario, which a feature declares " +
          s"in its feature file, $feature: in the `queries` object of its machine's object there",
        // (d): a machine in a type's companion, a Property at the top level, a machine object in
        // an object of the signature
        s"lift: $feature:22: BulbLamp holds a Model declaration inside Bulb, the companion of a " +
          "type: a machine object sits at the top level of a feature file, and its sections " +
          "directly in it",
        s"lift: $feature:28: loose is a Property, declared at the top level of a feature file: " +
          "it belongs in the `properties` object of its machine's object",
        s"lift: $feature:32: Inner holds a Model declaration inside Holder, an object of the " +
          "signature: a machine object sits at the top level of a feature file, and its " +
          "sections directly in it",
        // (b)
        s"lift: $feature:53: an initialization cycle: Switch.implements -> SwitchRealization -> " +
          "Switch.implements, each read while the one before it initializes, so one of them is " +
          "read half made: read it in a def, a lambda or a lazy val, or move what is read into " +
          "an object of its own",
        // (c)
        s"lift: $feature:68: properties belongs before queries at $feature:65: object Backwards " +
          "reads its header, then states, then refinement, then effects, then monitors, then " +
          "rules, then properties, then implements, then queries",
        s"lift: $feature:71: Late belongs before Backwards at $feature:59: a feature file reads " +
          "its header, then its types, then its signature, then its machine and composition " +
          "objects, then object exports",
        // (d)
        s"lift: $feature:77: extras holds a Model declaration in Misplaced, and is none of its " +
          "sections, states, refinement, effects, monitors, rules, properties, implements, " +
          "queries: its declarations belong in them",
        s"lift: $feature:82: lit is a Property, and belongs in the `properties` object of its " +
          "machine's object, not in Misplaced",
        s"lift: $feature:85: switchLit is declared over Switch, a machine object: it belongs in " +
          "Switch.properties",
        // (c): a Scenario after a Query; (d): a Query over another object's Scenario
        s"lift: $feature:99: late belongs before first at $feature:98: Asked.queries reads " +
          "its Scenarios, then its Queries",
        s"lift: $feature:100: borrowed is declared over flipped, which Switch.queries declares: " +
          "it belongs in Switch.queries",
        // (d): a val in exports that is no IR file
        s"lift: $feature:104: note is declared in exports, which holds the feature's IR files alone"
      )
    )

  // fn-126 R4 (e) and R17: each kind at its line, and none of the correct object forms beside them,
  // nor two objects' timers of one name, which their objects name apart (sectionOrder/SectionOrder.scala).
  concurrently("the declaration-order lint refuses sections out of order or out of place"):
    val jar = packaged("sectionOrder", materialize("sectionOrder"))
    val out = scratch.resolve("sectionOrder-out")
    val at = stored("sectionOrder")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val f = s"${at}SectionOrder.scala"
    val order = "object Backwards reads its header, then states, then refinement, then effects, " +
      "then monitors, then rules, then properties, then implements, then queries"
    val handBound = "a machine object says when each action fires in its `rules`, " +
      "`on(action) { in(p) ~> effects.x }`, and a derivation binds one in `rebind`"
    assertEquals(
      refused(result),
      Seq(
        s"lift: $f:39: the section implements sits at the top level of a feature file: a " +
          "machine's sections sit directly in its machine or composition object",
        s"lift: $f:101: states belongs before effects at $f:98: $order",
        s"lift: $f:107: rules belongs before properties at $f:104: $order",
        s"lift: $f:114: stray is a step function, and belongs in the `effects` object of its " +
          "machine's object, not in Misfiled",
        s"lift: $f:115: watched is a monitor, assumption, hole or channel, and belongs in the " +
          "`monitors` object of its machine's object, not in Misfiled",
        s"lift: $f:116: lit is vocabulary of Misfiled, declared outside its sections: it belongs " +
          "in the `states` object of its machine's object",
        s"lift: $f:117: toProduct is a member of Misfiled's refinement: declare it in " +
          "`object refinement extends Refinement(product)`, which holds the machine's refinement",
        s"lift: $f:126: dim is a monitor, assumption, hole or channel, declared in " +
          "Misfiled.properties: it belongs in the `monitors` object of its machine's object",
        s"lift: $f:127: borrowed is declared over Switch, a machine object: it belongs in " +
          "Switch.properties",
        s"lift: $f:134: a step function is bound by hand, `action ~> step`, in HandBound: $handBound",
        s"lift: $f:138: kept is a step function, declared inside HandBound.effects.more: it " +
          "belongs in the `effects` object of its machine's object",
        s"lift: $f:152: armed is read while Guarded.rules initializes, before it is declared at " +
          s"$f:153, so it is still null here: declare it before the declaration that reads it",
        s"lift: $f:158: a step function is bound by hand, `action ~> step`, in Escaped: $handBound",
        s"lift: $f:165: a step function is bound by hand, `action ~> step`, in Cored: $handBound"
      )
    )

  // A Model folder must name its feature file after itself (misnamed/Lamp.scala): a file of another
  // name in a folder with no feature file would otherwise be held to no reading order.
  concurrently("the declaration-order lint refuses a Model in a folder with no feature file"):
    val jar = packaged("misnamed", materialize("misnamed"))
    val out = scratch.resolve("misnamed-out")
    val at = stored("misnamed")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val home = "a Model folder declares its Models in its feature file, the file named after " +
      s"the folder, in $at"
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}Lamp.scala:16: Switch holds a Model declaration in a file not named after " +
          s"its folder: $home",
        s"lift: ${at}Lamp.scala:27: loose is a Property, declared in a file not named after its " +
          s"folder: $home",
        s"lift: ${at}Lamp.scala:32: Spare holds a Model declaration in a file not named after " +
          s"its folder: $home"
      )
    )

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

  // Captured.scala takes every name from its val or object, its starts and evidence by default and
  // reads a refinement with no given; expected/captured.json pins its IR. Every symbol-based
  // Definition ID is its declaration's fully qualified name, the objects it sits in included, every
  // top-level type is named in its package and every family is that package (fn-126 decision 23).
  test("the captured fixture's IDs, type names and families are where each is declared"):
    val model = new com.fasterxml.jackson.databind.ObjectMapper().readTree(ir("captured"))
    val pkg = "fixture.captured."
    for
      kind <- Seq("actions", "monitors", "assumptions", "holes", "channels", "realizations")
      declared = model.path(kind)
    do
      assert(declared.size() > 0, s"Captured.scala declares no $kind")
      for d <- declared.elements().asScala do
        val id = d.path("id").asText()
        assert(id.startsWith(pkg) && !id.contains("$"), s"$id is not a qualified name in $pkg")
    // A monitor at the top level, and one in `object Watched`.
    assertEquals(
      model.path("monitors").elements().asScala.map(_.path("id").asText()).toList.sorted,
      List("Watched.storedTwice", "storedOnce").map(pkg + _)
    )
    // Its actions sit in an actor object and in plain objects, which each name; the actor object
    // names the actor.
    assertEquals(
      model
        .path("actions")
        .elements()
        .asScala
        .map(a => a.path("id").asText() -> a.path("actor").asText())
        .toList
        .sorted,
      List(
        "background.expire" -> "system",
        "background.flush" -> "system",
        "client.put" -> "client",
        "faults.crash" -> "fault",
        "relaying.send" -> "client",
        "wire.deliver" -> "system",
        "wire.lose" -> "system"
      ).map((name, actor) => (pkg + name) -> actor).sorted
    )
    val types = model.path("types").elements().asScala.map(_.path("name").asText()).toList
    assert(types.nonEmpty, "Captured.scala declares no types")
    for name <- types do assert(name.startsWith(pkg), s"$name is not named in $pkg")
    for kind <- Seq("machines", "compositions"); m <- model.path(kind).elements().asScala do
      assertEquals(m.path("family").asText(), "fixture.captured", m.path("name").asText())

  // Sections.scala: a top-level action is named in its package, one in a top-level object by the
  // object too, and one in an actor directly in a machine's object by both objects; two objects tell
  // their actions of one name apart (fn-126 decision 23).
  test("an action is named after the package and the objects it is declared in"):
    val model = new com.fasterxml.jackson.databind.ObjectMapper().readTree(ir("sections"))
    val actions = model
      .path("actions")
      .elements()
      .asScala
      .map(a => a.path("id").asText() -> a.path("actor").asText())
      .toMap
    assertEquals(
      actions,
      Map(
        "fixture.sections.reset" -> "system",
        "fixture.sections.panel.flip" -> "panel",
        "fixture.sections.Switch.operator.press" -> "operator",
        "fixture.sections.leftHand.clap" -> "fixture",
        "fixture.sections.rightHand.clap" -> "fixture"
      )
    )

  // A machine object's members name its state type through the machine, `State`, as the type it
  // stands for: `def broken(s: State)` lifts as `def brokenLamp(s: Lamp)` does (lifts/Rules.scala).
  test("a state type named through its machine, State, lifts as the type it stands for"):
    import com.fasterxml.jackson.databind.node.ObjectNode
    val model = new com.fasterxml.jackson.databind.ObjectMapper().readTree(ir("rules"))
    def function(name: String) =
      val f = model
        .path("functions")
        .elements()
        .asScala
        .find(_.path("name").asText() == s"fixture.rules.Switch$$.states$$.$name")
        .getOrElse(fail(s"no function $name"))
        .deepCopy[ObjectNode]()
      f.remove(java.util.List.of("name", "position"))
      f.findParents("position").asScala.foreach {
        case o: ObjectNode => o.remove("position"): Unit
        case _             => ()
      }
      f.toPrettyString
    assertEquals(function("broken"), function("brokenLamp"))

  /** The root of the machine object `fixture.<fixture>.<Name>` a machine `name` is declared by. */
  private def objectRoot(fixture: String)(name: String): String =
    s"fixture.$fixture.${name.capitalize}"

  /**
   * The machines and Properties of one lift of `roots` of the lifts fixture `fixture`, by name, each
   * as a pair compares it: without its name, its machine and positions, and with each function it
   * refers to in place of the function's name, so two spellings may name their functions apart. A
   * root of the Temporal Models, and a machine object of the fixture, is named in full.
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
    val named =
      roots.map(r =>
        if r.startsWith("temporal.") || r.startsWith("fixture.") then r else s"$pkg.$r"
      )
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
    val (model, machine, _) =
      declarations("derived", pairs.flatMap((a, b) => Seq(a, b)).map(objectRoot("derived")))
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
    "enter, stay, disabled, because, in, implies, records and sticky lift as their core forms do"
  ):
    val (model, machine, property) =
      declarations(
        "sugar",
        Seq("fixture.sugar.Sugared", "fixture.sugar.Cored", "claims", "fixture.sugar.Watched")
      )
    assertEquals(machine("sugared"), machine("cored"))
    for form <- Seq("records", "implies", "paused") do
      assertEquals(property(s"${form}Sugar"), property(s"${form}Core"), form)
    // Each sticky monitor beside its `monitor` spelling: one monitor but for its ID, name, position
    // and the names of its functions, whose bodies are compared in their place.
    def monitor(name: String): String =
      import com.fasterxml.jackson.databind.node.ObjectNode
      val m = model
        .path("monitors")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"the sugar fixture lifted no monitor named $name"))
        .deepCopy[ObjectNode]()
      m.remove(java.util.List.of("id", "name", "position"))
      m.path("initial") match
        case i: ObjectNode => i.remove("position"): Unit
        case _             => ()
      for field <- Seq("next", "violated") do
        val f = model
          .path("functions")
          .elements()
          .asScala
          .find(_.path("name").asText() == m.path(field).asText())
          .get
          .deepCopy[ObjectNode]()
        f.remove(java.util.List.of("name", "position"))
        f.findParents("position").asScala.foreach {
          case o: ObjectNode => o.remove("position"): Unit
          case _             => ()
        }
        m.set(field, f): Unit
      m.toPrettyString
    for (sugar, spelled) <- Seq(
        "refusedOnce" -> "refusedOnceSpelled",
        "retriedLost" -> "retriedLostSpelled"
      )
    do assertEquals(monitor(sugar), monitor(spelled), sugar)
    assert(monitor("refusedOnce").contains("neverRefused"), monitor("refusedOnce"))
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

  // fn-112.9: the script helpers and the Temporal kit beside the core records they stand for, and
  // the kit's `field(_.name) :=` beside `Assignment.typed` (lifts/Scripts.scala).
  concurrently(
    "script helpers, the Temporal kit and field := lift as the core records they stand for"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val pkg = "fixture.scripts.Scripts$package$."
    val out = lifted("scripts-pairs")
    val roots = Seq("helpers", "records", "sugaredRequest", "coredRequest").map(pkg + _)
    val result = lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
    assert(!result.failed, result.diagnostics)
    val model = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(out))
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode => o.remove("position"): Unit
        case _             => ()
      n.elements().asScala.foreach(strip)
    def realization(name: String): JsonNode =
      val r = model
        .path("realizations")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"the scripts fixture lifted no realization $name"))
        .deepCopy[ObjectNode]()
      r.remove(java.util.List.of("id", "name"))
      strip(r)
      r
    assertEquals(realization("helpers").toPrettyString, realization("records").toPrettyString)
    def request(name: String) =
      realization(name).at("/scripts/0/items/0/command/rpc").toPrettyString
    assert(request("sugaredRequest").contains("task_queue.name"), request("sugaredRequest"))
    assertEquals(request("sugaredRequest"), request("coredRequest"))

  // fn-112.5: input tokens, inputs supplied by name and a bounded counter (lifts/Inputs.scala).
  concurrently("inputs supplied by name lift as their positional calls, and UpTo as the Int range"):
    import com.fasterxml.jackson.databind.JsonNode
    val (model, _, property) = declarations(
      "inputs",
      Seq(
        "byNameQuery",
        "byPositionQuery",
        "urgentByNameQuery",
        "urgentByPositionQuery",
        "omittedQuery",
        "omittedByPositionQuery"
      )
    )
    def named(kind: String, name: String): JsonNode = model
      .path(kind)
      .elements()
      .asScala
      .find(_.path("name").asText() == name)
      .getOrElse(fail(s"the inputs fixture lifted no $kind named $name"))
    def actionsOf(scenario: String): String =
      named("scenarios", scenario).path("actions").toPrettyString
    // A call by name is its positional twin, class by class: partial, reordered and defaulted.
    assertEquals(actionsOf("byName"), actionsOf("byPosition"))
    // fn-112.10: a call that omits every input is its positional twin at every first value.
    assertEquals(actionsOf("omitted"), actionsOf("omittedByPosition"))
    assertEquals(property("urgentByName"), property("urgentByPosition"))
    val respond = named("actions", "respond")
    assertEquals(
      respond.path("inputs").elements().asScala.map(_.path("name").asText()).toList,
      List("answer", "urgent"),
      "an input takes its token's val name"
    )
    // The control action reports results by name with no enum declared of that name.
    assertEquals(named("actions", "steer").path("results").asText(), "Delivery")
    assert(
      !model.path("types").elements().asScala.exists(_.path("name").asText().endsWith(".Delivery")),
      "results named a type"
    )
    // UpTo[2] lifts as the Int range 0..2, the record the live protocol state's attempt counter
    // lifts to in model/ir/activity.json, which the gate holds.
    assertEquals(
      named("types", "fixture.inputs.CountedState").at("/record/fields/1/type").toString,
      """{"intRange":{"high":"2"}}"""
    )

  // fn-112.11: the total each Query asserts (lifts/Totals.scala).
  concurrently(
    "a Query's total lifts infix, dotted, around expect and from a shared def's argument"
  ):
    import com.fasterxml.jackson.databind.node.ObjectNode
    val (model, _, _) = declarations("totals", Seq("queries", "lampTotals", "plainLampTotals"))
    def record(name: String): ObjectNode =
      val q = model
        .path("queries")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"the totals fixture lifted no Query named $name"))
        .deepCopy[ObjectNode]()
      q.remove(java.util.List.of("name", "position"))
      q
    def total(name: String): String = record(name).path("total").asText()
    def without(name: String): String =
      val q = record(name)
      q.remove("total")
      q.toPrettyString
    for name <- Seq("infixTotal", "dottedTotal", "totalThenExpect", "expectThenTotal") do
      assertEquals(total(name), "4", name)
    assertEquals(record("dottedTotal").toPrettyString, record("infixTotal").toPrettyString)
    assertEquals(record("expectThenTotal").toPrettyString, record("totalThenExpect").toPrettyString)
    val unexpected = record("totalThenExpect")
    assert(unexpected.has("expectedRun"), "expect left no expected run")
    unexpected.remove("expectedRun")
    assertEquals(unexpected.toPrettyString, record("infixTotal").toPrettyString)
    // Each instance of the shared def carries the total its call supplied, and nothing else apart.
    assertEquals(total("lamp.anyLit"), "12")
    assertEquals(total("plainLamp.anyLit"), "4")
    assertEquals(without("plainLamp.anyLit").replace("plainLamp", "lamp"), without("lamp.anyLit"))
    // A Query that asserts no total takes the count the lifter computes, the one an author writes.
    assertEquals(record("countedTotal").toPrettyString, record("infixTotal").toPrettyString)
    val (counted, _, _) = declarations("totals", Seq("lampCounted"))
    val (plainCounted, _, _) = declarations("totals", Seq("plainLampCounted"))
    def countedTotal(m: com.fasterxml.jackson.databind.JsonNode, name: String) =
      m.path("queries").elements().asScala.find(_.path("name").asText() == name).get.path("total")
    assertEquals(countedTotal(counted, "lamp.anyLit").asText(), "12")
    assertEquals(countedTotal(plainCounted, "plainLamp.anyLit").asText(), "4")

  // fn-112.12: claims one shared def declares together as a case-class bundle, read back by field
  // (lifts/Totals.scala), lift to the IR of the same claims declared directly on each machine.
  concurrently("a bundle of claims built by its constructor and read by field lifts as its claims"):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    def lifting(spelling: String): JsonNode =
      val out = lifted(s"bundle-$spelling")
      val roots = Seq("Lamp", "PlainLamp").map(m => s"fixture.totals.Totals$$package$$.$spelling$m")
      val result = lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
      assert(!result.failed, result.diagnostics)
      mapper.readTree(Files.readString(out).replace(spelling, "direct"))
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode => o.remove(java.util.List.of("position", "source")): Unit
        case _             => ()
      n.elements().asScala.foreach(strip)
    val (bundled, direct) = (lifting("bundled"), lifting("direct"))
    val names = bundled
      .path("properties")
      .elements()
      .asScala
      .map(p => s"${p.path("machine").asText()}.${p.path("name").asText()}")
    assertEquals(
      names.toList.sorted,
      List("lamp.directLit", "lamp.directUnlit", "plainLamp.directLit", "plainLamp.directUnlit")
    )
    strip(bundled)
    strip(direct)
    assertEquals(bundled.toPrettyString, direct.toPrettyString)

  // fn-120.1, fn-120.2: named choices (lifts/Choices.scala). Each name is on the step its alternative
  // wrote, in the order written; an alternative that calls a function calls a copy of it, named
  // `<function>$<choice>`, whose every step carries the name, through the functions it calls in
  // turn, while the function itself keeps its unnamed steps for the other calls.
  concurrently("choose names each alternative's step, and a copy of the function it calls"):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val (model, _, _) = declarations("choices", Seq("fixture.choices.Chosen"))
    val prefix = "fixture.choices.Choices$package$."
    def function(name: String): JsonNode = model
      .path("functions")
      .elements()
      .asScala
      .find(_.path("name").asText() == prefix + name)
      .getOrElse(fail(s"the choices fixture lifted no function $name"))
    def all(n: JsonNode, keep: JsonNode => Boolean): List[JsonNode] =
      (if keep(n) then List(n) else Nil) ++ n.elements().asScala.toList.flatMap(all(_, keep))
    def steps(f: JsonNode) = all(f, _.at("/construct/type").asText() == "umpire.Step")
    def choices(f: JsonNode) = steps(f).map(_.at("/construct/choice").asText())
    def calls(f: JsonNode) = all(f, _.has("call")).map(_.at("/call/function").asText())
    val written = Seq(
      "admitNamed" -> List("committed", "redelivered"),
      "pauseNamed" -> List("held", "dropped", "refused", "committed"),
      "pollNamed" -> List("committed", "held", "held", "dropped"),
      "answerNamed" -> List("committed", "refused"),
      "retryNamed" -> List("held"),
      "admitted$committed" -> List("committed", "committed"),
      "admitted$redelivered" -> List("redelivered", "redelivered"),
      "admitted" -> List("", ""),
      "resumeStep" -> Nil
    )
    for (name, expected) <- written do assertEquals(choices(function(name)), expected, name)
    // The alternatives in the order written: a copy's steps, the step written out, a copy's steps.
    val retry = function("retryNamed").path("body")
    assertEquals(retry.at("/binary/op").asText(), "OP_CONCAT")
    assertEquals(
      calls(retry),
      List("admitted$committed", "redeliveredStep$redelivered").map(prefix + _)
    )
    assertEquals(retry.at("/binary/left/binary/right/list/items").size(), 1)
    assertEquals(
      calls(function("redeliveredStep$redelivered")),
      List(prefix + "admitted$redelivered")
    )
    assertEquals(calls(function("resumeStep")), List(prefix + "admitted"))
    // A copy is its function but for the names: without its name, its choices and the suffix of the
    // copies it calls, it is the function, parameters, precondition and every branch alike.
    def unnamed(f: JsonNode, choice: String): JsonNode =
      val copy = f.deepCopy[JsonNode]()
      def strip(n: JsonNode): Unit =
        n match
          case o: ObjectNode =>
            o.remove(java.util.List.of("choice", "name")): Unit
            if o.has("function") then
              o.put("function", o.path("function").asText().stripSuffix("$" + choice)): Unit
          case _ => ()
        n.elements().asScala.foreach(strip)
      strip(copy)
      copy
    for (name, choice) <- Seq(
        "admitted" -> "committed",
        "admitted" -> "redelivered",
        "redeliveredStep" -> "redelivered"
      )
    do
      assertEquals(
        unnamed(function(s"$name$$$choice"), choice).toPrettyString,
        unnamed(function(name), choice).toPrettyString,
        s"$name$$$choice"
      )

  // fn-112.4, fn-112.7: typed composition selectors (lifts/Members.scala).
  // The production compositions of temporal/features/standaloneactivity, written with typed
  // selectors and derived by `withMember`, keep their exact members, syncs, replacement targets and
  // Scenario keys; and the keys of names that carry separators, over two members that bind actions
  // spelled alike.
  concurrently("typed members, syncs, replaces, withMember, synced and own keep the composed keys"):
    import com.fasterxml.jackson.databind.JsonNode
    // Each design is an object, named as the composition is with the first letter raised: its
    // Queries in its `queries`, or the design itself where no Query runs over it.
    val designs = "temporal.features.standaloneactivity.system."
    def designObject(d: String) = designs + d.head.toUpper + d.tail
    val overQueue = Seq("currentOverQueue", "staleOverQueue")
    val overMatching = Seq("currentOverMatching", "staleOverMatching", "currentOverLossyMatching")
    val unqueried = Seq("currentOverForgetful", "currentOverVolatile")
    val roots =
      (overQueue ++ overMatching).map(d => s"${designObject(d)}$$.queries$$.${d}Queries") ++
        unqueried.map(designObject) ++ Seq("switchQueries", "flickedBothOnce")
    val (model, _, _) = declarations("members", roots)
    def all(kind: String) = model.path(kind).elements().asScala.toList
    def named(name: String)(n: JsonNode) = n.path("name").asText() == name
    def texts(n: JsonNode, field: String) =
      n.elements().asScala.map(_.path(field).asText()).toList
    // Each derived design replaces the interface its own queue refines, not its base's.
    val replaced = Map(
      "currentOverQueue" -> "",
      "staleOverQueue" -> "",
      "currentOverMatching" -> "dispatchQueue",
      "staleOverMatching" -> "dispatchQueue",
      "currentOverForgetful" -> "dispatchQueue",
      "currentOverVolatile" -> "dispatchQueue",
      "currentOverLossyMatching" -> "dispatchQueueUnderStorageLoss"
    )
    for (d, target) <- replaced do
      val c = all("compositions").find(named(d)).getOrElse(fail(s"no composition $d"))
      assertEquals(texts(c.path("members"), "field"), List("activity", "queue"), d)
      assertEquals(texts(c.path("syncs"), "name"), List("dispatch", "admit", "settle"), d)
      assertEquals(c.path("members").get(1).path("replaces").asText(), target, d)
    def scenarioKeys(machine: String, name: String) = all("scenarios")
      .find(s => s.path("machine").asText() == machine && named(name)(s))
      .getOrElse(fail(s"no Scenario $name of $machine"))
      .path("keys")
      .elements()
      .asScala
      .map(_.asText())
      .toList
    val pause = "activity_control-pause"
    val invoked = List("dispatch", "queue_addActivityTask")
    for d <- overQueue do
      assertEquals(scenarioKeys(d, "staleDeliveryAfterPause"), List("dispatch", pause, "admit"))
      assertEquals(scenarioKeys(d, "admittedBeforePause"), List("dispatch", "admit", pause))
      assertEquals(scenarioKeys(d, "duplicateDelivery"), List("dispatch", "admit", "admit"))
    for d <- overMatching do
      val persisted = invoked :+ "queue_persistTask"
      assertEquals(scenarioKeys(d, "staleDeliveryAfterPause"), persisted ++ List(pause, "admit"))
      assertEquals(scenarioKeys(d, "admittedBeforePause"), persisted ++ List("admit", pause))
      assertEquals(
        scenarioKeys(d, "deliveredAgainAfterLostAck"),
        persisted ++ List("admit", "queue_ackLoss", "admit")
      )
      val matched = invoked :+ "queue_syncMatch"
      assertEquals(
        scenarioKeys(d, "crashAfterAdmissionCommit"),
        matched ++ List("admit", "queue_crash", "queue_addActivityTask", "queue_syncMatch", "admit")
      )
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
    // fn-112.10: a sync named after its first member's action, kept by `withMember` and selected
    // by `synced` from either member.
    for d <- Seq("flicks", "leftFlicks") do
      val c = all("compositions").find(named(d)).getOrElse(fail(s"no composition $d"))
      assertEquals(texts(c.path("syncs"), "name"), List("flick"), d)
    assertEquals(keys("bothFlick"), List("flick"))
    assertEquals(whenAction("flickedBoth"), "flick")
    def bound(machine: String) = all("machines")
      .find(named(machine))
      .get
      .path("steps")
      .elements()
      .asScala
      .map(_.path("action").asText().stripPrefix("fixture.members."))
      .toList
    assertEquals(bound("leftSwitch"), List("Left.tap", "Left.flick", "turnOn"))
    assertEquals(bound("rightSwitch"), List("Right.tap", "Right.flick"))

  // fn-112.12: an independent consumer of the shared task queue (lifts/TaskQueue.scala).
  test("a consumer of temporal/shared/taskqueue lifts the queue and nothing of the activity"):
    import com.fasterxml.jackson.databind.JsonNode
    val model = new com.fasterxml.jackson.databind.ObjectMapper().readTree(ir("taskqueue"))
    def all(kind: String) = model.path(kind).elements().asScala.toList
    def named(kind: String, name: String) =
      all(kind).find(_.path("name").asText() == name).getOrElse(fail(s"no $kind $name"))
    def names(n: JsonNode, field: String) = n.elements().asScala.map(_.path(field).asText()).toList
    // The queue's declarations are named where they are declared, temporal/shared/taskqueue.
    val queue = "temporal.shared.taskqueue."
    assertEquals(named("machines", "dispatchQueue").path("entity").asText(), "taskQueue")
    assertEquals(named("machines", "dispatchQueue").path("family").asText(), queue + "product")
    assertEquals(named("actions", "enqueue").path("id").asText(), queue + "queue.enqueue")
    for (c, queue, target) <- Seq(
        ("jobOverQueue", "dispatchQueue", ""),
        ("jobOverMatching", "matchingQueue", "dispatchQueue"),
        ("jobOverForgetful", "forgetfulQueue", "dispatchQueue")
      )
    do
      val members = named("compositions", c).path("members")
      assertEquals(names(members, "machine"), List("job", queue), c)
      assertEquals(members.get(1).path("replaces").asText(), target, c)
    // No name of the activity's package is lifted, only the shared queue's.
    val fromActivity = model.toString.linesIterator
      .flatMap(
        """"(temporal\.features\.standaloneactivity[^"]*)"""".r.findAllMatchIn(_).map(_.group(1))
      )
      .toSet
    assert(fromActivity.isEmpty, fromActivity.mkString(", "))
    assert(!all("machines").exists(_.path("entity").asText() == "activity"))

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
    do
      assertEquals(property("job", name), property("job", s"${name}Core"), name)
      assertEquals(found("job", name).path("transition").asBoolean, transition, name)
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
        "lift: model/irgen/testdata/samestate/SameState.scala:22: wrongPair pairs property, a Property of first, with scenario, a Scenario of second, and reads it through no refinement"
      )
    )

  concurrently("the lifter refuses a construct outside the subset, at its line"):
    val jar = packaged("unsupported", materialize("unsupported"))
    val lift = this.lift(
      s"$jar=${stored("unsupported")}",
      modelClasspath.toString,
      "/dev/null",
      "temporal.fixture.Unsupported"
    )
    assertNotEquals(lift.exit, 0)
    val line =
      "lift: model/irgen/testdata/unsupported/Unsupported.scala:18: `var out` has no IR form"
    refused(lift) match
      case Seq(refusal) => assert(refusal.startsWith(line), refusal)
      case refusals     => fail(s"one refusal, not $refusals")

  concurrently("the realization emitter refuses unknown constructors and fields at their lines"):
    val jar = packaged("realizationRefusals", materialize("realizationRefusals"))
    val cases = Seq(
      (
        "unknownConstructor",
        "lift: model/irgen/testdata/realizationRefusals/Refusals.scala:7: Unknown is no activation of Script in the IR"
      ),
      (
        "unknownField",
        "lift: model/irgen/testdata/realizationRefusals/Refusals.scala:10: Realization has no invented in the IR"
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

  for name <- sidecars do
    test(s"the $name fixture writes its expected law sidecar"):
      expect(s"$name.laws.json", laws(name))

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

  // ### fn-126 R20: the structure lint, over fixtures of features with their level folders

  /** A fixture whose features have folders, copied with them, its jar's path resolved. */
  private def materializeTree(fixture: String): Path =
    val from = testdata.resolve(fixture)
    val to = Files.createDirectories(scratch.resolve(fixture))
    val jar = java.util.regex.Matcher.quoteReplacement(modelJar.toString)
    val stream = Files.walk(from)
    try
      for source <- stream.iterator.asScala.toList if source.toString.endsWith(".scala") do
        val copy = to.resolve(from.relativize(source).toString)
        Files.createDirectories(copy.getParent)
        Files.writeString(
          copy,
          Files.readString(source).replaceAll("""(\.\./)+build/model-scala\.jar""", jar)
        )
    finally stream.close()
    to

  // The template a new feature copies (layout/lamp): a feature with its two levels and a zoom-in,
  // which the lint refuses nothing of and the lifter lifts.
  concurrently("the layout template lifts with no refusal"):
    val jar = packaged("layout", materializeTree("layout"))
    val out = scratch.resolve("layout-out")
    val result = liftIr(out, s"$jar=${stored("layout")},$modelJar=model/", "lamp")
    assertEquals(refused(result), Nil)
    assert(!result.failed, result.diagnostics)
    assertEquals(listed(out), Seq("lamp.json"))
    val lifted = Files.readString(out.resolve("lamp.json"))
    for machine <- Seq("lampProduct", "lampSystem", "bulb") do
      assert(lifted.contains(s"\"$machine\""), s"lamp.json lifts no machine $machine")

  // R20 (a): the folders of a feature with two levels, a missing root feature file or level file,
  // a feature of one level with a subfolder, and a package that does not mirror its folder
  // (layoutRefusals/a).
  concurrently("the structure lint refuses a feature's levels out of their folders"):
    val jar = packaged("layoutRefusals-a", materializeTree("layoutRefusals/a"))
    val out = scratch.resolve("layoutRefusals-a-out")
    val at = stored("layoutRefusals/a")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val two = "KettleSystem refines KettleProduct, so kettle has two levels, each in its folder: " +
      "the Product in product/Product.scala, the System in system/System.scala, beside the root " +
      "feature file named after the feature's folder, which holds the types, the signature and " +
      "object exports (model/irgen/testdata/layout/lamp is the template)"
    val urn = two.replace(
      "KettleSystem refines KettleProduct, so kettle",
      "UrnSystem refines UrnProduct, so urn"
    )
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}kettle/Kettle.scala:28: Stray is a machine object in kettle's root folder, " +
          s"${at}kettle/, whose feature file holds the types, the signature and object exports " +
          "alone: a feature with two levels declares its machines in product/ and system/",
        s"lift: ${at}kettle/system/Heater.scala:12: $two; ${at}kettle/system/System.scala is " +
          "missing",
        s"lift: ${at}kettle/system/element/Element.scala:8: Element is a machine object in " +
          s"${at}kettle/system/element/, which is no level folder of kettle: a feature with two " +
          "levels keeps its Models in product/ and system/, one file per subject beside the " +
          "level's own file, with no folder below them",
        s"lift: ${at}tap/fittings/Washer.scala:5: ${at}tap/fittings/Washer.scala declares package " +
          s"fixture.features.tap, which its folder, ${at}tap/fittings/, does not mirror: a " +
          "feature's subpackages are its folders, named alike, since the structure lint reads a " +
          "source's package and the order lint its path",
        s"lift: ${at}tap/product/Product.scala:6: tap has no machine that refines another of its " +
          "own, so it has one level, whose Models sit in its feature file: it has no product/ " +
          "folder",
        s"lift: ${at}tap/valve/Valve.scala:8: tap has no machine that refines another of its " +
          "own, so it has one level, whose Models sit in its feature file: it has no valve/ folder",
        s"lift: ${at}urn/system/System.scala:11: $urn; ${at}urn/ has no root feature file",
        s"lift: ${at}urn/system/System.scala:11: $urn; ${at}urn/product/Product.scala is missing"
      )
    )

  // R20 (c): a machine's sections by their closed names, which say what each holds, and
  // one object exports per feature, in its root feature file (layoutRefusals/c); a level folder's
  // file is held to a feature file's order, which has no exports there.
  concurrently("the structure lint refuses an unnamed section and a misplaced exports"):
    val jar = packaged("layoutRefusals-c", materializeTree("layoutRefusals/c"))
    val out = scratch.resolve("layoutRefusals-c-out")
    val at = stored("layoutRefusals/c")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val product = s"${at}kiln/product/Product.scala"
    val none = "and none of its sections, states, refinement, effects, monitors, rules, syncs, " +
      "properties, implements, queries: a section of another name sits at the top level of the " +
      "file, in the signature, and anything else in one of these"
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}kiln/Kiln.scala:11: kiln declares no object exports in its root feature " +
          s"file, ${at}kiln/Kiln.scala: a feature under features names its IR files there, in " +
          "one object exports",
        s"lift: $product:17: timers is an object in KilnProduct $none",
        s"lift: $product:21: helpers is an object in KilnProduct $none",
        s"lift: ${at}pump/system/System.scala:21: Gauge belongs before PumpSystem at " +
          s"${at}pump/system/System.scala:7: a level folder's file reads its header, then its " +
          "types, then its signature, then its machine and composition objects",
        s"lift: ${at}pump/system/System.scala:24: object exports sits in " +
          s"${at}pump/system/System.scala, not in pump's root feature file: a feature names its " +
          "IR files in one object exports, there"
      )
    )
