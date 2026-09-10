import Umpire.Evidence.Tests.Evaluation

/-! Stable Model Coordinate identity and exact R3 Evidence Link failures. -/

namespace Umpire.EvidenceTests

open Umpire

/-- The accepted trace produced by the complete synthetic evidence fixture. -/
def completeEvidenceBackedTrace : EvidenceBackedTrace :=
  (acceptedOf completeEvaluation).get (by native_decide)

/-- The unchecked form used only by negative admission fixtures. -/
def completeUncheckedEvidenceBackedTrace : UncheckedEvidenceBackedTrace :=
  uncheckedTraceOf completeEvidenceBackedTrace

/-- The first Evidence Link in the complete accepted trace. -/
def completeFirstEvidenceSupport : EvidenceSupport :=
  completeEvidenceBackedTrace.evidenceSupports.head?.get (by native_decide)

private def rehashEvidenceBackedTrace
    (trace : UncheckedEvidenceBackedTrace) : UncheckedEvidenceBackedTrace := {
  trace with
  traceId := (behaviorFingerprintOf <|
    trace.mappingDigest ++ ":" ++ reprStr trace.evidenceIdentities ++ ":" ++
      reprStr trace.recordSupport ++ ":" ++ reprStr trace.trace ++ ":" ++
      reprStr trace.evidenceSupports).render
}

private def zeroRecordUncheckedTrace : UncheckedEvidenceBackedTrace :=
  let mappingDigest := zeroRecordEvaluationPlan.behaviorFingerprint.render
  let closures := completeEvidence.closures ++ [zeroRecordClosure]
  let evidenceSupports := completeUncheckedEvidenceBackedTrace.evidenceSupports.map fun evidenceSupport => {
    evidenceSupport with mappingDigest, closureSupport := closures
  }
  rehashEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with
    checkedPlan := zeroRecordEvaluationPlan
    mappingDigest
    evidenceSupports
  }

/-- Accepted admission retains an explicit zero-record global closure. -/
example : (validateEvidenceBackedTrace zeroRecordUncheckedTrace).toOption.isSome = true := by
  native_decide

/-- Missing or inconsistent zero-record closure support still fails closed. -/
example :
    let missing := zeroRecordUncheckedTrace.evidenceSupports.map fun evidenceSupport => {
      evidenceSupport with closureSupport := completeEvidence.closures
    }
    let inconsistent := zeroRecordUncheckedTrace.evidenceSupports.map fun evidenceSupport => {
      evidenceSupport with closureSupport := completeEvidence.closures ++ [
        { zeroRecordClosure with lastSequence := 1 }
      ]
    }
    [missing, inconsistent].map (fun evidenceSupports =>
      match validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
        zeroRecordUncheckedTrace with evidenceSupports
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
    let plan := (Evidence.checkReading evaluationContext declaration).toOption.get (by native_decide)
    let evidenceSupports := completeUncheckedEvidenceBackedTrace.evidenceSupports.map fun evidenceSupport => {
      evidenceSupport with
      mappingDigest := plan.behaviorFingerprint.render
      appliedBound := plan.evidenceBound
    }
    let unchecked := rehashEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with
      checkedPlan := plan
      mappingDigest := plan.behaviorFingerprint.render
      appliedBound := plan.evidenceBound
      evidenceSupports
    }
    (match validateEvidenceBackedTrace unchecked with
      | .ok _ => none
      | .error diagnostic => some (diagnostic.kind, diagnostic.limit, diagnostic.observedCount)) =
      some (.evidenceBoundExhausted, some plan.evidenceBound,
        some unchecked.evidenceIdentities.length) := by
  native_decide

/-- Rehashed wrappers still fail when a rule's required disposition evidence is incomplete. -/
example :
    let evidenceSupports := completeUncheckedEvidenceBackedTrace.evidenceSupports.mapIdx fun index evidenceSupport =>
      if index == 0 then { evidenceSupport with appliedDispositions := evidenceSupport.appliedDispositions.tail }
      else evidenceSupport
    let mutated := rehashEvidenceBackedTrace { completeUncheckedEvidenceBackedTrace with evidenceSupports }
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
    let forged := { original with behaviorVersion := original.behaviorVersion ++ "/forged" }
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

def transitiveDeclaration : Evidence.Reading := {
  evaluationDeclaration with
  bindings := evaluationDeclaration.bindings ++ [transitiveName]
  rules := evaluationDeclaration.rules.map fun rule =>
    if rule.id == initialRule.id then
      { rule with value := .portable (.binding transitiveName.id) }
    else rule
}

/-- Evidence Links name both direct and transitive checked-binding dependencies. -/
example :
    let result := match Evidence.checkReading evaluationContext transitiveDeclaration with
      | .ok plan => evaluateEvidence plan completeEvidence
      | .error _ => .unknown {
          kind := .zeroUsableInterpretations
          planId := transitiveDeclaration.id
        }
    (acceptedOf result).map (fun trace => trace.evidenceSupports.head?.map EvidenceSupport.bindingIds) =
      some (some [normalizedName.id, transitiveName.id]) := by
  native_decide

/-- Exact statuses and diagnostics for invalid Evidence Link fixtures. -/
def evidenceSupportFailureKinds : List (ObservationStatus × Option ObservationFailureKind) := [
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with
    evidenceSupports := completeUncheckedEvidenceBackedTrace.evidenceSupports.tail
  }
  admissionStatusAndKind result,
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with
    evidenceSupports := completeFirstEvidenceSupport :: completeUncheckedEvidenceBackedTrace.evidenceSupports
  }
  admissionStatusAndKind result,
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with
    evidenceSupports := completeUncheckedEvidenceBackedTrace.evidenceSupports ++ [{
      completeFirstEvidenceSupport with coordinate := .fact 1 99
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
  let evidenceSupports := completeUncheckedEvidenceBackedTrace.evidenceSupports.map fun evidenceSupport => {
    evidenceSupport with closureSupport := [{
        kind := eventKind
        lastSequence := 99
      }]
  }
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with evidenceSupports
  }
  admissionStatusAndKind result,
  let evidenceSupports := completeUncheckedEvidenceBackedTrace.evidenceSupports.map fun evidenceSupport =>
    let recordId := evidenceSupport.evidenceIdentities.head?.getD (id "test.evidence.record.missing")
    { evidenceSupport with orderingSupport := [{
        recordId
        kind := eventKind
        sequence := 1
        causalParents := [recordId]
      }]
    }
  let result := validateEvidenceBackedTrace {
    completeUncheckedEvidenceBackedTrace with evidenceSupports
  }
  admissionStatusAndKind result
]

/-- Missing, duplicate, extra, inconsistent, and unsupported Evidence Links fail exactly. -/
example : evidenceSupportFailureKinds = [
  (.unknown, some .absentModelCoordinate),
  (.conflict, some .duplicateModelCoordinate),
  (.conflict, some .extraModelCoordinate),
  (.conflict, some .inconsistentEvidenceSupport),
  (.unknown, some .unconsumedReference),
  (.unknown, some .missingClosureSupport),
  (.unknown, some .missingOrderSupport)
] := by
  native_decide

/-- A zero step cannot alias the first selected-action coordinate during admission. -/
example :
    let evidenceSupports := completeUncheckedEvidenceBackedTrace.evidenceSupports.map fun evidenceSupport =>
      if evidenceSupport.coordinate == .selectedAction 1 then
        { evidenceSupport with coordinate := .selectedAction 0 }
      else
        evidenceSupport
    diagnosticKindOf (validateEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with evidenceSupports
    }) = some .absentModelCoordinate := by
  native_decide

/-- Closed evidence with a second step that repeats the first step's values. -/
def repeatedValueEvidence : SyntheticEvidence := {
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
    accepted.map (fun trace => trace.evidenceSupports.map EvidenceSupport.coordinate) = some [
      .initialState,
      .selectedAction 1,
      .outcome 1,
      .state 1,
      .fact 1 1,
      .fact 1 2,
      .selectedAction 2,
      .outcome 2,
      .state 2,
      .fact 2 1,
      .fact 2 2
    ] := by
  native_decide

def primaryEvidenceSource : DefinitionId := id "test.evidence.source.primary"
def auxiliaryEvidenceSource : DefinitionId := id "test.evidence.source.auxiliary"
def auxiliaryEvidenceId : DefinitionId := id "test.evidence.record.auxiliary"

def multiSourceEvidence : SyntheticEvidence := {
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
      multiSourceTrace.evidenceSupports.all fun link =>
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
    let first := multiSourceTrace.evidenceSupports.head?.get (by native_decide)
    let links := { first with orderingSupport := first.orderingSupport.tail } ::
      multiSourceTrace.evidenceSupports.tail
    let unchecked := uncheckedTraceOf multiSourceTrace
    (match validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      unchecked with evidenceSupports := links
    } with
      | .ok _ => none
      | .error failure => some (failure.kind, failure.relatedDefinitionIds)) =
      some (.missingOrderSupport, [first.ruleId]) := by
  native_decide

/-- Duplicate per-link ordering support fails at the responsible accepted-boundary rule. -/
example :
    let first := completeFirstEvidenceSupport
    let duplicate := first.orderingSupport.head?.get (by native_decide)
    let links := {
      first with orderingSupport := duplicate :: first.orderingSupport
    } :: completeUncheckedEvidenceBackedTrace.evidenceSupports.tail
    let result := validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with evidenceSupports := links
    }
    (match result with
      | .ok _ => none
      | .error failure => some (failure.kind, failure.relatedDefinitionIds)) =
      some (.missingOrderSupport, [first.ruleId]) := by
  native_decide

/-- Duplicate closure support preserves baseline kind and later-link rule identities. -/
example :
    let first := completeFirstEvidenceSupport
    let second := completeUncheckedEvidenceBackedTrace.evidenceSupports.tail.head?.get
      (by native_decide)
    let duplicate := first.closureSupport.head?.get (by native_decide)
    let firstLinks := {
      first with closureSupport := duplicate :: first.closureSupport
    } :: completeUncheckedEvidenceBackedTrace.evidenceSupports.tail
    let laterLinks := first :: {
      second with closureSupport := duplicate :: second.closureSupport
    } :: completeUncheckedEvidenceBackedTrace.evidenceSupports.tail.tail
    let firstLink := validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with evidenceSupports := firstLinks
    }
    let laterLink := validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      completeUncheckedEvidenceBackedTrace with evidenceSupports := laterLinks
    }
    (match firstLink with
      | .ok _ => none
      | .error failure => some (failure.kind, failure.relatedDefinitionIds),
      match laterLink with
      | .ok _ => none
      | .error failure => some (failure.kind, failure.relatedDefinitionIds),
      (validateEvidenceBackedTrace completeUncheckedEvidenceBackedTrace).toOption.isSome) =
      (some (.missingClosureSupport, [eventKind]),
        some (.missingClosureSupport, [second.ruleId]), true) := by
  native_decide

example :
    let first := multiSourceTrace.evidenceSupports.head?.get (by native_decide)
    let links := { first with closureSupport := first.closureSupport.tail } ::
      multiSourceTrace.evidenceSupports.tail
    let unchecked := uncheckedTraceOf multiSourceTrace
    (match validateEvidenceBackedTrace <| rehashEvidenceBackedTrace {
      unchecked with evidenceSupports := links
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

end Umpire.EvidenceTests
