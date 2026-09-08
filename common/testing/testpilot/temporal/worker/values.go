package worker

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
)

func cloneOutcome(outcome *testpilotspb.InstructionOutcome) *testpilotspb.InstructionOutcome {
	return proto.CloneOf(outcome)
}
