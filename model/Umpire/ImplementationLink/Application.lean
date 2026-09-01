import Umpire.ImplementationLink.Language
import Umpire.Observation.Evaluation

/-!
Total application of one checked Implementation Link to an already Evidence-backed source Model
Trace. Application replays the complete source trace through the retained source kernel before it
translates any value. Only `applied` exposes the complete destination trace; every failure exposes
one canonical diagnostic and no partial trace.

An explicitly checked observed-trace translation reuses the same envelope and positional Evidence
Links for values outside Target authority. Its result omits the authority proof and cannot be
confused with `AppliedImplementationLink`.
-/

namespace Umpire

/-- Application success and the four exhaustive non-success classes stay separate from Observation
and Property outcomes. -/
inductive ImplementationLinkStatus where
  | applied
  | invalid
  | unknown
  | conflict
  | unsupported
  deriving BEq, DecidableEq, Ord, Repr

def ImplementationLinkStatus.name : ImplementationLinkStatus → String
  | .applied => "applied"
  | .invalid => "invalid"
  | .unknown => "unknown"
  | .conflict => "conflict"
  | .unsupported => "unsupported"

/-- Exhaustive application failures for the bounded positional prototype. -/
inductive ImplementationLinkFailureKind where
  | staleSourceTarget
  | staleDestinationTarget
  | behaviorFingerprintDrift
  | sourceSetupMismatch
  | nonAuthoritativeSourceInitial
  | nonAuthoritativeSourceStep
  | invalidCoordinate
  | absentCoordinate
  | limitReached
  | duplicateCoordinate
  | contradictoryCoordinate
  | multipleMappings
  | evidenceLinkMismatch
  | knownGap
  | unsupportedVocabulary
  deriving BEq, DecidableEq, Ord, Repr

def ImplementationLinkFailureKind.name : ImplementationLinkFailureKind → String
  | .staleSourceTarget => "stale-source-target"
  | .staleDestinationTarget => "stale-destination-target"
  | .behaviorFingerprintDrift => "behavior-fingerprint-drift"
  | .sourceSetupMismatch => "source-setup-mismatch"
  | .nonAuthoritativeSourceInitial => "non-authoritative-source-initial"
  | .nonAuthoritativeSourceStep => "non-authoritative-source-step"
  | .invalidCoordinate => "invalid-coordinate"
  | .absentCoordinate => "absent-coordinate"
  | .limitReached => "limit-reached"
  | .duplicateCoordinate => "duplicate-coordinate"
  | .contradictoryCoordinate => "contradictory-coordinate"
  | .multipleMappings => "multiple-mappings"
  | .evidenceLinkMismatch => "evidence-link-mismatch"
  | .knownGap => "known-gap"
  | .unsupportedVocabulary => "unsupported-vocabulary"

/-- Each failure kind has exactly one status; there is no caller-selected classification. -/
def ImplementationLinkFailureKind.status : ImplementationLinkFailureKind → ImplementationLinkStatus
  | .staleSourceTarget
  | .staleDestinationTarget
  | .behaviorFingerprintDrift
  | .sourceSetupMismatch
  | .nonAuthoritativeSourceInitial
  | .nonAuthoritativeSourceStep
  | .invalidCoordinate => .invalid
  | .absentCoordinate
  | .limitReached => .unknown
  | .duplicateCoordinate
  | .contradictoryCoordinate
  | .multipleMappings
  | .evidenceLinkMismatch => .conflict
  | .knownGap
  | .unsupportedVocabulary => .unsupported

/-- Canonical failure provenance. Optional fields remain explicit so the identity binds absence too. -/
structure ImplementationLinkDiagnostic where
  implementationLinkId : DefinitionId
  implementationLinkBehaviorFingerprint : BehaviorFingerprint
  sourceTarget : ImplementationTargetReference
  destinationTarget : ImplementationTargetReference
  kind : ImplementationLinkFailureKind
  coordinate : Option ModelCoordinate := none
  relatedDefinitionIds : List DefinitionId := []
  sourceSetupBehaviorFingerprint : Option BehaviorFingerprint := none
  appliedLimit : Option Limit := none
  observedCount : Option Nat := none
  knownGapCode : Option DefinitionId := none
  knownGapReason : Option String := none
  unsupportedVocabularyKind : Option DefinitionKind := none
  evidenceLinkBehaviorFingerprint : Option BehaviorFingerprint := none
  identity : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

def ImplementationLinkDiagnostic.status
    (diagnostic : ImplementationLinkDiagnostic) : ImplementationLinkStatus :=
  diagnostic.kind.status

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort (fun left right => decide (left.value ≤ right.value)) |>.eraseDups

private def optionalJson (value : Option String) : String :=
  value.map quote |>.getD "null"

private def coordinateName : ModelCoordinate → String
  | .initialState => "initial-state"
  | .selectedAction step => "selected-action:" ++ toString step
  | .modelOutcome step => "model-outcome:" ++ toString step
  | .resultingState step => "resulting-state:" ++ toString step
  | .observation step position => "observation:" ++ toString step ++ ":" ++ toString position

private def targetReferenceIdentityJson (reference : ImplementationTargetReference) : String :=
  "{\"id\":" ++ quote reference.id.value ++
    ",\"kind\":" ++ quote reference.kind.name ++
    ",\"behaviorFingerprint\":" ++ quote reference.behaviorFingerprint.render ++ "}"

private def limitIdentityJson (limit : Limit) : String :=
  "{\"value\":" ++ toString limit.value ++ ",\"unit\":" ++ quote limit.unit.name ++ "}"

private def implementationLinkDiagnosticSemanticJson
    (implementationLinkId : DefinitionId)
    (implementationLinkBehaviorFingerprint : BehaviorFingerprint)
    (sourceTarget destinationTarget : ImplementationTargetReference)
    (kind : ImplementationLinkFailureKind)
    (coordinate : Option ModelCoordinate)
    (relatedDefinitionIds : List DefinitionId)
    (sourceSetupBehaviorFingerprint : Option BehaviorFingerprint)
    (appliedLimit : Option Limit)
    (observedCount : Option Nat)
    (knownGapCode : Option DefinitionId)
    (knownGapReason : Option String)
    (unsupportedVocabularyKind : Option DefinitionKind)
    (evidenceLinkBehaviorFingerprint : Option BehaviorFingerprint) : String :=
  "{\"implementationLinkId\":" ++ quote implementationLinkId.value ++
    ",\"implementationLinkBehaviorFingerprint\":" ++
      quote implementationLinkBehaviorFingerprint.render ++
    ",\"sourceTarget\":" ++ targetReferenceIdentityJson sourceTarget ++
    ",\"destinationTarget\":" ++ targetReferenceIdentityJson destinationTarget ++
    ",\"kind\":" ++ quote kind.name ++
    ",\"status\":" ++ quote kind.status.name ++
    ",\"coordinate\":" ++ optionalJson (coordinate.map coordinateName) ++
    ",\"relatedDefinitionIds\":" ++
      array (canonicalIds relatedDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++
    ",\"sourceSetupBehaviorFingerprint\":" ++
      optionalJson (sourceSetupBehaviorFingerprint.map BehaviorFingerprint.render) ++
    ",\"appliedLimit\":" ++ (appliedLimit.map limitIdentityJson |>.getD "null") ++
    ",\"observedCount\":" ++ (observedCount.map toString |>.getD "null") ++
    ",\"knownGapCode\":" ++ optionalJson (knownGapCode.map DefinitionId.value) ++
    ",\"knownGapReason\":" ++ optionalJson knownGapReason ++
    ",\"unsupportedVocabularyKind\":" ++
      optionalJson (unsupportedVocabularyKind.map DefinitionKind.name) ++
    ",\"evidenceLinkBehaviorFingerprint\":" ++
      optionalJson (evidenceLinkBehaviorFingerprint.map BehaviorFingerprint.render) ++ "}"

def canonicalImplementationLinkDiagnosticJson
    (diagnostic : ImplementationLinkDiagnostic) : String :=
  implementationLinkDiagnosticSemanticJson diagnostic.implementationLinkId
    diagnostic.implementationLinkBehaviorFingerprint diagnostic.sourceTarget
    diagnostic.destinationTarget diagnostic.kind diagnostic.coordinate
    diagnostic.relatedDefinitionIds diagnostic.sourceSetupBehaviorFingerprint
    diagnostic.appliedLimit diagnostic.observedCount diagnostic.knownGapCode
    diagnostic.knownGapReason diagnostic.unsupportedVocabularyKind
    diagnostic.evidenceLinkBehaviorFingerprint

/-- Whether a diagnostic still carries the identity of all its canonical provenance fields. -/
def ImplementationLinkDiagnostic.hasCanonicalIdentity
    (diagnostic : ImplementationLinkDiagnostic) : Bool :=
  diagnostic.identity == behaviorFingerprintOf (canonicalImplementationLinkDiagnosticJson diagnostic)

private def implementationLinkDiagnostic
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (kind : ImplementationLinkFailureKind)
    (coordinate : Option ModelCoordinate := none)
    (relatedDefinitionIds : List DefinitionId := [])
    (sourceSetupBehaviorFingerprint : Option BehaviorFingerprint := none)
    (appliedLimit : Option Limit := none)
    (observedCount : Option Nat := none)
    (knownGapCode : Option DefinitionId := none)
    (knownGapReason : Option String := none)
    (unsupportedVocabularyKind : Option DefinitionKind := none)
    (evidenceLinkBehaviorFingerprint : Option BehaviorFingerprint := none) :
    ImplementationLinkDiagnostic :=
  let relatedDefinitionIds := canonicalIds relatedDefinitionIds
  let sourceTarget := ImplementationTargetReference.ofTarget checked.sourceTarget
  let destinationTarget := ImplementationTargetReference.ofTarget checked.destinationTarget
  let semantic := implementationLinkDiagnosticSemanticJson checked.declaration.id
    checked.behaviorFingerprint sourceTarget destinationTarget kind coordinate relatedDefinitionIds
    sourceSetupBehaviorFingerprint appliedLimit observedCount knownGapCode knownGapReason
    unsupportedVocabularyKind evidenceLinkBehaviorFingerprint
  {
    implementationLinkId := checked.declaration.id
    implementationLinkBehaviorFingerprint := checked.behaviorFingerprint
    sourceTarget
    destinationTarget
    kind
    coordinate
    relatedDefinitionIds
    sourceSetupBehaviorFingerprint
    appliedLimit
    observedCount
    knownGapCode
    knownGapReason
    unsupportedVocabularyKind
    evidenceLinkBehaviorFingerprint
    identity := behaviorFingerprintOf semantic
  }

/-- A checked observed-trace translation can admit explicitly declared values outside Target
authority while retaining the checked Implementation Link's semantic correspondence. -/
structure ObservedTraceTranslationDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  observationMappings : List (ImplementationValueMapping ModelValue ModelValue)
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

inductive ObservedTraceTranslationErrorKind where
  | emptyDefinitionId
  | invalidDefinitionId
  | invalidVersion
  | behaviorFingerprintDrift
  | duplicateMapping
  | ambiguousMapping
  | incompatibleMapping
  | incompleteSupportPartition
  deriving BEq, DecidableEq, Ord, Repr

structure ObservedTraceTranslationError where
  kind : ObservedTraceTranslationErrorKind
  definitionId : DefinitionId
  relatedDefinitionIds : List DefinitionId := []
  deriving BEq, DecidableEq, Repr

private def observedMappingLe
    (left right : ImplementationValueMapping ModelValue ModelValue) : Bool :=
  decide (reprStr left.source < reprStr right.source) ||
    (left.source == right.source && decide (reprStr left.destination ≤ reprStr right.destination))

private def observedMappingJson
    (mapping : ImplementationValueMapping ModelValue ModelValue) : String :=
  "{\"source\":" ++ quote (reprStr mapping.source) ++
    ",\"destination\":" ++ quote (reprStr mapping.destination) ++ "}"

private def observedTraceTranslationSemanticJson
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (id : DefinitionId)
    (version : Nat)
    (observationMappings : List (ImplementationValueMapping ModelValue ModelValue)) : String :=
  "{\"definitionId\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"baseImplementationLinkId\":" ++ quote checked.declaration.id.value ++
    ",\"baseImplementationLinkBehaviorFingerprint\":" ++
      quote checked.behaviorFingerprint.render ++
    ",\"sourceTarget\":" ++ targetReferenceIdentityJson (.ofTarget checked.sourceTarget) ++
    ",\"destinationTarget\":" ++
      targetReferenceIdentityJson (.ofTarget checked.destinationTarget) ++
    ",\"observationMappings\":" ++
      array (observationMappings.map observedMappingJson) ++ "}"

private def canonicalObservedTraceTranslationJson
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (declaration : ObservedTraceTranslationDeclaration)
    (observationMappings : List (ImplementationValueMapping ModelValue ModelValue)) : String :=
  "{\"semantic\":" ++ observedTraceTranslationSemanticJson checked declaration.id
      declaration.version observationMappings ++
    ",\"source\":" ++ quote (reprStr declaration.source) ++
    ",\"documentation\":" ++ quote declaration.documentation ++ "}"

/-- Canonical observed-value extension of one already checked forward simulation. -/
structure CheckedObservedTraceTranslation
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue) where
  declaration : ObservedTraceTranslationDeclaration
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint

def CheckedObservedTraceTranslation.hasCanonicalIdentity
    (translation : CheckedObservedTraceTranslation checked) : Bool :=
  let mappings := translation.declaration.observationMappings
  translation.behaviorFingerprint == behaviorFingerprintOf
      (observedTraceTranslationSemanticJson checked translation.declaration.id
        translation.declaration.version mappings) &&
    translation.canonicalMetadata ==
      canonicalObservedTraceTranslationJson checked translation.declaration mappings

private def observedTranslationError
    (kind : ObservedTraceTranslationErrorKind)
    (declaration : ObservedTraceTranslationDeclaration)
    (relatedDefinitionIds : List DefinitionId := []) : ObservedTraceTranslationError := {
  kind
  definitionId := declaration.id
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

/-- Check an exact observed-value table against the semantic references of a checked link.
Values may be outside Target authority, but their Definition IDs may not invent a new meaning. -/
def checkObservedTraceTranslation
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (declaration : ObservedTraceTranslationDeclaration) :
    Except ObservedTraceTranslationError (CheckedObservedTraceTranslation checked) := do
  if declaration.id.value == "" then
    throw (observedTranslationError .emptyDefinitionId declaration)
  if !declaration.id.isNamespaced then
    throw (observedTranslationError .invalidDefinitionId declaration [declaration.id])
  if declaration.version == 0 then
    throw (observedTranslationError .invalidVersion declaration [declaration.id])
  if !checked.hasCanonicalIdentity then
    throw (observedTranslationError .behaviorFingerprintDrift declaration
      [checked.declaration.id])
  for mapping in declaration.observationMappings do
    if (declaration.observationMappings.filter fun other => other == mapping).length > 1 then
      throw (observedTranslationError .duplicateMapping declaration
        [mapping.source.definitionId, mapping.destination.definitionId])
    if (declaration.observationMappings.filter fun other =>
        other.source == mapping.source).length > 1 then
      throw (observedTranslationError .ambiguousMapping declaration
        [mapping.source.definitionId, mapping.destination.definitionId])
    if !(checked.declaration.observationMappings.any fun base =>
        base.source.definitionId == mapping.source.definitionId &&
          base.destination.definitionId == mapping.destination.definitionId) then
      throw (observedTranslationError .incompatibleMapping declaration
        [mapping.source.definitionId, mapping.destination.definitionId])
  for base in checked.declaration.observationMappings do
    if !(declaration.observationMappings.contains base) then
      throw (observedTranslationError .incompleteSupportPartition declaration
        [base.source.definitionId, base.destination.definitionId])
  let mappings := declaration.observationMappings.mergeSort observedMappingLe
  let checkedDeclaration := { declaration with observationMappings := mappings }
  pure {
    declaration := checkedDeclaration
    canonicalMetadata := canonicalObservedTraceTranslationJson checked checkedDeclaration mappings
    behaviorFingerprint := behaviorFingerprintOf
      (observedTraceTranslationSemanticJson checked checkedDeclaration.id
        checkedDeclaration.version mappings)
  }

/-- One destination fact retains its exact source coordinate, source fact, and Observation Evidence Link. -/
structure ImplementationLinkEvidenceLink where
  identity : BehaviorFingerprint
  implementationLinkId : DefinitionId
  implementationLinkBehaviorFingerprint : BehaviorFingerprint
  sourceTarget : ImplementationTargetReference
  destinationTarget : ImplementationTargetReference
  coordinate : ModelCoordinate
  sourceValue : ModelValue
  destinationValue : ModelValue
  sourceEvidenceLink : EvidenceLink
  sourceEvidenceLinkBehaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

private def evidenceLinkIdentityFor
    (implementationLinkId : DefinitionId)
    (implementationLinkBehaviorFingerprint : BehaviorFingerprint)
    (sourceTarget destinationTarget : ImplementationTargetReference)
    (coordinate : ModelCoordinate)
    (sourceValue destinationValue : ModelValue)
    (sourceEvidenceLink : EvidenceLink) : BehaviorFingerprint :=
  behaviorFingerprintOf <|
    "{\"implementationLinkId\":" ++ quote implementationLinkId.value ++
    ",\"implementationLinkBehaviorFingerprint\":" ++
      quote implementationLinkBehaviorFingerprint.render ++
    ",\"sourceTarget\":" ++ targetReferenceIdentityJson sourceTarget ++
    ",\"destinationTarget\":" ++ targetReferenceIdentityJson destinationTarget ++
    ",\"coordinate\":" ++ quote (coordinateName coordinate) ++
    ",\"sourceValue\":" ++ quote (reprStr sourceValue) ++
    ",\"destinationValue\":" ++ quote (reprStr destinationValue) ++
    ",\"sourceEvidenceLink\":" ++ quote (reprStr sourceEvidenceLink) ++ "}"

private def implementationLinkEvidenceLinkFor
    (implementationLinkId : DefinitionId)
    (implementationLinkBehaviorFingerprint : BehaviorFingerprint)
    (sourceTarget destinationTarget : ImplementationTargetReference)
    (coordinate : ModelCoordinate)
    (sourceValue destinationValue : ModelValue)
    (sourceEvidenceLink : EvidenceLink) : ImplementationLinkEvidenceLink := {
  identity := evidenceLinkIdentityFor implementationLinkId implementationLinkBehaviorFingerprint
    sourceTarget destinationTarget coordinate sourceValue destinationValue sourceEvidenceLink
  implementationLinkId
  implementationLinkBehaviorFingerprint
  sourceTarget
  destinationTarget
  coordinate
  sourceValue
  destinationValue
  sourceEvidenceLink
  sourceEvidenceLinkBehaviorFingerprint := behaviorFingerprintOf (reprStr sourceEvidenceLink)
}

private def implementationLinkEvidenceLink
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (coordinate : ModelCoordinate)
    (sourceValue destinationValue : ModelValue)
    (sourceEvidenceLink : EvidenceLink) : ImplementationLinkEvidenceLink :=
  implementationLinkEvidenceLinkFor checked.declaration.id checked.behaviorFingerprint
    (.ofTarget checked.sourceTarget) (.ofTarget checked.destinationTarget) coordinate
    sourceValue destinationValue sourceEvidenceLink

private def expectedCoordinates
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) : List ModelCoordinate :=
  .initialState :: (trace.steps.mapIdx fun index step =>
    let position := index + 1
    [.selectedAction position, .modelOutcome position, .resultingState position] ++
      step.observations.mapIdx fun observationIndex _ =>
        .observation position (observationIndex + 1)).flatten

private def modelValueAt
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (coordinate : ModelCoordinate) : Option ModelValue :=
  match coordinate with
  | .initialState => some trace.initialState
  | .selectedAction step => do
      if step == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        pure traceStep.selectedAction
  | .modelOutcome step => do
      if step == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        pure traceStep.modelOutcome
  | .resultingState step => do
      if step == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        pure traceStep.resultingState
  | .observation step position => do
      if step == 0 || position == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        traceStep.observations[position - 1]?

private def coordinateKind : ModelCoordinate → DefinitionKind
  | .initialState | .resultingState _ => .state
  | .selectedAction _ => .action
  | .modelOutcome _ => .outcome
  | .observation _ _ => .observation

private def evidenceLinkSetFingerprint (trace : EvidenceBackedTrace) : BehaviorFingerprint :=
  behaviorFingerprintOf (reprStr trace.evidenceLinks)

private def evidenceEnvelopeFailure?
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (trace : EvidenceBackedTrace) : Option ImplementationLinkDiagnostic :=
  match trace.evidenceLinks.find? fun evidenceLink =>
      (modelValueAt trace.trace evidenceLink.coordinate).isNone with
  | some evidenceLink => some <| implementationLinkDiagnostic checked .invalidCoordinate
      (some evidenceLink.coordinate) [evidenceLink.ruleId]
      (evidenceLinkBehaviorFingerprint := some (behaviorFingerprintOf (reprStr evidenceLink)))
  | none =>
      match trace.evidenceLinks.find? fun evidenceLink =>
          (trace.evidenceLinks.filter fun other =>
            other.coordinate == evidenceLink.coordinate).length > 1 with
      | some evidenceLink =>
          let matchingLinks := trace.evidenceLinks.filter fun other =>
            other.coordinate == evidenceLink.coordinate
          let kind := if matchingLinks.all fun other => other == evidenceLink then
            ImplementationLinkFailureKind.duplicateCoordinate
          else
            ImplementationLinkFailureKind.contradictoryCoordinate
          some <| implementationLinkDiagnostic checked kind (some evidenceLink.coordinate)
            (matchingLinks.map EvidenceLink.ruleId)
            (evidenceLinkBehaviorFingerprint := some (evidenceLinkSetFingerprint trace))
      | none =>
          match (expectedCoordinates trace.trace).find? fun coordinate =>
              !(trace.evidenceLinks.any fun evidenceLink => evidenceLink.coordinate == coordinate) with
          | some coordinate => some <| implementationLinkDiagnostic checked .absentCoordinate
              (some coordinate) (evidenceLinkBehaviorFingerprint :=
                some (evidenceLinkSetFingerprint trace))
          | none => none

private def supportedVocabularyKind : DefinitionKind → Bool
  | .state | .action | .outcome | .observation | .relation | .capability => true
  | _ => false

private def validateSemanticMapping
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (meaning : MeaningProvision) : Except ImplementationLinkDiagnostic Unit := do
  let mappings := match meaning.kind with
    | .relation => checked.declaration.relationMappings
    | .capability => checked.declaration.capabilityMappings
    | _ => []
  let gaps := match meaning.kind with
    | .relation => checked.declaration.relationKnownGaps
    | .capability => checked.declaration.capabilityKnownGaps
    | _ => []
  let matchingMappings := mappings.filter fun mapping => mapping.source.id == meaning.definitionId
  let matchingGaps := gaps.filter fun gap => gap.source == meaning.definitionId
  match matchingMappings, matchingGaps with
  | [_], [] => pure ()
  | [], [gap] => throw (implementationLinkDiagnostic checked .knownGap
      (relatedDefinitionIds := [meaning.definitionId, gap.code])
      (knownGapCode := some gap.code) (knownGapReason := some gap.reason))
  | [], [] => throw (implementationLinkDiagnostic checked .unsupportedVocabulary
      (relatedDefinitionIds := [meaning.definitionId])
      (unsupportedVocabularyKind := some meaning.kind))
  | _, _ => throw (implementationLinkDiagnostic checked .multipleMappings
      (relatedDefinitionIds := meaning.definitionId ::
        matchingMappings.map (fun mapping => mapping.destination.id) ++
        matchingGaps.map ImplementationLinkKnownGap.code))

private def validateVocabulary
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (trace : EvidenceBackedTrace) : Except ImplementationLinkDiagnostic Unit := do
  let sourceMeanings := (ObservationCheckContext.ofTarget checked.sourceTarget []).meanings
  for meaning in trace.vocabulary do
    if !supportedVocabularyKind meaning.kind then
      throw (implementationLinkDiagnostic checked .unsupportedVocabulary
        (relatedDefinitionIds := [meaning.definitionId])
        (unsupportedVocabularyKind := some meaning.kind))
    if !(sourceMeanings.any fun sourceMeaning => sourceMeaning == meaning) then
      throw (implementationLinkDiagnostic checked .behaviorFingerprintDrift
        (relatedDefinitionIds := [meaning.definitionId]))
    if meaning.kind == .relation || meaning.kind == .capability then
      validateSemanticMapping checked meaning
  for coordinate in expectedCoordinates trace.trace do
    let value ← match modelValueAt trace.trace coordinate with
      | some value => pure value
      | none => throw <| implementationLinkDiagnostic checked .invalidCoordinate (some coordinate)
    let kind := coordinateKind coordinate
    if !(trace.vocabulary.any fun meaning =>
        meaning.definitionId == value.definitionId && meaning.kind == kind) then
      throw <| implementationLinkDiagnostic checked .behaviorFingerprintDrift
        (some coordinate) [value.definitionId]

private def mappedSetup
    [BEq SourceSetup] [BEq DestinationSetup]
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (sourceSetup : SourceSetup) : Except ImplementationLinkDiagnostic DestinationSetup := do
  let sourceDomain ← match checked.sourceTarget.kernel.behaviorDomain with
    | .complete domain => pure domain
    | _ => throw <| implementationLinkDiagnostic checked .behaviorFingerprintDrift
  let sourceSetupBehaviorFingerprint :=
    behaviorFingerprintOf (sourceDomain.encodeSetup sourceSetup)
  if !sourceDomain.setups.contains sourceSetup then
    throw (implementationLinkDiagnostic checked .sourceSetupMismatch
      (sourceSetupBehaviorFingerprint := some sourceSetupBehaviorFingerprint))
  let mappings := checked.declaration.setupMappings.filter fun mapping =>
    mapping.source == sourceSetup
  let gaps := checked.declaration.setupKnownGaps.filter fun gap => gap.source == sourceSetup
  match mappings, gaps with
  | [mapping], [] =>
      let destination := checked.mapSetup sourceSetup
      if mapping.destination != destination then
        throw (implementationLinkDiagnostic checked .sourceSetupMismatch
          (sourceSetupBehaviorFingerprint := some sourceSetupBehaviorFingerprint))
      pure destination
  | [], [gap] => throw (implementationLinkDiagnostic checked .knownGap
      (relatedDefinitionIds := [gap.code])
      (sourceSetupBehaviorFingerprint := some sourceSetupBehaviorFingerprint)
      (knownGapCode := some gap.code) (knownGapReason := some gap.reason))
  | [], [] => throw (implementationLinkDiagnostic checked .sourceSetupMismatch
      (sourceSetupBehaviorFingerprint := some sourceSetupBehaviorFingerprint))
  | _, _ => throw (implementationLinkDiagnostic checked .multipleMappings
      (sourceSetupBehaviorFingerprint := some sourceSetupBehaviorFingerprint))

private abbrev AdmittedSourceSteps
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (state : ModelValue)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue)) :=
  PLift (AuthoritativeTraceSteps checked.sourceTarget.kernel state steps)

private abbrev ExactListMember (value : Value) (values : List Value) :=
  PLift (value ∈ values)

private def exactListMember? [DecidableEq Value]
    (value : Value) : (values : List Value) → Option (ExactListMember value values)
  | [] => none
  | first :: rest =>
      if equal : value = first then
          some ⟨by simp [equal]⟩
      else
        match exactListMember? value rest with
        | some member => some ⟨List.Mem.tail first member.down⟩
        | none => none

private def admittedSourceSteps
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (state : ModelValue)
    (position : Nat)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue)) :
    Except ImplementationLinkDiagnostic (AdmittedSourceSteps checked state steps) :=
  match steps with
  | [] => .ok ⟨True.intro⟩
  | step :: rest =>
      let result : TransitionResult ModelValue ModelValue ModelValue := {
        modelOutcome := step.modelOutcome
        resultingState := step.resultingState
        observations := step.observations
      }
      match exactListMember? result
          (checked.sourceTarget.kernel.steps state step.selectedAction) with
      | some admitted =>
        match admittedSourceSteps checked step.resultingState (position + 1) rest with
        | .ok admittedRest => .ok ⟨⟨checked.sourceTarget.kernel.stepSound
            state step.selectedAction result admitted.down, admittedRest.down⟩⟩
        | .error failure => .error failure
      | none =>
        .error <| implementationLinkDiagnostic checked .nonAuthoritativeSourceStep
          (some (.selectedAction position))
          (step.selectedAction.definitionId :: step.modelOutcome.definitionId ::
            step.resultingState.definitionId ::
            step.observations.map ModelValue.definitionId)

private abbrev AdmittedSourceTrace
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (sourceSetup : SourceSetup)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :=
  PLift (AuthoritativeModelTrace checked.sourceTarget.kernel sourceSetup trace)

private def admittedSourceTrace
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (sourceSetup : SourceSetup)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Except ImplementationLinkDiagnostic (AdmittedSourceTrace checked sourceSetup trace) := do
  match exactListMember? trace.initialState
      (checked.sourceTarget.kernel.initialStates sourceSetup) with
  | some admitted =>
    let admittedSteps ← admittedSourceSteps checked trace.initialState 1 trace.steps
    pure ⟨{
      initial := checked.sourceTarget.kernel.initialSound sourceSetup trace.initialState admitted.down
      steps := admittedSteps.down
    }⟩
  | none =>
    throw <| implementationLinkDiagnostic checked .nonAuthoritativeSourceInitial
      (some .initialState) [trace.initialState.definitionId]

private def mappedValue
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (coordinate : ModelCoordinate)
    (sourceValue : ModelValue)
    (mappings : List (ImplementationValueMapping ModelValue ModelValue))
    (knownGaps : List (ImplementationLinkKnownGap ModelValue))
    (mapValue : ModelValue → ModelValue) : Except ImplementationLinkDiagnostic ModelValue := do
  let matchingMappings := mappings.filter fun mapping => mapping.source == sourceValue
  let matchingGaps := knownGaps.filter fun gap => gap.source == sourceValue
  match matchingMappings, matchingGaps with
  | [mapping], [] =>
      let destination := mapValue sourceValue
      if mapping.destination == destination then
        pure destination
      else
        throw (implementationLinkDiagnostic checked .multipleMappings (some coordinate)
          [sourceValue.definitionId, mapping.destination.definitionId, destination.definitionId]
        )
  | [], [gap] => throw (implementationLinkDiagnostic checked .knownGap (some coordinate)
      [sourceValue.definitionId, gap.code]
      (knownGapCode := some gap.code) (knownGapReason := some gap.reason))
  | [], [] => throw (implementationLinkDiagnostic checked .absentCoordinate (some coordinate)
      [sourceValue.definitionId])
  | _, _ => throw (implementationLinkDiagnostic checked .multipleMappings (some coordinate)
      (sourceValue.definitionId ::
        matchingMappings.map (fun mapping => mapping.destination.definitionId) ++
        matchingGaps.map ImplementationLinkKnownGap.code))

private def mappedValueAt
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (coordinate : ModelCoordinate)
    (sourceValue : ModelValue) : Except ImplementationLinkDiagnostic ModelValue :=
  match coordinate with
  | .initialState | .resultingState _ => mappedValue checked coordinate sourceValue
      checked.declaration.stateMappings checked.declaration.stateKnownGaps checked.mapState
  | .selectedAction _ => mappedValue checked coordinate sourceValue
      checked.declaration.actionMappings checked.declaration.actionKnownGaps checked.mapAction
  | .modelOutcome _ => mappedValue checked coordinate sourceValue
      checked.declaration.outcomeMappings checked.declaration.outcomeKnownGaps checked.mapOutcome
  | .observation _ _ => mappedValue checked coordinate sourceValue
      checked.declaration.observationMappings checked.declaration.observationKnownGaps
      checked.mapObservation

private def CheckedImplementationLink.translateStep
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) :
    ModelTraceStep ModelValue ModelValue ModelValue ModelValue := {
  selectedAction := checked.mapAction step.selectedAction
  modelOutcome := checked.mapOutcome step.modelOutcome
  resultingState := checked.mapState step.resultingState
  observations := step.observations.map checked.mapObservation
}

private def CheckedImplementationLink.translateTrace
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := checked.mapState trace.initialState
  steps := trace.steps.map checked.translateStep
}

private theorem CheckedImplementationLink.stepsForward
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (state : ModelValue)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (admitted : AuthoritativeTraceSteps checked.sourceTarget.kernel state steps) :
    AuthoritativeTraceSteps checked.destinationTarget.kernel (checked.mapState state)
      (steps.map checked.translateStep) := by
  induction steps generalizing state with
  | nil => trivial
  | cons step rest induction =>
      exact ⟨checked.stepForward state step.selectedAction {
          modelOutcome := step.modelOutcome
          resultingState := step.resultingState
          observations := step.observations
        } admitted.1,
        induction step.resultingState admitted.2⟩

private theorem CheckedImplementationLink.traceForward
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (sourceSetup : SourceSetup)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (admitted : AuthoritativeModelTrace checked.sourceTarget.kernel sourceSetup trace) :
    AuthoritativeModelTrace checked.destinationTarget.kernel (checked.mapSetup sourceSetup)
      (checked.translateTrace trace) := {
  initial := checked.initialForward sourceSetup trace.initialState admitted.initial
  steps := checked.stepsForward trace.initialState trace.steps admitted.steps
}

private def buildImplementationLinkEvidenceLinksWith
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (sourceTrace : EvidenceBackedTrace)
    (destinationTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (mapValue : ModelCoordinate → ModelValue →
      Except ImplementationLinkDiagnostic ModelValue)
    (makeLink : ModelCoordinate → ModelValue → ModelValue → EvidenceLink →
      ImplementationLinkEvidenceLink) :
    Except ImplementationLinkDiagnostic (List ImplementationLinkEvidenceLink) := do
  let mut links := []
  for coordinate in expectedCoordinates sourceTrace.trace do
    let sourceValue ← match modelValueAt sourceTrace.trace coordinate with
      | some value => pure value
      | none => throw <| implementationLinkDiagnostic checked .invalidCoordinate (some coordinate)
    let sourceEvidenceLink ← match sourceTrace.evidenceLinks.find? fun evidenceLink =>
        evidenceLink.coordinate == coordinate with
      | some evidenceLink => pure evidenceLink
      | none => throw (implementationLinkDiagnostic checked .absentCoordinate (some coordinate)
          [sourceValue.definitionId])
    let destinationValue ← mapValue coordinate sourceValue
    match modelValueAt destinationTrace coordinate with
    | some actualDestination =>
        if actualDestination != destinationValue then
          throw <| implementationLinkDiagnostic checked .evidenceLinkMismatch (some coordinate)
            [sourceValue.definitionId, destinationValue.definitionId,
              actualDestination.definitionId]
            (evidenceLinkBehaviorFingerprint :=
              some (behaviorFingerprintOf (reprStr sourceEvidenceLink)))
    | none => throw <| implementationLinkDiagnostic checked .invalidCoordinate (some coordinate)
    links := links ++ [makeLink coordinate sourceValue destinationValue sourceEvidenceLink]
  pure links

private def buildImplementationLinkEvidenceLinks
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (sourceTrace : EvidenceBackedTrace)
    (destinationTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Except ImplementationLinkDiagnostic (List ImplementationLinkEvidenceLink) :=
  buildImplementationLinkEvidenceLinksWith checked sourceTrace destinationTrace
    (mappedValueAt checked) (implementationLinkEvidenceLink checked)

/-- Complete successful output, indexed by the exact checked link and carrying destination authority. -/
structure AppliedImplementationLink
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue) where
  sourceTraceId : String
  sourceSetup : SourceSetup
  destinationSetup : DestinationSetup
  trace : ModelTrace ModelValue ModelValue ModelValue ModelValue
  evidenceLinks : List ImplementationLinkEvidenceLink
  authoritative : AuthoritativeModelTrace checked.destinationTarget.kernel destinationSetup trace

/-- A non-success constructor cannot carry a destination Model Trace. -/
inductive ImplementationLinkResult
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue) where
  | applied (application : AppliedImplementationLink checked)
  | invalid (diagnostic : ImplementationLinkDiagnostic)
  | unknown (diagnostic : ImplementationLinkDiagnostic)
  | conflict (diagnostic : ImplementationLinkDiagnostic)
  | unsupported (diagnostic : ImplementationLinkDiagnostic)

def ImplementationLinkResult.status : ImplementationLinkResult checked → ImplementationLinkStatus
  | .applied _ => .applied
  | .invalid _ => .invalid
  | .unknown _ => .unknown
  | .conflict _ => .conflict
  | .unsupported _ => .unsupported

def ImplementationLinkResult.diagnostic? :
    ImplementationLinkResult checked → Option ImplementationLinkDiagnostic
  | .applied _ => none
  | .invalid diagnostic
  | .unknown diagnostic
  | .conflict diagnostic
  | .unsupported diagnostic => some diagnostic

/-- The only accessor for a destination trace returns `none` for every non-success constructor. -/
def ImplementationLinkResult.applied? :
    ImplementationLinkResult checked → Option (AppliedImplementationLink checked)
  | .applied application => some application
  | _ => none

private def resultOfDiagnostic
    (diagnostic : ImplementationLinkDiagnostic) : ImplementationLinkResult checked :=
  match diagnostic.kind.status with
  | .invalid => .invalid diagnostic
  | .unknown => .unknown diagnostic
  | .conflict => .conflict diagnostic
  | .unsupported => .unsupported diagnostic
  | .applied => .invalid diagnostic

private def applyCheckedImplementationLink
    [BEq SourceSetup] [BEq DestinationSetup]
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (sourceSetup : SourceSetup)
    (evidenceBackedTrace : EvidenceBackedTrace) :
    Except ImplementationLinkDiagnostic (AppliedImplementationLink checked) := do
  let sourceReference := ImplementationTargetReference.ofTarget checked.sourceTarget
  let destinationReference := ImplementationTargetReference.ofTarget checked.destinationTarget
  if checked.declaration.sourceTarget.id != sourceReference.id ||
      checked.declaration.sourceTarget.kind != .target then
    throw (implementationLinkDiagnostic checked .staleSourceTarget
      (relatedDefinitionIds := [checked.declaration.sourceTarget.id, sourceReference.id]))
  if checked.declaration.destinationTarget.id != destinationReference.id ||
      checked.declaration.destinationTarget.kind != .target then
    throw (implementationLinkDiagnostic checked .staleDestinationTarget
      (relatedDefinitionIds := [checked.declaration.destinationTarget.id, destinationReference.id]))
  if checked.declaration.sourceTarget.behaviorFingerprint != sourceReference.behaviorFingerprint ||
      checked.declaration.destinationTarget.behaviorFingerprint !=
        destinationReference.behaviorFingerprint ||
      !checked.hasCanonicalIdentity then
    throw (implementationLinkDiagnostic checked .behaviorFingerprintDrift
      (relatedDefinitionIds := [sourceReference.id, destinationReference.id]))
  match evidenceEnvelopeFailure? checked evidenceBackedTrace with
  | some failure => throw failure
  | none => pure ()
  let _ ← mappedSetup checked sourceSetup
  let sourceAuthority ← admittedSourceTrace checked sourceSetup evidenceBackedTrace.trace
  validateVocabulary checked evidenceBackedTrace
  if evidenceBackedTrace.trace.steps.length > checked.declaration.applicationLimit.value then
    throw (implementationLinkDiagnostic checked .limitReached
      (some (.selectedAction (checked.declaration.applicationLimit.value + 1)))
      (appliedLimit := some checked.declaration.applicationLimit)
      (observedCount := some evidenceBackedTrace.trace.steps.length))
  let destinationTrace := checked.translateTrace evidenceBackedTrace.trace
  let evidenceLinks ← buildImplementationLinkEvidenceLinks checked evidenceBackedTrace destinationTrace
  pure {
    sourceTraceId := evidenceBackedTrace.traceId
    sourceSetup
    destinationSetup := checked.mapSetup sourceSetup
    trace := destinationTrace
    evidenceLinks
    authoritative := checked.traceForward sourceSetup evidenceBackedTrace.trace sourceAuthority.down
  }

/-- Replay, validate, and translate one complete Evidence-backed source Model Trace. -/
def applyImplementationLink
    [BEq SourceSetup] [BEq DestinationSetup]
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (sourceSetup : SourceSetup)
    (evidenceBackedTrace : EvidenceBackedTrace) : ImplementationLinkResult checked :=
  match applyCheckedImplementationLink checked sourceSetup evidenceBackedTrace with
  | .ok application => .applied application
  | .error failure => resultOfDiagnostic failure

private def observedDiagnosticFrom
    (translation : CheckedObservedTraceTranslation checked)
    (diagnostic : ImplementationLinkDiagnostic) : ImplementationLinkDiagnostic :=
  let canonicalFields : ImplementationLinkDiagnostic := {
    diagnostic with
    implementationLinkId := translation.declaration.id
    implementationLinkBehaviorFingerprint := translation.behaviorFingerprint
    identity := behaviorFingerprintOf ""
  }
  { canonicalFields with
    identity := behaviorFingerprintOf (canonicalImplementationLinkDiagnosticJson canonicalFields) }

private def observedImplementationLinkDiagnostic
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked)
    (kind : ImplementationLinkFailureKind)
    (coordinate : Option ModelCoordinate := none)
    (relatedDefinitionIds : List DefinitionId := [])
    (sourceSetupBehaviorFingerprint : Option BehaviorFingerprint := none)
    (appliedLimit : Option Limit := none)
    (observedCount : Option Nat := none)
    (evidenceLinkBehaviorFingerprint : Option BehaviorFingerprint := none) :
    ImplementationLinkDiagnostic :=
  observedDiagnosticFrom translation <| implementationLinkDiagnostic checked kind coordinate
    relatedDefinitionIds sourceSetupBehaviorFingerprint appliedLimit observedCount
    (evidenceLinkBehaviorFingerprint := evidenceLinkBehaviorFingerprint)

private def observedMappedValueAt
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked)
    (coordinate : ModelCoordinate)
    (sourceValue : ModelValue) : Except ImplementationLinkDiagnostic ModelValue :=
  match coordinate with
  | .observation _ _ =>
      match translation.declaration.observationMappings.filter fun mapping =>
          mapping.source == sourceValue with
      | [mapping] => pure mapping.destination
      | [] => throw <| observedImplementationLinkDiagnostic checked translation .absentCoordinate
          (some coordinate) [sourceValue.definitionId]
      | mappings => throw <| observedImplementationLinkDiagnostic checked translation .multipleMappings
          (some coordinate) (sourceValue.definitionId ::
            mappings.map (fun mapping => mapping.destination.definitionId))
  | _ =>
      match mappedValueAt checked coordinate sourceValue with
      | .ok value => pure value
      | .error diagnostic => throw (observedDiagnosticFrom translation diagnostic)

private def translateObservedValues
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked)
    (step position : Nat) : List ModelValue →
    Except ImplementationLinkDiagnostic (List ModelValue)
  | [] => pure []
  | value :: rest => do
      let destination ← observedMappedValueAt checked translation
        (.observation step position) value
      let destinations ← translateObservedValues checked translation step (position + 1) rest
      pure (destination :: destinations)

private def translateObservedSteps
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked)
    (position : Nat) : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue) →
    Except ImplementationLinkDiagnostic
      (List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
  | [] => pure []
  | step :: rest => do
      let selectedAction ← observedMappedValueAt checked translation
        (.selectedAction position) step.selectedAction
      let modelOutcome ← observedMappedValueAt checked translation
        (.modelOutcome position) step.modelOutcome
      let resultingState ← observedMappedValueAt checked translation
        (.resultingState position) step.resultingState
      let observations ← translateObservedValues checked translation position 1 step.observations
      let translatedRest ← translateObservedSteps checked translation (position + 1) rest
      pure ({ selectedAction, modelOutcome, resultingState, observations } :: translatedRest)

private def translateObservedTrace
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Except ImplementationLinkDiagnostic
      (ModelTrace ModelValue ModelValue ModelValue ModelValue) := do
  let initialState ← observedMappedValueAt checked translation .initialState trace.initialState
  let steps ← translateObservedSteps checked translation 1 trace.steps
  pure { initialState, steps }

/-- Successful observed translation retains the full Evidence Link envelope and intentionally
contains no proof that the source or destination trace is Target-authoritative. -/
structure TranslatedObservedTrace
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked) where
  sourceTraceId : String
  sourceSetup : SourceSetup
  destinationSetup : DestinationSetup
  trace : ModelTrace ModelValue ModelValue ModelValue ModelValue
  evidenceLinks : List ImplementationLinkEvidenceLink

def TranslatedObservedTrace.hasAuthorityClaim
    (_ : TranslatedObservedTrace checked translation) : Bool := false

inductive ObservedTraceTranslationResult
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked) where
  | translated (result : TranslatedObservedTrace checked translation)
  | invalid (diagnostic : ImplementationLinkDiagnostic)
  | unknown (diagnostic : ImplementationLinkDiagnostic)
  | conflict (diagnostic : ImplementationLinkDiagnostic)
  | unsupported (diagnostic : ImplementationLinkDiagnostic)

def ObservedTraceTranslationResult.status :
    ObservedTraceTranslationResult checked translation → ImplementationLinkStatus
  | .translated _ => .applied
  | .invalid _ => .invalid
  | .unknown _ => .unknown
  | .conflict _ => .conflict
  | .unsupported _ => .unsupported

def ObservedTraceTranslationResult.diagnostic? :
    ObservedTraceTranslationResult checked translation → Option ImplementationLinkDiagnostic
  | .translated _ => none
  | .invalid diagnostic
  | .unknown diagnostic
  | .conflict diagnostic
  | .unsupported diagnostic => some diagnostic

def ObservedTraceTranslationResult.translated? :
    ObservedTraceTranslationResult checked translation →
      Option (TranslatedObservedTrace checked translation)
  | .translated result => some result
  | _ => none

private def observedResultOfDiagnostic
    (diagnostic : ImplementationLinkDiagnostic) :
    ObservedTraceTranslationResult checked translation :=
  match diagnostic.kind.status with
  | .invalid => .invalid diagnostic
  | .unknown => .unknown diagnostic
  | .conflict => .conflict diagnostic
  | .unsupported => .unsupported diagnostic
  | .applied => .invalid diagnostic

private def applyCheckedObservedTraceTranslation
    [BEq SourceSetup] [BEq DestinationSetup]
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked)
    (sourceSetup : SourceSetup)
    (evidenceBackedTrace : EvidenceBackedTrace) :
    Except ImplementationLinkDiagnostic (TranslatedObservedTrace checked translation) := do
  if !checked.hasCanonicalIdentity || !translation.hasCanonicalIdentity then
    throw (observedImplementationLinkDiagnostic checked translation .behaviorFingerprintDrift
      (relatedDefinitionIds := [checked.declaration.id, translation.declaration.id]))
  match evidenceEnvelopeFailure? checked evidenceBackedTrace with
  | some failure => throw (observedDiagnosticFrom translation failure)
  | none => pure ()
  let destinationSetup ← match mappedSetup checked sourceSetup with
    | .ok setup => pure setup
    | .error diagnostic => throw (observedDiagnosticFrom translation diagnostic)
  match validateVocabulary checked evidenceBackedTrace with
  | .ok _ => pure ()
  | .error diagnostic => throw (observedDiagnosticFrom translation diagnostic)
  if evidenceBackedTrace.trace.steps.length > checked.declaration.applicationLimit.value then
    throw (observedImplementationLinkDiagnostic checked translation .limitReached
      (some (.selectedAction (checked.declaration.applicationLimit.value + 1)))
      (appliedLimit := some checked.declaration.applicationLimit)
      (observedCount := some evidenceBackedTrace.trace.steps.length))
  let destinationTrace ← translateObservedTrace checked translation evidenceBackedTrace.trace
  let evidenceLinks ← match buildImplementationLinkEvidenceLinksWith checked evidenceBackedTrace
      destinationTrace (observedMappedValueAt checked translation)
      (implementationLinkEvidenceLinkFor translation.declaration.id
        translation.behaviorFingerprint (.ofTarget checked.sourceTarget)
        (.ofTarget checked.destinationTarget)) with
    | .ok links => pure links
    | .error diagnostic => throw (observedDiagnosticFrom translation diagnostic)
  pure {
    sourceTraceId := evidenceBackedTrace.traceId
    sourceSetup
    destinationSetup
    trace := destinationTrace
    evidenceLinks
  }

/-- Validate and translate one complete observed trace without asserting Target conformance. -/
def applyObservedTraceTranslation
    [BEq SourceSetup] [BEq DestinationSetup]
    {checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue}
    (translation : CheckedObservedTraceTranslation checked)
    (sourceSetup : SourceSetup)
    (evidenceBackedTrace : EvidenceBackedTrace) :
    ObservedTraceTranslationResult checked translation :=
  match applyCheckedObservedTraceTranslation checked translation sourceSetup evidenceBackedTrace with
  | .ok result => .translated result
  | .error diagnostic => observedResultOfDiagnostic diagnostic

end Umpire
