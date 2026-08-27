import Lean.Data.Json
import Umpire.Planning.Types

namespace Umpire

/-! Portable, environment-independent products of pure model planning. -/

def SelectionReason.name : SelectionReason → String
  | .satisfyingWitness => "satisfying-witness"
  | .violatingCounterexample => "violating-counterexample"
  | .behaviorSelection => "behavior-selection"

structure ObservationCheckpoint where
  transition : Nat
  observations : List ModelValue
  deriving BEq, DecidableEq, Repr

structure PlannedOccurrence where
  definitionId : DefinitionId
  actionDefinitionId : DefinitionId
  position : Nat
  authoredDefinitionId : Option DefinitionId
  deriving BEq, DecidableEq, Repr

structure PortableProperty where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  requirementDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

structure ArtifactProvenance where
  sourceDefinitionIds : List DefinitionId
  sourceLocations : List SourceLocation
  deriving BEq, DecidableEq, Repr

/--
The runtime-facing request remains explicit model data. Requested actions and model-owned outcomes
are deliberately separate, and no field claims that runtime execution or evidence occurred.
-/
structure DrivePlan where
  formatVersion : String
  artifactChecksum : ArtifactChecksum
  queryDefinitionId : DefinitionId
  queryBehaviorFingerprint : BehaviorFingerprint
  behaviorDefinitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  targetDefinitionId : DefinitionId
  targetBehaviorFingerprint : BehaviorFingerprint
  kernelDefinitionId : DefinitionId
  kernelBehaviorFingerprint : BehaviorFingerprint
  bindings : List RoleBinding
  symbolicRoles : List ResourceRole
  modelPreconditions : List SetupConstraint
  initialState : ModelValue
  requestedActions : List ModelValue
  modelOutcomes : List ModelValue
  resultingStates : List ModelValue
  linearExtension : List PlannedOccurrence
  selectedChoices : List ModelValue
  selectedVariants : List ModelValue
  requestedFaults : List ModelValue
  capabilityRequirementDefinitionIds : List DefinitionId
  expandedLimits : QueryLimits
  checkpoints : List ObservationCheckpoint
  selectionReason : SelectionReason
  explored : ExploredCounts
  knownGaps : List KnownGap
  provenance : ArtifactProvenance
  deriving BEq, DecidableEq, Repr

/-- The portable envelope consumed by later execution, checking, replay, and generation work. -/
structure ExperimentSpec where
  formatVersion : String
  artifactChecksum : ArtifactChecksum
  queryBehaviorFingerprint : BehaviorFingerprint
  plan : DrivePlan
  properties : List PortableProperty
  observationRequirementDefinitionIds : List DefinitionId
  provenance : ArtifactProvenance
  deriving BEq, DecidableEq, Repr

def canonicalPlannerKnownGaps : List KnownGap := [
  { kind := .input, code := DefinitionId.of "umpire.known-gap.execution-evidence" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.artifact-migrations" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.artifact-reading" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.evidence-evaluation" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.runtime-scheduler-order" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.runtime-storage-order" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.runtime-transport-order" },
  { kind := .claim, code := DefinitionId.of "umpire.known-gap.promotion" }
]

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

private def canonicalValues (values : List ModelValue) : List ModelValue :=
  values.mergeSort valueLe |>.eraseDups

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
  drivePlanChecksumOf (drivePlanContentJson plan)

def DrivePlan.hasValidArtifactChecksum (plan : DrivePlan) : Bool :=
  plan.artifactChecksum == plan.expectedArtifactChecksum

def canonicalDrivePlanJson (plan : DrivePlan) : String :=
  let content := drivePlanContentJson plan
  (content.dropEnd 1).toString ++
    ",\"artifactChecksum\":" ++ quote plan.artifactChecksum.render ++ "}"

def canonicalDrivePlanBytes (plan : DrivePlan) : String :=
  canonicalDrivePlanJson plan ++ "\n"

private def experimentSpecContentJson (spec : ExperimentSpec) : String :=
  "{\"formatVersion\":" ++ quote spec.formatVersion ++
    ",\"queryBehaviorFingerprint\":" ++ quote spec.queryBehaviorFingerprint.render ++
    ",\"plan\":" ++ canonicalDrivePlanJson spec.plan ++
    ",\"properties\":" ++ array (spec.properties.mergeSort propertyLe |>.map propertyJson) ++
    ",\"observationRequirementDefinitionIds\":" ++
      array (canonicalIds spec.observationRequirementDefinitionIds |>.map
        (quote ∘ DefinitionId.value)) ++
    ",\"provenance\":" ++ provenanceJson spec.provenance ++ "}"

def ExperimentSpec.expectedArtifactChecksum (spec : ExperimentSpec) : ArtifactChecksum :=
  experimentSpecChecksumOf (experimentSpecContentJson spec)

def ExperimentSpec.hasValidArtifactChecksum (spec : ExperimentSpec) : Bool :=
  spec.artifactChecksum == spec.expectedArtifactChecksum

def canonicalExperimentSpecJson (spec : ExperimentSpec) : String :=
  let content := experimentSpecContentJson spec
  (content.dropEnd 1).toString ++
    ",\"artifactChecksum\":" ++ quote spec.artifactChecksum.render ++ "}"

def canonicalExperimentSpecBytes (spec : ExperimentSpec) : String :=
  canonicalExperimentSpecJson spec ++ "\n"

private def plannedOccurrence
    (behavior : CheckedBehavior)
    (index : Nat)
    (action : ModelValue)
    (authored : Option NamedOccurrence) : PlannedOccurrence :=
  let definitionId := authored.map NamedOccurrence.id |>.getD
    (DefinitionId.of (behavior.id.value ++ ".selected-occurrence-" ++ toString (index + 1)))
  {
    definitionId
    actionDefinitionId := action.definitionId
    position := index + 1
    authoredDefinitionId := authored.map NamedOccurrence.id
  }

private def propertyReference (property : CheckedProperty) : PortableProperty := {
  definitionId := property.id
  behaviorFingerprint := property.behaviorFingerprint
  requirementDefinitionIds := canonicalIds property.requires
}

private def propertyObservationRequirements (property : CheckedProperty) : List DefinitionId :=
  property.access.meanings.filterMap fun meaning =>
    if meaning.kind == .observation || meaning.kind == .relation then
      some meaning.definitionId
    else
      none

private def artifactProvenance (query : CheckedQuery LawStatement) : ArtifactProvenance := {
  sourceDefinitionIds := canonicalIds ([
    query.id,
    query.behavior.id,
    query.target.id,
    query.target.kernel.metadata.id
  ] ++ query.form.properties.map CheckedProperty.id)
  sourceLocations := (query.source :: query.behavior.source :: query.target.source ::
    query.target.kernel.metadata.source :: query.form.properties.map CheckedProperty.source)
      |>.mergeSort sourceLe |>.eraseDups
}

/-- Artifact construction is a single deep seam over a selected, kernel-produced trace. -/
def artifactOfSelection
    (query : CheckedQuery LawStatement)
    (trace : BehaviorTrace)
    (reason : SelectionReason)
    (explored : ExploredCounts) : ExperimentSpec :=
  let actions := trace.trace.steps.map fun step => step.selectedAction
  let outcomes := trace.trace.steps.map fun step => step.modelOutcome
  let states := trace.trace.steps.map fun step => step.resultingState
  let slots := query.behavior.assignOccurrences (actions.map ModelValue.definitionId) |>.getD
      (actions.map fun _ => none)
  let extension := actions.zip slots |>.zipIdx |>.map fun ((action, authored), index) =>
    plannedOccurrence query.behavior index action authored
  let checkpoints := trace.trace.steps.zipIdx.map fun (step, index) => {
    transition := index + 1
    observations := step.observations
  }
  let provenance := artifactProvenance query
  let planWithoutChecksum : DrivePlan := {
    formatVersion := "umpire-drive-plan/v2"
    artifactChecksum := drivePlanChecksumOf ""
    queryDefinitionId := query.id
    queryBehaviorFingerprint := query.behaviorFingerprint
    behaviorDefinitionId := query.behavior.id
    behaviorFingerprint := query.behavior.behaviorFingerprint
    targetDefinitionId := query.target.id
    targetBehaviorFingerprint := query.target.behaviorFingerprint
    kernelDefinitionId := query.target.kernel.metadata.id
    kernelBehaviorFingerprint := query.target.behaviorFingerprint
    bindings := trace.setup.mergeSort bindingLe
    symbolicRoles := query.behavior.roles.filter fun role =>
      !(trace.setup.any fun binding => binding.role == role.id)
    modelPreconditions := query.behavior.setup
    initialState := trace.trace.initialState
    requestedActions := actions
    modelOutcomes := outcomes
    resultingStates := states
    linearExtension := extension
    selectedChoices := []
    selectedVariants := []
    requestedFaults := []
    capabilityRequirementDefinitionIds := canonicalIds (query.target.requiredCapabilities ++
      query.behavior.requires ++
      query.form.properties.flatMap CheckedProperty.requires)
    expandedLimits := query.limits
    checkpoints
    selectionReason := reason
    explored
    knownGaps := canonicalPlannerKnownGaps
    provenance
  }
  let plan := {
    planWithoutChecksum with
    artifactChecksum := planWithoutChecksum.expectedArtifactChecksum
  }
  let properties := query.form.properties.map propertyReference |>.mergeSort propertyLe
  let observationRequirementDefinitionIds := canonicalIds
    (query.form.properties.flatMap propertyObservationRequirements)
  let specWithoutChecksum : ExperimentSpec := {
    formatVersion := "umpire-experiment/v2"
    artifactChecksum := experimentSpecChecksumOf ""
    queryBehaviorFingerprint := query.behaviorFingerprint
    plan
    properties
    observationRequirementDefinitionIds
    provenance
  }
  {
    specWithoutChecksum with
    artifactChecksum := specWithoutChecksum.expectedArtifactChecksum
  }

end Umpire
