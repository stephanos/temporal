import Umpire.Observation.Tests.Structure

/-! Pure evaluation behavior and exact R2/R4 status boundaries. -/

namespace Umpire.ObservationTests

open Umpire

def completeEvaluation : ObservationResult :=
  evaluateFixture completeEvidence

def tenfoldEvaluationPlan : CheckedObservationPlan :=
  (checkObservation evaluationContext {
    evaluationDeclaration with evidenceBound := { value := 20, unit := .evidenceRecords }
  }).toOption.get (by native_decide)

def tenfoldEvaluationEvidence : EvidenceBundle := {
  completeEvidence with
  records := initialEvidence :: (List.range 19).map fun offset =>
    let sequence := offset + 2
    let recordId := id ("test.evidence.record.scale-" ++ toString sequence)
    let parentId := if offset == 0 then initialEvidenceId
      else id ("test.evidence.record.scale-" ++ toString (sequence - 1))
    { stepEvidence with id := recordId, sequence, causalParents := [parentId] }
  closures := [{ kind := eventKind, lastSequence := 20 }]
}

/-- Ten times the ordinary evidence size retains one complete accepted admission. -/
example :
    let result := evaluateEvidence tenfoldEvaluationPlan tenfoldEvaluationEvidence
    (result.status, (acceptedOf result).map fun trace =>
      (trace.evidenceIdentities.length, trace.evidenceLinks.length)) =
      (.accepted, some (20, 96)) := by
  native_decide

def zeroRecordKind : DefinitionId := id "test.evidence.kind.zero-record"

def zeroRecordEvaluationContext : ObservationCheckContext :=
  let profile := evaluationContext.profiles.head?.get (by native_decide)
  {
    evaluationContext with profiles := [{
      profile with kinds := profile.kinds ++ [{ id := zeroRecordKind, fields := [] }]
    }]
  }

def zeroRecordEvaluationDeclaration : ObservationMappingDeclaration := {
  evaluationDeclaration with closures := [{ kind := eventKind }, { kind := zeroRecordKind }]
}

def zeroRecordEvaluationPlan : CheckedObservationPlan :=
  (checkObservation zeroRecordEvaluationContext zeroRecordEvaluationDeclaration).toOption.get
    (by native_decide)

def zeroRecordClosure : EvidenceClosureFact := {
  kind := zeroRecordKind
  lastSequence := 0
}

def zeroRecordEvidence : EvidenceBundle := {
  completeEvidence with closures := completeEvidence.closures ++ [zeroRecordClosure]
}

/-- An explicit zero-record closure satisfies its checked global closure requirement. -/
example :
    let result := evaluateEvidence zeroRecordEvaluationPlan zeroRecordEvidence
    (result.status, (acceptedOf result).map fun trace => trace.trace) =
      (.accepted, some expectedTrace) := by
  native_decide

/-- Competing global closure failures retain checked-plan declaration precedence. -/
example :
    let result := evaluateEvidence zeroRecordEvaluationPlan {
      completeEvidence with closures := [{ zeroRecordClosure with lastSequence := 1 }]
    }
    (result.status, result.diagnostic?.map fun failure =>
      (failure.kind, failure.relatedDefinitionIds), acceptedOf result) =
      (.unknown, some (.missingClosure, [eventKind]), none) := by
  native_decide

/-- Complete closed evidence produces the independently authored Model Trace. -/
example : (acceptedOf completeEvaluation).map EvidenceBackedTrace.trace = some expectedTrace := by
  native_decide

/-- The exact evidence-record limit follows ordinary evaluation. -/
example : (acceptedOf (evaluateFixture {
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
    let overLimit := evaluateFixture {
      completeEvidence with records := completeEvidence.records ++ [
        { stepEvidence with id := secondStepEvidenceId, sequence := 3 },
        { stepEvidence with id := id "test.evidence.record.step-3", sequence := 4 }
      ]
    }
    (resultStatusOf overLimit, resultKindOf overLimit, acceptedOf overLimit) =
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

def observationEvaluationFailureCases : List (ObservationStatus × Option ObservationFailureKind) := [
  let result := evaluateFixture { completeEvidence with records := [] }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := completeEvidence.records ++ [
      { stepEvidence with id := secondStepEvidenceId, sequence := 3 },
      { stepEvidence with id := id "test.evidence.record.step-3", sequence := 4 }
    ]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
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
  let result := evaluateFixture { completeEvidence with closures := [] }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with
    records := [initialEvidence, { stepEvidence with sequence := 3 }]
    closures := [{ kind := eventKind, lastSequence := 3 }]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with causalParents := [id "test.evidence.record.missing"]
    }]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [{
      initialEvidence with fields := [
        textField roleField "initial",
        { field := nameField, value := .natural 1 }
      ]
    }, stepEvidence]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with bindingFacts := [{
        binding := id "test.binding.unknown"
        value := .text "unresolved"
      }]
    }]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with
    records := [initialEvidence, { stepEvidence with sequence := 1 }]
    closures := [{ kind := eventKind, lastSequence := 1 }]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with profile := id "test.evidence.profile.other"
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture { completeEvidence with profileVersion := 2 }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [{ initialEvidence with kind := id "test.evidence.kind.other" }]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [fieldMismatchRecord, stepEvidence]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [initialEvidence, { stepEvidence with id := initialEvidenceId }]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [conflictingFactRecord, stepEvidence]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with bindingFacts := [
        { binding := normalizedName.id, value := .text "one" },
        { binding := normalizedName.id, value := .text "two" }
      ]
    }]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with causalParents := [stepEvidenceId]
    }]
  }
  (result.status, resultKindOf result),
  let result := evaluateFixture {
    completeEvidence with records := [initialEvidence, {
      stepEvidence with faultTarget := some stepEvidenceId
    }]
  }
  (result.status, resultKindOf result)
]

/-- Every enumerated R2 failure has an exact semantic status and diagnostic. -/
example : observationEvaluationFailureCases = [
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
    let forward := evaluateFixture ambiguousEvidence
    let reverse := evaluateFixture {
      ambiguousEvidence with compatibleAlternatives := ambiguousEvidence.compatibleAlternatives.reverse
    }
    (forward, reverse) = (
      .unknown {
        kind := .compatibleAlternatives
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [id "test.interpretation.a", id "test.interpretation.b"]
        alternatives := [id "test.interpretation.a", id "test.interpretation.b"]
        missingDiscriminator := some (id "test.evidence.field.discriminator")
      },
      forward) := by
  native_decide

/-- Compatible alternatives without their missing discriminator fail as unresolved input. -/
example :
    let result := evaluateFixture { ambiguousEvidence with missingDiscriminator := none }
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
    let result := evaluateFixture contradictoryAlternativeEvidence
    (result.status, resultKindOf result, acceptedOf result) =
      (.conflict, some .contradictoryFact, none) := by
  native_decide

/-- Duplicate closure facts fail closed through the raw structural adapter without a trace. -/
example :
    let result := evaluateFixture {
      completeEvidence with closures := completeEvidence.closures ++ completeEvidence.closures
    }
    (result.status, result.diagnostic?.map fun failure =>
      (failure.kind, failure.relatedDefinitionIds), acceptedOf result) =
      (.unknown, some (.missingClosure, [eventKind]), none) := by
  native_decide

/-- Raw identity and origin failures retain their precedence and complete related identities. -/
example :
    let duplicateBeforeMixed := evaluateFixture {
      completeEvidence with records := [
        { initialEvidence with origin := some { source := structuralSourceA, ordinal := 0 } },
        { stepEvidence with id := initialEvidenceId }
      ]
    }
    let mixed := evaluateFixture {
      completeEvidence with records := [
        { initialEvidence with origin := some { source := structuralSourceA, ordinal := 0 } },
        stepEvidence
      ]
    }
    ([duplicateBeforeMixed, mixed].map fun result =>
      (result.status, result.diagnostic?.map fun failure =>
        (failure.kind, failure.relatedDefinitionIds), acceptedOf result)) = [
      (.conflict, some (.duplicateEvidenceIdentity, [initialEvidenceId]), none),
      (.unknown, some (.incomparableOrdering, [initialEvidenceId, stepEvidenceId]), none)
    ] := by
  native_decide

/-- Receipt checks precede global gaps, while source causality precedes receipt checks. -/
example :
    let globalFaultBeforeGap := evaluateFixture {
      completeEvidence with records := [initialEvidence, {
        stepEvidence with sequence := 3, faultTarget := some stepEvidenceId
      }]
    }
    let missingParentId := id "test.evidence.record.missing"
    let sourceCausalityBeforeFault := evaluateFixture {
      completeEvidence with
      records := [
        { initialEvidence with origin := some { source := structuralSourceA, ordinal := 0 } },
        { stepEvidence with
          origin := some { source := structuralSourceA, ordinal := 1 }
          causalParents := [missingParentId]
          faultTarget := some stepEvidenceId
        }
      ]
      closures := [{
        kind := eventKind
        lastSequence := 2
        source := some structuralSourceA
        recordCount := some 2
        byteCount := some 64
      }]
    }
    ([globalFaultBeforeGap, sourceCausalityBeforeFault].map fun result =>
      (result.status, result.diagnostic?.map fun failure =>
        (failure.kind, failure.relatedDefinitionIds), acceptedOf result)) = [
      (.conflict, some (.misdirectedFaultReceipt, [stepEvidenceId]), none),
      (.unknown, some (.missingCausalParent, [missingParentId, stepEvidenceId]), none)
    ] := by
  native_decide

end Umpire.ObservationTests
