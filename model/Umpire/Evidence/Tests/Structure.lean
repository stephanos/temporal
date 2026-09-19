import Umpire.Evidence.Evaluate.Structure
import Umpire.Evidence.Tests.Fixtures

/-!
Evidence structure verdicts per audience. A raw bundle is analyzed from its facts, closures, and
fault receipts; an accepted envelope from its Evidence Links. Every expected fault is literal.
-/

namespace Umpire.EvidenceTests

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

/-- A raw bundle's structure: facts in bundle order, its closures, and its fault receipts. -/
def rawStructure
    (facts : List EvidenceOrderingFact)
    (closures : List EvidenceClosureFact)
    (requiredKinds : List DefinitionId := [structuralKind])
    (receipts : List EvidenceStructure.FaultReceipt := []) : EvidenceStructure :=
  EvidenceStructure.analyze facts closures requiredKinds (faultReceipts := receipts)

/-- An accepted envelope carrying the same facts and closures through one Evidence Link. -/
def acceptedStructure
    (evidenceIdentities : List DefinitionId)
    (facts : List EvidenceOrderingFact)
    (closures : List EvidenceClosureFact)
    (requiredKinds : List DefinitionId := [structuralKind]) : EvidenceStructure :=
  EvidenceStructure.analyze [] [] requiredKinds (some {
    evidenceIdentities
    links := [{
      ruleId := structuralRuleA
      evidenceIdentities
      orderingSupport := facts
      closureSupport := closures
    }]
  })

def rawVerdict (evidence : EvidenceStructure) :
    Option (EvidenceStructure.OrderingFault .raw) × Option EvidenceStructure.ClosureFault :=
  (evidence.orderingFault? .raw, evidence.closureFault? .raw)

def acceptedVerdict (evidence : EvidenceStructure) :
    Option (EvidenceStructure.OrderingFault .accepted) × Option EvidenceStructure.ClosureFault :=
  (evidence.orderingFault? .accepted, evidence.closureFault? .accepted)

/-- Empty input fails only the required closure; an empty envelope has no closure support at all. -/
example :
    (rawVerdict (rawStructure [] []), acceptedVerdict (acceptedStructure [] [] []),
      (rawStructure [] []).factsInOrder) =
      ((none, some { related := [structuralKind] }), (none, some { related := [] }), []) := by
  native_decide

/-- Single-source facts and closures become one sequence-ordered fact set for both audiences. -/
example :
    let firstId := id "test.evidence.record.structural-1"
    let secondId := id "test.evidence.record.structural-2"
    let facts := [structuralFact secondId 2 none [firstId], structuralFact firstId 1]
    let closures := [{ kind := structuralKind, lastSequence := 2 }]
    ((rawStructure facts closures).factsInOrder.map EvidenceOrderingFact.recordId,
      rawVerdict (rawStructure facts closures),
      acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)) =
      ([firstId, secondId], (none, none), (none, none)) := by
  native_decide

/-- Multi-source facts retain canonical source-local order and closure calculations. -/
example :
    let a0 := id "test.evidence.record.a-0"
    let a1 := id "test.evidence.record.a-1"
    let b0 := id "test.evidence.record.b-0"
    let facts := [
      structuralFact b0 1 (some { source := structuralSourceB, ordinal := 0 }),
      structuralFact a1 2 (some { source := structuralSourceA, ordinal := 1 }) [a0],
      structuralFact a0 1 (some { source := structuralSourceA, ordinal := 0 })
    ]
    let closures := [
      { kind := structuralKind, lastSequence := 1, source := some structuralSourceB,
        recordCount := some 1, byteCount := some 16 },
      { kind := structuralKind, lastSequence := 2, source := some structuralSourceA,
        recordCount := some 2, byteCount := some 32 }
    ]
    ((rawStructure facts closures).factsInOrder.map EvidenceOrderingFact.recordId,
      rawVerdict (rawStructure facts closures),
      acceptedVerdict (acceptedStructure [a0, a1, b0] facts closures)) =
      ([a0, a1, b0], (none, none), (none, none)) := by
  native_decide

def structuralFailureOrderCases :
    List ((Option (EvidenceStructure.OrderingFault .raw) × Option EvidenceStructure.ClosureFault) ×
      (Option (EvidenceStructure.OrderingFault .accepted) × Option EvidenceStructure.ClosureFault)) := [
  let recordId := id "test.evidence.record.duplicate"
  let fact := structuralFact recordId 1
  let closures := [{ kind := structuralKind, lastSequence := 1 }]
  (rawVerdict (rawStructure [fact, fact] closures),
    acceptedVerdict (acceptedStructure [recordId] [fact, fact] closures)),
  let firstId := id "test.evidence.record.duplicate-sequence-a"
  let secondId := id "test.evidence.record.duplicate-sequence-b"
  let facts := [structuralFact firstId 1, structuralFact secondId 1]
  let closures := [{ kind := structuralKind, lastSequence := 1 }]
  (rawVerdict (rawStructure facts closures),
    acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)),
  let firstId := id "test.evidence.record.gap-a"
  let secondId := id "test.evidence.record.gap-b"
  let missingId := id "test.evidence.record.missing"
  let facts := [structuralFact firstId 1, structuralFact secondId 3 none [missingId]]
  let closures := [{ kind := structuralKind, lastSequence := 3 }]
  (rawVerdict (rawStructure facts closures),
    acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)),
  let firstId := id "test.evidence.record.reverse-a"
  let secondId := id "test.evidence.record.reverse-b"
  let facts := [
    structuralFact firstId 1 (some { source := structuralSourceA, ordinal := 0 }) [secondId],
    structuralFact secondId 2 (some { source := structuralSourceA, ordinal := 1 })
  ]
  let closures := [{
    kind := structuralKind
    lastSequence := 2
    source := some structuralSourceA
    recordCount := some 2
    byteCount := some 32
  }]
  (rawVerdict (rawStructure facts closures),
    acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)),
  let firstId := id "test.evidence.record.cycle-1"
  let secondId := id "test.evidence.record.cycle-2"
  let facts := [structuralFact firstId 1 none [secondId], structuralFact secondId 2 none [firstId]]
  let closures := [{ kind := structuralKind, lastSequence := 2 }]
  (rawVerdict (rawStructure facts closures),
    acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)),
  let recordId := id "test.evidence.record.duplicate-closure"
  let closure : EvidenceClosureFact := { kind := structuralKind, lastSequence := 1 }
  (rawVerdict (rawStructure [structuralFact recordId 1] [closure, closure]),
    acceptedVerdict (acceptedStructure [recordId] [structuralFact recordId 1] [closure, closure])),
  let recordId := id "test.evidence.record.closure-sequence"
  let closures := [{ kind := structuralKind, lastSequence := 2 }]
  (rawVerdict (rawStructure [structuralFact recordId 1] closures),
    acceptedVerdict (acceptedStructure [recordId] [structuralFact recordId 1] closures)),
  let firstId := id "test.evidence.record.count-1"
  let secondId := id "test.evidence.record.count-2"
  let facts := [
    structuralFact firstId 1 (some { source := structuralSourceA, ordinal := 0 }),
    structuralFact secondId 2 (some { source := structuralSourceA, ordinal := 1 }) [firstId]
  ]
  let closures := [{
    kind := structuralKind
    lastSequence := 2
    source := some structuralSourceA
    recordCount := some 1
    byteCount := some 32
  }]
  (rawVerdict (rawStructure facts closures),
    acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)),
  let recordId := id "test.evidence.record.closure-byte"
  let facts := [structuralFact recordId 1 (some { source := structuralSourceA, ordinal := 0 })]
  let closures := [{
    kind := structuralKind
    lastSequence := 1
    source := some structuralSourceA
    recordCount := some 1
  }]
  (rawVerdict (rawStructure facts closures),
    acceptedVerdict (acceptedStructure [recordId] facts closures))
]

/--
Identity, order, causality, and closure faults per audience. A duplicate identical fact is a raw
identity fault but only inconsistent link order in an envelope; a duplicate sequence reports both
records raw and only the later one accepted.
-/
example : structuralFailureOrderCases = [
  ((some { kind := .duplicateIdentity, related := [id "test.evidence.record.duplicate"] }, none),
    (some { kind := .inconsistentLinkOrder, related := [structuralRuleA] }, none)),
  ((some {
      kind := .incomparableOrder
      related := [
        id "test.evidence.record.duplicate-sequence-a",
        id "test.evidence.record.duplicate-sequence-b"
      ]
    }, none),
    (some {
      kind := .incomparableOrder
      related := [id "test.evidence.record.duplicate-sequence-b"]
    }, none)),
  ((some { kind := .sequenceGap, related := [id "test.evidence.record.gap-b"] }, none),
    (some { kind := .sequenceGap, related := [id "test.evidence.record.gap-b"] }, none)),
  ((some {
      kind := .contradictoryOrder
      related := [id "test.evidence.record.reverse-a", id "test.evidence.record.reverse-b"]
    }, none),
    (some {
      kind := .contradictoryOrder
      related := [id "test.evidence.record.reverse-a", id "test.evidence.record.reverse-b"]
    }, none)),
  ((some {
      kind := .contradictoryOrder
      related := [id "test.evidence.record.cycle-1", id "test.evidence.record.cycle-2"]
    }, none),
    (some {
      kind := .contradictoryOrder
      related := [id "test.evidence.record.cycle-1", id "test.evidence.record.cycle-2"]
    }, none)),
  ((none, some { related := [structuralKind] }), (none, some { related := [structuralKind] })),
  ((none, some { related := [structuralKind] }), (none, some { related := [structuralKind] })),
  ((none, some { related := [structuralSourceA, structuralKind] }),
    (none, some { related := [structuralSourceA, structuralKind] })),
  ((none, some { related := [structuralSourceA, structuralKind] }),
    (none, some { related := [structuralSourceA, structuralKind] }))
] := by
  native_decide

/-! Origin-mode matrix. Each row is one input judged by both audiences; the rows whose two faults
differ are the entries where the raw and accepted precedence tables disagree. -/

def matrixMissingParentId : DefinitionId := id "test.evidence.record.matrix-missing"
def matrixMissingTargetId : DefinitionId := id "test.evidence.record.matrix-missing-target"

/-- Global sequence: fault receipts precede every order fault in a raw bundle. -/
example :
    let initialId := id "test.evidence.record.matrix-global-initial"
    let stepId := id "test.evidence.record.matrix-global-step"
    let facts := [structuralFact initialId 1, structuralFact stepId 3 none [initialId]]
    let closures := [{ kind := structuralKind, lastSequence := 3 }]
    let receiptBeforeGap := rawStructure facts closures [structuralKind]
      [{ recordId := stepId, target := stepId }]
    (rawVerdict receiptBeforeGap, rawVerdict (rawStructure facts closures),
      acceptedVerdict (acceptedStructure [initialId, stepId] facts closures)) = (
      (some { kind := .misdirectedFaultReceipt, related := [stepId, stepId] }, none),
      (some { kind := .sequenceGap, related := [stepId] }, none),
      (some { kind := .sequenceGap, related := [stepId] }, none)) := by
  native_decide

/-- Global sequence: a receipt naming no record precedes a missing causal parent. -/
example :
    let firstId := id "test.evidence.record.matrix-global-0"
    let secondId := id "test.evidence.record.matrix-global-1"
    let facts := [structuralFact firstId 1, structuralFact secondId 2 none [matrixMissingParentId]]
    let closures := [{ kind := structuralKind, lastSequence := 2 }]
    (rawVerdict (rawStructure facts closures [structuralKind]
        [{ recordId := secondId, target := matrixMissingTargetId }]),
      acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)) = (
      (some {
        kind := .misdirectedFaultReceipt
        related := [secondId, matrixMissingTargetId]
      }, none),
      (some { kind := .missingCausalParent, related := [secondId, matrixMissingParentId] }, none)) := by
  native_decide

/-- Source sequence: a record's causal parents precede its own receipt, and a receipt that follows
its target in the same source is not misdirected. -/
example :
    let firstId := id "test.evidence.record.matrix-source-0"
    let secondId := id "test.evidence.record.matrix-source-1"
    let facts := [
      structuralFact firstId 1 (some { source := structuralSourceA, ordinal := 0 }),
      structuralFact secondId 2 (some { source := structuralSourceA, ordinal := 1 })
        [matrixMissingParentId]
    ]
    let closures := [{
      kind := structuralKind
      lastSequence := 2
      source := some structuralSourceA
      recordCount := some 2
      byteCount := some 32
    }]
    (rawVerdict (rawStructure facts closures [structuralKind]
        [{ recordId := secondId, target := matrixMissingTargetId }]),
      rawVerdict (rawStructure (facts.map fun fact => { fact with causalParents := [] }) closures
        [structuralKind] [{ recordId := secondId, target := firstId }]),
      acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)) = (
      (some { kind := .missingCausalParent, related := [secondId, matrixMissingParentId] }, none),
      (none, none),
      (some { kind := .missingCausalParent, related := [secondId, matrixMissingParentId] }, none)) := by
  native_decide

/-- Mixed origins: both audiences report incomparable order; only the raw audience names records. -/
example :
    let globalId := id "test.evidence.record.global"
    let sourcedId := id "test.evidence.record.sourced"
    let facts := [
      structuralFact sourcedId 2 (some { source := structuralSourceA, ordinal := 0 }),
      structuralFact globalId 1
    ]
    (rawVerdict (rawStructure facts []),
      acceptedVerdict (acceptedStructure [globalId, sourcedId] facts [])) = (
      (some { kind := .incomparableOrder, related := [globalId, sourcedId] }, none),
      (some { kind := .incomparableOrder, related := [] }, some { related := [] })) := by
  native_decide

/-- An envelope whose evidence identities differ from its ordered facts is uncovered before any
order fault; a raw bundle has no envelope to cover. -/
example :
    let firstId := id "test.evidence.record.gap-a"
    let secondId := id "test.evidence.record.gap-b"
    let facts := [structuralFact firstId 1, structuralFact secondId 3 none [firstId]]
    let closures := [{ kind := structuralKind, lastSequence := 3 }]
    ((acceptedStructure [secondId, firstId] facts closures).orderingFault? .accepted,
      (rawStructure facts closures).orderingFault? .raw) = (
      some { kind := .uncoveredEvidence, related := [] },
      some { kind := .sequenceGap, related := [secondId] }) := by
  native_decide

/-- Global closures: a raw bundle judges only required kinds, while an envelope first reports a fact
of any kind that no closure covers. -/
example :
    let firstId := id "test.evidence.record.global-closure-1"
    let secondId := id "test.evidence.record.global-closure-2"
    let facts := [
      structuralFact firstId 1,
      { structuralFact secondId 2 none [firstId] with kind := structuralAuxiliaryKind }
    ]
    let closures := [{ kind := structuralKind, lastSequence := 5 }]
    (rawVerdict (rawStructure facts closures),
      acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)) = (
      (none, some { related := [structuralKind] }),
      (none, some { related := [secondId, structuralAuxiliaryKind] })) := by
  native_decide

/-- Global closures: an unrequired closure without facts passes a raw bundle, and passes an envelope
only when it closes zero records. -/
example :
    let recordId := id "test.evidence.record.zero-closure"
    let facts := [structuralFact recordId 1]
    let closures (lastSequence : Nat) : List EvidenceClosureFact := [
      { kind := structuralKind, lastSequence := 1 },
      { kind := structuralAuxiliaryKind, lastSequence }
    ]
    (rawVerdict (rawStructure facts (closures 1)),
      acceptedVerdict (acceptedStructure [recordId] facts (closures 1)),
      acceptedVerdict (acceptedStructure [recordId] facts (closures 0))) = (
      (none, none),
      (none, some { related := [structuralAuxiliaryKind] }),
      (none, none)) := by
  native_decide

/-- Source closures: a raw bundle judges supplied closures before uncovered records, while an
envelope judges uncovered facts before supplied closures. -/
example :
    let firstId := id "test.evidence.record.source-closure-a"
    let secondId := id "test.evidence.record.source-closure-b"
    let facts := [
      structuralFact firstId 1 (some { source := structuralSourceA, ordinal := 0 }),
      structuralFact secondId 1 (some { source := structuralSourceB, ordinal := 0 })
    ]
    let closures := [{
      kind := structuralKind
      lastSequence := 1
      source := some structuralSourceA
      recordCount := some 9
      byteCount := some 16
    }]
    (rawVerdict (rawStructure facts closures),
      acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)) = (
      (none, some { related := [structuralSourceA, structuralKind] }),
      (none, some { related := [secondId, structuralSourceB, structuralKind] })) := by
  native_decide

/-- Source closures: bundle order decides which raw record an uncovered closure reports. -/
example :
    let a0 := id "test.evidence.record.a-0"
    let a1 := id "test.evidence.record.a-1"
    let first := structuralFact a0 1 (some { source := structuralSourceA, ordinal := 0 })
    let second := structuralFact a1 2 (some { source := structuralSourceA, ordinal := 1 }) [a0]
    ((rawStructure [first, second] []).closureFault? .raw,
      (rawStructure [second, first] []).closureFault? .raw) = (
      some { related := [a0, structuralSourceA, structuralKind] },
      some { related := [a1, structuralSourceA, structuralKind] }) := by
  native_decide

/-- An ordering fault and a closure fault can hold together; each verdict still reports its own. -/
example :
    let firstId := id "test.evidence.record.gap-a"
    let secondId := id "test.evidence.record.gap-b"
    let facts := [structuralFact firstId 1, structuralFact secondId 3 none [firstId]]
    let closures := [{ kind := structuralKind, lastSequence := 2 }]
    (rawVerdict (rawStructure facts closures),
      acceptedVerdict (acceptedStructure [firstId, secondId] facts closures)) = (
      (some { kind := .sequenceGap, related := [secondId] }, some { related := [structuralKind] }),
      (some { kind := .sequenceGap, related := [secondId] }, some { related := [structuralKind] })) := by
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
    let evidence := rawStructure tenfoldStructuralFacts [{
      kind := structuralKind
      lastSequence := 20
      source := some structuralSourceA
      recordCount := some 20
      byteCount := some 320
    }]
    (evidence.factsInOrder.length, rawVerdict evidence) = (20, (none, none)) := by
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
    (ruleId : DefinitionId) : EvidenceStructure.LinkSupport := {
  ruleId
  evidenceIdentities := [linkedStructuralFirstId, linkedStructuralSecondId]
  orderingSupport := linkedStructuralFacts
  closureSupport := linkedStructuralClosures
}

def linkedStructure (links : List EvidenceStructure.LinkSupport) : EvidenceStructure :=
  EvidenceStructure.analyze [] [] [structuralKind, structuralAuxiliaryKind] (some {
    evidenceIdentities := [linkedStructuralFirstId, linkedStructuralSecondId]
    links
  })

/-- Per-link duplicate closures are rejected without treating copies across links as duplicates. -/
example :
    let first := linkedStructuralSupport structuralRuleA
    let duplicate := first.closureSupport.head?.get (by native_decide)
    let withinLink := linkedStructure [{
      first with closureSupport := duplicate :: first.closureSupport
    }]
    let second := linkedStructuralSupport structuralRuleB
    let laterLink := linkedStructure [
      first,
      { second with closureSupport := duplicate :: second.closureSupport }
    ]
    let acrossLinks := linkedStructure [first, second]
    (withinLink.linkSupport.map EvidenceStructure.NormalizedLinkSupport.closures,
      acceptedVerdict withinLink,
      acceptedVerdict laterLink,
      acceptedVerdict acrossLinks) = ([linkedStructuralClosures],
        (none, some { related := [structuralKind] }),
        (none, some { related := [structuralRuleB] }),
        (none, none)) := by
  native_decide

/-- Missing support on one link identifies that link without re-analyzing the shared union. -/
example :
    let second := linkedStructuralSupport structuralRuleB
    let evidence := linkedStructure [
      linkedStructuralSupport structuralRuleA,
      {
        second with
        orderingSupport := second.orderingSupport.tail
        closureSupport := second.closureSupport.tail
      }
    ]
    acceptedVerdict evidence = (
      some { kind := .inconsistentLinkOrder, related := [structuralRuleB] },
      some { related := [structuralRuleB] }) := by
  native_decide

/-- Reordered link support is normalized once and retained under the responsible rule identity. -/
example :
    let second := linkedStructuralSupport structuralRuleB
    let evidence := linkedStructure [
      linkedStructuralSupport structuralRuleA,
      {
        second with
        orderingSupport := second.orderingSupport.reverse
        closureSupport := second.closureSupport.reverse
      }
    ]
    (evidence.linkSupport, evidence.factsInOrder, acceptedVerdict evidence) = ([
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
    ], linkedStructuralFacts, (none, none)) := by
  native_decide

end Umpire.EvidenceTests
