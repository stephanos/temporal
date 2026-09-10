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
	require.Equal(t, "sha256:9fa327849c3d0a48290bb16fec73a00be4cc1b6234862ee506a547f29b6d3b12", view.ArtifactChecksum)
}
