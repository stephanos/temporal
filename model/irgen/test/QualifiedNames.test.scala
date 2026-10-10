package umpire.irgen

import java.nio.file.{Files, Path}
import scala.concurrent.{blocking, Await, Future}
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.{Ran, Tools}

// A symbol-based Definition ID, a type's IR name and a machine's family are where the declaration
// is declared (fn-126 decision 23): its fully qualified Scala name, the package and every object it
// sits in, and for a family the package alone. No declaration names its own.
//
// The lifted fixtures that declare every symbol-based kind (actions, monitors, assumptions and holes;
// channels and the actions they derive; realizations) are moved three ways and lifted again: into
// another file of their package, which changes none of these names; under an object, which adds the
// object to the name of what moved; and into a subpackage, which adds the package to every one of
// them. The functions they declare carry the new owner, so the move took.
class QualifiedNames extends munit.FunSuite:
  override val munitTimeout: Duration = 20.minutes

  private val tools = Tools.here
  private val root = tools.directory
  private val build = root.resolve("model/build")
  private val lifts = root.resolve("model/irgen/testdata/lifts")
  private val modelJar = build.resolve("model-scala.jar")
  private val modelClasspath = build.resolve("model-scala.classpath")
  private lazy val scratch =
    Files.createTempDirectory(Files.createDirectories(build.resolve("history")), "names.")

  // The roots the lifter's fixtures name, as Fixtures lifts them: file, package, roots by val name,
  // or by object name for a machine object (capitalized).
  private val probed = Seq(
    ("declarations", "Declarations", "fixture.declarations", Seq("queries", "durableEventually")),
    ("channels", "Channels", "fixture.channels", Seq("Relay", "Tallying")),
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

  // Wraps a file's declarations, after its package clause and imports, in `object Moved`. A machine
  // or composition object stays at the top level, where its sections must sit (the declaration-order
  // lint), and so do the objects of the entities it reads while it initializes, which the header
  // imports; the rest is imported back for it.
  private def underObject(source: String): String =
    val lines = source.linesIterator.toVector
    // The last line of the header: a package clause, an import, or a line of a multi-line import.
    val header = lines.indices
      .foldLeft((-1, false)): (at, i) =>
        val (_, inImport) = at
        val line = lines(i)
        if line.startsWith("package ") then (i, false)
        else if line.startsWith("import ") then (i, line.contains("{") && !line.contains("}"))
        else if inImport then (i, line.trim != "}")
        else at
      ._1
    // The top-level statements after the header, each its first line and the indented ones after.
    val statements = lines
      .drop(header + 1)
      .foldLeft(Vector.empty[Vector[String]]): (done, line) =>
        if line.isBlank || line.head.isWhitespace || done.isEmpty then
          done.dropRight(1) :+ (done.lastOption.getOrElse(Vector.empty) :+ line)
        else done :+ Vector(line)
    val kept =
      """object \w+(Entity|Entities)\b.*|object \w+\s+extends (Machine|Derived|Composition)\b.*""".r
    def stays(s: Vector[String]): Boolean =
      kept.matches(s.take(2).map(_.trim).mkString(" ").replaceAll(":.*", ""))
    val (top, moved) = statements.partition(stays)
    // The givens moved, such as a state's catalog, are imported where one is.
    val givens = moved.exists(_.headOption.exists(_.startsWith("given ")))
    val imported = if givens then "import Moved.{given, *}" else "import Moved.*"
    (lines.take(header + 1) ++ Vector(imported, "", "object Moved:") ++
      moved.flatten.map(line => if line.isBlank then line else "  " + line) ++ Vector("") ++
      top.flatten).mkString("", "\n", "\n")

  // Moves a file's declarations into the subpackage `moved`, which still sees its parent's members.
  private def underPackage(source: String, pkg: String): String =
    source.replaceFirst(
      s"(?m)^package ${java.util.regex.Pattern.quote(pkg)}$$",
      java.util.regex.Matcher.quoteReplacement(s"package $pkg\npackage moved\n")
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

  // Lifts every probed fixture moved by `move`, written to the file `file` names, with the roots of
  // each in the package and owner `ownerOf` names; gives each fixture's name, package and IR.
  private def moved(
      name: String,
      move: (String, String) => String,
      file: String => String,
      ownerOf: (String, String) => (String, String)
  ): Seq[(String, String, String)] =
    val dir = Files.createDirectories(scratch.resolve(name))
    val stream = Files.newDirectoryStream(lifts, "*.scala")
    try
      for from <- stream.asScala do
        val written = from.getFileName.toString
        val source = Files
          .readString(from)
          .replace("../../../build/model-scala.jar", modelJar.toString)
          .replace("../../../build/api-scalapb.jar", build.resolve("api-scalapb.jar").toString)
        val probe = probed.find((_, stem, _, _) => written == s"$stem.scala")
        Files.writeString(
          dir.resolve(probe.fold(written)((_, stem, _, _) => file(stem))),
          probe.fold(source)((_, _, pkg, _) => move(source, pkg))
        )
    finally stream.close()
    val jar = scratch.resolve(s"$name.jar")
    tools
      .scalaCli(Seq("--power", "package", "--library", dir.toString, "-f", "-o", jar.toString))
      .orFail(): Unit
    val runs = probed.map: (fixture, stem, pkg, roots) =>
      // A val's root is its owner's member, its file's package object or an object; an object's
      // is its package's.
      val (objects, vals) = ownerOf(pkg, file(stem).stripSuffix(".scala"))
      val out = scratch.resolve(s"$name-$fixture.json")
      val arguments = Seq(
        s"$jar=model/irgen/testdata/lifts/,$modelJar=model/",
        modelClasspath.toString,
        out.toString
      )
        ++ roots.map(r => if r.head.isUpper then s"$objects.$r" else s"$vals.$r")
      (fixture, pkg, out, Future(blocking(lift(arguments*))))
    runs.map: (fixture, pkg, out, run) =>
      val ran = Await.result(run, munitTimeout)
      assert(!ran.failed, s"the $name $fixture probe did not lift:\n${ran.diagnostics}")
      (fixture, pkg, Files.readString(out))

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

  private def families(ir: String): Vector[String] =
    val model = mapper.readTree(ir)
    Seq("machines", "compositions")
      .flatMap(k => model.path(k).elements().asScala.map(_.path("family").asText()))
      .toVector
      .distinct
      .sorted

  private def functions(ir: String): Iterator[String] =
    mapper.readTree(ir).path("functions").elements().asScala.map(_.path("name").asText())

  private def stored(fixture: String) = Files.readString(lifts.resolve(s"expected/$fixture.json"))

  test("a declaration moved into another file of its package keeps its ID, type and family names"):
    for (fixture, pkg, ir) <- moved(
        "file",
        (source, _) => source,
        stem => s"Moved$stem.scala",
        (pkg, stem) => (pkg, s"$pkg.$stem$$package$$")
      )
    do
      val file = probed.collectFirst { case (`fixture`, stem, _, _) => s"Moved$stem" }.get
      assertEquals(ids(ir), ids(stored(fixture)), s"$fixture IDs moved to $file.scala")
      assertEquals(types(ir), types(stored(fixture)), s"$fixture types moved to $file.scala")
      assertEquals(families(ir), families(stored(fixture)), s"$fixture families")
      assert(
        functions(ir).exists(_.startsWith(s"$pkg.$file$$package$$.")),
        s"$fixture declares no function in $file.scala, so nothing moved"
      )

  test("a declaration moved under an object takes the object into its ID"):
    for (fixture, pkg, ir) <- moved(
        "object",
        (source, _) => underObject(source),
        stem => s"$stem.scala",
        (pkg, _) => (pkg, s"$pkg.Moved$$")
      )
    do
      val before = ids(stored(fixture))
      val now = ids(ir)
      def moved(id: String) =
        id.replaceFirst(java.util.regex.Pattern.quote(pkg + "."), pkg + ".Moved.")
      for kind <- kinds do
        val lifted = now(kind).toSet
        assertEquals(now(kind).size, before(kind).size, s"$fixture $kind")
        for was <- before(kind) do
          assert(lifted(was) || lifted(moved(was)), s"$fixture $kind $was lifted as neither")
      assert(
        kinds.exists(k => before(k).exists(w => now(k).contains(moved(w)))),
        s"$fixture moved no declaration under Moved"
      )

  test("a declaration moved into another package takes the package into its ID, type and family"):
    for (fixture, pkg, ir) <- moved(
        "package",
        underPackage,
        stem => s"$stem.scala",
        (pkg, stem) => (s"$pkg.moved", s"$pkg.moved.$stem$$package$$")
      )
    do
      def into(name: String) = name.replace(pkg + ".", pkg + ".moved.")
      val before = ids(stored(fixture))
      for kind <- kinds do
        assertEquals(ids(ir)(kind), before(kind).map(into).sorted, s"$fixture $kind")
      assertEquals(types(ir), into(types(stored(fixture))), s"$fixture types")
      assertEquals(families(ir), families(stored(fixture)).map(f => into(f + ".").stripSuffix(".")))

  test("every symbol-based kind is probed"):
    val declared = probed.flatMap((fixture, _, _, _) =>
      val all = ids(stored(fixture))
      kinds.filter(all(_).nonEmpty)
    )
    assertEquals(declared.distinct.sorted, kinds.sorted)
