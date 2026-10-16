package framework.lint

import framework.`export`.*

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.{Ran, Tools}
import FixtureBuild.buildFixture
import framework.check.RetainedScratch

class ReifyModuleFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout = 10.minutes
  private val tools = Tools.here

  test(
    "the body engine is shared with export but cannot depend on its declaration host"
  ):
    val scratch =
      retained(tools.directory.resolve(".flow/tmp"), "reify-module.")
    val sources = scratch.resolve("sources")
    Files.createDirectories(sources): Unit
    for (name, code) <- Seq(
        "Inspect.scala" -> "package framework.inspect.modulefixture; object Value { val n = true }",
        "Body.scala" -> "package framework.reify.modulefixture; object Body { val n = true }",
        "Host.scala" -> "package framework.`export`.modulefixture; object Host { val n = true }",
        "ExportGood.scala" -> "package framework.`export`.modulefixture; import framework.reify.modulefixture.Body; object Good { val n = Body.n }",
        "ReifyGood.scala" -> "package framework.reify.modulefixture; import framework.inspect.modulefixture.Value; object Good { val n = Value.n }",
        "ReifyBad.scala" -> "package framework.reify.modulefixture; import framework.`export`.modulefixture.Host; object Bad { val n = Host.n }",
        "DslBad.scala" -> "package framework.dsl.modulefixture; import framework.reify.modulefixture.Body; object Bad { val n = Body.n }"
      )
    do Files.writeString(sources.resolve(name), code): Unit
    val jar = scratch.resolve("modules.jar")
    val compiled = tools
      .scalaCli(
        Seq(
          "compile",
          sources.toString,
          "model/framework/dsl",
          "model/framework/realize",
          "--server=false",
          "--print-class-path",
          "--compilation-output",
          jar.toString
        ),
        framework.check.Output.KeptApart
      )
      .orFail()
    val zip = new java.util.zip.ZipFile(jar.toFile)
    val tastys = try
      zip
        .entries()
        .asScala
        .filter(_.getName.contains("modulefixture/"))
        .filter(_.getName.endsWith(".tasty"))
        .map { entry =>
          val path = scratch.resolve(entry.getName)
          Files.createDirectories(path.getParent): Unit
          val input = zip.getInputStream(entry)
          try Files.copy(input, path): Unit
          finally input.close()
          path.toString
        }
        .toList
    finally zip.close()
    assert(tastys.nonEmpty)
    val checker = new ConventionInspector(Map.empty, ModulePolicy.Generic)
    val classpath =
      compiled.output.trim.split(java.io.File.pathSeparator).toList
    assert(
      scala.tasty.inspector.TastyInspector
        .inspectAllTastyFiles(tastys, Nil, classpath)(checker)
    )
    val errors = checker.errors.map(_.getMessage).mkString("\n")
    assert(
      errors.contains(
        "framework.reify.modulefixture may not import framework.export.modulefixture"
      ),
      errors
    )
    assert(
      errors.contains(
        "framework.dsl.modulefixture may not import framework.reify.modulefixture"
      ),
      errors
    )
    assert(!errors.contains("ReifyGood.scala"), errors)
    assert(!errors.contains("ExportGood.scala"), errors)

class ConventionsFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private lazy val scratch =
    retained(build.resolve("history"), "conventions.")

  test("nested sibling packages keep their full authored module identity"):
    val sources = scratch.resolve("nested-module")
    Files.createDirectories(sources): Unit
    Files.writeString(
      sources.resolve("Sibling.scala"),
      """package temporal:
        |  package features.activity.modulefixture:
        |    object Shared { val value = true }
        |
        |  package actors.worker.modulefixture:
        |    import framework.dsl.Actor
        |    object Good { val value = Actor("worker") }
        |
        |  package actors.client.modulefixture:
        |    import temporal.features.activity.modulefixture.Shared
        |    object Bad { val value = Shared.value }
        |""".stripMargin
    ): Unit
    val jar = scratch.resolve("nested-module.jar")
    val compiled = tools
      .scalaCli(
        Seq(
          "compile",
          sources.toString,
          "model/framework/dsl",
          "model/framework/realize",
          "--server=false",
          "--print-class-path",
          "--compilation-output",
          jar.toString
        ),
        framework.check.Output.KeptApart
      )
      .orFail()
    val zip = java.util.zip.ZipFile(jar.toFile)
    val tastys = try
      zip
        .entries()
        .asScala
        .filter(_.getName.contains("modulefixture/"))
        .filter(_.getName.endsWith(".tasty"))
        .map { entry =>
          val path = scratch.resolve(entry.getName)
          Files.createDirectories(path.getParent): Unit
          val input = zip.getInputStream(entry)
          try Files.copy(input, path): Unit
          finally input.close()
          path.toString
        }
        .toList
    finally zip.close()
    assert(tastys.nonEmpty)
    val policy = new ModulePolicy:
      val root = "temporal"
      def permitted(from: String, to: String, test: Boolean): Option[Boolean] =
        Option.when(to == "temporal.features.activity.modulefixture")(false)
      def testModule(file: Path): Option[String] = None
    val checker = ConventionInspector(Map.empty, policy)
    val classpath =
      compiled.output.trim.split(java.io.File.pathSeparator).toList
    assert(
      scala.tasty.inspector.TastyInspector
        .inspectAllTastyFiles(tastys, Nil, classpath)(checker)
    )
    val errors = checker.errors.map(_.getMessage).mkString("\n")
    assert(
      errors.contains(
        "temporal.actors.client.modulefixture may not import " +
          "temporal.features.activity.modulefixture"
      ),
      errors
    )
    assert(!errors.contains("temporal may not"), errors)
    assert(!errors.contains("Good.scala"), errors)

  private def fixture(name: String): Path =
    val sources =
      ModelBuild.copied(
        tools.directory.resolve(s"model/framework/export/testdata/$name"),
        scratch.resolve(name)
      )
    val jar = scratch.resolve(s"$name.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--test",
          sources.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar

  private lazy val negative = fixture("conventions")
  private lazy val patterns = fixture("enumPatterns")

  test("relocated production TASTy retains authored dotted membership"):
    val result = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.lintConventions",
        s"${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString
      )
    )
    assertEquals(result.exit, 0, result.output)
    assert(
      !result.output.contains("R4.dotted-in"),
      result.output
    )

  private def lift(jar: Path, stored: String, root: String): (Ran, Path) =
    val output = scratch.resolve(root.split('.').last + ".json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=model/framework/export/testdata/$stored/,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        output.toString,
        root
      )
    )
    result -> output

  test("typed convention diagnostics cover whole-case matches and dotted in"):
    val (result, output) =
      lift(negative, "conventions", "fixture.conventions.Checked")
    assertNotEquals(result.exit, 0, result.output)
    assert(!Files.exists(output), result.output)
    for rule <- Seq("R4.enum-wildcard", "R4.dotted-in") do
      assert(result.output.contains(rule), result.output)
    for name <- Seq("plain", "bound", "typedWhole", "stateWhole") do
      assert(result.output.contains(name), result.output)
    assert(!result.output.contains("unrelated matches"), result.output)
    assert(!result.output.contains("unrelatedEnum matches"), result.output)
    assert(!result.output.contains("WorkflowPhase unqualified"), result.output)
    assertEquals(
      result.output.linesIterator.count(_.contains("R4.dotted-in")),
      2,
      result.output
    )
    assert(result.output.contains("Checks.scala:"), result.output)

  test(
    "compiler-synthetic membership positions do not reject authored dotted calls"
  ):
    // The mini's Protocol, whose phase tests are authored dotted (fn-166.11; the standalone
    // activity's ActivityRecord before).
    val output = scratch.resolve("activity-protocol.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        output.toString,
        "temporal.features.activity.mini.protocol.ActivityProtocol"
      )
    )
    assertEquals(result.exit, 0, result.output)
    assert(Files.isRegularFile(output), result.output)
    assert(!result.output.contains("R4.dotted-in"), result.output)

  test(
    "authored undotted membership remains refused at its exact source lines"
  ):
    val (result, _) =
      lift(negative, "conventions", "fixture.conventions.Checked")
    val refused = result.output.linesIterator
      .filter(_.contains("R4.dotted-in"))
      .map(_.dropWhile(_ != '/').takeWhile(_ != ' '))
      .toList
    assertEquals(
      refused,
      List(
        "/framework/export/testdata/conventions/Checks.scala:38:",
        "/framework/export/testdata/conventions/Checks.scala:39:"
      ),
      result.output
    )

  test(
    "an import of a phase type's members is refused at its line, at any level"
  ):
    val (result, output) =
      lift(negative, "conventions", "fixture.conventions.phases.Staged")
    assertNotEquals(result.exit, 0, result.output)
    assert(!Files.exists(output), result.output)
    val refused = result.output.linesIterator
      .filter(l => l.contains("/PhaseImports.scala:") && l.contains("R9.phase-import"))
      .map(l => l.drop(l.indexOf("PhaseImports.scala:")).takeWhile(_ != ' '))
      .toList
    assertEquals(
      refused.sorted,
      List(
        "PhaseImports.scala:24:",
        "PhaseImports.scala:29:",
        "PhaseImports.scala:6:"
      ),
      result.output
    )
    assert(
      result.output.contains(
        "brings the cases of the phase type Stage into scope: drop it and name each phase " +
          "through its type, `Stage.<case>`"
      ),
      result.output
    )

  test(
    "typed imports reject forbidden edges and unqualified cross-level types"
  ):
    val (result, _) =
      lift(negative, "conventions", "fixture.conventions.Checked")
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

  test(
    "the lint entry checks framework and tests with distinct module policies"
  ):
    val zip = new java.util.zip.ZipFile(negative.toFile)
    val entries = try
      zip.entries().asScala.map(_.getName).filter(_.endsWith(".tasty")).toSet
    finally zip.close()
    assert(
      entries.exists(_.startsWith("framework/dsl/conventionfixtures/")),
      entries.toString
    )
    assert(
      entries.exists(_.startsWith("temporal/capabilities/discoveryfixtures/")),
      entries.toString
    )
    assert(
      entries.exists(_.startsWith("framework/dsl/discoveryfixtures/")),
      entries.toString
    )
    val result = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.lintConventions",
        s"$negative=model/framework/export/testdata/conventions/,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString
      )
    )
    assertNotEquals(result.exit, 0, result.output)
    assert(
      result.output.contains("/FrameworkImports.scala:3: R4.module-import:"),
      result.output
    )
    assert(
      result.output.contains(
        "/FrameworkImports.test.scala:3: R4.module-import:"
      ),
      result.output
    )
    assert(
      result.output.contains("framework.dsl.conventionfixtures may not import"),
      result.output
    )
    assert(
      result.output.contains("framework.dsl.discoveryfixtures may not import"),
      result.output
    )
    assert(!result.output.contains("/Imports.test.scala:"), result.output)
    for site <- Seq(3, 4) do
      assert(
        result.output.contains(
          s"/QualifiedRefs.scala:$site: R4.module-reference:"
        ),
        result.output
      )
    assert(!result.output.contains("/Qualified.scala:"), result.output)

  test(
    "typed remainders refuse incomplete guarded binding and overlapping branches"
  ):
    for (root, reason) <- Seq(
        "Guarded" -> "typed remainder has a guard",
        "Incomplete" -> "typed remainder is incomplete",
        "Overlapping" -> "typed remainder overlaps a case",
        "Binding" -> "typed remainder has a binding",
        "EarlierBinding" -> "typed remainder has a binding or partial earlier pattern"
      )
    do
      val (result, output) =
        lift(patterns, "enumPatterns", s"fixture.enumpatterns.$root")
      assertNotEquals(result.exit, 0, result.output)
      assert(result.output.contains(reason), result.output)
      assert(result.output.contains("Patterns.scala:"), result.output)
      assert(!Files.exists(output), result.output)

  test(
    "literal alternatives and nonfinal constructor payload wildcards remain explicit"
  ):
    val (result, output) =
      lift(patterns, "enumPatterns", "fixture.enumpatterns.Explicit")
    assertEquals(result.exit, 0, result.output)
    val json = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(output))
    val body = json
      .path("machines")
      .get(0)
      .path("ends")
      .path("lambda")
      .path("body")
      .path("binary")
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
    assertEquals(
      cases.get(0).path("pattern").path("case").path("case").asText(),
      "done"
    )
    assert(
      cases
        .get(0)
        .path("pattern")
        .path("case")
        .path("fields")
        .get(0)
        .has("wildcard"),
      body.toString
    )
    assertEquals(
      cases
        .elements()
        .asScala
        .toSeq
        .drop(1)
        .map(
          _.path("pattern").path("literal").path("enum").path("case").asText()
        ),
      Seq("running", "cancelReceived")
    )

  test(
    "a final typed constructor remainder preserves a preceding payload wildcard"
  ):
    val (result, output) =
      lift(patterns, "enumPatterns", "fixture.enumpatterns.Remainder")
    assertEquals(result.exit, 0, result.output)
    val json = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(output))
    val body =
      json.path("machines").get(0).path("ends").path("lambda").path("body")
    val cases = body.path("match").path("cases")
    assertEquals(cases.size(), 2, body.toString)
    assertEquals(
      cases.get(0).path("pattern").path("case").path("case").asText(),
      "done"
    )
    assert(
      cases
        .get(0)
        .path("pattern")
        .path("case")
        .path("fields")
        .get(0)
        .has("wildcard"),
      body.toString
    )
    assert(cases.get(1).path("pattern").has("wildcard"), body.toString)

  test(
    "proved literal and role prefixes retain their existing binding patterns"
  ):
    for root <- Seq("BoundPrefix", "RolePrefix") do
      val (result, output) =
        lift(patterns, "enumPatterns", s"fixture.enumpatterns.$root")
      assertEquals(result.exit, 0, result.output)
      val json =
        new com.fasterxml.jackson.databind.ObjectMapper()
          .readTree(Files.readString(output))
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
        cases
          .get(0)
          .path("pattern")
          .path("bind")
          .path("pattern")
          .has("literal"),
        json.toString
      )
      assert(cases.get(1).path("pattern").has("wildcard"), json.toString)

  test(
    "keeps accepts its typed equality argument without accepting a stored chain"
  ):
    val (result, output) =
      lift(patterns, "enumPatterns", "fixture.enumpatterns.Keeps$.queries")
    assertEquals(result.exit, 0, result.output)
    val json = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(output))
    assertEquals(json.path("properties").size(), 1, json.toString)
    val (refused, _) = lift(
      patterns,
      "enumPatterns",
      "fixture.enumpatterns.StoredKeeps$.queries"
    )
    assertNotEquals(refused.exit, 0, refused.output)
    assert(
      refused.output.contains("not storedOnce: chain it onto that call"),
      refused.output
    )
