package runtime

import "go.temporal.io/server/tools/umpire/internal/artifactv2"

const (
	EvidenceSourceCleanup           = "umpire.evidence.source.cleanup"
	EvidenceSourceControlReceipt    = artifactv2.ControlReceiptSourceDefinitionID
	EvidenceSourceHistory           = "umpire.evidence.source.history"
	EvidenceSourceParticipantOutput = "umpire.evidence.source.participant-output"
)

const EvidenceKindParticipantCommand = "umpire.evidence.kind.participant-command"

const (
	EvidenceFieldCancellationCallbackCount   = "umpire.evidence.field.cancellation-callback-count"
	EvidenceFieldCancellationCompletedCount  = "umpire.evidence.field.cancellation-completed-count"
	EvidenceFieldCancellationRequestedCount  = "umpire.evidence.field.cancellation-requested-count"
	EvidenceFieldCapabilityDefinitionID      = artifactv2.ControlReceiptCapabilityFieldDefinitionID
	EvidenceFieldCommandKind                 = "umpire.evidence.field.command-kind"
	EvidenceFieldEndpointIdentity            = "umpire.evidence.field.endpoint-identity"
	EvidenceFieldErrorCode                   = "umpire.evidence.field.error-code"
	EvidenceFieldEventID                     = "umpire.evidence.field.event-id"
	EvidenceFieldEventType                   = "umpire.evidence.field.event-type"
	EvidenceFieldFaultDefinitionID           = artifactv2.ControlReceiptFaultFieldDefinitionID
	EvidenceFieldFaultReceiptDefinitionID    = artifactv2.ControlReceiptFaultReceiptFieldDefinitionID
	EvidenceFieldNamespaceIdentity           = "umpire.evidence.field.namespace-identity"
	EvidenceFieldOpenHandleCount             = "umpire.evidence.field.open-handle-count"
	EvidenceFieldOperationCorrelationID      = artifactv2.ControlReceiptOperationFieldDefinitionID
	EvidenceFieldRunCorrelationID            = "umpire.evidence.field.run-correlation-id"
	EvidenceFieldStatus                      = "umpire.evidence.field.status"
	EvidenceFieldSyntheticContributionCount  = "umpire.evidence.field.synthetic-contribution-count"
	EvidenceFieldSyntheticContributionMarker = "umpire.evidence.field.synthetic-contribution-marker"
	EvidenceFieldTaskQueueIdentity           = "umpire.evidence.field.task-queue-identity"
	EvidenceFieldWorkflowCorrelationID       = "umpire.evidence.field.workflow-correlation-id"
)
