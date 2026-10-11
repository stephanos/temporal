package umpire.irgen

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.{Ran, Tools}

class ConventionsFixtures extends munit.FunSuite:
  override val munitTimeout = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private lazy val scratch =
    Files.createTempDirectory(Files.createDirectories(build.resolve("history")), "conventions.")

  private def fixture(name: String): Path =
    val sources = Files.createDirectories(scratch.resolve(name))
    val stream = Files.newDirectoryStream(
      tools.directory.resolve(s"model/irgen/testdata/$name"),
      "*.scala"
    )
    try
      for from <- stream.asScala do
        Files.writeString(
          sources.resolve(from.getFileName),
          Files
            .readString(from)
            .replace("../../../build/model-scala.jar", build.resolve("model-scala.jar").toString)
        ): Unit
    finally stream.close()
    val jar = scratch.resolve(s"$name.jar")
    tools
      .scalaCli(
        Seq("--power", "package", "--library", "--test", sources.toString, "-f", "-o", jar.toString)
      )
      .orFail(): Unit
    jar

  private lazy val negative = fixture("conventions")
  private lazy val patterns = fixture("enumPatterns")

  private def lift(jar: Path, stored: String, root: String): (Ran, Path) =
    val output = scratch.resolve(root.split('.').last + ".json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=model/irgen/testdata/$stored/,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        output.toString,
        root
      )
    )
    result -> output

  test("typed convention diagnostics cover whole-case matches and dotted in"):
    val (result, output) = lift(negative, "conventions", "fixture.conventions.Checked")
    assertNotEquals(result.exit, 0, result.output)
    assert(!Files.exists(output), result.output)
    for rule <- Seq("R4.enum-wildcard", "R4.dotted-in") do
      assert(result.output.contains(rule), result.output)
    for name <- Seq("plain", "bound", "typedWhole", "stateWhole") do
      assert(result.output.contains(name), result.output)
    assert(!result.output.contains("unrelated matches"), result.output)
    assert(!result.output.contains("unrelatedEnum matches"), result.output)
    assert(!result.output.contains("WorkflowPhase unqualified"), result.output)
    assertEquals(result.output.linesIterator.count(_.contains("R4.dotted-in")), 2, result.output)
    assert(result.output.contains("Checks.scala:"), result.output)

  test("an import of a phase type's members is refused at its line, at any level"):
    val (result, output) = lift(negative, "conventions", "fixture.conventions.phases.Staged")
    assertNotEquals(result.exit, 0, result.output)
    assert(!Files.exists(output), result.output)
    val refused = result.output.linesIterator
      .filter(l => l.contains("/PhaseImports.scala:") && l.contains("R9.phase-import"))
      .map(l => l.drop(l.indexOf("PhaseImports.scala:")).takeWhile(_ != ' '))
      .toList
    assertEquals(
      refused.sorted,
      List("PhaseImports.scala:24:", "PhaseImports.scala:29:", "PhaseImports.scala:6:"),
      result.output
    )
    assert(
      result.output.contains(
        "brings the cases of the phase type Stage into scope: drop it and name each phase " +
          "through its type, `Stage.<case>`"
      ),
      result.output
    )

  test("typed imports reject forbidden edges and unqualified cross-level types"):
    val (result, _) = lift(negative, "conventions", "fixture.conventions.Checked")
    for rule <- Seq("R4.module-import", "R5.cross-level-import") do
      assert(result.output.contains(rule), result.output)
    for name <- Seq("Phase", "Fact", "State") do assert(result.output.contains(name), result.output)
    assert(result.output.contains("Imports.scala:"), result.output)
    for site <- Seq(8, 9); name <- Seq("Phase", "Fact", "State") do
      assert(
        result.output.linesIterator.exists(line =>
          line.contains(s"/Imports.scala:$site: R5.cross-level-import:") &&
            line.contains(s".$name unqualified")
        ),
        result.output
      )
    assert(result.output.contains("KindImports.scala:"), result.output)
    assert(result.output.contains("FormImports.scala:"), result.output)

  test("the lint entry checks framework and tests with distinct module policies"):
    val zip = new java.util.zip.ZipFile(negative.toFile)
    val entries = try zip.entries().asScala.map(_.getName).filter(_.endsWith(".tasty")).toSet
    finally zip.close()
    assert(entries.exists(_.startsWith("framework/conventionfixtures/")), entries.toString)
    assert(
      entries.exists(_.startsWith("temporal/capabilities/discoveryfixtures/")),
      entries.toString
    )
    assert(entries.exists(_.startsWith("framework/discoveryfixtures/")), entries.toString)
    val result = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "umpire.irgen.lintConventions",
        s"$negative=model/irgen/testdata/conventions/,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString
      )
    )
    assertNotEquals(result.exit, 0, result.output)
    assert(result.output.contains("/FrameworkImports.scala:3: R4.module-import:"), result.output)
    assert(
      result.output.contains("/FrameworkImports.test.scala:3: R4.module-import:"),
      result.output
    )
    assert(result.output.contains("framework.conventionfixtures may not import"), result.output)
    assert(result.output.contains("framework.discoveryfixtures may not import"), result.output)
    assert(!result.output.contains("/Imports.test.scala:"), result.output)
    for site <- Seq(3, 4) do
      assert(
        result.output.contains(s"/QualifiedRefs.scala:$site: R4.module-reference:"),
        result.output
      )
    assert(!result.output.contains("/Qualified.scala:"), result.output)

  test("typed remainders refuse incomplete guarded binding and overlapping branches"):
    for (root, reason) <- Seq(
        "Guarded" -> "typed remainder has a guard",
        "Incomplete" -> "typed remainder is incomplete",
        "Overlapping" -> "typed remainder overlaps a case",
        "Binding" -> "typed remainder has a binding",
        "EarlierBinding" -> "typed remainder has a binding or partial earlier pattern"
      )
    do
      val (result, output) = lift(patterns, "enumPatterns", s"fixture.enumpatterns.$root")
      assertNotEquals(result.exit, 0, result.output)
      assert(result.output.contains(reason), result.output)
      assert(result.output.contains("Patterns.scala:"), result.output)
      assert(!Files.exists(output), result.output)

  test("literal alternatives and nonfinal constructor payload wildcards remain explicit"):
    val (result, output) = lift(patterns, "enumPatterns", "fixture.enumpatterns.Explicit")
    assertEquals(result.exit, 0, result.output)
    val json = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(output))
    val body = json.path("machines").get(0).path("ends").path("lambda").path("body").path("binary")
    val alternatives = body
      .path("left")
      .path("match")
      .path("cases")
      .get(1)
      .path("pattern")
      .path("alternatives")
      .path("patterns")
    assertEquals(
      alternatives
        .elements()
        .asScala
        .toSeq
        .map(_.path("literal").path("enum").path("case").asText()),
      Seq("b", "c")
    )
    val cases = body.path("right").path("match").path("cases")
    assertEquals(cases.size(), 3, body.toString)
    assertEquals(cases.get(0).path("pattern").path("case").path("case").asText(), "done")
    assert(
      cases.get(0).path("pattern").path("case").path("fields").get(0).has("wildcard"),
      body.toString
    )
    assertEquals(
      cases
        .elements()
        .asScala
        .toSeq
        .drop(1)
        .map(_.path("pattern").path("literal").path("enum").path("case").asText()),
      Seq("running", "cancelReceived")
    )

  test("a final typed constructor remainder preserves a preceding payload wildcard"):
    val (result, output) = lift(patterns, "enumPatterns", "fixture.enumpatterns.Remainder")
    assertEquals(result.exit, 0, result.output)
    val json = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(output))
    val body = json.path("machines").get(0).path("ends").path("lambda").path("body")
    val cases = body.path("match").path("cases")
    assertEquals(cases.size(), 2, body.toString)
    assertEquals(cases.get(0).path("pattern").path("case").path("case").asText(), "done")
    assert(
      cases.get(0).path("pattern").path("case").path("fields").get(0).has("wildcard"),
      body.toString
    )
    assert(cases.get(1).path("pattern").has("wildcard"), body.toString)

  test("proved literal and role prefixes retain their existing binding patterns"):
    for root <- Seq("BoundPrefix", "RolePrefix") do
      val (result, output) = lift(patterns, "enumPatterns", s"fixture.enumpatterns.$root")
      assertEquals(result.exit, 0, result.output)
      val json =
        new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(output))
      val cases = json
        .path("machines")
        .get(0)
        .path("ends")
        .path("lambda")
        .path("body")
        .path("match")
        .path("cases")
      assertEquals(cases.size(), 2, json.toString)
      assert(cases.get(0).path("pattern").has("bind"), json.toString)
      assert(
        cases.get(0).path("pattern").path("bind").path("pattern").has("literal"),
        json.toString
      )
      assert(cases.get(1).path("pattern").has("wildcard"), json.toString)

  test("keeps accepts its typed equality argument without accepting a stored chain"):
    val (result, output) = lift(patterns, "enumPatterns", "fixture.enumpatterns.Keeps$.queries")
    assertEquals(result.exit, 0, result.output)
    val json = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(output))
    assertEquals(json.path("properties").size(), 1, json.toString)
    val (refused, _) = lift(patterns, "enumPatterns", "fixture.enumpatterns.StoredKeeps$.queries")
    assertNotEquals(refused.exit, 0, refused.output)
    assert(refused.output.contains("not storedOnce: chain it onto that call"), refused.output)
