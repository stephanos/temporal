package umpire.irgen

import java.nio.file.{Files, Path}
import scala.concurrent.{blocking, Await, Future}
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.{Ran, Tools}

/**
 * DefinitionScope: a symbol-based Definition ID is the owner of its `val` followed by the captured
 * name, so one `DefinitionScope` pin of the former owner keeps every ID of declarations moved under an
 * object or into another package, with no ID written per declaration.
 *
 * The lifted fixtures that declare every symbol-based kind (actions, monitors, assumptions and holes;
 * channels and the actions they derive; realizations) are moved both ways, given one pin of the owner
 * they left, and lifted again. The functions they declare carry the new owner, so the move took; the
 * IDs must be exactly those of testdata/lifts/expected.
 *
 * A type declared at the top level of a file is its package's, so the pin of the file's former owner
 * `<package>.<File>$package$` also keeps the name it had there: moved into another package, the
 * fixtures' `types` (their names, fields and every reference to a type) are exactly those of
 * testdata/lifts/expected but for positions. A type moved under an object is the object's member and
 * an object's pin leaves type names alone, so there they carry the object and are not the expected
 * ones.
 */
class DefinitionScope extends munit.FunSuite:
  override val munitTimeout: Duration = 20.minutes

  private val tools = Tools.here
  private val root = tools.directory
  private val build = root.resolve("model/build")
  private val lifts = root.resolve("model/irgen/testdata/lifts")
  private val modelJar = build.resolve("model-scala.jar")
  private val modelClasspath = build.resolve("model-scala.classpath")
  private lazy val scratch =
    Files.createTempDirectory(Files.createDirectories(build.resolve("history")), "scope.")

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
        "runOpens",
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
  private val kinds = Seq("actions", "monitors", "assumptions", "holes", "channels", "realizations")

  private def pin(former: String) = s"given DefinitionScope = DefinitionScope(\"$former\")"

  /**
   * Wraps a file's declarations, after its package clause and imports, in `object Moved`, which pins
   * the file's owner.
   */
  private def underObject(source: String, former: String): String =
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
    (lines.take(header + 1) ++ Vector("", "object Moved:", "  " + pin(former), "") ++
      lines.drop(header + 1).map(line => if line.isBlank then line else "  " + line))
      .mkString("", "\n", "\n")

  /**
   * Moves a file's declarations into the subpackage `moved`, which still sees its parent's members,
   * and pins the file's former owner at its top level.
   */
  private def underPackage(source: String, pkg: String, former: String): String =
    source.replaceFirst(
      s"(?m)^package ${java.util.regex.Pattern.quote(pkg)}$$",
      java.util.regex.Matcher.quoteReplacement(s"package $pkg\npackage moved\n\n${pin(former)}\n")
    )

  private def lift(arguments: String*): Ran =
    val java = Path.of(System.getProperty("java.home"), "bin", "java").toString
    tools.run(
      java,
      Seq("-cp", System.getProperty("java.class.path"), "umpire.irgen.lift") ++ arguments
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
      move: (String, String, String) => String,
      ownerOf: (String, String) => String
  ): Seq[(String, String, String, String)] =
    val dir = Files.createDirectories(scratch.resolve(name))
    val stream = Files.newDirectoryStream(lifts, "*.scala")
    try
      for from <- stream.asScala do
        val file = from.getFileName.toString
        val source = Files
          .readString(from)
          .replace("../../../build/model-scala.jar", modelJar.toString)
          .replace("../../../build/api-scalapb.jar", build.resolve("api-scalapb.jar").toString)
        val probe = probed.find((_, stem, _, _) => file == s"$stem.scala")
        Files.writeString(
          dir.resolve(file),
          probe.fold(source)((_, stem, pkg, _) => move(source, pkg, s"$pkg.$stem$$package$$"))
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
        s"$jar=model/irgen/testdata/lifts/,$modelJar=model/",
        modelClasspath.toString,
        out.toString
      )
        ++ roots.map(s"$owner." + _)
      (fixture, stem, pkg, owner, out, Future(blocking(lift(arguments*))))
    runs.map: (fixture, stem, pkg, owner, out, run) =>
      val ran = Await.result(run, munitTimeout)
      assert(!ran.failed, s"the $name $fixture probe did not lift:\n${ran.diagnostics}")
      (fixture, s"$pkg.$stem$$package$$", owner, Files.readString(out))

  // A fixture's types without their positions, which the move shifts.
  private def types(ir: String): String =
    val all = mapper.readTree(ir).path("types").deepCopy[com.fasterxml.jackson.databind.JsonNode]()
    def strip(n: com.fasterxml.jackson.databind.JsonNode): Unit =
      n match
        case o: com.fasterxml.jackson.databind.node.ObjectNode => o.remove("position"): Unit
        case _                                                 => ()
      n.elements().asScala.foreach(strip)
    strip(all)
    all.toPrettyString

  private def pinned(probes: Seq[(String, String, String, String)], typesKept: Boolean): Unit =
    for (fixture, former, owner, ir) <- probes do
      val stored = Files.readString(lifts.resolve(s"expected/$fixture.json"))
      val expected = ids(stored)
      val now = ids(ir)
      if typesKept then
        assertEquals(types(ir), types(stored), s"$fixture types under the pin of $former in $owner")
      else
        assertNotEquals(
          types(ir),
          types(stored),
          s"$fixture types moved under $owner kept their names"
        )
      val functions = mapper.readTree(ir).path("functions").elements().asScala.map(_.path("name"))
      assert(
        functions.exists(_.asText().startsWith(owner + ".")),
        s"$fixture declares no function under $owner, so nothing moved"
      )
      for kind <- kinds do
        assertEquals(
          now(kind),
          expected(kind),
          s"$fixture $kind under the pin of $former in $owner"
        )

  test("a declaration moved under an object keeps its ID under one pin of its former owner"):
    pinned(
      moved(
        "object",
        (source, _, former) => underObject(source, former),
        (_, pkg) => s"$pkg.Moved$$"
      ),
      typesKept = false
    )

  test(
    "a declaration moved into another package keeps its ID and its types their names under one pin"
  ):
    pinned(
      moved("package", underPackage, (stem, pkg) => s"$pkg.moved.$stem$$package$$"),
      typesKept = true
    )

  test("every symbol-based kind is probed"):
    val declared = probed.flatMap((fixture, _, _, _) =>
      val all = ids(Files.readString(lifts.resolve(s"expected/$fixture.json")))
      kinds.filter(all(_).nonEmpty)
    )
    assertEquals(declared.distinct.sorted, kinds.sorted)
