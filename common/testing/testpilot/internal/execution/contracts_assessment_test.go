package execution

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
)

// answeringMonitor answers what its test scripts and rewrites the event it is handed, as a Monitor
// that owns its snapshot may.
type answeringMonitor struct {
	decision Decision
	err      error
	verdict  *testpilotspb.Verdict
	closed   []*testpilotspb.Run
}

func (m *answeringMonitor) Observe(_ context.Context, event *testpilotspb.RunEvent) (Decision, error) {
	event.SourceId = "rewritten"
	return m.decision, m.err
}

func (m *answeringMonitor) Close(_ context.Context, run *testpilotspb.Run) (*testpilotspb.Verdict, error) {
	m.closed = append(m.closed, run)
	return m.verdict, m.err
}

type recordingObserver struct{ events []*testpilotspb.RunEvent }

func (o *recordingObserver) Observe(_ context.Context, event *testpilotspb.RunEvent) {
	o.events = append(o.events, event)
}

type observerMap map[string]int

func (observerMap) Observe(context.Context, *testpilotspb.RunEvent) {}

func TestObservedMonitorKeepsTheMonitorsAnswers(t *testing.T) {
	failure := errors.New("evaluation failed")
	for name, test := range map[string]struct {
		decision Decision
		err      error
		observed bool
	}{
		"continue":                     {Continue, nil, true},
		"stop":                         {Stop, nil, true},
		"failed evaluation":            {Continue, failure, false},
		"failed evaluation that stops": {Stop, failure, false},
	} {
		t.Run(name, func(t *testing.T) {
			monitor := &answeringMonitor{decision: test.decision, err: test.err, verdict: &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_VIOLATED}}
			observer := &recordingObserver{}
			observed, err := Observed(monitor, observer)
			require.NoError(t, err)

			event := &testpilotspb.RunEvent{Sequence: 3, SourceId: "source", Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC}
			decision, err := observed.Observe(t.Context(), event)
			require.Equal(t, test.decision, decision)
			require.Equal(t, test.err, err)
			// The observer's snapshot is its own: what the Monitor did to the event does not reach it,
			// and an event whose evaluation did not commit is not delivered at all.
			var want []*testpilotspb.RunEvent
			if test.observed {
				want = []*testpilotspb.RunEvent{{Sequence: 3, SourceId: "source", Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC}}
			}
			protorequire.ProtoSliceEqual(t, want, observer.events)

			run := &testpilotspb.Run{RunId: "run"}
			verdict, err := observed.Close(t.Context(), run)
			require.Same(t, monitor.verdict, verdict)
			require.Equal(t, test.err, err)
			require.Equal(t, []*testpilotspb.Run{run}, monitor.closed)
		})
	}
}

func TestObservedRejectsEveryNilCapableForm(t *testing.T) {
	monitor := &answeringMonitor{}
	for _, observer := range []EventObserver{nil, (*recordingObserver)(nil), observerMap(nil)} {
		_, err := Observed(monitor, observer)
		require.Error(t, err)
	}
	for _, missing := range []Monitor{nil, (*answeringMonitor)(nil)} {
		_, err := Observed(missing, &recordingObserver{})
		require.Error(t, err)
	}
}
