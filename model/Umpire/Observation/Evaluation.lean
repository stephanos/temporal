import Umpire.Observation.Language
import Umpire.SemanticInventory.Types

/-!
Pure Observation Evaluation of bounded synthetic Evidence. The boundary consumes a complete
checked plan and a finite typed bundle, then either returns one fully derived Model Trace or one
typed diagnostic. Raw Evidence is used only while evaluating the bundle and is absent from every
successful result. This layer establishes Model Facts; it does not perform Run Evaluation or Claim
Assessment.
-/

namespace Umpire

inductive EvidenceValue where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  deriving BEq, DecidableEq, Inhabited, Repr

def EvidenceValue.valueType : EvidenceValue → ObservationValueType
  | .text _ => .text
  | .natural _ => .natural
  | .boolean _ => .boolean

def EvidenceValue.render : EvidenceValue → String
  | .text value => value
  | .natural value => toString value
  | .boolean true => "true"
  | .boolean false => "false"

structure EvidenceFieldValue where
  field : DefinitionId
  value : EvidenceValue
  digestPolicy : Option DefinitionId := none
  reportedDigestToken : Option String := none
  deriving BEq, DecidableEq, Repr

structure EvidenceBindingFact where
  binding : DefinitionId
  value : EvidenceValue
  deriving BEq, DecidableEq, Repr

structure EvidenceOrigin where
  source : DefinitionId
  ordinal : Nat
  deriving BEq, DecidableEq, Repr

/-- One finite typed synthetic record. Its raw values never cross the Observation Evaluation boundary. -/
structure SyntheticEvidenceRecord where
  id : DefinitionId
  profile : DefinitionId
  profileVersion : Nat
  kind : DefinitionId
  sequence : Nat
  origin : Option EvidenceOrigin := none
  causalParents : List DefinitionId := []
  fields : List EvidenceFieldValue
  bindingFacts : List EvidenceBindingFact := []
  faultTarget : Option DefinitionId := none
  deriving BEq, DecidableEq, Repr

structure EvidenceClosureFact where
  kind : DefinitionId
  lastSequence : Nat
  source : Option DefinitionId := none
  recordCount : Option Nat := none
  byteCount : Option Nat := none
  deriving BEq, DecidableEq, Repr

structure CompatibleInterpretation where
  id : DefinitionId
  evidenceIdentities : List DefinitionId
  deriving BEq, DecidableEq, Repr

structure EvidenceGap where
  code : DefinitionId
  relatedDefinitionIds : List DefinitionId := []
  deriving BEq, DecidableEq, Repr

/-- The intentionally lossy Known Gap mapping admitted by Observation Evaluation. -/
def EvidenceGap.knownGapAdmissionMapping : KnownGapCarryMapping :=
  .observationAdmission

/-- Complete synthetic input envelope. Alternatives are preserved as data instead of selected. -/
structure EvidenceBundle where
  profile : DefinitionId
  profileVersion : Nat
  records : List SyntheticEvidenceRecord
  closures : List EvidenceClosureFact
  compatibleAlternatives : List CompatibleInterpretation := []
  missingDiscriminator : Option DefinitionId := none
  knownGaps : List EvidenceGap := []
  sourceClosed : Bool := true
  closedFieldKinds : List DefinitionId := []
  deriving BEq, DecidableEq, Repr

/-- The four exhaustive outcomes of Observation Evaluation. -/
inductive ObservationStatus where
  | accepted
  | unknown
  | conflict
  | unsupported
  deriving BEq, DecidableEq, Ord, Repr

def ObservationStatus.name : ObservationStatus → String
  | .accepted => "accepted"
  | .unknown => "unknown"
  | .conflict => "conflict"
  | .unsupported => "unsupported"

/-- Canonical documentation and exact constructor matchers for Observation outcomes. -/
def ObservationStatus.constructorClassifiers :
    List (OutcomeConstructorClassifier ObservationStatus) := [
  .ofValue .accepted {
    name := "accepted"
    description := "Observation Evaluation produced one Evidence-backed Model Trace."
  },
  .ofValue .unknown {
    name := "unknown"
    description := "Observation Evaluation could not decide from the available Evidence."
  },
  .ofValue .conflict {
    name := "conflict"
    description := "Observation Evaluation found contradictory Evidence."
  },
  .ofValue .unsupported {
    name := "unsupported"
    description := "Observation Evaluation does not support the supplied Evidence vocabulary."
  }
]

/-- Every Observation outcome matches exactly one descriptor. -/
theorem ObservationStatus.constructorClassifiers_exactlyOne :
    OutcomeConstructorClassifiers.ExactlyOne ObservationStatus.constructorClassifiers := by
  intro status
  cases status <;> rfl

inductive ObservationFailureKind where
  | emptyEvidence
  | evidenceBoundExhausted
  | knownGap
  | missingInitialState
  | missingClosure
  | sequenceGap
  | missingCausalParent
  | normalizationFailure
  | unresolvedBinding
  | incomparableOrdering
  | profileMismatch
  | profileVersionMismatch
  | kindMismatch
  | fieldMismatch
  | duplicateEvidenceIdentity
  | contradictoryFact
  | contradictoryBinding
  | contradictoryOrder
  | misdirectedFaultReceipt
  | compatibleAlternatives
  | zeroUsableInterpretations
  | absentModelCoordinate
  | duplicateModelCoordinate
  | extraModelCoordinate
  | inconsistentEvidenceLink
  | unconsumedReference
  | missingClosureSupport
  | missingOrderSupport
  | rawValueLeakage
  | redactedValueLeakage
  | rejectedValueLeakage
  | rejectedFieldPresent
  | digestPolicyMismatch
  | digestCollision
  | disallowedRawMaterial
  deriving BEq, DecidableEq, Ord, Repr

def ObservationFailureKind.status : ObservationFailureKind → ObservationStatus
  | .profileMismatch | .profileVersionMismatch | .kindMismatch | .fieldMismatch |
      .rawValueLeakage | .redactedValueLeakage | .rejectedValueLeakage |
      .rejectedFieldPresent | .digestPolicyMismatch | .disallowedRawMaterial => .unsupported
  | .duplicateEvidenceIdentity | .contradictoryFact | .contradictoryBinding |
      .contradictoryOrder | .misdirectedFaultReceipt | .duplicateModelCoordinate |
      .extraModelCoordinate | .inconsistentEvidenceLink | .digestCollision => .conflict
  | _ => .unknown

structure ObservationDiagnostic where
  kind : ObservationFailureKind
  planId : DefinitionId
  relatedDefinitionIds : List DefinitionId := []
  limit : Option EvidenceBound := none
  observedCount : Option Nat := none
  alternatives : List DefinitionId := []
  missingDiscriminator : Option DefinitionId := none
  deriving BEq, DecidableEq, Repr

def ObservationDiagnostic.status (diagnostic : ObservationDiagnostic) : ObservationStatus :=
  diagnostic.kind.status

structure EvidenceOrderingFact where
  recordId : DefinitionId
  kind : DefinitionId
  sequence : Nat
  origin : Option EvidenceOrigin := none
  causalParents : List DefinitionId
  deriving BEq, DecidableEq, Repr

inductive AppliedDispositionEvidence where
  | retained (normalizedValue : String)
  | redactedContribution
  | digestToken (policy : DefinitionId) (token : String)
  /-- Invalid constructor retained so independently supplied wrappers fail closed at validation. -/
  | raw (value : String)
  /-- Invalid constructor retained so rejected material cannot be smuggled into a wrapper. -/
  | rejectedMaterial (value : String)
  deriving BEq, DecidableEq, Repr

structure AppliedFieldDisposition where
  field : EvidenceFieldReference
  evidence : AppliedDispositionEvidence
  deriving BEq, DecidableEq, Repr

structure EvidenceFieldSupport where
  field : DefinitionId
  valueType : ObservationValueType
  evidence : AppliedDispositionEvidence
  deriving BEq, DecidableEq, Repr

structure EvidenceRecordSupport where
  recordId : DefinitionId
  origin : Option EvidenceOrigin
  kind : DefinitionId
  causalParents : List DefinitionId
  fields : List EvidenceFieldSupport
  deriving BEq, DecidableEq, Repr

/-- Why the checked mapping accepted one Model Fact from the supplied Evidence. -/
structure EvidenceLink where
  coordinate : ModelCoordinate
  mappingId : DefinitionId
  mappingVersion : Nat
  mappingDigest : String
  profileId : DefinitionId
  profileVersion : Nat
  evidenceIdentities : List DefinitionId
  ruleId : DefinitionId
  bindingIds : List DefinitionId
  orderingSupport : List EvidenceOrderingFact
  closureSupport : List EvidenceClosureFact
  appliedDispositions : List AppliedFieldDisposition
  appliedBound : EvidenceBound
  meaningDigest : String
  deriving BEq, DecidableEq, Repr

/-- Wide unchecked carrier used only while assembling and negatively testing trace admission. -/
structure UncheckedEvidenceBackedTrace where
  traceId : String
  checkedPlan : CheckedObservationPlan
  mappingId : DefinitionId
  mappingVersion : Nat
  mappingDigest : String
  source : SourceLocation
  profileId : DefinitionId
  profileVersion : Nat
  sourceClosed : Bool
  vocabulary : List MeaningProvision
  dispositions : List FieldDispositionDeclaration
  appliedBound : EvidenceBound
  evidenceIdentities : List DefinitionId
  recordSupport : List EvidenceRecordSupport
  trace : ModelTrace ModelValue ModelValue ModelValue ModelValue
  evidenceLinks : List EvidenceLink
  deriving BEq, DecidableEq, Repr

/--
Complete auditable wrapper around the unchanged immutable Model Trace. Its private constructor
ensures that only successful Observation admission can produce one.
-/
structure EvidenceBackedTrace where
  private mk ::
  traceId : String
  checkedPlan : CheckedObservationPlan
  mappingId : DefinitionId
  mappingVersion : Nat
  mappingDigest : String
  source : SourceLocation
  profileId : DefinitionId
  profileVersion : Nat
  sourceClosed : Bool
  vocabulary : List MeaningProvision
  dispositions : List FieldDispositionDeclaration
  appliedBound : EvidenceBound
  evidenceIdentities : List DefinitionId
  recordSupport : List EvidenceRecordSupport
  trace : ModelTrace ModelValue ModelValue ModelValue ModelValue
  evidenceLinks : List EvidenceLink
  deriving BEq, DecidableEq, Repr

private def EvidenceBackedTrace.ofUnchecked
    (unchecked : UncheckedEvidenceBackedTrace) : EvidenceBackedTrace := {
  traceId := unchecked.traceId
  checkedPlan := unchecked.checkedPlan
  mappingId := unchecked.mappingId
  mappingVersion := unchecked.mappingVersion
  mappingDigest := unchecked.mappingDigest
  source := unchecked.source
  profileId := unchecked.profileId
  profileVersion := unchecked.profileVersion
  sourceClosed := unchecked.sourceClosed
  vocabulary := unchecked.vocabulary
  dispositions := unchecked.dispositions
  appliedBound := unchecked.appliedBound
  evidenceIdentities := unchecked.evidenceIdentities
  recordSupport := unchecked.recordSupport
  trace := unchecked.trace
  evidenceLinks := unchecked.evidenceLinks
}

/-- Observation Evaluation never exposes a partial Model Trace; only `accepted` carries one. -/
inductive ObservationResult where
  | accepted (trace : EvidenceBackedTrace)
  | unknown (diagnostic : ObservationDiagnostic)
  | conflict (diagnostic : ObservationDiagnostic)
  | unsupported (diagnostic : ObservationDiagnostic)
  deriving BEq, DecidableEq, Repr

def ObservationResult.status : ObservationResult → ObservationStatus
  | .accepted _ => .accepted
  | .unknown _ => .unknown
  | .conflict _ => .conflict
  | .unsupported _ => .unsupported

def ObservationResult.diagnostic? : ObservationResult → Option ObservationDiagnostic
  | .accepted _ => none
  | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic => some diagnostic

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

private def diagnostic
    (plan : CheckedObservationPlan)
    (kind : ObservationFailureKind)
    (related : List DefinitionId := []) : ObservationDiagnostic := {
  kind
  planId := plan.id
  relatedDefinitionIds := canonicalIds related
}

private def resultOfDiagnostic (failure : ObservationDiagnostic) : ObservationResult :=
  match failure.status with
  | .unknown => .unknown failure
  | .conflict => .conflict failure
  | .unsupported => .unsupported failure
  | .accepted => .unknown failure

private def firstDuplicateId : List DefinitionId → Option DefinitionId
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateId (second :: rest)
  | _ => none

private def firstDuplicateField : List EvidenceFieldValue → Option EvidenceFieldValue
  | first :: second :: rest =>
      if first.field == second.field then some first else firstDuplicateField (second :: rest)
  | _ => none

private def fieldValueLe (left right : EvidenceFieldValue) : Bool := idLe left.field right.field

private def recordLe (left right : SyntheticEvidenceRecord) : Bool :=
  match left.origin, right.origin with
  | some leftOrigin, some rightOrigin =>
      decide (leftOrigin.source.value < rightOrigin.source.value) ||
        (leftOrigin.source == rightOrigin.source &&
          (leftOrigin.ordinal < rightOrigin.ordinal ||
            (leftOrigin.ordinal == rightOrigin.ordinal && idLe left.id right.id)))
  | none, none =>
      left.sequence < right.sequence || (left.sequence == right.sequence && idLe left.id right.id)
  | none, some _ => true
  | some _, none => false

private def interpretationLe (left right : CompatibleInterpretation) : Bool := idLe left.id right.id

private def firstContradictoryInterpretation :
    List CompatibleInterpretation → Option DefinitionId
  | first :: second :: rest =>
      if first.id == second.id && first.evidenceIdentities != second.evidenceIdentities then
        some first.id
      else
        firstContradictoryInterpretation (second :: rest)
  | _ => none

private def closureLe (left right : EvidenceClosureFact) : Bool :=
  match left.source, right.source with
  | some leftSource, some rightSource =>
      decide (leftSource.value < rightSource.value) ||
        (leftSource == rightSource && idLe left.kind right.kind)
  | none, none => idLe left.kind right.kind
  | none, some _ => true
  | some _, none => false

private def firstDuplicateClosure : List EvidenceClosureFact → Option EvidenceClosureFact
  | first :: second :: rest =>
      if first.source == second.source && first.kind == second.kind then some first
      else firstDuplicateClosure (second :: rest)
  | _ => none

namespace Observation.Internal

inductive StructuralOriginMode where
  | globalSequence
  | sourceSequence
  | mixed
  deriving BEq, DecidableEq, Repr

inductive StructuralFinding where
  | duplicateIdentity (recordId : DefinitionId) (conflicting : Bool)
  | mixedOrigins (recordIds : List DefinitionId)
  | duplicateSequence (firstId secondId : DefinitionId) (sequence : Nat)
  | sequenceGap
      (recordId : DefinitionId)
      (source : Option DefinitionId)
      (expected actual : Nat)
  | missingCausalParent (recordId : DefinitionId) (parentId : Option DefinitionId)
  | contradictoryOrder (recordId parentId : DefinitionId)
  | duplicateClosure
      (source : Option DefinitionId)
      (kind : DefinitionId)
      (conflicting : Bool)
  | closureWithoutFacts (source : Option DefinitionId) (kind : DefinitionId)
  | missingClosure
      (recordIds : List DefinitionId)
      (source : Option DefinitionId)
      (kind : DefinitionId)
  | closureSequenceMismatch
      (source : Option DefinitionId)
      (kind : DefinitionId)
      (expected actual : Nat)
  | closureCountMismatch
      (source : Option DefinitionId)
      (kind : DefinitionId)
      (expected : Nat)
      (actual : Option Nat)
  | closureByteCountMissing (source : Option DefinitionId) (kind : DefinitionId)
  | missingRequiredKind (kind : DefinitionId)
  | inconsistentOrderingSupport
      (ruleId : DefinitionId)
      (expected actual : List EvidenceOrderingFact)
  | inconsistentClosureSupport
      (ruleId : DefinitionId)
      (expected actual : List EvidenceClosureFact)
  deriving BEq, DecidableEq, Repr

structure ClosureExpectation where
  source : Option DefinitionId
  kind : DefinitionId
  recordIds : List DefinitionId
  lastSequence : Nat
  recordCount : Nat
  deriving BEq, DecidableEq, Repr

structure StructuralLinkSupport where
  ruleId : DefinitionId
  evidenceIdentities : List DefinitionId
  orderingSupport : List EvidenceOrderingFact
  closureSupport : List EvidenceClosureFact
  deriving BEq, DecidableEq, Repr

structure NormalizedStructuralLinkSupport where
  ruleId : DefinitionId
  evidenceIdentities : List DefinitionId
  facts : List EvidenceOrderingFact
  closures : List EvidenceClosureFact
  deriving BEq, DecidableEq, Repr

structure StructuralAnalysis where
  facts : List EvidenceOrderingFact
  closures : List EvidenceClosureFact
  originMode : StructuralOriginMode
  closureExpectations : List ClosureExpectation
  links : List NormalizedStructuralLinkSupport
  findings : List StructuralFinding
  deriving BEq, DecidableEq, Repr

private def factByRecordLe (left right : EvidenceOrderingFact) : Bool :=
  idLe left.recordId right.recordId

private def factBySequenceLe (left right : EvidenceOrderingFact) : Bool :=
  match left.origin, right.origin with
  | some leftOrigin, some rightOrigin =>
      decide (leftOrigin.source.value < rightOrigin.source.value) ||
        (leftOrigin.source == rightOrigin.source &&
          (leftOrigin.ordinal < rightOrigin.ordinal ||
            (leftOrigin.ordinal == rightOrigin.ordinal && idLe left.recordId right.recordId)))
  | none, none => left.sequence < right.sequence ||
      (left.sequence == right.sequence && idLe left.recordId right.recordId)
  | none, some _ => true
  | some _, none => false

private def canonicalFacts :
    List EvidenceOrderingFact → List EvidenceOrderingFact × List StructuralFinding
  | [] => ([], [])
  | [fact] => ([fact], [])
  | first :: second :: rest =>
      if first.recordId == second.recordId then
        let (facts, findings) := canonicalFacts (first :: rest)
        (facts, .duplicateIdentity first.recordId (first != second) :: findings)
      else
        let (facts, findings) := canonicalFacts (second :: rest)
        (first :: facts, findings)

private def canonicalClosures :
    List EvidenceClosureFact → List EvidenceClosureFact × List StructuralFinding
  | [] => ([], [])
  | [closure] => ([closure], [])
  | first :: second :: rest =>
      if first.source == second.source && first.kind == second.kind then
        let (closures, findings) := canonicalClosures (first :: rest)
        (closures, .duplicateClosure first.source first.kind (first != second) :: findings)
      else
        let (closures, findings) := canonicalClosures (second :: rest)
        (first :: closures, findings)

private partial def factDependsOn
    (facts : List EvidenceOrderingFact)
    (recordId target : DefinitionId)
    (visited : List DefinitionId := []) : Bool :=
  if recordId == target then true
  else if visited.contains recordId then false
  else
    match facts.find? fun fact => fact.recordId == recordId with
    | none => false
    | some fact => fact.causalParents.any fun parent =>
        factDependsOn facts parent target (recordId :: visited)

private def globalSequenceFindings
    (facts : List EvidenceOrderingFact) : List StructuralFinding := Id.run do
  let mut findings := []
  let mut expectedSequence := 1
  let mut previous : Option EvidenceOrderingFact := none
  for fact in facts do
    match previous with
    | some prior =>
        if fact.sequence == prior.sequence then
          findings := findings ++ [.duplicateSequence prior.recordId fact.recordId fact.sequence]
    | none => pure ()
    if fact.sequence != expectedSequence then
      findings := findings ++ [.sequenceGap fact.recordId none expectedSequence fact.sequence]
    if previous.isSome && fact.causalParents.isEmpty then
      findings := findings ++ [.missingCausalParent fact.recordId none]
    for parent in fact.causalParents do
      match facts.find? fun candidate => candidate.recordId == parent with
      | none => findings := findings ++ [.missingCausalParent fact.recordId (some parent)]
      | some parentFact =>
          if factDependsOn facts parent fact.recordId || parentFact.sequence >= fact.sequence then
            findings := findings ++ [.contradictoryOrder fact.recordId parent]
    previous := some fact
    expectedSequence := expectedSequence + 1
  pure findings

private def sourceSequenceFindings
    (facts : List EvidenceOrderingFact) : List StructuralFinding := Id.run do
  let mut findings := []
  let sources := canonicalIds <| facts.filterMap fun fact => fact.origin.map EvidenceOrigin.source
  for source in sources do
    let sourceFacts := facts.filter fun fact =>
      fact.origin.any fun origin => origin.source == source
    let mut expectedOrdinal := 0
    for fact in sourceFacts do
      match fact.origin with
      | some origin =>
          if origin.ordinal != expectedOrdinal then
            findings := findings ++ [
              .sequenceGap fact.recordId (some source) expectedOrdinal origin.ordinal]
      | none => pure ()
      expectedOrdinal := expectedOrdinal + 1
  for fact in facts do
    for parent in fact.causalParents do
      match facts.find? fun candidate => candidate.recordId == parent with
      | none => findings := findings ++ [.missingCausalParent fact.recordId (some parent)]
      | some parentFact =>
          let reversesSourceOrder := match fact.origin, parentFact.origin with
            | some factOrigin, some parentOrigin =>
                factOrigin.source == parentOrigin.source &&
                  parentOrigin.ordinal >= factOrigin.ordinal
            | _, _ => false
          if factDependsOn facts parent fact.recordId || reversesSourceOrder then
            findings := findings ++ [.contradictoryOrder fact.recordId parent]
  pure findings

private structure ClosureKey where
  source : Option DefinitionId
  kind : DefinitionId
  deriving BEq, DecidableEq, Repr

private def closureKeyLe (left right : ClosureKey) : Bool :=
  match left.source, right.source with
  | some leftSource, some rightSource =>
      decide (leftSource.value < rightSource.value) ||
        (leftSource == rightSource && idLe left.kind right.kind)
  | none, none => idLe left.kind right.kind
  | none, some _ => true
  | some _, none => false

private def closureExpectations
    (originMode : StructuralOriginMode)
    (facts : List EvidenceOrderingFact) : List ClosureExpectation :=
  let keys := facts.map fun fact => {
    source := if originMode == .sourceSequence then fact.origin.map EvidenceOrigin.source else none
    kind := fact.kind
  }
  let keys := keys.mergeSort closureKeyLe |>.eraseDups
  keys.map fun key =>
    let matchingFacts := facts.filter fun fact =>
      fact.kind == key.kind &&
        (key.source.isNone || fact.origin.map EvidenceOrigin.source == key.source)
    let lastSequence := matchingFacts.foldl (fun current fact =>
      let sequence := match key.source, fact.origin with
        | some _, some origin => origin.ordinal + 1
        | _, _ => fact.sequence
      Nat.max current sequence) 0
    {
      source := key.source
      kind := key.kind
      recordIds := matchingFacts.map EvidenceOrderingFact.recordId
      lastSequence
      recordCount := matchingFacts.length
    }

private def globalClosureFindings
    (requiredKinds : List DefinitionId)
    (closures : List EvidenceClosureFact)
    (expectations : List ClosureExpectation) : List StructuralFinding := Id.run do
  let mut findings := []
  for closure in closures do
    match expectations.find? fun expectation => expectation.kind == closure.kind with
    | none => findings := findings ++ [.closureWithoutFacts closure.source closure.kind]
    | some expectation =>
        if closure.lastSequence != expectation.lastSequence then
          findings := findings ++ [
            .closureSequenceMismatch none closure.kind
              expectation.lastSequence closure.lastSequence]
  for expectation in expectations do
    if !(closures.any fun closure => closure.kind == expectation.kind) then
      findings := findings ++ [
        .missingClosure expectation.recordIds none expectation.kind]
  for kind in requiredKinds do
    if !(expectations.any fun expectation => expectation.kind == kind) then
      findings := findings ++ [.missingRequiredKind kind]
  pure findings

private def sourceClosureFindings
    (requiredKinds : List DefinitionId)
    (closures : List EvidenceClosureFact)
    (expectations : List ClosureExpectation) : List StructuralFinding := Id.run do
  let mut findings := []
  for closure in closures do
    match expectations.find? fun expectation =>
        expectation.source == closure.source && expectation.kind == closure.kind with
    | none => findings := findings ++ [.closureWithoutFacts closure.source closure.kind]
    | some expectation =>
        if closure.lastSequence != expectation.lastSequence then
          findings := findings ++ [
            .closureSequenceMismatch closure.source closure.kind
              expectation.lastSequence closure.lastSequence]
        if closure.recordCount != some expectation.recordCount then
          findings := findings ++ [
            .closureCountMismatch closure.source closure.kind
              expectation.recordCount closure.recordCount]
        if closure.byteCount.isNone then
          findings := findings ++ [.closureByteCountMissing closure.source closure.kind]
  for expectation in expectations do
    if !(closures.any fun closure =>
        closure.source == expectation.source && closure.kind == expectation.kind) then
      findings := findings ++ [
        .missingClosure expectation.recordIds expectation.source expectation.kind]
  for kind in requiredKinds do
    if !(expectations.any fun expectation => expectation.kind == kind) then
      findings := findings ++ [.missingRequiredKind kind]
  pure findings

private def normalizeLinkSupport
    (originMode : StructuralOriginMode)
    (sharedFacts : List EvidenceOrderingFact)
    (sharedClosures : List EvidenceClosureFact)
    (support : StructuralLinkSupport) :
    NormalizedStructuralLinkSupport × List StructuralFinding :=
  let facts := match originMode with
    | .globalSequence => support.orderingSupport.mergeSort factByRecordLe
    | .sourceSequence | .mixed => support.orderingSupport.mergeSort factBySequenceLe
  let expectedFacts := match originMode with
    | .globalSequence =>
        (sharedFacts.filter fun fact => support.evidenceIdentities.contains fact.recordId).mergeSort
          factByRecordLe
    | .sourceSequence | .mixed => sharedFacts
  let orderingConsistent := match originMode with
    | .globalSequence =>
        canonicalIds (facts.map EvidenceOrderingFact.recordId) ==
            canonicalIds support.evidenceIdentities &&
          facts.all fun fact => sharedFacts.contains fact
    | .sourceSequence | .mixed => facts == expectedFacts
  let closures := support.closureSupport.mergeSort closureLe
  let findings :=
    (if orderingConsistent then [] else
      [.inconsistentOrderingSupport support.ruleId expectedFacts facts]) ++
    (if closures == sharedClosures then [] else
      [.inconsistentClosureSupport support.ruleId sharedClosures closures])
  ({
    ruleId := support.ruleId
    evidenceIdentities := support.evidenceIdentities
    facts
    closures
  }, findings)

def analyzeStructure
    (directFacts : List EvidenceOrderingFact)
    (directClosures : List EvidenceClosureFact)
    (requiredKinds : List DefinitionId := [])
    (linkSupport : List StructuralLinkSupport := []) : StructuralAnalysis :=
  let suppliedFacts := if linkSupport.isEmpty then directFacts
    else linkSupport.flatMap StructuralLinkSupport.orderingSupport
  let suppliedClosures := if linkSupport.isEmpty then directClosures
    else linkSupport.flatMap StructuralLinkSupport.closureSupport
  let factsById := suppliedFacts.mergeSort factByRecordLe
  let (facts, identityFindings) := canonicalFacts factsById
  let identityFindings := if linkSupport.isEmpty then identityFindings else
    identityFindings.filter fun finding => match finding with
      | .duplicateIdentity _ true => true
      | _ => false
  let facts := facts.mergeSort factBySequenceLe
  let originMode := if facts.isEmpty then .globalSequence
    else if facts.all fun fact => fact.origin.isSome then .sourceSequence
    else if facts.any fun fact => fact.origin.isSome then .mixed
    else .globalSequence
  let orderingFindings := match originMode with
    | .globalSequence => globalSequenceFindings facts
    | .sourceSequence => sourceSequenceFindings facts
    | .mixed => [.mixedOrigins (facts.map EvidenceOrderingFact.recordId)]
  let sortedClosures := suppliedClosures.mergeSort closureLe
  let (closures, duplicateClosureFindings) := canonicalClosures sortedClosures
  let duplicateClosureFindings := if linkSupport.isEmpty then duplicateClosureFindings else
    duplicateClosureFindings.filter fun finding => match finding with
      | .duplicateClosure _ _ true => true
      | _ => false
  let closureExpectations := closureExpectations originMode facts
  let closureFindings := match originMode with
    | .globalSequence => globalClosureFindings requiredKinds closures closureExpectations
    | .sourceSequence => sourceClosureFindings requiredKinds closures closureExpectations
    | .mixed => []
  let normalizedLinks := linkSupport.map fun support =>
    normalizeLinkSupport originMode facts closures support
  let links := normalizedLinks.map Prod.fst
  let linkFindings := normalizedLinks.flatMap Prod.snd
  {
    facts
    closures
    originMode
    closureExpectations
    links
    findings := identityFindings ++ orderingFindings ++ duplicateClosureFindings ++ closureFindings ++
      linkFindings
  }

end Observation.Internal

private def referenceLe (left right : EvidenceFieldReference) : Bool :=
  decide (left.kind.value < right.kind.value) ||
    (left.kind == right.kind && decide (left.field.value ≤ right.field.value))

private def canonicalReferences
    (references : List EvidenceFieldReference) : List EvidenceFieldReference :=
  references.mergeSort referenceLe |>.eraseDups

private def fieldDeclaration?
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (fieldId : DefinitionId) : Option EvidenceFieldDeclaration := do
  let kind ← plan.profile.kinds.find? fun declaration => declaration.id == record.kind
  kind.fields.find? fun declaration => declaration.id == fieldId

private def dispositionFor
    (plan : CheckedObservationPlan)
    (reference : EvidenceFieldReference) : Option FieldDisposition :=
  (plan.dispositions.find? fun declaration => declaration.field == reference).map
    FieldDispositionDeclaration.disposition

private def fieldValue?
    (record : SyntheticEvidenceRecord)
    (reference : EvidenceFieldReference) : Option EvidenceFieldValue :=
  if reference.kind != record.kind then none
  else record.fields.find? fun field => field.field == reference.field

def syntheticDigestToken
    (policy : DigestPolicyDeclaration)
    (normalizedValue : String) : String :=
  policy.name ++ "/v" ++ toString policy.version ++ ":" ++ toString normalizedValue.hash

private def findPolicy
    (plan : CheckedObservationPlan)
    (id : DefinitionId) : Option DigestPolicyDeclaration :=
  plan.digestPolicies.find? fun policy => policy.id == id

private def validateDigestMetadata
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (field : EvidenceFieldValue)
    (policyId : DefinitionId) : Except ObservationDiagnostic Unit := do
  if field.digestPolicy != some policyId then
    throw (diagnostic plan .digestPolicyMismatch [record.id, field.field, policyId])
  match findPolicy plan policyId with
    | some _ => pure ()
    | none => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field, policyId])

private def validateRecord
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except ObservationDiagnostic Unit := do
  if record.profile != plan.profile.id then
    throw (diagnostic plan .profileMismatch [record.id, record.profile])
  if record.profileVersion != plan.profile.version then
    throw (diagnostic plan .profileVersionMismatch [record.id, record.profile])
  if !(plan.profile.kinds.any fun kind => kind.id == record.kind) then
    throw (diagnostic plan .kindMismatch [record.id, record.kind])
  match firstDuplicateField (record.fields.mergeSort fieldValueLe) with
  | some duplicate =>
      throw (diagnostic plan .contradictoryFact [record.id, duplicate.field])
  | none => pure ()
  for field in record.fields do
    let declaration ← match fieldDeclaration? plan record field.field with
      | some declaration => pure declaration
      | none => throw (diagnostic plan .fieldMismatch [record.id, field.field])
    if declaration.valueType != field.value.valueType then
      throw (diagnostic plan .normalizationFailure [record.id, field.field])
    let reference : EvidenceFieldReference := { kind := record.kind, field := field.field }
    match dispositionFor plan reference with
    | some .reject => throw (diagnostic plan .rejectedFieldPresent [record.id, field.field])
    | some (.hash (some policy)) => validateDigestMetadata plan record field policy
    | some (.hash none) => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field])
    | some .retain | some .redact | none => pure ()
  for fact in record.bindingFacts do
    let matchingFacts := record.bindingFacts.filter fun other => other.binding == fact.binding
    if matchingFacts.any fun other => other.value != fact.value then
      throw (diagnostic plan .contradictoryBinding [record.id, fact.binding])

private partial def recordDependsOn
    (records : List SyntheticEvidenceRecord)
    (recordId target : DefinitionId)
    (visited : List DefinitionId := []) : Bool :=
  if recordId == target then true
  else if visited.contains recordId then false
  else
    match records.find? fun record => record.id == recordId with
    | none => false
    | some record => record.causalParents.any fun parent =>
        recordDependsOn records parent target (recordId :: visited)

private def validateMultiSourceSequenceAndCausality
    (plan : CheckedObservationPlan)
    (records : List SyntheticEvidenceRecord) : Except ObservationDiagnostic Unit := do
  let sources := canonicalIds <| records.filterMap fun record =>
    record.origin.map EvidenceOrigin.source
  for source in sources do
    let sourceRecords := (records.filter fun record =>
      record.origin.any fun origin => origin.source == source).mergeSort recordLe
    let mut expectedOrdinal := 0
    for record in sourceRecords do
      let origin ← match record.origin with
        | some origin => pure origin
        | none => throw (diagnostic plan .incomparableOrdering [record.id])
      if origin.ordinal != expectedOrdinal then
        throw (diagnostic plan .sequenceGap [record.id, source])
      expectedOrdinal := expectedOrdinal + 1
  for record in records do
    for parent in record.causalParents do
      let parentRecord ← match records.find? fun candidate => candidate.id == parent with
        | some candidate => pure candidate
        | none => throw (diagnostic plan .missingCausalParent [record.id, parent])
      if recordDependsOn records parent record.id then
        throw (diagnostic plan .contradictoryOrder [record.id, parent])
      match record.origin, parentRecord.origin with
      | some recordOrigin, some parentOrigin =>
          if recordOrigin.source == parentOrigin.source &&
              parentOrigin.ordinal >= recordOrigin.ordinal then
            throw (diagnostic plan .contradictoryOrder [record.id, parent])
      | _, _ => throw (diagnostic plan .incomparableOrdering [record.id, parent])
    match record.faultTarget with
    | some target =>
        let targetRecord ← match records.find? fun candidate => candidate.id == target with
          | some candidate => pure candidate
          | none => throw (diagnostic plan .misdirectedFaultReceipt [record.id, target])
        let sameSourceBefore := match targetRecord.origin, record.origin with
          | some targetOrigin, some recordOrigin =>
              targetOrigin.source == recordOrigin.source &&
                targetOrigin.ordinal < recordOrigin.ordinal
          | _, _ => false
        if !sameSourceBefore && !recordDependsOn records record.id target then
          throw (diagnostic plan .misdirectedFaultReceipt [record.id, target])
    | none => pure ()

private def validateSequenceAndCausality
    (plan : CheckedObservationPlan)
    (records : List SyntheticEvidenceRecord) : Except ObservationDiagnostic Unit := do
  let ids := records.map SyntheticEvidenceRecord.id
  match firstDuplicateId (ids.mergeSort idLe) with
  | some duplicate => throw (diagnostic plan .duplicateEvidenceIdentity [duplicate])
  | none => pure ()
  if records.all fun record => record.origin.isSome then
    validateMultiSourceSequenceAndCausality plan records
    return
  if records.any fun record => record.origin.isSome then
    throw (diagnostic plan .incomparableOrdering ids)
  for record in records do
    match record.faultTarget with
    | some target =>
        match records.find? fun candidate => candidate.id == target with
        | some targetRecord =>
            if targetRecord.sequence >= record.sequence then
              throw (diagnostic plan .misdirectedFaultReceipt [record.id, target])
        | none => throw (diagnostic plan .misdirectedFaultReceipt [record.id, target])
    | none => pure ()
  let ordered := records.mergeSort recordLe
  let mut expected := 1
  let mut previous : Option SyntheticEvidenceRecord := none
  for record in ordered do
    match previous with
    | some prior =>
        if record.sequence == prior.sequence then
          throw (diagnostic plan .incomparableOrdering [prior.id, record.id])
    | none => pure ()
    if record.sequence != expected then
      throw (diagnostic plan .sequenceGap [record.id])
    if previous.isSome && record.causalParents.isEmpty then
      throw (diagnostic plan .missingCausalParent [record.id])
    for parent in record.causalParents do
      let parentRecord ← match records.find? fun candidate => candidate.id == parent with
        | some candidate => pure candidate
        | none => throw (diagnostic plan .missingCausalParent [record.id, parent])
      if parentRecord.sequence >= record.sequence then
        throw (diagnostic plan .contradictoryOrder [record.id, parent])
    previous := some record
    expected := expected + 1

private def validateClosures
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle) : Except ObservationDiagnostic Unit := do
  if bundle.records.all fun record => record.origin.isSome then
    let closures := bundle.closures.mergeSort closureLe
    match firstDuplicateClosure closures with
    | some duplicate => throw (diagnostic plan .missingClosure [duplicate.kind])
    | none => pure ()
    for closure in closures do
      let source ← match closure.source with
        | some source => pure source
        | none => throw (diagnostic plan .missingClosure [closure.kind])
      if !(plan.closures.any fun required => required.kind == closure.kind) then
        throw (diagnostic plan .missingClosure [closure.kind])
      let sourceRecords := bundle.records.filter fun record =>
        record.kind == closure.kind &&
          record.origin.any fun origin => origin.source == source
      let lastSequence := sourceRecords.foldl (fun current record =>
        Nat.max current (record.origin.map (fun origin => origin.ordinal + 1) |>.getD 0)) 0
      if sourceRecords.isEmpty || closure.lastSequence != lastSequence ||
          closure.recordCount != some sourceRecords.length || closure.byteCount.isNone then
        throw (diagnostic plan .missingClosure [source, closure.kind])
    for record in bundle.records do
      let origin ← match record.origin with
        | some origin => pure origin
        | none => throw (diagnostic plan .missingClosure [record.id])
      if !(closures.any fun closure => closure.source == some origin.source &&
          closure.kind == record.kind) then
        throw (diagnostic plan .missingClosure [record.id, origin.source, record.kind])
    for required in plan.closures do
      if !(bundle.records.any fun record => record.kind == required.kind) then
        throw (diagnostic plan .missingClosure [required.kind])
    return
  for required in plan.closures do
    let closure ← match bundle.closures.find? fun fact => fact.kind == required.kind with
      | some closure => pure closure
      | none => throw (diagnostic plan .missingClosure [required.kind])
    let kindSequences := bundle.records.filter (fun record => record.kind == required.kind)
      |>.map SyntheticEvidenceRecord.sequence
    let lastSequence := kindSequences.foldl Nat.max 0
    if closure.lastSequence != lastSequence then
      throw (diagnostic plan .missingClosure [required.kind])

private partial def rulePathExists
    (ordering : List ObservationOrdering)
    (current target : DefinitionId)
    (visited : List DefinitionId := []) : Bool :=
  if current == target then true
  else if visited.contains current then false
  else
    (ordering.filter fun edge => edge.before == current).any fun edge =>
      rulePathExists ordering edge.after target (current :: visited)

private def ruleLe
    (plan : CheckedObservationPlan)
    (left right : CheckedObservationRule) : Bool :=
  if rulePathExists plan.ordering left.id right.id then true
  else if rulePathExists plan.ordering right.id left.id then false
  else idLe left.id right.id

private partial def expressionBindingIds
    (plan : CheckedObservationPlan)
    (expression : CheckedObservationExpression)
    (visited : List DefinitionId := []) : List DefinitionId :=
  match expression with
  | .binding id _ _ =>
      if visited.contains id then []
      else match plan.bindings.find? fun binding => binding.id == id with
        | some binding => id :: expressionBindingIds plan binding.expression (id :: visited)
        | none => [id]
  | .normalize _ operand | .present operand | .not operand |
      .contributionMarker operand | .digestToken _ operand =>
      expressionBindingIds plan operand visited
  | .equals left right | .and left right | .or left right =>
      expressionBindingIds plan left visited ++ expressionBindingIds plan right visited
  | _ => []

private partial def expressionReferences
    (plan : CheckedObservationPlan)
    (expression : CheckedObservationExpression)
    (visited : List DefinitionId := []) : List EvidenceFieldReference :=
  match expression with
  | .field reference _ _ => [reference]
  | .binding id _ _ =>
      if visited.contains id then []
      else match plan.bindings.find? fun binding => binding.id == id with
        | some binding => expressionReferences plan binding.expression (id :: visited)
        | none => []
  | .normalize _ operand | .present operand | .not operand |
      .contributionMarker operand | .digestToken _ operand =>
      expressionReferences plan operand visited
  | .equals left right | .and left right | .or left right =>
      expressionReferences plan left visited ++ expressionReferences plan right visited
  | _ => []

mutual
  private partial def evaluateExpression
      (plan : CheckedObservationPlan)
      (record : SyntheticEvidenceRecord)
      (expression : CheckedObservationExpression)
      (visited : List DefinitionId := []) : Except ObservationDiagnostic EvidenceValue := do
    match expression with
    | .text value => pure (.text value)
    | .natural value => pure (.natural value)
    | .boolean value => pure (.boolean value)
    | .field reference _ _ =>
        match fieldValue? record reference with
        | some field => pure field.value
        | none => throw (diagnostic plan .unresolvedBinding [record.id, reference.field])
    | .binding id _ _ => evaluateBinding plan record id visited
    | .normalize operator operand =>
        let value ← evaluateExpression plan record operand visited
        match operator, value with
        | .textTrimV1, .text text => pure (.text text.trimAscii.copy)
        | .textLowercaseV1, .text text => pure (.text text.toLower)
        | .naturalRenderV1, .natural value => pure (.text (toString value))
        | _, _ => throw (diagnostic plan .normalizationFailure [record.id])
    | .present operand =>
        match evaluateExpression plan record operand visited with
        | .ok _ => pure (.boolean true)
        | .error _ => pure (.boolean false)
    | .equals left right =>
        let leftValue ← evaluateExpression plan record left visited
        let rightValue ← evaluateExpression plan record right visited
        pure (.boolean (leftValue == rightValue))
    | .and left right =>
        match ← evaluateExpression plan record left visited,
            ← evaluateExpression plan record right visited with
        | .boolean leftValue, .boolean rightValue => pure (.boolean (leftValue && rightValue))
        | _, _ => throw (diagnostic plan .normalizationFailure [record.id])
    | .or left right =>
        match ← evaluateExpression plan record left visited,
            ← evaluateExpression plan record right visited with
        | .boolean leftValue, .boolean rightValue => pure (.boolean (leftValue || rightValue))
        | _, _ => throw (diagnostic plan .normalizationFailure [record.id])
    | .not operand =>
        match ← evaluateExpression plan record operand visited with
        | .boolean value => pure (.boolean (!value))
        | _ => throw (diagnostic plan .normalizationFailure [record.id])
    | .contributionMarker operand =>
        let _ ← evaluateExpression plan record operand visited
        pure (.text "contributed")
    | .digestToken policy operand =>
        let value ← evaluateExpression plan record operand visited
        pure (.text (syntheticDigestToken policy value.render))

  private partial def evaluateBinding
      (plan : CheckedObservationPlan)
      (record : SyntheticEvidenceRecord)
      (id : DefinitionId)
      (visited : List DefinitionId) : Except ObservationDiagnostic EvidenceValue := do
    if visited.contains id then
      throw (diagnostic plan .unresolvedBinding [record.id, id])
    let binding ← match plan.bindings.find? fun candidate => candidate.id == id with
      | some binding => pure binding
      | none => throw (diagnostic plan .unresolvedBinding [record.id, id])
    let value ← evaluateExpression plan record binding.expression (id :: visited)
    let facts := record.bindingFacts.filter fun fact => fact.binding == id
    if facts.any fun fact => fact.value != value then
      throw (diagnostic plan .contradictoryBinding [record.id, id])
    pure value
end

private def validateBindingFacts
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except ObservationDiagnostic Unit := do
  for fact in record.bindingFacts do
    let value ← evaluateBinding plan record fact.binding []
    if value != fact.value then
      throw (diagnostic plan .contradictoryBinding [record.id, fact.binding])

private def conditionHolds
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (condition : Option CheckedObservationExpression) : Except ObservationDiagnostic Bool := do
  match condition with
  | none => pure true
  | some expression =>
      match ← evaluateExpression plan record expression with
      | .boolean value => pure value
      | _ => throw (diagnostic plan .normalizationFailure [record.id])

private structure DigestClaim where
  recordId : DefinitionId
  fieldId : DefinitionId
  policyId : DefinitionId
  normalizedValue : EvidenceValue
  computedToken : String
  effectiveToken : String

private partial def digestClaimsInExpression
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (expression : CheckedObservationExpression)
    (visited : List DefinitionId := []) : Except ObservationDiagnostic (List DigestClaim) := do
  match expression with
  | .binding id _ _ =>
      if visited.contains id then pure []
      else match plan.bindings.find? fun binding => binding.id == id with
        | some binding => digestClaimsInExpression plan record binding.expression (id :: visited)
        | none => throw (diagnostic plan .unresolvedBinding [record.id, id])
  | .digestToken policy operand =>
      let normalizedValue ← evaluateExpression plan record operand visited
      let computedToken := syntheticDigestToken policy normalizedValue.render
      let mut claims := []
      for reference in canonicalReferences (expressionReferences plan operand visited) do
        if dispositionFor plan reference == some (.hash (some policy.id)) then
          let field ← match fieldValue? record reference with
            | some field => pure field
            | none => throw (diagnostic plan .unresolvedBinding [record.id, reference.field])
          claims := {
            recordId := record.id
            fieldId := reference.field
            policyId := policy.id
            normalizedValue
            computedToken
            effectiveToken := field.reportedDigestToken.getD computedToken
          } :: claims
      pure claims.reverse
  | .normalize _ operand | .present operand | .not operand | .contributionMarker operand =>
      digestClaimsInExpression plan record operand visited
  | .equals left right | .and left right | .or left right =>
      return (← digestClaimsInExpression plan record left visited) ++
        (← digestClaimsInExpression plan record right visited)
  | _ => pure []

private def digestClaimsForRule
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (rule : CheckedObservationRule) : Except ObservationDiagnostic (List DigestClaim) := do
  return (← digestClaimsInExpression plan record rule.value) ++
    (← rule.condition.toList.mapM (digestClaimsInExpression plan record)).flatten

private def ruleReferencesRecord
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (rule : CheckedObservationRule) : Bool :=
  let references := canonicalReferences <|
    expressionReferences plan rule.value ++
      rule.condition.toList.flatMap (expressionReferences plan)
  references.isEmpty || references.all fun reference => reference.kind == record.kind

private def detectDigestIssues
    (plan : CheckedObservationPlan)
    (records : List SyntheticEvidenceRecord) : Except ObservationDiagnostic Unit := do
  let mut claims := []
  for record in records do
    for rule in plan.rules do
      if ruleReferencesRecord plan record rule then
        if ← conditionHolds plan record rule.condition then
          claims := claims ++ (← digestClaimsForRule plan record rule)
  for claim in claims do
    match claims.find? fun other =>
        claim.policyId == other.policyId && claim.effectiveToken == other.effectiveToken &&
          claim.normalizedValue != other.normalizedValue with
    | some other =>
        throw (diagnostic plan .digestCollision [claim.recordId, other.recordId])
    | none => pure ()
  for claim in claims do
    if claim.effectiveToken != claim.computedToken then
      throw (diagnostic plan .digestPolicyMismatch
        [claim.recordId, claim.fieldId, claim.policyId])

private def normalizedRetainedValue
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (reference : EvidenceFieldReference) : Except ObservationDiagnostic String := do
  match plan.bindings.find? fun binding =>
      (expressionReferences plan binding.expression).contains reference with
  | some binding => return (← evaluateBinding plan record binding.id []).render
  | none =>
      match fieldValue? record reference with
      | some field => pure field.value.render
      | none => throw (diagnostic plan .unresolvedBinding [record.id, reference.field])

private def appliedDisposition
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (rule : CheckedObservationRule)
    (reference : EvidenceFieldReference) : Except ObservationDiagnostic AppliedFieldDisposition := do
  let disposition ← match dispositionFor plan reference with
    | some disposition => pure disposition
    | none => throw (diagnostic plan .disallowedRawMaterial [record.id, reference.field])
  let _field ← match fieldValue? record reference with
    | some field => pure field
    | none => throw (diagnostic plan .unresolvedBinding [record.id, reference.field])
  let evidence ← match disposition with
    | .retain => pure (.retained (← normalizedRetainedValue plan record reference))
    | .redact => pure .redactedContribution
    | .hash (some policyId) =>
        let claims ← digestClaimsForRule plan record rule
        let claim ← match claims.find? fun claim =>
            claim.fieldId == reference.field && claim.policyId == policyId with
          | some claim => pure claim
          | none => throw (diagnostic plan .digestPolicyMismatch [record.id, reference.field])
        if claim.effectiveToken != claim.computedToken then
          throw (diagnostic plan .digestPolicyMismatch [record.id, reference.field, policyId])
        pure (.digestToken policyId claim.computedToken)
    | .hash none => throw (diagnostic plan .digestPolicyMismatch [record.id, reference.field])
    | .reject => throw (diagnostic plan .rejectedFieldPresent [record.id, reference.field])
  pure { field := reference, evidence }

private def evidenceFieldSupport
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (field : EvidenceFieldValue) : Except ObservationDiagnostic EvidenceFieldSupport := do
  let reference : EvidenceFieldReference := { kind := record.kind, field := field.field }
  let disposition ← match dispositionFor plan reference with
    | some disposition => pure disposition
    | none => throw (diagnostic plan .disallowedRawMaterial [record.id, field.field])
  let evidence ← match disposition with
    | .retain => pure (.retained field.value.render)
    | .redact => pure .redactedContribution
    | .hash (some policyId) =>
        let policy ← match findPolicy plan policyId with
          | some policy => pure policy
          | none => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field, policyId])
        pure (.digestToken policyId
          (field.reportedDigestToken.getD (syntheticDigestToken policy field.value.render)))
    | .hash none => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field])
    | .reject => throw (diagnostic plan .rejectedFieldPresent [record.id, field.field])
  pure { field := field.field, valueType := field.value.valueType, evidence }

private def evidenceRecordSupport
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except ObservationDiagnostic EvidenceRecordSupport := do
  let fields ← (record.fields.mergeSort fieldValueLe).mapM (evidenceFieldSupport plan record)
  pure {
    recordId := record.id
    origin := record.origin
    kind := record.kind
    causalParents := canonicalIds record.causalParents
    fields
  }

private structure Emission where
  record : SyntheticEvidenceRecord
  rule : CheckedObservationRule
  value : ModelValue
  bindingIds : List DefinitionId
  dispositions : List AppliedFieldDisposition
  deriving BEq, Repr

private def emissionsFor
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except ObservationDiagnostic (List Emission) := do
  let mut emissions := []
  for rule in plan.rules.mergeSort (ruleLe plan) do
    if ruleReferencesRecord plan record rule then
      if ← conditionHolds plan record rule.condition then
        let value ← evaluateExpression plan record rule.value
        let rendered ← match value with
          | .text rendered => pure rendered
          | _ => throw (diagnostic plan .normalizationFailure [record.id, rule.id])
        let references := canonicalReferences <|
          expressionReferences plan rule.value ++
            rule.condition.toList.flatMap (expressionReferences plan)
        let mut dispositions := []
        for reference in references do
          dispositions := (← appliedDisposition plan record rule reference) :: dispositions
        emissions := {
          record
          rule
          value := { definitionId := rule.output, value := rendered }
          bindingIds := canonicalIds <|
            expressionBindingIds plan rule.value ++
              rule.condition.toList.flatMap (expressionBindingIds plan)
          dispositions := dispositions.reverse
        } :: emissions
  pure emissions.reverse

private def orderingFact (record : SyntheticEvidenceRecord) : EvidenceOrderingFact := {
  recordId := record.id
  kind := record.kind
  sequence := record.sequence
  origin := record.origin
  causalParents := canonicalIds record.causalParents
}

private def evidenceLinkFor
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle)
    (coordinate : ModelCoordinate)
    (emission : Emission) : EvidenceLink := {
  coordinate
  mappingId := plan.id
  mappingVersion := plan.version
  mappingDigest := plan.behaviorFingerprint.render
  profileId := plan.profile.id
  profileVersion := plan.profile.version
  evidenceIdentities := [emission.record.id]
  ruleId := emission.rule.id
  bindingIds := emission.bindingIds
  orderingSupport := if emission.record.origin.isSome then
      bundle.records.mergeSort recordLe |>.map orderingFact
    else [orderingFact emission.record]
  closureSupport := bundle.closures.mergeSort closureLe
  appliedDispositions := emission.dispositions
  appliedBound := plan.evidenceBound
  meaningDigest := emission.rule.meaning.canonicalBehavior
}

private def singleEmission
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (kind : DefinitionKind)
    (emissions : List Emission) : Except ObservationDiagnostic Emission :=
  match emissions.filter fun emission => emission.rule.outputKind == kind with
  | [emission] => pure emission
  | [] => throw (diagnostic plan .sequenceGap [record.id])
  | multiple => throw (diagnostic plan .contradictoryFact
      (record.id :: multiple.map fun emission => emission.rule.output))

private def ensureComparableEmissions
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (emissions : List Emission) : Except ObservationDiagnostic Unit := do
  for left in emissions do
    for right in emissions do
      if left.rule.id != right.rule.id &&
          !rulePathExists plan.ordering left.rule.id right.rule.id &&
          !rulePathExists plan.ordering right.rule.id left.rule.id then
        throw (diagnostic plan .incomparableOrdering [record.id, left.rule.id, right.rule.id])

private def evidenceBackedTraceId
    (mappingDigest : String)
    (evidenceIdentities : List DefinitionId)
    (recordSupport : List EvidenceRecordSupport)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (evidenceLinks : List EvidenceLink) : String :=
  (behaviorFingerprintOf <|
    mappingDigest ++ ":" ++ reprStr evidenceIdentities ++ ":" ++ reprStr recordSupport ++
      ":" ++ reprStr trace ++ ":" ++ reprStr evidenceLinks).render

private structure RecordEmissions where
  record : SyntheticEvidenceRecord
  emissions : List Emission

private def recordPrecedes
    (records : List SyntheticEvidenceRecord)
    (left right : SyntheticEvidenceRecord) : Bool :=
  let sourceLocal := match left.origin, right.origin with
    | some leftOrigin, some rightOrigin =>
        leftOrigin.source == rightOrigin.source && leftOrigin.ordinal < rightOrigin.ordinal
    | _, _ => false
  sourceLocal || recordDependsOn records right.id left.id

private def evaluateChecked
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle) : Except ObservationDiagnostic UncheckedEvidenceBackedTrace := do
  if bundle.records.length > plan.evidenceBound.value then
    throw {
      (diagnostic plan .evidenceBoundExhausted) with
      limit := some plan.evidenceBound
      observedCount := some bundle.records.length
    }
  if !bundle.sourceClosed then
    throw (diagnostic plan .missingClosure)
  if !bundle.knownGaps.isEmpty then
    throw (diagnostic plan .knownGap <|
      bundle.knownGaps.flatMap fun gap => gap.code :: gap.relatedDefinitionIds)
  if bundle.records.isEmpty then
    throw (diagnostic plan .emptyEvidence)
  if bundle.profile != plan.profile.id then
    throw (diagnostic plan .profileMismatch [bundle.profile])
  if bundle.profileVersion != plan.profile.version then
    throw (diagnostic plan .profileVersionMismatch [bundle.profile])
  let records := bundle.records.mergeSort recordLe
  for record in records do
    validateRecord plan record
  if !bundle.closedFieldKinds.isEmpty then
    for record in records do
      let declaration ← match plan.profile.kinds.find? fun kind => kind.id == record.kind with
        | some declaration => pure declaration
        | none => throw (diagnostic plan .kindMismatch [record.id, record.kind])
      if bundle.closedFieldKinds.contains record.kind then
        let expectedFields := declaration.fields.map EvidenceFieldDeclaration.id |>.mergeSort idLe
        let actualFields := record.fields.map EvidenceFieldValue.field |>.mergeSort idLe
        if actualFields != expectedFields then
          throw (diagnostic plan .fieldMismatch [record.id, record.kind])
  validateSequenceAndCausality plan records
  validateClosures plan bundle
  for record in records do
    validateBindingFacts plan record
  detectDigestIssues plan records
  if !bundle.compatibleAlternatives.isEmpty then
    let missingDiscriminator ← match bundle.missingDiscriminator with
      | some missingDiscriminator => pure missingDiscriminator
      | none => throw (diagnostic plan .unresolvedBinding)
    let interpretations := bundle.compatibleAlternatives.map fun interpretation => {
      interpretation with evidenceIdentities := canonicalIds interpretation.evidenceIdentities
    }
    let interpretations := interpretations.mergeSort interpretationLe
    match firstContradictoryInterpretation interpretations with
    | some interpretation => throw (diagnostic plan .contradictoryFact [interpretation])
    | none => pure ()
    let alternatives := interpretations.map CompatibleInterpretation.id |>.eraseDups
    throw {
      (diagnostic plan .compatibleAlternatives alternatives) with
      alternatives
      missingDiscriminator := some missingDiscriminator
    }
  let recordEmissions ← records.mapM fun record => do
    pure { record, emissions := ← emissionsFor plan record }
  let multiSource := records.all fun record => record.origin.isSome
  let firstRecord :: remainingRecords ← if multiSource then do
      let emissions := recordEmissions.flatMap RecordEmissions.emissions
      let stateEmissions := emissions.filter fun emission => emission.rule.outputKind == .state
      let initialCandidates := stateEmissions.filter fun candidate =>
        stateEmissions.all fun other =>
          (candidate.record.id == other.record.id && candidate.rule.id == other.rule.id) ||
            recordPrecedes records candidate.record other.record
      let initial ← match initialCandidates with
        | [initial] => pure initial
        | [] => throw (diagnostic plan .missingInitialState)
        | multiple => throw (diagnostic plan .contradictoryFact
            (multiple.map fun emission => emission.record.id))
      let remaining := emissions.filter fun emission =>
        emission.record.id != initial.record.id || emission.rule.id != initial.rule.id
      ensureComparableEmissions plan initial.record remaining
      let ordered := remaining.mergeSort fun left right =>
        if left.rule.id == right.rule.id then recordLe left.record right.record
        else ruleLe plan left.rule right.rule
      pure [{ record := initial.record, emissions := [initial] },
        { record := initial.record, emissions := ordered }]
    else pure recordEmissions
    | throw (diagnostic plan .emptyEvidence)
  let initialCandidates := firstRecord.emissions.filter fun emission =>
    emission.rule.outputKind == .state
  let initial ← match initialCandidates with
    | [initial] => pure initial
    | [] => throw (diagnostic plan .missingInitialState [firstRecord.record.id])
    | multiple => throw (diagnostic plan .contradictoryFact
        (firstRecord.record.id :: multiple.map fun emission => emission.rule.output))
  if firstRecord.emissions.any fun emission => emission.rule.outputKind != .state then
    throw (diagnostic plan .unconsumedReference [firstRecord.record.id])
  let mut steps := []
  let mut evidenceLinks := [evidenceLinkFor plan bundle .initialState initial]
  let mut stepPosition := 1
  for item in remainingRecords do
    ensureComparableEmissions plan item.record item.emissions
    let action ← singleEmission plan item.record .action item.emissions
    let outcome ← singleEmission plan item.record .outcome item.emissions
    let state ← singleEmission plan item.record .state item.emissions
    let observations := item.emissions.filter fun emission => emission.rule.outputKind == .observation
    let usable := item.emissions.filter fun emission =>
      emission.rule.outputKind == .action || emission.rule.outputKind == .outcome ||
        emission.rule.outputKind == .state || emission.rule.outputKind == .observation
    if usable.length != item.emissions.length then
      throw (diagnostic plan .unconsumedReference [item.record.id])
    steps := steps ++ [{
      selectedAction := action.value
      modelOutcome := outcome.value
      resultingState := state.value
      observations := observations.map Emission.value
    }]
    evidenceLinks := evidenceLinks ++ [
      evidenceLinkFor plan bundle (.selectedAction stepPosition) action,
      evidenceLinkFor plan bundle (.modelOutcome stepPosition) outcome,
      evidenceLinkFor plan bundle (.resultingState stepPosition) state
    ] ++ observations.mapIdx fun observationIndex observation =>
      evidenceLinkFor plan bundle (.observation stepPosition (observationIndex + 1)) observation
    stepPosition := stepPosition + 1
  let trace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
    initialState := initial.value
    steps
  }
  let evidenceIdentities := records.map SyntheticEvidenceRecord.id
  let recordSupport ← records.mapM (evidenceRecordSupport plan)
  let unchecked : UncheckedEvidenceBackedTrace := {
    traceId := evidenceBackedTraceId plan.behaviorFingerprint.render evidenceIdentities recordSupport trace
      evidenceLinks
    checkedPlan := plan
    mappingId := plan.id
    mappingVersion := plan.version
    mappingDigest := plan.behaviorFingerprint.render
    source := plan.source
    profileId := plan.profile.id
    profileVersion := plan.profile.version
    sourceClosed := true
    vocabulary := plan.meanings
    dispositions := plan.dispositions
    appliedBound := plan.evidenceBound
    evidenceIdentities
    recordSupport
    trace
    evidenceLinks
  }
  pure unchecked

private def evidenceLinkEvidenceIds (evidenceLinks : List EvidenceLink) : List DefinitionId :=
  canonicalIds (evidenceLinks.flatMap EvidenceLink.evidenceIdentities)

private def orderingFactByRecordLe (left right : EvidenceOrderingFact) : Bool :=
  idLe left.recordId right.recordId

private def orderingFactBySequenceLe (left right : EvidenceOrderingFact) : Bool :=
  match left.origin, right.origin with
  | some leftOrigin, some rightOrigin =>
      decide (leftOrigin.source.value < rightOrigin.source.value) ||
        (leftOrigin.source == rightOrigin.source &&
          (leftOrigin.ordinal < rightOrigin.ordinal ||
            (leftOrigin.ordinal == rightOrigin.ordinal && idLe left.recordId right.recordId)))
  | none, none => left.sequence < right.sequence ||
      (left.sequence == right.sequence && idLe left.recordId right.recordId)
  | none, some _ => true
  | some _, none => false

private partial def orderingDependsOn
    (facts : List EvidenceOrderingFact)
    (recordId target : DefinitionId)
    (visited : List DefinitionId := []) : Bool :=
  if recordId == target then true
  else if visited.contains recordId then false
  else
    match facts.find? fun fact => fact.recordId == recordId with
    | none => false
    | some fact => fact.causalParents.any fun parent =>
        orderingDependsOn facts parent target (recordId :: visited)

private def canonicalOrderingFacts
    (mappingId : DefinitionId) :
    List EvidenceOrderingFact → Except ObservationDiagnostic (List EvidenceOrderingFact)
  | [] => pure []
  | [fact] => pure [fact]
  | first :: second :: rest =>
      if first.recordId == second.recordId then
        if first != second then
          throw {
            kind := .missingOrderSupport
            planId := mappingId
            relatedDefinitionIds := [first.recordId]
          }
        else
          canonicalOrderingFacts mappingId (second :: rest)
      else
        return first :: (← canonicalOrderingFacts mappingId (second :: rest))

private def validateOrderingProvenance
    (trace : UncheckedEvidenceBackedTrace) : Except ObservationDiagnostic (List EvidenceOrderingFact) := do
  let ordered := (trace.evidenceLinks.flatMap EvidenceLink.orderingSupport).mergeSort
    orderingFactByRecordLe
  let facts := (← canonicalOrderingFacts trace.mappingId ordered).mergeSort orderingFactBySequenceLe
  if facts.map EvidenceOrderingFact.recordId != trace.evidenceIdentities then
    throw { kind := .missingOrderSupport, planId := trace.mappingId }
  if facts.all fun fact => fact.origin.isSome then
    let sources := canonicalIds <| facts.filterMap fun fact => fact.origin.map EvidenceOrigin.source
    for source in sources do
      let sourceFacts := facts.filter fun fact =>
        fact.origin.any fun origin => origin.source == source
      let mut expectedOrdinal := 0
      for fact in sourceFacts do
        let origin ← match fact.origin with
          | some origin => pure origin
          | none => throw { kind := .missingOrderSupport, planId := trace.mappingId }
        if origin.ordinal != expectedOrdinal then
          throw {
            kind := .missingOrderSupport
            planId := trace.mappingId
            relatedDefinitionIds := [fact.recordId, source]
          }
        expectedOrdinal := expectedOrdinal + 1
    for fact in facts do
      for parent in fact.causalParents do
        let parentFact ← match facts.find? fun candidate => candidate.recordId == parent with
          | some parentFact => pure parentFact
          | none => throw {
              kind := .missingOrderSupport
              planId := trace.mappingId
              relatedDefinitionIds := [fact.recordId, parent]
            }
        if orderingDependsOn facts parent fact.recordId then
          throw {
            kind := .missingOrderSupport
            planId := trace.mappingId
            relatedDefinitionIds := [fact.recordId, parent]
          }
        match fact.origin, parentFact.origin with
        | some factOrigin, some parentOrigin =>
            if factOrigin.source == parentOrigin.source &&
                parentOrigin.ordinal >= factOrigin.ordinal then
              throw {
                kind := .missingOrderSupport
                planId := trace.mappingId
                relatedDefinitionIds := [fact.recordId, parent]
              }
        | _, _ => throw { kind := .missingOrderSupport, planId := trace.mappingId }
    for evidenceLink in trace.evidenceLinks do
      let support := evidenceLink.orderingSupport.mergeSort orderingFactBySequenceLe
      if support != facts then
        throw {
          kind := .missingOrderSupport
          planId := trace.mappingId
          relatedDefinitionIds := [evidenceLink.ruleId]
        }
    return facts
  if facts.any fun fact => fact.origin.isSome then
    throw { kind := .missingOrderSupport, planId := trace.mappingId }
  let mut expectedSequence := 1
  for fact in facts do
    if fact.sequence != expectedSequence ||
        (expectedSequence > 1 && fact.causalParents.isEmpty) then
      throw {
        kind := .missingOrderSupport
        planId := trace.mappingId
        relatedDefinitionIds := [fact.recordId]
      }
    for parent in fact.causalParents do
      let parentFact ← match facts.find? fun candidate => candidate.recordId == parent with
        | some parentFact => pure parentFact
        | none => throw {
            kind := .missingOrderSupport
            planId := trace.mappingId
            relatedDefinitionIds := [fact.recordId, parent]
          }
      if parentFact.sequence >= fact.sequence then
        throw {
          kind := .missingOrderSupport
          planId := trace.mappingId
          relatedDefinitionIds := [fact.recordId, parent]
        }
    expectedSequence := expectedSequence + 1
  for evidenceLink in trace.evidenceLinks do
    let support := evidenceLink.orderingSupport.mergeSort orderingFactByRecordLe
    if canonicalIds (support.map EvidenceOrderingFact.recordId) !=
        canonicalIds evidenceLink.evidenceIdentities ||
        support.any fun fact => !facts.contains fact then
      throw {
        kind := .missingOrderSupport
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId]
      }
  pure facts

private def validateClosureProvenance
    (trace : UncheckedEvidenceBackedTrace)
    (ordering : List EvidenceOrderingFact) : Except ObservationDiagnostic Unit := do
  let closures := trace.evidenceLinks.head?.map fun evidenceLink =>
    evidenceLink.closureSupport.mergeSort closureLe
  let closures := closures.getD []
  if closures.isEmpty || !trace.sourceClosed then
    throw { kind := .missingClosureSupport, planId := trace.mappingId }
  match firstDuplicateClosure closures with
  | some duplicate => throw {
      kind := .missingClosureSupport
      planId := trace.mappingId
      relatedDefinitionIds := [duplicate.kind]
    }
  | none => pure ()
  for evidenceLink in trace.evidenceLinks do
    if evidenceLink.closureSupport.mergeSort closureLe != closures then
      throw {
        kind := .missingClosureSupport
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId]
      }
  if ordering.all fun fact => fact.origin.isSome then
    for fact in ordering do
      let origin ← match fact.origin with
        | some origin => pure origin
        | none => throw { kind := .missingClosureSupport, planId := trace.mappingId }
      if !(closures.any fun closure => closure.source == some origin.source &&
          closure.kind == fact.kind) then
        throw {
          kind := .missingClosureSupport
          planId := trace.mappingId
          relatedDefinitionIds := [fact.recordId, origin.source, fact.kind]
        }
    for closure in closures do
      let source ← match closure.source with
        | some source => pure source
        | none => throw {
            kind := .missingClosureSupport
            planId := trace.mappingId
            relatedDefinitionIds := [closure.kind]
          }
      let sourceFacts := ordering.filter fun fact =>
        fact.kind == closure.kind && fact.origin.any fun origin => origin.source == source
      let lastSequence := sourceFacts.foldl (fun current fact =>
        Nat.max current (fact.origin.map (fun origin => origin.ordinal + 1) |>.getD 0)) 0
      if sourceFacts.isEmpty || closure.lastSequence != lastSequence ||
          closure.recordCount != some sourceFacts.length || closure.byteCount.isNone then
        throw {
          kind := .missingClosureSupport
          planId := trace.mappingId
          relatedDefinitionIds := [source, closure.kind]
        }
    return
  if ordering.any fun fact => fact.origin.isSome then
    throw { kind := .missingClosureSupport, planId := trace.mappingId }
  for fact in ordering do
    if !(closures.any fun closure => closure.kind == fact.kind) then
      throw {
        kind := .missingClosureSupport
        planId := trace.mappingId
        relatedDefinitionIds := [fact.recordId, fact.kind]
      }
  for closure in closures do
    let lastSequence := (ordering.filter fun fact => fact.kind == closure.kind)
      |>.foldl (fun current fact => Nat.max current fact.sequence) 0
    if closure.lastSequence != lastSequence then
      throw {
        kind := .missingClosureSupport
        planId := trace.mappingId
        relatedDefinitionIds := [closure.kind]
      }

private def validateAppliedDisposition
    (trace : UncheckedEvidenceBackedTrace)
    (evidenceLink : EvidenceLink)
    (applied : AppliedFieldDisposition) : Except ObservationDiagnostic Unit := do
  let expected ← match trace.dispositions.find? fun declaration => declaration.field == applied.field with
    | some declaration => pure declaration.disposition
    | none => throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
      }
  match applied.evidence with
  | .raw _ => throw {
      kind := .rawValueLeakage
      planId := trace.mappingId
      relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
    }
  | .rejectedMaterial _ => throw {
      kind := .rejectedValueLeakage
      planId := trace.mappingId
      relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
    }
  | .retained _ =>
      match expected with
      | .retain => pure ()
      | .redact => throw {
          kind := .redactedValueLeakage
          planId := trace.mappingId
          relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
        }
      | .reject => throw {
          kind := .rejectedValueLeakage
          planId := trace.mappingId
          relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
        }
      | .hash _ => throw {
          kind := .digestPolicyMismatch
          planId := trace.mappingId
          relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
        }
  | .redactedContribution =>
      if expected == .redact then pure () else throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
      }
  | .digestToken policy _ =>
      if expected == .hash (some policy) then pure () else throw {
        kind := .digestPolicyMismatch
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field, policy]
      }

private def renderedEvidenceValue
    (valueType : ObservationValueType)
    (rendered : String) : Option EvidenceValue :=
  match valueType with
  | .text => some (.text rendered)
  | .natural => rendered.toNat?.map EvidenceValue.natural
  | .boolean =>
      if rendered == "true" then some (.boolean true)
      else if rendered == "false" then some (.boolean false)
      else none

private partial def evaluateProvenanceExpression
    (trace : UncheckedEvidenceBackedTrace)
    (evidenceLink : EvidenceLink)
    (expression : CheckedObservationExpression)
    (visited : List DefinitionId := []) : Except ObservationDiagnostic EvidenceValue := do
  let failure : ObservationDiagnostic := {
    kind := .inconsistentEvidenceLink
    planId := trace.mappingId
    relatedDefinitionIds := [evidenceLink.ruleId]
  }
  match expression with
  | .text value => pure (.text value)
  | .natural value => pure (.natural value)
  | .boolean value => pure (.boolean value)
  | .field reference valueType _ =>
      let applied ← match evidenceLink.appliedDispositions.find? fun item => item.field == reference with
        | some applied => pure applied
        | none => throw failure
      match applied.evidence with
      | .retained rendered =>
          match renderedEvidenceValue valueType rendered with
          | some value => pure value
          | none => throw failure
      | _ => throw failure
  | .binding id _ _ =>
      if visited.contains id then throw failure
      else match trace.checkedPlan.bindings.find? fun binding => binding.id == id with
        | some binding =>
            evaluateProvenanceExpression trace evidenceLink binding.expression (id :: visited)
        | none => throw failure
  | .normalize operator operand =>
      let value ← evaluateProvenanceExpression trace evidenceLink operand visited
      match operator, value with
      | .textTrimV1, .text text => pure (.text text.trimAscii.copy)
      | .textLowercaseV1, .text text => pure (.text text.toLower)
      | .naturalRenderV1, .natural value => pure (.text (toString value))
      | _, _ => throw failure
  | .present operand =>
      let references := canonicalReferences <|
        expressionReferences trace.checkedPlan operand visited
      if references.isEmpty then
        match evaluateProvenanceExpression trace evidenceLink operand visited with
        | .ok _ => pure (.boolean true)
        | .error _ => pure (.boolean false)
      else
        pure (.boolean (references.all fun reference =>
          evidenceLink.appliedDispositions.any fun applied => applied.field == reference))
  | .equals left right =>
      pure (.boolean ((← evaluateProvenanceExpression trace evidenceLink left visited) ==
        (← evaluateProvenanceExpression trace evidenceLink right visited)))
  | .and left right =>
      match ← evaluateProvenanceExpression trace evidenceLink left visited,
          ← evaluateProvenanceExpression trace evidenceLink right visited with
      | .boolean leftValue, .boolean rightValue => pure (.boolean (leftValue && rightValue))
      | _, _ => throw failure
  | .or left right =>
      match ← evaluateProvenanceExpression trace evidenceLink left visited,
          ← evaluateProvenanceExpression trace evidenceLink right visited with
      | .boolean leftValue, .boolean rightValue => pure (.boolean (leftValue || rightValue))
      | _, _ => throw failure
  | .not operand =>
      match ← evaluateProvenanceExpression trace evidenceLink operand visited with
      | .boolean value => pure (.boolean (!value))
      | _ => throw failure
  | .contributionMarker operand =>
      let references := canonicalReferences <|
        expressionReferences trace.checkedPlan operand visited
      if references.isEmpty || !(references.all fun reference =>
          evidenceLink.appliedDispositions.any fun applied =>
            applied.field == reference && applied.evidence == .redactedContribution) then
        throw failure
      pure (.text "contributed")
  | .digestToken policy operand =>
      let references := canonicalReferences <|
        expressionReferences trace.checkedPlan operand visited
      let tokens := references.filterMap fun reference =>
        (evidenceLink.appliedDispositions.find? fun applied => applied.field == reference).bind fun applied =>
          match applied.evidence with
          | .digestToken appliedPolicy token =>
              if appliedPolicy == policy.id then some token else none
          | _ => none
      match tokens with
      | [] => throw failure
      | first :: rest =>
          if tokens.length != references.length || !(rest.all fun token => token == first) then
            throw failure
          pure (.text first)

private def validateCheckedProvenance
    (trace : UncheckedEvidenceBackedTrace)
    (evidenceLink : EvidenceLink) : Except ObservationDiagnostic Unit := do
  let plan := trace.checkedPlan
  let rule ← match plan.rules.find? fun candidate => candidate.id == evidenceLink.ruleId with
    | some rule => pure rule
    | none => throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId]
      }
  let value ← match trace.trace.valueAt? evidenceLink.coordinate with
    | some value => pure value
    | none => throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId]
      }
  let expectedBindings := canonicalIds <|
    expressionBindingIds plan rule.value ++
      rule.condition.toList.flatMap (expressionBindingIds plan)
  let expectedReferences := canonicalReferences <|
    expressionReferences plan rule.value ++
      rule.condition.toList.flatMap (expressionReferences plan)
  let actualReferences := evidenceLink.appliedDispositions.map AppliedFieldDisposition.field
  let computedValue ← evaluateProvenanceExpression trace evidenceLink rule.value
  let conditionHolds ← match rule.condition with
    | none => pure true
    | some condition =>
        match ← evaluateProvenanceExpression trace evidenceLink condition with
        | .boolean value => pure value
        | _ => pure false
  if rule.output != value.definitionId ||
      rule.outputKind != evidenceLink.coordinate.definitionKind ||
      computedValue != .text value.value || !conditionHolds ||
      rule.meaning.canonicalBehavior != evidenceLink.meaningDigest ||
      evidenceLink.bindingIds != expectedBindings ||
      actualReferences != expectedReferences then
    throw {
      kind := .inconsistentEvidenceLink
      planId := trace.mappingId
      relatedDefinitionIds := [evidenceLink.ruleId]
    }

private def validateRecordSupport
    (trace : UncheckedEvidenceBackedTrace)
    (ordering : List EvidenceOrderingFact) : Except ObservationDiagnostic Unit := do
  if trace.recordSupport.map EvidenceRecordSupport.recordId != trace.evidenceIdentities then
    throw { kind := .unconsumedReference, planId := trace.mappingId }
  for support in trace.recordSupport do
    let order ← match ordering.find? fun fact => fact.recordId == support.recordId with
      | some order => pure order
      | none => throw {
          kind := .missingOrderSupport
          planId := trace.mappingId
          relatedDefinitionIds := [support.recordId]
        }
    if support.origin != order.origin || support.kind != order.kind ||
        support.causalParents != order.causalParents then
      throw {
        kind := .missingOrderSupport
        planId := trace.mappingId
        relatedDefinitionIds := [support.recordId]
      }
    let fieldIds := support.fields.map EvidenceFieldSupport.field
    if fieldIds.eraseDups.length != fieldIds.length then
      throw {
        kind := .contradictoryFact
        planId := trace.mappingId
        relatedDefinitionIds := [support.recordId]
      }
    for field in support.fields do
      let kind ← match trace.checkedPlan.profile.kinds.find? fun kind => kind.id == support.kind with
        | some kind => pure kind
        | none => throw {
            kind := .kindMismatch
            planId := trace.mappingId
            relatedDefinitionIds := [support.recordId, support.kind]
          }
      let declaration ← match kind.fields.find? fun declaration => declaration.id == field.field with
        | some declaration => pure declaration
        | none => throw {
            kind := .fieldMismatch
            planId := trace.mappingId
            relatedDefinitionIds := [support.recordId, field.field]
          }
      if declaration.valueType != field.valueType then
        throw {
          kind := .normalizationFailure
          planId := trace.mappingId
          relatedDefinitionIds := [support.recordId, field.field]
        }
      let disposition ← match trace.dispositions.find? fun item =>
          item.field == { kind := support.kind, field := field.field } with
        | some disposition => pure disposition.disposition
        | none => throw {
            kind := .disallowedRawMaterial
            planId := trace.mappingId
            relatedDefinitionIds := [support.recordId, field.field]
          }
      let valid := match disposition, field.evidence with
        | .retain, .retained _ => true
        | .redact, .redactedContribution => true
        | .hash (some expected), .digestToken actual _ => expected == actual
        | _, _ => false
      if !valid then
        throw {
          kind := .inconsistentEvidenceLink
          planId := trace.mappingId
          relatedDefinitionIds := [support.recordId, field.field]
        }

/-- Admit a complete unchecked carrier as an immutable semantic trace. -/
def validateEvidenceBackedTrace
    (trace : UncheckedEvidenceBackedTrace) : Except ObservationDiagnostic EvidenceBackedTrace := do
  let plan := trace.checkedPlan
  if !plan.hasCanonicalIdentity ||
      trace.mappingId != plan.id || trace.mappingVersion != plan.version ||
      trace.mappingDigest != plan.behaviorFingerprint.render || trace.source != plan.source ||
      trace.profileId != plan.profile.id || trace.profileVersion != plan.profile.version ||
      trace.vocabulary != plan.meanings || trace.dispositions != plan.dispositions ||
      trace.appliedBound != plan.evidenceBound then
    throw { kind := .inconsistentEvidenceLink, planId := trace.mappingId }
  if trace.evidenceIdentities.length > trace.appliedBound.value then
    throw {
      (diagnostic plan .evidenceBoundExhausted) with
      limit := some trace.appliedBound
      observedCount := some trace.evidenceIdentities.length
    }
  let expected := trace.trace.coordinates
  let actual := trace.evidenceLinks.map EvidenceLink.coordinate
  for coordinate in expected do
    let count := (actual.filter fun candidate => candidate == coordinate).length
    if count == 0 then
      throw { kind := .absentModelCoordinate, planId := trace.mappingId }
    if count > 1 then
      throw { kind := .duplicateModelCoordinate, planId := trace.mappingId }
  if actual.any fun coordinate => !expected.contains coordinate then
    throw { kind := .extraModelCoordinate, planId := trace.mappingId }
  for evidenceLink in trace.evidenceLinks do
    if evidenceLink.mappingId != trace.mappingId ||
        evidenceLink.mappingVersion != trace.mappingVersion ||
        evidenceLink.mappingDigest != trace.mappingDigest ||
        evidenceLink.profileId != trace.profileId ||
        evidenceLink.profileVersion != trace.profileVersion ||
        evidenceLink.appliedBound != trace.appliedBound ||
        evidenceLink.evidenceIdentities.isEmpty ||
        evidenceLink.evidenceIdentities.any (fun id => !trace.evidenceIdentities.contains id) ||
        !(trace.vocabulary.any fun meaning => meaning.canonicalBehavior == evidenceLink.meaningDigest) then
      throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId]
      }
  let linkedEvidence := evidenceLinkEvidenceIds trace.evidenceLinks
  if linkedEvidence.isEmpty || linkedEvidence.any fun identity =>
      !trace.evidenceIdentities.contains identity then
    throw { kind := .unconsumedReference, planId := trace.mappingId }
  if trace.recordSupport.map EvidenceRecordSupport.recordId != trace.evidenceIdentities then
    throw { kind := .unconsumedReference, planId := trace.mappingId }
  let ordering ← validateOrderingProvenance trace
  validateClosureProvenance trace ordering
  validateRecordSupport trace ordering
  for evidenceLink in trace.evidenceLinks do
    for applied in evidenceLink.appliedDispositions do
      validateAppliedDisposition trace evidenceLink applied
    validateCheckedProvenance trace evidenceLink
  if trace.traceId != evidenceBackedTraceId trace.mappingDigest trace.evidenceIdentities
      trace.recordSupport trace.trace trace.evidenceLinks then
    throw { kind := .inconsistentEvidenceLink, planId := trace.mappingId }
  pure (EvidenceBackedTrace.ofUnchecked trace)

/-- Evaluate Evidence without exposing an intermediate or partially constructed Model Trace. -/
def evaluateEvidence
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle) : ObservationResult :=
  match evaluateChecked plan bundle with
  | .ok unchecked =>
      match validateEvidenceBackedTrace unchecked with
      | .ok trace => .accepted trace
      | .error failure => resultOfDiagnostic failure
  | .error failure => resultOfDiagnostic failure

end Umpire
