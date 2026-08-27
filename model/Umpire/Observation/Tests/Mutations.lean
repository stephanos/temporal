import Umpire.Observation.Tests.Aggregation

/-!
Independent cross-layer mutations. Expected traces, diagnostics, and verdicts are literal test data;
none is projected from the implementation result being checked.
-/

namespace Umpire.ObservationTests

open Umpire

/-! Model mutations are rejected by the independent qualified-trace comparison, not another layer. -/

def mutatedExpectedTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
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

/-- Literal canonical mapping identity; no expected derivation field is implementation-derived. -/
def literalMappingDigest : String :=
  "umpire-semantic/v1:{\"id\":\"test.mapping.qualification\",\"version\":1,\"profile\":{\"id\":\"test.evidence.profile\",\"version\":1,\"kinds\":[{\"id\":\"test.evidence.kind.event\",\"fields\":[{\"id\":\"test.evidence.field.hashed\",\"type\":\"text\"},{\"id\":\"test.evidence.field.name\",\"type\":\"text\"},{\"id\":\"test.evidence.field.rejected\",\"type\":\"text\"},{\"id\":\"test.evidence.field.role\",\"type\":\"text\"},{\"id\":\"test.evidence.field.secret\",\"type\":\"text\"}]}]},\"digestPolicies\":[{\"id\":\"test.digest.synthetic\",\"name\":\"synthetic.digest\",\"version\":1}],\"bindings\":[{\"id\":\"test.binding.normalized-name\",\"type\":\"text\",\"expression\":{\"expression\":{\"operator\":\"text.trim\",\"version\":1,\"operand\":{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"t" ++
  "est.evidence.field.name\"},\"disposition\":\"retain\",\"policy\":null}},\"type\":\"text\",\"informationFlow\":\"retained\"}}],\"rules\":[{\"id\":\"test.rule.contribution\",\"output\":\"test.observation.contribution\",\"outputKind\":\"observation\",\"meaning\":{\"id\":\"test.observation.contribution\",\"kind\":\"observation\",\"semanticDigest\":\"test.observation.contribution/meaning-v1\"},\"value\":{\"expression\":{\"operator\":\"contribution-marker\",\"operand\":{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.secret\"},\"disposition\":\"redact\",\"policy\":null}},\"type\":\"text\",\"informationFlow\":\"contribution-marker\"},\"condition\":{\"expression\":{\"operator\":\"equals\",\"operands\":[{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field" ++
  "\":\"test.evidence.field.role\"},\"disposition\":\"retain\",\"policy\":null},{\"literal\":\"text\",\"value\":\"step\"}]},\"type\":\"boolean\",\"informationFlow\":\"retained\"}},{\"id\":\"test.rule.digest\",\"output\":\"test.observation.digest\",\"outputKind\":\"observation\",\"meaning\":{\"id\":\"test.observation.digest\",\"kind\":\"observation\",\"semanticDigest\":\"test.observation.digest/meaning-v1\"},\"value\":{\"expression\":{\"operator\":\"digest-token\",\"policy\":{\"id\":\"test.digest.synthetic\",\"name\":\"synthetic.digest\",\"version\":1},\"operand\":{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.hashed\"},\"disposition\":\"hash\",\"policy\":\"test.digest.synthetic\"}},\"type\":\"text\",\"informationFlow\":\"digest-token\"},\"condition\":{\"expres" ++
  "sion\":{\"operator\":\"equals\",\"operands\":[{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.role\"},\"disposition\":\"retain\",\"policy\":null},{\"literal\":\"text\",\"value\":\"step\"}]},\"type\":\"boolean\",\"informationFlow\":\"retained\"}},{\"id\":\"test.rule.initial-state\",\"output\":\"test.state.operation\",\"outputKind\":\"state\",\"meaning\":{\"id\":\"test.state.operation\",\"kind\":\"state\",\"semanticDigest\":\"test.state.operation/meaning-v1\"},\"value\":{\"expression\":{\"binding\":\"test.binding.normalized-name\"},\"type\":\"text\",\"informationFlow\":\"retained\"},\"condition\":{\"expression\":{\"operator\":\"equals\",\"operands\":[{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.role\"},\"disposition\":\"retai" ++
  "n\",\"policy\":null},{\"literal\":\"text\",\"value\":\"initial\"}]},\"type\":\"boolean\",\"informationFlow\":\"retained\"}},{\"id\":\"test.rule.step-action\",\"output\":\"test.action.start\",\"outputKind\":\"action\",\"meaning\":{\"id\":\"test.action.start\",\"kind\":\"action\",\"semanticDigest\":\"test.action.start/meaning-v1\"},\"value\":{\"expression\":{\"literal\":\"text\",\"value\":\"start\"},\"type\":\"text\",\"informationFlow\":\"literal\"},\"condition\":{\"expression\":{\"operator\":\"equals\",\"operands\":[{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.role\"},\"disposition\":\"retain\",\"policy\":null},{\"literal\":\"text\",\"value\":\"step\"}]},\"type\":\"boolean\",\"informationFlow\":\"retained\"}},{\"id\":\"test.rule.step-outcome\",\"output\":\"test.outcom" ++
  "e.success\",\"outputKind\":\"outcome\",\"meaning\":{\"id\":\"test.outcome.success\",\"kind\":\"outcome\",\"semanticDigest\":\"test.outcome.success/meaning-v1\"},\"value\":{\"expression\":{\"literal\":\"text\",\"value\":\"ok\"},\"type\":\"text\",\"informationFlow\":\"literal\"},\"condition\":{\"expression\":{\"operator\":\"equals\",\"operands\":[{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.role\"},\"disposition\":\"retain\",\"policy\":null},{\"literal\":\"text\",\"value\":\"step\"}]},\"type\":\"boolean\",\"informationFlow\":\"retained\"}},{\"id\":\"test.rule.step-state\",\"output\":\"test.state.completed\",\"outputKind\":\"state\",\"meaning\":{\"id\":\"test.state.completed\",\"kind\":\"state\",\"semanticDigest\":\"test.state.completed/meaning-v1\"},\"value\":{\"ex" ++
  "pression\":{\"literal\":\"text\",\"value\":\"done\"},\"type\":\"text\",\"informationFlow\":\"literal\"},\"condition\":{\"expression\":{\"operator\":\"equals\",\"operands\":[{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.role\"},\"disposition\":\"retain\",\"policy\":null},{\"literal\":\"text\",\"value\":\"step\"}]},\"type\":\"boolean\",\"informationFlow\":\"retained\"}}],\"ordering\":[{\"before\":\"test.rule.contribution\",\"after\":\"test.rule.digest\"},{\"before\":\"test.rule.initial-state\",\"after\":\"test.rule.step-action\"},{\"before\":\"test.rule.step-action\",\"after\":\"test.rule.step-outcome\"},{\"before\":\"test.rule.step-outcome\",\"after\":\"test.rule.step-state\"},{\"before\":\"test.rule.step-state\",\"after\":\"test.rule.contribution\"}],\"clo" ++
  "sures\":[{\"kind\":\"test.evidence.kind.event\"}],\"dispositions\":[{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.hashed\"},\"disposition\":\"hash\",\"policy\":\"test.digest.synthetic\"},{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.name\"},\"disposition\":\"retain\",\"policy\":null},{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.rejected\"},\"disposition\":\"reject\",\"policy\":null},{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.role\"},\"disposition\":\"retain\",\"policy\":null},{\"field\":{\"kind\":\"test.evidence.kind.event\",\"field\":\"test.evidence.field.secret\"},\"disposition\":\"redact\",\"policy\":null}],\"evidenceBound\":{\"value" ++
  "\":3,\"unit\":\"evidence-records\"},\"meanings\":[{\"id\":\"test.action.start\",\"kind\":\"action\",\"semanticDigest\":\"test.action.start/meaning-v1\"},{\"id\":\"test.observation.contribution\",\"kind\":\"observation\",\"semanticDigest\":\"test.observation.contribution/meaning-v1\"},{\"id\":\"test.observation.digest\",\"kind\":\"observation\",\"semanticDigest\":\"test.observation.digest/meaning-v1\"},{\"id\":\"test.outcome.success\",\"kind\":\"outcome\",\"semanticDigest\":\"test.outcome.success/meaning-v1\"},{\"id\":\"test.state.completed\",\"kind\":\"state\",\"semanticDigest\":\"test.state.completed/meaning-v1\"},{\"id\":\"test.state.operation\",\"kind\":\"state\",\"semanticDigest\":\"test.state.operation/meaning-v1\"}]}"

def literalDerivation
    (mappingDigest : String)
    (coordinate : SemanticCoordinate)
    (evidenceIdentity ruleId : DefinitionId)
    (bindingIds : List DefinitionId)
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

/-- Qualification must match the literal mapping identity and every authored derivation field. -/
example : (completeQualifiedTrace.mappingDigest, completeQualifiedTrace.derivations) =
    (literalMappingDigest, literalDerivations literalMappingDigest) := by
  native_decide

def literalQualifiedTrace : QualifiedTrace := {
  completeQualifiedTrace with
  mappingDigest := literalMappingDigest
  derivations := literalDerivations literalMappingDigest
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
