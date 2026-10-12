// What a change to a path of the repository puts in scope, and what every Scala suite of the model
// covers (fn-166.13). `make model-check-changed` (ChangeCheck.scala) reads only this file to turn a diff
// into the features to compile, lift and compare, and the suites to run. It is data, kept in one
// place, so a move of the tree (fn-169.1 moves the lifter, the checks and the migration harness
// into model/framework/{export,check,migrate} and the Temporal parts into model/temporal/...)
// updates the path rules and the modules below and nothing else.
package umpire.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// What a changed path is part of.
enum Area derives CanEqual:
  // The DSL and the generic realization core.
  case Framework

  // The lifter (the IR generator) and its fixtures.
  case Lifter

  // The gate and the checks, their tests and fixtures.
  case Check

  // The migration harness, its rules and their mini proofs.
  case Migrate

  // Code every feature compiles against: foundations, actors, capabilities, realize, `Bounds`.
  case Shared

  // The build settings every Scala module reads.
  case Build

  // The model targets of the Makefile.
  case Make

  // The recorded rename maps of the equivalence baselines below.
  case Baselines

  // A feature's own sources or outputs; the feature is resolved from the path (ModelSources).
  case Feature

  // Prose: it selects nothing, unless a suite names the file among its paths (GuideSourcesTest).
  case Docs

  // Build output, scratch and everything outside the model's Scala: selects nothing.
  case Outside

// A suite's declared reach: the areas it reads, the features whose Models it compiles, lifts or
// reads (a feature that is behind skips it), and single files it reads outside those areas.
// `anyFeature` marks a suite that reads every compiled feature's sources or tables, whichever they
// are; a change to any feature selects it, and a behind feature never skips it.
final case class Covers(
    areas: Set[Area],
    features: Set[String] = Set.empty,
    paths: Set[String] = Set.empty,
    anyFeature: Boolean = false
):
  def and(more: Area*): Covers = copy(areas = areas ++ more)
  def reading(feature: String*): Covers = copy(features = features ++ feature)
  def files(path: String*): Covers = copy(paths = paths ++ path)
  def everyFeature: Covers = copy(anyFeature = true)

// A Scala module whose suites run in one JVM: where its suites are, and what scala-cli builds it from
// (the Models' sources come from ModelSources, for the features in scope).
final case class Module(name: String, suiteRoots: Seq[String], inputs: Seq[String])
    derives CanEqual:
  def models: Boolean = inputs.isEmpty

object Coverage:
  // The features that are not behind: a DSL change keeps them current (owner, 2026-10-11). Every
  // other discovered feature is behind until the owner's sweep runs the migration rules on it.
  val current: Set[String] = Set("activity_mini")

  // The step-table baseline each current feature's meaning is compared with, and the recorded
  // rename map from that baseline to the tree. A task that renames appends its entries to the map.
  final case class Baseline(directory: String, renames: String)
  val baselines: Map[String, Baseline] = Map(
    "activity_mini" -> Baseline(
      ".flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2",
      "model/check/baselines/activity_mini.map"
    )
  )

  // In the source phase a feature's checked-in IR is regenerated only at the sweep, so a fresh lift
  // that differs from it is reported as held, not as a failure; meaning is proven by the step-table
  // equivalence above. The batch-end and sweep gates (`make model-mini`, `umpire-check-model`) keep
  // the strict compare.
  val committedIrHeld: Boolean = true

  // A path's area: the first rule whose glob matches it (`*` within a segment, `**` across them).
  // The planned fn-169.1 homes are listed already; their rules match nothing until the move.
  val rules: Seq[(String, Area)] = Seq(
    "model/build/**" -> Area.Outside,
    "**/.scala-build/**" -> Area.Outside,
    "**/.bsp/**" -> Area.Outside,
    "**/*.md" -> Area.Docs,
    // fn-169.1: the generic lifter, checks and migration harness move into the framework, and their
    // Temporal parts beside the Models; listed before model/framework/** and model/temporal/**.
    "model/framework/export/**" -> Area.Lifter,
    "model/framework/check/**" -> Area.Check,
    "model/framework/migrate/**" -> Area.Migrate,
    "model/temporal/export/**" -> Area.Lifter,
    "model/temporal/check/**" -> Area.Check,
    "model/temporal/migrate/**" -> Area.Migrate,
    "model/framework/**" -> Area.Framework,
    "model/irgen/**" -> Area.Lifter,
    "model/check/baselines/**" -> Area.Baselines,
    "model/check/**" -> Area.Check,
    "model/testdata/**" -> Area.Check,
    "model/migrate/**" -> Area.Migrate,
    "model/temporal/features/**" -> Area.Feature,
    "model/temporal/**" -> Area.Shared,
    "model/project.scala" -> Area.Build,
    "model/OutputPaths.scala" -> Area.Build,
    "model/.scalafix.conf" -> Area.Build,
    "model/.scalafmt.conf" -> Area.Build,
    "mise.toml" -> Area.Build,
    "Makefile" -> Area.Make,
    "**" -> Area.Outside
  )

  // The areas whose change selects every current feature: each feeds the compile, the lift or the
  // checks of every feature.
  val selectsEveryFeature: Set[Area] = Set(
    Area.Framework,
    Area.Lifter,
    Area.Check,
    Area.Migrate,
    Area.Shared,
    Area.Build,
    Area.Make,
    Area.Baselines
  )

  // The modules and where their suites are. fn-169.1 moves the lifter's, the checks' and the
  // harness's suites with their code.
  val modules: Seq[Module] = Seq(
    Module("models", Seq("model/framework", "model/temporal"), Nil),
    Module("lifter", Seq("model/irgen/test"), Seq("model/irgen")),
    Module("check", Seq("model/check/test"), Seq("model/check")),
    Module("migrate", Seq("model/migrate/test"), Seq("model/migrate"))
  )

  private val mini = "activity_mini"
  private val standalone = "activity/standalone"
  private val nexusFeatures = Seq("nexus/workflow", "nexus/standalone")
  // What every suite of a module reads by building on it.
  private val framework = Covers(Set(Area.Framework, Area.Build))
  private val models = framework.and(Area.Shared)
  private val lifter =
    models.and(Area.Lifter).files("model/check/Tools.scala", "model/check/Coverage.scala")
  private val check = Covers(Set(Area.Check, Area.Build))
  private val migrate = Covers(Set(Area.Migrate, Area.Build))

  // Every suite, by its class name, and what it covers. A suite missing here fails the check
  // (CoverageSuite, and the command refuses to run).
  val suites: Map[String, Covers] = Map(
    // The framework's own suites.
    "framework.BoundedIntegersSuite" -> framework,
    "framework.Choices" -> framework,
    "framework.EffectsTest" -> framework,
    "framework.Equality" -> framework,
    "framework.ExpectationTest" -> framework,
    "framework.FacetsTest" -> framework,
    "framework.GapsSuite" -> framework,
    "framework.InputDefaults" -> framework,
    "framework.Inputs" -> framework,
    "framework.IrFileDiscoveryTest" -> framework,
    "framework.MachineLawsHoleTest" -> framework,
    "framework.ObservationTest" -> framework,
    "framework.PhasedSuite" -> framework,
    "framework.RolesSuite" -> framework,
    "framework.RuleSourceTest" -> framework,
    "framework.RulesTest" -> framework,
    "framework.StepTablesTest" -> framework,
    "framework.WitnessTest" -> framework,
    // The shared code's suites; the laws, pins and IR files of every compiled feature.
    "framework.ExportedMachineLawsTest" -> models.everyFeature,
    "framework.MachineTablePins" -> models.everyFeature,
    "temporal.IrFilesTest" -> models.everyFeature,
    "temporal.PropertyGroupMigrationTest" -> models.reading(standalone +: nexusFeatures*),
    "temporal.RolePredicatesTest" -> models.reading(standalone +: nexusFeatures*),
    "framework.CapabilityNamesSuite" -> models,
    "temporal.capabilities.CapabilityPropertiesTest" -> models.reading(
      standalone +: nexusFeatures*
    ),
    "framework.ClosableSuite" -> models,
    "framework.PausableSuite" -> models,
    "framework.CapabilityExpirableSuite" -> models,
    "framework.CapabilityRetryableSuite" -> models,
    "temporal.realize.TemporalExpectationTest" -> models,
    "temporal.realize.RejectionsTest" -> models,
    // The features' suites.
    "framework.AnswerEnteringTest" -> models.reading(mini),
    "temporal.features.activity.mini.protocol.MiniProtocolTest" -> models.reading(mini),
    "framework.NexusOutcomePins" -> models.reading(nexusFeatures*),
    "framework.NexusRoleRefinements" -> models.reading(nexusFeatures*),
    "framework.ActivityByIDRegression" -> models.reading(standalone),
    "framework.ActivityDispatchRegression" -> models.reading(standalone),
    "framework.ActivityHeartbeatRegression" -> models.reading(standalone),
    "framework.ActivityPrecisionRegression" -> models.reading(standalone),
    "framework.ActivityRejectionRegression" -> models.reading(standalone),
    "framework.ActivityResetRegression" -> models.reading(standalone),
    "framework.ActivityRetryRegression" -> models.reading(standalone),
    "framework.ActivitySubjectRegression" -> models.reading(standalone),
    "framework.ActivityVisibilityRegression" -> models.reading(standalone),
    "framework.CapabilityRetryDeadlineFeatures" -> models.reading(standalone, "nexus/workflow"),
    "framework.RoleRefinements" -> models.reading(standalone),
    "framework.StandaloneActivityPins" -> models.reading(standalone),
    // The lifter's suites (fn-166.11): each builds its own fixtures under testdata against the
    // packaged Models, admitting only what the fixture's roots reach; these also lift or import the
    // mini feature. A test of a real feature inside them is held for the sweep (`SweepHeld`).
    "umpire.irgen.AdmissionSuite" -> lifter,
    "umpire.irgen.BoundedIntegersTest" -> lifter,
    "umpire.irgen.BoundsTest" -> lifter.reading(mini),
    "umpire.irgen.CapabilityNamesTest" -> lifter.reading(mini),
    "umpire.irgen.ConventionsFixtures" -> lifter.reading(mini),
    "umpire.irgen.InputDeadlinePresetsTest" -> lifter.reading(mini),
    "umpire.irgen.DeadlinePresetsSuite" -> lifter.reading(mini),
    "umpire.irgen.DefaultEndsSuite" -> lifter,
    "umpire.irgen.DerivedIds" -> lifter,
    "umpire.irgen.EligibleWitnessFixtures" -> lifter,
    "umpire.irgen.EnteringTest" -> lifter,
    "umpire.irgen.FacetAdmissionTest" -> lifter.reading(mini),
    "umpire.irgen.FacetsTest" -> lifter,
    "umpire.irgen.FixtureIdentityContractSuite" -> lifter,
    "umpire.irgen.HintSnapshotsSuite" -> lifter.reading(mini),
    "umpire.irgen.MiniLayoutFixtures" -> lifter.reading(mini),
    "umpire.irgen.MiniRealizationFixtures" -> lifter.reading(mini),
    "umpire.irgen.LayoutWitnessFixtures" -> lifter,
    "umpire.irgen.WitnessFixtures" -> lifter,
    "umpire.irgen.ActivityHeartbeatFixtures" -> lifter,
    "umpire.irgen.ExpectationFixtures" -> lifter,
    "umpire.irgen.Fixtures" -> lifter.reading(mini),
    "umpire.irgen.GapResolutionSuite" -> lifter,
    "umpire.irgen.GapsSuite" -> lifter,
    "umpire.irgen.InputDefaultsTest" -> lifter,
    "umpire.irgen.LayoutFilesFixtures" -> lifter,
    "umpire.irgen.ObservationsTest" -> lifter,
    "umpire.irgen.Overlap" -> lifter,
    "umpire.irgen.PhaseCapabilitiesSuite" -> lifter,
    "umpire.irgen.PropertyGroupsFixtures" -> lifter,
    "umpire.irgen.QualifiedNames" -> lifter.reading(mini),
    "umpire.irgen.QualifiedPhasesTest" -> lifter,
    "umpire.irgen.QualifiedRulesTest" -> lifter,
    "umpire.irgen.RolesSuite" -> lifter,
    "umpire.irgen.SourcePositionsFixtures" -> lifter,
    "umpire.irgen.StatusTableSeamFixtures" -> lifter,
    // The checks' suites. The repository rules read the framework's, the lifter's and every
    // feature's sources; the coverage suite reads this file and every module's suites.
    "umpire.check.BlockFormRuleSuite" -> check,
    "umpire.check.CapabilityVocabularySuite" -> check.and(Area.Framework, Area.Lifter),
    "umpire.check.CommentRuleSuite" ->
      check.and(Area.Framework, Area.Shared, Area.Lifter).everyFeature,
    "umpire.check.CoverageSuite" ->
      check.and(Area.Framework, Area.Shared, Area.Lifter, Area.Migrate, Area.Make),
    "umpire.check.EqualitySuite" -> check,
    "umpire.check.EquivalenceSuite" -> check,
    "umpire.check.GateEquivalenceOutputsSuite" -> check,
    "umpire.check.GateSuite" -> check,
    "umpire.check.MakeFeaturesSuite" -> check.and(Area.Make),
    "umpire.check.ModelSourcesSuite" -> check,
    "umpire.check.ProtoLiteralsSuite" -> check,
    "umpire.check.SourceMetricsSuite" -> check,
    "umpire.check.StepLedgerSuite" -> check,
    "umpire.check.SyntaxRuleSuite" -> check
      .and(Area.Framework, Area.Shared, Area.Lifter)
      .everyFeature,
    "umpire.check.ToolsSuite" -> check,
    // The migration rules' suites, which rewrite source text; some read the mini feature's or the
    // shared code's files as they are.
    "umpire.migrate.BoundedIntegersTest" -> migrate,
    "umpire.migrate.BoundsInventoryTest" -> migrate.and(Area.Shared).reading(mini),
    "umpire.migrate.CapabilityNamesTest" -> migrate,
    "umpire.migrate.CentralBoundsTest" -> migrate.and(Area.Shared),
    "umpire.migrate.EntersTest" -> migrate,
    "umpire.migrate.FacetWrappingTest" -> migrate,
    "umpire.migrate.FamilyNamesTest" -> migrate.and(Area.Shared).reading(mini),
    "umpire.migrate.GapsSuite" -> migrate,
    "umpire.migrate.GuideSourcesTest" ->
      migrate.and(Area.Framework, Area.Shared).reading(mini).files("model/SEMANTICS.md"),
    "umpire.migrate.InputDefaultsTest" -> migrate,
    "umpire.migrate.ObservationReadTest" -> migrate,
    "umpire.migrate.QualifiedActionsTest" -> migrate,
    "umpire.migrate.QualifiedPhasesTest" -> migrate,
    "umpire.migrate.QueueAliasTest" -> migrate.reading(mini)
  )

  // A path's area by the rules above.
  def area(path: String): Area =
    rules.collectFirst { case (glob, area) if matches(glob, path) => area }.getOrElse(Area.Outside)

  // `**/` also matches no directory at all, so `**/*.md` matches `AGENTS.md`.
  private def matches(glob: String, path: String): Boolean =
    def segment(part: String) =
      part.split("\\*", -1).map(java.util.regex.Pattern.quote).mkString("[^/]*")
    val pattern = glob
      .split("\\*\\*/", -1)
      .map(_.split("\\*\\*", -1).map(segment).mkString(".*"))
      .mkString("(?:.*/)?")
    path.matches(pattern)

  // A suite as the sources declare it: its module, its file and its class name.
  final case class Declared(module: Module, file: String, name: String)

  private val suiteClass =
    "(?m)^(?:final |private |sealed )*class ([A-Za-z0-9_]+)[^\\n]*?\\bextends\\s+(?:munit\\.)?FunSuite\\b".r
  private val packageClause = "package\\s+([A-Za-z0-9_.]+)\\s*".r

  // Every suite the modules' sources declare. A source's package is its leading package clauses, so
  // a `package` line inside a string of a test is not read as one.
  def declared(root: Path): Vector[Declared] =
    modules.toVector.flatMap: module =>
      module.suiteRoots.toVector.flatMap: suiteRoot =>
        val base = root.resolve(suiteRoot)
        if !Files.isDirectory(base) then Vector.empty
        else
          val stream = Files.walk(base)
          val files =
            try
              stream.iterator.asScala
                .filter: path =>
                  val name = path.getFileName.toString
                  val skipped = base
                    .relativize(path)
                    .iterator
                    .asScala
                    .toSeq
                    .init
                    .exists(p => Set("testdata", "build", ".scala-build", ".bsp")(p.toString))
                  Files.isRegularFile(path) && !skipped &&
                  (if module.models then name.endsWith(".test.scala") else name.endsWith(".scala"))
                .toVector
                .sortBy(_.toString)
            finally stream.close()
          files.flatMap: path =>
            val source = Files.readString(path)
            val packages = source.linesIterator
              .map(_.trim)
              .takeWhile(line =>
                line.isEmpty || line.startsWith("//") || line.startsWith("package ")
              )
              .collect { case packageClause(name) => name }
              .toVector
            val prefix = if packages.isEmpty then "" else packages.mkString(".") + "."
            suiteClass
              .findAllMatchIn(source)
              .map(found =>
                Declared(module, root.relativize(path).toString, prefix + found.group(1))
              )
              .toVector
