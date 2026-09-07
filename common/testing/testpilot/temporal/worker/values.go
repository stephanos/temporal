package worker

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

type activationValues struct {
	entrypoint string
	remaining  int64
	values     map[testpilot.ValueReference]*testpilotspb.Value
}

func newActivationValues(entrypoint string, work int64) *activationValues {
	return &activationValues{entrypoint: entrypoint, remaining: work, values: make(map[testpilot.ValueReference]*testpilotspb.Value)}
}

func (v *activationValues) store(instructionID string, snapshot *testpilot.OutcomeSnapshot) {
	if snapshot == nil {
		return
	}
	for field, value := range snapshot.Fields {
		v.values[testpilot.ValueReference{Kind: testpilot.OutcomeReference, Entrypoint: v.entrypoint, ID: instructionID, Field: int32(field)}] = proto.CloneOf(value)
	}
}

func (v *activationValues) lookup(reference testpilot.ValueReference) *testpilotspb.Value {
	return proto.CloneOf(v.values[reference])
}

func cloneOutcome(outcome *testpilotspb.InstructionOutcome) *testpilotspb.InstructionOutcome {
	return proto.CloneOf(outcome)
}
