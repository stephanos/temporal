package prototypetemporal

import "testing"

func TestReplayInspectionWorkflowFailure(t *testing.T) {
	TestPrototypeWorkflow(t)
	t.Fatal("Temporal inspection canary after workflow timers and activity retries completed")
}
