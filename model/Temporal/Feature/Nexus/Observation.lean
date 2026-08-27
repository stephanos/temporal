import Temporal.Feature.Nexus.Operations
import Umpire.Observation

/-!
One synthetic evidence profile for the ordinary Nexus lifecycle. This module performs only the
offline handoff from a finite typed `EvidenceBundle` to qualification and semantic verdicts; it does
not start Temporal, collect live evidence, persist raw records, or promote a result.
-/

namespace Temporal.Feature.Nexus.Observation

open Umpire
open Temporal.Feature.Nexus.Lifecycle

private def declarationId (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Temporal/Feature/Nexus/Observation.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

namespace Profile

def id : DeclarationId := declarationId "temporal.nexus.synthetic.basic-lifecycle.profile"
def lifecycleKind : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.kind.lifecycle"
def stateField : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.field.state"
def actionField : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.field.action"
def outcomeField : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.field.outcome"
def observationField : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.field.observation"
def rejectedField : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.field.raw-detail"

/-- The sole synthetic Temporal profile admitted by this Observation mapping. -/
def declaration : EvidenceProfileDeclaration := {
  id
  source
  kinds := [{
    id := lifecycleKind
    fields := [
      { id := stateField, valueType := .text },
      { id := actionField, valueType := .text },
      { id := outcomeField, valueType := .text },
      { id := observationField, valueType := .text },
      { id := rejectedField, valueType := .text }
    ]
  }]
}

end Profile

namespace Mapping

def id : DeclarationId := declarationId "temporal.nexus.synthetic.basic-lifecycle.mapping"
def stateRuleId : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.rule.state"
def startRuleId : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.rule.action.start"
def cancelRuleId : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.rule.action.cancel"
def succeedRuleId : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.rule.action.succeed"
def outcomeRuleId : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.rule.outcome"
def observationRuleId : DeclarationId :=
  declarationId "temporal.nexus.synthetic.basic-lifecycle.rule.observation"

end Mapping

private def field (fieldId : DeclarationId) : ObservationExpression :=
  .field { kind := Profile.lifecycleKind, field := fieldId }

private def equalsText (fieldId : DeclarationId) (value : String) : ObservationExpression :=
  .equals (field fieldId) (.text value)

private def nonempty (fieldId : DeclarationId) : ObservationExpression :=
  .not (equalsText fieldId "")

private def rule
    (ruleId output : DeclarationId)
    (outputKind : DeclarationKind)
    (fieldId : DeclarationId)
    (condition : ObservationExpression) : ObservationRule := {
  id := ruleId
  output
  outputKind
  value := .portable (field fieldId)
  condition := some (.portable condition)
}

private def mappingDeclaration : ObservationMappingDeclaration := {
  id := Mapping.id
  source
  profile := Profile.id
  rules := [
    rule Mapping.stateRuleId operationStateId .state Profile.stateField
      (.present (field Profile.stateField)),
    rule Mapping.startRuleId startActionId .action Profile.actionField
      (equalsText Profile.actionField startAction.value),
    rule Mapping.cancelRuleId cancelActionId .action Profile.actionField
      (equalsText Profile.actionField cancelAction.value),
    rule Mapping.succeedRuleId reportSuccessActionId .action Profile.actionField
      (equalsText Profile.actionField reportSuccessAction.value),
    rule Mapping.outcomeRuleId transitionOutcomeId .outcome Profile.outcomeField
      (nonempty Profile.outcomeField),
    rule Mapping.observationRuleId lifecycleObservationId .observation Profile.observationField
      (nonempty Profile.observationField)
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
    { field := { kind := Profile.lifecycleKind, field := Profile.stateField },
      disposition := .retain },
    { field := { kind := Profile.lifecycleKind, field := Profile.actionField },
      disposition := .retain },
    { field := { kind := Profile.lifecycleKind, field := Profile.outcomeField },
      disposition := .retain },
    { field := { kind := Profile.lifecycleKind, field := Profile.observationField },
      disposition := .retain },
    { field := { kind := Profile.lifecycleKind, field := Profile.rejectedField },
      disposition := .reject }
  ]
  evidenceBound := { value := 2, unit := .evidenceRecords }
  documentation := "Synthetic scheduled-to-terminal evidence for the ordinary Nexus lifecycle."
}

def checkedPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation (ObservationCheckContext.ofTarget target [Profile.declaration]) mappingDeclaration

private theorem checkedPlanResult_isSome : checkedPlanResult.toOption.isSome = true := by
  native_decide

def checkedPlan : CheckedObservationPlan :=
  checkedPlanResult.toOption.get checkedPlanResult_isSome

/-- Typed offline output; no raw evidence is retained in any field. -/
structure OfflineObservation where
  qualification : QualificationResult
  verdicts : List SemanticPropertyVerdict
  summary : StrictQuerySummary
  deriving BEq, DecidableEq, Repr

/-- The complete typed handoff available to a future adapter that can produce an `EvidenceBundle`. -/
def evaluateSyntheticEvidence (bundle : EvidenceBundle) : OfflineObservation :=
  let qualification := qualifyEvidence checkedPlan bundle
  let verdict := evaluateQualifiedProperty
    Temporal.Feature.Nexus.Operations.AsyncStart.query
    Temporal.Feature.Nexus.Operations.AsyncStart.property
    qualification
  {
    qualification
    verdicts := [verdict]
    summary := summarizeQueryVerdicts
      Temporal.Feature.Nexus.Operations.AsyncStart.query [verdict]
  }

end Temporal.Feature.Nexus.Observation
