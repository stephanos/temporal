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

end Umpire.ObservationCheckTests
