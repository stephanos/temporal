package replay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/campaign"
)

var (
	runSatisfied = func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
		return &testpilotspb.Run{
				RunId: "r", Events: []*testpilotspb.RunEvent{{Sequence: 1, Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED}},
				Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED},
			},
			&testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED}, nil
	}
	runIncomplete = func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
		return &testpilotspb.Run{
				RunId: "r", Events: []*testpilotspb.RunEvent{{Sequence: 1, Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED}},
				Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE, Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED},
			},
			&testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_INCONCLUSIVE}, nil
	}
)

// reduction is one reduction over the corpus's violated subject: the binder scripts every bind
// in order, the subject's two first, and every candidate carries the subject's own Case, so an
// unscripted attempt reproduces.
type reduction struct {
	subject *Subject
	binder  *scriptedBinder
	bridge  *Bridge
	fake    *fakeReplayBridge
	reducer Reducer
}

func newReduction(t *testing.T, script []attemptScript, sweep ...fakeEdit) *reduction {
	t.Helper()
	subject, prepare := admittedSubject(t)
	bridge, fake := newFakeReplayBridge(t, json.RawMessage(subject.Canonical), subject.Case.GetCaseId(), sweep...)
	admitted, err := bridge.Admit(t.Context(), "set", subject.Driver.Profile, Named{Query: "q"}, fake.identity)
	require.NoError(t, err)
	binder := &scriptedBinder{prepare: prepare, script: script}
	clock := time.Unix(0, 0)
	return &reduction{subject: subject, binder: binder, bridge: bridge, fake: fake, reducer: Reducer{
		Bridge: bridge, Admitted: admitted, Binder: binder, Prepare: prepare, Subject: subject,
		Limits: DefaultLimits, Now: func() time.Time { return clock },
	}}
}

func (r *reduction) run(ctx context.Context, t *testing.T) (Reduction, error) {
	t.Helper()
	subjectReruns, err := Rerun(ctx, r.binder, r.subject.Target())
	require.NoError(t, err)
	return r.reducer.Reduce(ctx, subjectReruns)
}

func fates(report Reduction) []string {
	var out []string
	for _, settled := range report.Edits {
		out = append(out, settled.Edit.Edit+"="+settled.Fate)
	}
	return out
}

// One sweep: the first candidate reproduces on both Runs and is retained, an inapplicable edit is
// passed over with no Run, the last candidate's Runs are satisfied and it is not reproduced. The
// result is the bridge's: minimized, with the retained candidate's digest.
func TestReduceMinimizesInOneSweep(t *testing.T) {
	r := newReduction(t, []attemptScript{nil, nil, nil, nil, runSatisfied, runSatisfied},
		edit(2, "c"), fakeEdit{Edit: edit(1, "b").Edit, inapplicable: true}, edit(0, "a"))
	report, err := r.run(t.Context(), t)
	require.NoError(t, err)
	require.True(t, report.Attempted)
	require.Equal(t, "minimized", report.Status)
	require.Equal(t, "candidate-2", report.Retained)
	require.Equal(t, 6, report.Runs, "two for the subject, two per candidate, none for the inapplicable edit")
	require.Equal(t, []string{"dropPrefixStep 2=retained", "dropPrefixStep 1=inapplicable", "dropPrefixStep 0=not-reproduced"}, fates(report))
	require.Len(t, report.Candidates, 2)
	require.Equal(t, []Class{ClassReproduced, ClassReproduced}, report.Candidates[0].Classes)
	require.Equal(t, []Class{ClassNotReproduced, ClassNotReproduced}, report.Candidates[1].Classes)
	require.Empty(t, report.Limit)
	require.False(t, report.Stopped)
}

// A subject whose reruns did not reproduce its key is not reduced: no candidate is asked for and
// the report says why.
func TestReduceIsNotAttemptedOnAnUnreproducedSubject(t *testing.T) {
	r := newReduction(t, []attemptScript{runSatisfied, nil}, edit(0, "a"))
	report, err := r.run(t.Context(), t)
	require.NoError(t, err)
	require.False(t, report.Attempted)
	require.Equal(t, ReductionNotAttempted, report.Status)
	require.Contains(t, report.NotAttempted, string(ClassNotReproduced))
	require.Empty(t, report.Candidates)
	for _, frame := range r.fake.frames {
		require.NotEqual(t, "next", frame.Frame, "no candidate is asked for")
	}
	require.Equal(t, "stopped", r.fake.frames[len(r.fake.frames)-1].Status)
}

// An indeterminate Run is rerun alone once, one Run spent: a reproducing retry makes the pair
// reproduced; one still indeterminate settles the edit undecided and the reduction incomplete,
// naming it, never counting it as not reproduced.
func TestReduceRetriesAnIndeterminateRunOnce(t *testing.T) {
	r := newReduction(t, []attemptScript{nil, nil, runIncomplete, nil, nil}, edit(1, "b"), edit(0, "a"))
	report, err := r.run(t.Context(), t)
	require.NoError(t, err)
	require.Equal(t, []Class{ClassIndeterminate, ClassReproduced, ClassReproduced}, report.Candidates[0].Classes)
	require.Equal(t, ClassReproduced, report.Candidates[0].Class)
	require.Equal(t, "retained", report.Candidates[0].Fate)

	r = newReduction(t, []attemptScript{nil, nil, runIncomplete, nil, runIncomplete}, edit(1, "b"), edit(0, "a"))
	report, err = r.run(t.Context(), t)
	require.NoError(t, err)
	require.Equal(t, "incomplete", report.Status)
	require.Contains(t, report.Reason, "dropPrefixStep 1")
	require.Equal(t, ClassIndeterminate, report.Candidates[0].Class)
	require.Equal(t, "undecided", report.Candidates[0].Fate)
	require.Equal(t, 5, report.Runs)
	require.Len(t, report.Candidates, 1, "nothing is handed out after an undecided edit")
}

// A pair that one conclusive Run already decides is not retried: a not-reproduced Run beside an
// indeterminate one makes the pair not reproduced, and no Run is spent on the indeterminate one.
func TestReduceDoesNotRetryADecidedPair(t *testing.T) {
	r := newReduction(t, []attemptScript{nil, nil, runSatisfied, runIncomplete}, edit(0, "a"))
	report, err := r.run(t.Context(), t)
	require.NoError(t, err)
	require.Equal(t, []Class{ClassNotReproduced, ClassIndeterminate}, report.Candidates[0].Classes)
	require.Equal(t, ClassNotReproduced, report.Candidates[0].Class)
	require.Equal(t, 4, report.Runs, "no retry for a decided pair")
	require.Equal(t, "irreducible", report.Status)
}

// A sweep the bridge capped runs and ends incomplete at the edit cap, which the report names.
func TestReduceNamesTheEditCapOfACappedSweep(t *testing.T) {
	r := newReduction(t, nil, edit(0, "a"))
	r.fake.capped = true
	subjectReruns, err := Rerun(t.Context(), r.binder, r.subject.Target())
	require.NoError(t, err)
	admitted := r.reducer.Admitted
	admitted.Capped = true
	r.reducer.Admitted = admitted
	report, err := r.reducer.Reduce(t.Context(), subjectReruns)
	require.NoError(t, err)
	require.Equal(t, LimitEdits, report.Limit)
	require.Equal(t, "incomplete", report.Status)
}

// A candidate that does not prepare is reported rejected and never rerun.
func TestReduceReportsAPreparationRejectionWithoutARun(t *testing.T) {
	r := newReduction(t, nil, edit(0, "a"))
	r.reducer.Prepare = func(string, *testpilotspb.Case) (*testpilot.PreparedCase, error) {
		return nil, &testpilot.PreparationError{Category: testpilot.PreparationUnsupported, Path: "program", Detail: "opcode outside the Profile"}
	}
	subjectReruns, err := Rerun(t.Context(), r.binder, r.subject.Target())
	require.NoError(t, err)
	binds := r.binder.binds
	report, err := r.reducer.Reduce(t.Context(), subjectReruns)
	require.NoError(t, err)
	require.Equal(t, binds, r.binder.binds, "a rejected candidate is never bound")
	require.Equal(t, "rejected", report.Candidates[0].Fate)
	require.Contains(t, report.Candidates[0].Reason, "opcode outside the Profile")
	require.Equal(t, "irreducible", report.Status)
}

// Each limit is checked before the work it bounds and the report names it.
func TestReduceStopsAtEachLimitNamingIt(t *testing.T) {
	for name, probe := range map[string]struct {
		limits func(*Limits)
		clock  time.Duration
		limit  string
		runs   int
	}{
		"runs before a candidate": {limits: func(l *Limits) { l.Runs = 5 }, limit: LimitRuns, runs: 4},
		"wall time":               {limits: func(*Limits) {}, clock: 26 * time.Minute, limit: LimitWallTime, runs: 2},
		"case bytes":              {limits: func(l *Limits) { l.CaseBytes = 10 }, limit: LimitCaseBytes, runs: 2},
		"edits":                   {limits: func(l *Limits) { l.Edits = 1 }, limit: LimitEdits, runs: 2},
		"run events":              {limits: func(l *Limits) { l.RunEvents = 1 }, limit: LimitRunEvents, runs: 2},
	} {
		t.Run(name, func(t *testing.T) {
			r := newReduction(t, nil, edit(1, "b"), edit(0, "a"))
			probe.limits(&r.reducer.Limits)
			start := time.Unix(0, 0)
			calls := 0
			r.reducer.Now = func() time.Time {
				calls++
				if calls == 1 {
					return start
				}
				return start.Add(probe.clock)
			}
			report, err := r.run(t.Context(), t)
			require.NoError(t, err)
			require.Equal(t, probe.limit, report.Limit)
			require.Equal(t, "incomplete", report.Status)
			require.Equal(t, probe.runs, report.Runs)
		})
	}
	r := newReduction(t, nil, edit(0, "a"))
	r.reducer.Limits.ReportBytes = 16
	report, err := r.run(t.Context(), t)
	var limit *campaign.LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitReportBytes, report.Limit)
}

// A stop while a candidate's Runs are open names the candidate lost, never settles it, and still
// asks the bridge for the result.
func TestReduceNamesTheCandidateLostToAStop(t *testing.T) {
	r := newReduction(t, nil, edit(1, "b"), edit(0, "a"))
	ctx, cancel := context.WithCancel(t.Context())
	subjectReruns, err := Rerun(ctx, r.binder, r.subject.Target())
	require.NoError(t, err)
	r.binder.onRun = cancel
	binds := r.binder.binds
	report, err := r.reducer.Reduce(ctx, subjectReruns)
	require.NoError(t, err)
	require.True(t, report.Stopped)
	require.Empty(t, report.Failure, "a stop is not a failure")
	require.Equal(t, "candidate-1", report.Lost)
	require.Empty(t, report.Candidates, "a lost candidate is not settled")
	require.Equal(t, "incomplete", report.Status)
	require.Equal(t, binds+1, r.binder.binds, "no Run is dispatched after the stop")
	require.Equal(t, 3, report.Runs, "the Run that closed after the stop is counted")
}

// A rerun that cannot release is a failure the report names, never a class.
func TestReduceNamesARerunFailure(t *testing.T) {
	r := newReduction(t, nil, edit(0, "a"))
	subjectReruns, err := Rerun(t.Context(), r.binder, r.subject.Target())
	require.NoError(t, err)
	r.binder.release = errors.New("namespace still held")
	report, err := r.reducer.Reduce(t.Context(), subjectReruns)
	require.NoError(t, err)
	require.Contains(t, report.Failure, "namespace still held")
	require.False(t, report.Stopped)
	require.Equal(t, "incomplete", report.Status)
	require.Equal(t, 3, report.Runs, "the Run that closed before the release failed is counted")
}

// The same inputs and classes give the same decisions and the same report bytes.
func TestReduceIsDeterministic(t *testing.T) {
	render := func() []byte {
		r := newReduction(t, []attemptScript{nil, nil, nil, runIncomplete, nil, runSatisfied, runSatisfied},
			edit(2, "c"), fakeEdit{Edit: edit(1, "b").Edit, inapplicable: true}, edit(0, "a"))
		report, err := r.run(t.Context(), t)
		require.NoError(t, err)
		rendered, err := report.Render()
		require.NoError(t, err)
		return rendered
	}
	first, second := render(), render()
	require.True(t, bytes.Equal(first, second), "%s\n%s", first, second)
}
