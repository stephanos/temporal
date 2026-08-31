package runevaluation

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoundedLiveNexusNegativeControl(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	requireNegativeControlMutationAndStatusControls(t, repositoryRoot)
	requirePairedLiveNexusNegativeControl(t, repositoryRoot)
}
