package umpire3_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIndependentLayout(t *testing.T) {
	root := "."
	for _, path := range []string{
		"artifact",
		"environment",
		"model",
		"protocol",
		"runtime",
		"temporal",
	} {
		require.DirExists(t, filepath.Join(root, path))
	}

	require.FileExists(t, filepath.Join(root, "model", "lean-toolchain"))
	require.FileExists(t, filepath.Join(root, "model", "lakefile.toml"))
	require.FileExists(t, filepath.Join(root, "model", "mise.toml"))
}

func TestGoPackagesDoNotImportPreviousUmpires(t *testing.T) {
	violations, err := findForbiddenImports(".")
	require.NoError(t, err)
	require.Empty(t, violations)
}

func TestDependencyGuardRejectsPreviousUmpire(t *testing.T) {
	fixture := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fixture, "bad.go"), []byte(`package bad

import _ "go.temporal.io/server/tests/umpire2"
`), 0o600))

	violations, err := findForbiddenImports(fixture)
	require.NoError(t, err)
	require.Equal(t, []string{"bad.go imports go.temporal.io/server/tests/umpire2"}, violations)
}

func TestProofHygieneRejectsAdmissions(t *testing.T) {
	fixture := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fixture, "Bad.lean"), []byte("theorem bad : True := by sorry\n"), 0o600))

	command := exec.Command("sh", filepath.Join("model", "check-proof-hygiene.sh"), fixture)
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "Bad.lean")
}

func TestProofHygieneAcceptsCheckedDefinitions(t *testing.T) {
	fixture := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fixture, "Good.lean"), []byte("theorem good : True := by trivial\n"), 0o600))

	command := exec.Command("sh", filepath.Join("model", "check-proof-hygiene.sh"), fixture)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestManifestCommand(t *testing.T) {
	command := exec.Command("go", "run", "-tags", "test_dep", "./cmd/umpire3-manifest", "-lean-version", "4.33.0")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.JSONEq(t, `{
  "formatVersion": "umpire3/v1",
  "toolchain": {
    "lean": "4.33.0"
  }
}`, string(output))
}

func findForbiddenImports(root string) ([]string, error) {
	forbidden := []string{
		"go.temporal.io/server/common/testing/umpire",
		"go.temporal.io/server/tests/umpire1",
		"go.temporal.io/server/tests/umpire2",
	}

	var violations []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			for _, prefix := range forbidden {
				if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
					violations = append(violations, fmt.Sprintf("%s imports %s", filepath.Base(path), importPath))
				}
			}
		}
		return nil
	})
	return violations, err
}
