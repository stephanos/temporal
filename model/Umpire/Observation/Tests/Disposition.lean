import Umpire.Observation.Tests.Derivation

/-! Runtime disposition enforcement and forbidden raw-value non-retention. -/

namespace Umpire.ObservationTests

open Umpire

def dispositionFailureKinds : List (QualificationStatus × Option QualificationFailureKind) := [
  let derivation := completeQualifiedTrace.derivations.head!
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := [{
      derivation with appliedDispositions := [{
        field := { kind := eventKind, field := secretField }
        evidence := .raw "forbidden-secret"
      }]
    }] ++ completeQualifiedTrace.derivations.tail
  }
  (.unsupported, diagnosticKindOf result),
  let derivation := completeQualifiedTrace.derivations.head!
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := [{
      derivation with appliedDispositions := [{
        field := { kind := eventKind, field := secretField }
        evidence := .retained "forbidden-secret"
      }]
    }] ++ completeQualifiedTrace.derivations.tail
  }
  (.unsupported, diagnosticKindOf result),
  let derivation := completeQualifiedTrace.derivations.head!
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := [{
      derivation with appliedDispositions := [{
        field := { kind := eventKind, field := rejectedField }
        evidence := .rejectedMaterial "forbidden-rejected"
      }]
    }] ++ completeQualifiedTrace.derivations.tail
  }
  (.unsupported, diagnosticKindOf result)
]

example : dispositionFailureKinds = [
  (.unsupported, some .rawValueLeakage),
  (.unsupported, some .redactedValueLeakage),
  (.unsupported, some .rejectedValueLeakage)
] := by
  native_decide

def rejectedEvidence : EvidenceBundle := {
  completeEvidence with records := [initialEvidence, {
    stepEvidence with fields := stepEvidence.fields ++ [textField rejectedField "forbidden-rejected"]
  }]
}

def digestMismatchEvidence : EvidenceBundle := {
  completeEvidence with records := [initialEvidence, {
    stepEvidence with fields := stepEvidence.fields.map fun fieldValue =>
      if fieldValue.field == hashedField then
        { fieldValue with digestPolicy := some (id "test.digest.other") }
      else fieldValue
  }]
}

def digestCollisionEvidence : EvidenceBundle := {
  completeEvidence with records := [initialEvidence, {
    stepEvidence with fields := stepEvidence.fields.map fun fieldValue =>
      if fieldValue.field == hashedField then
        { fieldValue with reportedDigestToken := some "synthetic.digest/v1:collision" }
      else fieldValue
  }, {
    stepEvidence with
    id := secondStepEvidenceId
    sequence := 3
    causalParents := [stepEvidenceId]
    fields := stepEvidence.fields.map fun fieldValue =>
      if fieldValue.field == hashedField then
        { fieldValue with
          value := .text "different-hash-material"
          reportedDigestToken := some "synthetic.digest/v1:collision" }
      else fieldValue
  }]
  closures := [{ kind := eventKind, lastSequence := 3 }]
}

/-- Runtime disposition failures are unsupported except true same-bundle digest collisions. -/
example :
    let rejected := qualifyFixture rejectedEvidence
    let mismatch := qualifyFixture digestMismatchEvidence
    let collision := qualifyFixture digestCollisionEvidence
    ((rejected.status, resultKindOf rejected),
      (mismatch.status, resultKindOf mismatch),
      (collision.status, resultKindOf collision)) =
      ((.unsupported, some .rejectedFieldPresent),
        (.unsupported, some .digestPolicyMismatch),
        (.conflict, some .digestCollision)) := by
  native_decide

/-- Qualification output never retains redacted, hashed, or rejected raw material. -/
example :
    let rendered := reprStr completeQualification
    (rendered.contains "forbidden-secret",
      rendered.contains "forbidden-hash-material",
      rendered.contains "forbidden-rejected") = (false, false, false) := by
  native_decide

end Umpire.ObservationTests
