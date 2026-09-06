package worker

import (
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

type activationValues struct {
	entrypoint string
	remaining  int64
	values     map[testpilot.ValueReference]*testpilotpb.Value
}

func newActivationValues(entrypoint string, work int64) *activationValues {
	return &activationValues{entrypoint: entrypoint, remaining: work, values: make(map[testpilot.ValueReference]*testpilotpb.Value)}
}

func (v *activationValues) store(instructionID string, snapshot *testpilot.OutcomeSnapshot) {
	if snapshot == nil {
		return
	}
	for field, value := range snapshot.Fields {
		v.values[testpilot.ValueReference{Kind: testpilot.OutcomeReference, Entrypoint: v.entrypoint, ID: instructionID, Field: int32(field)}] = proto.CloneOf(value)
	}
}

func (v *activationValues) lookup(reference testpilot.ValueReference) *testpilotpb.Value {
	return proto.CloneOf(v.values[reference])
}

func cloneOutcome(outcome *testpilotpb.InstructionOutcome) *testpilotpb.InstructionOutcome {
	return proto.CloneOf(outcome)
}
