package native

import (
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/protocol"
)

func NormalizeCounterexample(
	view protocol.FirstOrderView,
	counterexample *CounterexampleError,
	receipt protocol.TraceReplayReceipt,
) (protocol.SemanticTrace, error) {
	if counterexample == nil || counterexample.Replica < 0 || len(counterexample.Actions) == 0 {
		return protocol.SemanticTrace{}, errors.New("complete native counterexample is required")
	}
	if !slices.Equal(counterexample.Actions, receipt.Actions) {
		return protocol.SemanticTrace{}, errors.New("native counterexample actions do not match canonical replay")
	}
	machine, err := protocol.NewFirstOrderMachine(view)
	if err != nil {
		return protocol.SemanticTrace{}, err
	}
	if err := validateCounterexamplePath(machine, counterexample); err != nil {
		return protocol.SemanticTrace{}, err
	}
	trace, err := protocol.SemanticTraceFromTraceReplayReceipt(
		protocol.SemanticTraceProducerNative, receipt)
	if err != nil {
		return protocol.SemanticTrace{}, fmt.Errorf("normalize checked native counterexample: %w", err)
	}
	return trace, nil
}

func validateCounterexamplePath(
	machine protocol.FirstOrderMachine,
	counterexample *CounterexampleError,
) error {
	states, err := machine.InitialStates()
	if err != nil {
		return err
	}
	for _, action := range counterexample.Actions {
		next := make([]protocol.FirstOrderState, 0)
		for _, state := range states {
			successors, successorErr := machine.Successors(state)
			if successorErr != nil {
				return successorErr
			}
			for _, successor := range successors {
				if successor.Action == action {
					next = append(next, successor.State)
				}
			}
		}
		if len(next) == 0 {
			return fmt.Errorf("native counterexample action %q is not canonically reachable", action)
		}
		states = next
	}
	want, err := machine.StateKey(counterexample.State)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(states, func(state protocol.FirstOrderState) bool {
		key, keyErr := machine.StateKey(state)
		return keyErr == nil && key == want
	}) {
		return errors.New("native counterexample state does not match its canonical action path")
	}
	safe, err := machine.Invariant(counterexample.State)
	if err != nil {
		return err
	}
	if safe {
		return errors.New("native counterexample final state does not violate the property")
	}
	return nil
}
