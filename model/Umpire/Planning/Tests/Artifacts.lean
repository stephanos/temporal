import Umpire.Planning.Tests.Fixtures

/-! Inspectability, optional occurrences, byte stability, and semantic identity checks. -/

namespace Umpire.PlanningTests

open Umpire

def witnessSpec (seed : Nat := 17) : Option ExperimentSpec :=
  (run 2 (.witness property) .shortest 10 seed false).artifact

def incidentalWitnessSpec : Option ExperimentSpec :=
  let query := checkedQuery 2 (.witness property) .shortest 10 17 false
  let incidental : CheckedQuery (fun _ => True) := {
    query with
    documentation := "changed query documentation"
    behavior := { query.behavior with documentation := "changed behavior documentation" }
    form := .witness { property with documentation := "changed property documentation" }
  }
  (plan incidental (incrementalKernel 2)).artifact

def selectedArtifactIsInspectable : Bool :=
  match witnessSpec with
  | none => false
  | some spec =>
      spec.plan.initialState == initial &&
      spec.plan.requestedActions == [requestValue] &&
      spec.plan.modelOutcomes == [acceptedValue] &&
      spec.plan.linearExtension.map PlannedOccurrence.definitionId == [occurrence] &&
      spec.plan.linearExtension.map PlannedOccurrence.actionDefinitionId == [request] &&
      spec.plan.linearExtension.length == spec.plan.requestedActions.length &&
      spec.plan.bindings == setup &&
      spec.plan.symbolicRoles == [] &&
      spec.plan.expandedLimits == limits &&
      spec.plan.selectionReason == .satisfyingWitness &&
      spec.plan.checkpoints.length == 1 &&
      spec.plan.knownGaps.toList == canonicalPlannerKnownGaps.toList &&
      spec.properties.map PortableProperty.definitionId == [property.id]

/-! A selected trace is compiled into an inspectable plan that separates requests from outcomes. -/
example : selectedArtifactIsInspectable := by
  native_decide

def optionalBehavior : CheckedBehavior := {
  behavior with
  requiredOccurrences := []
  behaviorFingerprint := behaviorFingerprintOf "behavior/optional-v1"
}

/-! The linear extension contains every selected action, including optional occurrences. -/
example :
    ((run 2 (.select [property]) .shortest 10 17 false optionalBehavior).artifact.map fun spec =>
      (spec.plan.linearExtension.length,
        spec.plan.linearExtension.map PlannedOccurrence.actionDefinitionId)) =
      some (1, [request]) := by
  native_decide

/-! Independent planning and rendering of semantically identical checked inputs is byte-identical. -/
example :
    witnessSpec.map canonicalExperimentSpecJson =
      incidentalWitnessSpec.map canonicalExperimentSpecJson := by
  native_decide

/-! The empty checked-intent facade preserves ordinary planning bytes exactly. -/
example :
    let query := checkedQuery 2 (.witness property) .shortest 10 17 false
    let withIntent := planWithArtifactIntent query (incrementalKernel 2) (.empty query)
    withIntent.toOption.bind (fun run => run.artifact.map canonicalExperimentSpecBytes) =
      witnessSpec.map canonicalExperimentSpecBytes := by
  native_decide

/-! Ordinary planning continues to leave every reserved intent array empty. -/
example : witnessSpec.map (fun spec =>
    (spec.plan.selectedChoices, spec.plan.selectedVariants, spec.plan.requestedFaults)) =
    some ([], [], []) := by
  native_decide

/-! A meaning-bearing Query input is part of the exact Artifact Checksum. -/
example :
    witnessSpec.map ExperimentSpec.artifactChecksum !=
      (witnessSpec 18).map ExperimentSpec.artifactChecksum := by
  native_decide

private theorem witnessSpec_isSome : witnessSpec.isSome = true := by
  native_decide

private def checksumSpec : ExperimentSpec :=
  witnessSpec.get witnessSpec_isSome

/-! Planning Artifacts expose only checked Known Gaps. -/
#check (DrivePlan.knownGaps : DrivePlan → KnownGapSet)

private def optionalKnownGap (subject : Option DefinitionId) (detail : Option String) : KnownGap := {
  kind := .input
  code := id "planner.known-gap.optional"
  subject
  detail
}

/-! Known Gap subject and detail preserve independent absent and present encodings. -/
example : [
    canonicalKnownGapJson (optionalKnownGap none none),
    canonicalKnownGapJson (optionalKnownGap (some targetId) none),
    canonicalKnownGapJson (optionalKnownGap none (some "line\n\"quoted\"")),
    canonicalKnownGapJson (optionalKnownGap (some targetId) (some "complete"))
  ] = [
    "{\"kind\":\"input\",\"code\":\"planner.known-gap.optional\"," ++
      "\"subject\":null,\"detail\":null}",
    "{\"kind\":\"input\",\"code\":\"planner.known-gap.optional\"," ++
      "\"subject\":\"planner.target.fixture\",\"detail\":null}",
    "{\"kind\":\"input\",\"code\":\"planner.known-gap.optional\"," ++
      "\"subject\":null,\"detail\":\"line\\n\\\"quoted\\\"\"}",
    "{\"kind\":\"input\",\"code\":\"planner.known-gap.optional\"," ++
      "\"subject\":\"planner.target.fixture\",\"detail\":\"complete\"}"
  ] := by
  native_decide

private def authoredDefinitionIdLine (plan : DrivePlan) : Option String :=
  (canonicalDrivePlanJson plan).splitOn "\n" |>.find? fun line =>
    line.contains "\"authoredDefinitionId\""

/-! Planned occurrences preserve exact absent and present authored Definition ID fields. -/
example :
    let present := checksumSpec.plan
    let absent := {
      present with
      linearExtension := present.linearExtension.map fun occurrence =>
        { occurrence with authoredDefinitionId := none }
    }
    [authoredDefinitionIdLine present, authoredDefinitionIdLine absent] = [
      some "      \"authoredDefinitionId\": \"planner.occurrence.request\"",
      some "      \"authoredDefinitionId\": null"
    ] ∧ absent.expectedArtifactChecksum != present.artifactChecksum := by
  native_decide

private def mutationId : DefinitionId :=
  id "planner.mutation.value"

private def mutationValue : ModelValue :=
  value mutationId "mutated"

private def mutationFingerprint : BehaviorFingerprint :=
  behaviorFingerprintOf "planner/mutation"

private def mutationPrecondition : SetupConstraint := {
  id := id "planner.setup.mutated"
  relation := .equal
  left := .value initial
  right := .value completed
}

private def mutationRole : ResourceRole := {
  id := id "planner.role.mutated"
  valueKind := .state
}

private def drivePlanContentMutations (plan : DrivePlan) : List DrivePlan := [
  { plan with formatVersion := "umpire-drive-plan/unsupported" },
  { plan with queryDefinitionId := mutationId },
  { plan with queryBehaviorFingerprint := mutationFingerprint },
  { plan with behaviorDefinitionId := mutationId },
  { plan with behaviorFingerprint := mutationFingerprint },
  { plan with targetDefinitionId := mutationId },
  { plan with targetBehaviorFingerprint := mutationFingerprint },
  { plan with kernelDefinitionId := mutationId },
  { plan with kernelBehaviorFingerprint := mutationFingerprint },
  { plan with bindings := [] },
  { plan with symbolicRoles := [mutationRole] },
  { plan with modelPreconditions := [mutationPrecondition] },
  { plan with initialState := mutationValue },
  { plan with requestedActions := [mutationValue] },
  { plan with modelOutcomes := [mutationValue] },
  { plan with resultingStates := [mutationValue] },
  { plan with linearExtension := [] },
  { plan with selectedChoices := [mutationValue] },
  { plan with selectedVariants := [mutationValue] },
  { plan with requestedFaults := [mutationValue] },
  { plan with capabilityRequirementDefinitionIds := [mutationId] },
  { plan with expandedLimits := {
      plan.expandedLimits with search := { value := 11, unit := .candidateEvaluations }
    } },
  { plan with checkpoints := [] },
  { plan with selectionReason := .behaviorSelection },
  { plan with explored := { plan.explored with transitions := plan.explored.transitions + 1 } },
  { plan with knownGaps := KnownGapSet.empty },
  { plan with provenance := { plan.provenance with sourceDefinitionIds := [] } },
  { plan with provenance := { plan.provenance with sourceLocations := [] } }
]

private def firstPlannerKnownGap : KnownGap := {
  kind := .input
  code := id "umpire.known-gap.execution-evidence"
}

private def checkedKnownGaps (gaps : List KnownGap) : KnownGapSet :=
  (KnownGapSet.ofUnordered gaps).toOption.getD KnownGapSet.empty

private def remainingPlannerKnownGaps : List KnownGap :=
  canonicalPlannerKnownGaps.toList.drop 1

private def knownGapRowMutations (plan : DrivePlan) : List DrivePlan := [
  { plan with knownGaps := (checkedKnownGaps
      (({ firstPlannerKnownGap with kind := .capabilityContract } : KnownGap) ::
        remainingPlannerKnownGaps)) },
  { plan with knownGaps := (checkedKnownGaps
      (({ firstPlannerKnownGap with code := id "umpire.known-gap.changed" } : KnownGap) ::
        remainingPlannerKnownGaps)) },
  { plan with knownGaps := (checkedKnownGaps
      (({ firstPlannerKnownGap with subject := some mutationId } : KnownGap) ::
        remainingPlannerKnownGaps)) },
  { plan with knownGaps := (checkedKnownGaps
      (({ firstPlannerKnownGap with detail := some "changed" } : KnownGap) ::
        remainingPlannerKnownGaps)) }
]

private def experimentSpecContentMutations (spec : ExperimentSpec) : List ExperimentSpec := [
  { spec with formatVersion := "umpire-experiment/unsupported" },
  { spec with queryBehaviorFingerprint := mutationFingerprint },
  { spec with plan := { spec.plan with requestedActions := [mutationValue] } },
  { spec with properties := [] },
  { spec with observationRequirementDefinitionIds := [mutationId] },
  { spec with provenance := { spec.provenance with sourceDefinitionIds := [] } },
  { spec with provenance := { spec.provenance with sourceLocations := [] } }
]

/-! Every persisted DrivePlan category participates in its Artifact Checksum. -/
example :
    (drivePlanContentMutations checksumSpec.plan).all fun mutated =>
      mutated.expectedArtifactChecksum != checksumSpec.plan.artifactChecksum := by
  native_decide

/-! Every field of a complete Known Gap row participates in the owning Artifact Checksum. -/
example :
    (knownGapRowMutations checksumSpec.plan).all fun mutated =>
      mutated.expectedArtifactChecksum != checksumSpec.plan.artifactChecksum := by
  native_decide

/-! Every persisted ExperimentSpec category, including its complete nested plan, participates. -/
example :
    (experimentSpecContentMutations checksumSpec).all fun mutated =>
      mutated.expectedArtifactChecksum != checksumSpec.artifactChecksum := by
  native_decide

/-! Only an Artifact's own checksum field is excluded from its checksum input. -/
example :
    let changedPlanChecksum := {
      checksumSpec.plan with artifactChecksum := drivePlanChecksumOf "changed"
    }
    let changedSpecChecksum := {
      checksumSpec with artifactChecksum := experimentSpecChecksumOf "changed"
    }
    changedPlanChecksum.expectedArtifactChecksum = checksumSpec.plan.expectedArtifactChecksum ∧
      changedSpecChecksum.expectedArtifactChecksum = checksumSpec.expectedArtifactChecksum := by
  native_decide

/-! Both stored Artifact Checksums are reproducible and valid for their complete canonical content. -/
example :
    checksumSpec.plan.hasValidArtifactChecksum ∧ checksumSpec.hasValidArtifactChecksum ∧
      checksumSpec.plan.expectedArtifactChecksum = checksumSpec.plan.expectedArtifactChecksum ∧
      checksumSpec.expectedArtifactChecksum = checksumSpec.expectedArtifactChecksum := by
  native_decide

/-! DrivePlan and ExperimentSpec use distinct checksum domains for identical canonical content. -/
example :
    (drivePlanChecksumOf "same-content").render !=
      (experimentSpecChecksumOf "same-content").render := by
  native_decide

/-! Persisted canonical bytes add exactly one LF to an LF-free canonical JSON object. -/
example :
    canonicalDrivePlanBytes checksumSpec.plan = canonicalDrivePlanJson checksumSpec.plan ++ "\n" ∧
      canonicalExperimentSpecBytes checksumSpec = canonicalExperimentSpecJson checksumSpec ++ "\n" ∧
      !(canonicalDrivePlanJson checksumSpec.plan).endsWith "\n" ∧
      !(canonicalExperimentSpecJson checksumSpec).endsWith "\n" ∧
      !(canonicalDrivePlanBytes checksumSpec.plan).endsWith "\n\n" ∧
      !(canonicalExperimentSpecBytes checksumSpec).endsWith "\n\n" := by
  native_decide

/-! Lean and Go share unrestricted canonical base-10 naturals beyond machine-word range. -/
example :
    toString (18446744073709551616 : Nat) ++ "\n" =
      include_str "Fixtures/NaturalAboveUint64.json" := by
  native_decide

end Umpire.PlanningTests
