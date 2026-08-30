import Umpire.Target

/-!
The Implementation Link language relates two independently checked Targets without importing either
Target family. Authored declarations are inert finite tables. A separately supplied witness carries
the forward-simulation functions and proofs, while `checkImplementationLink` validates and
canonicalizes every serializable part before returning a checked value. Proofs and mapping functions
never participate in canonical identity bytes.
-/

namespace Umpire

/-- One typed source value has one declared destination value. -/
structure ImplementationValueMapping (Source Destination : Type) where
  source : Source
  destination : Destination
  deriving BEq, DecidableEq, Repr

/-- One explicit omission from the prototype's supported source domain. -/
structure ImplementationLinkKnownGap (Source : Type) where
  source : Source
  code : DefinitionId
  reason : String
  deriving BEq, DecidableEq, Repr

/-- A semantic reference binds kind and meaning fingerprint, not only a reusable Definition ID. -/
structure ImplementationSemanticReference where
  id : DefinitionId
  kind : DefinitionKind
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

structure ImplementationSemanticMapping where
  source : ImplementationSemanticReference
  destination : ImplementationSemanticReference
  deriving BEq, DecidableEq, Repr

/-- An exact checked-Target reference retained in an inert Implementation Link declaration. -/
structure ImplementationTargetReference where
  id : DefinitionId
  kind : DefinitionKind
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

def ImplementationTargetReference.ofTarget
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation) :
    ImplementationTargetReference := {
  id := target.id
  kind := .target
  behaviorFingerprint := target.behaviorFingerprint
}

/-- Fingerprint one resolved target-owned semantic definition exactly as checking does. -/
private def implementationSemanticFingerprint
    (definition : DefinitionMetadata)
    (canonicalBehavior : String) : BehaviorFingerprint :=
  behaviorFingerprintOf <|
    "{\"id\":" ++ Lean.Json.compress (.str definition.id.value) ++
    ",\"kind\":" ++ Lean.Json.compress (.str definition.kind.name) ++
    ",\"version\":" ++ toString definition.version ++
    ",\"canonicalBehavior\":" ++ Lean.Json.compress (.str canonicalBehavior) ++ "}"

private def implementationLawLe (left right : LawDefinition) : Bool :=
  decide (left.id.value < right.id.value) ||
    (left.id == right.id && decide (left.body ≤ right.body))

private def implementationLawJson (law : LawDefinition) : String :=
  "{\"id\":" ++ Lean.Json.compress (.str law.id.value) ++
    ",\"body\":" ++ Lean.Json.compress (.str law.body) ++ "}"

private def implementationCapabilityFingerprint
    (definition : DefinitionMetadata)
    (contract : CapabilityContract) : BehaviorFingerprint :=
  let laws := contract.requiredLaws.mergeSort implementationLawLe
  behaviorFingerprintOf <|
    "{\"id\":" ++ Lean.Json.compress (.str definition.id.value) ++
    ",\"kind\":" ++ Lean.Json.compress (.str definition.kind.name) ++
    ",\"definitionVersion\":" ++ toString definition.version ++
    ",\"contractVersion\":" ++ toString contract.version ++
    ",\"canonicalBehavior\":" ++ Lean.Json.compress (.str contract.canonicalBehavior) ++
    ",\"requiredLaws\":[" ++
      String.intercalate "," (laws.map implementationLawJson) ++ "]}"

private structure ImplementationProvidedMeaning where
  provider : DefinitionId
  meaning : MeaningProvision

private def implementationProvidedMeaningLe
    (left right : ImplementationProvidedMeaning) : Bool :=
  decide (left.meaning.definitionId.value < right.meaning.definitionId.value) ||
    (left.meaning.definitionId == right.meaning.definitionId &&
      decide (left.meaning.kind.name < right.meaning.kind.name)) ||
    (left.meaning.definitionId == right.meaning.definitionId &&
      left.meaning.kind == right.meaning.kind && decide (left.provider.value ≤ right.provider.value))

private def implementationMeaningKeyLe
    (left right : DefinitionId × DefinitionKind) : Bool :=
  decide (left.1.value < right.1.value) ||
    (left.1 == right.1 && decide (left.2.name ≤ right.2.name))

private def implementationProviderIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort (fun left right => decide (left.value ≤ right.value)) |>.eraseDups

private def resolvedImplementationMeanings
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation) :
    List MeaningProvision :=
  let provided := target.providers.flatMap fun provider =>
    provider.meanings.map fun meaning => { provider := provider.id, meaning }
  let canonicalProvided := provided.mergeSort implementationProvidedMeaningLe
  let keys := canonicalProvided.map (fun item => (item.meaning.definitionId, item.meaning.kind))
    |>.mergeSort implementationMeaningKeyLe |>.eraseDups
  let reconciliations := target.connectors.flatMap CapabilityConnector.reconciliations
  keys.flatMap fun key =>
    let candidates := canonicalProvided.filter fun item =>
      item.meaning.definitionId == key.1 && item.meaning.kind == key.2
    match candidates with
    | [] => []
    | first :: _ =>
        if candidates.all fun item =>
            item.meaning.canonicalBehavior == first.meaning.canonicalBehavior then
          [first.meaning]
        else
          let providers := implementationProviderIds
            (candidates.map ImplementationProvidedMeaning.provider)
          match reconciliations.find? fun reconciliation =>
              reconciliation.definitionId == key.1 && reconciliation.kind == key.2 &&
                implementationProviderIds reconciliation.providers == providers with
          | some reconciliation => [{
              definitionId := reconciliation.definitionId
              kind := reconciliation.kind
              canonicalBehavior := reconciliation.canonicalBehavior
            }]
          | none => []

private def implementationSemanticReferences
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation) :
    List ImplementationSemanticReference :=
  let relationReferences := resolvedImplementationMeanings target |>.filterMap fun meaning => do
    if meaning.kind != .relation then none
    let definition ← target.definitions.find? fun item =>
      item.id == meaning.definitionId && item.kind == meaning.kind
    pure {
      id := definition.id
      kind := definition.kind
      behaviorFingerprint := implementationSemanticFingerprint definition meaning.canonicalBehavior
    }
  let capabilityIds := target.providers.map (fun provider => provider.contract.id)
    |>.mergeSort (fun left right => decide (left.value ≤ right.value)) |>.eraseDups
  let capabilityReferences := capabilityIds.filterMap fun id => do
    let definition ← target.definitions.find? fun item =>
      item.id == id && item.kind == .capability
    let providers := target.providers.filter fun provider => provider.contract.id == id
    let first ← providers.head?
    let fingerprint := implementationCapabilityFingerprint definition first.contract
    if providers.all fun provider =>
        implementationCapabilityFingerprint definition provider.contract == fingerprint then
      pure {
        id := definition.id
        kind := definition.kind
        behaviorFingerprint := fingerprint
      }
    else
      none
  (relationReferences ++ capabilityReferences).mergeSort fun left right =>
    decide (left.id.value < right.id.value) ||
      (left.id == right.id && decide (left.kind.name ≤ right.kind.name))

/-- Resolve one relation or capability reference from a checked Target's authoritative semantics. -/
def implementationSemanticReference?
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation)
    (id : DefinitionId)
    (kind : DefinitionKind) : Option ImplementationSemanticReference :=
  (implementationSemanticReferences target).find? fun reference =>
    reference.id == id && reference.kind == kind

private def conflictingCapabilityId?
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation) : Option DefinitionId :=
  let ids := target.requiredCapabilities
  ids.find? fun id =>
    let providers := target.providers.filter fun provider => provider.contract.id == id
    match target.definitions.find? (fun definition =>
        definition.id == id && definition.kind == .capability), providers with
    | some definition, first :: rest =>
        let fingerprint := implementationCapabilityFingerprint definition first.contract
        !(rest.all fun provider =>
          implementationCapabilityFingerprint definition provider.contract == fingerprint)
    | _, _ => false

/-- The complete serializable, domain-neutral authored correspondence. -/
structure ImplementationLinkDeclaration
    (SourceSetup SourceState SourceAction SourceOutcome SourceObservation : Type)
    (DestinationSetup DestinationState DestinationAction DestinationOutcome
      DestinationObservation : Type) where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  sourceTarget : ImplementationTargetReference
  destinationTarget : ImplementationTargetReference
  setupMappings : List (ImplementationValueMapping SourceSetup DestinationSetup)
  stateMappings : List (ImplementationValueMapping SourceState DestinationState)
  actionMappings : List (ImplementationValueMapping SourceAction DestinationAction)
  outcomeMappings : List (ImplementationValueMapping SourceOutcome DestinationOutcome)
  observationMappings : List (ImplementationValueMapping SourceObservation DestinationObservation)
  relationMappings : List ImplementationSemanticMapping
  capabilityMappings : List ImplementationSemanticMapping
  setupKnownGaps : List (ImplementationLinkKnownGap SourceSetup) := []
  stateKnownGaps : List (ImplementationLinkKnownGap SourceState) := []
  actionKnownGaps : List (ImplementationLinkKnownGap SourceAction) := []
  outcomeKnownGaps : List (ImplementationLinkKnownGap SourceOutcome) := []
  observationKnownGaps : List (ImplementationLinkKnownGap SourceObservation) := []
  relationKnownGaps : List (ImplementationLinkKnownGap DefinitionId) := []
  capabilityKnownGaps : List (ImplementationLinkKnownGap DefinitionId) := []
  applicationLimit : Limit
  documentation : String := ""

inductive ImplementationLinkObligation where
  | initialForward
  | stepForward
  | requiredCoverage
  deriving BEq, DecidableEq, Ord, Repr

def ImplementationLinkObligation.name : ImplementationLinkObligation → String
  | .initialForward => "initial-forward"
  | .stepForward => "step-forward"
  | .requiredCoverage => "required-coverage"

/-- Runtime-checkable witness labels supplement the witness's exact dependent indices. -/
structure ImplementationLinkWitnessIndex where
  definitionId : DefinitionId
  declarationVersion : Nat
  sourceTarget : ImplementationTargetReference
  destinationTarget : ImplementationTargetReference
  deriving BEq, DecidableEq, Repr

def implementationLinkWitnessIndex
    (declaration : ImplementationLinkDeclaration
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation)
    (source : CheckedTarget SourceLawStatement SourceSetup SourceState SourceAction
      SourceOutcome SourceObservation)
    (destination : CheckedTarget DestinationLawStatement DestinationSetup DestinationState
      DestinationAction DestinationOutcome DestinationObservation) :
    ImplementationLinkWitnessIndex := {
  definitionId := declaration.id
  declarationVersion := declaration.version
  sourceTarget := .ofTarget source
  destinationTarget := .ofTarget destination
}

/-- Coverage proves the finite support/Known Gap partition against the exact authored tables. -/
structure ImplementationLinkRequiredCoverage
    (declaration : ImplementationLinkDeclaration
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation)
    (source : CheckedTarget SourceLawStatement SourceSetup SourceState SourceAction
      SourceOutcome SourceObservation)
    (mapSetup : SourceSetup → DestinationSetup)
    (mapState : SourceState → DestinationState)
    (mapAction : SourceAction → DestinationAction)
    (mapOutcome : SourceOutcome → DestinationOutcome)
    (mapObservation : SourceObservation → DestinationObservation) : Prop where
  setup : ∀ value, source.kernel.setupDomain value →
    ({ source := value, destination := mapSetup value } ∈ declaration.setupMappings) ∨
      ∃ gap, gap ∈ declaration.setupKnownGaps ∧ gap.source = value
  state : ∀ value, source.kernel.stateDomain value →
    ({ source := value, destination := mapState value } ∈ declaration.stateMappings) ∨
      ∃ gap, gap ∈ declaration.stateKnownGaps ∧ gap.source = value
  action : ∀ value, source.kernel.actionDomain value →
    ({ source := value, destination := mapAction value } ∈ declaration.actionMappings) ∨
      ∃ gap, gap ∈ declaration.actionKnownGaps ∧ gap.source = value
  outcome : ∀ value, source.kernel.outcomeDomain value →
    ({ source := value, destination := mapOutcome value } ∈ declaration.outcomeMappings) ∨
      ∃ gap, gap ∈ declaration.outcomeKnownGaps ∧ gap.source = value
  observation : ∀ value, source.kernel.observationDomain value →
    ({ source := value, destination := mapObservation value } ∈ declaration.observationMappings) ∨
      ∃ gap, gap ∈ declaration.observationKnownGaps ∧ gap.source = value
  relation : List.Perm
    (declaration.relationMappings.map (fun mapping => mapping.source.id) ++
      declaration.relationKnownGaps.map ImplementationLinkKnownGap.source)
    (source.definitions.filter (fun definition => definition.kind == .relation) |>.map
      DefinitionMetadata.id)
  capability : List.Perm
    (declaration.capabilityMappings.map (fun mapping => mapping.source.id) ++
      declaration.capabilityKnownGaps.map ImplementationLinkKnownGap.source)
    source.requiredCapabilities

/-- Exact proof-carrying bounded forward simulation for one declaration and two checked Targets. -/
structure ImplementationLinkWitness
    (declaration : ImplementationLinkDeclaration
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation)
    (source : CheckedTarget SourceLawStatement SourceSetup SourceState SourceAction
      SourceOutcome SourceObservation)
    (destination : CheckedTarget DestinationLawStatement DestinationSetup DestinationState
      DestinationAction DestinationOutcome DestinationObservation) where
  index : ImplementationLinkWitnessIndex
  mapSetup : SourceSetup → DestinationSetup
  mapState : SourceState → DestinationState
  mapAction : SourceAction → DestinationAction
  mapOutcome : SourceOutcome → DestinationOutcome
  mapObservation : SourceObservation → DestinationObservation
  initialForward : ∀ setup state,
    source.kernel.authoritativeInitial setup state →
      destination.kernel.authoritativeInitial (mapSetup setup) (mapState state)
  stepForward : ∀ state action result,
    source.kernel.authoritativeStep state action result →
      destination.kernel.authoritativeStep (mapState state) (mapAction action) {
        modelOutcome := mapOutcome result.modelOutcome
        resultingState := mapState result.resultingState
        observations := result.observations.map mapObservation
      }
  requiredCoverage : ImplementationLinkRequiredCoverage declaration source mapSetup mapState
    mapAction mapOutcome mapObservation

/-- Missing proof obligations remain representable only at the authored checking boundary. -/
inductive ImplementationLinkWitnessAuthoring
    (declaration : ImplementationLinkDeclaration
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation)
    (source : CheckedTarget SourceLawStatement SourceSetup SourceState SourceAction
      SourceOutcome SourceObservation)
    (destination : CheckedTarget DestinationLawStatement DestinationSetup DestinationState
      DestinationAction DestinationOutcome DestinationObservation) where
  | complete (witness : ImplementationLinkWitness declaration source destination)
  | incomplete (index : ImplementationLinkWitnessIndex)
      (missing : List ImplementationLinkObligation)

instance : Coe (ImplementationLinkWitness declaration source destination)
    (ImplementationLinkWitnessAuthoring declaration source destination) :=
  ⟨ImplementationLinkWitnessAuthoring.complete⟩

/-- Exact source-kernel admission for every positional step of a Model Trace. -/
def AuthoritativeTraceSteps
    (kernel : TransitionKernel Setup State Action Outcome Observation) :
    State → List (ModelTraceStep State Action Outcome Observation) → Prop
  | _, [] => True
  | state, step :: rest =>
      kernel.authoritativeStep state step.selectedAction {
        modelOutcome := step.modelOutcome
        resultingState := step.resultingState
        observations := step.observations
      } ∧ AuthoritativeTraceSteps kernel step.resultingState rest

structure AuthoritativeModelTrace
    (kernel : TransitionKernel Setup State Action Outcome Observation)
    (setup : Setup)
    (trace : ModelTrace State Action Outcome Observation) : Prop where
  initial : kernel.authoritativeInitial setup trace.initialState
  steps : AuthoritativeTraceSteps kernel trace.initialState trace.steps

def ImplementationLinkWitness.translateStep
    (witness : ImplementationLinkWitness
      (SourceSetup := SourceSetup) (SourceState := SourceState) (SourceAction := SourceAction)
      (SourceOutcome := SourceOutcome) (SourceObservation := SourceObservation)
      (DestinationSetup := DestinationSetup) (DestinationState := DestinationState)
      (DestinationAction := DestinationAction) (DestinationOutcome := DestinationOutcome)
      (DestinationObservation := DestinationObservation) declaration source destination)
    (step : ModelTraceStep SourceState SourceAction SourceOutcome SourceObservation) :
    ModelTraceStep DestinationState DestinationAction DestinationOutcome DestinationObservation := {
  selectedAction := witness.mapAction step.selectedAction
  modelOutcome := witness.mapOutcome step.modelOutcome
  resultingState := witness.mapState step.resultingState
  observations := step.observations.map witness.mapObservation
}

def ImplementationLinkWitness.translateTrace
    (witness : ImplementationLinkWitness
      (SourceSetup := SourceSetup) (SourceState := SourceState) (SourceAction := SourceAction)
      (SourceOutcome := SourceOutcome) (SourceObservation := SourceObservation)
      (DestinationSetup := DestinationSetup) (DestinationState := DestinationState)
      (DestinationAction := DestinationAction) (DestinationOutcome := DestinationOutcome)
      (DestinationObservation := DestinationObservation) declaration source destination)
    (trace : ModelTrace SourceState SourceAction SourceOutcome SourceObservation) :
    ModelTrace DestinationState DestinationAction DestinationOutcome DestinationObservation := {
  initialState := witness.mapState trace.initialState
  steps := trace.steps.map witness.translateStep
}

private theorem ImplementationLinkWitness.stepsForward
    (witness : ImplementationLinkWitness
      (SourceSetup := SourceSetup) (SourceState := SourceState) (SourceAction := SourceAction)
      (SourceOutcome := SourceOutcome) (SourceObservation := SourceObservation)
      (DestinationSetup := DestinationSetup) (DestinationState := DestinationState)
      (DestinationAction := DestinationAction) (DestinationOutcome := DestinationOutcome)
      (DestinationObservation := DestinationObservation) declaration source destination)
    (state : SourceState)
    (steps : List (ModelTraceStep SourceState SourceAction SourceOutcome SourceObservation))
    (admitted : AuthoritativeTraceSteps source.kernel state steps) :
    AuthoritativeTraceSteps destination.kernel (witness.mapState state)
      (steps.map witness.translateStep) := by
  induction steps generalizing state with
  | nil => trivial
  | cons step rest induction =>
      exact ⟨witness.stepForward state step.selectedAction {
          modelOutcome := step.modelOutcome
          resultingState := step.resultingState
          observations := step.observations
        } admitted.1,
        induction step.resultingState admitted.2⟩

/-- The trace theorem is derived from initial and step forward simulation; authors supply no trace proof. -/
theorem ImplementationLinkWitness.traceForward
    (witness : ImplementationLinkWitness
      (SourceSetup := SourceSetup) (SourceState := SourceState) (SourceAction := SourceAction)
      (SourceOutcome := SourceOutcome) (SourceObservation := SourceObservation)
      (DestinationSetup := DestinationSetup) (DestinationState := DestinationState)
      (DestinationAction := DestinationAction) (DestinationOutcome := DestinationOutcome)
      (DestinationObservation := DestinationObservation) declaration source destination)
    (setup : SourceSetup)
    (trace : ModelTrace SourceState SourceAction SourceOutcome SourceObservation)
    (admitted : AuthoritativeModelTrace source.kernel setup trace) :
    AuthoritativeModelTrace destination.kernel (witness.mapSetup setup)
      (witness.translateTrace trace) := {
  initial := witness.initialForward setup trace.initialState admitted.initial
  steps := witness.stepsForward trace.initialState trace.steps admitted.steps
}

inductive ImplementationLinkErrorKind where
  | emptyDefinitionId
  | invalidDefinitionId
  | invalidVersion
  | staleSourceTarget
  | staleDestinationTarget
  | wrongKind
  | behaviorFingerprintDrift
  | incompatibleCapability
  | duplicateMapping
  | ambiguousMapping
  | unknownSourceValue
  | unknownDestinationValue
  | invalidKnownGap
  | duplicateKnownGap
  | incompleteSupportPartition
  | contradictorySupportPartition
  | invalidLimitValue
  | invalidLimitUnit
  | missingInitialForward
  | missingStepForward
  | missingRequiredCoverage
  | witnessDeclarationMismatch
  | witnessSourceMismatch
  | witnessDestinationMismatch
  deriving BEq, DecidableEq, Ord, Repr

def ImplementationLinkErrorKind.name : ImplementationLinkErrorKind → String
  | .emptyDefinitionId => "empty-definition-id"
  | .invalidDefinitionId => "invalid-definition-id"
  | .invalidVersion => "invalid-version"
  | .staleSourceTarget => "stale-source-target"
  | .staleDestinationTarget => "stale-destination-target"
  | .wrongKind => "wrong-kind"
  | .behaviorFingerprintDrift => "behavior-fingerprint-drift"
  | .incompatibleCapability => "incompatible-capability"
  | .duplicateMapping => "duplicate-mapping"
  | .ambiguousMapping => "ambiguous-mapping"
  | .unknownSourceValue => "unknown-source-value"
  | .unknownDestinationValue => "unknown-destination-value"
  | .invalidKnownGap => "invalid-known-gap"
  | .duplicateKnownGap => "duplicate-known-gap"
  | .incompleteSupportPartition => "incomplete-support-partition"
  | .contradictorySupportPartition => "contradictory-support-partition"
  | .invalidLimitValue => "invalid-limit-value"
  | .invalidLimitUnit => "invalid-limit-unit"
  | .missingInitialForward => "missing-initial-forward"
  | .missingStepForward => "missing-step-forward"
  | .missingRequiredCoverage => "missing-required-coverage"
  | .witnessDeclarationMismatch => "witness-declaration-mismatch"
  | .witnessSourceMismatch => "witness-source-mismatch"
  | .witnessDestinationMismatch => "witness-destination-mismatch"

structure ImplementationLinkError where
  kind : ImplementationLinkErrorKind
  implementationLinkId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- A checked link retains proof functions for use while exposing only canonical declaration data. -/
structure CheckedImplementationLink
    (SourceLawStatement : LawDefinition → Prop)
    (DestinationLawStatement : LawDefinition → Prop)
    (SourceSetup SourceState SourceAction SourceOutcome SourceObservation : Type)
    (DestinationSetup DestinationState DestinationAction DestinationOutcome
      DestinationObservation : Type) where
  private mk ::
  declaration : ImplementationLinkDeclaration
    SourceSetup SourceState SourceAction SourceOutcome SourceObservation
    DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation
  sourceTarget : CheckedTarget SourceLawStatement SourceSetup SourceState SourceAction
    SourceOutcome SourceObservation
  destinationTarget : CheckedTarget DestinationLawStatement DestinationSetup DestinationState
    DestinationAction DestinationOutcome DestinationObservation
  mapSetup : SourceSetup → DestinationSetup
  mapState : SourceState → DestinationState
  mapAction : SourceAction → DestinationAction
  mapOutcome : SourceOutcome → DestinationOutcome
  mapObservation : SourceObservation → DestinationObservation
  initialForward : ∀ setup state,
    sourceTarget.kernel.authoritativeInitial setup state →
      destinationTarget.kernel.authoritativeInitial (mapSetup setup) (mapState state)
  stepForward : ∀ state action result,
    sourceTarget.kernel.authoritativeStep state action result →
      destinationTarget.kernel.authoritativeStep (mapState state) (mapAction action) {
        modelOutcome := mapOutcome result.modelOutcome
        resultingState := mapState result.resultingState
        observations := result.observations.map mapObservation
      }
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def sourcePath (source : SourceLocation) : String :=
  if source.path == "" then "<unknown>" else source.path

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort (fun left right => decide (left.value ≤ right.value)) |>.eraseDups

private def implementationLinkError
    (kind : ImplementationLinkErrorKind)
    (id : DefinitionId)
    (source : SourceLocation)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : ImplementationLinkError := {
  kind
  implementationLinkId := if id.value == "" then
    DefinitionId.of "umpire.implementation-link.anonymous"
  else id
  sourcePath := sourcePath source
  offendingValue
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

private def targetReferenceJson (reference : ImplementationTargetReference) : String :=
  "{\"id\":" ++ quote reference.id.value ++
    ",\"kind\":" ++ quote reference.kind.name ++
    ",\"behaviorFingerprint\":" ++ quote reference.behaviorFingerprint.render ++ "}"

private def semanticReferenceJson (reference : ImplementationSemanticReference) : String :=
  "{\"id\":" ++ quote reference.id.value ++
    ",\"kind\":" ++ quote reference.kind.name ++
    ",\"behaviorFingerprint\":" ++ quote reference.behaviorFingerprint.render ++ "}"

private def semanticMappingJson (mapping : ImplementationSemanticMapping) : String :=
  "{\"source\":" ++ semanticReferenceJson mapping.source ++
    ",\"destination\":" ++ semanticReferenceJson mapping.destination ++ "}"

private def semanticMappingKey (mapping : ImplementationSemanticMapping) : String :=
  semanticMappingJson mapping

private def semanticMappingLe
    (left right : ImplementationSemanticMapping) : Bool :=
  decide (semanticMappingKey left ≤ semanticMappingKey right)

private def valueMappingJson
    (encodeSource : Source → String)
    (encodeDestination : Destination → String)
    (mapping : ImplementationValueMapping Source Destination) : String :=
  "{\"source\":" ++ quote (encodeSource mapping.source) ++
    ",\"destination\":" ++ quote (encodeDestination mapping.destination) ++ "}"

private def valueMappingLe
    (encodeSource : Source → String)
    (encodeDestination : Destination → String)
    (left right : ImplementationValueMapping Source Destination) : Bool :=
  decide (valueMappingJson encodeSource encodeDestination left ≤
    valueMappingJson encodeSource encodeDestination right)

private def knownGapJson
    (encodeSource : Source → String)
    (gap : ImplementationLinkKnownGap Source) : String :=
  "{\"source\":" ++ quote (encodeSource gap.source) ++
    ",\"code\":" ++ quote gap.code.value ++
    ",\"reason\":" ++ quote gap.reason ++ "}"

private def knownGapLe
    (encodeSource : Source → String)
    (left right : ImplementationLinkKnownGap Source) : Bool :=
  decide (knownGapJson encodeSource left ≤ knownGapJson encodeSource right)

private def semanticGapJson (gap : ImplementationLinkKnownGap DefinitionId) : String :=
  knownGapJson DefinitionId.value gap

private def semanticGapLe
    (left right : ImplementationLinkKnownGap DefinitionId) : Bool :=
  knownGapLe DefinitionId.value left right

private def sourceLocationJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def canonicalizeDeclaration
    (declaration : ImplementationLinkDeclaration
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation)
    (sourceDomain : TargetBehaviorDomain
      (Setup := SourceSetup) (State := SourceState) (Action := SourceAction)
      (Outcome := SourceOutcome) (Observation := SourceObservation)
      sourceSetupDomain sourceStateDomain sourceActionDomain sourceOutcomeDomain
      sourceObservationDomain sourceInitialStates sourceSteps)
    (destinationDomain : TargetBehaviorDomain
      (Setup := DestinationSetup) (State := DestinationState) (Action := DestinationAction)
      (Outcome := DestinationOutcome) (Observation := DestinationObservation)
      destinationSetupDomain destinationStateDomain destinationActionDomain destinationOutcomeDomain
      destinationObservationDomain destinationInitialStates destinationSteps) := {
  declaration with
  setupMappings := declaration.setupMappings.mergeSort
    (valueMappingLe sourceDomain.encodeSetup destinationDomain.encodeSetup)
  stateMappings := declaration.stateMappings.mergeSort
    (valueMappingLe sourceDomain.encodeState destinationDomain.encodeState)
  actionMappings := declaration.actionMappings.mergeSort
    (valueMappingLe sourceDomain.encodeAction destinationDomain.encodeAction)
  outcomeMappings := declaration.outcomeMappings.mergeSort
    (valueMappingLe sourceDomain.encodeOutcome destinationDomain.encodeOutcome)
  observationMappings := declaration.observationMappings.mergeSort
    (valueMappingLe sourceDomain.encodeObservation destinationDomain.encodeObservation)
  relationMappings := declaration.relationMappings.mergeSort semanticMappingLe
  capabilityMappings := declaration.capabilityMappings.mergeSort semanticMappingLe
  setupKnownGaps := declaration.setupKnownGaps.mergeSort (knownGapLe sourceDomain.encodeSetup)
  stateKnownGaps := declaration.stateKnownGaps.mergeSort (knownGapLe sourceDomain.encodeState)
  actionKnownGaps := declaration.actionKnownGaps.mergeSort (knownGapLe sourceDomain.encodeAction)
  outcomeKnownGaps := declaration.outcomeKnownGaps.mergeSort (knownGapLe sourceDomain.encodeOutcome)
  observationKnownGaps := declaration.observationKnownGaps.mergeSort
    (knownGapLe sourceDomain.encodeObservation)
  relationKnownGaps := declaration.relationKnownGaps.mergeSort semanticGapLe
  capabilityKnownGaps := declaration.capabilityKnownGaps.mergeSort semanticGapLe
}

private def implementationLinkSemanticJson
    (declaration : ImplementationLinkDeclaration
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation)
    (sourceDomain : TargetBehaviorDomain
      (Setup := SourceSetup) (State := SourceState) (Action := SourceAction)
      (Outcome := SourceOutcome) (Observation := SourceObservation)
      sourceSetupDomain sourceStateDomain sourceActionDomain sourceOutcomeDomain
      sourceObservationDomain sourceInitialStates sourceSteps)
    (destinationDomain : TargetBehaviorDomain
      (Setup := DestinationSetup) (State := DestinationState) (Action := DestinationAction)
      (Outcome := DestinationOutcome) (Observation := DestinationObservation)
      destinationSetupDomain destinationStateDomain destinationActionDomain destinationOutcomeDomain
      destinationObservationDomain destinationInitialStates destinationSteps) : String :=
  "{\"id\":" ++ quote declaration.id.value ++
    ",\"version\":" ++ toString declaration.version ++
    ",\"sourceTarget\":" ++ targetReferenceJson declaration.sourceTarget ++
    ",\"destinationTarget\":" ++ targetReferenceJson declaration.destinationTarget ++
    ",\"setupMappings\":" ++ array (declaration.setupMappings.map
      (valueMappingJson sourceDomain.encodeSetup destinationDomain.encodeSetup)) ++
    ",\"stateMappings\":" ++ array (declaration.stateMappings.map
      (valueMappingJson sourceDomain.encodeState destinationDomain.encodeState)) ++
    ",\"actionMappings\":" ++ array (declaration.actionMappings.map
      (valueMappingJson sourceDomain.encodeAction destinationDomain.encodeAction)) ++
    ",\"outcomeMappings\":" ++ array (declaration.outcomeMappings.map
      (valueMappingJson sourceDomain.encodeOutcome destinationDomain.encodeOutcome)) ++
    ",\"observationMappings\":" ++ array (declaration.observationMappings.map
      (valueMappingJson sourceDomain.encodeObservation destinationDomain.encodeObservation)) ++
    ",\"relationMappings\":" ++ array (declaration.relationMappings.map semanticMappingJson) ++
    ",\"capabilityMappings\":" ++ array (declaration.capabilityMappings.map semanticMappingJson) ++
    ",\"setupKnownGaps\":" ++ array (declaration.setupKnownGaps.map
      (knownGapJson sourceDomain.encodeSetup)) ++
    ",\"stateKnownGaps\":" ++ array (declaration.stateKnownGaps.map
      (knownGapJson sourceDomain.encodeState)) ++
    ",\"actionKnownGaps\":" ++ array (declaration.actionKnownGaps.map
      (knownGapJson sourceDomain.encodeAction)) ++
    ",\"outcomeKnownGaps\":" ++ array (declaration.outcomeKnownGaps.map
      (knownGapJson sourceDomain.encodeOutcome)) ++
    ",\"observationKnownGaps\":" ++ array (declaration.observationKnownGaps.map
      (knownGapJson sourceDomain.encodeObservation)) ++
    ",\"relationKnownGaps\":" ++ array (declaration.relationKnownGaps.map semanticGapJson) ++
    ",\"capabilityKnownGaps\":" ++ array (declaration.capabilityKnownGaps.map semanticGapJson) ++
    ",\"applicationLimit\":" ++ canonicalLimitJson declaration.applicationLimit ++
    ",\"obligations\":[\"initial-forward\",\"step-forward\",\"required-coverage\"]}"

private def canonicalImplementationLinkJson
    (declaration : ImplementationLinkDeclaration
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation)
    (semantic : String) : String :=
  "{\"semantic\":" ++ semantic ++
    ",\"source\":" ++ sourceLocationJson declaration.source ++
    ",\"documentation\":" ++ quote declaration.documentation ++ "}"

private def validateTargetReference
    (definitionId : DefinitionId)
    (source : SourceLocation)
    (role : String)
    (reference expected : ImplementationTargetReference) : Except ImplementationLinkError Unit := do
  if reference.kind != .target then
    throw (implementationLinkError .wrongKind definitionId source
      (role ++ ":" ++ reference.kind.name) [reference.id])
  if reference.id != expected.id then
    throw (implementationLinkError
      (if role == "source" then .staleSourceTarget else .staleDestinationTarget)
      definitionId source reference.id.value [reference.id, expected.id])
  if reference.behaviorFingerprint != expected.behaviorFingerprint then
    throw (implementationLinkError .behaviorFingerprintDrift definitionId source
      role [reference.id])

private def validateKnownGap
    (definitionId : DefinitionId)
    (source : SourceLocation)
    (label : String)
    (gap : ImplementationLinkKnownGap Value) : Except ImplementationLinkError Unit := do
  if !gap.code.isNamespaced || gap.reason == "" then
    throw (implementationLinkError .invalidKnownGap definitionId source
      (label ++ ":" ++ gap.code.value) [gap.code])

private def validateValueTable
    [BEq Source] [BEq Destination]
    (definitionId : DefinitionId)
    (source : SourceLocation)
    (label : String)
    (encodeSource : Source → String)
    (sourceValues : List Source)
    (destinationValues : List Destination)
    (mappings : List (ImplementationValueMapping Source Destination))
    (knownGaps : List (ImplementationLinkKnownGap Source)) : Except ImplementationLinkError Unit := do
  for mapping in mappings do
    if (mappings.filter fun other => other == mapping).length > 1 then
      throw (implementationLinkError .duplicateMapping definitionId source
        (label ++ ":" ++ encodeSource mapping.source))
    if (mappings.filter fun other => other.source == mapping.source).length > 1 then
      throw (implementationLinkError .ambiguousMapping definitionId source
        (label ++ ":" ++ encodeSource mapping.source))
    if !sourceValues.contains mapping.source then
      throw (implementationLinkError .unknownSourceValue definitionId source
        (label ++ ":" ++ encodeSource mapping.source))
    if !destinationValues.contains mapping.destination then
      throw (implementationLinkError .unknownDestinationValue definitionId source
        (label ++ ":" ++ encodeSource mapping.source))
  for gap in knownGaps do
    validateKnownGap definitionId source label gap
    if !sourceValues.contains gap.source then
      throw (implementationLinkError .unknownSourceValue definitionId source
        (label ++ ":" ++ encodeSource gap.source) [gap.code])
    if (knownGaps.filter fun other => other.source == gap.source).length > 1 then
      throw (implementationLinkError .duplicateKnownGap definitionId source
        (label ++ ":" ++ encodeSource gap.source) [gap.code])
  for value in sourceValues do
    let mappedCount := (mappings.filter fun mapping => mapping.source == value).length
    let gapCount := (knownGaps.filter fun gap => gap.source == value).length
    if mappedCount + gapCount == 0 then
      throw (implementationLinkError .incompleteSupportPartition definitionId source
        (label ++ ":" ++ encodeSource value))
    if mappedCount + gapCount > 1 then
      throw (implementationLinkError .contradictorySupportPartition definitionId source
        (label ++ ":" ++ encodeSource value))

private def definitionById
    (definitions : List DefinitionMetadata)
    (id : DefinitionId) : Option DefinitionMetadata :=
  definitions.find? fun definition => definition.id == id

private def validateSemanticReference
    (definitionId : DefinitionId)
    (source : SourceLocation)
    (role : String)
    (expectedKind : DefinitionKind)
    (definitions : List DefinitionMetadata)
    (references : List ImplementationSemanticReference)
    (reference : ImplementationSemanticReference) : Except ImplementationLinkError Unit := do
  if reference.kind != expectedKind then
    throw (implementationLinkError .wrongKind definitionId source
      (role ++ ":" ++ reference.kind.name) [reference.id])
  match definitionById definitions reference.id with
  | none =>
      throw (implementationLinkError
        (if role == "source" then .unknownSourceValue else .unknownDestinationValue)
        definitionId source reference.id.value [reference.id])
  | some definition =>
      if definition.kind != expectedKind then
        throw (implementationLinkError .wrongKind definitionId source
          (role ++ ":" ++ definition.kind.name) [reference.id])
      match references.find? fun expected =>
          expected.id == reference.id && expected.kind == reference.kind with
      | none =>
          throw (implementationLinkError
            (if role == "source" then .unknownSourceValue else .unknownDestinationValue)
            definitionId source reference.id.value [reference.id])
      | some expected =>
          if expected.behaviorFingerprint != reference.behaviorFingerprint then
            throw (implementationLinkError .behaviorFingerprintDrift definitionId source
              role [reference.id])

private def validateSemanticTable
    (definitionId : DefinitionId)
    (source : SourceLocation)
    (label : String)
    (expectedKind : DefinitionKind)
    (sourceDefinitions destinationDefinitions : List DefinitionMetadata)
    (sourceReferences destinationReferences : List ImplementationSemanticReference)
    (requiredSourceIds : List DefinitionId)
    (allowedDestinationIds : List DefinitionId)
    (mappings : List ImplementationSemanticMapping)
    (knownGaps : List (ImplementationLinkKnownGap DefinitionId)) :
    Except ImplementationLinkError Unit := do
  for mapping in mappings do
    if (mappings.filter fun other => other == mapping).length > 1 then
      throw (implementationLinkError .duplicateMapping definitionId source
        (label ++ ":" ++ mapping.source.id.value) [mapping.source.id])
    if (mappings.filter fun other => other.source.id == mapping.source.id).length > 1 then
      throw (implementationLinkError .ambiguousMapping definitionId source
        (label ++ ":" ++ mapping.source.id.value) [mapping.source.id])
    validateSemanticReference definitionId source "source" expectedKind sourceDefinitions
      sourceReferences mapping.source
    validateSemanticReference definitionId source "destination" expectedKind
      destinationDefinitions destinationReferences mapping.destination
    if !requiredSourceIds.contains mapping.source.id then
      throw (implementationLinkError .unknownSourceValue definitionId source
        (label ++ ":" ++ mapping.source.id.value) [mapping.source.id])
    if !allowedDestinationIds.contains mapping.destination.id then
      throw (implementationLinkError .unknownDestinationValue definitionId source
        (label ++ ":" ++ mapping.destination.id.value) [mapping.destination.id])
  for gap in knownGaps do
    validateKnownGap definitionId source label gap
    if !requiredSourceIds.contains gap.source then
      throw (implementationLinkError .unknownSourceValue definitionId source
        (label ++ ":" ++ gap.source.value) [gap.source, gap.code])
    if (knownGaps.filter fun other => other.source == gap.source).length > 1 then
      throw (implementationLinkError .duplicateKnownGap definitionId source
        (label ++ ":" ++ gap.source.value) [gap.source, gap.code])
  for id in requiredSourceIds do
    let mappedCount := (mappings.filter fun mapping => mapping.source.id == id).length
    let gapCount := (knownGaps.filter fun gap => gap.source == id).length
    if mappedCount + gapCount == 0 then
      throw (implementationLinkError .incompleteSupportPartition definitionId source
        (label ++ ":" ++ id.value) [id])
    if mappedCount + gapCount > 1 then
      throw (implementationLinkError .contradictorySupportPartition definitionId source
        (label ++ ":" ++ id.value) [id])

private def validateWitnessIndex
    (definitionId : DefinitionId)
    (sourceLocation : SourceLocation)
    (expected actual : ImplementationLinkWitnessIndex) : Except ImplementationLinkError Unit := do
  if actual.definitionId != expected.definitionId ||
      actual.declarationVersion != expected.declarationVersion then
    throw (implementationLinkError .witnessDeclarationMismatch definitionId sourceLocation
      actual.definitionId.value [actual.definitionId, expected.definitionId])
  if actual.sourceTarget != expected.sourceTarget then
    throw (implementationLinkError .witnessSourceMismatch definitionId sourceLocation
      actual.sourceTarget.id.value [actual.sourceTarget.id, expected.sourceTarget.id])
  if actual.destinationTarget != expected.destinationTarget then
    throw (implementationLinkError .witnessDestinationMismatch definitionId sourceLocation
      actual.destinationTarget.id.value [actual.destinationTarget.id, expected.destinationTarget.id])

private def missingObligationError : ImplementationLinkObligation → ImplementationLinkErrorKind
  | .initialForward => .missingInitialForward
  | .stepForward => .missingStepForward
  | .requiredCoverage => .missingRequiredCoverage

private def obligationLe
    (left right : ImplementationLinkObligation) : Bool :=
  decide (left.name ≤ right.name)

private def checkImplementationLinkWithDomains
    [BEq SourceSetup] [BEq SourceState] [BEq SourceAction] [BEq SourceOutcome]
    [BEq SourceObservation] [BEq DestinationSetup] [BEq DestinationState]
    [BEq DestinationAction] [BEq DestinationOutcome] [BEq DestinationObservation]
    (declaration : ImplementationLinkDeclaration
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation)
    (source : CheckedTarget SourceLawStatement SourceSetup SourceState SourceAction
      SourceOutcome SourceObservation)
    (destination : CheckedTarget DestinationLawStatement DestinationSetup DestinationState
      DestinationAction DestinationOutcome DestinationObservation)
    (sourceDomain : TargetBehaviorDomain source.kernel.setupDomain source.kernel.stateDomain
      source.kernel.actionDomain source.kernel.outcomeDomain source.kernel.observationDomain
      source.kernel.initialStates source.kernel.steps)
    (destinationDomain : TargetBehaviorDomain destination.kernel.setupDomain
      destination.kernel.stateDomain destination.kernel.actionDomain destination.kernel.outcomeDomain
      destination.kernel.observationDomain destination.kernel.initialStates destination.kernel.steps)
    (authoredWitness : ImplementationLinkWitnessAuthoring declaration source destination) :
    Except ImplementationLinkError (CheckedImplementationLink SourceLawStatement
      DestinationLawStatement SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation) := do
  if declaration.id.value == "" then
    throw (implementationLinkError .emptyDefinitionId declaration.id declaration.source "<empty>")
  if !declaration.id.isNamespaced then
    throw (implementationLinkError .invalidDefinitionId declaration.id declaration.source
      declaration.id.value [declaration.id])
  if declaration.version == 0 then
    throw (implementationLinkError .invalidVersion declaration.id declaration.source "0")
  validateTargetReference declaration.id declaration.source "source" declaration.sourceTarget
    (.ofTarget source)
  validateTargetReference declaration.id declaration.source "destination"
    declaration.destinationTarget (.ofTarget destination)
  if declaration.applicationLimit.value == 0 then
    throw (implementationLinkError .invalidLimitValue declaration.id declaration.source "0")
  if declaration.applicationLimit.unit != .semanticTransitions then
    throw (implementationLinkError .invalidLimitUnit declaration.id declaration.source
      declaration.applicationLimit.unit.name)
  match conflictingCapabilityId? source with
  | some id =>
      throw (implementationLinkError .incompatibleCapability declaration.id declaration.source
        ("source:" ++ id.value) [id])
  | none => pure ()
  match conflictingCapabilityId? destination with
  | some id =>
      throw (implementationLinkError .incompatibleCapability declaration.id declaration.source
        ("destination:" ++ id.value) [id])
  | none => pure ()
  let canonical := canonicalizeDeclaration declaration sourceDomain destinationDomain
  validateValueTable declaration.id declaration.source "setup" sourceDomain.encodeSetup
    sourceDomain.setups destinationDomain.setups canonical.setupMappings canonical.setupKnownGaps
  validateValueTable declaration.id declaration.source "state" sourceDomain.encodeState
    sourceDomain.states destinationDomain.states canonical.stateMappings canonical.stateKnownGaps
  validateValueTable declaration.id declaration.source "action" sourceDomain.encodeAction
    sourceDomain.actions destinationDomain.actions canonical.actionMappings canonical.actionKnownGaps
  validateValueTable declaration.id declaration.source "outcome" sourceDomain.encodeOutcome
    sourceDomain.outcomes destinationDomain.outcomes canonical.outcomeMappings canonical.outcomeKnownGaps
  validateValueTable declaration.id declaration.source "observation" sourceDomain.encodeObservation
    sourceDomain.observations destinationDomain.observations canonical.observationMappings
    canonical.observationKnownGaps
  let relations := source.definitions.filter (fun definition => definition.kind == .relation)
    |>.map DefinitionMetadata.id
  let destinationRelations := destination.definitions.filter
    (fun definition => definition.kind == .relation) |>.map DefinitionMetadata.id
  let sourceSemanticReferences := implementationSemanticReferences source
  let destinationSemanticReferences := implementationSemanticReferences destination
  validateSemanticTable declaration.id declaration.source "relation" .relation source.definitions
    destination.definitions sourceSemanticReferences destinationSemanticReferences relations
    destinationRelations canonical.relationMappings
    canonical.relationKnownGaps
  validateSemanticTable declaration.id declaration.source "capability" .capability source.definitions
    destination.definitions sourceSemanticReferences destinationSemanticReferences
    source.requiredCapabilities destination.requiredCapabilities
    canonical.capabilityMappings
    canonical.capabilityKnownGaps
  let expectedIndex := implementationLinkWitnessIndex declaration source destination
  let semantic := implementationLinkSemanticJson canonical sourceDomain destinationDomain
  let metadata := canonicalImplementationLinkJson canonical semantic
  match authoredWitness with
  | .incomplete index missing =>
      validateWitnessIndex declaration.id declaration.source expectedIndex index
      let obligation := (missing.mergeSort obligationLe).head?.getD .requiredCoverage
      throw (implementationLinkError (missingObligationError obligation) declaration.id
        declaration.source obligation.name)
  | .complete witness =>
      validateWitnessIndex declaration.id declaration.source expectedIndex witness.index
      pure {
        declaration := canonical
        sourceTarget := source
        destinationTarget := destination
        mapSetup := witness.mapSetup
        mapState := witness.mapState
        mapAction := witness.mapAction
        mapOutcome := witness.mapOutcome
        mapObservation := witness.mapObservation
        initialForward := witness.initialForward
        stepForward := witness.stepForward
        canonicalMetadata := metadata
        behaviorFingerprint := behaviorFingerprintOf semantic
      }

/-- Check one exact bounded forward simulation, returning no partial value on any typed failure. -/
def checkImplementationLink
    [BEq SourceSetup] [BEq SourceState] [BEq SourceAction] [BEq SourceOutcome]
    [BEq SourceObservation] [BEq DestinationSetup] [BEq DestinationState]
    [BEq DestinationAction] [BEq DestinationOutcome] [BEq DestinationObservation]
    (declaration : ImplementationLinkDeclaration
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation)
    (source : CheckedTarget SourceLawStatement SourceSetup SourceState SourceAction
      SourceOutcome SourceObservation)
    (destination : CheckedTarget DestinationLawStatement DestinationSetup DestinationState
      DestinationAction DestinationOutcome DestinationObservation)
    (witness : ImplementationLinkWitnessAuthoring declaration source destination) :
    Except ImplementationLinkError (CheckedImplementationLink SourceLawStatement
      DestinationLawStatement SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome DestinationObservation) :=
  match source.kernel.behaviorDomain, destination.kernel.behaviorDomain with
  | .complete sourceDomain, .complete destinationDomain =>
      checkImplementationLinkWithDomains declaration source destination sourceDomain destinationDomain
        witness
  | _, _ =>
      .error (implementationLinkError .incompleteSupportPartition declaration.id declaration.source
        "checked-target-behavior-domain")

def canonicalImplementationLinkErrorJson (linkError : ImplementationLinkError) : String :=
  "{\"kind\":" ++ quote linkError.kind.name ++
    ",\"implementationLinkId\":" ++ quote linkError.implementationLinkId.value ++
    ",\"sourcePath\":" ++ quote linkError.sourcePath ++
    ",\"offendingValue\":" ++ quote linkError.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      array (canonicalIds linkError.relatedDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++ "}"

/-- Recompute the canonical checked identity before a retained link is applied to a trace. -/
def CheckedImplementationLink.hasCanonicalIdentity
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup SourceState SourceAction SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationAction DestinationOutcome
      DestinationObservation) : Bool :=
  match checked.sourceTarget.kernel.behaviorDomain,
      checked.destinationTarget.kernel.behaviorDomain with
  | .complete sourceDomain, .complete destinationDomain =>
      let declaration := canonicalizeDeclaration checked.declaration sourceDomain destinationDomain
      let semantic := implementationLinkSemanticJson declaration sourceDomain destinationDomain
      checked.behaviorFingerprint == behaviorFingerprintOf semantic &&
        checked.canonicalMetadata == canonicalImplementationLinkJson declaration semantic
  | _, _ => false

end Umpire
