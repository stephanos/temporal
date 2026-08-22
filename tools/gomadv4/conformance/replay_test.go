package conformance

import (
	"errors"
	"testing"

	"go.temporal.io/server/tools/gomadv4/trace"
	"go.temporal.io/server/tools/gomadv4/virtualtime"
)

func TestReplayAcceptsBehaviorAndRejectionCases(t *testing.T) {
	corpus := corpusForReplay(t)
	report, err := Replay(corpus, Limits{MaxTraces: 10, MaxSteps: 10, MaxRejections: 10})
	if err != nil {
		t.Fatal(err)
	}
	if report.Traces != 1 || report.Steps != 2 || report.Rejections != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestReplayReportsFirstObservableDivergence(t *testing.T) {
	corpus := corpusForReplay(t)
	corpus.Traces[0].Steps[1].ObservableDelta.CurrentTime = 99

	_, err := Replay(corpus, Limits{MaxTraces: 10, MaxSteps: 10, MaxRejections: 10})
	var divergence *Divergence
	if !errors.As(err, &divergence) {
		t.Fatalf("Replay() error = %T %v, want *Divergence", err, err)
	}
	if divergence.Trace != "schedule-and-advance" || divergence.Ordinal != 1 || divergence.Dimension != "observable_delta" {
		t.Fatalf("divergence = %#v", divergence)
	}
	if divergence.PreState.Now != 0 || divergence.ActualPostState.Now != 2 {
		t.Fatalf("divergence witness = %#v", divergence)
	}
}

func corpusForReplay(t *testing.T) trace.Corpus {
	t.Helper()
	state := virtualtime.NewState(0)
	scheduled, err := virtualtime.Step(state, virtualtime.ScheduleTimer("timer", 2))
	if err != nil {
		t.Fatal(err)
	}
	advanced, err := virtualtime.Step(scheduled.PostState, virtualtime.AdvanceTime())
	if err != nil {
		t.Fatal(err)
	}
	return trace.Corpus{
		Schema: "gomadv4.virtual-time-corpus/v1",
		Traces: []trace.BehaviorTrace{{
			Name: "schedule-and-advance", InitialState: state.Snapshot(),
			Steps: []trace.StepRecord{
				{Ordinal: 0, Action: scheduled.Action, PreStateIdentity: scheduled.PreStateIdentity, PostStateIdentity: scheduled.PostStateIdentity, ObservableDelta: scheduled.Delta},
				{Ordinal: 1, Action: advanced.Action, PreStateIdentity: advanced.PreStateIdentity, PostStateIdentity: advanced.PostStateIdentity, ObservableDelta: advanced.Delta},
			},
		}},
		Rejections: []trace.RejectionCase{{
			Name: "advance-empty", InitialState: state.Snapshot(), Action: virtualtime.AdvanceTime(),
			PreStateIdentity: state.Identity(), Code: virtualtime.RejectionNoPendingTimer,
		}},
	}
}
