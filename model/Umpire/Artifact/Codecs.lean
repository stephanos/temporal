import Lean.Data.Json
import Umpire.Artifact.Types
import Umpire.Json

namespace Umpire

/-! Canonical v2 bytes and Artifact Checksum derivation for the retained planning Artifacts. -/

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def valueLe (left right : ModelValue) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId && decide (left.value ≤ right.value))

private def bindingLe (left right : RoleBinding) : Bool :=
  decide (left.role.value < right.role.value) ||
    (left.role == right.role && valueLe left.value right.value)

private def sourceLe (left right : SourceLocation) : Bool :=
  decide (left.path < right.path) ||
    (left.path == right.path && decide (left.line < right.line)) ||
    (left.path == right.path && left.line == right.line && decide (left.column < right.column)) ||
    (left.path == right.path && left.line == right.line && left.column == right.column &&
      decide (left.provenance ≤ right.provenance))

private def propertyLe (left right : PortableProperty) : Bool :=
  decide (left.definitionId.value ≤ right.definitionId.value)

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

private def valueJson (value : ModelValue) : String :=
  "{\"definitionId\":" ++ quote value.definitionId.value ++
    ",\"value\":" ++ quote value.value ++ "}"

private def roleJson (role : ResourceRole) : String :=
  "{\"definitionId\":" ++ quote role.id.value ++
    ",\"valueKind\":" ++ quote role.valueKind.name ++ "}"

private def bindingJson (binding : RoleBinding) : String :=
  "{\"roleDefinitionId\":" ++ quote binding.role.value ++
    ",\"value\":" ++ valueJson binding.value ++ "}"

private def operandJson : SetupOperand → String
  | .role identity =>
      "{\"kind\":\"role\",\"definitionId\":" ++ quote identity.value ++ "}"
  | .value value =>
      "{\"kind\":\"value\",\"value\":" ++ valueJson value ++ "}"

private def preconditionJson (constraint : SetupConstraint) : String :=
  "{\"definitionId\":" ++ quote constraint.id.value ++
    ",\"relation\":" ++ quote constraint.relation.name ++
    ",\"left\":" ++ operandJson constraint.left ++
    ",\"right\":" ++ operandJson constraint.right ++ "}"

private def limitJson (bound : Limit) : String :=
  "{\"value\":" ++ toString bound.value ++
    ",\"unit\":" ++ quote bound.unit.name ++ "}"

private def limitsJson (limits : QueryLimits) : String :=
  "{\"behavior\":{\"transitions\":" ++ limitJson limits.behavior.transitions ++
    ",\"selectedActions\":" ++ limitJson limits.behavior.selectedActions ++ "}" ++
    ",\"search\":{\"value\":" ++ toString limits.search.value ++
    ",\"unit\":" ++ quote limits.search.unit.name ++ "}}"

private def exploredJson (explored : ExploredCounts) : String :=
  "{\"setups\":" ++ toString explored.setups ++
    ",\"traces\":" ++ toString explored.traces ++
    ",\"transitions\":" ++ toString explored.transitions ++
    ",\"propertyEvaluations\":" ++ toString explored.propertyEvaluations ++ "}"

private def checkpointJson (checkpoint : ObservationCheckpoint) : String :=
  "{\"transition\":" ++ toString checkpoint.transition ++
    ",\"observations\":" ++ array (checkpoint.observations.map valueJson) ++ "}"

private def plannedOccurrenceJson (occurrence : PlannedOccurrence) : String :=
  "{\"definitionId\":" ++ quote occurrence.definitionId.value ++
    ",\"actionDefinitionId\":" ++ quote occurrence.actionDefinitionId.value ++
    ",\"position\":" ++ toString occurrence.position ++
    ",\"authoredDefinitionId\":" ++
      (occurrence.authoredDefinitionId.map (quote ∘ DefinitionId.value) |>.getD "null") ++ "}"

private def sourceJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def provenanceJson (provenance : ArtifactProvenance) : String :=
  "{\"sourceDefinitionIds\":" ++
      array (canonicalIds provenance.sourceDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++
    ",\"sourceLocations\":" ++
      array (provenance.sourceLocations.mergeSort sourceLe |>.eraseDups |>.map sourceJson) ++ "}"

private def propertyJson (property : PortableProperty) : String :=
  "{\"definitionId\":" ++ quote property.definitionId.value ++
    ",\"behaviorFingerprint\":" ++ quote property.behaviorFingerprint.render ++
    ",\"requirementDefinitionIds\":" ++
      array (canonicalIds property.requirementDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++ "}"

private def drivePlanContentJson (plan : DrivePlan) : String :=
  "{\"formatVersion\":" ++ quote plan.formatVersion ++
    ",\"queryDefinitionId\":" ++ quote plan.queryDefinitionId.value ++
    ",\"queryBehaviorFingerprint\":" ++ quote plan.queryBehaviorFingerprint.render ++
    ",\"behaviorDefinitionId\":" ++ quote plan.behaviorDefinitionId.value ++
    ",\"behaviorFingerprint\":" ++ quote plan.behaviorFingerprint.render ++
    ",\"targetDefinitionId\":" ++ quote plan.targetDefinitionId.value ++
    ",\"targetBehaviorFingerprint\":" ++ quote plan.targetBehaviorFingerprint.render ++
    ",\"kernelDefinitionId\":" ++ quote plan.kernelDefinitionId.value ++
    ",\"kernelBehaviorFingerprint\":" ++ quote plan.kernelBehaviorFingerprint.render ++
    ",\"bindings\":" ++ array (plan.bindings.mergeSort bindingLe |>.map bindingJson) ++
    ",\"symbolicRoles\":" ++ array (plan.symbolicRoles.map roleJson) ++
    ",\"modelPreconditions\":" ++ array (plan.modelPreconditions.map preconditionJson) ++
    ",\"initialState\":" ++ valueJson plan.initialState ++
    ",\"requestedActions\":" ++ array (plan.requestedActions.map valueJson) ++
    ",\"modelOutcomes\":" ++ array (plan.modelOutcomes.map valueJson) ++
    ",\"resultingStates\":" ++ array (plan.resultingStates.map valueJson) ++
    ",\"linearExtension\":" ++ array (plan.linearExtension.map plannedOccurrenceJson) ++
    ",\"selectedChoices\":" ++ array (plan.selectedChoices.map valueJson) ++
    ",\"selectedVariants\":" ++ array (plan.selectedVariants.map valueJson) ++
    ",\"requestedFaults\":" ++ array (plan.requestedFaults.map valueJson) ++
    ",\"capabilityRequirementDefinitionIds\":" ++
      array (canonicalIds plan.capabilityRequirementDefinitionIds |>.map
        (quote ∘ DefinitionId.value)) ++
    ",\"expandedLimits\":" ++ limitsJson plan.expandedLimits ++
    ",\"checkpoints\":" ++ array (plan.checkpoints.map checkpointJson) ++
    ",\"selectionReason\":" ++ quote plan.selectionReason.name ++
    ",\"explored\":" ++ exploredJson plan.explored ++
    ",\"knownGaps\":" ++ array (plan.knownGaps.map canonicalKnownGapJson) ++
    ",\"provenance\":" ++ provenanceJson plan.provenance ++ "}"

def DrivePlan.expectedArtifactChecksum (plan : DrivePlan) : ArtifactChecksum :=
  drivePlanChecksumOf (Json.prettyBytes (drivePlanContentJson plan))

def DrivePlan.hasValidArtifactChecksum (plan : DrivePlan) : Bool :=
  plan.artifactChecksum == plan.expectedArtifactChecksum

private def sealedDrivePlanJson (plan : DrivePlan) : String :=
  let content := drivePlanContentJson plan
  (content.dropEnd 1).toString ++
    ",\"artifactChecksum\":" ++ quote plan.artifactChecksum.render ++ "}"

def canonicalDrivePlanJson (plan : DrivePlan) : String :=
  Json.pretty (sealedDrivePlanJson plan)

def canonicalDrivePlanBytes (plan : DrivePlan) : String :=
  canonicalDrivePlanJson plan ++ "\n"

private def experimentSpecContentJson (spec : ExperimentSpec) : String :=
    "{\"formatVersion\":" ++ quote spec.formatVersion ++
    ",\"queryBehaviorFingerprint\":" ++ quote spec.queryBehaviorFingerprint.render ++
    ",\"plan\":" ++ sealedDrivePlanJson spec.plan ++
    ",\"properties\":" ++ array (spec.properties.mergeSort propertyLe |>.map propertyJson) ++
    ",\"observationRequirementDefinitionIds\":" ++
      array (canonicalIds spec.observationRequirementDefinitionIds |>.map
        (quote ∘ DefinitionId.value)) ++
    ",\"provenance\":" ++ provenanceJson spec.provenance ++ "}"

def ExperimentSpec.expectedArtifactChecksum (spec : ExperimentSpec) : ArtifactChecksum :=
  experimentSpecChecksumOf (Json.prettyBytes (experimentSpecContentJson spec))

def ExperimentSpec.hasValidArtifactChecksum (spec : ExperimentSpec) : Bool :=
  spec.artifactChecksum == spec.expectedArtifactChecksum

private def sealedExperimentSpecJson (spec : ExperimentSpec) : String :=
  let content := experimentSpecContentJson spec
  (content.dropEnd 1).toString ++
    ",\"artifactChecksum\":" ++ quote spec.artifactChecksum.render ++ "}"

def canonicalExperimentSpecJson (spec : ExperimentSpec) : String :=
  Json.pretty (sealedExperimentSpecJson spec)

def canonicalExperimentSpecBytes (spec : ExperimentSpec) : String :=
  canonicalExperimentSpecJson spec ++ "\n"

end Umpire
