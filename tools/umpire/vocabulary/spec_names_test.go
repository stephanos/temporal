package vocabulary_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/leannames"
)

// TestUmpireSpecNamesResolveAgainstTheModelTree checks the mechanical half of the
// glossary contract: every dotted Lean name the specification cites in backticks names
// a module, a namespace, or a declaration that exists, unless the rule that cites it is
// marked planned and the spec that owns delivering it is still open. Whether each term
// is defined once and used consistently stays a review judgment.
func TestUmpireSpecNamesResolveAgainstTheModelTree(t *testing.T) {
	t.Parallel()

	repositoryRoot := checkoutRoot(t)

	index, err := leannames.Build(filepath.Join(repositoryRoot, "model"))
	require.NoError(t, err)
	modules, namespaces, declarations := index.Size()
	// A truncated walk would report that every name resolved, so the index has to look
	// like the real tree before its answers mean anything.
	require.Greater(t, modules, 200, "model tree index looks truncated")
	require.Positive(t, namespaces)
	require.Greater(t, declarations, 1000, "model tree index looks truncated")

	document, err := os.ReadFile(filepath.Join(repositoryRoot, ".plans", "UMPIRE4_SPEC.md"))
	require.NoError(t, err)

	names := leannames.ExtractSpecNames(string(document))
	require.NotEmpty(t, names)

	specsDirectory := filepath.Join(repositoryRoot, ".flow", "specs")
	openSpec := map[string]bool{}
	var unresolved []string
	for _, name := range names {
		if name.PlannedSpec != "" {
			open, ok := openSpec[name.PlannedSpec]
			if !ok {
				open, err = leannames.SpecIsOpen(specsDirectory, name.PlannedSpec)
				require.NoError(t, err)
				openSpec[name.PlannedSpec] = open
			}
			if !open {
				unresolved = append(unresolved, fmt.Sprintf(
					"UMPIRE4_SPEC.md:%d: %s is marked planned under %s, which is not open",
					name.Line, name.Name, name.PlannedSpec))
			}
			continue
		}
		if !index.Resolve(name.Name) {
			unresolved = append(unresolved, fmt.Sprintf(
				"UMPIRE4_SPEC.md:%d: %s names no module, namespace, or declaration",
				name.Line, name.Name))
		}
	}
	slices.Sort(unresolved)
	require.Empty(t, unresolved)
}

func checkoutRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}
