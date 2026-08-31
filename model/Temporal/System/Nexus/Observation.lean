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
def commandKindField : DefinitionId := definitionId "umpire.evidence.field.command-kind"
def endpointIdentityField : DefinitionId := definitionId "umpire.evidence.field.endpoint-identity"
def errorCodeField : DefinitionId := definitionId "umpire.evidence.field.error-code"
def namespaceIdentityField : DefinitionId :=
  definitionId "umpire.evidence.field.namespace-identity"
def openHandleCountField : DefinitionId := definitionId "umpire.evidence.field.open-handle-count"
def taskQueueIdentityField : DefinitionId :=
  definitionId "umpire.evidence.field.task-queue-identity"
def endpointDigestPolicyId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.digest.endpoint"

def declaration : EvidenceProfileDeclaration := {
  id
  source
  kinds := [
    { id := cleanupKind, fields := [
        { id := commandKindField, valueType := .text },
        { id := endpointIdentityField, valueType := .text },
        { id := errorCodeField, valueType := .text },
        { id := namespaceIdentityField, valueType := .text },
        { id := openHandleCountField, valueType := .natural },
        { id := operationCorrelationField, valueType := .text },
        { id := runCorrelationField, valueType := .text },
        { id := statusField, valueType := .text },
        { id := taskQueueIdentityField, valueType := .text },
        { id := workflowCorrelationField, valueType := .text }
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
        { id := commandKindField, valueType := .text },
        { id := endpointIdentityField, valueType := .text },
        { id := errorCodeField, valueType := .text },
        { id := namespaceIdentityField, valueType := .text },
        { id := operationCorrelationField, valueType := .text },
        { id := runCorrelationField, valueType := .text },
        { id := statusField, valueType := .text },
        { id := taskQueueIdentityField, valueType := .text },
        { id := workflowCorrelationField, valueType := .text }
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
    { field := { kind := Profile.cleanupKind, field := Profile.commandKindField },
      disposition := .retain },
    { field := { kind := Profile.cleanupKind, field := Profile.endpointIdentityField },
      disposition := .hash (some Profile.endpointDigestPolicyId) },
    { field := { kind := Profile.cleanupKind, field := Profile.errorCodeField },
      disposition := .retain },
    { field := { kind := Profile.cleanupKind, field := Profile.namespaceIdentityField },
      disposition := .hash (some Profile.endpointDigestPolicyId) },
    { field := { kind := Profile.cleanupKind, field := Profile.openHandleCountField },
      disposition := .retain },
    { field := { kind := Profile.cleanupKind, field := Profile.operationCorrelationField },
      disposition := .retain },
    { field := { kind := Profile.cleanupKind, field := Profile.runCorrelationField },
      disposition := .retain },
    { field := { kind := Profile.cleanupKind, field := Profile.statusField },
      disposition := .retain },
    { field := { kind := Profile.cleanupKind, field := Profile.taskQueueIdentityField },
      disposition := .hash (some Profile.endpointDigestPolicyId) },
    { field := { kind := Profile.cleanupKind, field := Profile.workflowCorrelationField },
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
    { field := { kind := Profile.participantKind, field := Profile.commandKindField },
      disposition := .retain },
    { field := { kind := Profile.participantKind, field := Profile.endpointIdentityField },
      disposition := .hash (some Profile.endpointDigestPolicyId) },
    { field := { kind := Profile.participantKind, field := Profile.errorCodeField },
      disposition := .retain },
    { field := { kind := Profile.participantKind, field := Profile.namespaceIdentityField },
      disposition := .hash (some Profile.endpointDigestPolicyId) },
    { field := { kind := Profile.participantKind, field := Profile.operationCorrelationField },
      disposition := .retain },
    { field := { kind := Profile.participantKind, field := Profile.runCorrelationField },
      disposition := .retain },
    { field := { kind := Profile.participantKind, field := Profile.statusField },
      disposition := .retain },
    { field := { kind := Profile.participantKind, field := Profile.taskQueueIdentityField },
      disposition := .hash (some Profile.endpointDigestPolicyId) },
    { field := { kind := Profile.participantKind, field := Profile.workflowCorrelationField },
      disposition := .retain }
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

namespace DuplicateDelivery

/-!
The negative control has a distinct checked profile and mapping identity, but it reuses the same
Observation compiler and System meanings as the ordinary caller-closure path. Mechanical delivery
stays one; only the exact labeled one-plus-one evidence relation emits semantic count two.
-/

def mechanicalCallbackCount : Nat := 1
def syntheticContributionCount : Nat := 1
def semanticCancellationCount : Nat := 2

def faultDefinitionId : DefinitionId :=
  definitionId "temporal.nexus.caller-closure.fault.duplicate-delivery-observation"
def cancellationCapabilityId : DefinitionId :=
  definitionId "nexus.capability.cancellation"
def forceCloseOccurrenceId : DefinitionId :=
  definitionId "workflow-nexus.occurrence.force-close"
def faultReceiptId : DefinitionId :=
  definitionId "temporal.nexus.caller-closure.fault-receipt.duplicate-delivery-observation"
def injectedMarker : String :=
  "temporal.nexus.caller-closure.marker.injected-duplicate-delivery-observation"

namespace Profile

def id : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.duplicate-delivery.profile"

def cleanupKind := Temporal.System.Nexus.Observation.Profile.cleanupKind
def controlReceiptKind := Temporal.System.Nexus.Observation.Profile.controlReceiptKind
def historyKind := Temporal.System.Nexus.Observation.Profile.historyKind
def participantKind := Temporal.System.Nexus.Observation.Profile.participantKind

def actionField := Temporal.System.Nexus.Observation.Profile.actionField
def attemptField := Temporal.System.Nexus.Observation.Profile.attemptField
def occurrenceField := Temporal.System.Nexus.Observation.Profile.occurrenceField
def statusField := Temporal.System.Nexus.Observation.Profile.statusField
def eventIdField := Temporal.System.Nexus.Observation.Profile.eventIdField
def eventTypeField := Temporal.System.Nexus.Observation.Profile.eventTypeField
def operationCorrelationField :=
  Temporal.System.Nexus.Observation.Profile.operationCorrelationField
def runCorrelationField := Temporal.System.Nexus.Observation.Profile.runCorrelationField
def workflowCorrelationField :=
  Temporal.System.Nexus.Observation.Profile.workflowCorrelationField
def cancellationCountField :=
  Temporal.System.Nexus.Observation.Profile.cancellationCountField
def endpointDigestPolicyId :=
  Temporal.System.Nexus.Observation.Profile.endpointDigestPolicyId

def faultDefinitionField : DefinitionId :=
  definitionId "umpire.evidence.field.fault-definition-id"
def faultReceiptField : DefinitionId :=
  definitionId "umpire.evidence.field.fault-receipt-definition-id"
def capabilityDefinitionField : DefinitionId :=
  definitionId "umpire.evidence.field.capability-definition-id"
def syntheticContributionCountField : DefinitionId :=
  definitionId "umpire.evidence.field.synthetic-contribution-count"
def syntheticMarkerField : DefinitionId :=
  definitionId "umpire.evidence.field.synthetic-contribution-marker"
def cancellationRequestedCountField : DefinitionId :=
  definitionId "umpire.evidence.field.cancellation-requested-count"
def cancellationCompletedCountField : DefinitionId :=
  definitionId "umpire.evidence.field.cancellation-completed-count"

private def extendKind (kind : EvidenceKindDeclaration) : EvidenceKindDeclaration :=
  if kind.id == controlReceiptKind then {
    kind with fields := kind.fields ++ [
      { id := faultDefinitionField, valueType := .text },
      { id := faultReceiptField, valueType := .text },
      { id := capabilityDefinitionField, valueType := .text },
      { id := operationCorrelationField, valueType := .text }
    ]
  } else if kind.id == participantKind then {
    kind with fields := kind.fields ++ [
      { id := faultDefinitionField, valueType := .text },
      { id := faultReceiptField, valueType := .text },
      { id := capabilityDefinitionField, valueType := .text },
      { id := syntheticContributionCountField, valueType := .natural },
      { id := syntheticMarkerField, valueType := .text },
      { id := cancellationRequestedCountField, valueType := .natural },
      { id := cancellationCompletedCountField, valueType := .natural }
    ]
  } else kind

def declaration : EvidenceProfileDeclaration := {
  Temporal.System.Nexus.Observation.Profile.declaration with
  id
  kinds := Temporal.System.Nexus.Observation.Profile.declaration.kinds.map extendKind
}

end Profile

namespace Mapping

def id : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.duplicate-delivery.mapping"
def stateRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.duplicate-delivery.rule.state"
def actionRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.duplicate-delivery.rule.action"
def outcomeRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.duplicate-delivery.rule.outcome"
def deliveryRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.duplicate-delivery.rule.delivery"
def cancellationCountRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.duplicate-delivery.rule.cancellation-count"
def ownershipRuleId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.duplicate-delivery.rule.ownership"

end Mapping

private def profileField (kind fieldId : DefinitionId) : ObservationExpression :=
  .field { kind, field := fieldId }

private def profileEqualsText
    (kind fieldId : DefinitionId)
    (value : String) : ObservationExpression :=
  .equals (profileField kind fieldId) (.text value)

private def profileEqualsNatural
    (kind fieldId : DefinitionId)
    (value : Nat) : ObservationExpression :=
  .equals (profileField kind fieldId) (.natural value)

private def allConditions : List ObservationExpression → ObservationExpression
  | [] => .boolean true
  | [condition] => condition
  | condition :: rest => .and condition (allConditions rest)

private def faultConstantRule
    (ruleId output : DefinitionId)
    (outputKind : DefinitionKind)
    (value : String)
    (condition : ObservationExpression) : ObservationRule := {
  id := ruleId
  output
  outputKind
  value := .portable (.text value)
  condition := some (.portable condition)
}

def stateRule : ObservationRule := {
  id := Mapping.stateRuleId
  output := stateId
  outputKind := .state
  value := .portable (profileField Profile.historyKind Profile.eventTypeField)
  condition := some (.portable (.or
    (profileEqualsText Profile.historyKind Profile.eventTypeField
      "temporal.history.WorkflowExecutionStarted")
    (profileEqualsText Profile.historyKind Profile.eventTypeField
      "temporal.history.WorkflowExecutionCanceled")))
}

def actionRule : ObservationRule :=
  faultConstantRule Mapping.actionRuleId actionId .action forceCloseAction.value <|
    allConditions [
      profileEqualsText Profile.controlReceiptKind Profile.actionField
        "workflow.action.force-close",
      profileEqualsNatural Profile.controlReceiptKind Profile.attemptField 1,
      profileEqualsText Profile.controlReceiptKind Profile.occurrenceField
        forceCloseOccurrenceId.value,
      profileEqualsText Profile.controlReceiptKind Profile.statusField "accepted",
      profileEqualsText Profile.controlReceiptKind Profile.faultDefinitionField
        faultDefinitionId.value,
      profileEqualsText Profile.controlReceiptKind Profile.faultReceiptField faultReceiptId.value,
      profileEqualsText Profile.controlReceiptKind Profile.capabilityDefinitionField
        cancellationCapabilityId.value
    ]

def outcomeRule : ObservationRule :=
  faultConstantRule Mapping.outcomeRuleId outcomeId .outcome
    cancellationUpgradedOutcome.value
    (profileEqualsText Profile.historyKind Profile.eventTypeField
      "temporal.history.WorkflowExecutionCanceled")

def deliveryRule : ObservationRule :=
  faultConstantRule Mapping.deliveryRuleId deliveryObservationId .observation
    deliveryObservation.value
    (profileEqualsText Profile.historyKind Profile.eventTypeField
      "temporal.history.WorkflowExecutionCanceled")

def semanticCountRule : ObservationRule := {
  id := Mapping.cancellationCountRuleId
  output := cancellationCountObservationId
  outputKind := .observation
  value := .portable (.text (toString semanticCancellationCount))
  condition := some (.portable <| allConditions [
    profileEqualsNatural Profile.participantKind Profile.cancellationCountField
      mechanicalCallbackCount,
    profileEqualsNatural Profile.participantKind Profile.syntheticContributionCountField
      syntheticContributionCount,
    profileEqualsText Profile.participantKind Profile.syntheticMarkerField injectedMarker,
    profileEqualsNatural Profile.participantKind Profile.cancellationRequestedCountField 1,
    profileEqualsNatural Profile.participantKind Profile.cancellationCompletedCountField 1,
    profileEqualsText Profile.participantKind Profile.faultDefinitionField faultDefinitionId.value,
    profileEqualsText Profile.participantKind Profile.faultReceiptField faultReceiptId.value,
    profileEqualsText Profile.participantKind Profile.capabilityDefinitionField
      cancellationCapabilityId.value,
    .present (profileField Profile.participantKind Profile.operationCorrelationField),
    .present (profileField Profile.participantKind Profile.runCorrelationField),
    .present (profileField Profile.participantKind Profile.workflowCorrelationField)
  ])
}

def ownershipRule : ObservationRule :=
  faultConstantRule Mapping.ownershipRuleId ownershipObservationId .observation
    ownershipObservation.value
    (profileEqualsText Profile.historyKind Profile.eventTypeField
      "temporal.history.WorkflowExecutionCanceled")

private def additionalDispositions : List FieldDispositionDeclaration := [
  {
    field := { kind := Profile.controlReceiptKind, field := Profile.faultDefinitionField }
    disposition := .retain
  },
  {
    field := { kind := Profile.controlReceiptKind, field := Profile.faultReceiptField }
    disposition := .retain
  },
  {
    field := { kind := Profile.controlReceiptKind, field := Profile.capabilityDefinitionField }
    disposition := .retain
  },
  {
    field := { kind := Profile.controlReceiptKind, field := Profile.operationCorrelationField }
    disposition := .retain
  },
  {
    field := {
      kind := Profile.participantKind
      field := Profile.syntheticContributionCountField
    }
    disposition := .retain
  },
  {
    field := { kind := Profile.participantKind, field := Profile.syntheticMarkerField }
    disposition := .retain
  },
  {
    field := { kind := Profile.participantKind, field := Profile.faultDefinitionField }
    disposition := .retain
  },
  {
    field := { kind := Profile.participantKind, field := Profile.faultReceiptField }
    disposition := .retain
  },
  {
    field := { kind := Profile.participantKind, field := Profile.capabilityDefinitionField }
    disposition := .retain
  },
  {
    field := {
      kind := Profile.participantKind
      field := Profile.cancellationRequestedCountField
    }
    disposition := .retain
  },
  {
    field := {
      kind := Profile.participantKind
      field := Profile.cancellationCompletedCountField
    }
    disposition := .retain
  }
]

def mappingDeclaration : ObservationMappingDeclaration := {
  id := Mapping.id
  source
  profile := Profile.id
  digestPolicies := Temporal.System.Nexus.Observation.mappingDeclaration.digestPolicies
  rules := [stateRule, actionRule, outcomeRule, deliveryRule, semanticCountRule, ownershipRule]
  ordering := [
    { before := Mapping.actionRuleId, after := Mapping.outcomeRuleId },
    { before := Mapping.outcomeRuleId, after := Mapping.stateRuleId },
    { before := Mapping.stateRuleId, after := Mapping.deliveryRuleId },
    { before := Mapping.deliveryRuleId, after := Mapping.cancellationCountRuleId },
    { before := Mapping.cancellationCountRuleId, after := Mapping.ownershipRuleId }
  ]
  closures := Temporal.System.Nexus.Observation.mappingDeclaration.closures
  dispositions := Temporal.System.Nexus.Observation.mappingDeclaration.dispositions ++
    additionalDispositions
  evidenceBound := Temporal.System.Nexus.Observation.mappingDeclaration.evidenceBound
  documentation :=
    "Closed duplicate-delivery evidence derives count two from callback one plus contribution one."
}

private def canonicalPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation (ObservationCheckContext.ofTarget target [Profile.declaration])
    mappingDeclaration

private theorem canonicalPlanResult_isSome : canonicalPlanResult.toOption.isSome = true := by
  native_decide

def checkedPlan : CheckedObservationPlan :=
  canonicalPlanResult.toOption.get canonicalPlanResult_isSome

private def qualificationDiagnostic
    (kind : ObservationFailureKind)
    (relatedDefinitionIds : List DefinitionId := []) : ObservationDiagnostic := {
  kind
  planId := checkedPlan.id
  relatedDefinitionIds
}

private def evidenceText?
    (record : SyntheticEvidenceRecord)
    (field : DefinitionId) : Option String := do
  let fieldValue ← record.fields.find? fun candidate => candidate.field == field
  match fieldValue.value with
  | .text value => some value
  | _ => none

private def evidenceNatural?
    (record : SyntheticEvidenceRecord)
    (field : DefinitionId) : Option Nat := do
  let fieldValue ← record.fields.find? fun candidate => candidate.field == field
  match fieldValue.value with
  | .natural value => some value
  | _ => none

private def requiredText
    (record : SyntheticEvidenceRecord)
    (field : DefinitionId) : Except ObservationDiagnostic String :=
  match evidenceText? record field with
  | some value => pure value
  | none => throw (qualificationDiagnostic .unresolvedBinding [record.id, field])

private def requireText
    (record : SyntheticEvidenceRecord)
    (field : DefinitionId)
    (expected : String) : Except ObservationDiagnostic Unit := do
  if (← requiredText record field) != expected then
    throw (qualificationDiagnostic .contradictoryFact [record.id, field])

private def requireNatural
    (record : SyntheticEvidenceRecord)
    (field : DefinitionId)
    (expected : Nat) : Except ObservationDiagnostic Unit :=
  match evidenceNatural? record field with
  | some value =>
      if value == expected then pure ()
      else throw (qualificationDiagnostic .contradictoryFact [record.id, field])
  | none => throw (qualificationDiagnostic .unresolvedBinding [record.id, field])

private def exactRecordOfKind
    (bundle : EvidenceBundle)
    (kind : DefinitionId) : Except ObservationDiagnostic SyntheticEvidenceRecord :=
  match bundle.records.filter fun record => record.kind == kind with
  | [record] => pure record
  | [] => throw (qualificationDiagnostic .unresolvedBinding [kind])
  | records => throw (qualificationDiagnostic .contradictoryFact
      (kind :: records.map SyntheticEvidenceRecord.id))

private def historyRecordLe
    (left right : SyntheticEvidenceRecord) : Bool :=
  match left.origin, right.origin with
  | some leftOrigin, some rightOrigin => leftOrigin.ordinal ≤ rightOrigin.ordinal
  | none, none => left.sequence ≤ right.sequence
  | none, some _ => true
  | some _, none => false

private def expectedHistoryEventTypes : List String := [
  "temporal.history.WorkflowExecutionStarted",
  "temporal.history.NexusOperationCancelRequested",
  "temporal.history.NexusOperationCancelRequestCompleted",
  "temporal.history.WorkflowExecutionCanceled"
]

private def exactHistoryEvent
    (records : List SyntheticEvidenceRecord)
    (eventType : String) : Except ObservationDiagnostic SyntheticEvidenceRecord :=
  match records.filter fun record =>
      evidenceText? record Profile.eventTypeField == some eventType with
  | [record] => pure record
  | [] => throw (qualificationDiagnostic .unresolvedBinding
      [Profile.historyKind, Profile.eventTypeField])
  | multiple => throw (qualificationDiagnostic .contradictoryFact
      (multiple.map SyntheticEvidenceRecord.id))

private def requireMatchingText
    (records : List SyntheticEvidenceRecord)
    (field : DefinitionId)
    (expected : String) : Except ObservationDiagnostic Unit := do
  if expected == "" then
    throw (qualificationDiagnostic .unresolvedBinding [field])
  for record in records do
    if (← requiredText record field) != expected then
      throw (qualificationDiagnostic .contradictoryBinding [record.id, field])

private def checkDuplicateDeliveryEvidence
    (bundle : EvidenceBundle) : Except ObservationDiagnostic Unit := do
  let control ← exactRecordOfKind bundle Profile.controlReceiptKind
  let participant ← exactRecordOfKind bundle Profile.participantKind
  let history := (bundle.records.filter fun record => record.kind == Profile.historyKind)
    |>.mergeSort historyRecordLe
  let started ← exactHistoryEvent history expectedHistoryEventTypes[0]
  let requested ← exactHistoryEvent history expectedHistoryEventTypes[1]
  let completed ← exactHistoryEvent history expectedHistoryEventTypes[2]
  let canceled ← exactHistoryEvent history expectedHistoryEventTypes[3]
  if history.length != expectedHistoryEventTypes.length then
    throw (qualificationDiagnostic .contradictoryFact
      (history.map SyntheticEvidenceRecord.id))
  let actualEventTypes ← history.mapM fun record => requiredText record Profile.eventTypeField
  if actualEventTypes != expectedHistoryEventTypes then
    throw (qualificationDiagnostic .contradictoryOrder
      (history.map SyntheticEvidenceRecord.id))
  if !started.causalParents.isEmpty || requested.causalParents != [started.id] ||
      completed.causalParents != [requested.id] || canceled.causalParents != [completed.id] ||
      !participant.causalParents.contains completed.id || participant.faultTarget != some completed.id then
    throw (qualificationDiagnostic .contradictoryOrder [
      started.id, requested.id, completed.id, canceled.id, participant.id
    ])
  requireText control Profile.faultDefinitionField faultDefinitionId.value
  requireText control Profile.faultReceiptField faultReceiptId.value
  requireText control Profile.capabilityDefinitionField cancellationCapabilityId.value
  requireText participant Profile.faultDefinitionField faultDefinitionId.value
  requireText participant Profile.faultReceiptField faultReceiptId.value
  requireText participant Profile.capabilityDefinitionField cancellationCapabilityId.value
  requireText participant Profile.syntheticMarkerField injectedMarker
  requireNatural participant Profile.cancellationCountField mechanicalCallbackCount
  requireNatural participant Profile.syntheticContributionCountField syntheticContributionCount
  requireNatural participant Profile.cancellationRequestedCountField 1
  requireNatural participant Profile.cancellationCompletedCountField 1
  let operationCorrelation ← requiredText participant Profile.operationCorrelationField
  let runCorrelation ← requiredText participant Profile.runCorrelationField
  let workflowCorrelation ← requiredText participant Profile.workflowCorrelationField
  requireMatchingText (control :: history) Profile.operationCorrelationField operationCorrelation
  requireMatchingText history Profile.runCorrelationField runCorrelation
  requireMatchingText history Profile.workflowCorrelationField workflowCorrelation

private def resultOfQualificationDiagnostic
    (diagnostic : ObservationDiagnostic) : ObservationResult :=
  match diagnostic.status with
  | .unknown => .unknown diagnostic
  | .conflict => .conflict diagnostic
  | .unsupported => .unsupported diagnostic
  | .accepted => .unknown diagnostic

/-- Qualify the accepted generic Observation result against the exact lifecycle, causal receipt,
and shared-correlation contract of the duplicate-delivery negative control. -/
def qualifyDuplicateDeliveryObservation (bundle : EvidenceBundle) : ObservationResult :=
  match evaluateEvidence checkedPlan bundle with
  | .accepted trace =>
      match checkDuplicateDeliveryEvidence bundle with
      | .ok _ => .accepted trace
      | .error diagnostic => resultOfQualificationDiagnostic diagnostic
  | .unknown diagnostic => .unknown diagnostic
  | .conflict diagnostic => .conflict diagnostic
  | .unsupported diagnostic => .unsupported diagnostic

inductive DuplicateDeliveryCheckErrorKind where
  | observation
  | contractDrift
  deriving BEq, DecidableEq, Repr

structure DuplicateDeliveryCheckError where
  kind : DuplicateDeliveryCheckErrorKind
  observationError : Option ObservationError := none
  deriving Repr

/-- Compile a candidate with the reusable Observation checker, then require the exact closed
negative-control contract. Canonically equivalent declaration reordering remains accepted. -/
def checkDuplicateDeliveryObservation
    (profile : EvidenceProfileDeclaration)
    (declaration : ObservationMappingDeclaration) :
    Except DuplicateDeliveryCheckError CheckedObservationPlan := do
  let candidate ← (checkObservation (ObservationCheckContext.ofTarget target [profile]) declaration)
    |>.mapError fun observationError => {
      kind := .observation
      observationError := some observationError
    }
  if candidate.behaviorFingerprint != checkedPlan.behaviorFingerprint then
    throw { kind := .contractDrift }
  pure candidate

def profileBehaviorFingerprint : BehaviorFingerprint :=
  behaviorFingerprintOf (reprStr checkedPlan.profile)

def profileVersion : Nat := 1

def mappingBehaviorFingerprint : BehaviorFingerprint := checkedPlan.behaviorFingerprint

def mappingVersion : Nat := checkedPlan.version

def qualificationBehaviorFingerprint : BehaviorFingerprint := behaviorFingerprintOf <| reprStr (
  expectedHistoryEventTypes,
  Profile.operationCorrelationField,
  Profile.runCorrelationField,
  Profile.workflowCorrelationField,
  Profile.faultDefinitionField,
  Profile.faultReceiptField,
  Profile.capabilityDefinitionField,
  faultDefinitionId,
  faultReceiptId,
  cancellationCapabilityId,
  mechanicalCallbackCount,
  syntheticContributionCount
)

def programId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.duplicate-delivery.observation-program"

def programVersion : Nat := 1

def programBehaviorFingerprint : BehaviorFingerprint := behaviorFingerprintOf <|
  programId.value ++ "/v" ++ toString programVersion ++ "/" ++
    Profile.id.value ++ "/" ++ profileBehaviorFingerprint.render ++ "/" ++
    Mapping.id.value ++ "/" ++ mappingBehaviorFingerprint.render ++ "/" ++
    qualificationBehaviorFingerprint.render

end DuplicateDelivery

end Temporal.System.Nexus.Observation
