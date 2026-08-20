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
		"campaign",
		"canary",
		"compiler",
		"environment",
		"evidence",
		"explore",
		"fault",
		"migration",
		"model",
		"participant",
		"process",
		"profile",
		"protocol",
		"qualification",
		"regress",
		"replay",
		"runner",
		"runtime",
		"temporal",
		"umpire3test",
	} {
		require.DirExists(t, filepath.Join(root, path))
	}

	require.FileExists(t, filepath.Join(root, "model", "lean-toolchain"))
	require.FileExists(t, filepath.Join(root, "model", "lakefile.toml"))
	require.FileExists(t, filepath.Join(root, "model", "mise.toml"))
	require.FileExists(t, filepath.Join(root, "cmd", "umpire3", "main.go"))
}

func TestGoPackagesDoNotImportPreviousUmpires(t *testing.T) {
	violations, err := findForbiddenImports(".")
	require.NoError(t, err)
	for _, path := range []string{"../umpire3_test.go", "../umpire3_probe_test.go", "../umpire3_regress_test.go"} {
		rootViolations, rootErr := findForbiddenImports(path)
		require.NoError(t, rootErr)
		violations = append(violations, rootViolations...)
	}
	require.Empty(t, violations)
}

func TestRetainedUmpire2RootTestsDoNotImportUmpire3(t *testing.T) {
	var violations []string
	for _, path := range []string{"../umpire2_test.go", "../umpire2_probe_test.go", "../umpire2_regress_test.go"} {
		rootViolations, err := findImportsWithPrefixes(path, []string{"go.temporal.io/server/tests/umpire3"})
		require.NoError(t, err)
		violations = append(violations, rootViolations...)
	}
	require.Empty(t, violations)
}

func TestRootUmpireTestsUseIndependentSideBySideFiles(t *testing.T) {
	testsRoot := ".."
	for _, name := range []string{
		"umpire2_test.go",
		"umpire2_probe_test.go",
		"umpire2_regress_test.go",
		"umpire3_test.go",
		"umpire3_probe_test.go",
		"umpire3_regress_test.go",
	} {
		require.FileExists(t, filepath.Join(testsRoot, name))
	}
	for _, name := range []string{"umpire_test.go", "umpire_probe_test.go", "umpire_regress_test.go"} {
		_, err := os.Stat(filepath.Join(testsRoot, name))
		require.ErrorIs(t, err, os.ErrNotExist)
	}
}

func TestBlackBoxProfileDoesNotImportServerObservationInternals(t *testing.T) {
	forbidden := []string{"go.temporal.io/server/service", "go.temporal.io/server/common/persistence",
		"go.temporal.io/server/api/historyservice", "go.temporal.io/server/api/matchingservice"}
	violations, err := findImportsWithPrefixes("profile", forbidden)
	require.NoError(t, err)
	require.Empty(t, violations)
}

func TestFoundationalPackageImportDirection(t *testing.T) {
	allowed := map[string][]string{
		"protocol":    nil,
		"evidence":    nil,
		"process":     nil,
		"compiler":    {"protocol"},
		"environment": {"protocol"},
		"fault":       {"protocol"},
		"participant": {"protocol"},
	}
	for packageName, dependencies := range allowed {
		t.Run(packageName, func(t *testing.T) {
			violations, err := findUnexpectedUmpire3Imports(packageName, dependencies)
			require.NoError(t, err)
			require.Empty(t, violations)
		})
	}
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
  "formatVersion": "umpire3/v2",
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
	return findImportsWithPrefixes(root, forbidden)
}

func findImportsWithPrefixes(root string, forbidden []string) ([]string, error) {
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

func findUnexpectedUmpire3Imports(root string, allowed []string) ([]string, error) {
	const prefix = "go.temporal.io/server/tests/umpire3/"
	allowedImports := make(map[string]struct{}, len(allowed))
	for _, dependency := range allowed {
		allowedImports[prefix+dependency] = struct{}{}
	}
	var violations []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
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
			if !strings.HasPrefix(importPath, prefix) {
				continue
			}
			if _, ok := allowedImports[importPath]; ok {
				continue
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			violations = append(violations, fmt.Sprintf("%s imports %s", relative, importPath))
		}
		return nil
	})
	return violations, err
}
