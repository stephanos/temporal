// Temporal's coverage registry (fn-166.13): what a change to a path of the repository puts in scope,
// and what every Scala suite of the model covers. `make model-check-changed` (ChangeCheck.scala,
// model/framework/check) reads only this file to turn a diff into the features to compile, lift and
// compare, and the suites to run. It is data, kept in one place, so a move of the tree updates the
// path rules and the modules below and nothing else.
package temporal.check

import framework.check.*

object Coverage extends Registry:
  val source = "model/temporal/check/Coverage.scala"

  // The features that are not behind: a DSL change keeps them current (owner, 2026-10-11). Every
  // other discovered feature is behind until the owner's sweep runs the migration rules on it.
  val current: Set[String] = Set("activity_mini")

  // The step-table baseline each current feature's meaning is compared with, and the recorded
  // rename map from that baseline to the tree. A task that renames appends its entries to the map.
  val baselines: Map[String, Baseline] = Map(
    "activity_mini" -> Baseline(
      ".flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2",
      "model/temporal/check/baselines/activity_mini.map"
    )
  )

  // In the source phase a feature's checked-in IR is regenerated only at the sweep, so a fresh lift
  // that differs from it is reported as held, not as a failure; meaning is proven by the step-table
  // equivalence above. The batch-end and sweep gates (`make model-mini`, `umpire-check-model`) keep
  // the strict compare.
  val committedIrHeld: Boolean = true

  // A path's area: the first rule whose glob matches it (`*` within a segment, `**` across them).
  // The lifter, the checks and the migration harness live in the framework, and their Temporal parts
  // beside the Models (fn-169.1); they are listed before model/framework/** and model/temporal/**.
  val rules: Seq[(String, Area)] = Seq(
    "model/build/**" -> Area.Outside,
    "**/.scala-build/**" -> Area.Outside,
    "**/.bsp/**" -> Area.Outside,
    "**/*.md" -> Area.Docs,
    "model/framework/export/**" -> Area.Lifter,
    "model/framework/lint/**" -> Area.Lifter,
    "model/framework/inspect/**" -> Area.Lifter,
    "model/temporal/export/**" -> Area.Lifter,
    "model/temporal/check/baselines/**" -> Area.Baselines,
    "model/framework/check/**" -> Area.Check,
    "model/temporal/check/**" -> Area.Check,
    "model/framework/migrate/**" -> Area.Migrate,
    "model/temporal/migrate/**" -> Area.Migrate,
    "model/framework/**" -> Area.Framework,
    "model/testdata/**" -> Area.Check,
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

  // The modules and where their suites are: each tool's suites sit beside its sources, in its
  // framework half and its Temporal half, and run in a build of both, the framework's first.
  private val tooling = Seq("export", "check", "migrate")
  val modules: Seq[Module] = Seq(
    Module(
      "models",
      Seq("model/framework/dsl", "model/framework/realize", "model/temporal"),
      Nil,
      skipped = tooling.map(t => s"model/temporal/$t")
    )
  ) ++ Seq("lifter" -> "export", "check" -> "check", "migrate" -> "migrate").map((name, t) =>
    val halves = Seq(s"model/framework/$t", s"model/temporal/$t") ++
      (if t == "export" then Seq("model/framework/lint", "model/framework/inspect") else Nil)
    Module(name, halves, halves)
  )

  private val mini = "activity_mini"
  private val standalone = "activity/standalone"
  private val nexusFeatures = Seq("nexus/workflow", "nexus/standalone")
  // What every suite of a module reads by building on it.
  private val framework = Covers(Set(Area.Framework, Area.Build))
  private val models = framework.and(Area.Shared)
  private val lifter = models
    .and(Area.Lifter)
    .files(
      "model/framework/check/Tools.scala",
      "model/framework/check/Registry.scala",
      "model/temporal/check/Coverage.scala"
    )
  private val check = Covers(Set(Area.Check, Area.Build))
  private val migrate = Covers(Set(Area.Migrate, Area.Build))

  // Every suite, by its class name, and what it covers. A suite missing here fails the check
  // (CoverageSuite, and the command refuses to run).
  val suites: Map[String, Covers] = Map(
    // The framework's own suites.
    "framework.dsl.BoundedIntegersSuite" -> framework,
    "framework.dsl.Choices" -> framework,
    "framework.dsl.EffectsTest" -> framework,
    "framework.dsl.Equality" -> framework,
    "framework.dsl.ExpectationTest" -> framework,
    "framework.dsl.FacetsTest" -> framework,
    "framework.dsl.GapsSuite" -> framework,
    "framework.dsl.InputDefaults" -> framework,
    "framework.dsl.Inputs" -> framework,
    "framework.dsl.IrFileDiscoveryTest" -> framework,
    "framework.dsl.MachineLawsHoleTest" -> framework,
    "framework.dsl.ObservationTest" -> framework,
    "framework.dsl.PhasedSuite" -> framework,
    "framework.dsl.RolesSuite" -> framework,
    "framework.dsl.RuleSourceTest" -> framework,
    "framework.dsl.RulesTest" -> framework,
    "framework.dsl.StepTablesTest" -> framework,
    "framework.dsl.SugarNativeTest" -> framework,
    "framework.dsl.WitnessTest" -> framework,
    "framework.realize.RealizationConstructionTest" -> framework,
    // The shared code's suites; the laws, pins and IR files of every compiled feature.
    "framework.dsl.ExportedMachineLawsTest" -> models.everyFeature,
    "framework.dsl.MachineTablePins" -> models.everyFeature,
    "temporal.IrFilesTest" -> models.everyFeature,
    "temporal.PropertyGroupMigrationTest" -> models.reading(standalone +: nexusFeatures*),
    "temporal.RolePredicatesTest" -> models.reading(standalone +: nexusFeatures*),
    "framework.dsl.CapabilityNamesSuite" -> models,
    "temporal.capabilities.CapabilityPropertiesTest" -> models.reading(
      standalone +: nexusFeatures*
    ),
    "framework.dsl.ClosableSuite" -> models,
    "framework.dsl.PausableSuite" -> models,
    "framework.dsl.CapabilityExpirableSuite" -> models,
    "framework.dsl.CapabilityRetryableSuite" -> models,
    "temporal.realize.TemporalExpectationTest" -> models,
    "temporal.realize.RejectionsTest" -> models,
    "temporal.realize.TemporalRealizationConstructionTest" -> models,
    "temporal.realize.TemporalRealizationConversionTest" -> models,
    // The features' suites.
    "framework.dsl.AnswerEnteringTest" -> models.reading(mini),
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
    "framework.lint.AdmissionSuite" -> lifter,
    "framework.export.BoundedIntegersTest" -> lifter,
    "framework.export.RetiredSpellingsTest" -> lifter,
    "framework.export.TypedDeclarationsTest" -> lifter,
    "framework.export.DefinitionExpansionTest" -> lifter,
    "framework.export.DeclarationSugarFixtures" -> lifter,
    "framework.lint.LiftableSubsetFixtures" -> lifter,
    "framework.export.SugarFixtures" -> lifter,
    "framework.export.BoundsTest" -> lifter.reading(mini),
    "framework.export.CapabilityNamesTest" -> lifter.reading(mini),
    "framework.lint.ConventionsFixtures" -> lifter.reading(mini),
    "framework.export.InputDeadlinePresetsTest" -> lifter.reading(mini),
    "framework.export.DeadlinePresetsSuite" -> lifter.reading(mini),
    "framework.export.DefaultEndsSuite" -> lifter,
    "framework.export.DerivedIds" -> lifter,
    "framework.lint.EligibleWitnessFixtures" -> lifter,
    "framework.export.EnteringTest" -> lifter,
    "framework.export.FacetAdmissionTest" -> lifter.reading(mini),
    "framework.export.FacetsTest" -> lifter,
    "framework.export.FixtureIdentityContractSuite" -> lifter,
    "framework.export.HintSnapshotsSuite" -> lifter.reading(mini),
    "framework.export.MiniLayoutFixtures" -> lifter.reading(mini),
    "framework.export.MiniRealizationFixtures" -> lifter.reading(mini),
    "framework.export.ConstructedRealizationsSuite" -> lifter.reading(mini),
    "framework.export.LayoutWitnessFixtures" -> lifter,
    "framework.export.WitnessFixtures" -> lifter,
    "framework.export.ActivityHeartbeatFixtures" -> lifter,
    "framework.export.ExpectationFixtures" -> lifter,
    "framework.export.Fixtures" -> lifter.reading(mini),
    "framework.export.LintStagesSuite" -> lifter.reading(mini),
    "framework.lint.OrderFixtures" -> lifter,
    "framework.lint.StructureFixtures" -> lifter,
    "framework.lint.MarkersFixtures" -> lifter,
    "framework.export.GapResolutionSuite" -> lifter,
    "framework.export.GapsSuite" -> lifter,
    "framework.export.InputDefaultsTest" -> lifter,
    "framework.lint.LayoutFilesFixtures" -> lifter,
    "framework.export.ObservationsTest" -> lifter,
    "framework.export.Overlap" -> lifter,
    "framework.export.PhaseCapabilitiesSuite" -> lifter,
    "framework.lint.PropertyGroupsFixtures" -> lifter,
    "framework.export.QualifiedNames" -> lifter.reading(mini),
    "framework.lint.QualifiedPhasesTest" -> lifter,
    "framework.export.QualifiedRulesTest" -> lifter,
    "framework.export.RolesSuite" -> lifter,
    "framework.export.SourcePositionsFixtures" -> lifter,
    "framework.export.StatusTableSeamFixtures" -> lifter,
    "temporal.export.LifterCoverageSuite" -> lifter,
    // The checks' suites. The repository rules read the framework's, the lifter's and every
    // feature's sources; the coverage suite reads this file and every module's suites.
    "framework.check.lint.BlockFormRuleSuite" -> check,
    "framework.check.lint.BoundarySuite" -> check,
    "temporal.check.CapabilityVocabularySuite" -> check.and(Area.Framework, Area.Lifter),
    "framework.check.lint.CommentRuleSuite" ->
      check.and(Area.Framework, Area.Shared, Area.Lifter).everyFeature,
    "temporal.check.CoverageSuite" ->
      check.and(Area.Framework, Area.Shared, Area.Lifter, Area.Migrate, Area.Make),
    "temporal.check.EqualitySuite" -> check,
    "framework.check.equivalence.EquivalenceSuite" -> check,
    "framework.check.equivalence.InterpreterTablesSuite" -> check,
    "framework.check.equivalence.LiftsSuite" -> check,
    "temporal.check.GateEquivalenceOutputsSuite" -> check,
    "temporal.check.GateSuite" -> check,
    // It also holds the repository's framework to its boundary.
    "temporal.check.GateLintSuite" -> check.and(Area.Framework, Area.Lifter, Area.Migrate),
    "temporal.check.MakeFeaturesSuite" -> check.and(Area.Make),
    "framework.check.ModelSourcesSuite" -> check,
    "framework.check.SamplesSuite" -> check.and(Area.Framework, Area.Lifter),
    "temporal.check.lint.ProtoLiteralsSuite" -> check,
    "temporal.check.SourceMetricsSuite" -> check,
    "framework.check.equivalence.StepLedgerSuite" -> check,
    "framework.check.lint.SyntaxRuleSuite" -> check
      .and(Area.Framework, Area.Shared, Area.Lifter)
      .everyFeature,
    "framework.check.ToolsSuite" -> check,
    "framework.check.RetainedSuite" -> check,
    "framework.check.lint.GuideSourcesTest" ->
      check.and(Area.Framework, Area.Shared).reading(mini).files("model/SEMANTICS.md"),
    // The migration rules' suites, which rewrite source text; some read the mini feature's or the
    // shared code's files as they are.
    "temporal.migrate.BoundedIntegersTest" -> migrate,
    "temporal.migrate.BoundsInventoryTest" -> migrate.and(Area.Shared).reading(mini),
    "temporal.migrate.CapabilityNamesTest" -> migrate,
    "temporal.migrate.CentralBoundsTest" -> migrate.and(Area.Shared),
    "temporal.migrate.EntersTest" -> migrate,
    "temporal.migrate.FacetWrappingTest" -> migrate,
    "temporal.migrate.FamilyNamesTest" -> migrate.and(Area.Shared).reading(mini),
    "temporal.migrate.GapsSuite" -> migrate,
    "temporal.migrate.InputDefaultsTest" -> migrate,
    "temporal.migrate.ObservationReadTest" -> migrate,
    "temporal.migrate.QualifiedActionsTest" -> migrate,
    "temporal.migrate.QualifiedPhasesTest" -> migrate,
    "temporal.migrate.DslPackageTest" -> migrate.and(Area.Shared).reading(mini),
    "temporal.migrate.QueueAliasTest" -> migrate.reading(mini)
  )
