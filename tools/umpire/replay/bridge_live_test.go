package replay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/campaign"
	"go.temporal.io/server/tools/umpire/internal/casefile"
)

// replayBridgeExecutable is the replay bridge the model package builds; `make
// umpire-check-replay-bridge` builds it, and a checkout without it skips.
func replayBridgeExecutable(t *testing.T) (executable, modelRoot string) {
	t.Helper()
	modelRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "model"))
	require.NoError(t, err)
	executable = filepath.Join(modelRoot, ".lake", "build", "bin", "umpire-replay-bridge")
	if _, err := os.Stat(executable); err != nil {
		t.Skipf("replay bridge is not built at %s: %v", executable, err)
	}
	return executable, modelRoot
}

// The real bridge admits the negative control by the SHA-256 of its fixture's canonical bytes,
// the same identity Admit derives, finds its one edit inapplicable, and reports it irreducible; a
// subject whose identity is not the fixture's is crossed.
func TestLiveReplayBridgeAdmitsTheControlByItsBytes(t *testing.T) {
	executable, modelRoot := replayBridgeExecutable(t)
	fixture, err := os.ReadFile(controlCasePath)
	require.NoError(t, err)
	canonical, err := casefile.Canonical(fixture)
	require.NoError(t, err)
	digest := sha256.Sum256(canonical)
	identity := hex.EncodeToString(digest[:])
	source, err := testpilot.DecodeCaseProtoJSON(canonical)
	require.NoError(t, err)

	var stderr bytes.Buffer
	bridge, err := StartBridge(t.Context(), campaign.Options{Executable: executable, Dir: modelRoot, Stderr: &stderr})
	require.NoError(t, err)
	t.Cleanup(func() { _ = bridge.Close() })
	admitted, err := bridge.Admit(t.Context(), "nexusCallerControl", "live-replay", Named{Query: "forgedCompletion"}, identity)
	require.NoError(t, err, "stderr: %s", stderr.String())
	require.Equal(t, source.GetCaseId(), admitted.CaseID)
	require.Len(t, admitted.Edits, 1)
	require.Equal(t, []Edit{{Edit: "dropPrefixStep 0", Index: 0, Action: admitted.Edits[0].Action}}, admitted.Edits)
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	require.True(t, next.Exhausted)
	require.Equal(t, "inapplicable", next.Skipped[0].Fate)
	finished, err := bridge.Finish(t.Context(), "")
	require.NoError(t, err)
	require.Equal(t, "irreducible", finished.Status)
	require.Equal(t, admitted.Subject, finished.Retained)
	require.NotNil(t, finished.Proposal)
	require.Equal(t, admitted.Subject, finished.Proposal.Digest)
	root := filepath.Join(t.TempDir(), "proposals")
	written := WriteProposal(root, finished.Proposal)
	require.Equal(t, ProposalWritten, written.Status, written.Error)
	require.Equal(t, filepath.Join(root, "nexusCallerControl-"+admitted.Subject+".lean"), written.Written)

	crossed, err := StartBridge(t.Context(), campaign.Options{Executable: executable, Dir: modelRoot, Stderr: &stderr})
	require.NoError(t, err)
	t.Cleanup(func() { _ = crossed.Close() })
	other := sha256.Sum256([]byte("another Case"))
	_, err = crossed.Admit(t.Context(), "nexusCallerControl", "live-replay", Named{Query: "forgedCompletion"}, hex.EncodeToString(other[:]))
	var crossedErr *CrossedError
	require.ErrorAs(t, err, &crossedErr)
}
