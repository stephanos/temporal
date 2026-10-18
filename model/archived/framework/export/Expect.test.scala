package framework.`export`

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import framework.check.{Ran, RetainedScratch, Tools}
import FixtureBuild.buildFixture

// Expectation blocks lift to the ordinary predicates their core forms are (fn-164.3): each block of
// testdata/expect lifts to the same function as its independently written core twin, apart from
// the function's name and its source positions, and its function is positioned at the block.
class ExpectFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val stored = "model/framework/export/testdata/expect/"
  private val mapper = new ObjectMapper()
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "expect-lifter.")
  private lazy val jar: Path =
    val sources = Files.createDirectories(scratch.resolve("sources"))
    val stream = Files.newDirectoryStream(tools.directory.resolve(stored), "*.scala")
    try
      for file <- stream.asScala do
        Files.writeString(
          sources.resolve(file.getFileName),
          ModelBuild.fixtureSettings(Files.readString(file))
        ): Unit
    finally stream.close()
    val out = scratch.resolve("expect.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          sources.toString,
          "-f",
          "-o",
          out.toString
        )
      )
      .orFail(): Unit
    out

  private def lift(root: String): JsonNode =
    val out = scratch.resolve(s"$root.json")
    val ran: Ran = InProcess.lift(
      tools.directory,
      Seq(s"$jar=$stored,${ModelBuild.jar}=model/", ModelBuild.classpath.toString, out.toString) :+
        s"framework.fixture.expect.Expect$$package$$.$root"
    )
    assertEquals(ran.exit, 0, ran.output)
    mapper.readTree(Files.readString(out))
  private lazy val sides: Map[String, JsonNode] =
    Seq("queries", "cores", "skewedQueries").map(r => r -> lift(r)).toMap
  // The lift that holds a Property: blocks, their core twins, or the skewed twin.
  private def lifted(name: String): JsonNode =
    if name == "skewed" then sides("skewedQueries")
    else if name.endsWith("Core") then sides("cores")
    else sides("queries")

  private def withoutPositions(node: JsonNode): JsonNode =
    if node.isObject then
      val out = mapper.createObjectNode()
      for row <- node.fields().asScala if row.getKey != "position" && row.getKey != "name" do
        out.set[JsonNode](row.getKey, withoutPositions(row.getValue)): Unit
      out
    else if node.isArray then
      val out = mapper.createArrayNode()
      node.elements().asScala.foreach(n => out.add(withoutPositions(n)): Unit)
      out
    else node

  private def property(name: String): JsonNode =
    lifted(name)
      .path("properties")
      .elements()
      .asScala
      .find(_.path("name").asText() == name)
      .getOrElse(
        fail(s"no Property $name in ${lifted(name).path("properties")}")
      )
  private def function(property: String): JsonNode =
    val named = this.property(property).path("holds").asText()
    lifted(property)
      .path("functions")
      .elements()
      .asScala
      .find(_.path("name").asText() == named)
      .getOrElse(
        fail(s"no function $named")
      )
  // The Property and its predicate, without the names and positions that tell twins apart.
  private def shape(name: String): (JsonNode, JsonNode) =
    val p = property(name).deepCopy[com.fasterxml.jackson.databind.node.ObjectNode]()
    p.remove("holds"): Unit
    (withoutPositions(p), withoutPositions(function(name)))

  for name <- Seq("heartbeat", "finished", "reset", "settled", "explicit") do
    test(s"the $name expectation lifts to its core twin's predicate"):
      assertEquals(shape(name), shape(name + "Core"))

  test(
    "a block that only forwards retains the standalone expectation it calls, as a core lambda does"
  ):
    val forwarded = property("forwarded").path("holds").asText()
    assertEquals(forwarded, "framework.fixture.expect.Expect$package$.beatingNow")
    assertEquals(
      property("forwardedCore").path("holds").asText(),
      "framework.fixture.expect.Expect$package$.beatingNowCore"
    )
    assertEquals(shape("forwarded"), shape("forwardedCore"))

  test("a twin of another association is told apart"):
    assertNotEquals(shape("heartbeat")._2, shape("skewed")._2)

  test("an expectation's parameters are its context names, after and before by default"):
    def params(name: String) =
      function(name).path("params").elements().asScala.map(_.path("name").asText()).toList
    assertEquals(params("heartbeat"), List("after"))
    assertEquals(params("reset"), List("before", "after"))
    assertEquals(params("explicit"), List("b", "a"))
    assertEquals(property("reset").path("transition").asBoolean(), true)
    assertEquals(property("heartbeat").path("transition").asBoolean(), false)

  test("an expectation's predicate is positioned at its authored block"):
    val source = Files.readAllLines(tools.directory.resolve(stored + "Expect.scala")).asScala
    for name <- Seq("heartbeat", "finished", "reset", "settled", "explicit") do
      val line = source.indexWhere(_.startsWith(s"val $name = ")) + 1
      val at = function(name).path("position")
      assertEquals(at.path("file").asText().split('/').last, "Expect.scala", name)
      assert(
        at.path("line").asInt() >= line && at.path("line").asInt() <= line + 1,
        s"$name at $at, authored at $line"
      )
