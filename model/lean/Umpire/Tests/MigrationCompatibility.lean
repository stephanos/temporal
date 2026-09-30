import Umpire.Scenario.Tests.Fixtures
import Umpire.Examples.Switch
import Umpire.Examples.SwitchTests
import Umpire.Evidence.Tests.Fixtures
import Umpire.Search.Tests.Fixtures
import Umpire.Property.Tests.Fixtures
import Umpire.Query.Tests.Fixtures
import Umpire.Model.Tests.Fixtures

/-!
Executable compatibility matrix for the domain-neutral Switch migration.

This cross-layer fixture intentionally lives outside `Umpire.Model.*`: it exercises the checked
Target through Query, Planning, and Artifact while the focused Target suite stays import-pure. The
switch is declared by the Model commands; what is exercised here is the expert route beside them --
the same kernel composed again through `DraftModel.make`, relocated, varied and broken on purpose --
so that the two routes keep answering alike.
-/

namespace Umpire.Tests.MigrationCompatibility

open Umpire
open Umpire.Examples.Switch

#check (Umpire.Examples.Switch.source : SourceLocation)
#check (Umpire.Examples.Switch.definitions : List DefinitionMetadata)
#check (Umpire.Examples.Switch.target : QueryModel Umpire.Examples.Switch.LawStatement)

example : [
    Umpire.ModelTests.source "Parameterized/TargetFixture.lean",
    Umpire.ScenarioTests.source,
    Umpire.PropertyTests.source,
    Umpire.QueryTests.source,
    Umpire.SearchTests.source,
    Umpire.EvidenceTests.source
  ] = [
    { path := "Parameterized/TargetFixture.lean", line := 1, column := 1,
      provenance := "lean-test" },
    { path := "Umpire/Scenario/Tests.lean", line := 1, column := 1,
      provenance := "lean-test" },
    { path := "Umpire/Property/Tests.lean", line := 1, column := 1,
      provenance := "lean-test" },
    { path := "Umpire/Query/Tests.lean", line := 1, column := 1,
      provenance := "lean-test" },
    { path := "Umpire/Planning/Tests.lean", line := 1, column := 1,
      provenance := "lean-test" },
    { path := "Umpire/Observation/Tests/Fixtures.lean", line := 1, column := 1,
      provenance := "lean-test" }
  ] := by
  native_decide

example : [
    Umpire.ModelTests.metadata "fixture.action.default" .action,
    Umpire.ModelTests.metadata "fixture.law.explicit" .law "explicit-contract/v2",
    Umpire.ScenarioTests.metadata "fixture.behavior.state" .state,
    Umpire.PropertyTests.metadata "fixture.property.observation" .fact,
    Umpire.QueryTests.metadata (DefinitionId.of "fixture.query.target") .target
      "query-target/v1",
    Umpire.SearchTests.metadata (DefinitionId.of "fixture.planning.kernel") .machine
      "planning-kernel/v1",
    Umpire.EvidenceTests.metadata "fixture.observation.mapping" .fact
  ] = [
    { id := DefinitionId.of "fixture.action.default", kind := .action,
      source := {
        path := "Umpire/ModelTests.lean"
        line := 1
        column := 1
        provenance := "lean-test"
      },
      version := 1, behaviorVersion := "contract-v1", documentation := "" },
    { id := DefinitionId.of "fixture.law.explicit", kind := .law,
      source := {
        path := "Umpire/ModelTests.lean"
        line := 1
        column := 1
        provenance := "lean-test"
      },
      version := 1, behaviorVersion := "explicit-contract/v2", documentation := "" },
    { id := DefinitionId.of "fixture.behavior.state", kind := .state,
      source := {
        path := "Umpire/Scenario/Tests.lean"
        line := 1
        column := 1
        provenance := "lean-test"
      },
      version := 1, behaviorVersion := "fixture.behavior.state/v1", documentation := "" },
    { id := DefinitionId.of "fixture.property.observation", kind := .fact,
      source := {
        path := "Umpire/Property/Tests.lean"
        line := 1
        column := 1
        provenance := "lean-test"
      },
      version := 1, behaviorVersion := "fixture.property.observation/v1", documentation := "" },
    { id := DefinitionId.of "fixture.query.target", kind := .target,
      source := {
        path := "Umpire/Query/Tests.lean"
        line := 1
        column := 1
        provenance := "lean-test"
      },
      version := 1, behaviorVersion := "query-target/v1", documentation := "query fixture" },
    { id := DefinitionId.of "fixture.planning.kernel", kind := .machine,
      source := {
        path := "Umpire/Planning/Tests.lean"
        line := 1
        column := 1
        provenance := "lean-test"
      },
      version := 1, behaviorVersion := "planning-kernel/v1",
      documentation := "planning fixture" },
    { id := DefinitionId.of "fixture.observation.mapping", kind := .fact,
      source := {
        path := "Umpire/Observation/Tests/Fixtures.lean"
        line := 1
        column := 1
        provenance := "lean-test"
      },
      version := 1, behaviorVersion := "fixture.observation.mapping/v1", documentation := "" }
  ] := by
  native_decide

/-- The domain-neutral part of the closed migration inventory. -/
def compatibilityFamilies : List String := ["switch"]

private def occurrenceAt
    (definitionId owner : DefinitionId)
    (role : SourceRefRole)
    (line column : Nat) : SourceRef := {
  id := {
    sourcePath := "Umpire/Tests/MigrationCompatibility.lean"
    line
    column
    endLine := line
    endColumn := column + 8
    localOrdinal := 0
  }
  definitionId
  path := { role, owner }
}

private def authoringAt (line column : Nat) : DraftModel LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  targetAuthoring.withOccurrences [occurrenceAt targetId targetId .modelSpec line column]

private def checkedSummary
    (result : Except LocatedError (QueryModel LawStatement)) :
    Option (String × BehaviorFingerprint × Option (List ModelValue)) :=
  result.toOption.map fun checked =>
    (checked.canonicalMetadata, checked.behaviorFingerprint,
      match checked.planning with
      | .unavailable => none
      | .available capability => some capability.actions)

/-! Moving a compiler-only occurrence cannot change any checked semantic product. -/
example : [
    checkedSummary (checkModel (authoringAt 12 3)),
    checkedSummary (checkModel (authoringAt 420 19))
  ] == [
    some (canonicalCheckedModelJson target, target.behaviorFingerprint,
      some [flipAction]),
    some (canonicalCheckedModelJson target, target.behaviorFingerprint,
      some [flipAction])
  ] := by
  native_decide

example : target.source = source ∧
    (checkModel (authoringAt 420 19)).toOption.map CheckedModel.source = some source := by
  native_decide

private def earlyTarget : QueryModel LawStatement :=
  model (authoringAt 12 3)

private def relocatedTarget : QueryModel LawStatement :=
  model (authoringAt 420 19)

private def exactActionDeclaration : Query := {
  id := exactActionQueryId
  source
  target := targetId
  form := .find flipProperty
  behavior := exactActionBehavior
  limits
  policy := shortestPolicy
}

/-! Query canonical bytes depend on stable target semantics, not elaboration layout. -/
example : [
    (Query.check (.ofTarget earlyTarget) exactActionDeclaration).toOption.map
      canonicalQueryJson,
    (Query.check (.ofTarget relocatedTarget) exactActionDeclaration).toOption.map
      canonicalQueryJson
  ] = [some (canonicalQueryJson exactActionQuery), some (canonicalQueryJson exactActionQuery)] := by
  native_decide

/-- The exact-action Query's shape, admitted against each relocated target the way the switch admits
it against its own: the search view is the target's finite planning, read at admission. -/
private def exactActionShape : Query.Shape := {
  id := exactActionQueryId
  source
  target := targetId
  form := .find
  limits
  policy := shortestPolicy
}

private def earlyRun : Except KnownGapError PlanResult :=
  ((Search.admit earlyTarget authoredProperty (some exactActionBehaviorDeclaration)
    exactActionShape).toOption.get (by native_decide)).search

private def relocatedRun : Except KnownGapError PlanResult :=
  ((Search.admit relocatedTarget authoredProperty (some exactActionBehaviorDeclaration)
    exactActionShape).toOption.get (by native_decide)).search

private def expectedSwitchArtifactJson : String :=
  include_str "../Examples/Fixtures/SwitchCompiledArtifact.json"

/-! Planning both layouts preserves the committed canonical artifact bytes. -/
example : [
    earlyRun.toOption.bind (fun run => run.artifact.map canonicalPlanBytes),
    relocatedRun.toOption.bind (fun run => run.artifact.map canonicalPlanBytes)
  ] = [some expectedSwitchArtifactJson, some expectedSwitchArtifactJson] := by
  native_decide

/-! The expert route preserves the exact planner result as well as the golden Artifact bytes. -/
example : earlyRun.toOption.map PlanResult.result = some exactActionRun.result ∧
    relocatedRun.toOption.map PlanResult.result = some exactActionRun.result := by
  native_decide

private def wrongKindDefinition : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  modelSpec with requiredCapabilities := [flipActionId]
}

private def wrongKindAuthoringAt (line column : Nat) : DraftModel LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make wrongKindDefinition modelProviders (occurrences := [
    occurrenceAt flipActionId targetId .capabilityRequirement line column
  ])

private def diagnosticSummary
    (result : Except LocatedError (QueryModel LawStatement)) :
    Option (DefinitionErrorKind × SourceRefRole × String × Nat × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic => some
      (diagnostic.error.kind, diagnostic.path.role, diagnostic.offending.sourcePath,
        diagnostic.offending.line, diagnostic.offending.column)

/-! The diagnostic follows the authored occurrence while stable provenance remains unchanged. -/
example : [
    diagnosticSummary (checkModel (wrongKindAuthoringAt 31 4)),
    diagnosticSummary (checkModel (wrongKindAuthoringAt 503 27))
  ] = [
    some (.wrongKind, .capabilityRequirement,
      "Umpire/Tests/MigrationCompatibility.lean", 31, 4),
    some (.wrongKind, .capabilityRequirement,
      "Umpire/Tests/MigrationCompatibility.lean", 503, 27)
  ] := by
  native_decide

example : wrongKindDefinition.source = source := by
  native_decide

private def expertModelSpec : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := modelSpec.id
  source := modelSpec.source
  definitions := modelSpec.definitions
  requiredCapabilities := modelSpec.requiredCapabilities
  providers := [switchProvider]
  connectors := []
  resolvedSetups := modelSpec.resolvedSetups
  machine := modelSpec.machine
}

private def invalidDefinitionIdMetadata : DefinitionMetadata := {
  id := DefinitionId.of "action"
  kind := .action
  source
  behaviorVersion := "invalid-definition-id/v1"
}

private def invalidDefinitionIdTarget : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertModelSpec with
  definitions := invalidDefinitionIdMetadata :: expertModelSpec.definitions
}

private def missingProviderDeclaration : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertModelSpec with providers := []
}

private def providerWithoutLaw : Provider LawStatement := {
  switchProvider with lawProofs := []
}

private def missingLawDeclaration : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertModelSpec with providers := [providerWithoutLaw]
}

private def incompleteKernelDeclaration : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertModelSpec with
  machine := .incomplete machine.metadata
    [DefinitionId.of "umpire.kernel-proof.step-complete"]
}

private def targetErrorKind
    (result : Except DefinitionError (QueryModel LawStatement)) : Option DefinitionErrorKind :=
  match result with
  | .ok _ => none
  | .error failure => some failure.kind

/-! Target-owned invalid Definition IDs, providers, laws, and kernel availability stay typed at Target. -/
example : [
    targetErrorKind ((checkModel (DraftModel.make invalidDefinitionIdTarget) |>.mapError LocatedError.error)),
    targetErrorKind ((checkModel (DraftModel.make missingProviderDeclaration) |>.mapError LocatedError.error)),
    targetErrorKind ((checkModel (DraftModel.make missingLawDeclaration) |>.mapError LocatedError.error)),
    targetErrorKind ((checkModel (DraftModel.make incompleteKernelDeclaration) |>.mapError LocatedError.error))
  ] = [some .invalidDefinitionId, some .missingProvider, some .missingLaw, some .incompleteMachine] := by
  native_decide

private def queryErrorKind
    (result : Except QueryError (CheckedQuery LawStatement)) : Option QueryErrorKind :=
  match result with
  | .ok _ => none
  | .error failure => some failure.kind

private def invalidBounds : Limits := { limits with steps := { value := 0, unit := .steps } }

private def invalidBoundDeclaration : Query := {
  exactActionDeclaration with limits := invalidBounds
}

private def exhaustiveDeclaration : Query := {
  exactActionDeclaration with policy := { shortestPolicy with strategy := .exhaustive }
}

private def noFinitePlanningTarget : QueryModel LawStatement :=
  model targetAuthoring.withoutPlanning

private def mismatchedTrace : Scenario.Trace := {
  setup := switchSetup
  trace := {
    initialState := offState
    steps := [{
      selectedAction := flipAction
      outcome := appliedOutcome
      state := offState
      facts := [powerOffObservation]
    }]
  }
}

private def mismatchedBehavior : CheckedScenario := {
  exactTraceBehavior with
  traceExactly := some mismatchedTrace
  behaviorFingerprint := behaviorFingerprintOf "switch-behavior-target-kernel-mismatch/v1"
}

private def mismatchedDeclaration : Query := {
  exactActionDeclaration with behavior := mismatchedBehavior
}

/-! Query owns limits, finite-completeness, and exact-trace/kernel mismatch failures. -/
example : [
    queryErrorKind (Query.check (.ofTarget target) invalidBoundDeclaration),
    queryErrorKind (Query.check (.ofTarget noFinitePlanningTarget) exhaustiveDeclaration),
    queryErrorKind (Query.check (.ofTarget target) mismatchedDeclaration)
  ] = [some .invalidLimit, some .missingFiniteCompleteness, some .targetKernelMismatch] := by
  native_decide

/-!
Switch's imported golden tests pin its exact Query and artifact bytes and its planner outcomes.
This inventory assertion makes omission of the domain-neutral family an executable failure.
-/
example : compatibilityFamilies = ["switch"] := by
  rfl

end Umpire.Tests.MigrationCompatibility
