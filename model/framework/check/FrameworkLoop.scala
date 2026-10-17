package framework.check

import java.io.PrintStream
import java.nio.file.{Files, Path}
import framework.check.lint.Boundary

object FrameworkSuites extends Registry:
  val source = "model/framework/check/FrameworkLoop.scala"
  val current: Set[String] = Set.empty
  val baselines: Map[String, Baseline] = Map.empty
  val committedIrHeld = true
  val selectsEveryFeature: Set[Area] = Set.empty
  val rules = Seq(
    "model/framework/dsl/**" -> Area.Framework,
    "model/framework/realize/**" -> Area.Framework,
    "model/framework/check/**" -> Area.Check,
    "model/framework/**" -> Area.Lifter
  )
  val modules = Seq(
    // The core's tests include the native samples' suites (package framework.dsl); the reusable
    // library compiled for constructed fixtures stays DSL/realize only (ModelSources.frameworkCore).
    Module(
      "core",
      Seq("model/framework/dsl", "model/framework/realize", "model/framework/samples"),
      Seq("model/framework/dsl", "model/framework/realize", "model/framework/samples")
    ),
    Module("check", Seq("model/framework/check"), Seq("model/framework/check")),
    Module(
      "export",
      Seq("model/framework/export", "model/framework/lint"),
      Seq("model/framework/export")
    )
  )
  def suites: Map[String, Covers] = declared(Tools.here.directory)
    .map(d => d.name -> Covers(Set(area(d.file))))
    .toMap

  private val genericExport = Set(
    "ConstructedActorsSuite",
    "ConstructedBindingsSuite",
    "ConstructedDerivationsSuite",
    "ConstructedEndpointsSuite",
    "ConstructedEvidenceSuite",
    "ConstructedFacetsSuite",
    "ConstructedHeadersSuite",
    "ConstructedInboxesSuite",
    "ConstructedNamesSuite",
    "ConstructedRealizationsSuite",
    "ConstructedReferencesSuite",
    "ConstructedResidualSignatureSuite",
    "ConstructedRulesSuite",
    "ConstructedSignatureSuite",
    "FiniteSchemaSuite",
    "ReifyBoundarySuite",
    "InlineSugarSuite",
    "ExpectFixtures",
    "ExpectCaptureSuite",
    "DerivedIds",
    "RolesSuite",
    "FacetAdmissionTest",
    "ReifyModuleFixtures",
    "BoundedIntegersTest",
    "InputDefaultsTest",
    "EnteringTest",
    "DefinitionExpansionTest",
    "DeclarationSugarFixtures",
    "SugarFixtures",
    "QualifiedRulesTest",
    "FacetsTest",
    "AdmissionSuite",
    "EligibleWitnessFixtures",
    "EffectDefinitionsSuite",
    "PropertyGroupsFixtures",
    "QualifiedPhasesTest",
    "OrderFixtures",
    "SourcePositionsFixtures",
    "LiftableSubsetFixtures",
    "StructureFixtures",
    "MarkersFixtures",
    "WitnessFixtures",
    "LayoutWitnessFixtures",
    "FixtureIdentityContractSuite",
    "FrameworkFixtureSettingsSuite",
    "ConventionsFixtures",
    "MiniLayoutFixtures",
    "LintStagesSuite",
    "Overlap"
  )
  private val integration = Map(
    "BoundsTest" -> "system Bounds and mini IR",
    "CapabilityNamesTest" -> "system Bounds, capabilities and mini IR/waivers",
    "DefaultEndsSuite" -> "system capability definitions",
    "PhaseCapabilitiesSuite" -> "system capability definitions",
    "ObservationsTest" -> "system realization kit and activity/service API messages",
    "TypedDeclarationsTest" -> "system API and execution messages",
    "RetiredSpellingsTest" -> "system realization history kit",
    "InputDeadlinePresetsTest" -> "system realization kit, activity definitions and service API",
    "DeadlinePresetsSuite" -> "system realization kit, activity definitions and service API",
    "GapsSuite" -> "system realization gap fixture",
    "GapResolutionSuite" -> "system capabilities, realization kit and API",
    "StatusTableSeamFixtures" -> "negative fixture constructs a system service request",
    "LayoutFilesFixtures" -> "realization kit and service API in module-policy fixtures",
    "QualifiedNames" -> "system capabilities, realization kit and API fixtures",
    "Fixtures" -> "system-backed lifts fixture set",
    "HintSnapshotsSuite" -> "system realization hints",
    "MiniRealizationFixtures" -> "real mini realizations and generated IR",
    "ActivityHeartbeatFixtures" -> "activity heartbeat and service API",
    "ExpectationFixtures" -> "system realization expectations",
    "ReifySamplesSuite" ->
      "the samples' native Set, Map, plain Int and Option[Int] state need the IR schema of fn-173.4 (schema and Go held)"
  )

  def held(suite: Declared): Option[String] =
    if suite.name == "framework.check.lint.GuideSourcesTest" then
      Some("SEMANTICS excerpts read the owner's held system sources")
    else if suite.module.name != "export" || genericExport(suite.name.split('.').last) then None
    else
      Some(
        integration.getOrElse(
          suite.name.split('.').last,
          throw GateError(
            s"unclassified framework suite ${suite.name}: declare generic coverage or a held integration reason"
          )
        ) + "; owner hold"
      )

final class FrameworkLoop(outer: Tools, out: PrintStream, record: Path):
  private val root = outer.directory
  private val build = Files.createDirectories(record.resolve("build"))
  private val tools = outer.withoutEnvironment(
    "FEATURES",
    "BASE",
    "INTERPRETER_PARITY",
    "SAMPLE_REPORT_INPUT",
    "UMPIRE_LIFTER_UPDATE"
  )
  private val report = record.resolve("record.md")

  private def note(line: String): Unit =
    out.println(line)
    Files.writeString(
      report,
      line + "\n",
      java.nio.file.StandardOpenOption.CREATE,
      java.nio.file.StandardOpenOption.APPEND
    ): Unit

  private def run(
      title: String,
      arguments: Seq[String],
      runner: Tools = tools,
      apart: Boolean = false
  ): Ran =
    note(s"RUN $title: scala-cli ${arguments.mkString(" ")}")
    val log = record.resolve(title + ".log")
    val ran = runner.scalaCli(arguments, if apart then Output.KeptApart else Output.Logged(log))
    if apart then Files.writeString(log, ran.output + ran.standardError): Unit
    note(s"EXIT $title: ${ran.exit}")
    ran.orFail()

  def check(): Unit =
    val findings = Boundary.findings(root, Seq("temporal", "io.temporal.api"))
    if findings.nonEmpty then throw GateError(findings.mkString("\n"))
    note("PASS framework source/build-directive boundary")
    val declared = FrameworkSuites.declared(root)
    if declared.isEmpty then throw GateError("no framework suites discovered")
    val selected = declared.filter: suite =>
      FrameworkSuites.held(suite) match
        case Some(reason) => note(s"HELD ${suite.name} (${suite.file}): $reason"); false
        case None         => note(s"GENERIC ${suite.name} (${suite.file})"); true
    val ir = root.resolve("model/build/ir-scalapb.jar")
    if !Files.isRegularFile(ir) then
      throw GateError(s"missing existing IR jar: $ir; schema generation is held")
    val core = build.resolve("framework.jar")
    val compiled = run(
      "core",
      Seq(
        "compile",
        "--server=false",
        "--print-class-path",
        "--compilation-output",
        core.toString,
        "--scalac-option",
        "-sourceroot",
        "--scalac-option",
        root.resolve("model").toString
      ) ++ ModelSources.frameworkCore,
      apart = true
    )
    val entries = compiled.output.trim.split(java.io.File.pathSeparator).toSeq
    if entries.isEmpty || !Files.isRegularFile(core) ||
      entries.exists(entry => !Files.exists(Path.of(entry))) ||
      entries.exists(entry =>
        Seq("api-scalapb.jar", "model-scala.jar").contains(Path.of(entry).getFileName.toString)
      )
    then throw GateError("framework compilation returned an invalid or system-dependent classpath")
    Files.writeString(
      build.resolve("framework.classpath"),
      (core.toString +: entries.drop(1)).mkString(java.io.File.pathSeparator)
    ): Unit
    run(
      "plugin",
      Seq(
        "--power",
        "package",
        "--server=false",
        "--assembly",
        "--preamble=false",
        "--main-class-in-manifest=false",
        "--provided",
        "org.scala-lang:scala3-compiler_3",
        "model/framework/reify/plugin",
        "-f",
        "-o",
        build.resolve("reify.jar").toString
      )
    ): Unit
    val testing = tools.withEnvironment("UMPIRE_FRAMEWORK_BUILD" -> build.toString)
    for module <- FrameworkSuites.modules do
      val suites = selected.filter(_.module == module)
      if suites.isEmpty then throw GateError(s"no generic ${module.name} suites selected")
      val tested = run(
        s"test-${module.name}",
        Seq("run", "--test", "--server=false") ++ module.inputs ++
          Seq("--main-class", "org.junit.runner.JUnitCore", "--") ++ suites.map(_.name),
        testing
      )
      val count = "(?m)^OK \\(([0-9]+) tests?\\)".r
        .findFirstMatchIn(tested.output)
        .map(_.group(1).toInt)
        .getOrElse(0)
      if count == 0 then throw GateError(s"${module.name} reported no executed tests")
      tested.output.linesIterator.filter(_.startsWith("HELD test ")).toVector.distinct.foreach(note)
      note(
        s"PASS ${module.name}: ${suites.size} suites, $count tests; ignored integration cases earn no pass credit"
      )
    note(
      s"PASS framework gate: ${selected.size} generic suites; ${declared.size - selected.size} held suites"
    )

@main def checkFramework(): Unit =
  val tools = Tools.here
  val record = Files.createTempDirectory(
    Files.createDirectories(tools.directory.resolve(".flow/tmp/checks")),
    "framework."
  )
  println(s"Framework check record: $record")
  FrameworkLoop(tools, System.out, record).check()
