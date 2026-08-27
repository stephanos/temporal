import Umpire.Observation.Tests.Fixtures

/-! Pure qualification behavior and exact R2/R4 status boundaries. -/

namespace Umpire.ObservationTests

open Umpire

def completeQualification : QualificationResult :=
  qualifyFixture completeEvidence

/-- Complete closed evidence produces the independently authored Model Trace. -/
example : (qualifiedOf completeQualification).map QualifiedTrace.trace = some expectedTrace := by
  native_decide

/-- The exact evidence-record limit follows ordinary qualification. -/
example : (qualifiedOf (qualifyFixture {
    completeEvidence with
    records := completeEvidence.records ++ [{
      stepEvidence with
      id := secondStepEvidenceId
      sequence := 3
      causalParents := [stepEvidenceId]
    }]
    closures := [{ kind := eventKind, lastSequence := 3 }]
  })).isSome = true := by
  native_decide

/-- Limit plus one is canonical unknown and exposes no partial trace. -/
example :
    let overLimit := qualifyFixture {
      completeEvidence with records := completeEvidence.records ++ [
        { stepEvidence with id := secondStepEvidenceId, sequence := 3 },
        { stepEvidence with id := id "test.evidence.record.step-3", sequence := 4 }
      ]
    }
    (resultStatusOf overLimit, resultKindOf overLimit, qualifiedOf overLimit) =
      (.unknown, some .evidenceBoundExhausted, none) := by
  native_decide

def fieldMismatchRecord : SyntheticEvidenceRecord := {
  initialEvidence with
  fields := initialEvidence.fields ++ [textField (id "test.evidence.field.unknown") "value"]
}

def conflictingFactRecord : SyntheticEvidenceRecord := {
  initialEvidence with
  fields := initialEvidence.fields ++ [textField roleField "step"]
}

def qualificationFailureCases : List (QualificationStatus × Option QualificationFailureKind) := [
  let result := qualifyFixture { completeEvidence with records := [] }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := completeEvidence.records ++ [
      { stepEvidence with id := secondStepEvidenceId, sequence := 3 },
      { stepEvidence with id := id "test.evidence.record.step-3", sequence := 4 }
    ]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with
    records := [{
      initialEvidence with fields := [
        textField roleField "other",
        textField nameField "ready"
      ]
    }]
    closures := [{ kind := eventKind, lastSequence := 1 }]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture { completeEvidence with closures := [] }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with
    records := [initialEvidence, { stepEvidence with sequence := 3 }]
    closures := [{ kind := eventKind, lastSequence := 3 }]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with causalParents := [id "test.evidence.record.missing"]
    }]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [{
      initialEvidence with fields := [
        textField roleField "initial",
        { field := nameField, value := .natural 1 }
      ]
    }, stepEvidence]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with bindingFacts := [{
        binding := id "test.binding.unknown"
        value := .text "unresolved"
      }]
    }]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with
    records := [initialEvidence, { stepEvidence with sequence := 1 }]
    closures := [{ kind := eventKind, lastSequence := 1 }]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with profile := id "test.evidence.profile.other"
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture { completeEvidence with profileVersion := 2 }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [{ initialEvidence with kind := id "test.evidence.kind.other" }]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [fieldMismatchRecord, stepEvidence]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [initialEvidence, { stepEvidence with id := initialEvidenceId }]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [conflictingFactRecord, stepEvidence]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with bindingFacts := [
        { binding := normalizedName.id, value := .text "one" },
        { binding := normalizedName.id, value := .text "two" }
      ]
    }]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with causalParents := [stepEvidenceId]
    }]
  }
  (result.status, resultKindOf result),
  let result := qualifyFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with faultTarget := some stepEvidenceId
    }]
  }
  (result.status, resultKindOf result)
]

/-- Every enumerated R2 failure has an exact semantic status and diagnostic. -/
example : qualificationFailureCases = [
  (.unknown, some .emptyEvidence),
  (.unknown, some .evidenceBoundExhausted),
  (.unknown, some .missingInitialState),
  (.unknown, some .missingClosure),
  (.unknown, some .sequenceGap),
  (.unknown, some .missingCausalParent),
  (.unknown, some .normalizationFailure),
  (.unknown, some .unresolvedBinding),
  (.unknown, some .incomparableOrdering),
  (.unsupported, some .profileMismatch),
  (.unsupported, some .profileVersionMismatch),
  (.unsupported, some .kindMismatch),
  (.unsupported, some .fieldMismatch),
  (.conflict, some .duplicateEvidenceIdentity),
  (.conflict, some .contradictoryFact),
  (.conflict, some .contradictoryBinding),
  (.conflict, some .contradictoryOrder),
  (.conflict, some .misdirectedFaultReceipt)
] := by
  native_decide

def ambiguousEvidence : EvidenceBundle := {
  completeEvidence with
  compatibleAlternatives := [
    { id := id "test.interpretation.b", evidenceIdentities := [stepEvidenceId] },
    { id := id "test.interpretation.a", evidenceIdentities := [initialEvidenceId] }
  ]
  missingDiscriminator := some (id "test.evidence.field.discriminator")
}

/-- Compatible interpretations remain canonical alternatives; input order never selects one. -/
example :
    let forward := qualifyFixture ambiguousEvidence
    let reverse := qualifyFixture {
      ambiguousEvidence with compatibleAlternatives := ambiguousEvidence.compatibleAlternatives.reverse
    }
    (forward, reverse) = (
      .unknown {
        kind := .compatibleAlternatives
        planId := qualificationDeclaration.id
        relatedDefinitionIds := [id "test.interpretation.a", id "test.interpretation.b"]
        alternatives := [id "test.interpretation.a", id "test.interpretation.b"]
        missingDiscriminator := some (id "test.evidence.field.discriminator")
      },
      forward) := by
  native_decide

/-- Compatible alternatives without their missing discriminator fail as unresolved input. -/
example :
    let result := qualifyFixture { ambiguousEvidence with missingDiscriminator := none }
    (result.status, resultKindOf result) = (.unknown, some .unresolvedBinding) := by
  native_decide

def contradictoryAlternativeEvidence : EvidenceBundle := {
  ambiguousEvidence with
  compatibleAlternatives := [
    { id := id "test.interpretation.same", evidenceIdentities := [initialEvidenceId] },
    { id := id "test.interpretation.same", evidenceIdentities := [stepEvidenceId] }
  ]
}

/-- One interpretation identity cannot silently collapse contradictory evidence sets. -/
example :
    let result := qualifyFixture contradictoryAlternativeEvidence
    (result.status, resultKindOf result, qualifiedOf result) =
      (.conflict, some .contradictoryFact, none) := by
  native_decide

end Umpire.ObservationTests
