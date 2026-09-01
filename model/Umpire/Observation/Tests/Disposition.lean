import Umpire.Observation.Tests.EvidenceLink

/-! Runtime disposition enforcement and forbidden raw-value non-retention. -/

namespace Umpire.ObservationTests

open Umpire

def dispositionFailureKinds : List (ObservationStatus × Option ObservationFailureKind) := [
  let evidenceLink := completeFirstEvidenceLink
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with evidenceLinks := [{
      evidenceLink with appliedDispositions := [{
        field := { kind := eventKind, field := secretField }
        evidence := .raw "forbidden-secret"
      }]
    }] ++ completeUncheckedEvidenceBackedTrace.evidenceLinks.tail
  }
  admissionStatusAndKind result,
  let evidenceLink := completeFirstEvidenceLink
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with evidenceLinks := [{
      evidenceLink with appliedDispositions := [{
        field := { kind := eventKind, field := secretField }
        evidence := .retained "forbidden-secret"
      }]
    }] ++ completeUncheckedEvidenceBackedTrace.evidenceLinks.tail
  }
  admissionStatusAndKind result,
  let evidenceLink := completeFirstEvidenceLink
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with evidenceLinks := [{
      evidenceLink with appliedDispositions := [{
        field := { kind := eventKind, field := rejectedField }
        evidence := .rejectedMaterial "forbidden-rejected"
      }]
    }] ++ completeUncheckedEvidenceBackedTrace.evidenceLinks.tail
  }
  admissionStatusAndKind result
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
  completeEvidence with
  records := [initialEvidence, {
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

def normalizedDigestRule : ObservationRule := {
  digestRule with
  value := .portable (.digestToken digestPolicyId
    (.normalize { name := "text.lowercase", version := 1 }
      (.normalize { name := "text.trim", version := 1 } (field hashedField))))
}

def normalizedDigestDeclaration : ObservationMappingDeclaration := {
  evaluationDeclaration with
  rules := evaluationDeclaration.rules.map fun rule =>
    if rule.id == digestRule.id then { rule with value := normalizedDigestRule.value } else rule
}

def normalizedDigestEvidence : EvidenceBundle := {
  completeEvidence with
  records := [initialEvidence, {
    stepEvidence with fields := stepEvidence.fields.map fun fieldValue =>
      if fieldValue.field == hashedField then
        { fieldValue with
          value := .text "  FORBIDDEN-HASH-MATERIAL  "
          reportedDigestToken := some (syntheticDigestToken digestPolicy "forbidden-hash-material") }
      else fieldValue
  }]
}

/-- Reported digest validation follows the checked normalized operand, not the raw field value. -/
example :
    let result := match checkObservation evaluationContext normalizedDigestDeclaration with
      | .ok plan => evaluateEvidence plan normalizedDigestEvidence
      | .error _ => .unknown {
          kind := .zeroUsableInterpretations
          planId := normalizedDigestDeclaration.id
        }
    result.status = .accepted := by
  native_decide

def irrelevantReportedTokenEvidence : EvidenceBundle :=
  let expectedToken := syntheticDigestToken digestPolicy "forbidden-hash-material"
  {
    completeEvidence with
    records := [{
      initialEvidence with fields := initialEvidence.fields.map fun fieldValue =>
        if fieldValue.field == nameField then
          { fieldValue with reportedDigestToken := some expectedToken }
        else fieldValue
    }, {
      stepEvidence with fields := stepEvidence.fields.map fun fieldValue =>
        if fieldValue.field == hashedField then
          { fieldValue with reportedDigestToken := some expectedToken }
        else fieldValue
    }]
  }

/-- Digest claims on non-hashed material cannot create a false same-bundle collision. -/
example : (evaluateFixture irrelevantReportedTokenEvidence).status = .accepted := by
  native_decide

/-- Runtime disposition failures are unsupported except true same-bundle digest collisions. -/
example :
    let rejected := evaluateFixture rejectedEvidence
    let mismatch := evaluateFixture digestMismatchEvidence
    let collision := evaluateFixture digestCollisionEvidence
    ((rejected.status, resultKindOf rejected),
      (mismatch.status, resultKindOf mismatch),
      (collision.status, resultKindOf collision)) =
      ((.unsupported, some .rejectedFieldPresent),
        (.unsupported, some .digestPolicyMismatch),
        (.conflict, some .digestCollision)) := by
  native_decide

/-- Evaluation output never retains redacted, hashed, or rejected raw material. -/
example :
    let rendered := reprStr completeEvaluation
    (rendered.contains "forbidden-secret",
      rendered.contains "forbidden-hash-material",
      rendered.contains "forbidden-rejected") = (false, false, false) := by
  native_decide

end Umpire.ObservationTests
