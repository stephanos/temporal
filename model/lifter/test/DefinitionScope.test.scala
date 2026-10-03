package umpire.lift

import java.nio.file.{Files, Path}
import scala.concurrent.{blocking, Await, Future}
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.gate.{Ran, Tools}

/**
 * The probe behind fn-112's DefinitionScope: a symbol-based Definition ID is the compiler owner of
 * its `val` followed by the captured name, so one pin of the former owner reproduces every ID of a
 * declaration moved under an object or into another package, with no ID written per declaration.
 *
 * The lifted fixtures that declare every symbol-based kind (actions, monitors, assumptions and holes;
 * channels and the actions they derive; realizations) are moved both ways and lifted again. Each ID of
 * the moved IR must carry the new owner, and replacing that one owner by the former one must give
 * exactly the IDs of testdata/lifts/expected.
 */
class DefinitionScope extends munit.FunSuite:
  override val munitTimeout: Duration = 20.minutes

  private val tools = Tools.here
  private val root = tools.directory
  private val gen = root.resolve("model/gen")
  private val lifts = root.resolve("model/lifter/testdata/lifts")
  private val modelJar = gen.resolve("model-scala.jar")
  private val modelClasspath = gen.resolve("model-scala.classpath")
  private lazy val scratch =
    Files.createTempDirectory(Files.createDirectories(gen.resolve("history")), "scope.")

  // The roots the lifter's fixtures name, as Fixtures lifts them: file, package, roots by val name.
  private val probed = Seq(
    ("declarations", "Declarations", "fixture.declarations", Seq("queries", "durableEventually")),
    ("channels", "Channels", "fixture.channels", Seq("relay", "tallying")),
    (
      "realizations",
      "Realizations",
      "fixture.realizations",
      Seq(
        "learnedRun",
        "pauseRace",
        "pauseRaceQuery",
        "doorRealization",
        "doorOpens",
        "errandRealization",
        "errandRetry",
        "errandWithdrawn",
        "tallyRealization",
        "tallyOpens"
      )
    )
  )
  // A realization fixture root that is not moved.
  private val unmoved = Map(
    "realizations" -> Seq("temporal.nexuscaller.Claims$package$.syncCompletion")
  )
  private val kinds = Seq("actions", "monitors", "assumptions", "holes", "channels", "realizations")

  /** Wraps a file's declarations, after its package clause and imports, in `object Moved`. */
  private def underObject(source: String): String =
    val lines = source.linesIterator.toVector
    // The last line of the header: a package clause, an import, or a line of a multi-line import.
    val header = lines.indices
      .foldLeft((-1, false)): (at, i) =>
        val (last, inImport) = at
        val line = lines(i)
        if line.startsWith("package ") then (i, false)
        else if line.startsWith("import ") then (i, line.contains("{") && !line.contains("}"))
        else if inImport then (i, line.trim != "}")
        else at
      ._1
    (lines.take(header + 1) ++ Vector("", "object Moved:") ++
      lines.drop(header + 1).map(line => if line.isBlank then line else "  " + line))
      .mkString("", "\n", "\n")

  /** Moves a file's declarations into the subpackage `moved`, which still sees its parent's members. */
  private def underPackage(source: String, pkg: String): String =
    source.replaceFirst(
      s"(?m)^package ${java.util.regex.Pattern.quote(pkg)}$$",
      s"package $pkg\npackage moved"
    )

  private def lift(arguments: String*): Ran =
    val java = Path.of(System.getProperty("java.home"), "bin", "java").toString
    tools.run(
      java,
      Seq("-cp", System.getProperty("java.class.path"), "umpire.lift.lift") ++ arguments
    )

  private val mapper = new com.fasterxml.jackson.databind.ObjectMapper()

  private def ids(ir: String): Map[String, Vector[String]] =
    val model = mapper.readTree(ir)
    kinds
      .map(kind =>
        kind -> model.path(kind).elements().asScala.map(_.path("id").asText()).toVector.sorted
      )
      .toMap

  /** Lifts every probed fixture moved by `move`, and its new owner per fixture. */
  private def moved(
      name: String,
      move: (String, String) => String,
      ownerOf: (String, String) => String
  ): Seq[(String, String, String, String)] =
    val dir = Files.createDirectories(scratch.resolve(name))
    val stream = Files.newDirectoryStream(lifts, "*.scala")
    try
      for from <- stream.asScala do
        val file = from.getFileName.toString
        val source = Files
          .readString(from)
          .replace("../../../gen/model-scala.jar", modelJar.toString)
          .replace("../../../gen/api-scalapb.jar", gen.resolve("api-scalapb.jar").toString)
        val probe = probed.find((_, stem, _, _) => file == s"$stem.scala")
        Files.writeString(
          dir.resolve(file),
          probe.fold(source)((_, _, pkg, _) => move(source, pkg))
        )
    finally stream.close()
    val jar = scratch.resolve(s"$name.jar")
    tools
      .scalaCli(Seq("--power", "package", "--library", dir.toString, "-f", "-o", jar.toString))
      .orFail()
    val runs = probed.map: (fixture, stem, pkg, roots) =>
      val owner = ownerOf(stem, pkg)
      val out = scratch.resolve(s"$name-$fixture.json")
      val arguments = Seq(
        s"$jar=model/lifter/testdata/lifts/,$modelJar=model/",
        modelClasspath.toString,
        out.toString
      )
        ++ roots.map(s"$owner." + _) ++ unmoved.getOrElse(fixture, Nil)
      (fixture, stem, pkg, owner, out, Future(blocking(lift(arguments*))))
    runs.map: (fixture, stem, pkg, owner, out, run) =>
      val ran = Await.result(run, munitTimeout)
      assert(!ran.failed, s"the $name $fixture probe did not lift:\n${ran.diagnostics}")
      (fixture, s"$pkg.$stem$$package$$", owner, Files.readString(out))

  private def pinned(probes: Seq[(String, String, String, String)]): Unit =
    for (fixture, former, owner, ir) <- probes do
      val expected = ids(Files.readString(lifts.resolve(s"expected/$fixture.json")))
      val now = ids(ir)
      for kind <- kinds do
        assert(
          !now(kind).exists(_.startsWith(former + ".")),
          s"$fixture $kind kept the owner $former"
        )
        // One pin of the former owner, and the captured names it is followed by.
        val scoped = now(kind).map(id =>
          if id.startsWith(owner + ".") then former + id.drop(owner.length) else id
        )
        assertEquals(
          scoped.sorted,
          expected(kind),
          s"$fixture $kind under the pin $owner -> $former"
        )
      assert(
        kinds.exists(kind => now(kind).exists(_.startsWith(owner + "."))),
        s"$fixture declares nothing under $owner"
      )

  test("a declaration moved under an object keeps its ID under one pin of its former owner"):
    pinned(moved("object", (source, _) => underObject(source), (_, pkg) => s"$pkg.Moved$$"))

  test("a declaration moved into another package keeps its ID under one pin of its former owner"):
    pinned(
      moved("package", underPackage, (stem, pkg) => s"$pkg.moved.$stem$$package$$")
    )

  test("every symbol-based kind is probed"):
    val declared = probed.flatMap((fixture, _, _, _) =>
      val all = ids(Files.readString(lifts.resolve(s"expected/$fixture.json")))
      kinds.filter(all(_).nonEmpty)
    )
    assertEquals(declared.distinct.sorted, kinds.sorted)
