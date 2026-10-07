package umpire.irgen

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.JsonNode
import umpire.check.{Ran, Tools}

class PhaseCapabilitiesSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/irgen/testdata/phaseCapabilities/"
  private val modelJar = build.resolve("model-scala.jar")
  private val scratch =
    Files.createTempDirectory(Files.createDirectories(build.resolve("history")), "phases.")
  private val fixtureJar = scratch.resolve("phases.jar")
  private val mapper = new com.fasterxml.jackson.databind.ObjectMapper()

  override def beforeAll(): Unit =
    val source = tools.directory.resolve(stored)
    val stream = Files.newDirectoryStream(source, "*.scala")
    try
      for from <- stream.asScala do
        Files.writeString(
          scratch.resolve(from.getFileName),
          Files.readString(from).replace("../../../build/model-scala.jar", modelJar.toString)
        )
    finally stream.close()
    tools
      .scalaCli(
        Seq("--power", "package", "--library", scratch.toString, "-f", "-o", fixtureJar.toString)
      )
      .orFail(): Unit

  private def lift(roots: String*): (Ran, Path) =
    val out = scratch.resolve(roots.head + ".json")
    val java = Path.of(System.getProperty("java.home"), "bin", "java").toString
    val ran = tools.run(
      java,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "umpire.irgen.lift",
        s"$fixtureJar=$stored,$modelJar=model/",
        build.resolve("model-scala.classpath").toString,
        out.toString
      ) ++ roots.map("fixture.phasecapabilities." + _)
    )
    (ran, out)

  private def enumCases(expression: JsonNode): Seq[String] =
    expression
      .path("binary")
      .path("right")
      .path("list")
      .path("items")
      .elements()
      .asScala
      .map(_.path("literal").path("enum").path("case").asText())
      .toSeq

  private def enumTypes(node: JsonNode): Set[String] =
    if node.isArray then node.elements().asScala.flatMap(enumTypes).toSet
    else if node.isObject then
      val own = Option(node.get("enum")).map(_.path("type").asText()).filter(_.nonEmpty).toSet
      own ++ node.elements().asScala.flatMap(enumTypes)
    else Set.empty

  test("Pausable and Pollable lower exact role sets through direct and derived projections"):
    val (ran, out) = lift(
      "Direct$.capabilities",
      "OtherDirect$.capabilities",
      "DirectDerived$.capabilities",
      "DirectPair$.capabilities",
      "DirectDerivedPair$.capabilities"
    )
    ran.orFail()
    val model = mapper.readTree(Files.readString(out))
    val properties = model.path("properties").elements().asScala.toSeq
    assertEquals(properties.size, 5)
    assertEquals(
      properties.map(_.path("machine").asText()).distinct.sorted,
      Seq("direct", "directDerived", "directDerivedPair", "directPair", "otherDirect")
    )
    for property <- properties do
      assertEquals(
        property.path("name").asText(),
        s"${property.path("machine").asText()}.pausedIsNotDispatched"
      )
      val holds = model
        .path("functions")
        .elements()
        .asScala
        .find(_.path("name") == property.path("holds"))
        .get
      val fromName = holds
        .path("body")
        .path("binary")
        .path("left")
        .path("unary")
        .path("operand")
        .path("call")
        .path("function")
        .asText()
      val paused = model
        .path("functions")
        .elements()
        .asScala
        .find(_.path("name").asText() == fromName)
        .get
        .path("body")
      assertEquals(enumCases(paused), Seq("paused"))
      val neverName = holds
        .path("body")
        .path("binary")
        .path("right")
        .path("unary")
        .path("operand")
        .path("call")
        .path("function")
        .asText()
      val running = model
        .path("functions")
        .elements()
        .asScala
        .find(_.path("name").asText() == neverName)
        .get
        .path("body")
      assertEquals(enumCases(running), Seq("pausedWhileHeld", "started"))
      val pausedPhase = paused.path("binary").path("left").path("field")
      val runningPhase = running.path("binary").path("left").path("field")
      assertEquals(pausedPhase.path("field").asText(), "phase")
      assertEquals(runningPhase.path("field").asText(), "phase")
      if property.path("machine").asText().contains("Pair") then
        assertEquals(pausedPhase.path("base").path("field").path("field").asText(), "left")
        assertEquals(runningPhase.path("base").path("field").path("field").asText(), "left")

  test("Pausable and Pollable refuse phases missing their owned roles"):
    for (root, line, role, phase) <- Seq(
        ("MissingSuspended", 98, "Suspended", "NoPausedPhase"),
        ("MissingHeld", 116, "Held", "NoHeldPhase")
      )
    do
      val (ran, _) = lift(s"$root$$.capabilities")
      assert(ran.failed, ran.output)
      assertEquals(
        ran.output.linesIterator.filter(_.startsWith("lift:")).toSeq,
        Seq(
          s"lift: ${stored}Pausing.scala:$line: ${root.head.toLower}${root.tail}'s phase type fixture.phasecapabilities.$phase has no $role case"
        )
      )

  test("capability Properties use their owning object when short machine names collide"):
    for branch <- Seq("first", "second") do
      val (ran, out) = lift(s"$branch.Same$$.capabilities")
      ran.orFail()
      val model = mapper.readTree(Files.readString(out))
      val property = model
        .path("properties")
        .elements()
        .asScala
        .find(_.path("name").asText() == "same.terminalStatesAreFinal")
        .get
      val holds = model
        .path("functions")
        .elements()
        .asScala
        .find(_.path("name").asText() == property.path("holds").asText())
        .get
      assertEquals(
        enumTypes(holds.path("body")),
        Set(s"fixture.phasecapabilities.$branch.ClosePhase")
      )
