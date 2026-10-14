package replay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Each status of a proposal: none for a result that proposes nothing, not-compiled with the
// bridge's reason, compiled without a root, written under one, and write-failed for a path that
// escapes the root or a destination that exists, which is never replaced.
func TestWriteProposalReportsEveryStatus(t *testing.T) {
	require.Equal(t, ProposalReport{Status: ProposalNone}, WriteProposal(t.TempDir(), nil))

	failed := WriteProposal(t.TempDir(), &BridgeProposal{Digest: "d", Error: "nonFoundResult: no trace"})
	require.Equal(t, ProposalNotCompiled, failed.Status)
	require.Equal(t, "nonFoundResult: no trace", failed.Error)

	sha := "sha256:abc"
	proposal := &BridgeProposal{Digest: "d", SHA256: &sha, Path: "set-d.lean", Source: "-- proposal"}
	compiled := WriteProposal("", proposal)
	require.Equal(t, ProposalCompiled, compiled.Status)
	require.Empty(t, compiled.Written)

	temporary, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	root := filepath.Join(temporary, "proposals")
	written := WriteProposal(root, proposal)
	require.Equal(t, ProposalWritten, written.Status)
	require.Equal(t, filepath.Join(root, "set-d.lean"), written.Written)
	source, err := os.ReadFile(written.Written)
	require.NoError(t, err)
	require.Equal(t, "-- proposal", string(source))

	again := WriteProposal(root, &BridgeProposal{Digest: "d", SHA256: &sha, Path: "set-d.lean", Source: "-- replaced"})
	require.Equal(t, ProposalWriteFailed, again.Status)
	require.Contains(t, again.Error, "file exists")
	source, err = os.ReadFile(written.Written)
	require.NoError(t, err)
	require.Equal(t, "-- proposal", string(source), "an existing proposal is never replaced")

	escaping := WriteProposal(root, &BridgeProposal{Digest: "e", SHA256: &sha, Path: "../escaped.lean", Source: "x"})
	require.Equal(t, ProposalWriteFailed, escaping.Status)
	require.NoFileExists(t, filepath.Join(filepath.Dir(root), "escaped.lean"))
}
