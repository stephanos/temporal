package framework.`export`

import TestEquality.given

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.JsonNode
import framework.check.{Ran, Tools}
import FixtureBuild.buildFixture
import framework.check.RetainedScratch

class DefaultEndsSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/framework/export/testdata/defaultEnds/"
  private val modelJar = build.resolve("model-scala.jar")
  private val scratch =
    retained(build.resolve("history"), "ends.")
  private val fixtureJar = scratch.resolve("ends.jar")
  private val mapper = new com.fasterxml.jackson.databind.ObjectMapper()

  override def beforeAll(): Unit =
    val source = tools.directory.resolve(stored)
    val stream = Files.newDirectoryStream(source, "*.scala")
    try
      for from <- stream.asScala do
        Files.writeString(
          scratch.resolve(from.getFileName),
          Files.readString(from).replace("../../../../build/model-scala.jar", modelJar.toString)
        )
    finally stream.close()
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          scratch.toString,
          "-f",
          "-o",
          fixtureJar.toString
        )
      )
      .orFail(): Unit

  private def lift(roots: String*): (Ran, Path) =
    val out = scratch.resolve(roots.head + ".json")
    val ran = InProcess.lift(
      tools.directory,
      Seq(
        s"$fixtureJar=$stored,$modelJar=model/",
        build.resolve("model-scala.classpath").toString,
        out.toString
      ) ++
        roots.map("fixture.defaultends." + _)
    )
    (ran, out)

  test("default machine and nested composition ends read Closed; derivations forward source ends"):
    val (ran, out) = lift("DefaultEnd", "WrittenEnd", "DerivedEnd", "DefaultPair", "DerivedPair")
    ran.orFail(): Unit
    val model = mapper.readTree(Files.readString(out))
    def ends(kind: String, name: String): JsonNode =
      model
        .path(kind)
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .get
        .path("ends")
    def cases(end: JsonNode) = end
      .path("lambda")
      .path("body")
      .path("binary")
      .path("right")
      .path("list")
      .path("items")
      .elements()
      .asScala
      .map(_.path("literal").path("enum").path("case").asText())
      .toSeq
    for (kind, name) <- Seq(
        "machines" -> "defaultEnd",
        "machines" -> "writtenEnd",
        "compositions" -> "defaultPair"
      )
    do assertEquals(cases(ends(kind, name)), Seq("done", "failed"), name)
    def unpositioned(node: JsonNode): JsonNode =
      if node.isObject then
        val result = mapper.createObjectNode()
        for field <- node.fields().asScala if field.getKey != "position"
        do result.set[JsonNode](field.getKey, unpositioned(field.getValue)): Unit
        result
      else if node.isArray then
        val result = mapper.createArrayNode()
        node.elements().asScala.foreach(value => result.add(unpositioned(value)))
        result
      else node
    assertEquals(
      unpositioned(ends("machines", "defaultEnd")),
      unpositioned(ends("machines", "writtenEnd"))
    )
    val projected = ends("compositions", "defaultPair")
      .path("lambda")
      .path("body")
      .path("binary")
      .path("left")
      .path("field")
    assertEquals(projected.path("field").asText(), "phase")
    assertEquals(projected.path("base").path("field").path("field").asText(), "left")
    assertEquals(ends("machines", "derivedEnd"), ends("machines", "defaultEnd"))
    assertEquals(ends("compositions", "derivedPair"), ends("compositions", "defaultPair"))

  test("an explicit end wins even when the phase has no Closed case"):
    val (ran, out) = lift("OpenOverride", "OpenOverridePair")
    ran.orFail(): Unit
    val model = mapper.readTree(Files.readString(out))
    for kind <- Seq("machines", "compositions") do
      val end = model.path(kind).get(0).path("ends")
      assertEquals(end.path("lambda").path("body").path("binary").path("op").asText(), "OP_EQ")

  test("missing Closed is refused statically with the object and phase type"):
    for (name, line) <- Seq("OpenEnd" -> 61, "OpenDefaultPair" -> 68) do
      val (ran, _) = lift(name)
      assert(ran.failed, ran.output)
      assertEquals(
        ran.output.linesIterator.filter(_.startsWith("lift:")).toSeq,
        Seq(
          s"lift: ${stored}Ends.scala:$line: ${name.head.toLower}${name.tail}'s phase type fixture.defaultends.OpenPhase has no Closed case"
        )
      )

  test("Closable brings both Properties exactly where declared, with typed derived projections"):
    val (ran, out) = lift(
      "ClosingEnd$.capabilities",
      "OtherClosing$.capabilities",
      "ClosingDerived$.capabilities",
      "ClosingPair$.capabilities",
      "ClosingDerivedPair$.capabilities",
      "ClosingDerivedAgain$.capabilities",
      "WrittenEnd"
    )
    ran.orFail(): Unit
    val model = mapper.readTree(Files.readString(out))
    val properties = model.path("properties").elements().asScala.toSeq
    assertEquals(
      properties.map(_.path("machine").asText()).distinct.sorted,
      Seq(
        "closingDerived",
        "closingDerivedAgain",
        "closingDerivedPair",
        "closingEnd",
        "closingPair",
        "otherClosing"
      )
    )
    for machine <- properties.map(_.path("machine").asText()).distinct do
      assertEquals(
        properties
          .filter(
            _.path("machine").asText() == machine
          )
          .map(_.path("name").asText())
          .sorted,
        Seq(s"$machine.closedIsRejectedUniformly", s"$machine.terminalStatesAreFinal")
      )
    for p <- properties if p.path("name").asText().endsWith("terminalStatesAreFinal") do
      assertEquals(
        p.path("origin").path("name").asText(),
        "temporal.capabilities.product.Closable.terminalStatesAreFinal"
      )
      val function =
        model
          .path("functions")
          .elements()
          .asScala
          .find(_.path("name") == p.path("holds"))
          .get
      val membership = function
        .path("body")
        .path("binary")
        .path("left")
        .path("unary")
        .path("operand")
        .path("binary")
      assertEquals(membership.path("op").asText(), "OP_CONTAINS")
      val cases = membership
        .path("right")
        .path("list")
        .path("items")
        .elements()
        .asScala
        .map(_.path("literal").path("enum").path("case").asText())
        .toSeq
      assertEquals(cases, Seq("done", "failed"))
      val phase = membership.path("left").path("field")
      assertEquals(phase.path("field").asText(), "phase")
      if p.path("machine").asText().contains("Pair") || p
          .path("machine")
          .asText() == "closingDerivedAgain"
      then assertEquals(phase.path("base").path("field").path("field").asText(), "left")

  test("Closable refuses a missing Phased and a missing Closed by machine name"):
    val authored = Files.readAllLines(tools.directory.resolve(stored + "Closing.scala")).asScala
    def declarationLine(root: String, fragment: String): Int =
      val first = authored.indexWhere(_.startsWith("object " + root))
      assert(first >= 0, root)
      val next = authored.indexWhere(_.startsWith("object "), first + 1)
      val last = if next < 0 then authored.size else next
      val calls = authored.zipWithIndex
        .slice(first, last)
        .filter((text, _) => text.contains(fragment))
      assertEquals(calls.size, 1, root)
      calls.head._2 + 1
    for (root, fragment, message) <- Seq(
        (
          "ClosingUnphased",
          "Closable[Snapshot",
          "closingUnphased declares Closable but no phase: mix in Phased[State, Phase](_.phase)"
        ),
        (
          "ClosingOpen",
          "Phased[OpenSnapshot",
          "closingOpen's phase type fixture.defaultends.OpenPhase has no Closed case"
        )
      )
    do
      val (ran, _) = lift(s"$root$$.capabilities")
      assert(ran.failed, ran.output)
      assertEquals(
        ran.output.linesIterator.filter(_.startsWith("lift:")).toSeq,
        Seq(s"lift: ${stored}Closing.scala:${declarationLine(root, fragment)}: $message")
      )
