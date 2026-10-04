package testpilot

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/lower"
)

// UMPIRE_CASE_NAME_GOLDENS=write rewrites the generated Case names golden; a changed golden is a
// changed set of shard units, reviewed as such.
const caseNameGoldensVariable = "UMPIRE_CASE_NAME_GOLDENS"

// Every lowered Case runs as TestTestpilotGeneratedCases/<name>, the depth-2 name CI shards by and
// the salt optimizer times, so the names are pinned: each is its file's stem, unique, a single
// path segment, and the sorted set is the committed golden.
func TestGeneratedCaseNames(t *testing.T) {
	entries, err := GeneratedCases(filepath.Join("..", "..", "..", "model", "cases"))
	require.NoError(t, err)
	var names []string
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.Standing != lower.Lowered {
			continue
		}
		name := GeneratedCaseName(entry)
		require.Equal(t, strings.TrimSuffix(entry.File, "-case.json"), name)
		require.NotEmpty(t, name, entry.File)
		require.NotContains(t, name, "/", "a / would split the depth-2 test name")
		require.False(t, strings.ContainsFunc(name, unicode.IsSpace), "%q carries whitespace", name)
		require.False(t, seen[name], "%q names two lowered Cases", name)
		seen[name] = true
		names = append(names, name)
	}
	slices.Sort(names)
	listed := strings.Join(names, "\n") + "\n"

	path := filepath.Join("testdata", "generated-case-names.txt")
	if os.Getenv(caseNameGoldensVariable) == "write" {
		require.NoError(t, os.WriteFile(path, []byte(listed), 0o644))
	}
	golden, err := os.ReadFile(path)
	require.NoError(t, err, "write the golden with %s=write", caseNameGoldensVariable)
	require.Equal(t, string(golden), listed,
		"a renamed, added or removed Case moves a shard unit (TestTestpilotGeneratedCases/<name> is the depth-2 name CI shards and times); rewrite the golden with %s=write in the same change",
		caseNameGoldensVariable)
}
