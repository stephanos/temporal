package capability

import "go.temporal.io/server/tests/umpire3/regress"

const (
	Nexus               = regress.CapabilityNexus
	NexusWorkerControl  = regress.CapabilityNexusWorkerControl
	NexusObservation    = regress.CapabilityNexusObservation
	FailoverControl     = regress.CapabilityFailoverControl
	Update              = regress.CapabilityUpdate
	WorkflowTaskControl = regress.CapabilityWorkflowTaskControl
	HistoryObservation  = regress.CapabilityHistoryObservation
)
