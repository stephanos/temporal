package umpire.irgen

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.JsonNode
import umpire.check.Tools

class EnteringTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch = umpire.check.Scratch.directory("entering.")
  private val modelJar = ModelBuild.jar
  private val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
  private val source =
    Files.readString(tools.directory.resolve("model/irgen/testdata/lifts/Entering.scala"))

  private def lifted(name: String, text: String) =
    val dir = Files.createDirectory(scratch.resolve(name))
    Files.writeString(dir.resolve("Entering.scala"), text): Unit
    Files.writeString(
      dir.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = dir.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          dir.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val output = dir.resolve("fixture.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=model/irgen/testdata/lifts/,$modelJar=model/",
        ModelBuild.classpath.toString,
        output.toString,
        "fixture.entering.Entering",
        "fixture.entering.Cored"
      )
    )
    (result, output)

  test(
    "isolated Recorded rules and actual choose arms retain core values, identities and source positions"
  ):
    val (result, output) = lifted("accepted", source)
    assertEquals(result.exit, 0, result.diagnostics)
    val ir = mapper.readTree(Files.readString(output))
    val functions =
      ir.path("functions").elements().asScala.map(f => f.path("name").asText() -> f).toMap
    def body(n: String) = functions(n).path("body")
    def arm(n: String) = body(n).path("if").path("then")
    val allowedReferences = Map(
      "fixture.entering.Entering$.effects$.restOrRetry" -> "fixture.entering.Cored$.effects$.restOrRetry",
      "fixture.entering.Entering$.effects$.restOrRetry$retried" -> "fixture.entering.Cored$.effects$.restOrRetry$retried"
    )
    var positions = 0
    def same(a: JsonNode, b: JsonNode, path: String): Unit =
      if a.isObject && b.isObject then
        assertEquals(a.fieldNames().asScala.toSet, b.fieldNames().asScala.toSet, path)
        for key <- a.fieldNames().asScala do
          if key == "position" then
            for p <- List(a.path(key), b.path(key)) do
              assertEquals(p.path("file").asText(), "model/irgen/testdata/lifts/Entering.scala")
              assert(
                p.path("line").asInt() > 0 && p.path("line").asInt() <= source.linesIterator.size
              )
              assertEquals(p.fieldNames().asScala.toSet, Set("file", "line"))
              positions += 1
          else same(a.path(key), b.path(key), path + "/" + key)
      else if a.isArray && b.isArray then
        assertEquals(a.size(), b.size(), path)
        a.elements().asScala.zip(b.elements().asScala).zipWithIndex.foreach { case ((x, y), i) =>
          same(x, y, path + s"/$i")
        }
      else
        assertEquals(
          allowedReferences.getOrElse(a.asText(), a.toString),
          if allowedReferences.contains(a.asText()) then b.asText() else b.toString,
          path
        )
    same(arm("entering.rules.press"), body("fixture.entering.Cored$.effects$.work"), "work")
    same(
      body("entering.rules.press").path("if").path("else").path("if").path("then"),
      body("cored.rules.press").path("if").path("else").path("if").path("then"),
      "choose-call"
    )
    same(
      body("fixture.entering.Entering$.effects$.restOrRetry"),
      body("fixture.entering.Cored$.effects$.restOrRetry"),
      "actual-choose"
    )
    same(arm("entering.rules.drop"), body("fixture.entering.Cored$.effects$.settle"), "waiting")
    assert(positions > 30)
    assertEquals(
      ir.path("machines").elements().asScala.map(_.path("name").asText()).toSet,
      Set("entering", "cored")
    )
    assertEquals(
      ir.path("actions").elements().asScala.map(_.path("id").asText()).toSet,
      Set("fixture.entering.hand.press", "fixture.entering.hand.drop")
    )

  test("the isolated lifter refuses an absent typed writer"):
    val invalid = source.replace("(_.stage, (s, p) => s.copy(stage = p))", "(_.stage)")
    val (result, output) = lifted("missing-writer", invalid)
    assertEquals(result.exit, 1, result.diagnostics)
    assert(result.diagnostics.contains("declares none"), result.diagnostics)
    assert(!Files.exists(output))
