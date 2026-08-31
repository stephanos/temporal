import Umpire.ImplementationLink.Tests.Application
import Umpire.Observation.Check

/-! Domain-neutral Run Evaluation composition over checked Observation and Implementation Link inputs. -/

namespace Umpire.ObservationCheckTests

open Umpire
open Umpire.ImplementationLinkApplicationTests

def repeatedRunEvaluation := checkRunEvaluation observationPlan repeatedEvidence checkedLink
  Umpire.Examples.Switch.switchSetup Umpire.Examples.Switch.exploratoryQuery
  [Umpire.Examples.Switch.flipProperty]

def satisfiedObservationDeclaration : ObservationMappingDeclaration := {
  observationDeclaration with
  id := DefinitionId.of "test.run-evaluation.observation.satisfied"
  rules := observationDeclaration.rules.map fun rule =>
    if rule.id == outcomeRuleId then
      { rule with output := Umpire.Examples.Switch.appliedOutcomeId }
    else
      rule
}

def satisfiedObservationPlan : CheckedObservationPlan :=
  (checkObservation
    (ObservationCheckContext.ofTarget Umpire.Examples.Switch.target [evidenceProfile])
    satisfiedObservationDeclaration).toOption.get (by native_decide)

def satisfiedStepRecord : SyntheticEvidenceRecord :=
  let base := stepRecord firstStepRecordId 2 initialRecordId "applied" "on"
  { base with fields := base.fields.map fun fieldValue =>
      if fieldValue.field == observationField then
        { field := observationField, value := .text "on" }
      else
        fieldValue }

def satisfiedEvidence : EvidenceBundle := {
  profile := profileId
  profileVersion := 1
  records := [satisfiedStepRecord, initialRecord]
  closures := [{ kind := evidenceKind, lastSequence := 2 }]
}

def satisfiedRunEvaluation := checkRunEvaluation satisfiedObservationPlan satisfiedEvidence checkedLink
  Umpire.Examples.Switch.switchSetup Umpire.Examples.Switch.exploratoryQuery
  [Umpire.Examples.Switch.flipProperty]

/-- Accepted Observation reaches the translated Feature trace and preserves a violated verdict. -/
example :
    (repeatedRunEvaluation.observation.status,
      repeatedRunEvaluation.implementationLink.map ImplementationLinkResult.status,
      repeatedRunEvaluation.querySummary.status,
      repeatedRunEvaluation.querySummary.verdicts.map fun verdict =>
        (verdict.propertyId, verdict.status)) =
      (.accepted, some .applied, .violated,
        [(Umpire.Examples.Switch.flipPropertyId, .violated)]) := by
  native_decide

/-- Accepted composition preserves every clause result from the unchanged Feature evaluator. -/
example :
    let expected := evaluateProperty Umpire.Examples.Switch.flipProperty
      Umpire.Examples.Switch.appliedTrace.trace
    (satisfiedRunEvaluation.querySummary.status,
      satisfiedRunEvaluation.querySummary.verdicts.map fun verdict =>
        (verdict.status, verdict.clauses.map fun clause =>
          (clause.clauseId, clause.status))) =
      (.satisfied, [(.satisfied, expected.clauses.map fun clause =>
        (clause.clauseId, if clause.satisfied then .satisfied else .violated))]) := by
  native_decide

/-- Repeated equal Feature values retain their distinct positional Evidence Links. -/
example :
    repeatedRunEvaluation.querySummary.verdicts.flatMap (fun verdict =>
      verdict.clauses.flatMap SemanticClauseVerdict.coordinates) = [
        .selectedAction 1,
        .resultingState 1,
        .selectedAction 2,
        .resultingState 2
      ] := by
  native_decide

def observationFailureRunEvaluation := checkRunEvaluation observationPlan
  { repeatedEvidence with closures := [] } checkedLink Umpire.Examples.Switch.switchSetup
  Umpire.Examples.Switch.exploratoryQuery [Umpire.Examples.Switch.flipProperty]

/-- A non-success Observation emits the complete unresolved partition and skips the link. -/
example :
    ((observationFailureRunEvaluation.observation.status,
      observationFailureRunEvaluation.implementationLink.map ImplementationLinkResult.status,
      observationFailureRunEvaluation.querySummary.status,
      observationFailureRunEvaluation.querySummary.verdicts.map fun verdict =>
        (verdict.status, verdict.traceId, verdict.clauses.isEmpty)) ==
      (.unknown, none, .incomplete, [(.unknown, none, true)])) = true := by
  native_decide

def implementationLinkFailureRunEvaluation := checkRunEvaluation observationPlan repeatedEvidence
  checkedLink [] Umpire.Examples.Switch.exploratoryQuery [Umpire.Examples.Switch.flipProperty]

/-- A failed checked Implementation Link remains distinct and cannot become a Property result. -/
example :
    (implementationLinkFailureRunEvaluation.implementationLink.map ImplementationLinkResult.status,
      implementationLinkFailureRunEvaluation.querySummary.status,
      implementationLinkFailureRunEvaluation.querySummary.verdicts.map fun verdict =>
        (verdict.status, verdict.clauses.isEmpty,
          verdict.diagnostic.map SemanticVerdictDiagnostic.kind)) =
      (some .invalid, .incomplete,
        [(.unsupported, true, some .semanticTraceUnavailable)]) := by
  native_decide

def logicalTimePropertyDeclaration : PropertyDeclaration := {
  Umpire.Examples.Switch.propertyDeclaration with
  id := DefinitionId.of "test.run-evaluation.property.logical-time"
  logicalTimeSource := some Umpire.Examples.Switch.powerObservationId
  clauses := [
    .ordered (DefinitionId.of "test.run-evaluation.property.logical-time.clause")
      {
        field := .observation
        reference := Umpire.Examples.Switch.powerObservationId
        constraint := .present
      }
      {
        field := .observation
        reference := Umpire.Examples.Switch.powerObservationId
        constraint := .present
      }
      .logicalTime
  ]
}

def logicalTimeProperty : CheckedProperty :=
  (checkProperty (PropertyCheckContext.ofTarget Umpire.Examples.Switch.target)
    (.portable logicalTimePropertyDeclaration)).toOption.get (by native_decide)

def logicalTimeQuery : CheckedQuery Umpire.Examples.Switch.LawStatement :=
  (checkQuery (QueryCheckContext.ofTarget Umpire.Examples.Switch.target) {
    id := DefinitionId.of "test.run-evaluation.query.logical-time"
    source := Umpire.Examples.Switch.source
    target := Umpire.Examples.Switch.target.id
    form := .select [logicalTimeProperty]
    behavior := Umpire.Examples.Switch.exploratoryQuery.behavior
    limits := Umpire.Examples.Switch.exploratoryQuery.limits
    policy := Umpire.Examples.Switch.exploratoryQuery.policy
  }).toOption.get (by native_decide)

def missingLogicalTimeRunEvaluation := checkRunEvaluation observationPlan repeatedEvidence checkedLink
  Umpire.Examples.Switch.switchSetup logicalTimeQuery [logicalTimeProperty]

/-- Invalid logical time is unresolved before the unchanged Property evaluator can report false. -/
example :
    (missingLogicalTimeRunEvaluation.implementationLink.map ImplementationLinkResult.status,
      missingLogicalTimeRunEvaluation.querySummary.status,
      missingLogicalTimeRunEvaluation.querySummary.verdicts.map fun verdict =>
        (verdict.status, verdict.clauses.isEmpty,
          verdict.diagnostic.map SemanticVerdictDiagnostic.kind)) =
      (some .applied, .incomplete, [(.unknown, true, some .missingLogicalTime)]) := by
  native_decide

def otherTargetId : DefinitionId := DefinitionId.of "test.run-evaluation.target.other"

def otherTargetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  Umpire.Examples.Switch.targetDefinition with
  id := otherTargetId
  definitions := Umpire.Examples.Switch.targetDefinition.definitions.map fun definition =>
    if definition.id == Umpire.Examples.Switch.targetId then
      { definition with id := otherTargetId }
    else
      definition
}

def otherTargetAuthoring : AuthoredTarget Umpire.Examples.Switch.LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make otherTargetDefinition Umpire.Examples.Switch.targetComposition
    (.available Umpire.Examples.Switch.transitionKernel rfl Umpire.Examples.Switch.finitePlanning)

def otherTarget : QueryTarget Umpire.Examples.Switch.LawStatement :=
  checkedTarget otherTargetAuthoring

def otherTargetProperty : CheckedProperty :=
  (checkProperty (PropertyCheckContext.ofTarget otherTarget)
    (.portable Umpire.Examples.Switch.propertyDeclaration)).toOption.get (by native_decide)

def otherTargetQuery : CheckedQuery Umpire.Examples.Switch.LawStatement :=
  (checkQuery (QueryCheckContext.ofTarget otherTarget) {
    id := DefinitionId.of "test.run-evaluation.query.other-target"
    source := Umpire.Examples.Switch.source
    target := otherTarget.id
    form := .select [otherTargetProperty]
    behavior := Umpire.Examples.Switch.exploratoryQuery.behavior
    limits := Umpire.Examples.Switch.exploratoryQuery.limits
    policy := Umpire.Examples.Switch.exploratoryQuery.policy
  }).toOption.get (by native_decide)

def mismatchedTargetRunEvaluation := checkRunEvaluation observationPlan repeatedEvidence checkedLink
  Umpire.Examples.Switch.switchSetup otherTargetQuery [otherTargetProperty]

/-- A Query checked for another destination target cannot reach Property evaluation. -/
example :
    (mismatchedTargetRunEvaluation.implementationLink.map ImplementationLinkResult.status,
      mismatchedTargetRunEvaluation.querySummary.status,
      mismatchedTargetRunEvaluation.querySummary.verdicts.map fun verdict =>
        (verdict.status, verdict.clauses.isEmpty,
          verdict.diagnostic.map SemanticVerdictDiagnostic.kind)) =
      (some .applied, .incomplete,
        [(.unsupported, true, some .semanticTraceUnavailable)]) := by
  native_decide

def incompleteRunEvaluation := checkRunEvaluation observationPlan repeatedEvidence checkedLink
  Umpire.Examples.Switch.switchSetup Umpire.Examples.Switch.exploratoryQuery []

/-- Incomplete Property inputs remain inspectable and cannot produce a partial success. -/
example :
    (incompleteRunEvaluation.querySummary.status,
      incompleteRunEvaluation.querySummary.verdicts,
      incompleteRunEvaluation.querySummary.missingProperties) =
      (.incomplete, [], [Umpire.Examples.Switch.flipPropertyId]) := by
  native_decide

def initialOffPropertyDeclaration : PropertyDeclaration := {
  Umpire.Examples.Switch.propertyDeclaration with
  id := DefinitionId.of "test.run-evaluation.property.initial-off"
  clauses := [
    .stateInvariant (DefinitionId.of "test.run-evaluation.property.initial-off.clause") {
      field := .state
      reference := Umpire.Examples.Switch.powerStateId
      constraint := .equals Umpire.Examples.Switch.offState.value
    }
  ]
}

def initialOffProperty : CheckedProperty :=
  (checkProperty (PropertyCheckContext.ofTarget Umpire.Examples.Switch.target)
    (.portable initialOffPropertyDeclaration)).toOption.get (by native_decide)

def twoPropertyQuery : CheckedQuery Umpire.Examples.Switch.LawStatement := {
  Umpire.Examples.Switch.exploratoryQuery with
  form := .select [Umpire.Examples.Switch.flipProperty, initialOffProperty]
}

def orderedRunEvaluation (properties : List CheckedProperty) :=
  checkRunEvaluation observationPlan repeatedEvidence checkedLink
    Umpire.Examples.Switch.switchSetup twoPropertyQuery properties

def overBoundEvidence : EvidenceBundle := {
  repeatedEvidence with
  records := repeatedEvidence.records ++ [{
    stepRecord (DefinitionId.of "test.run-evaluation.evidence.step-3") 4 secondStepRecordId with
    causalParents := [secondStepRecordId]
  }]
  closures := [{ kind := evidenceKind, lastSequence := 4 }]
}

def overBoundRunEvaluation := checkRunEvaluation observationPlan overBoundEvidence checkedLink
  Umpire.Examples.Switch.switchSetup Umpire.Examples.Switch.exploratoryQuery
  [Umpire.Examples.Switch.flipProperty]

/-- N Evidence records evaluate normally; N+1 is unresolved before link or Property evaluation. -/
example :
    ((repeatedRunEvaluation.observation.status,
      overBoundRunEvaluation.observation.status,
      overBoundRunEvaluation.observation.diagnostic?.map ObservationDiagnostic.kind,
      overBoundRunEvaluation.implementationLink.map ImplementationLinkResult.status,
      overBoundRunEvaluation.querySummary.verdicts.map fun verdict =>
        (verdict.status, verdict.clauses.isEmpty)) ==
      (.accepted, .unknown, some .evidenceBoundExhausted, none, [(.unknown, true)])) = true := by
  native_decide

/-- Checked Property values are reusable and output ordering is independent of supplied order. -/
example :
    let first := orderedRunEvaluation [initialOffProperty, Umpire.Examples.Switch.flipProperty]
    let second := orderedRunEvaluation [Umpire.Examples.Switch.flipProperty, initialOffProperty]
    first.querySummary = second.querySummary ∧
      first.querySummary.verdicts.map SemanticPropertyVerdict.propertyId = [
        Umpire.Examples.Switch.flipPropertyId,
        initialOffProperty.id
      ] := by
  native_decide

def observedRunEvaluation := checkObservedRunEvaluation
  (evaluateEvidence observationPlan observedEvidence) checkedLink checkedObservedTranslation
  Umpire.Examples.Switch.switchSetup Umpire.Examples.Switch.exploratoryQuery
  [Umpire.Examples.Switch.flipProperty]

/-- The shared composition kernel retains the observed translation while exposing no authority
claim for its translated trace. -/
example :
    observedRunEvaluation.implementationLink.map (fun result =>
      (result.status, result.translated?.map TranslatedObservedTrace.hasAuthorityClaim)) =
        some (.applied, some false) := by
  native_decide

/-- Adding the observed adapter does not change the strict composition result. -/
example :
    (repeatedRunEvaluation.observation.status,
      repeatedRunEvaluation.implementationLink.map ImplementationLinkResult.status,
      repeatedRunEvaluation.querySummary.status) = (.accepted, some .applied, .violated) := by
  native_decide

end Umpire.ObservationCheckTests
