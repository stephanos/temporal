import Umpire.Observation.Tests.Fixtures

/-! Pure evaluation behavior and exact R2/R4 status boundaries. -/

namespace Umpire.ObservationTests

open Umpire

def completeEvaluation : ObservationResult :=
  evaluateFixture completeEvidence

def structuralKind : DefinitionId := id "test.evidence.kind.structural"
def structuralSourceA : DefinitionId := id "test.evidence.source.a"
def structuralSourceB : DefinitionId := id "test.evidence.source.b"

def structuralFact
    (recordId : DefinitionId)
    (sequence : Nat)
    (origin : Option EvidenceOrigin := none)
    (causalParents : List DefinitionId := []) : EvidenceOrderingFact := {
  recordId
  kind := structuralKind
  sequence
  origin
  causalParents
}

/-- Empty structural input retains the required-kind closure failure without inventing facts. -/
example :
    let analysis := Observation.Internal.analyzeStructure [] [] [structuralKind]
    (analysis.facts, analysis.originMode, analysis.closureExpectations, analysis.findings) =
      ([], .globalSequence, [], [.missingRequiredKind structuralKind]) := by
  native_decide

/-- Single-source facts and closures become one reusable sequence-ordered fact set. -/
example :
    let firstId := id "test.evidence.record.structural-1"
    let secondId := id "test.evidence.record.structural-2"
    let analysis := Observation.Internal.analyzeStructure
      [structuralFact secondId 2 none [firstId], structuralFact firstId 1]
      [{ kind := structuralKind, lastSequence := 2 }]
      [structuralKind]
    (analysis.facts.map EvidenceOrderingFact.recordId,
      analysis.originMode,
      analysis.closureExpectations,
      analysis.findings) =
      ([firstId, secondId], .globalSequence,
        [{
          source := none
          kind := structuralKind
          recordIds := [firstId, secondId]
          lastSequence := 2
          recordCount := 2
        }], []) := by
  native_decide

/-- Structural closure coverage follows normalized facts without a separate required-kind list. -/
example :
    let recordId := id "test.evidence.record.unclosed"
    let analysis := Observation.Internal.analyzeStructure [structuralFact recordId 1] []
    analysis.findings = [.missingClosure [recordId] none structuralKind] := by
  native_decide

/-- Multi-source facts retain canonical source-local order and closure calculations. -/
example :
    let a0 := id "test.evidence.record.a-0"
    let a1 := id "test.evidence.record.a-1"
    let b0 := id "test.evidence.record.b-0"
    let analysis := Observation.Internal.analyzeStructure [
      structuralFact b0 1 (some { source := structuralSourceB, ordinal := 0 }),
      structuralFact a1 2 (some { source := structuralSourceA, ordinal := 1 }) [a0],
      structuralFact a0 1 (some { source := structuralSourceA, ordinal := 0 })
    ] [
      { kind := structuralKind, lastSequence := 1, source := some structuralSourceB,
        recordCount := some 1, byteCount := some 16 },
      { kind := structuralKind, lastSequence := 2, source := some structuralSourceA,
        recordCount := some 2, byteCount := some 32 }
    ] [structuralKind]
    (analysis.facts.map EvidenceOrderingFact.recordId,
      analysis.originMode,
      analysis.closureExpectations,
      analysis.findings) =
      ([a0, a1, b0], .sourceSequence, [
        {
          source := some structuralSourceA
          kind := structuralKind
          recordIds := [a0, a1]
          lastSequence := 2
          recordCount := 2
        },
        {
          source := some structuralSourceB
          kind := structuralKind
          recordIds := [b0]
          lastSequence := 1
          recordCount := 1
        }
      ], []) := by
  native_decide

/-- Causal cycles retain both record and parent identities for boundary-owned diagnostics. -/
example :
    let firstId := id "test.evidence.record.cycle-1"
    let secondId := id "test.evidence.record.cycle-2"
    let analysis := Observation.Internal.analyzeStructure
      [structuralFact firstId 1 none [secondId], structuralFact secondId 2 none [firstId]]
      [{ kind := structuralKind, lastSequence := 2 }]
      [structuralKind]
    analysis.findings = [
      .contradictoryOrder firstId secondId,
      .contradictoryOrder secondId firstId
    ] := by
  native_decide

/-- Mixed origin modes are represented with the complete normalized identity order. -/
example :
    let globalId := id "test.evidence.record.global"
    let sourcedId := id "test.evidence.record.sourced"
    let analysis := Observation.Internal.analyzeStructure [
      structuralFact sourcedId 2 (some { source := structuralSourceA, ordinal := 0 }),
      structuralFact globalId 1
    ] [] [structuralKind]
    (analysis.originMode, analysis.findings) =
      (.mixed, [.mixedOrigins [globalId, sourcedId]]) := by
  native_decide

/-- Source closure count mismatches retain expected and supplied counts. -/
example :
    let firstId := id "test.evidence.record.count-1"
    let secondId := id "test.evidence.record.count-2"
    let analysis := Observation.Internal.analyzeStructure [
      structuralFact firstId 1 (some { source := structuralSourceA, ordinal := 0 }),
      structuralFact secondId 2 (some { source := structuralSourceA, ordinal := 1 }) [firstId]
    ] [{
      kind := structuralKind
      lastSequence := 2
      source := some structuralSourceA
      recordCount := some 1
      byteCount := some 32
    }] [structuralKind]
    analysis.findings = [
      .closureCountMismatch (some structuralSourceA) structuralKind 2 (some 1)
    ] := by
  native_decide

def tenfoldStructuralFacts : List EvidenceOrderingFact :=
  (List.range 20).map fun ordinal =>
    let recordId := id ("test.evidence.record.scale-" ++ toString ordinal)
    let parents := if ordinal == 0 then [] else
      [id ("test.evidence.record.scale-" ++ toString (ordinal - 1))]
    structuralFact recordId (ordinal + 1)
      (some { source := structuralSourceA, ordinal }) parents

/-- Ten times the ordinary two-record fixture stays within one normalized analysis result. -/
example :
    let analysis := Observation.Internal.analyzeStructure tenfoldStructuralFacts [{
      kind := structuralKind
      lastSequence := 20
      source := some structuralSourceA
      recordCount := some 20
      byteCount := some 320
    }] [structuralKind]
    (analysis.facts.length,
      analysis.closureExpectations.map fun expectation =>
        (expectation.lastSequence, expectation.recordCount),
      analysis.findings) = (20, [(20, 20)], []) := by
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

end Umpire.ObservationTests
