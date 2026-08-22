package conformance

import (
	"errors"
	"fmt"
	"testing"

	"go.temporal.io/server/tools/common/formal/model"
	"go.temporal.io/server/tools/common/formal/trace"
)

type counterState struct {
	Value int `json:"value"`
}

type counterAction struct {
	Delta int `json:"delta"`
}

type counterDelta struct {
	Value int `json:"value"`
}

type counterRejection struct {
	code string
}

func (rejection *counterRejection) Error() string {
	return rejection.code
}

func TestReplayAcceptsBehaviorAndRejectionCases(t *testing.T) {
	report, err := Replay(counterCorpus(), counterAdapter(), Limits{MaxTraces: 10, MaxSteps: 10, MaxRejections: 10})
	if err != nil {
		t.Fatal(err)
	}
	if report.Traces != 1 || report.Steps != 1 || report.Rejections != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestReplayReportsFirstObservableDivergence(t *testing.T) {
	corpus := counterCorpus()
	corpus.Traces[0].Steps[0].ObservableDelta.Value = 99

	_, err := Replay(corpus, counterAdapter(), Limits{MaxTraces: 10, MaxSteps: 10, MaxRejections: 10})
	var divergence *Divergence[counterState, counterDelta]
	if !errors.As(err, &divergence) {
		t.Fatalf("Replay() error = %T %v, want *Divergence", err, err)
	}
	if divergence.Trace != "increment" || divergence.Ordinal != 0 || divergence.Dimension != "observable_delta" {
		t.Fatalf("divergence = %#v", divergence)
	}
	if divergence.PreState.Value != 0 || divergence.ActualPostState == nil || divergence.ActualPostState.Value != 1 {
		t.Fatalf("divergence witness = %#v", divergence)
	}
}

func counterAdapter() Adapter[counterState, counterAction, counterDelta, counterState, string] {
	return Adapter[counterState, counterAction, counterDelta, counterState, string]{
		Subject:  "counter",
		Restore:  func(snapshot counterState) (counterState, error) { return snapshot, nil },
		Snapshot: func(state counterState) counterState { return state },
		Identity: func(state counterState) string { return fmt.Sprintf("state:%d", state.Value) },
		Step: func(state counterState, action counterAction) (model.Transition[counterState, counterAction, counterDelta], error) {
			if action.Delta == 0 {
				return model.Transition[counterState, counterAction, counterDelta]{}, &counterRejection{code: "zero_delta"}
			}
			post := counterState{Value: state.Value + action.Delta}
			return model.Transition[counterState, counterAction, counterDelta]{
				PostState: post,
				Observation: model.Observation[counterAction, counterDelta]{
					Action: action, PreStateIdentity: fmt.Sprintf("state:%d", state.Value), PostStateIdentity: fmt.Sprintf("state:%d", post.Value), ObservableDelta: counterDelta{Value: action.Delta},
				},
			}, nil
		},
		RejectionCode: func(err error) (string, bool) {
			var rejection *counterRejection
			if !errors.As(err, &rejection) {
				return "", false
			}
			return rejection.code, true
		},
		EqualObservableDelta: func(left, right counterDelta) bool { return left == right },
	}
}

func counterCorpus() trace.Corpus[counterState, counterAction, counterDelta, string] {
	return trace.Corpus[counterState, counterAction, counterDelta, string]{
		Traces: []trace.BehaviorTrace[counterState, counterAction, counterDelta]{{
			Name: "increment", InitialState: counterState{},
			Steps: []trace.StepRecord[counterAction, counterDelta]{{
				Ordinal: 0, Action: counterAction{Delta: 1}, PreStateIdentity: "state:0", PostStateIdentity: "state:1", ObservableDelta: counterDelta{Value: 1},
			}},
		}},
		Rejections: []trace.RejectionCase[counterState, counterAction, string]{{
			Name: "zero", InitialState: counterState{}, Action: counterAction{}, PreStateIdentity: "state:0", Code: "zero_delta",
		}},
	}
}
