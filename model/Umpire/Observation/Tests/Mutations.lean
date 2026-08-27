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

def literalEvidenceBackedTrace : EvidenceBackedTrace := {
  completeEvidenceBackedTrace with
  mappingDigest := literalMappingDigest
  evidenceLinks := literalEvidenceLinks literalMappingDigest
}

def literalFirstEvidenceLink : EvidenceLink :=
  literalEvidenceBackedTrace.evidenceLinks.head?.get (by native_decide)

def missingCoordinateMutation : EvidenceBackedTrace := {
  literalEvidenceBackedTrace with
  evidenceLinks := literalEvidenceBackedTrace.evidenceLinks.tail
}

def duplicateModelCoordinateMutation : EvidenceBackedTrace := {
  literalEvidenceBackedTrace with
  evidenceLinks := literalFirstEvidenceLink :: literalEvidenceBackedTrace.evidenceLinks
}

def shiftedCoordinateMutation : EvidenceBackedTrace := {
  literalEvidenceBackedTrace with
  evidenceLinks := literalEvidenceBackedTrace.evidenceLinks.map fun evidenceLink =>
    if evidenceLink.coordinate == .observation 1 2 then
      { evidenceLink with coordinate := .observation 1 3 }
    else
      evidenceLink
}

def missingOrderingMutation : EvidenceBackedTrace := {
  literalEvidenceBackedTrace with
  evidenceLinks := literalEvidenceBackedTrace.evidenceLinks.map fun evidenceLink => {
    evidenceLink with
    orderingSupport := evidenceLink.orderingSupport.map fun fact =>
      if fact.recordId == stepEvidenceId then
        { fact with causalParents := [stepEvidenceId] }
      else
        fact
  }
}

def redactedCleartextMutation : EvidenceBackedTrace := {
  literalEvidenceBackedTrace with
  evidenceLinks := [{
    literalFirstEvidenceLink with
    appliedDispositions := [{
      field := { kind := eventKind, field := secretField }
      evidence := .retained "forbidden-secret"
    }]
  }] ++ literalEvidenceBackedTrace.evidenceLinks.tail
}

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
    let baseline := evaluateObservationProperty (verdictQuery [satisfiedProperty])
      satisfiedProperty completeEvaluation
    let mutant := evaluateObservationProperty (verdictQuery [propertyMutation])
      propertyMutation completeEvaluation
    (completeEvaluation.status,
      diagnosticKindOf (validateEvidenceBackedTrace literalEvidenceBackedTrace),
      baseline.status,
      mutant.status) =
      (.accepted, none, .satisfied, .violated) := by
  native_decide

end Umpire.ObservationTests
