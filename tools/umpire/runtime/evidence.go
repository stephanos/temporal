package runtime

import "go.temporal.io/server/tools/umpire/internal/artifactv2"

const (
	EvidenceSourceCleanup           = "umpire.evidence.source.cleanup"
	EvidenceSourceControlReceipt    = artifactv2.ControlReceiptSourceDefinitionID
	EvidenceSourceHistory           = "umpire.evidence.source.history"
	EvidenceSourceParticipantOutput = "umpire.evidence.source.participant-output"
)

const (
	EvidenceFieldCancellationCallbackCount = "umpire.evidence.field.cancellation-callback-count"
	EvidenceFieldCommandKind               = "umpire.evidence.field.command-kind"
	EvidenceFieldEndpointIdentity          = "umpire.evidence.field.endpoint-identity"
	EvidenceFieldErrorCode                 = "umpire.evidence.field.error-code"
	EvidenceFieldEventID                   = "umpire.evidence.field.event-id"
	EvidenceFieldEventType                 = "umpire.evidence.field.event-type"
	EvidenceFieldNamespaceIdentity         = "umpire.evidence.field.namespace-identity"
	EvidenceFieldOpenHandleCount           = "umpire.evidence.field.open-handle-count"
	EvidenceFieldOperationCorrelationID    = "umpire.evidence.field.operation-correlation-id"
	EvidenceFieldRunCorrelationID          = "umpire.evidence.field.run-correlation-id"
	EvidenceFieldStatus                    = "umpire.evidence.field.status"
	EvidenceFieldTaskQueueIdentity         = "umpire.evidence.field.task-queue-identity"
	EvidenceFieldWorkflowCorrelationID     = "umpire.evidence.field.workflow-correlation-id"
)
