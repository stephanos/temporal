import Umpire.Artifact.Codecs

namespace Umpire

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

/-- Canonical semantic values for the selected role bindings, retaining one value per role. -/
def selectedVariantValues (intent : ArtifactIntent) : List ModelValue :=
  intent.selectedVariants.map RoleBinding.value |>.mergeSort valueLe

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

/-!
Artifact intent enriches a valid model-selected v2 planning Artifact without Execution references.
Runtime configuration closes the participant and cleanup references downstream.
The v2 planning Artifact remains unchanged and carries none of those runtime bindings.
-/

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
  if !spec.plan.hasValidArtifactChecksum || !spec.hasValidArtifactChecksum then
    throw (artifactIntentError .identityDrift spec.plan.queryDefinitionId
      [spec.plan.queryDefinitionId, query.id])
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
    selectedVariants := intent.selectedVariantValues
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
