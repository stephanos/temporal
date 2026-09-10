package leannames_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/leannames"
)

func writeLean(t *testing.T, root, relativePath, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relativePath))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestBuildIndexesModulesNamespacesAndDeclarations(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeLean(t, root, "Umpire/Model/Check.lean", `import Umpire.Core

namespace Umpire

/-- The checked model. -/
structure CheckedModel where
  identity : String
  vocabulary : String

private def checkModel (draft : DraftModel) : Except String CheckedModel := by sorry

namespace Table

@[simp] theorem rows_eq_rows : True := trivial

end Table

end Umpire
`)
	writeLean(t, root, "Umpire/Core.lean", "namespace Umpire\ninductive LimitUnit where\n  | steps\n  | actions\nend Umpire\n")
	// A Lake build tree holds copies plus dependency sources; indexing it would claim
	// names this model does not define.
	writeLean(t, root, ".lake/build/lib/Elsewhere/Ghost.lean", "namespace Elsewhere\ndef ghost := 1\nend Elsewhere\n")

	index, err := leannames.Build(root)
	require.NoError(t, err)

	for _, name := range []string{
		"Umpire.Model.Check",             // a module derived from its path
		"Umpire.Model",                   // the directory above it
		"Umpire.CheckedModel",            // a structure inside `namespace Umpire`
		"Umpire.checkModel",              // a `private def` inside `namespace Umpire`
		"Umpire.Table.rows_eq_rows",      // a theorem inside a nested namespace
		"Umpire.CheckedModel.identity",   // a structure field
		"Umpire.CheckedModel.vocabulary", // its sibling
		"Umpire.LimitUnit.steps",         // an inductive constructor
		"Umpire.LimitUnit.actions",       // its sibling
	} {
		require.True(t, index.Resolve(name), "expected %s to resolve", name)
	}

	for _, name := range []string{
		"Umpire.Model.Missing",
		"Umpire.checkedModel",
		"Umpire.CheckedModel.identity.deeper",
		"Elsewhere.Ghost",
		"Elsewhere.ghost",
	} {
		require.False(t, index.Resolve(name), "expected %s not to resolve", name)
	}

	modules, namespaces, declarations := index.Size()
	require.Equal(t, 2, modules)
	require.Positive(t, namespaces)
	require.Positive(t, declarations)
}

func TestBuildRejectsAMissingModelRoot(t *testing.T) {
	t.Parallel()

	_, err := leannames.Build(filepath.Join(t.TempDir(), "absent"))
	require.Error(t, err)
}

func TestBuildIgnoresDeclarationKeywordsInsideABody(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeLean(t, root, "Umpire/Fixture.lean", "namespace Umpire\n\n"+
		"def outer : Nat :=\n"+
		"  let inner := 1\n"+
		"  inner\n\n"+
		"macro \"keyword\" name:ident : command => `(command| def bodyDef := 1)\n\n"+
		"end Umpire\n")

	index, err := leannames.Build(root)
	require.NoError(t, err)
	require.True(t, index.Resolve("Umpire.outer"))
	// The macro's leading token is a string literal, so it names no declaration, and
	// the quoted `def` inside its body is indented past column zero.
	require.False(t, index.Resolve("Umpire.keyword"))
	require.False(t, index.Resolve("Umpire.bodyDef"))
}
