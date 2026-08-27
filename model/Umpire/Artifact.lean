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
  action : DefinitionId
  position : Nat
  authoredDefinitionId : Option DefinitionId
  deriving BEq, DecidableEq, Repr

structure PortableProperty where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  requirements : List DefinitionId
  deriving BEq, DecidableEq, Repr

structure ArtifactProvenance where
  sourceDefinitionIds : List DefinitionId
  sources : List SourceLocation
  deriving BEq, DecidableEq, Repr

/--
The runtime-facing request remains explicit model data. Requested actions and model-owned outcomes
are deliberately separate, and no field claims that runtime execution or evidence occurred.
-/
structure DrivePlan where
  formatVersion : String
  semanticIdentity : String
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
  semanticPreconditions : List SetupConstraint
  initialState : ModelValue
  requestedActions : List ModelValue
  modelOutcomes : List ModelValue
  resultingStates : List ModelValue
  linearExtension : List PlannedOccurrence
  selectedChoices : List ModelValue
  selectedVariants : List ModelValue
  requestedFaults : List ModelValue
  capabilityRequirements : List DefinitionId
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
  semanticIdentity : String
  queryBehaviorFingerprint : BehaviorFingerprint
  plan : DrivePlan
  properties : List PortableProperty
  observationRequirements : List DefinitionId
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
  "{\"identity\":" ++ quote value.definitionId.value ++
    ",\"value\":" ++ quote value.value ++ "}"

private def roleJson (role : ResourceRole) : String :=
  "{\"identity\":" ++ quote role.id.value ++
    ",\"valueKind\":" ++ quote role.valueKind.name ++ "}"

private def bindingJson (binding : RoleBinding) : String :=
  "{\"role\":" ++ quote binding.role.value ++
    ",\"value\":" ++ valueJson binding.value ++ "}"

private def operandJson : SetupOperand → String
  | .role identity =>
      "{\"kind\":\"role\",\"identity\":" ++ quote identity.value ++ "}"
  | .value value =>
      "{\"kind\":\"value\",\"value\":" ++ valueJson value ++ "}"

private def preconditionJson (constraint : SetupConstraint) : String :=
  "{\"identity\":" ++ quote constraint.id.value ++
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
  "{\"identity\":" ++ quote occurrence.definitionId.value ++
    ",\"action\":" ++ quote occurrence.action.value ++
    ",\"position\":" ++ toString occurrence.position ++
    ",\"authoredIdentity\":" ++
      (occurrence.authoredDefinitionId.map (quote ∘ DefinitionId.value) |>.getD "null") ++ "}"

private def sourceJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def provenanceJson (provenance : ArtifactProvenance) : String :=
  "{\"sourceIdentities\":" ++
      array (canonicalIds provenance.sourceDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++
    ",\"sources\":" ++ array (provenance.sources.mergeSort sourceLe |>.eraseDups |>.map sourceJson) ++ "}"

private def propertyJson (property : PortableProperty) : String :=
  "{\"identity\":" ++ quote property.definitionId.value ++
    ",\"behaviorFingerprint\":" ++ quote property.behaviorFingerprint.render ++
    ",\"requirements\":" ++
      array (canonicalIds property.requirements |>.map (quote ∘ DefinitionId.value)) ++ "}"

private def drivePlanSemanticJson (plan : DrivePlan) : String :=
  "{\"formatVersion\":" ++ quote plan.formatVersion ++
    ",\"queryIdentity\":" ++ quote plan.queryDefinitionId.value ++
    ",\"querySemanticDigest\":" ++ quote plan.queryBehaviorFingerprint.render ++
    ",\"behaviorIdentity\":" ++ quote plan.behaviorDefinitionId.value ++
    ",\"behaviorSemanticDigest\":" ++ quote plan.behaviorFingerprint.render ++
    ",\"targetIdentity\":" ++ quote plan.targetDefinitionId.value ++
    ",\"targetSemanticDigest\":" ++ quote plan.targetBehaviorFingerprint.render ++
    ",\"kernelIdentity\":" ++ quote plan.kernelDefinitionId.value ++
    ",\"kernelSemanticDigest\":" ++ quote plan.kernelBehaviorFingerprint.render ++
    ",\"bindings\":" ++ array (plan.bindings.mergeSort bindingLe |>.map bindingJson) ++
    ",\"symbolicRoles\":" ++ array (plan.symbolicRoles.map roleJson) ++
    ",\"semanticPreconditions\":" ++ array (plan.semanticPreconditions.map preconditionJson) ++
    ",\"initialState\":" ++ valueJson plan.initialState ++
    ",\"requestedActions\":" ++ array (plan.requestedActions.map valueJson) ++
    ",\"modelOutcomes\":" ++ array (plan.modelOutcomes.map valueJson) ++
    ",\"resultingStates\":" ++ array (plan.resultingStates.map valueJson) ++
    ",\"linearExtension\":" ++ array (plan.linearExtension.map plannedOccurrenceJson) ++
    ",\"selectedChoices\":" ++ array (plan.selectedChoices.map valueJson) ++
    ",\"selectedVariants\":" ++ array (plan.selectedVariants.map valueJson) ++
    ",\"requestedFaults\":" ++ array (plan.requestedFaults.map valueJson) ++
    ",\"capabilityRequirements\":" ++
      array (canonicalIds plan.capabilityRequirements |>.map (quote ∘ DefinitionId.value)) ++
    ",\"expandedBounds\":" ++ limitsJson plan.expandedLimits ++
    ",\"checkpoints\":" ++ array (plan.checkpoints.map checkpointJson) ++
    ",\"selectionReason\":" ++ quote plan.selectionReason.name ++
    ",\"explored\":" ++ exploredJson plan.explored ++
    ",\"omissions\":" ++ array (plan.knownGaps.map fun gap => quote gap.code.value) ++ "}"

def canonicalDrivePlanJson (plan : DrivePlan) : String :=
  let semantic := drivePlanSemanticJson plan
  (semantic.dropEnd 1).toString ++
    ",\"semanticIdentity\":" ++ quote plan.semanticIdentity ++
    ",\"provenance\":" ++ provenanceJson plan.provenance ++ "}"

private def experimentSpecSemanticJson (spec : ExperimentSpec) : String :=
  "{\"formatVersion\":" ++ quote spec.formatVersion ++
    ",\"querySemanticDigest\":" ++ quote spec.queryBehaviorFingerprint.render ++
    ",\"planSemanticIdentity\":" ++ quote spec.plan.semanticIdentity ++
    ",\"plan\":" ++ drivePlanSemanticJson spec.plan ++
    ",\"properties\":" ++ array (spec.properties.mergeSort propertyLe |>.map propertyJson) ++
    ",\"observationRequirements\":" ++
      array (canonicalIds spec.observationRequirements |>.map (quote ∘ DefinitionId.value)) ++ "}"

def canonicalExperimentSpecJson (spec : ExperimentSpec) : String :=
  let semantic := experimentSpecSemanticJson spec
  (semantic.dropEnd 1).toString ++
    ",\"semanticIdentity\":" ++ quote spec.semanticIdentity ++
    ",\"provenance\":" ++ provenanceJson spec.provenance ++ "}"

private def plannedOccurrence
    (behavior : CheckedBehavior)
    (index : Nat)
    (action : ModelValue)
    (authored : Option NamedOccurrence) : PlannedOccurrence :=
  let definitionId := authored.map NamedOccurrence.id |>.getD
    (DefinitionId.of (behavior.id.value ++ ".selected-occurrence-" ++ toString (index + 1)))
  {
    definitionId
    action := action.definitionId
    position := index + 1
    authoredDefinitionId := authored.map NamedOccurrence.id
  }

private def propertyReference (property : CheckedProperty) : PortableProperty := {
  definitionId := property.id
  behaviorFingerprint := property.behaviorFingerprint
  requirements := canonicalIds property.requires
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
  sources := (query.source :: query.behavior.source :: query.target.source ::
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
  let planWithoutIdentity : DrivePlan := {
    formatVersion := "umpire-drive-plan/v1"
    semanticIdentity := ""
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
    semanticPreconditions := query.behavior.setup
    initialState := trace.trace.initialState
    requestedActions := actions
    modelOutcomes := outcomes
    resultingStates := states
    linearExtension := extension
    selectedChoices := []
    selectedVariants := []
    requestedFaults := []
    capabilityRequirements := canonicalIds (query.target.requiredCapabilities ++
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
    planWithoutIdentity with
    semanticIdentity := (behaviorFingerprintOf (drivePlanSemanticJson planWithoutIdentity)).render
  }
  let properties := query.form.properties.map propertyReference |>.mergeSort propertyLe
  let observationRequirements := canonicalIds
    (query.form.properties.flatMap propertyObservationRequirements)
  let specWithoutIdentity : ExperimentSpec := {
    formatVersion := "umpire-experiment/v1"
    semanticIdentity := ""
    queryBehaviorFingerprint := query.behaviorFingerprint
    plan
    properties
    observationRequirements
    provenance
  }
  {
    specWithoutIdentity with
    semanticIdentity := (behaviorFingerprintOf (experimentSpecSemanticJson specWithoutIdentity)).render
  }

end Umpire
