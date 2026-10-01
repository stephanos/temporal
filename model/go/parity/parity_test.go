// Package parity compares what the Go Models compute with what the Lean Models compute, as dumped
// by lean/Dump.lean into testdata/lean. A difference fails with the first differing entry, so a
// mismatch in an order or a key spelling points at the rule that broke.
package parity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/nexuscaller"
	"go.temporal.io/server/model/go/standaloneactivity"
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/go/worker"
)

// leanDump reads one dump. The dumps are git-ignored and only a Lean build regenerates them, so a
// checkout without them skips the comparison; a dump missing from a present directory still fails.
func leanDump(t *testing.T, name string) []byte {
	t.Helper()
	dir := filepath.Join("testdata", "lean")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("no Lean dumps in %s; model/go/leandump/dump.sh writes them", dir)
	}
	encoded, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	return encoded
}

func readLean[T any](t *testing.T, name string) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(leanDump(t, name), &v))
	return v
}

// firstDifference names the first index at which two lists differ, for a readable failure.
func firstDifference[T comparable](want, got []T) string {
	for i := range min(len(want), len(got)) {
		if want[i] != got[i] {
			return fmt.Sprintf("index %d: lean %v, go %v", i, want[i], got[i])
		}
	}
	if len(want) != len(got) {
		return fmt.Sprintf("lengths differ: lean %d, go %d", len(want), len(got))
	}
	return ""
}

func requireSameList[T comparable](t *testing.T, what string, want, got []T) {
	t.Helper()
	if d := firstDifference(want, got); d != "" {
		t.Fatalf("%s: %s", what, d)
	}
}

func requireSameTable(t *testing.T, dump string, got *umpire.Table) {
	t.Helper()
	want := readLean[umpire.Table](t, dump)
	requireSameList(t, "states", want.States, got.States)
	requireSameList(t, "actions", want.Actions, got.Actions)
	requireSameList(t, "outcomes", want.Outcomes, got.Outcomes)
	requireSameList(t, "facts", want.Facts, got.Facts)
	requireSameList(t, "starts", want.Starts, got.Starts)
	requireSameList(t, "ends", want.Ends, got.Ends)
	requireSameList(t, "reachable", want.Reachable, got.Reachable)
	require.Len(t, got.Rows, len(want.Rows), "rows")
	for i := range want.Rows {
		w, g := want.Rows[i], got.Rows[i]
		g.Results = stripSteps(g.Results)
		require.Equal(t, w, g, "row %d", i)
	}
}

func stripSteps(results []umpire.Result) []umpire.Result {
	out := make([]umpire.Result, len(results))
	for i, r := range results {
		r.Step = nil
		out[i] = r
	}
	return out
}

func requireSameIDs(t *testing.T, dump string, got *umpire.Table) {
	t.Helper()
	want := readLean[umpire.IDs](t, dump)
	ids := got.IDs()
	require.Equal(t, want.Target, ids.Target)
	requireSameList(t, "state ids", want.States, ids.States)
	require.Equal(t, want.StateFields, ids.StateFields, "state field ids")
	requireSameList(t, "action ids", want.Actions, ids.Actions)
	requireSameList(t, "outcome ids", want.Outcomes, ids.Outcomes)
	requireSameList(t, "fact ids", want.Facts, ids.Facts)
}

func TestTables(t *testing.T) {
	for _, tc := range []struct {
		name  string
		model umpire.Model
	}{
		{"nexusProduct", nexuscaller.NexusProduct},
		{"nexusProtocol", nexuscaller.NexusProtocol},
		{"workerPolling", worker.PollingMachine},
		{"handlerWorker", nexuscaller.HandlerWorker},
		{"nexusCaller", nexuscaller.NexusCaller},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table, err := tc.model.Table()
			require.NoError(t, err)
			requireSameTable(t, "table-"+tc.name+".json", table)
			requireSameIDs(t, "ids-"+tc.name+".json", table)
		})
	}
}

func TestRefinement(t *testing.T) {
	want := readLean[struct {
		Rejected *string                `json:"rejected"`
		Rows     []umpire.RefinementRow `json:"rows"`
	}](t, "refinement-nexusProtocol.json")
	require.Nil(t, want.Rejected)
	got, err := nexuscaller.NexusProtocol.Refinement()
	require.NoError(t, err)
	require.Len(t, got.Rows, len(want.Rows))
	for i := range want.Rows {
		require.Equal(t, want.Rows[i], got.Rows[i], "refinement row %d", i)
	}
}

func TestQueries(t *testing.T) {
	queries := append(append([]*umpire.Query{}, nexuscaller.FunctionalQueries...),
		nexuscaller.TerminalHolds, nexuscaller.StoppedWorkerRepliesNothing)
	for _, q := range queries {
		t.Run(q.Name, func(t *testing.T) {
			want := readLean[struct {
				Outcome string        `json:"outcome"`
				Witness *umpire.Trace `json:"witness"`
			}](t, "query-"+q.Name+".json")
			got, err := q.Answer()
			require.NoError(t, err)
			require.Equal(t, want.Outcome, string(got.Outcome), got.Explanation)
			if want.Witness != nil {
				require.Equal(t, want.Witness, got.Witness)
			}
		})
	}
}

func TestExplorationTargets(t *testing.T) {
	want := readLean[[]umpire.CoverageTarget](t, "targets-nexusCallerExploration.json")
	got, err := nexuscaller.NexusCallerExploration.Targets()
	require.NoError(t, err)
	require.Len(t, got, len(want))
	for i := range want {
		require.Equal(t, want[i], got[i], "target %d", i)
	}
}

// The standalone activity's product machine is the only part of that Model Lean elaborates: its
// protocol machine has 288 states, past the elaborator's bound of 256.
func TestActivityProduct(t *testing.T) {
	table, err := standaloneactivity.ActivityProduct.Table()
	require.NoError(t, err)
	requireSameTable(t, "activity-table-activityProduct.json", table)
	requireSameIDs(t, "activity-ids-activityProduct.json", table)
}
