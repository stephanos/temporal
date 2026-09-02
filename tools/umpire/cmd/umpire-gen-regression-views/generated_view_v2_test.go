package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedViewExtractorAcceptsCanonicalV2Artifact(t *testing.T) {
	repositoryRoot := testRepositoryRoot(t)
	entry := manifestEntry{
		Identity:           "switch.query.exact-action",
		FixturePath:        "model/Umpire/Examples/testdata/switch-experiment-spec.json",
		GoOutputPath:       "tools/umpire/regression/switch_generated_view_test.go",
		MarkdownOutputPath: "model/Umpire/Examples/Generated/Switch.md",
	}
	encoded, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(entry.FixturePath)))
	require.NoError(t, err)

	view, err := extractGeneratedView(entry, encoded, filepath.Join(repositoryRoot, "model"))
	require.NoError(t, err)
	require.Equal(t, "umpire-experiment/v2", view.Format)
	require.Equal(t, "switch.query.exact-action", view.Identity)
	require.Equal(t, "sha256:ac3fde668a79ff0433106e28f8ec9579a36f9f7d0ab09845d01b563289b560fd", view.ArtifactChecksum)
}
