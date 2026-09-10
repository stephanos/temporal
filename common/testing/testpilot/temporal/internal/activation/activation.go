package activation

import (
	"context"
	"errors"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

// State belongs to one serial interpretation; distinct states may share a prepared plan.
type State struct {
	entrypoint   string
	instructions []testpilot.InstructionPlan
	states       []instructionState
	remaining    int64
	values       map[testpilot.ValueReference]*testpilotspb.Value
}

type instructionState uint8

const (
	unevaluated instructionState = iota
	enabled
	skipped
	admitted
	failed
)

func New(plan testpilot.EntrypointPlan) (*State, error) {
	if plan == (testpilot.EntrypointPlan{}) {
		return nil, errors.New("activation requires a prepared entrypoint")
	}
	if plan.Kind() != testpilotspb.ENTRYPOINT_KIND_WORKFLOW && plan.Kind() != testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER {
		return nil, errors.New("activation requires a workflow or Nexus-handler entrypoint")
	}
	instructions := plan.Instructions()
	return &State{
		entrypoint: plan.ID(), instructions: instructions,
		states: make([]instructionState, len(instructions)), remaining: plan.RuntimeWorkLimit(),
		values: make(map[testpilot.ValueReference]*testpilotspb.Value),
	}, nil
}

func (s *State) Evaluate(ctx context.Context, index int) (*testpilotspb.Value, bool, error) {
	if err := s.check(index, unevaluated); err != nil {
		return nil, false, err
	}
	s.states[index] = failed
	input, active, work, err := s.instructions[index].EvaluateInput(ctx, s.lookup, s.remaining)
	if chargeErr := s.consume(work); chargeErr != nil {
		return nil, false, chargeErr
	}
	if err != nil {
		return nil, false, err
	}
	if active {
		s.states[index] = enabled
	} else {
		s.states[index] = skipped
	}
	return input, active, nil
}

func (s *State) Admit(ctx context.Context, index int, outcome *testpilotspb.InstructionOutcome) error {
	if err := s.check(index, enabled); err != nil {
		return err
	}
	s.states[index] = failed
	instruction := s.instructions[index]
	snapshot, work, err := instruction.ValidateOutcome(ctx, outcome, s.remaining)
	if chargeErr := s.consume(work); chargeErr != nil {
		return chargeErr
	}
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	id := instruction.Source().GetInstructionId()
	for field, value := range snapshot.Fields {
		s.values[testpilot.ValueReference{Kind: testpilot.OutcomeReference, Entrypoint: s.entrypoint, ID: id, Field: int32(field)}] = value
	}
	s.states[index] = admitted
	return nil
}

func (s *State) check(index int, expected instructionState) error {
	if s == nil || index < 0 || index >= len(s.instructions) {
		return fmt.Errorf("invalid activation instruction index %d", index)
	}
	if s.states[index] != expected {
		return fmt.Errorf("instruction %d is not ready for this activation operation", index)
	}
	return nil
}

func (s *State) consume(work int64) error {
	if work < 0 || s.remaining < 0 || work > s.remaining {
		s.remaining = 0
		return errors.New("activation runtime work ceiling exceeded")
	}
	s.remaining -= work
	return nil
}

func (s *State) lookup(reference testpilot.ValueReference) *testpilotspb.Value {
	return proto.CloneOf(s.values[reference])
}
