package conformance

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/common/formal/model"
	"go.temporal.io/server/tools/common/formal/trace"
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

type Adapter[S, A, D, R any, C ~string] struct {
	Subject              string
	Restore              func(S) (R, error)
	Snapshot             func(R) S
	Identity             func(R) string
	Step                 func(R, A) (model.Transition[R, A, D], error)
	RejectionCode        func(error) (C, bool)
	EqualObservableDelta func(D, D) bool
}

type Divergence[S, D any] struct {
	Subject                 string `json:"subject"`
	Trace                   string `json:"trace"`
	Ordinal                 int    `json:"ordinal"`
	Dimension               string `json:"dimension"`
	Expected                string `json:"expected"`
	Actual                  string `json:"actual"`
	PreState                S      `json:"pre_state"`
	ActualPostState         *S     `json:"actual_post_state,omitempty"`
	ExpectedPostIdentity    string `json:"expected_post_identity,omitempty"`
	ExpectedObservableDelta D      `json:"expected_observable_delta"`
	ActualObservableDelta   D      `json:"actual_observable_delta"`
}

func (divergence *Divergence[S, D]) Error() string {
	return fmt.Sprintf("%s trace %q diverged at step %d (%s): got %s, want %s", divergence.Subject, divergence.Trace, divergence.Ordinal, divergence.Dimension, divergence.Actual, divergence.Expected)
}

func Replay[S, A, D, R any, C ~string](corpus trace.Corpus[S, A, D, C], adapter Adapter[S, A, D, R, C], limits Limits) (Report, error) {
	if err := validateAdapter(adapter); err != nil {
		return Report{}, err
	}
	if limits.MaxTraces <= 0 || limits.MaxSteps <= 0 || limits.MaxRejections <= 0 {
		return Report{}, fmt.Errorf("%s conformance limits must be positive", adapter.Subject)
	}
	if len(corpus.Traces) > limits.MaxTraces {
		return Report{}, fmt.Errorf("%s trace count %d exceeds %d", adapter.Subject, len(corpus.Traces), limits.MaxTraces)
	}
	if len(corpus.Rejections) > limits.MaxRejections {
		return Report{}, fmt.Errorf("%s rejection count %d exceeds %d", adapter.Subject, len(corpus.Rejections), limits.MaxRejections)
	}
	report := Report{Traces: len(corpus.Traces), Rejections: len(corpus.Rejections)}
	for _, behavior := range corpus.Traces {
		state, err := adapter.Restore(behavior.InitialState)
		if err != nil {
			return report, fmt.Errorf("restore %s trace %q: %w", adapter.Subject, behavior.Name, err)
		}
		if len(behavior.Steps) > limits.MaxSteps-report.Steps {
			return report, fmt.Errorf("%s step count exceeds %d", adapter.Subject, limits.MaxSteps)
		}
		for ordinal, expected := range behavior.Steps {
			if expected.Ordinal != ordinal {
				return report, divergence(adapter, behavior.Name, ordinal, "ordinal", fmt.Sprint(ordinal), fmt.Sprint(expected.Ordinal), state, nil, expected)
			}
			if adapter.Identity(state) != expected.PreStateIdentity {
				return report, divergence(adapter, behavior.Name, ordinal, "pre_state_identity", expected.PreStateIdentity, adapter.Identity(state), state, nil, expected)
			}
			actual, err := adapter.Step(state, expected.Action)
			if err != nil {
				return report, divergence(adapter, behavior.Name, ordinal, "action_rejected", "accepted", err.Error(), state, nil, expected)
			}
			if actual.PostStateIdentity != expected.PostStateIdentity {
				return report, divergence(adapter, behavior.Name, ordinal, "post_state_identity", expected.PostStateIdentity, actual.PostStateIdentity, state, &actual, expected)
			}
			if !adapter.EqualObservableDelta(actual.ObservableDelta, expected.ObservableDelta) {
				return report, divergence(adapter, behavior.Name, ordinal, "observable_delta", compactJSON(expected.ObservableDelta), compactJSON(actual.ObservableDelta), state, &actual, expected)
			}
			state = actual.PostState
			report.Steps++
		}
	}
	for _, expected := range corpus.Rejections {
		state, err := adapter.Restore(expected.InitialState)
		if err != nil {
			return report, fmt.Errorf("restore %s rejection %q: %w", adapter.Subject, expected.Name, err)
		}
		if adapter.Identity(state) != expected.PreStateIdentity {
			return report, fmt.Errorf("%s rejection %q pre-state identity = %q, want %q", adapter.Subject, expected.Name, adapter.Identity(state), expected.PreStateIdentity)
		}
		before := adapter.Identity(state)
		_, err = adapter.Step(state, expected.Action)
		code, ok := adapter.RejectionCode(err)
		if !ok {
			return report, fmt.Errorf("%s rejection %q action was not rejected with a typed rejection: %v", adapter.Subject, expected.Name, err)
		}
		if code != expected.Code {
			return report, fmt.Errorf("%s rejection %q code = %q, want %q", adapter.Subject, expected.Name, code, expected.Code)
		}
		if adapter.Identity(state) != before {
			return report, fmt.Errorf("%s rejection %q mutated its input state", adapter.Subject, expected.Name)
		}
	}
	return report, nil
}

func validateAdapter[S, A, D, R any, C ~string](adapter Adapter[S, A, D, R, C]) error {
	if adapter.Subject == "" || adapter.Restore == nil || adapter.Snapshot == nil || adapter.Identity == nil || adapter.Step == nil || adapter.RejectionCode == nil || adapter.EqualObservableDelta == nil {
		return errors.New("formal conformance adapter is incomplete")
	}
	return nil
}

func divergence[S, A, D, R any, C ~string](adapter Adapter[S, A, D, R, C], name string, ordinal int, dimension, expectedValue, actualValue string, pre R, actual *model.Transition[R, A, D], expected trace.StepRecord[A, D]) *Divergence[S, D] {
	witness := &Divergence[S, D]{
		Subject: adapter.Subject, Trace: name, Ordinal: ordinal, Dimension: dimension, Expected: expectedValue, Actual: actualValue,
		PreState: adapter.Snapshot(pre), ExpectedPostIdentity: expected.PostStateIdentity,
		ExpectedObservableDelta: expected.ObservableDelta,
	}
	if actual != nil {
		post := adapter.Snapshot(actual.PostState)
		witness.ActualPostState = &post
		witness.ActualObservableDelta = actual.ObservableDelta
	}
	return witness
}

func compactJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("<encode error: %v>", err)
	}
	return string(encoded)
}
