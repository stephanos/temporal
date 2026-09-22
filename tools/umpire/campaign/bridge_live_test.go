package campaign

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
)

// bridgeExecutable is the bridge the model package builds; `make umpire-check-exploration-bridge`
// builds it before the Lean and Go proofs run, and a checkout without it skips.
func bridgeExecutable(t *testing.T) (executable, modelRoot string) {
	t.Helper()
	modelRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "model"))
	require.NoError(t, err)
	executable = filepath.Join(modelRoot, ".lake", "build", "bin", "umpire-explore")
	if _, err := os.Stat(executable); err != nil {
		t.Skipf("exploration bridge is not built at %s: %v", executable, err)
	}
	return executable, modelRoot
}

// The real bridge over the caller Model's exploratory set: initialize, one candidate whose Case
// decodes as a Case of the identity the frame names, and finish, with the client's guard holding
// in between.
func TestLiveBridgeHandsOutOneDecodableCandidateAtATime(t *testing.T) {
	executable, modelRoot := bridgeExecutable(t)
	var stderr bytes.Buffer
	bridge, err := Start(t.Context(), Options{Executable: executable, Dir: modelRoot, Stderr: &stderr})
	require.NoError(t, err)
	t.Cleanup(func() { _ = bridge.Close() })

	opened, err := bridge.Initialize(t.Context(), "nexusCallerExploration", "live-bridge")
	require.NoError(t, err)
	require.Equal(t, "four", opened.Budget)
	require.Equal(t, 4, opened.Limits.Steps)
	require.NotEmpty(t, opened.Targets)

	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	require.NotNil(t, next.Candidate)
	require.NotEmpty(t, next.Skipped, "the first row target is unrealizable under the caller realization and is skipped")
	source, err := testpilot.DecodeCaseProtoJSON(next.Candidate.Case)
	require.NoError(t, err)
	require.Equal(t, next.Candidate.CaseID, source.GetCaseId())
	require.Contains(t, next.Candidate.Covers, next.Candidate.Target)
	_, err = bridge.Next(t.Context())
	require.ErrorIs(t, err, ErrCandidateOutstanding)

	detail := "no deployment"
	credited, err := bridge.Observe(t.Context(), next.Candidate.Identity, Result{PrepareRejected: &detail})
	require.NoError(t, err)
	require.Equal(t, "prepare-rejected", credited.Observation)
	require.Empty(t, credited.Credited)

	finished, err := bridge.Finish(t.Context(), "stopped")
	require.NoError(t, err)
	require.Equal(t, "stopped", finished.Status)
	require.Equal(t, len(next.Skipped)+1, finished.Summary.Selected)
	require.NoError(t, bridge.Close())
	require.Contains(t, stderr.String(), "candidate "+next.Candidate.Identity)
}
