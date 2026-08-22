package conformance

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv4/trace"
	"go.temporal.io/server/tools/gomadv4/virtualtime"
)

type Limits struct {
	MaxTraces     int
	MaxSteps      int
	MaxRejections int
}

type Report struct {
	Traces     int `json:"traces"`
	Steps      int `json:"steps"`
	Rejections int `json:"rejections"`
}

type Divergence struct {
	Trace                   string                      `json:"trace"`
	Ordinal                 int                         `json:"ordinal"`
	Dimension               string                      `json:"dimension"`
	Expected                string                      `json:"expected"`
	Actual                  string                      `json:"actual"`
	PreState                virtualtime.StateSnapshot   `json:"pre_state"`
	ActualPostState         virtualtime.StateSnapshot   `json:"actual_post_state"`
	ExpectedPostIdentity    string                      `json:"expected_post_identity,omitempty"`
	ExpectedObservableDelta virtualtime.ObservableDelta `json:"expected_observable_delta"`
	ActualObservableDelta   virtualtime.ObservableDelta `json:"actual_observable_delta"`
}

func (divergence *Divergence) Error() string {
	return fmt.Sprintf("virtual time trace %q diverged at step %d (%s): got %s, want %s", divergence.Trace, divergence.Ordinal, divergence.Dimension, divergence.Actual, divergence.Expected)
}

func Replay(corpus trace.Corpus, limits Limits) (Report, error) {
	if limits.MaxTraces <= 0 || limits.MaxSteps <= 0 || limits.MaxRejections <= 0 {
		return Report{}, errors.New("virtual time conformance limits must be positive")
	}
	if len(corpus.Traces) > limits.MaxTraces {
		return Report{}, fmt.Errorf("virtual time trace count %d exceeds %d", len(corpus.Traces), limits.MaxTraces)
	}
	if len(corpus.Rejections) > limits.MaxRejections {
		return Report{}, fmt.Errorf("virtual time rejection count %d exceeds %d", len(corpus.Rejections), limits.MaxRejections)
	}
	report := Report{Traces: len(corpus.Traces), Rejections: len(corpus.Rejections)}
	for _, behavior := range corpus.Traces {
		state, err := virtualtime.Restore(behavior.InitialState)
		if err != nil {
			return report, fmt.Errorf("restore virtual time trace %q: %w", behavior.Name, err)
		}
		if len(behavior.Steps) > limits.MaxSteps-report.Steps {
			return report, fmt.Errorf("virtual time step count exceeds %d", limits.MaxSteps)
		}
		for ordinal, expected := range behavior.Steps {
			if expected.Ordinal != ordinal {
				return report, divergence(behavior.Name, ordinal, "ordinal", fmt.Sprint(expected.Ordinal), fmt.Sprint(ordinal), state, virtualtime.Transition{}, expected)
			}
			if state.Identity() != expected.PreStateIdentity {
				return report, divergence(behavior.Name, ordinal, "pre_state_identity", expected.PreStateIdentity, state.Identity(), state, virtualtime.Transition{}, expected)
			}
			actual, err := virtualtime.Step(state, expected.Action)
			if err != nil {
				return report, divergence(behavior.Name, ordinal, "action_rejected", "accepted", err.Error(), state, virtualtime.Transition{}, expected)
			}
			if actual.PostStateIdentity != expected.PostStateIdentity {
				return report, divergence(behavior.Name, ordinal, "post_state_identity", expected.PostStateIdentity, actual.PostStateIdentity, state, actual, expected)
			}
			if !equalJSON(actual.Delta, expected.ObservableDelta) {
				return report, divergence(behavior.Name, ordinal, "observable_delta", compactJSON(expected.ObservableDelta), compactJSON(actual.Delta), state, actual, expected)
			}
			state = actual.PostState
			report.Steps++
		}
	}
	for _, expected := range corpus.Rejections {
		state, err := virtualtime.Restore(expected.InitialState)
		if err != nil {
			return report, fmt.Errorf("restore virtual time rejection %q: %w", expected.Name, err)
		}
		if state.Identity() != expected.PreStateIdentity {
			return report, fmt.Errorf("virtual time rejection %q pre-state identity = %q, want %q", expected.Name, state.Identity(), expected.PreStateIdentity)
		}
		before := state.Identity()
		_, err = virtualtime.Step(state, expected.Action)
		var rejection *virtualtime.Rejection
		if !errors.As(err, &rejection) {
			return report, fmt.Errorf("virtual time rejection %q action was not rejected with a typed rejection: %v", expected.Name, err)
		}
		if rejection.Code != expected.Code {
			return report, fmt.Errorf("virtual time rejection %q code = %q, want %q", expected.Name, rejection.Code, expected.Code)
		}
		if state.Identity() != before {
			return report, fmt.Errorf("virtual time rejection %q mutated its input state", expected.Name)
		}
	}
	return report, nil
}

func divergence(name string, ordinal int, dimension, expectedValue, actualValue string, pre virtualtime.State, actual virtualtime.Transition, expected trace.StepRecord) *Divergence {
	witness := &Divergence{
		Trace: name, Ordinal: ordinal, Dimension: dimension, Expected: expectedValue, Actual: actualValue,
		PreState: pre.Snapshot(), ExpectedPostIdentity: expected.PostStateIdentity,
		ExpectedObservableDelta: expected.ObservableDelta, ActualObservableDelta: actual.Delta,
	}
	if actual.PostStateIdentity != "" {
		witness.ActualPostState = actual.PostState.Snapshot()
	}
	return witness
}

func equalJSON(left, right any) bool {
	return compactJSON(left) == compactJSON(right)
}

func compactJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("<encode error: %v>", err)
	}
	return string(encoded)
}
