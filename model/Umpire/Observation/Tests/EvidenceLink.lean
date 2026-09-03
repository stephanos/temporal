import Umpire.Observation.Tests.Evaluation

/-! Stable Model Coordinate identity and exact R3 Evidence Link failures. -/

namespace Umpire.ObservationTests

open Umpire

/-- The accepted trace produced by the complete synthetic evidence fixture. -/
def completeEvidenceBackedTrace : EvidenceBackedTrace :=
  (acceptedOf completeEvaluation).get (by native_decide)

/-- The unchecked form used only by negative admission fixtures. -/
def completeUncheckedEvidenceBackedTrace : UncheckedEvidenceBackedTrace :=
  uncheckedTraceOf completeEvidenceBackedTrace

/-- The first Evidence Link in the complete accepted trace. -/
def completeFirstEvidenceLink : EvidenceLink :=
  completeEvidenceBackedTrace.evidenceLinks.head?.get (by native_decide)

private def rehashEvidenceBackedTrace
    (trace : UncheckedEvidenceBackedTrace) : UncheckedEvidenceBackedTrace := {
  trace with
  traceId := (behaviorFingerprintOf <|
    trace.mappingDigest ++ ":" ++ reprStr trace.evidenceIdentities ++ ":" ++
      reprStr trace.recordSupport ++ ":" ++ reprStr trace.trace ++ ":" ++
      reprStr trace.evidenceLinks).render
}

private def zeroRecordUncheckedTrace : UncheckedEvidenceBackedTrace :=
  let mappingDigest := zeroRecordEvaluationPlan.behaviorFingerprint.render
  let closures := completeEvidence.closures ++ [zeroRecordClosure]
  let evidenceLinks := completeUncheckedEvidenceBackedTrace.evidenceLinks.map fun evidenceLink => {
    evidenceLink with mappingDigest, closureSupport := closures
  }
  rehashEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with
    checkedPlan := zeroRecordEvaluationPlan
    mappingDigest
    evidenceLinks
  }

/-- Accepted admission retains an explicit zero-record global closure. -/
example : (validateEvidenceBackedTrace zeroRecordUncheckedTrace).toOption.isSome = true := by
  native_decide

/-- Missing or inconsistent zero-record closure support still fails closed. -/
example :
    let missing := zeroRecordUncheckedTrace.evidenceLinks.map fun evidenceLink => {
      evidenceLink with closureSupport := completeEvidence.closures
    }
    let inconsistent := zeroRecordUncheckedTrace.evidenceLinks.map fun evidenceLink => {
      evidenceLink with closureSupport := completeEvidence.closures ++ [
        { zeroRecordClosure with lastSequence := 1 }
      ]
    }
    [missing, inconsistent].map (fun evidenceLinks =>
      match validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
        zeroRecordUncheckedTrace with evidenceLinks
      } with
      | .ok _ => none
      | .error failure => some (failure.kind, failure.relatedDefinitionIds)) =
      [some (.missingClosureSupport, [zeroRecordKind]),
        some (.missingClosureSupport, [zeroRecordKind])] := by
  native_decide

/-- A canonical plan's evidence bound is enforced again at unchecked trace admission. -/
example :
    let declaration := {
      evaluationDeclaration with
      evidenceBound := { value := 1, unit := .evidenceRecords }
    }
    let plan := (checkObservation evaluationContext declaration).toOption.get (by native_decide)
    let evidenceLinks := completeUncheckedEvidenceBackedTrace.evidenceLinks.map fun evidenceLink => {
      evidenceLink with
      mappingDigest := plan.behaviorFingerprint.render
      appliedBound := plan.evidenceBound
    }
    let unchecked := rehashEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with
      checkedPlan := plan
      mappingDigest := plan.behaviorFingerprint.render
      appliedBound := plan.evidenceBound
      evidenceLinks
    }
    (match validateEvidenceBackedTrace unchecked with
      | .ok _ => none
      | .error diagnostic => some (diagnostic.kind, diagnostic.limit, diagnostic.observedCount)) =
      some (.evidenceBoundExhausted, some plan.evidenceBound,
        some unchecked.evidenceIdentities.length) := by
  native_decide

/-- Rehashed wrappers still fail when a rule's required disposition evidence is incomplete. -/
example :
    let evidenceLinks := completeUncheckedEvidenceBackedTrace.evidenceLinks.mapIdx fun index evidenceLink =>
      if index == 0 then { evidenceLink with appliedDispositions := evidenceLink.appliedDispositions.tail }
      else evidenceLink
    let mutated := rehashEvidenceBackedTrace { completeUncheckedEvidenceBackedTrace with evidenceLinks }
    diagnosticKindOf (validateEvidenceBackedTrace mutated) != none := by
  native_decide

/-- Rehashing cannot make a Model Value inconsistent with its disposition evidence valid. -/
example :
    let mutated := rehashEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with trace := {
        completeUncheckedEvidenceBackedTrace.trace with initialState := {
          completeUncheckedEvidenceBackedTrace.trace.initialState with value := "tampered"
        }
      }
    }
    diagnosticKindOf (validateEvidenceBackedTrace mutated) != none := by
  native_decide

/-- Wrapper vocabulary remains exactly the canonical checked-plan vocabulary. -/
example :
    let original := completeUncheckedEvidenceBackedTrace.vocabulary.head?.get (by native_decide)
    let forged := { original with canonicalBehavior := original.canonicalBehavior ++ "/forged" }
    diagnosticKindOf (validateEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with
      vocabulary := completeUncheckedEvidenceBackedTrace.vocabulary ++ [forged]
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
    completeUncheckedEvidenceBackedTrace with
    evidenceLinks := completeUncheckedEvidenceBackedTrace.evidenceLinks.tail
  }
  admissionStatusAndKind result,
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with
    evidenceLinks := completeFirstEvidenceLink :: completeUncheckedEvidenceBackedTrace.evidenceLinks
  }
  admissionStatusAndKind result,
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with
    evidenceLinks := completeUncheckedEvidenceBackedTrace.evidenceLinks ++ [{
      completeFirstEvidenceLink with coordinate := .observation 1 99
    }]
  }
  admissionStatusAndKind result,
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with trace := {
      completeUncheckedEvidenceBackedTrace.trace with initialState := {
        completeUncheckedEvidenceBackedTrace.trace.initialState with value := "tampered"
      }
    }
  }
  admissionStatusAndKind result,
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with evidenceIdentities :=
      completeUncheckedEvidenceBackedTrace.evidenceIdentities ++ [id "test.evidence.record.unconsumed"]
  }
  admissionStatusAndKind result,
  let evidenceLinks := completeUncheckedEvidenceBackedTrace.evidenceLinks.map fun evidenceLink => {
    evidenceLink with closureSupport := [{
        kind := eventKind
        lastSequence := 99
      }]
  }
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with evidenceLinks
  }
  admissionStatusAndKind result,
  let evidenceLinks := completeUncheckedEvidenceBackedTrace.evidenceLinks.map fun evidenceLink =>
    let recordId := evidenceLink.evidenceIdentities.head?.getD (id "test.evidence.record.missing")
    { evidenceLink with orderingSupport := [{
        recordId
        kind := eventKind
        sequence := 1
        causalParents := [recordId]
      }]
    }
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with evidenceLinks
  }
  admissionStatusAndKind result
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
    let evidenceLinks := completeUncheckedEvidenceBackedTrace.evidenceLinks.map fun evidenceLink =>
      if evidenceLink.coordinate == .selectedAction 1 then
        { evidenceLink with coordinate := .selectedAction 0 }
      else
        evidenceLink
    diagnosticKindOf (validateEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with evidenceLinks
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
    let unchecked := uncheckedTraceOf multiSourceTrace
    (match validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      unchecked with evidenceLinks := links
    } with
      | .ok _ => none
      | .error failure => some (failure.kind, failure.relatedDefinitionIds)) =
      some (.missingOrderSupport, [first.ruleId]) := by
  native_decide

/-- Duplicate per-link ordering support fails at the responsible accepted-boundary rule. -/
example :
    let first := completeFirstEvidenceLink
    let duplicate := first.orderingSupport.head?.get (by native_decide)
    let links := {
      first with orderingSupport := duplicate :: first.orderingSupport
    } :: completeUncheckedEvidenceBackedTrace.evidenceLinks.tail
    let result := validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with evidenceLinks := links
    }
    (match result with
      | .ok _ => none
      | .error failure => some (failure.kind, failure.relatedDefinitionIds)) =
      some (.missingOrderSupport, [first.ruleId]) := by
  native_decide

/-- Per-link duplicate closure support names its rule while cross-link copies remain valid. -/
example :
    let first := completeFirstEvidenceLink
    let duplicate := first.closureSupport.head?.get (by native_decide)
    let links := {
      first with closureSupport := duplicate :: first.closureSupport
    } :: completeUncheckedEvidenceBackedTrace.evidenceLinks.tail
    let withinLink := validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with evidenceLinks := links
    }
    (match withinLink with
      | .ok _ => none
      | .error failure => some (failure.kind, failure.relatedDefinitionIds),
      (validateEvidenceBackedTrace completeUncheckedEvidenceBackedTrace).toOption.isSome) =
      (some (.missingClosureSupport, [first.ruleId]), true) := by
  native_decide

example :
    let first := multiSourceTrace.evidenceLinks.head?.get (by native_decide)
    let links := { first with closureSupport := first.closureSupport.tail } ::
      multiSourceTrace.evidenceLinks.tail
    let unchecked := uncheckedTraceOf multiSourceTrace
    (match validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      unchecked with evidenceLinks := links
    } with
      | .ok _ => none
      | .error failure => some (failure.kind, failure.relatedDefinitionIds)) =
      some (.missingClosureSupport, [first.ruleId]) := by
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
