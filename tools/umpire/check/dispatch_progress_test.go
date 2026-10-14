package check

// The Temporal example of a composition's bounded progress (model/temporal/features/activity/
// standalone/system/DispatchProgress.scala): over the activity restricted to dispatch and a worker
// that never stops, a dispatchable attempt starts within one composed step, the sync counted once;
// over a worker that may stop, the same claim is refuted by a deadlock whose witness ends with the
// activity's poll locally enabled and the sync disabled. The safety promise holds over both.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
	"go.temporal.io/server/tools/umpire/ir"
)

// dispatchProgressIR is the checked-in IR the example lifts into; UMPIRE_MODEL_IR names another
// directory of lifted IR files, such as one lifted before the feature's generated IR is regenerated.
func dispatchProgressIR(t *testing.T) *umpirespb.Model {
	path := ir.ModelPath("../../..", "activity-standalone.json")
	if dir := os.Getenv("UMPIRE_MODEL_IR"); dir != "" {
		path = filepath.Join(dir, "activity-standalone.json")
	}
	m, err := ir.Load(path)
	require.NoError(t, err)
	return m
}

func TestDispatchProgressHoldsWithAServingWorkerAndNotWithAStoppingOne(t *testing.T) {
	m := dispatchProgressIR(t)
	c := newChecker(m, DefaultScope, m)
	receipts := map[string]Receipt{}
	for _, p := range m.GetProgress() {
		if p.GetMachine() == "servedDispatch" || p.GetMachine() == "unservedDispatch" {
			for _, x := range c.progress(p) {
				receipts[receiptKey(x)] = x
			}
		}
	}
	for _, q := range m.GetQueries() {
		if owner := q.GetScenario().GetMachine(); owner == "servedDispatch" || owner == "unservedDispatch" {
			x := c.query(q)
			require.Equal(t, Verified, x.Kind, "%s: %s", receiptKey(x), x.Explanation)
			require.True(t, x.Exercised, receiptKey(x))
		}
	}

	for _, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind, umpire.DeadlineKind} {
		x, ok := receipts["progress servedDispatch startsWithinOneStep "+string(part)]
		require.True(t, ok, "no %s receipt among %v", part, receipts)
		require.Equal(t, Verified, x.Kind, x.Explanation)
		require.True(t, x.Exercised, "the source is reachable and not already the destination")
		require.Contains(t, x.Assumptions, "servedDispatchRuns")
	}

	dead := receipts["progress unservedDispatch startsWithinOneStep "+string(umpire.DeadlockKind)]
	require.Equal(t, Counterexample, dead.Kind, dead.Explanation)
	require.Contains(t, dead.Assumptions, "unservedDispatchRuns")
	require.NotEmpty(t, dead.Witness.Steps)
	require.Equal(t, "worker_stop", taken(dead.Witness)[len(dead.Witness.Steps)-1],
		"the worker's own stop is the composed step that leaves the sync disabled")
	missed := receipts["progress unservedDispatch startsWithinOneStep "+string(umpire.DeadlineKind)]
	require.Equal(t, Counterexample, missed.Kind, missed.Explanation)

	_, subject, err := bind(m, DefaultScope).progress(progressNamed(m, "unservedDispatch", "startsWithinOneStep"))
	require.NoError(t, err)
	require.NoError(t, subject.table.Replay(dead.Witness))
	last := dead.Witness.Steps[len(dead.Witness.Steps)-1].State.Value
	state, err := subject.state(last)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(state.Type, ".StandaloneActivityState"), state.Type)
	require.Len(t, state.Fields, 2, "the witness explains both members' states")
	require.Contains(t, state.Fields[1].Key(), "stopped")
}

func progressNamed(m *umpirespb.Model, owner, name string) *umpirespb.Progress {
	for _, p := range m.GetProgress() {
		if p.GetMachine() == owner && p.GetName() == name {
			return p
		}
	}
	return nil
}
