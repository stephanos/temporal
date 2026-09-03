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

private def fieldSpec
    (kind : DefinitionId)
    (field : String)
    (valueType : ObservationValueType) : ObservationFieldSpec := {
  kind
  field := definitionId field
  valueType
}

def cleanupCommandKindFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.command-kind" .text
def cleanupEndpointIdentityFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.endpoint-identity" .text
def cleanupErrorCodeFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.error-code" .text
def cleanupNamespaceIdentityFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.namespace-identity" .text
def cleanupOpenHandleCountFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.open-handle-count" .natural
def cleanupOperationCorrelationFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.operation-correlation-id" .text
def cleanupRunCorrelationFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.run-correlation-id" .text
def cleanupStatusFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.status" .text
def cleanupTaskQueueIdentityFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.task-queue-identity" .text
def cleanupWorkflowCorrelationFieldSpec : ObservationFieldSpec :=
  fieldSpec cleanupKind "umpire.evidence.field.workflow-correlation-id" .text

def commandKindField : DefinitionId := cleanupCommandKindFieldSpec.field
def endpointIdentityField : DefinitionId := cleanupEndpointIdentityFieldSpec.field
def errorCodeField : DefinitionId := cleanupErrorCodeFieldSpec.field
def namespaceIdentityField : DefinitionId := cleanupNamespaceIdentityFieldSpec.field
def openHandleCountField : DefinitionId := cleanupOpenHandleCountFieldSpec.field
def operationCorrelationField : DefinitionId := cleanupOperationCorrelationFieldSpec.field
def runCorrelationField : DefinitionId := cleanupRunCorrelationFieldSpec.field
def statusField : DefinitionId := cleanupStatusFieldSpec.field
def taskQueueIdentityField : DefinitionId := cleanupTaskQueueIdentityFieldSpec.field
def workflowCorrelationField : DefinitionId := cleanupWorkflowCorrelationFieldSpec.field

def controlReceiptActionFieldSpec : ObservationFieldSpec :=
  fieldSpec controlReceiptKind "umpire.evidence.field.action-definition-id" .text
def controlReceiptAttemptFieldSpec : ObservationFieldSpec :=
  fieldSpec controlReceiptKind "umpire.evidence.field.attempt" .natural
def controlReceiptOccurrenceFieldSpec : ObservationFieldSpec :=
  fieldSpec controlReceiptKind "umpire.evidence.field.occurrence-definition-id" .text
def controlReceiptStatusFieldSpec : ObservationFieldSpec := {
  cleanupStatusFieldSpec with kind := controlReceiptKind
}

def actionField : DefinitionId := controlReceiptActionFieldSpec.field
def attemptField : DefinitionId := controlReceiptAttemptFieldSpec.field
def occurrenceField : DefinitionId := controlReceiptOccurrenceFieldSpec.field

def historyEventIdFieldSpec : ObservationFieldSpec :=
  fieldSpec historyKind "umpire.evidence.field.event-id" .natural
def historyEventTypeFieldSpec : ObservationFieldSpec :=
  fieldSpec historyKind "umpire.evidence.field.event-type" .text
def historyOperationCorrelationFieldSpec : ObservationFieldSpec := {
  cleanupOperationCorrelationFieldSpec with kind := historyKind
}
def historyRunCorrelationFieldSpec : ObservationFieldSpec := {
  cleanupRunCorrelationFieldSpec with kind := historyKind
}
def historyWorkflowCorrelationFieldSpec : ObservationFieldSpec := {
  cleanupWorkflowCorrelationFieldSpec with kind := historyKind
}

def eventIdField : DefinitionId := historyEventIdFieldSpec.field
def eventTypeField : DefinitionId := historyEventTypeFieldSpec.field

def participantCancellationCountFieldSpec : ObservationFieldSpec :=
  fieldSpec participantKind "umpire.evidence.field.cancellation-callback-count" .natural
def participantCommandKindFieldSpec : ObservationFieldSpec := {
  cleanupCommandKindFieldSpec with kind := participantKind
}
def participantEndpointIdentityFieldSpec : ObservationFieldSpec := {
  cleanupEndpointIdentityFieldSpec with kind := participantKind
}
def participantErrorCodeFieldSpec : ObservationFieldSpec := {
  cleanupErrorCodeFieldSpec with kind := participantKind
}
def participantNamespaceIdentityFieldSpec : ObservationFieldSpec := {
  cleanupNamespaceIdentityFieldSpec with kind := participantKind
}
def participantOperationCorrelationFieldSpec : ObservationFieldSpec := {
  cleanupOperationCorrelationFieldSpec with kind := participantKind
}
def participantRunCorrelationFieldSpec : ObservationFieldSpec := {
  cleanupRunCorrelationFieldSpec with kind := participantKind
}
def participantStatusFieldSpec : ObservationFieldSpec := {
  cleanupStatusFieldSpec with kind := participantKind
}
def participantTaskQueueIdentityFieldSpec : ObservationFieldSpec := {
  cleanupTaskQueueIdentityFieldSpec with kind := participantKind
}
def participantWorkflowCorrelationFieldSpec : ObservationFieldSpec := {
  cleanupWorkflowCorrelationFieldSpec with kind := participantKind
}

def cancellationCountField : DefinitionId := participantCancellationCountFieldSpec.field
def endpointDigestPolicyId : DefinitionId :=
  definitionId "temporal.system.nexus.caller-closure.digest.endpoint"

def declaration : EvidenceProfileDeclaration := {
  id
  source
  kinds := [
    { id := cleanupKind, fields := [
        cleanupCommandKindFieldSpec.declaration,
        cleanupEndpointIdentityFieldSpec.declaration,
        cleanupErrorCodeFieldSpec.declaration,
        cleanupNamespaceIdentityFieldSpec.declaration,
        cleanupOpenHandleCountFieldSpec.declaration,
        cleanupOperationCorrelationFieldSpec.declaration,
        cleanupRunCorrelationFieldSpec.declaration,
        cleanupStatusFieldSpec.declaration,
        cleanupTaskQueueIdentityFieldSpec.declaration,
        cleanupWorkflowCorrelationFieldSpec.declaration
      ] },
    { id := controlReceiptKind, fields := [
        controlReceiptActionFieldSpec.declaration,
        controlReceiptAttemptFieldSpec.declaration,
        controlReceiptOccurrenceFieldSpec.declaration,
        controlReceiptStatusFieldSpec.declaration
      ] },
    { id := historyKind, fields := [
        historyEventIdFieldSpec.declaration,
        historyEventTypeFieldSpec.declaration,
        historyOperationCorrelationFieldSpec.declaration,
        historyRunCorrelationFieldSpec.declaration,
        historyWorkflowCorrelationFieldSpec.declaration
      ] },
    { id := participantKind, fields := [
        participantCancellationCountFieldSpec.declaration,
        participantCommandKindFieldSpec.declaration,
        participantEndpointIdentityFieldSpec.declaration,
        participantErrorCodeFieldSpec.declaration,
        participantNamespaceIdentityFieldSpec.declaration,
        participantOperationCorrelationFieldSpec.declaration,
        participantRunCorrelationFieldSpec.declaration,
        participantStatusFieldSpec.declaration,
        participantTaskQueueIdentityFieldSpec.declaration,
        participantWorkflowCorrelationFieldSpec.declaration
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

private def field (fieldSpec : ObservationFieldSpec) : ObservationExpression :=
  fieldSpec.expression

private def equalsText
    (fieldSpec : ObservationFieldSpec)
    (value : String) : ObservationExpression :=
  .equals (field fieldSpec) (.text value)

private def equalsNatural
    (fieldSpec : ObservationFieldSpec)
    (value : Nat) : ObservationExpression :=
  .equals (field fieldSpec) (.natural value)

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
      value := .portable (field Profile.historyEventTypeFieldSpec)
      condition := portableCondition (.or
        (equalsText Profile.historyEventTypeFieldSpec
          "temporal.history.WorkflowExecutionStarted")
        (equalsText Profile.historyEventTypeFieldSpec
          "temporal.history.WorkflowExecutionCanceled"))
    },
    constantRule Mapping.actionRuleId actionId .action forceCloseAction.value
      (.and
        (equalsText Profile.controlReceiptActionFieldSpec
          "workflow.action.force-close")
        (.and
          (equalsNatural Profile.controlReceiptAttemptFieldSpec 1)
          (equalsText Profile.controlReceiptStatusFieldSpec "accepted"))),
    constantRule Mapping.outcomeRuleId outcomeId .outcome cancellationUpgradedOutcome.value
      (equalsText Profile.historyEventTypeFieldSpec
        "temporal.history.WorkflowExecutionCanceled"),
    constantRule Mapping.deliveryRuleId deliveryObservationId .observation
      deliveryObservation.value
      (equalsText Profile.historyEventTypeFieldSpec
        "temporal.history.WorkflowExecutionCanceled"),
    {
      id := Mapping.cancellationCountRuleId
      output := cancellationCountObservationId
      outputKind := .observation
      value := .portable (.normalize { name := "natural.render", version := 1 }
        (field Profile.participantCancellationCountFieldSpec))
      condition := portableCondition
        (equalsNatural Profile.participantCancellationCountFieldSpec 1)
    },
    constantRule Mapping.ownershipRuleId ownershipObservationId .observation
      ownershipObservation.value
      (equalsText Profile.historyEventTypeFieldSpec
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
    Profile.cleanupCommandKindFieldSpec.disposition .retain,
    Profile.cleanupEndpointIdentityFieldSpec.disposition
      (.hash (some Profile.endpointDigestPolicyId)),
    Profile.cleanupErrorCodeFieldSpec.disposition .retain,
    Profile.cleanupNamespaceIdentityFieldSpec.disposition
      (.hash (some Profile.endpointDigestPolicyId)),
    Profile.cleanupOpenHandleCountFieldSpec.disposition .retain,
    Profile.cleanupOperationCorrelationFieldSpec.disposition .retain,
    Profile.cleanupRunCorrelationFieldSpec.disposition .retain,
    Profile.cleanupStatusFieldSpec.disposition .retain,
    Profile.cleanupTaskQueueIdentityFieldSpec.disposition
      (.hash (some Profile.endpointDigestPolicyId)),
    Profile.cleanupWorkflowCorrelationFieldSpec.disposition .retain,
    Profile.controlReceiptActionFieldSpec.disposition .retain,
    Profile.controlReceiptAttemptFieldSpec.disposition .retain,
    Profile.controlReceiptOccurrenceFieldSpec.disposition .retain,
    Profile.controlReceiptStatusFieldSpec.disposition .retain,
    Profile.historyEventIdFieldSpec.disposition .retain,
    Profile.historyEventTypeFieldSpec.disposition .retain,
    Profile.historyOperationCorrelationFieldSpec.disposition .retain,
    Profile.historyRunCorrelationFieldSpec.disposition .retain,
    Profile.historyWorkflowCorrelationFieldSpec.disposition .retain,
    Profile.participantCancellationCountFieldSpec.disposition .retain,
    Profile.participantCommandKindFieldSpec.disposition .retain,
    Profile.participantEndpointIdentityFieldSpec.disposition
      (.hash (some Profile.endpointDigestPolicyId)),
    Profile.participantErrorCodeFieldSpec.disposition .retain,
    Profile.participantNamespaceIdentityFieldSpec.disposition
      (.hash (some Profile.endpointDigestPolicyId)),
    Profile.participantOperationCorrelationFieldSpec.disposition .retain,
    Profile.participantRunCorrelationFieldSpec.disposition .retain,
    Profile.participantStatusFieldSpec.disposition .retain,
    Profile.participantTaskQueueIdentityFieldSpec.disposition
      (.hash (some Profile.endpointDigestPolicyId)),
    Profile.participantWorkflowCorrelationFieldSpec.disposition .retain
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

def actionFieldSpec : ObservationFieldSpec :=
  Temporal.System.Nexus.Observation.Profile.controlReceiptActionFieldSpec
def attemptFieldSpec : ObservationFieldSpec :=
  Temporal.System.Nexus.Observation.Profile.controlReceiptAttemptFieldSpec
def occurrenceFieldSpec : ObservationFieldSpec :=
  Temporal.System.Nexus.Observation.Profile.controlReceiptOccurrenceFieldSpec
def controlReceiptStatusFieldSpec : ObservationFieldSpec :=
  Temporal.System.Nexus.Observation.Profile.controlReceiptStatusFieldSpec
def eventTypeFieldSpec : ObservationFieldSpec :=
  Temporal.System.Nexus.Observation.Profile.historyEventTypeFieldSpec
def cancellationCountFieldSpec : ObservationFieldSpec :=
  Temporal.System.Nexus.Observation.Profile.participantCancellationCountFieldSpec
def participantOperationCorrelationFieldSpec : ObservationFieldSpec :=
  Temporal.System.Nexus.Observation.Profile.participantOperationCorrelationFieldSpec
def participantRunCorrelationFieldSpec : ObservationFieldSpec :=
  Temporal.System.Nexus.Observation.Profile.participantRunCorrelationFieldSpec
def participantWorkflowCorrelationFieldSpec : ObservationFieldSpec :=
  Temporal.System.Nexus.Observation.Profile.participantWorkflowCorrelationFieldSpec

def controlReceiptFaultDefinitionFieldSpec : ObservationFieldSpec := {
  kind := controlReceiptKind
  field := definitionId "umpire.evidence.field.fault-definition-id"
  valueType := .text
}
def controlReceiptFaultReceiptFieldSpec : ObservationFieldSpec := {
  kind := controlReceiptKind
  field := definitionId "umpire.evidence.field.fault-receipt-definition-id"
  valueType := .text
}
def controlReceiptCapabilityDefinitionFieldSpec : ObservationFieldSpec := {
  kind := controlReceiptKind
  field := definitionId "umpire.evidence.field.capability-definition-id"
  valueType := .text
}
def controlReceiptOperationCorrelationFieldSpec : ObservationFieldSpec := {
  kind := controlReceiptKind
  field := operationCorrelationField
  valueType := .text
}
def participantFaultDefinitionFieldSpec : ObservationFieldSpec := {
  controlReceiptFaultDefinitionFieldSpec with kind := participantKind
}
def participantFaultReceiptFieldSpec : ObservationFieldSpec := {
  controlReceiptFaultReceiptFieldSpec with kind := participantKind
}
def participantCapabilityDefinitionFieldSpec : ObservationFieldSpec := {
  controlReceiptCapabilityDefinitionFieldSpec with kind := participantKind
}
def participantSyntheticContributionCountFieldSpec : ObservationFieldSpec := {
  kind := participantKind
  field := definitionId "umpire.evidence.field.synthetic-contribution-count"
  valueType := .natural
}
def participantSyntheticMarkerFieldSpec : ObservationFieldSpec := {
  kind := participantKind
  field := definitionId "umpire.evidence.field.synthetic-contribution-marker"
  valueType := .text
}
def participantCancellationRequestedCountFieldSpec : ObservationFieldSpec := {
  kind := participantKind
  field := definitionId "umpire.evidence.field.cancellation-requested-count"
  valueType := .natural
}
def participantCancellationCompletedCountFieldSpec : ObservationFieldSpec := {
  kind := participantKind
  field := definitionId "umpire.evidence.field.cancellation-completed-count"
  valueType := .natural
}

def faultDefinitionField : DefinitionId := controlReceiptFaultDefinitionFieldSpec.field
def faultReceiptField : DefinitionId := controlReceiptFaultReceiptFieldSpec.field
def capabilityDefinitionField : DefinitionId := controlReceiptCapabilityDefinitionFieldSpec.field
def syntheticContributionCountField : DefinitionId :=
  participantSyntheticContributionCountFieldSpec.field
def syntheticMarkerField : DefinitionId := participantSyntheticMarkerFieldSpec.field
def cancellationRequestedCountField : DefinitionId :=
  participantCancellationRequestedCountFieldSpec.field
def cancellationCompletedCountField : DefinitionId :=
  participantCancellationCompletedCountFieldSpec.field

private def extendKind (kind : EvidenceKindDeclaration) : EvidenceKindDeclaration :=
  if kind.id == controlReceiptKind then {
    kind with fields := kind.fields ++ [
      controlReceiptFaultDefinitionFieldSpec.declaration,
      controlReceiptFaultReceiptFieldSpec.declaration,
      controlReceiptCapabilityDefinitionFieldSpec.declaration,
      controlReceiptOperationCorrelationFieldSpec.declaration
    ]
  } else if kind.id == participantKind then {
    kind with fields := kind.fields ++ [
      participantFaultDefinitionFieldSpec.declaration,
      participantFaultReceiptFieldSpec.declaration,
      participantCapabilityDefinitionFieldSpec.declaration,
      participantSyntheticContributionCountFieldSpec.declaration,
      participantSyntheticMarkerFieldSpec.declaration,
      participantCancellationRequestedCountFieldSpec.declaration,
      participantCancellationCompletedCountFieldSpec.declaration
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

private def profileField (fieldSpec : ObservationFieldSpec) : ObservationExpression :=
  fieldSpec.expression

private def profileEqualsText
    (fieldSpec : ObservationFieldSpec)
    (value : String) : ObservationExpression :=
  .equals (profileField fieldSpec) (.text value)

private def profileEqualsNatural
    (fieldSpec : ObservationFieldSpec)
    (value : Nat) : ObservationExpression :=
  .equals (profileField fieldSpec) (.natural value)

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
  value := .portable (profileField Profile.eventTypeFieldSpec)
  condition := some (.portable (.or
    (profileEqualsText Profile.eventTypeFieldSpec
      "temporal.history.WorkflowExecutionStarted")
    (profileEqualsText Profile.eventTypeFieldSpec
      "temporal.history.WorkflowExecutionCanceled")))
}

def actionRule : ObservationRule :=
  faultConstantRule Mapping.actionRuleId actionId .action forceCloseAction.value <|
    allConditions [
      profileEqualsText Profile.actionFieldSpec
        "workflow.action.force-close",
      profileEqualsNatural Profile.attemptFieldSpec 1,
      profileEqualsText Profile.occurrenceFieldSpec
        forceCloseOccurrenceId.value,
      profileEqualsText Profile.controlReceiptStatusFieldSpec "accepted",
      profileEqualsText Profile.controlReceiptFaultDefinitionFieldSpec
        faultDefinitionId.value,
      profileEqualsText Profile.controlReceiptFaultReceiptFieldSpec faultReceiptId.value,
      profileEqualsText Profile.controlReceiptCapabilityDefinitionFieldSpec
        cancellationCapabilityId.value
    ]

def outcomeRule : ObservationRule :=
  faultConstantRule Mapping.outcomeRuleId outcomeId .outcome
    cancellationUpgradedOutcome.value
    (profileEqualsText Profile.eventTypeFieldSpec
      "temporal.history.WorkflowExecutionCanceled")

def deliveryRule : ObservationRule :=
  faultConstantRule Mapping.deliveryRuleId deliveryObservationId .observation
    deliveryObservation.value
    (profileEqualsText Profile.eventTypeFieldSpec
      "temporal.history.WorkflowExecutionCanceled")

def semanticCountRule : ObservationRule := {
  id := Mapping.cancellationCountRuleId
  output := cancellationCountObservationId
  outputKind := .observation
  value := .portable (.text (toString semanticCancellationCount))
  condition := some (.portable <| allConditions [
    profileEqualsNatural Profile.cancellationCountFieldSpec
      mechanicalCallbackCount,
    profileEqualsNatural Profile.participantSyntheticContributionCountFieldSpec
      syntheticContributionCount,
    profileEqualsText Profile.participantSyntheticMarkerFieldSpec injectedMarker,
    profileEqualsNatural Profile.participantCancellationRequestedCountFieldSpec 1,
    profileEqualsNatural Profile.participantCancellationCompletedCountFieldSpec 1,
    profileEqualsText Profile.participantFaultDefinitionFieldSpec faultDefinitionId.value,
    profileEqualsText Profile.participantFaultReceiptFieldSpec faultReceiptId.value,
    profileEqualsText Profile.participantCapabilityDefinitionFieldSpec
      cancellationCapabilityId.value,
    .present (profileField Profile.participantOperationCorrelationFieldSpec),
    .present (profileField Profile.participantRunCorrelationFieldSpec),
    .present (profileField Profile.participantWorkflowCorrelationFieldSpec)
  ])
}

def ownershipRule : ObservationRule :=
  faultConstantRule Mapping.ownershipRuleId ownershipObservationId .observation
    ownershipObservation.value
    (profileEqualsText Profile.eventTypeFieldSpec
      "temporal.history.WorkflowExecutionCanceled")

private def additionalDispositions : List FieldDispositionDeclaration := [
  Profile.controlReceiptFaultDefinitionFieldSpec.disposition .retain,
  Profile.controlReceiptFaultReceiptFieldSpec.disposition .retain,
  Profile.controlReceiptCapabilityDefinitionFieldSpec.disposition .retain,
  Profile.controlReceiptOperationCorrelationFieldSpec.disposition .retain,
  Profile.participantSyntheticContributionCountFieldSpec.disposition .retain,
  Profile.participantSyntheticMarkerFieldSpec.disposition .retain,
  Profile.participantFaultDefinitionFieldSpec.disposition .retain,
  Profile.participantFaultReceiptFieldSpec.disposition .retain,
  Profile.participantCapabilityDefinitionFieldSpec.disposition .retain,
  Profile.participantCancellationRequestedCountFieldSpec.disposition .retain,
  Profile.participantCancellationCompletedCountFieldSpec.disposition .retain
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

private def observationDiagnostic
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
  | none => throw (observationDiagnostic .unresolvedBinding [record.id, field])

private def requireText
    (record : SyntheticEvidenceRecord)
    (field : DefinitionId)
    (expected : String) : Except ObservationDiagnostic Unit := do
  if (← requiredText record field) != expected then
    throw (observationDiagnostic .contradictoryFact [record.id, field])

private def requireNatural
    (record : SyntheticEvidenceRecord)
    (field : DefinitionId)
    (expected : Nat) : Except ObservationDiagnostic Unit :=
  match evidenceNatural? record field with
  | some value =>
      if value == expected then pure ()
      else throw (observationDiagnostic .contradictoryFact [record.id, field])
  | none => throw (observationDiagnostic .unresolvedBinding [record.id, field])

private def exactRecordOfKind
    (bundle : EvidenceBundle)
    (kind : DefinitionId) : Except ObservationDiagnostic SyntheticEvidenceRecord :=
  match bundle.records.filter fun record => record.kind == kind with
  | [record] => pure record
  | [] => throw (observationDiagnostic .unresolvedBinding [kind])
  | records => throw (observationDiagnostic .contradictoryFact
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
  | [] => throw (observationDiagnostic .unresolvedBinding
      [Profile.historyKind, Profile.eventTypeField])
  | multiple => throw (observationDiagnostic .contradictoryFact
      (multiple.map SyntheticEvidenceRecord.id))

private def requireMatchingText
    (records : List SyntheticEvidenceRecord)
    (field : DefinitionId)
    (expected : String) : Except ObservationDiagnostic Unit := do
  if expected == "" then
    throw (observationDiagnostic .unresolvedBinding [field])
  for record in records do
    if (← requiredText record field) != expected then
      throw (observationDiagnostic .contradictoryBinding [record.id, field])

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
    throw (observationDiagnostic .contradictoryFact
      (history.map SyntheticEvidenceRecord.id))
  let actualEventTypes ← history.mapM fun record => requiredText record Profile.eventTypeField
  if actualEventTypes != expectedHistoryEventTypes then
    throw (observationDiagnostic .contradictoryOrder
      (history.map SyntheticEvidenceRecord.id))
  if !started.causalParents.isEmpty || requested.causalParents != [started.id] ||
      completed.causalParents != [requested.id] || canceled.causalParents != [completed.id] ||
      !participant.causalParents.contains completed.id || participant.faultTarget != some completed.id then
    throw (observationDiagnostic .contradictoryOrder [
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

private def resultOfObservationDiagnostic
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
      | .error diagnostic => resultOfObservationDiagnostic diagnostic
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
