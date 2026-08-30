import Temporal.System.Nexus.CallerClosure
import Umpire.Observation

/-!
# Nexus caller-closure Observation mapping

The checked mapping consumes only the four closed fn-19 source kinds and establishes a System
caller-closure trace. It owns no runtime collection, Feature meaning, or Property evaluation.
-/

namespace Temporal.System.Nexus.Observation

open Umpire
open Temporal.System.Nexus.CallerClosure

private def definitionId (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Temporal/System/Nexus/Observation.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

namespace Profile

def id : DefinitionId := definitionId "temporal.system.nexus.caller-closure.profile"
def cleanupKind : DefinitionId := definitionId "umpire.evidence.kind.cleanup"
def controlReceiptKind : DefinitionId := definitionId "umpire.evidence.kind.control-receipt"
def historyKind : DefinitionId := definitionId "umpire.evidence.kind.workflow-history-event"
def participantKind : DefinitionId := definitionId "umpire.evidence.kind.participant-command"

def actionField : DefinitionId := definitionId "umpire.evidence.field.action-definition-id"
def attemptField : DefinitionId := definitionId "umpire.evidence.field.attempt"
def occurrenceField : DefinitionId := definitionId "umpire.evidence.field.occurrence-definition-id"
def statusField : DefinitionId := definitionId "umpire.evidence.field.status"
def eventIdField : DefinitionId := definitionId "umpire.evidence.field.event-id"
def eventTypeField : DefinitionId := definitionId "umpire.evidence.field.event-type"
def operationCorrelationField : DefinitionId :=
  definitionId "umpire.evidence.field.operation-correlation-id"
def runCorrelationField : DefinitionId := definitionId "umpire.evidence.field.run-correlation-id"
def workflowCorrelationField : DefinitionId :=
  definitionId "umpire.evidence.field.workflow-correlation-id"
def cancellationCountField : DefinitionId :=
  definitionId "umpire.evidence.field.cancellation-callback-count"
def endpointIdentityField : DefinitionId := definitionId "umpire.evidence.field.endpoint-identity"
def openHandleCountField : DefinitionId := definitionId "umpire.evidence.field.open-handle-count"
def endpointDigestPolicyId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.digest.endpoint"

def declaration : EvidenceProfileDeclaration := {
  id
  source
  kinds := [
    { id := cleanupKind, fields := [
        { id := openHandleCountField, valueType := .natural },
        { id := statusField, valueType := .text }
      ] },
    { id := controlReceiptKind, fields := [
        { id := actionField, valueType := .text },
        { id := attemptField, valueType := .natural },
        { id := occurrenceField, valueType := .text },
        { id := statusField, valueType := .text }
      ] },
    { id := historyKind, fields := [
        { id := eventIdField, valueType := .natural },
        { id := eventTypeField, valueType := .text },
        { id := operationCorrelationField, valueType := .text },
        { id := runCorrelationField, valueType := .text },
        { id := workflowCorrelationField, valueType := .text }
      ] },
    { id := participantKind, fields := [
        { id := cancellationCountField, valueType := .natural },
        { id := endpointIdentityField, valueType := .text }
      ] }
  ]
}

end Profile

namespace Mapping

def id : DefinitionId := definitionId "temporal.system.nexus.caller-closure.mapping"
def stateRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.rule.state"
def actionRuleId : DefinitionId := definitionId "temporal.system.nexus.caller-closure.rule.action"
def outcomeRuleId : DefinitionId := definitionId "temporal.system.nexus.caller-closure.rule.outcome"
def deliveryRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.rule.delivery"
def cancellationCountRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.rule.cancellation-count"
def ownershipRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.rule.ownership"

end Mapping

private def field (kind field : DefinitionId) : ObservationExpression :=
  .field { kind, field }

private def equalsText
    (kind fieldId : DefinitionId)
    (value : String) : ObservationExpression :=
  .equals (field kind fieldId) (.text value)

private def equalsNatural
    (kind fieldId : DefinitionId)
    (value : Nat) : ObservationExpression :=
  .equals (field kind fieldId) (.natural value)

private def portableCondition (condition : ObservationExpression) :
    Option ObservationExpressionAuthoring :=
  some (.portable condition)

private def constantRule
    (ruleId output : DefinitionId)
    (outputKind : DefinitionKind)
    (value : String)
    (condition : ObservationExpression) : ObservationRule := {
  id := ruleId
  output
  outputKind
  value := .portable (.text value)
  condition := portableCondition condition
}

def mappingDeclaration : ObservationMappingDeclaration := {
  id := Mapping.id
  source
  profile := Profile.id
  digestPolicies := [{
    id := Profile.endpointDigestPolicyId
    name := "synthetic.digest"
    version := 1
  }]
  rules := [
    {
      id := Mapping.stateRuleId
      output := stateId
      outputKind := .state
      value := .portable (field Profile.historyKind Profile.eventTypeField)
      condition := portableCondition (.or
        (equalsText Profile.historyKind Profile.eventTypeField
          "temporal.history.WorkflowExecutionStarted")
        (equalsText Profile.historyKind Profile.eventTypeField
          "temporal.history.WorkflowExecutionCanceled"))
    },
    constantRule Mapping.actionRuleId actionId .action forceCloseAction.value
      (.and
        (equalsText Profile.controlReceiptKind Profile.actionField
          "workflow.action.force-close")
        (.and
          (equalsNatural Profile.controlReceiptKind Profile.attemptField 1)
          (equalsText Profile.controlReceiptKind Profile.statusField "accepted"))),
    constantRule Mapping.outcomeRuleId outcomeId .outcome cancellationUpgradedOutcome.value
      (equalsText Profile.historyKind Profile.eventTypeField
        "temporal.history.WorkflowExecutionCanceled"),
    constantRule Mapping.deliveryRuleId deliveryObservationId .observation
      deliveryObservation.value
      (equalsText Profile.historyKind Profile.eventTypeField
        "temporal.history.WorkflowExecutionCanceled"),
    {
      id := Mapping.cancellationCountRuleId
      output := cancellationCountObservationId
      outputKind := .observation
      value := .portable (.normalize { name := "natural.render", version := 1 }
        (field Profile.participantKind Profile.cancellationCountField))
      condition := portableCondition
        (equalsNatural Profile.participantKind Profile.cancellationCountField 1)
    },
    constantRule Mapping.ownershipRuleId ownershipObservationId .observation
      ownershipObservation.value
      (equalsText Profile.historyKind Profile.eventTypeField
        "temporal.history.WorkflowExecutionCanceled")
  ]
  ordering := [
    { before := Mapping.actionRuleId, after := Mapping.outcomeRuleId },
    { before := Mapping.outcomeRuleId, after := Mapping.stateRuleId },
    { before := Mapping.stateRuleId, after := Mapping.deliveryRuleId },
    { before := Mapping.deliveryRuleId, after := Mapping.cancellationCountRuleId },
    { before := Mapping.cancellationCountRuleId, after := Mapping.ownershipRuleId }
  ]
  closures := [
    { kind := Profile.cleanupKind },
    { kind := Profile.controlReceiptKind },
    { kind := Profile.historyKind },
    { kind := Profile.participantKind }
  ]
  dispositions := [
    { field := { kind := Profile.cleanupKind, field := Profile.openHandleCountField },
      disposition := .retain },
    { field := { kind := Profile.cleanupKind, field := Profile.statusField },
      disposition := .retain },
    { field := { kind := Profile.controlReceiptKind, field := Profile.actionField },
      disposition := .retain },
    { field := { kind := Profile.controlReceiptKind, field := Profile.attemptField },
      disposition := .retain },
    { field := { kind := Profile.controlReceiptKind, field := Profile.occurrenceField },
      disposition := .retain },
    { field := { kind := Profile.controlReceiptKind, field := Profile.statusField },
      disposition := .retain },
    { field := { kind := Profile.historyKind, field := Profile.eventIdField },
      disposition := .retain },
    { field := { kind := Profile.historyKind, field := Profile.eventTypeField },
      disposition := .retain },
    { field := { kind := Profile.historyKind, field := Profile.operationCorrelationField },
      disposition := .retain },
    { field := { kind := Profile.historyKind, field := Profile.runCorrelationField },
      disposition := .retain },
    { field := { kind := Profile.historyKind, field := Profile.workflowCorrelationField },
      disposition := .retain },
    { field := { kind := Profile.participantKind, field := Profile.cancellationCountField },
      disposition := .retain },
    { field := { kind := Profile.participantKind, field := Profile.endpointIdentityField },
      disposition := .hash (some Profile.endpointDigestPolicyId) }
  ]
  evidenceBound := { value := 4096, unit := .evidenceRecords }
  documentation := "Closed four-source Nexus caller-closure evidence to one System trace."
}

def checkedPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation (ObservationCheckContext.ofTarget target [Profile.declaration])
    mappingDeclaration

private theorem checkedPlanResult_isSome : checkedPlanResult.toOption.isSome = true := by
  native_decide

def checkedPlan : CheckedObservationPlan :=
  checkedPlanResult.toOption.get checkedPlanResult_isSome

end Temporal.System.Nexus.Observation
