package telemetry

const (
	ComponentPersistence     = "persistence"
	ComponentQueueArchival   = "queue.archival"
	ComponentQueueMemory     = "queue.memory"
	ComponentQueueOutbound   = "queue.outbound"
	ComponentQueueTimer      = "queue.timer"
	ComponentQueueTransfer   = "queue.transfer"
	ComponentQueueVisibility = "queue.visibility"
	ComponentUpdateRegistry  = "update.registry"

	NamespaceKey     = "temporalNamespace"
	NamespaceIDKey   = "temporalNamespaceID"
	WorkflowIDKey    = "temporalWorkflowID"
	WorkflowRunIDKey = "temporalRunID"
)
