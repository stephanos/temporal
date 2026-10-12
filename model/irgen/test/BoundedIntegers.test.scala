package umpire.irgen

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.Tools

class BoundedIntegersTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch = Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "bounded-integers.")
  private val json = new com.fasterxml.jackson.databind.ObjectMapper()
  private lazy val modelJar =
    val jar = scratch.resolve("framework.jar")
    tools.scalaCli(Seq("--power", "package", "--server=false", "--library", "model/project.scala", "model/framework", "-f", "-o", jar.toString)).orFail(): Unit
    jar
  private lazy val classpath =
    val result = tools.scalaCli(Seq("compile", "--server=false", "--print-class-path", "model/project.scala", "model/framework")).orFail()
    val cp = result.diagnostics.trim.split("\n").last
    val path = scratch.resolve("framework.classpath")
    Files.writeString(path, s"$modelJar:${cp.substring(cp.indexOf(':') + 1)}"): Unit
    path
  private val accepted = Files.readString(tools.directory.resolve("model/irgen/testdata/boundedIntegers/Bounds.scala"))

  private def lift(name: String, source: String, root: String) =
    val directory = Files.createDirectory(scratch.resolve(name))
    Files.writeString(directory.resolve("Bounds.scala"), source): Unit
    Files.writeString(directory.resolve("project.scala"), "//> using scala 3.9.0\n//> using jvm 27\n"): Unit
    val jar = directory.resolve("fixture.jar")
    tools.scalaCli(Seq("--power", "package", "--server=false", "--library", "--classpath", Files.readString(classpath).trim, directory.toString, "-f", "-o", jar.toString)).orFail(): Unit
    val output = directory.resolve("fixture.json")
    val result = InProcess.lift(tools.directory, Seq(s"$jar=model/irgen/testdata/boundedIntegers/,$modelJar=model/", classpath.toString, output.toString, s"fixture.boundedintegers.$root"))
    (result, output)

  private def record(ir: com.fasterxml.jackson.databind.JsonNode, name: String) =
    ir.path("types").elements().asScala.find(_.path("name").asText() == s"fixture.boundedintegers.$name").get.path("record").path("fields").elements().asScala.toVector

  test("each literal, singleton, zero and named opaque field retains its own range"):
    val (result, output) = lift("fields", accepted, "Fields")
    assertEquals(result.exit, 0, result.diagnostics)
    val fields = record(json.readTree(Files.readString(output)), "Counters")
    assertEquals(fields.map(_.path("type").path("intRange").path("high").asInt()), Vector(0, 1, 3, 2))

  test("differently bounded optional fields retain distinct ranges and enum identities"):
    val (result, output) = lift("options", accepted, "Options")
    assertEquals(result.exit, 0, result.diagnostics)
    val ir = json.readTree(Files.readString(output))
    val names = record(ir, "Optional").map(_.path("type").path("named").asText())
    assertEquals(names.distinct.size, 2)
    val highs = names.map(n => ir.path("types").elements().asScala.find(_.path("name").asText() == n).get.path("enum").path("cases").get(1).path("fields").get(0).path("type").path("intRange").path("high").asInt())
    assertEquals(highs, Vector(1, 3))

  test("differently bounded inboxes associate with their declared channel and message range"):
    val (result, output) = lift("messages", accepted, "Messages")
    assertEquals(result.exit, 0, result.diagnostics)
    val ir = json.readTree(Files.readString(output))
    assertEquals(record(ir, "Mail").map(_.path("type").path("channel").asText()), Vector("fixture.boundedintegers.small", "fixture.boundedintegers.large"))
    assertEquals(ir.path("channels").elements().asScala.map(_.path("message").path("intRange").path("high").asInt()).toSet, Set(1, 3))

  test("plain Int fields refuse at the field even with a local finite range"):
    val source = accepted.replace("zero: UpTo[0]", "zero: Int").replace("object Fields extends", "given Finite[Counters] =\n  given Finite[Int] = Finite.upTo(0)\n  Finite.derived\n\nobject Fields extends").replace("Counters(UpTo(0),", "Counters(0,")
    val (result, output) = lift("int-field", source, "Fields")
    assertEquals(result.exit, 1, result.diagnostics)
    assert(result.diagnostics.contains("Counters.zero") && result.diagnostics.contains("UpTo[N]"), result.diagnostics)
    assert(result.diagnostics.contains("Bounds.scala:13:"), result.diagnostics)
    assert(!Files.exists(output))

  test("plain Int channels refuse their declaration even with Finite.upTo"):
    val source = accepted.replace("UpTo[1]", "Int").replace("channel[Int](capacity = 1, order = Order.fifo, loss = Loss.reliable)", "channel[Int](capacity = 1, order = Order.fifo, loss = Loss.reliable)(using Finite.upTo(1))").replace("final case class Optional", "given Finite[Int] = Finite.upTo(1)\nfinal case class Optional")
    val (result, output) = lift("int-channel", source, "Messages")
    assertEquals(result.exit, 1, result.diagnostics)
    assert(result.diagnostics.contains("channel small") && result.diagnostics.contains("UpTo[N]"), result.diagnostics)
    assert(!Files.exists(output))
