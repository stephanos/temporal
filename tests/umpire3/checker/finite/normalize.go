package finite

import (
	"errors"
	"fmt"
	"slices"

	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
)

func NormalizeCounterexample(
	view protocolchecker.FirstOrderView,
	counterexample *CounterexampleError,
	receipt protocolchecker.TraceReplayReceipt,
) (protocolchecker.SemanticTrace, error) {
	if counterexample == nil || counterexample.Replica < 0 || len(counterexample.Actions) == 0 {
		return protocolchecker.SemanticTrace{}, errors.New("complete native counterexample is required")
	}
	if !slices.Equal(counterexample.Actions, receipt.Actions) {
		return protocolchecker.SemanticTrace{}, errors.New("native counterexample actions do not match canonical replay")
	}
	machine, err := NewMachine(view)
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	if err := validateCounterexamplePath(machine, counterexample); err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	if err := receipt.Validate(); err != nil {
		return protocolchecker.SemanticTrace{}, fmt.Errorf("normalize checked native counterexample: %w", err)
	}
	steps := make([]protocolchecker.SemanticTraceStep, len(receipt.Actions))
	for index, action := range receipt.Actions {
		steps[index] = protocolchecker.SemanticTraceStep{Action: action}
	}
	axioms := append([]string{}, view.Relation.Axioms...)
	slices.Sort(axioms)
	trace := protocolchecker.SemanticTrace{
		FormatVersion: protocolchecker.SemanticTraceFormatVersion,
		Kind:          protocolchecker.SemanticTraceFinite,
		Producer:      protocolchecker.SemanticTraceProducerNative,
		Target:        receipt.Target,
		Property:      receipt.Property,
		World:         receipt.World,
		Variant:       receipt.Variant,
		SemanticHash:  receipt.SemanticHash,
		Resources:     append([]protocolchecker.FirstOrderResource{}, view.Resources...),
		Steps:         steps,
		States:        []string{},
		LoopStart:     -1,
		Binding: protocolchecker.SemanticTraceBinding{
			Declaration: view.Relation.Declaration,
			Axioms:      axioms,
			TrustBadge:  view.Relation.TrustBadge,
		},
		Replay: protocolchecker.SemanticTraceReplay{
			Digest:     receipt.TraceDigest,
			Status:     receipt.Status,
			TrustBadge: receipt.TrustBadge,
			Axioms:     append([]string{}, receipt.Axioms...),
		},
		Omissions: []string{},
	}
	if err := trace.Validate(); err != nil {
		return protocolchecker.SemanticTrace{}, fmt.Errorf("normalize checked native counterexample: %w", err)
	}
	return trace, nil
}

func validateCounterexamplePath(
	machine Machine,
	counterexample *CounterexampleError,
) error {
	states, err := machine.InitialStates()
	if err != nil {
		return err
	}
	for _, action := range counterexample.Actions {
		next := make([]protocolchecker.FirstOrderState, 0)
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
	if !slices.ContainsFunc(states, func(state protocolchecker.FirstOrderState) bool {
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
