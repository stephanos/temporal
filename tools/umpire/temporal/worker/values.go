package worker

import (
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/protobuf/proto"
)

type activationValues struct {
	entrypoint string
	remaining  int64
	values     map[umpire.ValueReference]*umpirespb.Value
}

func newActivationValues(entrypoint string, work int64) *activationValues {
	return &activationValues{entrypoint: entrypoint, remaining: work, values: make(map[umpire.ValueReference]*umpirespb.Value)}
}

func (v *activationValues) store(instructionID string, snapshot *umpire.OutcomeSnapshot) {
	if snapshot == nil {
		return
	}
	for field, value := range snapshot.Fields {
		v.values[umpire.ValueReference{Kind: umpire.OutcomeReference, Entrypoint: v.entrypoint, ID: instructionID, Field: int32(field)}] = proto.CloneOf(value)
	}
}

func (v *activationValues) lookup(reference umpire.ValueReference) *umpirespb.Value {
	return proto.CloneOf(v.values[reference])
}

func cloneOutcome(outcome *umpirespb.InstructionOutcome) *umpirespb.InstructionOutcome {
	return proto.CloneOf(outcome)
}
