import Temporal.Feature.Nexus.Operations
import Temporal.Shared
import Umpire.Observation

/-!
One synthetic Evidence profile for the ordinary Nexus lifecycle. This module performs only the
offline handoff from a finite typed `EvidenceBundle` to Observation Evaluation and semantic
verdicts; it does not start Temporal, collect live evidence, persist raw records, or promote a
result.
-/

namespace Temporal.Feature.Nexus.Observation

open Umpire
open Temporal.Feature.Nexus.Lifecycle

private def definitionId (value : String) : DefinitionId :=
  Temporal.Shared.definitionId value

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Observation.lean"

namespace Profile

def id : DefinitionId := definitionId "temporal.nexus.synthetic.basic-lifecycle.profile"
def lifecycleKind : DefinitionId :=
  definitionId "temporal.nexus.synthetic.basic-lifecycle.kind.lifecycle"
def stateFieldSpec : ObservationFieldSpec := {
  kind := lifecycleKind
  field := definitionId "temporal.nexus.synthetic.basic-lifecycle.field.state"
  valueType := .text
}
def actionFieldSpec : ObservationFieldSpec := {
  kind := lifecycleKind
  field := definitionId "temporal.nexus.synthetic.basic-lifecycle.field.action"
  valueType := .text
}
def outcomeFieldSpec : ObservationFieldSpec := {
  kind := lifecycleKind
  field := definitionId "temporal.nexus.synthetic.basic-lifecycle.field.outcome"
  valueType := .text
}
def observationFieldSpec : ObservationFieldSpec := {
  kind := lifecycleKind
  field := definitionId "temporal.nexus.synthetic.basic-lifecycle.field.observation"
  valueType := .text
}
def rejectedFieldSpec : ObservationFieldSpec := {
  kind := lifecycleKind
  field := definitionId "temporal.nexus.synthetic.basic-lifecycle.field.raw-detail"
  valueType := .text
}
def stateField : DefinitionId := stateFieldSpec.field
def actionField : DefinitionId := actionFieldSpec.field
def outcomeField : DefinitionId := outcomeFieldSpec.field
def observationField : DefinitionId := observationFieldSpec.field
def rejectedField : DefinitionId := rejectedFieldSpec.field

/-- The sole synthetic Temporal profile admitted by this Observation mapping. -/
def declaration : EvidenceProfileDeclaration := {
  id
  source
  kinds := [{
    id := lifecycleKind
    fields := [
      stateFieldSpec.declaration,
      actionFieldSpec.declaration,
      outcomeFieldSpec.declaration,
      observationFieldSpec.declaration,
      rejectedFieldSpec.declaration
    ]
  }]
}

end Profile

namespace Mapping

def id : DefinitionId := definitionId "temporal.nexus.synthetic.basic-lifecycle.mapping"
def stateRuleId : DefinitionId :=
  definitionId "temporal.nexus.synthetic.basic-lifecycle.rule.state"
def startRuleId : DefinitionId :=
  definitionId "temporal.nexus.synthetic.basic-lifecycle.rule.action.start"
def cancelRuleId : DefinitionId :=
  definitionId "temporal.nexus.synthetic.basic-lifecycle.rule.action.cancel"
def succeedRuleId : DefinitionId :=
  definitionId "temporal.nexus.synthetic.basic-lifecycle.rule.action.succeed"
def outcomeRuleId : DefinitionId :=
  definitionId "temporal.nexus.synthetic.basic-lifecycle.rule.outcome"
def observationRuleId : DefinitionId :=
  definitionId "temporal.nexus.synthetic.basic-lifecycle.rule.observation"

end Mapping

private def field (fieldSpec : ObservationFieldSpec) : ObservationExpression :=
  fieldSpec.expression

private def equalsText (fieldSpec : ObservationFieldSpec) (value : String) : ObservationExpression :=
  .equals (field fieldSpec) (.text value)

private def equalsAny
    (fieldSpec : ObservationFieldSpec) : List String → ObservationExpression
  | [] => .boolean false
  | value :: values =>
      values.foldl (fun condition candidate =>
        .or condition (equalsText fieldSpec candidate)) (equalsText fieldSpec value)

private def rule
    (ruleId output : DefinitionId)
    (outputKind : DefinitionKind)
    (fieldSpec : ObservationFieldSpec)
    (condition : ObservationExpression) : ObservationRule := {
  id := ruleId
  output
  outputKind
  value := .portable (field fieldSpec)
  condition := some (.portable condition)
}

private def mappingDeclaration : ObservationMappingDeclaration := {
  id := Mapping.id
  source
  profile := Profile.id
  rules := [
    rule Mapping.stateRuleId operationStateId .state Profile.stateFieldSpec
      (equalsAny Profile.stateFieldSpec [
        scheduledState.value, startedState.value, canceledState.value, succeededState.value
      ]),
    rule Mapping.startRuleId startActionId .action Profile.actionFieldSpec
      (equalsText Profile.actionFieldSpec startAction.value),
    rule Mapping.cancelRuleId cancelActionId .action Profile.actionFieldSpec
      (equalsText Profile.actionFieldSpec cancelAction.value),
    rule Mapping.succeedRuleId reportSuccessActionId .action Profile.actionFieldSpec
      (equalsText Profile.actionFieldSpec reportSuccessAction.value),
    rule Mapping.outcomeRuleId transitionOutcomeId .outcome Profile.outcomeFieldSpec
      (equalsAny Profile.outcomeFieldSpec [
        startedOutcome.value, canceledOutcome.value, succeededOutcome.value
      ]),
    rule Mapping.observationRuleId lifecycleObservationId .observation Profile.observationFieldSpec
      (equalsAny Profile.observationFieldSpec [
        startedObservation.value, canceledObservation.value, succeededObservation.value
      ])
  ]
  ordering := [
    { before := Mapping.startRuleId, after := Mapping.cancelRuleId },
    { before := Mapping.cancelRuleId, after := Mapping.succeedRuleId },
    { before := Mapping.succeedRuleId, after := Mapping.outcomeRuleId },
    { before := Mapping.outcomeRuleId, after := Mapping.stateRuleId },
    { before := Mapping.stateRuleId, after := Mapping.observationRuleId }
  ]
  closures := [{ kind := Profile.lifecycleKind }]
  dispositions := [
    Profile.stateFieldSpec.disposition .retain,
    Profile.actionFieldSpec.disposition .retain,
    Profile.outcomeFieldSpec.disposition .retain,
    Profile.observationFieldSpec.disposition .retain,
    Profile.rejectedFieldSpec.disposition .reject
  ]
  evidenceBound := { value := 2, unit := .evidenceRecords }
  documentation := "Synthetic scheduled-to-terminal evidence for the ordinary Nexus lifecycle."
}

def checkedPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation (ObservationCheckContext.ofTarget target [Profile.declaration]) mappingDeclaration

private theorem checkedPlanResult_isSome : checkedPlanResult.toOption.isSome = true := by
  native_decide

def checkedPlan : CheckedObservationPlan :=
  checkedObservation
    (ObservationCheckContext.ofTarget target [Profile.declaration])
    mappingDeclaration
    checkedPlanResult_isSome

/-- Typed offline output; no raw evidence is retained in any field. -/
structure OfflineObservation where
  evaluation : ObservationResult
  verdicts : List SemanticPropertyVerdict
  summary : StrictQuerySummary
  deriving BEq, DecidableEq, Repr

/-- The complete typed handoff available to a future adapter that can produce an `EvidenceBundle`. -/
def evaluateSyntheticEvidence (bundle : EvidenceBundle) : OfflineObservation :=
  let evaluation := evaluateEvidence checkedPlan bundle
  let verdict := match evaluation with
    | .accepted trace => evaluateObservationProperty
        Temporal.Feature.Nexus.Operations.AsyncStart.query
        Temporal.Feature.Nexus.Operations.AsyncStart.property trace
    | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic =>
        observationEvaluationFailureVerdict
          Temporal.Feature.Nexus.Operations.AsyncStart.query
          Temporal.Feature.Nexus.Operations.AsyncStart.property diagnostic
  {
    evaluation
    verdicts := [verdict]
    summary := summarizeQueryVerdicts
      Temporal.Feature.Nexus.Operations.AsyncStart.query [verdict]
  }

end Temporal.Feature.Nexus.Observation
