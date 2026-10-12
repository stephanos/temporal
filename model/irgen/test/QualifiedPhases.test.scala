package umpire.irgen

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.Tools

class QualifiedPhasesTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val stored = "model/irgen/testdata/qualifiedPhases/"
  private val task = tools.directory.resolve(".flow/tmp/fn159/task3")
  private val modelJar = task.resolve("scoped-model.jar")
  private val scratch = Files.createTempDirectory(task, "phases.")
  private val names = for
    level <- Vector("File", "Machine", "Section", "Rules")
    form <- Vector("Wildcard", "Selective", "Renamed")
    layout <- Vector("Line", "Inline")
  yield level + form + layout

  private def packaged(name: String, sources: Vector[String]) =
    val directory = Files.createDirectory(scratch.resolve(name))
    for source <- "Shared" +: sources do
      Files.copy(
        tools.directory.resolve(stored + source + ".scala"),
        directory.resolve(source + ".scala")
      ): Unit
    Files.writeString(
      directory.resolve("project.scala"),
      s"//> using scala 3.9.0\n//> using jvm 27\n//> using jar $modelJar\n"
    ): Unit
    val jar = directory.resolve("fixture.jar")
    tools
      .scalaCli(
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
    jar

  private lazy val importsJar = packaged("imports", names)
  private lazy val headingsJar = packaged("headings", Vector("Headings"))

  private def lift(jar: java.nio.file.Path, roots: Seq[String]) =
    val output = scratch.resolve(roots.head + ".json")
    val ran = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$stored,$modelJar=model/",
        task.resolve("scoped-model.classpath").toString,
        output.toString
      ) ++ roots.map("fixture.qualifiedphases." + _)
    )
    (ran, output)

  test(
    "phase imports refuse wildcard, selective and renamed forms at every authored level and position"
  ):
    val (ran, output) = lift(importsJar, names)
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(!Files.exists(output))
    for name <- names do
      val source = Files.readAllLines(tools.directory.resolve(stored + name + ".scala")).asScala
      val line = source.indexWhere(_.contains("import Stage.")) + 1
      val diagnostic = ran.diagnostics.linesIterator.find(_.contains(s"$name.scala:$line:"))
      assert(diagnostic.nonEmpty, ran.diagnostics)
      assert(diagnostic.get.contains("phase type Stage"), diagnostic.get)
      assert(diagnostic.get.contains("Stage.<case>"), diagnostic.get)
    assertEquals(ran.diagnostics.linesIterator.count(_.contains("R9.phase-import")), names.size)

  test(
    "qualified phases, eta-expanded sets, roles and Property when accept unrelated enum imports"
  ):
    val (ran, output) =
      lift(headingsJar, Vector("Accepted", "Accepted.properties", "QualifiedSet", "Inherited"))
    assertEquals(ran.exit, 0, ran.diagnostics)
    val ir = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(output))
    assertEquals(ir.path("machines").size(), 3)
    assertEquals(ir.path("properties").size(), 1)

  test("bare phase and bare eta-expanded set independently refuse at their authored lines"):
    for (name, bare, remedy) <- Vector(
        ("BarePhase", "idle", "when(states.idle)"),
        ("BareSet", "working", "when(states.working)")
      )
    do
      val (ran, output) = lift(headingsJar, Vector(name))
      assertEquals(ran.exit, 1, ran.diagnostics)
      assert(!Files.exists(output))
      val source = Files.readAllLines(tools.directory.resolve(stored + "Headings.scala")).asScala
      val line = source.indexWhere(_.contains(s"when($bare)")) + 1
      assert(ran.diagnostics.contains(s"Headings.scala:$line:"), ran.diagnostics)
      assert(ran.diagnostics.contains(remedy), ran.diagnostics)
