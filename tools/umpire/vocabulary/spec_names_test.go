package vocabulary_test

import (
	"os"
	"path/filepath"
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

	unresolved, err := leannames.Unresolved(
		index, names, "UMPIRE4_SPEC.md", filepath.Join(repositoryRoot, ".flow", "specs"))
	require.NoError(t, err)
	require.Empty(t, unresolved)
}
