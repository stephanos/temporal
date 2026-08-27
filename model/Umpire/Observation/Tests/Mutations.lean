import Umpire.Observation.Tests.Aggregation

/-!
Independent cross-layer mutations. Expected traces, diagnostics, and verdicts are literal test data;
none is projected from the implementation result being checked.
-/

namespace Umpire.ObservationTests

open Umpire

/-! Model mutations are rejected by the independent qualified-trace comparison, not another layer. -/

def mutatedExpectedTrace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
  expectedTrace with
  initialState := { expectedTrace.initialState with value := "unexpected" }
}

/-- A model-only mutation leaves qualification valid while failing the independently authored oracle. -/
example :
    let actual := (qualifiedOf completeQualification).map QualifiedTrace.trace
    (completeQualification.status, actual, actual == some mutatedExpectedTrace) =
      (.qualified, some expectedTrace, false) := by
  native_decide

/-! Mapping mutations fail at compilation before any evidence can be interpreted. -/

def unknownOperatorMutation : ObservationMappingDeclaration := {
  baseDeclaration with
  bindings := [{ normalizedName with
    expression := .portable
      (.normalize { name := "text.unknown", version := 1 } (field nameField)) }]
}

def wrongBindingTypeMutation : ObservationMappingDeclaration := {
  baseDeclaration with
  bindings := [{ normalizedName with valueType := .natural }]
}

def clearValueTaintMutation : ObservationMappingDeclaration := {
  baseDeclaration with
  rules := baseDeclaration.rules.map fun rule =>
    if rule.id == contributionRule.id then
      { rule with value := .portable (field secretField) }
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

/-! Evidence-volume mutations fail at qualification, after the mapping has compiled. -/

def boundedDeclaration : ObservationMappingDeclaration := {
  qualificationDeclaration with
  evidenceBound := { value := 2, unit := .evidenceRecords }
}

def boundedPlan : CheckedObservationPlan :=
  (checkObservation qualificationContext boundedDeclaration).toOption.get (by native_decide)

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

/-- N records qualify, while N+1 is unknown with the literal bound diagnostic and no trace. -/
example :
    let atLimit := qualifyEvidence boundedPlan completeEvidence
    let overLimit := qualifyEvidence boundedPlan limitPlusOneEvidence
    (atLimit.status, overLimit, qualifiedOf overLimit) = (
      .qualified,
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

def literalDerivation
    (mappingDigest : String)
    (coordinate : SemanticCoordinate)
    (evidenceIdentity ruleId : DeclarationId)
    (bindingIds : List DeclarationId)
    (orderingSupport : List EvidenceOrderingFact)
    (appliedDispositions : List AppliedFieldDisposition)
    (meaningDigest : String) : SemanticDerivation := {
  coordinate
  mappingId := id "test.mapping.qualification"
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

/-- Independently authored derivations for every semantic slot in `expectedTrace`. -/
def literalDerivations (mappingDigest : String) : List SemanticDerivation := [
  literalDerivation mappingDigest .initialState
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
  literalDerivation mappingDigest (.selectedAction 1)
    (id "test.evidence.record.step-1") (id "test.rule.step-action") [] literalStepOrdering [{
      field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.role" }
      evidence := .retained "step"
    }] "test.action.start/meaning-v1",
  literalDerivation mappingDigest (.modelOutcome 1)
    (id "test.evidence.record.step-1") (id "test.rule.step-outcome") [] literalStepOrdering [{
      field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.role" }
      evidence := .retained "step"
    }] "test.outcome.success/meaning-v1",
  literalDerivation mappingDigest (.resultingState 1)
    (id "test.evidence.record.step-1") (id "test.rule.step-state") [] literalStepOrdering [{
      field := { kind := id "test.evidence.kind.event", field := id "test.evidence.field.role" }
      evidence := .retained "step"
    }] "test.state.completed/meaning-v1",
  literalDerivation mappingDigest (.observation 1 1)
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
  literalDerivation mappingDigest (.observation 1 2)
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

/-- Qualification must match every independently authored derivation field and slot. -/
example : completeQualifiedTrace.derivations =
    literalDerivations completeQualifiedTrace.mappingDigest := by
  native_decide

def literalQualifiedTrace : QualifiedTrace := {
  completeQualifiedTrace with
  derivations := literalDerivations completeQualifiedTrace.mappingDigest
}

def literalFirstDerivation : SemanticDerivation :=
  literalQualifiedTrace.derivations.head?.get (by native_decide)

def missingCoordinateMutation : QualifiedTrace := {
  literalQualifiedTrace with
  derivations := literalQualifiedTrace.derivations.tail
}

def duplicateCoordinateMutation : QualifiedTrace := {
  literalQualifiedTrace with
  derivations := literalFirstDerivation :: literalQualifiedTrace.derivations
}

def shiftedCoordinateMutation : QualifiedTrace := {
  literalQualifiedTrace with
  derivations := literalQualifiedTrace.derivations.map fun derivation =>
    if derivation.coordinate == .observation 1 2 then
      { derivation with coordinate := .observation 1 3 }
    else
      derivation
}

def missingOrderingMutation : QualifiedTrace := {
  literalQualifiedTrace with
  derivations := literalQualifiedTrace.derivations.map fun derivation => {
    derivation with
    orderingSupport := derivation.orderingSupport.map fun fact =>
      if fact.recordId == stepEvidenceId then
        { fact with causalParents := [stepEvidenceId] }
      else
        fact
  }
}

def redactedCleartextMutation : QualifiedTrace := {
  literalQualifiedTrace with
  derivations := [{
    literalFirstDerivation with
    appliedDispositions := [{
      field := { kind := eventKind, field := secretField }
      evidence := .retained "forbidden-secret"
    }]
  }] ++ literalQualifiedTrace.derivations.tail
}

/-- Missing, duplicate, shifted, unordered, and cleartext-tainted wrappers fail at named boundaries. -/
example : [
    diagnosticKindOf (validateQualifiedTrace missingCoordinateMutation),
    diagnosticKindOf (validateQualifiedTrace duplicateCoordinateMutation),
    diagnosticKindOf (validateQualifiedTrace shiftedCoordinateMutation),
    diagnosticKindOf (validateQualifiedTrace missingOrderingMutation),
    diagnosticKindOf (validateQualifiedTrace redactedCleartextMutation)
  ] = [
    some .absentCoordinate,
    some .duplicateCoordinate,
    some .absentCoordinate,
    some .missingOrderSupport,
    some .redactedValueLeakage
  ] := by
  native_decide

/-! Property mutations change only the semantic verdict over the same qualified evidence. -/

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

/-- The unchanged qualification stays valid; only the independently checked Property verdict moves. -/
example :
    let baseline := evaluateQualifiedProperty (verdictQuery [satisfiedProperty])
      satisfiedProperty completeQualification
    let mutant := evaluateQualifiedProperty (verdictQuery [propertyMutation])
      propertyMutation completeQualification
    (completeQualification.status,
      diagnosticKindOf (validateQualifiedTrace literalQualifiedTrace),
      baseline.status,
      mutant.status) =
      (.qualified, none, .satisfied, .violated) := by
  native_decide

end Umpire.ObservationTests
