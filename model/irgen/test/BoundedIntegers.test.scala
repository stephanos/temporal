package umpire.irgen

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.Tools

class BoundedIntegersTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch =
    Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "bounded-integers.")
  private val json = new com.fasterxml.jackson.databind.ObjectMapper()
  // The packaged Models (ModelBuild): the framework this fixture reads, with the shared code and
  // the mini, which its lifts admit only where its roots reach them.
  private def modelJar = ModelBuild.jar
  private def classpath = ModelBuild.classpath
  private val accepted =
    Files.readString(tools.directory.resolve("model/irgen/testdata/boundedIntegers/Bounds.scala"))

  private def lift(name: String, source: String, root: String) =
    val directory = Files.createDirectory(scratch.resolve(name))
    Files.writeString(directory.resolve("Bounds.scala"), source): Unit
    Files.writeString(
      directory.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = directory.resolve("fixture.jar")
    tools
      .scalaCli(
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
    val output = directory.resolve("fixture.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$directory/,$modelJar=model/",
        classpath.toString,
        output.toString,
        s"fixture.boundedintegers.$root"
      )
    )
    (result, output)

  private def record(ir: com.fasterxml.jackson.databind.JsonNode, name: String) =
    ir.path("types")
      .elements()
      .asScala
      .find(_.path("name").asText() == s"fixture.boundedintegers.$name")
      .get
      .path("record")
      .path("fields")
      .elements()
      .asScala
      .toVector

  test("each literal, singleton, zero and named opaque field retains its own range"):
    val (result, output) = lift("fields", accepted, "Fields")
    assertEquals(result.exit, 0, result.diagnostics)
    val fields = record(json.readTree(Files.readString(output)), "Counters")
    assertEquals(
      fields.map(_.path("type").path("intRange").path("high").asInt()),
      Vector(0, 1, 3, 2)
    )

  test("differently bounded optional fields retain distinct ranges and enum identities"):
    val (result, output) = lift("options", accepted, "Options")
    assertEquals(result.exit, 0, result.diagnostics)
    val ir = json.readTree(Files.readString(output))
    val names = record(ir, "Optional").map(_.path("type").path("named").asText())
    assertEquals(names.distinct.size, 2)
    val highs = names.map(n =>
      ir.path("types")
        .elements()
        .asScala
        .find(_.path("name").asText() == n)
        .get
        .path("enum")
        .path("cases")
        .get(1)
        .path("fields")
        .get(0)
        .path("type")
        .path("intRange")
        .path("high")
        .asInt()
    )
    assertEquals(highs, Vector(1, 3))

  test("differently bounded inboxes associate with their declared channel and message range"):
    val (result, output) = lift("messages", accepted, "Messages")
    assertEquals(result.exit, 0, result.diagnostics)
    val ir = json.readTree(Files.readString(output))
    assertEquals(
      record(ir, "Mail").map(_.path("type").path("channel").asText()),
      Vector("fixture.boundedintegers.small", "fixture.boundedintegers.large")
    )
    assertEquals(
      ir.path("channels")
        .elements()
        .asScala
        .map(_.path("message").path("intRange").path("high").asInt())
        .toSet,
      Set(1, 3)
    )

  test("plain Int fields refuse at the field even with a local finite range"):
    val source = accepted
      .replace("zero: UpTo[0]", "zero: Int")
      .replace(
        "object Fields extends",
        "given Finite[Counters] =\n  given Finite[Int] = Finite.upTo(0)\n  Finite.derived\n\nobject Fields extends"
      )
      .replace("Counters(UpTo(0),", "Counters(0,")
    val (result, output) = lift("int-field", source, "Fields")
    assertEquals(result.exit, 1, result.diagnostics)
    assert(
      result.diagnostics.contains("Counters.zero") && result.diagnostics.contains("UpTo[N]"),
      result.diagnostics
    )
    val fieldLine = source.linesIterator.toVector.indexWhere(_.contains("zero: Int")) + 1
    assert(result.diagnostics.contains(s"Bounds.scala:$fieldLine:"), result.diagnostics)
    assert(!Files.exists(output))

  test("negative and nonconstant bounds refuse without an initialization bypass"):
    for (name, replacement, extra, diagnostic) <- List(
        ("negative", "UpTo[-1]", "", "UpTo[-1] has no values"),
        (
          "nonconstant",
          "UpTo[Int]",
          "given ValueOf[Int] = new ValueOf(2)\n",
          "UpTo bounds a counter by a constant"
        )
      )
    do
      val source = accepted
        .replace("UpTo[0]", replacement)
        .replace("object Fields extends", extra + "object Fields extends")
      val (result, output) = lift(name, source, "Fields")
      assertEquals(result.exit, 1, result.diagnostics)
      assert(result.diagnostics.contains(diagnostic), result.diagnostics)
      assert(!Files.exists(output))

  test("an enum case's plain Int refuses at its own field declaration"):
    val source =
      accepted + "\ngiven Finite[Int] = Finite.upTo(2)\nenum Bad:\n  case counted(n: Int)\nobject BadFields extends Machine[Bad, Outcome, Nothing]:\n  val init = Bad.counted(0)\n  def end(s: State) = true\n  object rules extends Bindings()\n"
    val (result, output) = lift("enum-int-field", source, "BadFields")
    assertEquals(result.exit, 1, result.diagnostics)
    val fieldLine = source.linesIterator.toVector.indexWhere(_.contains("case counted(n: Int)")) + 1
    assert(
      result.diagnostics.contains("Bad$.counted.n") && result.diagnostics.contains(
        s"Bounds.scala:$fieldLine:"
      ),
      result.diagnostics
    )
    assert(!Files.exists(output))

  test("plain Int channels refuse their declaration even with Finite.upTo"):
    val source = """package fixture.boundedintegers
import framework.*
import framework.Finite.given
enum Outcome:
  case accepted
val small: Channel[Int] = channel[Int](capacity = 1, order = Order.fifo, loss = Loss.reliable)(using Finite.upTo(1))
final case class Mail(first: Inbox[Int])
given Finite[Mail] =
  given Finite[Inbox[Int]] = small.contents
  Finite.derived
object Messages extends Machine[Mail, Outcome, Nothing]:
  val init = Mail(small.empty)
  def end(s: State) = true
  object rules extends Bindings()
"""
    val (result, output) = lift("int-channel", source, "Messages")
    assertEquals(result.exit, 1, result.diagnostics)
    assert(
      result.diagnostics.contains("channel small") && result.diagnostics.contains("UpTo[N]"),
      result.diagnostics
    )
    assert(result.diagnostics.contains("Bounds.scala:6:"), result.diagnostics)
    assert(!Files.exists(output))

  test("affected fixtures preserve every value and exact declared coordinate delta"):
    val directory = Files.createDirectory(scratch.resolve("affected-fixtures"))
    val stored = "model/irgen/testdata/lifts/"
    for name <- List("Presence.scala", "Channels.scala", "Rejects.scala", "CollidingRejects.scala")
    do Files.copy(tools.directory.resolve(stored + name), directory.resolve(name)): Unit
    Files.writeString(
      directory.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n//> using options -Wconf:src=/Rejects[.]scala$&id=E176&msg=framework.StepBinding:s\n"
    ): Unit
    val jar = directory.resolve("fixtures.jar")
    tools
      .scalaCli(
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
    def run(name: String, roots: String*) =
      val output = directory.resolve(name + ".json")
      val result = InProcess.lift(
        tools.directory,
        Seq(s"$jar=$stored,$modelJar=model/", classpath.toString, output.toString) ++ roots
      )
      (result, output)
    for (name, roots) <- List(
        "presence" -> Seq("fixture.presence.Presence"),
        "channels" -> Seq("fixture.channels.Relay", "fixture.channels.Tallying")
      )
    do
      val (result, output) = run(name, roots*)
      assertEquals(result.exit, 0, result.diagnostics)
      FixturePositions.checkBounded(name, Files.readString(output))
      val corrupt = json.readTree(Files.readString(output))
      corrupt
        .path("functions")
        .get(0)
        .path("body")
        .asInstanceOf[com.fasterxml.jackson.databind.node.ObjectNode]
        .put("wrong", true): Unit
      intercept[IllegalArgumentException](
        FixturePositions.checkBounded(name, json.writeValueAsString(corrupt))
      ): Unit
      val misplaced = json.readTree(Files.readString(output))
      misplaced
        .path("types")
        .get(0)
        .path("position")
        .asInstanceOf[com.fasterxml.jackson.databind.node.ObjectNode]
        .put("line", 99999): Unit
      intercept[IllegalArgumentException](
        FixturePositions.checkBounded(name, json.writeValueAsString(misplaced))
      ): Unit
    val (refused, output) = run(
      "rejects",
      "fixture.rejects.Counter",
      "fixture.rejects.Counting",
      "fixture.rejects.Rejects$package$.omittedEmpty"
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(refused.diagnostics.contains("countdown calls itself"), refused.diagnostics)
    assert(
      refused.diagnostics.contains(
        "Rejects.scala:158: channel counts's messages are Int"
      ) && refused.diagnostics.contains("UpTo[N]"),
      refused.diagnostics
    )
    assert(
      refused.diagnostics.contains("input unreached of reach has no default"),
      refused.diagnostics
    )
    assert(!Files.exists(output))
