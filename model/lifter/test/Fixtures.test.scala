package umpire.lift

import java.nio.file.{Files, Path}
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
          Files.readString(from).replace("../../../gen/model-scala.jar", modelJar.toString)
        )
    finally stream.close()
    to

  // The lifter's positions of a materialized fixture: its stored files.
  private def stored(fixture: String) = s"model/lifter/testdata/$fixture/"

  /** The lines of the stored files at which the fixture's build fails, as `<file>:<line>:<column>`. */
  private def refusals(fixture: String): Seq[String] =
    val built = tools.scalaCli(Seq("compile", materialize(fixture).toString))
    assert(built.failed, s"model/lifter/testdata/$fixture built")
    val at = """\[error\] \S*/([^/\s:]+\.scala):(\d+):(\d+)""".r
    built.errors.collect { case at(file, line, column) =>
      assert(Files.isRegularFile(testdata.resolve(fixture).resolve(file)), built.output)
      s"$file:$line:$column"
    }

  private def packaged(name: String, sources: Path): Path =
    val jar = scratch.resolve(s"$name.jar")
    tools
      .scalaCli(Seq("--power", "package", "--library", sources.toString, "-f", "-o", jar.toString))
      .orFail()
    jar

  // The lifter is run as `lift` runs it, in a JVM of its own, since it ends the JVM on a refusal:
  // this build's classes, and the arguments of its command line.
  private def lift(arguments: String*): Ran =
    val java = Path.of(System.getProperty("java.home"), "bin", "java").toString
    tools.run(
      java,
      Seq("-cp", System.getProperty("java.class.path"), "umpire.lift.lift") ++ arguments
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
    "guessing"
  ).map("fixture.rejects.Rejects$package$." + _)

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

  test("the build refuses a warning -Werror makes an error, at its line"):
    assertEquals(refusals("werror"), Seq("Evidence.scala:29:16"))

  test("the build refuses crossed types, at their lines"):
    assertEquals(refusals("crossed"), Seq("Crossed.scala:35:14", "Crossed.scala:45:28"))

  test("the lifter refuses a construct outside the subset, at its line"):
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

  for (name, _) <- fixtures do
    test(s"the $name fixture lifts to its expected IR"):
      expect(s"$name.json", ir(name))

  test("every rejected declaration is refused at its line, and no IR is written"):
    expect("rejects.txt", rejections())

  test("the expected files are the ones the fixtures lift to"):
    assertEquals(leftOver(), Nil)

  test("a declaration's identity does not move with its line"):
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

  test("a jar's prefix may be an argument of its own"):
    val out = lifted("presence-prefix-argument")
    val lift = this.lift(
      (Seq(liftsJar.toString, modelClasspath.toString, out.toString, stored("lifts"))
        ++ fixtures.toMap.apply("presence"))*
    )
    assert(!lift.failed, lift.diagnostics)
    assertEquals(Files.readString(out), ir("presence"))

  test("a lift without roots, or without its arguments, is refused"):
    val none = lift(s"$modelJar=model/", modelClasspath.toString, lifted("no-roots").toString)
    assertEquals(none.exit, 1)
    assertEquals(refused(none), Seq("lift: no roots: name the declarations to lift"))
    val usage = lift("only-one")
    assertEquals(usage.exit, 2)
    assert(
      usage.output.contains("usage: lift <jar=prefix>,... <classpath file> <out.json> <root>..."),
      usage.output
    )
