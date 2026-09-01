import Umpire.Observation.Tests.Evaluation

/-! Stable Model Coordinate identity and exact R3 Evidence Link failures. -/

namespace Umpire.ObservationTests

open Umpire

/-- The accepted trace produced by the complete synthetic evidence fixture. -/
def completeEvidenceBackedTrace : EvidenceBackedTrace :=
  (acceptedOf completeEvaluation).get (by native_decide)

/-- The first Evidence Link in the complete accepted trace. -/
def completeFirstEvidenceLink : EvidenceLink :=
  completeEvidenceBackedTrace.evidenceLinks.head?.get (by native_decide)

private def rehashEvidenceBackedTrace (trace : EvidenceBackedTrace) : EvidenceBackedTrace := {
  trace with
  traceId := (behaviorFingerprintOf <|
    trace.mappingDigest ++ ":" ++ reprStr trace.evidenceIdentities ++ ":" ++
      reprStr trace.recordSupport ++ ":" ++ reprStr trace.trace ++ ":" ++
      reprStr trace.evidenceLinks).render
}

/-- Rehashed wrappers still fail when a rule's required disposition evidence is incomplete. -/
example :
    let evidenceLinks := completeEvidenceBackedTrace.evidenceLinks.mapIdx fun index evidenceLink =>
      if index == 0 then { evidenceLink with appliedDispositions := evidenceLink.appliedDispositions.tail }
      else evidenceLink
    let mutated := rehashEvidenceBackedTrace { completeEvidenceBackedTrace with evidenceLinks }
    diagnosticKindOf (validateEvidenceBackedTrace mutated) != none := by
  native_decide

/-- Rehashing cannot make a Model Value inconsistent with its disposition evidence valid. -/
example :
    let mutated := rehashEvidenceBackedTrace {
      completeEvidenceBackedTrace with trace := {
        completeEvidenceBackedTrace.trace with initialState := {
          completeEvidenceBackedTrace.trace.initialState with value := "tampered"
        }
      }
    }
    diagnosticKindOf (validateEvidenceBackedTrace mutated) != none := by
  native_decide

/-- Wrapper vocabulary remains exactly the canonical checked-plan vocabulary. -/
example :
    let original := completeEvidenceBackedTrace.vocabulary.head?.get (by native_decide)
    let forged := { original with canonicalBehavior := original.canonicalBehavior ++ "/forged" }
    diagnosticKindOf (validateEvidenceBackedTrace {
      completeEvidenceBackedTrace with vocabulary := completeEvidenceBackedTrace.vocabulary ++ [forged]
    }) != none := by
  native_decide

def transitiveName : ObservationBinding := {
  id := id "test.binding.transitive-name"
  valueType := .text
  expression := .portable (.binding normalizedName.id)
}

def transitiveDeclaration : ObservationMappingDeclaration := {
  evaluationDeclaration with
  bindings := evaluationDeclaration.bindings ++ [transitiveName]
  rules := evaluationDeclaration.rules.map fun rule =>
    if rule.id == initialRule.id then
      { rule with value := .portable (.binding transitiveName.id) }
    else rule
}

/-- Evidence Links name both direct and transitive checked-binding dependencies. -/
example :
    let result := match checkObservation evaluationContext transitiveDeclaration with
      | .ok plan => evaluateEvidence plan completeEvidence
      | .error _ => .unknown {
          kind := .zeroUsableInterpretations
          planId := transitiveDeclaration.id
        }
    (acceptedOf result).map (fun trace => trace.evidenceLinks.head?.map EvidenceLink.bindingIds) =
      some (some [normalizedName.id, transitiveName.id]) := by
  native_decide

/-- Exact statuses and diagnostics for invalid Evidence Link fixtures. -/
def evidenceLinkFailureKinds : List (ObservationStatus × Option ObservationFailureKind) := [
  let result := validateEvidenceBackedTrace {
    completeEvidenceBackedTrace with evidenceLinks := completeEvidenceBackedTrace.evidenceLinks.tail
  }
  (.unknown, diagnosticKindOf result),
  let result := validateEvidenceBackedTrace {
    completeEvidenceBackedTrace with
    evidenceLinks := completeFirstEvidenceLink :: completeEvidenceBackedTrace.evidenceLinks
  }
  (.conflict, diagnosticKindOf result),
  let result := validateEvidenceBackedTrace {
    completeEvidenceBackedTrace with evidenceLinks := completeEvidenceBackedTrace.evidenceLinks ++ [{
      completeFirstEvidenceLink with coordinate := .observation 1 99
    }]
  }
  (.conflict, diagnosticKindOf result),
  let result := validateEvidenceBackedTrace {
    completeEvidenceBackedTrace with trace := {
      completeEvidenceBackedTrace.trace with initialState := {
        completeEvidenceBackedTrace.trace.initialState with value := "tampered"
      }
    }
  }
  (.conflict, diagnosticKindOf result),
  let result := validateEvidenceBackedTrace {
    completeEvidenceBackedTrace with evidenceIdentities :=
      completeEvidenceBackedTrace.evidenceIdentities ++ [id "test.evidence.record.unconsumed"]
  }
  (.unknown, diagnosticKindOf result),
  let evidenceLinks := completeEvidenceBackedTrace.evidenceLinks.map fun evidenceLink => {
    evidenceLink with closureSupport := [{
        kind := eventKind
        lastSequence := 99
      }]
  }
  let result := validateEvidenceBackedTrace {
    completeEvidenceBackedTrace with evidenceLinks
  }
  (.unknown, diagnosticKindOf result),
  let evidenceLinks := completeEvidenceBackedTrace.evidenceLinks.map fun evidenceLink =>
    let recordId := evidenceLink.evidenceIdentities.head?.getD (id "test.evidence.record.missing")
    { evidenceLink with orderingSupport := [{
        recordId
        kind := eventKind
        sequence := 1
        causalParents := [recordId]
      }]
    }
  let result := validateEvidenceBackedTrace {
    completeEvidenceBackedTrace with evidenceLinks
  }
  (.unknown, diagnosticKindOf result)
]

/-- Missing, duplicate, extra, inconsistent, and unsupported Evidence Links fail exactly. -/
example : evidenceLinkFailureKinds = [
  (.unknown, some .absentModelCoordinate),
  (.conflict, some .duplicateModelCoordinate),
  (.conflict, some .extraModelCoordinate),
  (.conflict, some .inconsistentEvidenceLink),
  (.unknown, some .unconsumedReference),
  (.unknown, some .missingClosureSupport),
  (.unknown, some .missingOrderSupport)
] := by
  native_decide

/-- A zero step cannot alias the first selected-action coordinate during admission. -/
example :
    let evidenceLinks := completeEvidenceBackedTrace.evidenceLinks.map fun evidenceLink =>
      if evidenceLink.coordinate == .selectedAction 1 then
        { evidenceLink with coordinate := .selectedAction 0 }
      else
        evidenceLink
    diagnosticKindOf (validateEvidenceBackedTrace {
      completeEvidenceBackedTrace with evidenceLinks
    }) = some .absentModelCoordinate := by
  native_decide

/-- Closed evidence with a second step that repeats the first step's values. -/
def repeatedValueEvidence : EvidenceBundle := {
  completeEvidence with
  records := completeEvidence.records ++ [{
    stepEvidence with
    id := secondStepEvidenceId
    sequence := 3
    causalParents := [stepEvidenceId]
  }]
  closures := [{ kind := eventKind, lastSequence := 3 }]
}

/-- Equal Model Values at different slots retain distinct one-based coordinates. -/
example :
    let accepted := acceptedOf (evaluateFixture repeatedValueEvidence)
    accepted.map (fun trace => trace.evidenceLinks.map EvidenceLink.coordinate) = some [
      .initialState,
      .selectedAction 1,
      .modelOutcome 1,
      .resultingState 1,
      .observation 1 1,
      .observation 1 2,
      .selectedAction 2,
      .modelOutcome 2,
      .resultingState 2,
      .observation 2 1,
      .observation 2 2
    ] := by
  native_decide

def primaryEvidenceSource : DefinitionId := id "test.evidence.source.primary"
def auxiliaryEvidenceSource : DefinitionId := id "test.evidence.source.auxiliary"
def auxiliaryEvidenceId : DefinitionId := id "test.evidence.record.auxiliary"

def multiSourceEvidence : EvidenceBundle := {
  completeEvidence with
  records := [
    { stepEvidence with origin := some { source := primaryEvidenceSource, ordinal := 1 } },
    {
      id := auxiliaryEvidenceId
      profile := profileId
      profileVersion := 1
      kind := eventKind
      sequence := 1
      origin := some { source := auxiliaryEvidenceSource, ordinal := 0 }
      fields := [textField roleField "support"]
    },
    { initialEvidence with origin := some { source := primaryEvidenceSource, ordinal := 0 } }
  ]
  closures := [
    { kind := eventKind, lastSequence := 2, source := some primaryEvidenceSource,
      recordCount := some 2, byteCount := some 64 },
    { kind := eventKind, lastSequence := 1, source := some auxiliaryEvidenceSource,
      recordCount := some 1, byteCount := some 16 }
  ]
}

def multiSourceTrace : EvidenceBackedTrace :=
  (acceptedOf (evaluateFixture multiSourceEvidence)).get (by native_decide)

/-! Independent source-local order and causal order remain complete provenance without inventing a
cross-source step. -/
example :
    (multiSourceTrace.trace,
      multiSourceTrace.recordSupport.map fun support =>
          (support.recordId, support.origin, support.fields.map fun field =>
            (field.field, field.valueType)),
      multiSourceTrace.evidenceLinks.all fun link =>
        link.orderingSupport.map (fun fact => (fact.recordId, fact.origin)) == [
          (auxiliaryEvidenceId,
            some { source := auxiliaryEvidenceSource, ordinal := 0 }),
          (initialEvidenceId, some { source := primaryEvidenceSource, ordinal := 0 }),
          (stepEvidenceId, some { source := primaryEvidenceSource, ordinal := 1 })
        ] && link.closureSupport.length == 2) == (
      expectedTrace,
      [
        (auxiliaryEvidenceId, some { source := auxiliaryEvidenceSource, ordinal := 0 },
          [(roleField, .text)]),
        (initialEvidenceId, some { source := primaryEvidenceSource, ordinal := 0 },
          [(nameField, .text), (roleField, .text)]),
        (stepEvidenceId, some { source := primaryEvidenceSource, ordinal := 1 },
          [(hashedField, .text), (roleField, .text), (secretField, .text)])
      ],
      true) := by
  native_decide

/-! Every multi-source coordinate retains the complete source-local ordering and closure proof. -/
example :
    let first := multiSourceTrace.evidenceLinks.head?.get (by native_decide)
    let links := { first with orderingSupport := first.orderingSupport.tail } ::
      multiSourceTrace.evidenceLinks.tail
    diagnosticKindOf (validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      multiSourceTrace with evidenceLinks := links
    }) = some .missingOrderSupport := by
  native_decide

example :
    let first := multiSourceTrace.evidenceLinks.head?.get (by native_decide)
    let links := { first with closureSupport := first.closureSupport.tail } ::
      multiSourceTrace.evidenceLinks.tail
    diagnosticKindOf (validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      multiSourceTrace with evidenceLinks := links
    }) = some .missingClosureSupport := by
  native_decide

/-! Source-local causal orphans and cycles retain their exact fn-4 diagnostic classes. -/
example :
    let orphan := { multiSourceEvidence with records := multiSourceEvidence.records.map fun record =>
      if record.id == stepEvidenceId then
        { record with causalParents := [id "test.evidence.record.missing"] }
      else record }
    let cyclic := { multiSourceEvidence with records := multiSourceEvidence.records.map fun record =>
      if record.id == initialEvidenceId then { record with causalParents := [stepEvidenceId] }
      else record }
    (evaluateFixture orphan).diagnostic?.map ObservationDiagnostic.kind == some .missingCausalParent &&
      (evaluateFixture cyclic).diagnostic?.map ObservationDiagnostic.kind == some .contradictoryOrder := by
  native_decide

end Umpire.ObservationTests
