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

def guardedSwitchPropertyDeclaration : Property := {
  Umpire.Examples.Switch.authoredProperty with
  id := DefinitionId.of "test.run-evaluation.property.guarded"
  version := 2
  clauses := [.branches {
    id := DefinitionId.of "test.run-evaluation.property.guarded.group"
    source := Umpire.Examples.Switch.source
    guard := .atom {
      field := .selectedAction
      reference := Umpire.Examples.Switch.flipActionId
      constraint := .equals (.text "flip")
    }
    cases := [{
      id := DefinitionId.of "test.run-evaluation.property.guarded.case"
      source := Umpire.Examples.Switch.source
      guard := .atom {
        field := .priorState
        reference := Umpire.Examples.Switch.powerStateId
        constraint := .equals (.text "off")
      }
      clauses := [{
        id := DefinitionId.of "test.run-evaluation.property.guarded.case.state"
        source := Umpire.Examples.Switch.source
        expectation := .atom {
          field := .resultingState
          reference := Umpire.Examples.Switch.powerStateId
          constraint := .equals (.text "on")
        }
      }]
    }]
    complete := true
    exclusive := true
  }]
}

def guardedTemporalSwitchPropertyDeclaration : Property := {
  Umpire.Examples.Switch.authoredProperty with
  id := DefinitionId.of "test.run-evaluation.property.guarded-temporal"
  version := 2
  clauses := [.eventuallyWithin
      (id := (DefinitionId.of "test.run-evaluation.property.guarded-temporal.clause"))
      (source := Umpire.Examples.Switch.source)
      (guard := some (.atom {
      field := .selectedAction
      reference := Umpire.Examples.Switch.flipActionId
      constraint := .equals (.text "flip")
    }))
      (exception := none)
      (trigger := {
      field := .selectedAction
      reference := Umpire.Examples.Switch.flipActionId
      constraint := .present
    })
      (response := {
      field := .outcome
      reference := Umpire.Examples.Switch.appliedOutcomeId
      constraint := .present
    })
      (limit := { value := 0, unit := .steps })]
}

private def guardedRunEvaluationResult : Option
    (StrictQueryStatus × SemanticVerdictStatus × Option SemanticVerdictFailureKind) := do
  let property ← (Property.check
    (PropertyCheckContext.ofTarget Umpire.Examples.Switch.target)
    (guardedSwitchPropertyDeclaration)).toOption
  let query := {
    Umpire.Examples.Switch.exploratoryQuery with form := .pick [property]
  }
  let evaluation := checkRunEvaluation observationPlan repeatedEvidence checkedLink
    Umpire.Examples.Switch.switchSetup query [property]
  let verdict ← evaluation.querySummary.verdicts.head?
  pure (evaluation.querySummary.status, verdict.status,
    verdict.diagnostic.map SemanticVerdictDiagnostic.kind)

/- Checked guarded Properties cannot pass through translated Observation success. -/
#guard guardedRunEvaluationResult ==
  some (.incomplete, .unsupported, some .unsupportedPropertyClause)

private def guardedTemporalRunEvaluationResult : Option
    (StrictQueryStatus × SemanticVerdictStatus ×
      Option (SemanticVerdictFailureKind × List DefinitionId)) := do
  let property ← (Property.check
    (PropertyCheckContext.ofTarget Umpire.Examples.Switch.target)
    (guardedTemporalSwitchPropertyDeclaration)).toOption
  let query := {
    Umpire.Examples.Switch.exploratoryQuery with form := .pick [property]
  }
  let evaluation := checkRunEvaluation observationPlan repeatedEvidence checkedLink
    Umpire.Examples.Switch.switchSetup query [property]
  let verdict ← evaluation.querySummary.verdicts.head?
  pure (evaluation.querySummary.status, verdict.status,
    verdict.diagnostic.map fun diagnostic =>
      (diagnostic.kind, diagnostic.relatedDefinitionIds))

/- Translated Observation cannot emit success after dropping a temporal trigger guard. -/
#guard guardedTemporalRunEvaluationResult == some (
  .incomplete,
  .unsupported,
  some (.unsupportedPropertyClause,
    [DefinitionId.of "test.run-evaluation.property.guarded-temporal.clause"]))

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
    let expected := (evaluatePropertyOnTrace Umpire.Examples.Switch.flipProperty
      Umpire.Examples.Switch.appliedTrace.trace).toOption.get (by native_decide)
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
        .state 1,
        .selectedAction 2,
        .state 2
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

def divergentObservationFailureProperty : CheckedProperty := {
  Umpire.Examples.Switch.flipProperty with clauses := []
}

def divergentObservationFailureRunEvaluation := checkRunEvaluation observationPlan
  { repeatedEvidence with closures := [] } checkedLink Umpire.Examples.Switch.switchSetup
  Umpire.Examples.Switch.exploratoryQuery [divergentObservationFailureProperty]

def divergentObservedRunEvaluation := checkObservedRunEvaluation
  (evaluateEvidence observationPlan { repeatedEvidence with closures := [] }) checkedLink
  checkedObservedTranslation Umpire.Examples.Switch.switchSetup
  Umpire.Examples.Switch.exploratoryQuery [divergentObservationFailureProperty]

/-- Query/Property identity remains authoritative when Observation fails before Link invocation. -/
example :
    ((divergentObservationFailureRunEvaluation.implementationLink.map
        ImplementationLinkResult.status,
      divergentObservationFailureRunEvaluation.querySummary.verdicts.map fun verdict =>
        (verdict.status, verdict.diagnostic.map SemanticVerdictDiagnostic.kind)),
      (divergentObservedRunEvaluation.implementationLink.map
        ObservedTraceTranslationResult.status,
      divergentObservedRunEvaluation.querySummary.verdicts.map fun verdict =>
        (verdict.status, verdict.diagnostic.map SemanticVerdictDiagnostic.kind))) =
      ((none, [(.unsupported, some .queryPropertyMismatch)]),
        (none, [(.unsupported, some .queryPropertyMismatch)])) := by
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

def logicalTimePropertyDeclaration : Property := {
  Umpire.Examples.Switch.authoredProperty with
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
  (Property.check (PropertyCheckContext.ofTarget Umpire.Examples.Switch.target)
    (logicalTimePropertyDeclaration)).toOption.get (by native_decide)

def logicalTimeQuery : CheckedQuery Umpire.Examples.Switch.LawStatement :=
  (Query.check (QueryCheckContext.ofTarget Umpire.Examples.Switch.target) {
    id := DefinitionId.of "test.run-evaluation.query.logical-time"
    source := Umpire.Examples.Switch.source
    target := Umpire.Examples.Switch.target.id
    form := .pick [logicalTimeProperty]
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

def otherModelSpec : ModelSpec Umpire.Examples.Switch.LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  Umpire.Examples.Switch.modelSpec with
  id := otherTargetId
  definitions := Umpire.Examples.Switch.modelSpec.definitions.map fun definition =>
    if definition.id == Umpire.Examples.Switch.targetId then
      { definition with id := otherTargetId }
    else
      definition
}

def otherTargetAuthoring : DraftModel Umpire.Examples.Switch.LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make otherModelSpec Umpire.Examples.Switch.modelProviders
    (.available Umpire.Examples.Switch.machine rfl Umpire.Examples.Switch.finitePlanning)

def otherTarget : QueryModel Umpire.Examples.Switch.LawStatement :=
  model otherTargetAuthoring

def otherTargetProperty : CheckedProperty :=
  (Property.check (PropertyCheckContext.ofTarget otherTarget)
    (Umpire.Examples.Switch.authoredProperty)).toOption.get (by native_decide)

def otherTargetQuery : CheckedQuery Umpire.Examples.Switch.LawStatement :=
  (Query.check (QueryCheckContext.ofTarget otherTarget) {
    id := DefinitionId.of "test.run-evaluation.query.other-target"
    source := Umpire.Examples.Switch.source
    target := otherTarget.id
    form := .pick [otherTargetProperty]
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

def initialOffPropertyDeclaration : Property := {
  Umpire.Examples.Switch.authoredProperty with
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
  (Property.check (PropertyCheckContext.ofTarget Umpire.Examples.Switch.target)
    (initialOffPropertyDeclaration)).toOption.get (by native_decide)

def twoPropertyQuery : CheckedQuery Umpire.Examples.Switch.LawStatement := {
  Umpire.Examples.Switch.exploratoryQuery with
  form := .pick [Umpire.Examples.Switch.flipProperty, initialOffProperty]
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
