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

/-- One requested fault bound to the authored occurrence and action it intends to affect. -/
structure ArtifactFaultIntent where
  definitionId : DefinitionId
  occurrenceDefinitionId : DefinitionId
  actionDefinitionId : DefinitionId
  capabilityDefinitionId : DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Checked semantic intent that may be projected onto one planner-selected Artifact. -/
structure ArtifactIntent where
  queryDefinitionId : DefinitionId
  queryBehaviorFingerprint : BehaviorFingerprint
  behaviorDefinitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  targetDefinitionId : DefinitionId
  targetBehaviorFingerprint : BehaviorFingerprint
  kernelDefinitionId : DefinitionId
  kernelBehaviorFingerprint : BehaviorFingerprint
  selectedChoices : List ModelValue
  selectedVariants : List RoleBinding
  requestedFaults : List ArtifactFaultIntent
  additionalCapabilityRequirementDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

inductive ArtifactIntentErrorKind where
  | invalidDefinitionId
  | duplicateEntry
  | identityDrift
  | missingOccurrence
  | occurrenceMismatch
  | invalidCapability
  | variantMismatch
  deriving BEq, DecidableEq, Ord, Repr

def ArtifactIntentErrorKind.name : ArtifactIntentErrorKind → String
  | .invalidDefinitionId => "invalid-definition-id"
  | .duplicateEntry => "duplicate-entry"
  | .identityDrift => "identity-drift"
  | .missingOccurrence => "missing-occurrence"
  | .occurrenceMismatch => "occurrence-mismatch"
  | .invalidCapability => "invalid-capability"
  | .variantMismatch => "variant-mismatch"

structure ArtifactIntentError where
  kind : ArtifactIntentErrorKind
  definitionId : DefinitionId
  relatedDefinitionIds : List DefinitionId
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

private def faultIntentLe (left right : ArtifactFaultIntent) : Bool :=
  decide (left.definitionId.value ≤ right.definitionId.value)

private def canonicalFaultIntents
    (faults : List ArtifactFaultIntent) : List ArtifactFaultIntent :=
  faults.mergeSort faultIntentLe

private def artifactIntentError
    (kind : ArtifactIntentErrorKind)
    (definitionId : DefinitionId)
    (relatedDefinitionIds : List DefinitionId := []) : ArtifactIntentError := {
  kind
  definitionId
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

private def firstDuplicateId : List DefinitionId → Option DefinitionId
  | first :: rest => if rest.contains first then some first else firstDuplicateId rest
  | [] => none

private def targetHasCapability
    (query : CheckedQuery LawStatement)
    (capability : DefinitionId) : Bool :=
  query.target.requiredCapabilities.contains capability ||
    query.target.providers.any fun provider => provider.contract.id == capability

private def targetDefinesCapability
    (query : CheckedQuery LawStatement)
    (capability : DefinitionId) : Bool :=
  query.target.definitions.any fun metadata =>
    metadata.id == capability && metadata.kind == .capability

private def intentIdentityMatches
    (query : CheckedQuery LawStatement)
    (intent : ArtifactIntent) : Bool :=
  intent.queryDefinitionId == query.id &&
    intent.queryBehaviorFingerprint == query.behaviorFingerprint &&
    intent.behaviorDefinitionId == query.behavior.id &&
    intent.behaviorFingerprint == query.behavior.behaviorFingerprint &&
    intent.targetDefinitionId == query.target.id &&
    intent.targetBehaviorFingerprint == query.target.behaviorFingerprint &&
    intent.kernelDefinitionId == query.target.kernel.metadata.id &&
    intent.kernelBehaviorFingerprint == query.target.behaviorFingerprint

namespace ArtifactIntent

/-- The checked empty intent used by ordinary planning. -/
def empty (query : CheckedQuery LawStatement) : ArtifactIntent := {
  queryDefinitionId := query.id
  queryBehaviorFingerprint := query.behaviorFingerprint
  behaviorDefinitionId := query.behavior.id
  behaviorFingerprint := query.behavior.behaviorFingerprint
  targetDefinitionId := query.target.id
  targetBehaviorFingerprint := query.target.behaviorFingerprint
  kernelDefinitionId := query.target.kernel.metadata.id
  kernelBehaviorFingerprint := query.target.behaviorFingerprint
  selectedChoices := []
  selectedVariants := []
  requestedFaults := []
  additionalCapabilityRequirementDefinitionIds := []
}

/-- Recheck that intent still belongs to the exact Query closure that will be planned. -/
def validateFor
    (intent : ArtifactIntent)
    (query : CheckedQuery LawStatement) : Except ArtifactIntentError Unit := do
  if !intentIdentityMatches query intent then
    throw (artifactIntentError .identityDrift intent.queryDefinitionId [query.id])
  for choice in intent.selectedChoices do
    if !choice.definitionId.isNamespaced || !(DefinitionId.of choice.value).isNamespaced then
      throw (artifactIntentError .invalidDefinitionId choice.definitionId
        [choice.definitionId, DefinitionId.of choice.value])
  for variant in intent.selectedVariants do
    if !variant.role.isNamespaced || !variant.value.definitionId.isNamespaced then
      throw (artifactIntentError .invalidDefinitionId variant.role
        [variant.role, variant.value.definitionId])
  for fault in intent.requestedFaults do
    if !fault.definitionId.isNamespaced || !fault.occurrenceDefinitionId.isNamespaced ||
        !fault.actionDefinitionId.isNamespaced || !fault.capabilityDefinitionId.isNamespaced then
      throw (artifactIntentError .invalidDefinitionId fault.definitionId [
        fault.definitionId, fault.occurrenceDefinitionId, fault.actionDefinitionId,
        fault.capabilityDefinitionId
      ])
  for capability in intent.additionalCapabilityRequirementDefinitionIds do
    if !capability.isNamespaced then
      throw (artifactIntentError .invalidDefinitionId capability [capability])
  match firstDuplicateId (intent.selectedChoices.map ModelValue.definitionId) with
  | some duplicate =>
      throw (artifactIntentError .duplicateEntry duplicate [duplicate])
  | none => pure ()
  match firstDuplicateId (intent.selectedVariants.map RoleBinding.role) with
  | some duplicate =>
      throw (artifactIntentError .duplicateEntry duplicate [duplicate])
  | none => pure ()
  match firstDuplicateId (intent.requestedFaults.map ArtifactFaultIntent.definitionId) with
  | some duplicate =>
      throw (artifactIntentError .duplicateEntry duplicate [duplicate])
  | none => pure ()
  for variant in intent.selectedVariants do
    let knownRole := query.behavior.roles.any fun role =>
      role.id == variant.role &&
        query.target.definitions.any fun metadata =>
          metadata.id == variant.value.definitionId && metadata.kind == role.valueKind
    let available := query.target.resolvedSetups.any fun setup => setup.contains variant
    if !knownRole || !available then
      throw (artifactIntentError .variantMismatch variant.role
        [variant.role, variant.value.definitionId])
  for fault in intent.requestedFaults do
    match query.behavior.requiredOccurrences.find? fun occurrence =>
        occurrence.id == fault.occurrenceDefinitionId with
    | none =>
        throw (artifactIntentError .missingOccurrence fault.occurrenceDefinitionId
          [fault.definitionId, fault.occurrenceDefinitionId])
    | some occurrence =>
        if occurrence.action != fault.actionDefinitionId then
          throw (artifactIntentError .occurrenceMismatch fault.occurrenceDefinitionId
            [fault.definitionId, occurrence.action, fault.actionDefinitionId])
    if !intent.additionalCapabilityRequirementDefinitionIds.contains
        fault.capabilityDefinitionId then
      throw (artifactIntentError .invalidCapability fault.capabilityDefinitionId
        [fault.definitionId, fault.capabilityDefinitionId])
  for capability in intent.additionalCapabilityRequirementDefinitionIds do
    if !targetDefinesCapability query capability || !targetHasCapability query capability then
      throw (artifactIntentError .invalidCapability capability [capability, query.target.id])

end ArtifactIntent

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

private def artifactMatchesQuery
    (query : CheckedQuery LawStatement)
    (spec : ExperimentSpec) : Bool :=
  spec.queryBehaviorFingerprint == query.behaviorFingerprint &&
    spec.plan.queryDefinitionId == query.id &&
    spec.plan.queryBehaviorFingerprint == query.behaviorFingerprint &&
    spec.plan.behaviorDefinitionId == query.behavior.id &&
    spec.plan.behaviorFingerprint == query.behavior.behaviorFingerprint &&
    spec.plan.targetDefinitionId == query.target.id &&
    spec.plan.targetBehaviorFingerprint == query.target.behaviorFingerprint &&
    spec.plan.kernelDefinitionId == query.target.kernel.metadata.id &&
    spec.plan.kernelBehaviorFingerprint == query.target.behaviorFingerprint

/-- Canonically project checked intent onto an ordinary target-owned planner Artifact. -/
def ExperimentSpec.withArtifactIntent
    (spec : ExperimentSpec)
    (query : CheckedQuery LawStatement)
    (intent : ArtifactIntent) : Except ArtifactIntentError ExperimentSpec := do
  intent.validateFor query
  if !artifactMatchesQuery query spec then
    throw (artifactIntentError .identityDrift spec.plan.queryDefinitionId
      [spec.plan.queryDefinitionId, query.id])
  for variant in intent.selectedVariants do
    if !spec.plan.bindings.contains variant then
      throw (artifactIntentError .variantMismatch variant.role
        [variant.role, variant.value.definitionId])
  let mut requestedFaults := []
  for fault in canonicalFaultIntents intent.requestedFaults do
    let occurrence ← match spec.plan.linearExtension.find? fun planned =>
        planned.authoredDefinitionId == some fault.occurrenceDefinitionId with
      | some occurrence => pure occurrence
      | none => throw (artifactIntentError .missingOccurrence fault.occurrenceDefinitionId
          [fault.definitionId, fault.occurrenceDefinitionId])
    if occurrence.actionDefinitionId != fault.actionDefinitionId then
      throw (artifactIntentError .occurrenceMismatch fault.occurrenceDefinitionId
        [fault.definitionId, occurrence.actionDefinitionId, fault.actionDefinitionId])
    requestedFaults := requestedFaults ++ [{
      definitionId := fault.definitionId
      value := occurrence.definitionId.value
    }]
  let planWithoutChecksum : DrivePlan := {
    spec.plan with
    artifactChecksum := drivePlanChecksumOf ""
    selectedChoices := canonicalValues intent.selectedChoices
    selectedVariants := canonicalValues (intent.selectedVariants.map RoleBinding.value)
    requestedFaults := canonicalValues requestedFaults
    capabilityRequirementDefinitionIds := canonicalIds
      (spec.plan.capabilityRequirementDefinitionIds ++
        intent.additionalCapabilityRequirementDefinitionIds)
  }
  let plan := {
    planWithoutChecksum with
    artifactChecksum := planWithoutChecksum.expectedArtifactChecksum
  }
  let specWithoutChecksum : ExperimentSpec := {
    spec with
    artifactChecksum := experimentSpecChecksumOf ""
    plan
  }
  pure {
    specWithoutChecksum with
    artifactChecksum := specWithoutChecksum.expectedArtifactChecksum
  }

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
