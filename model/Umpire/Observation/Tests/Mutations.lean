import Umpire.Observation.Tests.Aggregation

/-!
Independent cross-layer mutations. Expected traces, diagnostics, and verdicts are literal test data;
none is projected from the implementation result being checked.
-/

namespace Umpire.ObservationTests

open Umpire

/-! Model mutations are rejected by the independent accepted-trace comparison, not another layer. -/

def mutatedExpectedTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  expectedTrace with
  initialState := { expectedTrace.initialState with value := "unexpected" }
}

/-- A model-only mutation leaves evaluation valid while failing the independently authored oracle. -/
example :
    let actual := (acceptedOf completeEvaluation).map EvidenceBackedTrace.trace
    (completeEvaluation.status, actual, actual == some mutatedExpectedTrace) =
      (.accepted, some expectedTrace, false) := by
  native_decide

/-! Mapping mutations fail at compilation before any evidence can be interpreted. -/

def unknownOperatorMutation : ObservationMappingDeclaration := {
  baseDeclaration with
  bindings := [{ normalizedName with
    expression := .portable
      (.normalize { name := "text.unknown", version := 1 } (field nameFieldSpec)) }]
}

def wrongBindingTypeMutation : ObservationMappingDeclaration := {
  baseDeclaration with
  bindings := [{ normalizedName with valueType := .natural }]
}

def clearValueTaintMutation : ObservationMappingDeclaration := {
  baseDeclaration with
  rules := baseDeclaration.rules.map fun rule =>
    if rule.id == contributionRule.id then
      { rule with value := .portable (field secretFieldSpec) }
    else
      rule
}

/-- Operator, type, and information-flow mutations have exact compile-time owners. -/
example : [
    errorKindOf (checkObservation context unknownOperatorMutation),
    errorKindOf (checkObservation context wrongBindingTypeMutation),
    errorKindOf (checkObservation context clearValueTaintMutation)
  ] = [
    some .unknownOperator,
    some .incompatibleBinding,
    some .unauthorizedClearValueFlow
  ] := by
  native_decide

/-! Evidence-volume mutations fail at evaluation, after the mapping has compiled. -/

def boundedDeclaration : ObservationMappingDeclaration := {
  evaluationDeclaration with
  evidenceBound := { value := 2, unit := .evidenceRecords }
}

def boundedPlan : CheckedObservationPlan :=
  (checkObservation evaluationContext boundedDeclaration).toOption.get (by native_decide)

def limitPlusOneEvidence : EvidenceBundle := {
  completeEvidence with
  records := completeEvidence.records ++ [{
    stepEvidence with
    id := secondStepEvidenceId
    sequence := 3
    causalParents := [stepEvidenceId]
  }]
  closures := [{ kind := eventKind, lastSequence := 3 }]
}

/-- N records are accepted, while N+1 is unknown with the literal bound diagnostic and no trace. -/
example :
    let atLimit := evaluateEvidence boundedPlan completeEvidence
    let overLimit := evaluateEvidence boundedPlan limitPlusOneEvidence
    (atLimit.status, overLimit, acceptedOf overLimit) = (
      .accepted,
      .unknown {
        kind := .evidenceBoundExhausted
        planId := boundedDeclaration.id
        limit := some { value := 2, unit := .evidenceRecords }
        observedCount := some 3
      },
      none) := by
  native_decide

/-! Wrapper mutations fail at coordinate, ordering, and disposition validation. -/

def literalInitialOrdering : List EvidenceOrderingFact := [{
  recordId := id "test.evidence.record.initial"
  kind := id "test.evidence.kind.event"
  sequence := 1
  causalParents := []
}]

def literalStepOrdering : List EvidenceOrderingFact := [{
  recordId := id "test.evidence.record.step-1"
  kind := id "test.evidence.kind.event"
  sequence := 2
  causalParents := [id "test.evidence.record.initial"]
}]

def literalClosure : List EvidenceClosureFact := [{
  kind := id "test.evidence.kind.event"
  lastSequence := 2
}]

/-- Literal canonical mapping identity; no expected Evidence Link field is implementation-derived. -/
def literalMappingDigest : String :=
  "sha256:9b8e76bdd7b9490b3bd28c70820bf78e4648a65378eba1dbcff74bbe5306d40a"

def literalEvidenceLink
    (mappingDigest : String)
    (coordinate : ModelCoordinate)
    (evidenceIdentity ruleId : DefinitionId)
    (bindingIds : List DefinitionId)
    (orderingSupport : List EvidenceOrderingFact)
    (appliedDispositions : List AppliedFieldDisposition)
    (meaningDigest : String) : EvidenceLink := {
  coordinate
  mappingId := id "test.mapping.observation-evaluation"
  mappingVersion := 1
  mappingDigest
  profileId := id "test.evidence.profile"
  profileVersion := 1
  evidenceIdentities := [evidenceIdentity]
  ruleId
  bindingIds
  orderingSupport
  closureSupport := literalClosure
  appliedDispositions
  appliedBound := { value := 3, unit := .evidenceRecords }
  meaningDigest
}

/-- Independently authored Evidence Links for every Model Trace slot in `expectedTrace`. -/
def literalEvidenceLinks (mappingDigest : String) : List EvidenceLink := [
  literalEvidenceLink mappingDigest .initialState
    (id "test.evidence.record.initial") (id "test.rule.initial-state")
    [id "test.binding.normalized-name"] literalInitialOrdering [
      {
        field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.name" }
        evidence := .retained "ready"
      },
      {
        field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.role" }
        evidence := .retained "initial"
      }
    ] "test.state.operation/meaning-v1",
  literalEvidenceLink mappingDigest (.selectedAction 1)
    (id "test.evidence.record.step-1") (id "test.rule.step-action") [] literalStepOrdering [{
      field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.role" }
      evidence := .retained "step"
    }] "test.action.start/meaning-v1",
  literalEvidenceLink mappingDigest (.modelOutcome 1)
    (id "test.evidence.record.step-1") (id "test.rule.step-outcome") [] literalStepOrdering [{
      field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.role" }
      evidence := .retained "step"
    }] "test.outcome.success/meaning-v1",
  literalEvidenceLink mappingDigest (.resultingState 1)
    (id "test.evidence.record.step-1") (id "test.rule.step-state") [] literalStepOrdering [{
      field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.role" }
      evidence := .retained "step"
    }] "test.state.completed/meaning-v1",
  literalEvidenceLink mappingDigest (.observation 1 1)
    (id "test.evidence.record.step-1") (id "test.rule.contribution") [] literalStepOrdering [
      {
        field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.role" }
        evidence := .retained "step"
      },
      {
        field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.secret" }
        evidence := .redactedContribution
      }
    ] "test.observation.contribution/meaning-v1",
  literalEvidenceLink mappingDigest (.observation 1 2)
    (id "test.evidence.record.step-1") (id "test.rule.digest") [] literalStepOrdering [
      {
        field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.hashed" }
        evidence := .digestToken (id "test.digest.synthetic")
          "synthetic.digest/v1:3006720707513255331"
      },
      {
        field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.role" }
        evidence := .retained "step"
      }
    ] "test.observation.digest/meaning-v1"
]

/-- Observation Evaluation must match the literal mapping identity and every authored Evidence Link field. -/
example : (completeEvidenceBackedTrace.mappingDigest, completeEvidenceBackedTrace.evidenceLinks) =
    (literalMappingDigest, literalEvidenceLinks literalMappingDigest) := by
  native_decide

def literalUncheckedEvidenceBackedTrace : UncheckedEvidenceBackedTrace := {
  completeUncheckedEvidenceBackedTrace with
  mappingDigest := literalMappingDigest
  evidenceLinks := literalEvidenceLinks literalMappingDigest
}

def literalFirstEvidenceLink : EvidenceLink :=
  literalUncheckedEvidenceBackedTrace.evidenceLinks.head?.get (by native_decide)

def missingCoordinateMutation : UncheckedEvidenceBackedTrace := {
  literalUncheckedEvidenceBackedTrace with
  evidenceLinks := literalUncheckedEvidenceBackedTrace.evidenceLinks.tail
}

def duplicateModelCoordinateMutation : UncheckedEvidenceBackedTrace := {
  literalUncheckedEvidenceBackedTrace with
  evidenceLinks := literalFirstEvidenceLink :: literalUncheckedEvidenceBackedTrace.evidenceLinks
}

def shiftedCoordinateMutation : UncheckedEvidenceBackedTrace := {
  literalUncheckedEvidenceBackedTrace with
  evidenceLinks := literalUncheckedEvidenceBackedTrace.evidenceLinks.map fun evidenceLink =>
    if evidenceLink.coordinate == .observation 1 2 then
      { evidenceLink with coordinate := .observation 1 3 }
    else
      evidenceLink
}

def missingOrderingMutation : UncheckedEvidenceBackedTrace := {
  literalUncheckedEvidenceBackedTrace with
  evidenceLinks := literalUncheckedEvidenceBackedTrace.evidenceLinks.map fun evidenceLink => {
    evidenceLink with
    orderingSupport := evidenceLink.orderingSupport.map fun fact =>
      if fact.recordId == stepEvidenceId then
        { fact with causalParents := [stepEvidenceId] }
      else
        fact
  }
}

def redactedCleartextMutation : UncheckedEvidenceBackedTrace := {
  literalUncheckedEvidenceBackedTrace with
  evidenceLinks := [{
    literalFirstEvidenceLink with
    appliedDispositions := [{
      field := { kind := eventKind, field := secretField }
      evidence := .retained "forbidden-secret"
    }]
  }] ++ literalUncheckedEvidenceBackedTrace.evidenceLinks.tail
}

private def rehashAcceptedEnvelope
    (trace : UncheckedEvidenceBackedTrace) : UncheckedEvidenceBackedTrace := {
  trace with
  traceId := (behaviorFingerprintOf <|
    trace.mappingDigest ++ ":" ++ reprStr trace.evidenceIdentities ++ ":" ++
      reprStr trace.recordSupport ++ ":" ++ reprStr trace.trace ++ ":" ++
      reprStr trace.evidenceLinks).render
}

private def updateFirstEvidenceLink
    (trace : UncheckedEvidenceBackedTrace)
    (update : EvidenceLink → EvidenceLink) : UncheckedEvidenceBackedTrace :=
  match trace.evidenceLinks with
  | [] => trace
  | first :: rest => { trace with evidenceLinks := update first :: rest }

private def updateFirstRecordSupport
    (trace : UncheckedEvidenceBackedTrace)
    (update : EvidenceRecordSupport → EvidenceRecordSupport) : UncheckedEvidenceBackedTrace :=
  match trace.recordSupport with
  | [] => trace
  | first :: rest => { trace with recordSupport := update first :: rest }

private def admissionDiagnostic?
    (trace : UncheckedEvidenceBackedTrace) : Option ObservationDiagnostic :=
  match validateEvidenceBackedTrace trace with
  | .ok _ => none
  | .error diagnostic => some diagnostic

def noncanonicalPlanIdentityMutation : UncheckedEvidenceBackedTrace := {
  literalUncheckedEvidenceBackedTrace with
  checkedPlan := {
    literalUncheckedEvidenceBackedTrace.checkedPlan with
    canonicalMetadata := literalUncheckedEvidenceBackedTrace.checkedPlan.canonicalMetadata ++ "/forged"
  }
}

def admissionBoundPlan : CheckedObservationPlan :=
  (checkObservation evaluationContext {
    evaluationDeclaration with evidenceBound := { value := 1, unit := .evidenceRecords }
  }).toOption.get (by native_decide)

def boundOverflowAdmissionMutation : UncheckedEvidenceBackedTrace :=
  let links := literalUncheckedEvidenceBackedTrace.evidenceLinks.map fun evidenceLink => {
    evidenceLink with
    mappingDigest := admissionBoundPlan.behaviorFingerprint.render
    appliedBound := admissionBoundPlan.evidenceBound
  }
  rehashAcceptedEnvelope {
    literalUncheckedEvidenceBackedTrace with
    checkedPlan := admissionBoundPlan
    mappingDigest := admissionBoundPlan.behaviorFingerprint.render
    appliedBound := admissionBoundPlan.evidenceBound
    evidenceLinks := links
  }

def linkMetadataMutations : List UncheckedEvidenceBackedTrace := [
  updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with mappingId := id "test.mapping.forged"
  },
  updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with mappingVersion := link.mappingVersion + 1
  },
  updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with mappingDigest := link.mappingDigest ++ "/forged"
  },
  updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with profileId := id "test.evidence.profile.forged"
  },
  updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with profileVersion := link.profileVersion + 1
  },
  updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with appliedBound := { value := link.appliedBound.value + 1, unit := .evidenceRecords }
  },
  updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with evidenceIdentities := []
  },
  updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with meaningDigest := link.meaningDigest ++ "/forged"
  }
]

def unconsumedIdentityMutation : UncheckedEvidenceBackedTrace := {
  literalUncheckedEvidenceBackedTrace with
  evidenceIdentities := literalUncheckedEvidenceBackedTrace.evidenceIdentities ++
    [id "test.evidence.record.unconsumed"]
}

def duplicateOrderingSupportMutation : UncheckedEvidenceBackedTrace :=
  rehashAcceptedEnvelope <| updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with orderingSupport := link.orderingSupport.head?.toList ++ link.orderingSupport
  }

def duplicateClosureSupportMutation : UncheckedEvidenceBackedTrace :=
  rehashAcceptedEnvelope <| updateFirstEvidenceLink literalUncheckedEvidenceBackedTrace fun link => {
    link with closureSupport := link.closureSupport.head?.toList ++ link.closureSupport
  }

def malformedRecordSupportMutation : UncheckedEvidenceBackedTrace :=
  rehashAcceptedEnvelope <| updateFirstRecordSupport literalUncheckedEvidenceBackedTrace fun support => {
    support with fields := support.fields.head?.toList ++ support.fields
  }

def recordSupportMutations : List UncheckedEvidenceBackedTrace := [
  {
    literalUncheckedEvidenceBackedTrace with
    recordSupport := literalUncheckedEvidenceBackedTrace.recordSupport.tail
  },
  rehashAcceptedEnvelope <| updateFirstRecordSupport literalUncheckedEvidenceBackedTrace fun support => {
    support with origin := some { source := id "test.evidence.source.forged", ordinal := 0 }
  },
  malformedRecordSupportMutation,
  rehashAcceptedEnvelope <| updateFirstRecordSupport literalUncheckedEvidenceBackedTrace fun support => {
    support with fields := match support.fields with
      | [] => []
      | first :: rest => { first with valueType := .natural } :: rest
  },
  rehashAcceptedEnvelope <| updateFirstRecordSupport literalUncheckedEvidenceBackedTrace fun support => {
    support with fields := match support.fields with
      | [] => []
      | first :: rest => { first with evidence := .raw "forged" } :: rest
  },
  rehashAcceptedEnvelope <| updateFirstRecordSupport literalUncheckedEvidenceBackedTrace fun support => {
    support with fields := support.fields.tail
  }
]

def digestPolicyAdmissionMutation : UncheckedEvidenceBackedTrace :=
  rehashAcceptedEnvelope {
    literalUncheckedEvidenceBackedTrace with
    evidenceLinks := literalUncheckedEvidenceBackedTrace.evidenceLinks.map fun link =>
      if link.ruleId == digestRule.id then {
        link with appliedDispositions := link.appliedDispositions.map fun applied =>
          if applied.field.field == hashedField then {
            applied with evidence := .digestToken (id "test.digest.forged") "forged"
          } else applied
      } else link
  }

def expressionAdmissionMutation : UncheckedEvidenceBackedTrace :=
  rehashAcceptedEnvelope {
    literalUncheckedEvidenceBackedTrace with trace := {
      literalUncheckedEvidenceBackedTrace.trace with initialState := {
        literalUncheckedEvidenceBackedTrace.trace.initialState with value := "tampered"
      }
    }
  }

def traceIdentityMutation : UncheckedEvidenceBackedTrace := {
  literalUncheckedEvidenceBackedTrace with
  traceId := literalUncheckedEvidenceBackedTrace.traceId ++ "/forged"
}

/-- Every accepted-envelope mutation fails with its complete admission diagnostic and no trace. -/
example :
    ([noncanonicalPlanIdentityMutation, boundOverflowAdmissionMutation,
        missingCoordinateMutation] ++ linkMetadataMutations ++ [
        unconsumedIdentityMutation,
        duplicateOrderingSupportMutation,
        duplicateClosureSupportMutation
      ] ++ recordSupportMutations ++ [
        redactedCleartextMutation,
        digestPolicyAdmissionMutation,
        expressionAdmissionMutation,
        traceIdentityMutation
      ]).map admissionDiagnostic? = [
      some { kind := .inconsistentEvidenceLink, planId := evaluationDeclaration.id },
      some {
        kind := .evidenceBoundExhausted
        planId := evaluationDeclaration.id
        limit := some { value := 1, unit := .evidenceRecords }
        observedCount := some 2
      },
      some { kind := .absentModelCoordinate, planId := evaluationDeclaration.id },
    ] ++ List.replicate 8 (some {
        kind := .inconsistentEvidenceLink
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [initialRule.id]
      }) ++ [
      some { kind := .unconsumedReference, planId := evaluationDeclaration.id },
      some {
        kind := .missingOrderSupport
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [initialRule.id]
      },
      some {
        kind := .missingClosureSupport
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [eventKind]
      },
      some { kind := .unconsumedReference, planId := evaluationDeclaration.id },
      some {
        kind := .missingOrderSupport
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [initialEvidenceId]
      },
      some {
        kind := .contradictoryFact
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [initialEvidenceId]
      },
      some {
        kind := .normalizationFailure
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [initialEvidenceId, nameField]
      },
      some {
        kind := .inconsistentEvidenceLink
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [initialEvidenceId, nameField]
      },
      some {
        kind := .inconsistentEvidenceLink
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [initialRule.id, nameField]
      },
      some {
        kind := .redactedValueLeakage
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [initialRule.id, secretField]
      },
      some {
        kind := .digestPolicyMismatch
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [digestRule.id, hashedField, id "test.digest.forged"]
      },
      some {
        kind := .inconsistentEvidenceLink
        planId := evaluationDeclaration.id
        relatedDefinitionIds := [initialRule.id]
      },
      some { kind := .inconsistentEvidenceLink, planId := evaluationDeclaration.id }
    ] := by
  native_decide

/-- Competing envelope failures retain plan, bound, coordinate, structural, and field precedence. -/
example : [
    admissionDiagnostic? {
      noncanonicalPlanIdentityMutation with
      evidenceLinks := noncanonicalPlanIdentityMutation.evidenceLinks.tail
    },
    admissionDiagnostic? {
      boundOverflowAdmissionMutation with
      evidenceLinks := boundOverflowAdmissionMutation.evidenceLinks.tail
    },
    admissionDiagnostic? <| updateFirstEvidenceLink missingCoordinateMutation fun link => {
      link with mappingVersion := link.mappingVersion + 1
    },
    admissionDiagnostic? {
      duplicateOrderingSupportMutation with
      recordSupport := malformedRecordSupportMutation.recordSupport
    },
    admissionDiagnostic? {
      malformedRecordSupportMutation with
      traceId := traceIdentityMutation.traceId
    }
  ] = [
    some { kind := .inconsistentEvidenceLink, planId := evaluationDeclaration.id },
    some {
      kind := .evidenceBoundExhausted
      planId := evaluationDeclaration.id
      limit := some { value := 1, unit := .evidenceRecords }
      observedCount := some 2
    },
    some { kind := .absentModelCoordinate, planId := evaluationDeclaration.id },
    some {
      kind := .missingOrderSupport
      planId := evaluationDeclaration.id
      relatedDefinitionIds := [initialRule.id]
    },
    some {
      kind := .contradictoryFact
      planId := evaluationDeclaration.id
      relatedDefinitionIds := [initialEvidenceId]
    }
  ] := by
  native_decide

/-- Missing, duplicate, shifted, unordered, and cleartext-tainted wrappers fail at named boundaries. -/
example : [
    diagnosticKindOf (validateEvidenceBackedTrace missingCoordinateMutation),
    diagnosticKindOf (validateEvidenceBackedTrace duplicateModelCoordinateMutation),
    diagnosticKindOf (validateEvidenceBackedTrace shiftedCoordinateMutation),
    diagnosticKindOf (validateEvidenceBackedTrace missingOrderingMutation),
    diagnosticKindOf (validateEvidenceBackedTrace redactedCleartextMutation)
  ] = [
    some .absentModelCoordinate,
    some .duplicateModelCoordinate,
    some .absentModelCoordinate,
    some .missingOrderSupport,
    some .redactedValueLeakage
  ] := by
  native_decide

/-! Property mutations change only the semantic verdict over the same accepted evidence. -/

def propertyMutationDeclaration : PropertyDeclaration := {
  satisfiedPropertyDeclaration with
  clauses := [
    .stateInvariant (id "test.property.observation.satisfied.initial")
      (verdictPattern .state operationState (.equals "unexpected"))
  ]
}

def propertyMutation : CheckedProperty :=
  (checkProperty verdictPropertyContext (.portable propertyMutationDeclaration))
    |>.toOption.get (by native_decide)

/-- The unchanged evaluation stays valid; only the independently checked Property verdict moves. -/
example :
    let trace := (acceptedOf completeEvaluation).get (by native_decide)
    let baseline := evaluateObservationProperty (verdictQuery [satisfiedProperty])
      satisfiedProperty trace
    let mutant := evaluateObservationProperty (verdictQuery [propertyMutation])
      propertyMutation trace
    (completeEvaluation.status,
      diagnosticKindOf (validateEvidenceBackedTrace literalUncheckedEvidenceBackedTrace),
      baseline.status,
      mutant.status) =
      (.accepted, none, .satisfied, .violated) := by
  native_decide

end Umpire.ObservationTests
