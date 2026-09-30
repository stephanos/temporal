package goir

// The Nexus caller Model, lifted from model/scala into ir/nexus-caller.json and interpreted here,
// against what the Lean Model computes, as dumped into model/go/parity/testdata/lean. Parity means the
// IR carries the Model whole: nothing of the Scala code is run to get these tables.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/umpire"
)

const irPath = "../ir/nexus-caller.json"

func machines(t *testing.T) map[string]*Machine {
	t.Helper()
	m, err := Load(irPath)
	require.NoError(t, err)
	built, err := Build(m)
	require.NoError(t, err)
	return built
}

func readLean[T any](t *testing.T, name string) T {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("..", "..", "go", "parity", "testdata", "lean", name))
	require.NoError(t, err)
	var v T
	require.NoError(t, json.Unmarshal(encoded, &v))
	return v
}

func TestTablesAndIDsEqualLean(t *testing.T) {
	built := machines(t)
	for dump, name := range map[string]string{"nexusProduct": "nexusProduct", "nexusProtocol": "nexusProtocol",
		"workerPolling": "polling", "handlerWorker": "handlerWorker"} {
		t.Run(dump, func(t *testing.T) {
			got := built[name].Table
			want := readLean[umpire.Table](t, "table-"+dump+".json")
			require.Equal(t, want.States, got.States, "states")
			require.Equal(t, want.Actions, got.Actions, "actions")
			require.Equal(t, want.Outcomes, got.Outcomes, "outcomes")
			require.Equal(t, want.Facts, got.Facts, "facts")
			require.Equal(t, want.Starts, got.Starts, "starts")
			require.Equal(t, want.Ends, got.Ends, "ends")
			require.Equal(t, want.Reachable, got.Reachable, "reachable")
			require.Len(t, got.Rows, len(want.Rows))
			for i := range want.Rows {
				require.Equal(t, want.Rows[i], got.Rows[i], "row %d", i)
			}
			ids := readLean[umpire.IDs](t, "ids-"+dump+".json")
			require.Equal(t, ids, got.IDs())
		})
	}
}

func TestRefinementEqualsLean(t *testing.T) {
	want := readLean[struct {
		Rejected *string                `json:"rejected"`
		Rows     []umpire.RefinementRow `json:"rows"`
	}](t, "refinement-nexusProtocol.json")
	require.Nil(t, want.Rejected)
	require.Equal(t, want.Rows, machines(t)["nexusProtocol"].Refinement)
}

// The Behavior Fingerprint of the protocol machine, computed by model/go's canonical encoding over the
// interpreted table, is the one the Lean Model and the checked-in Cases carry.
func TestTargetFingerprintEqualsLean(t *testing.T) {
	require.Equal(t, "sha256:b38647500819c04cd82e179972ed23c76609d69e99a424748596295ce067af14",
		machines(t)["nexusProtocol"].Table.TargetFingerprint())
}

func TestStuckStatesAndEvidence(t *testing.T) {
	built := machines(t)
	for _, name := range []string{"nexusProduct", "nexusProtocol", "polling", "handlerWorker"} {
		require.Empty(t, built[name].Table.Stuck, name)
	}
	require.Equal(t, [][2]string{{"nexusOperationScheduled", "nexusOperationScheduled"}, {"nexusOperationStarted", "nexusOperationStarted"},
		{"nexusOperationCompleted", "nexusOperationCompleted"}, {"nexusOperationFailed", "nexusOperationFailed"},
		{"nexusOperationCanceled", "nexusOperationCanceled"}, {"nexusOperationTimedOut", "nexusOperationTimedOut"},
		{"pendingAttempts", "pendingAttempts"}}, built["nexusProtocol"].Table.Evidence)
}
