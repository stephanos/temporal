package protocolmigration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	repositoryRoot      = "../../../../.."
	baselineRoot        = "testdata/baseline"
	baselineFixtureRoot = baselineRoot + "/fixtures"
)

func loadBaseline(t *testing.T) *Baseline {
	t.Helper()
	baseline, err := LoadBaseline(filepath.Join(baselineRoot, "descriptors.binpb"))
	require.NoError(t, err)
	return baseline
}

func readFixturePair(t *testing.T, fixture string) (baseline, regenerated []byte) {
	t.Helper()
	baseline, err := os.ReadFile(filepath.Join(baselineFixtureRoot, filepath.FromSlash(fixture)))
	require.NoError(t, err)
	regenerated, err = os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(fixture)))
	require.NoError(t, err)
	return baseline, regenerated
}

// TestBaselineFixturesMapToRegeneratedFixtures is the fn-87 no-Verdict-change oracle: every
// pre-migration fixture, strictly decoded through the frozen snapshot and mapped under Declared,
// equals the fixture its generator writes today.
func TestBaselineFixturesMapToRegeneratedFixtures(t *testing.T) {
	t.Parallel()

	baseline := loadBaseline(t)
	fixtures, err := PairedFixtures(baselineFixtureRoot, repositoryRoot, Added)
	require.NoError(t, err)

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			t.Parallel()

			old, regenerated := readFixturePair(t, fixture)
			require.NoError(t, baseline.Check(fixture, old, regenerated, Declared))
		})
	}
}
