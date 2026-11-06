//go:build test_dep

package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"umpire.local/temporal/activityrelations"
)

// newNoveltyCollector captures a harness's actual invocations. When UMPIRE_NOVELTY_CAPTURE names
// a directory, the corpus and its report are written there as <harness>-corpus.jsonl and
// <harness>-report.json once the test ends.
func newNoveltyCollector(t *testing.T, harness string) *activityrelations.Collector {
	t.Helper()
	collector, err := activityrelations.NewCollector()
	require.NoError(t, err)
	t.Cleanup(func() {
		rows := collector.Rows()
		var corpus bytes.Buffer
		sum, err := activityrelations.WriteCorpus(&corpus, rows)
		require.NoError(t, err)
		report := collector.Count(rows, sum)
		total := report.Total
		require.Equal(t, total.Attempted, total.Excluded+total.RPCError+total.NotApplicable+total.Refused+total.RefusedProjection+total.Success,
			"every captured invocation is classified exactly once")
		encoded, err := json.MarshalIndent(report, "", "  ")
		require.NoError(t, err)
		t.Logf("novelty %s report:\n%s", harness, encoded)
		directory := os.Getenv("UMPIRE_NOVELTY_CAPTURE")
		if directory == "" {
			return
		}
		require.NoError(t, os.MkdirAll(directory, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(directory, harness+"-corpus.jsonl"), corpus.Bytes(), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(directory, harness+"-report.json"), append(encoded, '\n'), 0o600))
	})
	return collector
}

// TestNoveltyCorpusCounts recomputes a frozen corpus's counts with the same functions and inputs
// and requires the frozen report exactly. UMPIRE_NOVELTY_CORPUS names a captured corpus directory
// and its harness; without it the committed control corpus is used.
func TestNoveltyCorpusCounts(t *testing.T) {
	collector, err := activityrelations.NewCollector()
	require.NoError(t, err)
	corpusPath := filepath.Join("..", "model", "hosts", "go", "activityrelations", "testdata", "control-corpus.jsonl")
	reportPath := filepath.Join("..", "model", "hosts", "go", "activityrelations", "testdata", "control-report.json")
	if directory := os.Getenv("UMPIRE_NOVELTY_CORPUS"); directory != "" {
		harness := os.Getenv("UMPIRE_NOVELTY_HARNESS")
		require.NotEmpty(t, harness, "UMPIRE_NOVELTY_HARNESS names the captured harness")
		corpusPath = filepath.Join(directory, harness+"-corpus.jsonl")
		reportPath = filepath.Join(directory, harness+"-report.json")
	}
	file, err := os.Open(corpusPath)
	require.NoError(t, err)
	defer func() { require.NoError(t, file.Close()) }()
	rows, sum, err := activityrelations.ReadCorpus(file)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	recomputed, err := json.MarshalIndent(collector.Count(rows, sum), "", "  ")
	require.NoError(t, err)
	frozen, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	require.Equal(t, string(frozen), string(recomputed)+"\n")
}
