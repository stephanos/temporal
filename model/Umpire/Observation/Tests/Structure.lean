import Umpire.Observation.Evaluation.Structure
import Umpire.Observation.Tests.Fixtures

/-! Normalized Observation structural-analysis behavior and exact finding order. -/

namespace Umpire.ObservationTests

open Umpire

def structuralKind : DefinitionId := id "test.evidence.kind.structural"
def structuralAuxiliaryKind : DefinitionId := id "test.evidence.kind.structural-auxiliary"
def structuralSourceA : DefinitionId := id "test.evidence.source.a"
def structuralSourceB : DefinitionId := id "test.evidence.source.b"
def structuralRuleA : DefinitionId := id "test.observation.rule.structural-a"
def structuralRuleB : DefinitionId := id "test.observation.rule.structural-b"

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

def structuralFailureOrderCases : List (List Observation.Internal.StructuralFinding) := [
  let recordId := id "test.evidence.record.duplicate"
  let fact := structuralFact recordId 1
  (Observation.Internal.analyzeStructure [fact, fact]
    [{ kind := structuralKind, lastSequence := 1 }] [structuralKind]).findings,
  let firstId := id "test.evidence.record.duplicate-sequence-a"
  let secondId := id "test.evidence.record.duplicate-sequence-b"
  (Observation.Internal.analyzeStructure
    [structuralFact firstId 1, structuralFact secondId 1]
    [{ kind := structuralKind, lastSequence := 1 }] [structuralKind]).findings,
  let firstId := id "test.evidence.record.gap-a"
  let secondId := id "test.evidence.record.gap-b"
  let missingId := id "test.evidence.record.missing"
  (Observation.Internal.analyzeStructure
    [structuralFact firstId 1, structuralFact secondId 3 none [missingId]]
    [{ kind := structuralKind, lastSequence := 3 }] [structuralKind]).findings,
  let firstId := id "test.evidence.record.reverse-a"
  let secondId := id "test.evidence.record.reverse-b"
  (Observation.Internal.analyzeStructure [
    structuralFact firstId 1
      (some { source := structuralSourceA, ordinal := 0 }) [secondId],
    structuralFact secondId 2
      (some { source := structuralSourceA, ordinal := 1 })
  ] [{
    kind := structuralKind
    lastSequence := 2
    source := some structuralSourceA
    recordCount := some 2
    byteCount := some 32
  }] [structuralKind]).findings,
  let recordId := id "test.evidence.record.duplicate-closure"
  let closure : EvidenceClosureFact := { kind := structuralKind, lastSequence := 1 }
  (Observation.Internal.analyzeStructure [structuralFact recordId 1]
    [closure, closure] [structuralKind]).findings,
  let recordId := id "test.evidence.record.closure-sequence"
  (Observation.Internal.analyzeStructure [structuralFact recordId 1]
    [{ kind := structuralKind, lastSequence := 2 }] [structuralKind]).findings,
  let recordId := id "test.evidence.record.closure-byte"
  (Observation.Internal.analyzeStructure [
    structuralFact recordId 1 (some { source := structuralSourceA, ordinal := 0 })
  ] [{
    kind := structuralKind
    lastSequence := 1
    source := some structuralSourceA
    recordCount := some 1
  }] [structuralKind]).findings
]

/-- Identity, order, causality, and closure failures retain their normalized finding order. -/
example : structuralFailureOrderCases = [
  [.duplicateIdentity (id "test.evidence.record.duplicate") false],
  [
    .duplicateSequence
      (id "test.evidence.record.duplicate-sequence-a")
      (id "test.evidence.record.duplicate-sequence-b") 1,
    .sequenceGap (id "test.evidence.record.duplicate-sequence-b") none 2 1,
    .missingCausalParent (id "test.evidence.record.duplicate-sequence-b") none
  ],
  [
    .sequenceGap (id "test.evidence.record.gap-b") none 2 3,
    .missingCausalParent
      (id "test.evidence.record.gap-b") (some (id "test.evidence.record.missing"))
  ],
  [
    .contradictoryOrder
      (id "test.evidence.record.reverse-a") (id "test.evidence.record.reverse-b")
  ],
  [.duplicateClosure none structuralKind false],
  [.closureSequenceMismatch none structuralKind 1 2],
  [.closureByteCountMissing (some structuralSourceA) structuralKind]
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

def linkedStructuralFirstId : DefinitionId := id "test.evidence.record.linked-a"
def linkedStructuralSecondId : DefinitionId := id "test.evidence.record.linked-b"

def linkedStructuralFacts : List EvidenceOrderingFact := [
  structuralFact linkedStructuralFirstId 1
    (some { source := structuralSourceA, ordinal := 0 }),
  {
    structuralFact linkedStructuralSecondId 1
      (some { source := structuralSourceB, ordinal := 0 }) with
    kind := structuralAuxiliaryKind
  }
]

def linkedStructuralClosures : List EvidenceClosureFact := [
  {
    kind := structuralKind
    lastSequence := 1
    source := some structuralSourceA
    recordCount := some 1
    byteCount := some 16
  },
  {
    kind := structuralAuxiliaryKind
    lastSequence := 1
    source := some structuralSourceB
    recordCount := some 1
    byteCount := some 16
  }
]

def linkedStructuralSupport
    (ruleId : DefinitionId) : Observation.Internal.StructuralLinkSupport := {
  ruleId
  evidenceIdentities := [linkedStructuralFirstId, linkedStructuralSecondId]
  orderingSupport := linkedStructuralFacts
  closureSupport := linkedStructuralClosures
}

/-- Per-link duplicate closures are rejected without treating copies across links as duplicates. -/
example :
    let first := linkedStructuralSupport structuralRuleA
    let duplicate := first.closureSupport.head?.get (by native_decide)
    let withinLink := Observation.Internal.analyzeStructure [] []
      [structuralKind, structuralAuxiliaryKind] [{
        first with closureSupport := duplicate :: first.closureSupport
      }]
    let second := linkedStructuralSupport structuralRuleB
    let laterLink := Observation.Internal.analyzeStructure [] []
      [structuralKind, structuralAuxiliaryKind] [
        first,
        { second with closureSupport := duplicate :: second.closureSupport }
      ]
    let acrossLinks := Observation.Internal.analyzeStructure [] []
      [structuralKind, structuralAuxiliaryKind] [
        first,
        second
      ]
    (withinLink.links.map Observation.Internal.NormalizedStructuralLinkSupport.closures,
      withinLink.findings,
      laterLink.findings,
      acrossLinks.findings) = ([linkedStructuralClosures], [
        .duplicateClosureSupport structuralRuleA 0
          (some structuralSourceA) structuralKind false
      ], [
        .duplicateClosureSupport structuralRuleB 1
          (some structuralSourceA) structuralKind false
      ], []) := by
  native_decide

/-- Missing support on one link identifies that link without re-analyzing the shared union. -/
example :
    let second := linkedStructuralSupport structuralRuleB
    let analysis := Observation.Internal.analyzeStructure [] []
      [structuralKind, structuralAuxiliaryKind] [
        linkedStructuralSupport structuralRuleA,
        {
          second with
          orderingSupport := second.orderingSupport.tail
          closureSupport := second.closureSupport.tail
        }
      ]
    analysis.findings = [
      .inconsistentOrderingSupport structuralRuleB
        linkedStructuralFacts linkedStructuralFacts.tail,
      .inconsistentClosureSupport structuralRuleB
        linkedStructuralClosures linkedStructuralClosures.tail
    ] := by
  native_decide

/-- Reordered link support is normalized once and retained under the responsible rule identity. -/
example :
    let second := linkedStructuralSupport structuralRuleB
    let analysis := Observation.Internal.analyzeStructure [] []
      [structuralKind, structuralAuxiliaryKind] [
        linkedStructuralSupport structuralRuleA,
        {
          second with
          orderingSupport := second.orderingSupport.reverse
          closureSupport := second.closureSupport.reverse
        }
      ]
    (analysis.links, analysis.findings) = ([
      {
        ruleId := structuralRuleA
        evidenceIdentities := [linkedStructuralFirstId, linkedStructuralSecondId]
        facts := linkedStructuralFacts
        closures := linkedStructuralClosures
      },
      {
        ruleId := structuralRuleB
        evidenceIdentities := [linkedStructuralFirstId, linkedStructuralSecondId]
        facts := linkedStructuralFacts
        closures := linkedStructuralClosures
      }
    ], []) := by
  native_decide

end Umpire.ObservationTests
