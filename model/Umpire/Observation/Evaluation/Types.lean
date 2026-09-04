import Umpire.Observation.Compiler
import Umpire.SemanticInventory.Types

/-!
Inert Observation Evaluation contracts for raw Evidence, diagnostics, structural support,
Evidence Links, and the unchecked carrier used during accepted-trace admission.
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

end Umpire
