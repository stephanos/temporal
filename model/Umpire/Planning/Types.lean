import Umpire.Query

/-! Result metadata shared by artifact construction and the Planning implementation. -/

namespace Umpire

inductive KnownGapKind where
  | capabilityContract
  | input
  | interpretation
  | claim
  deriving BEq, DecidableEq, Ord, Repr

def KnownGapKind.name : KnownGapKind → String
  | .capabilityContract => "capability-contract"
  | .input => "input"
  | .interpretation => "interpretation"
  | .claim => "claim"

def KnownGapKind.parse? : String → Option KnownGapKind
  | "capability-contract" => some .capabilityContract
  | "input" => some .input
  | "interpretation" => some .interpretation
  | "claim" => some .claim
  | _ => none

/-- An exact, closed statement of behavior or evidence that is absent from the current result. -/
structure KnownGap where
  kind : KnownGapKind
  code : DefinitionId
  subject : Option DefinitionId := none
  detail : Option String := none
  deriving BEq, DecidableEq, Ord, Repr

inductive KnownGapErrorKind where
  | invalidCode
  | invalidSubject
  | duplicate
  | conflictingDetail
  | noncanonicalOrder
  deriving BEq, DecidableEq, Ord, Repr

def KnownGapErrorKind.name : KnownGapErrorKind → String
  | .invalidCode => "invalid-code"
  | .invalidSubject => "invalid-subject"
  | .duplicate => "duplicate"
  | .conflictingDetail => "conflicting-detail"
  | .noncanonicalOrder => "noncanonical-order"

structure KnownGapError where
  kind : KnownGapErrorKind
  code : DefinitionId
  subject : Option DefinitionId
  deriving BEq, DecidableEq, Repr

private def knownGapKindRank : KnownGapKind → String
  | .capabilityContract => "0"
  | .input => "1"
  | .interpretation => "2"
  | .claim => "3"

private def knownGapKey (gap : KnownGap) : String :=
  String.intercalate "\u001f" [
    knownGapKindRank gap.kind,
    gap.code.value,
    gap.subject.map DefinitionId.value |>.getD "",
    gap.detail.getD ""
  ]

private def knownGapLe (left right : KnownGap) : Bool :=
  decide (knownGapKey left ≤ knownGapKey right)

private def sameKnownGapSubject (left right : KnownGap) : Bool :=
  left.kind == right.kind && left.code == right.code && left.subject == right.subject

private def firstKnownGapProblem : List KnownGap → Option KnownGapError
  | first :: second :: rest =>
      if first == second then
        some { kind := .duplicate, code := second.code, subject := second.subject }
      else if sameKnownGapSubject first second && first.detail != second.detail then
        some { kind := .conflictingDetail, code := second.code, subject := second.subject }
      else
        firstKnownGapProblem (second :: rest)
  | _ => none

/-- Reject malformed, duplicate, conflicting, or noncanonically ordered Known Gaps. -/
def validateKnownGaps (gaps : List KnownGap) : Except KnownGapError Unit := do
  for gap in gaps do
    if !gap.code.isNamespaced then
      throw { kind := .invalidCode, code := gap.code, subject := gap.subject }
    match gap.subject with
    | some subject =>
        if !subject.isNamespaced then
          throw { kind := .invalidSubject, code := gap.code, subject := gap.subject }
    | none => pure ()
  let canonical := gaps.mergeSort knownGapLe
  if canonical != gaps then
    let gap := gaps.getD 0 {
      kind := .interpretation
      code := DefinitionId.of "umpire.known-gap.unknown"
    }
    throw { kind := .noncanonicalOrder, code := gap.code, subject := gap.subject }
  match firstKnownGapProblem canonical with
  | some problem => throw problem
  | none => pure ()

private def quote (value : String) : String := Lean.Json.compress (.str value)

def canonicalKnownGapJson (gap : KnownGap) : String :=
  "{\"kind\":" ++ quote gap.kind.name ++
    ",\"code\":" ++ quote gap.code.value ++
    ",\"subject\":" ++ (gap.subject.map (quote ∘ DefinitionId.value) |>.getD "null") ++
    ",\"detail\":" ++ (gap.detail.map quote |>.getD "null") ++ "}"

structure ExploredCounts where
  setups : Nat := 0
  traces : Nat := 0
  transitions : Nat := 0
  propertyEvaluations : Nat := 0
  deriving BEq, DecidableEq, Repr

structure PlanningCompleteness where
  established : Bool
  limits : QueryLimits
  finiteEvidenceFingerprints : List BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

structure PlanningMetadata where
  explored : ExploredCounts
  completeness : PlanningCompleteness
  deriving BEq, DecidableEq, Repr

inductive SelectionReason where
  | satisfyingWitness
  | violatingCounterexample
  | behaviorSelection
  deriving BEq, DecidableEq, Ord, Repr

end Umpire
