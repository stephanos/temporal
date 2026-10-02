package model

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/golden"
)

func TestCheckerAndProducerHaveOneLiveOwner(t *testing.T) {
	root, err := golden.Root()
	require.NoError(t, err)
	base := filepath.Join(root, "tools", "umpire")
	require.NoError(t, filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range parsed.Imports {
			name, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			require.NotEqual(t, "go.temporal.io/server/model/go/"+"umpire", name, path)
			require.NotEqual(t, "go.temporal.io/server/model/go/caseproducer", name, path)
			if strings.HasSuffix(name, "/internal/checker") {
				rel, err := filepath.Rel(filepath.Join(base, "model"), path)
				if err != nil {
					return err
				}
				require.True(t, filepath.Dir(rel) == "." || strings.HasPrefix(rel, "internal/checker/"), "%s imports the private checker", path)
			}
			if strings.HasSuffix(name, "/internal/producer") {
				require.True(t, strings.HasPrefix(path, filepath.Join(base, "lower")+string(filepath.Separator)), "%s imports the private producer", path)
			}
		}
		return nil
	}))
}

func modelImportProblem(file, imported string) string {
	const module = "go.temporal.io/server/"
	if !strings.HasPrefix(imported, module) {
		return ""
	}
	name := strings.TrimPrefix(imported, module)
	if strings.HasPrefix(name, "model0/") || strings.HasPrefix(name, "tools/umpire0/") || strings.HasPrefix(name, "model/") {
		return "archive or retired model import"
	}
	if strings.HasPrefix(file, "tools/umpire/") && (strings.HasPrefix(name, "tests/") || strings.HasPrefix(name, "tools/canary")) {
		return "tooling imports a consumer"
	}
	if !strings.HasPrefix(file, "tools/umpire/") {
		return ""
	}
	part := strings.TrimPrefix(file, "tools/umpire/")
	owner := strings.Split(part, "/")[0]
	test := strings.HasSuffix(file, "_test.go")
	if strings.HasPrefix(name, "tools/umpire/model/internal/checker") && owner != "model" {
		return "checker is private to reader"
	}
	if strings.HasPrefix(name, "tools/umpire/lower/internal/producer") && owner != "lower" {
		return "producer is private to lowering"
	}
	if name == "tools/umpire/internal/golden" && !test {
		return "golden support is test-only"
	}
	if strings.HasPrefix(name, "common/testing/testpilot/") && owner != "cmd" {
		helper := strings.TrimPrefix(name, "common/testing/testpilot/")
		allowed := false
		switch owner {
		case "explore":
			allowed = helper == "campaign" || helper == "replay" || helper == "recordedrun"
		case "internal":
			allowed = strings.HasPrefix(part, "internal/cli/") && helper == "publish"
		case "lower":
			allowed = test && !strings.Contains(part, "/internal/") && (helper == "temporal" || helper == "recordedrun")
		case "conformance":
			allowed = test && (helper == "temporal" || helper == "temporal/control")
		default:
			return "unapproved Testpilot helper dependency"
		}
		if !allowed {
			return "unapproved Testpilot helper dependency"
		}
	}
	if owner == "model" && (strings.HasPrefix(name, "common/testing/testpilot") || strings.HasPrefix(name, "api/testpilot/")) {
		return "reader imports Testpilot"
	}
	if strings.HasPrefix(part, "model/internal/checker/") && strings.HasPrefix(name, "tools/umpire/") && !strings.HasPrefix(name, "tools/umpire/model/internal/checker") {
		return "checker imports a higher layer"
	}
	if strings.HasPrefix(name, "tools/umpire/") {
		dependency := strings.TrimPrefix(name, "tools/umpire/")
		dependency = strings.Split(dependency, "/")[0]
		if dependency == owner || owner == "cmd" || (test && name == "tools/umpire/internal/golden") {
			return ""
		}
		allowed := map[string][]string{"lower": {"model"}, "conformance": {"model"}, "export": {"model"}, "explore": {"model", "lower"}, "model": {}}
		if test {
			if owner == "lower" && !strings.Contains(part, "/internal/") {
				allowed[owner] = append(allowed[owner], "explore", "conformance")
			}
			if owner == "conformance" {
				allowed[owner] = append(allowed[owner], "lower")
			}
		}
		for _, edge := range allowed[owner] {
			if dependency == edge {
				return ""
			}
		}
		return "unapproved model dependency"
	}
	if owner == "export" && (strings.HasPrefix(name, "common/testing/testpilot") || strings.HasPrefix(name, "api/testpilot/")) {
		return "export imports Testpilot"
	}
	return ""
}

func TestLiveModelDependencyGraph(t *testing.T) {
	root, err := golden.Root()
	require.NoError(t, err)
	for _, archive := range []string{"model0", "tools/umpire0"} {
		require.FileExists(t, filepath.Join(root, archive, "go.mod"))
	}
	require.NoDirExists(t, filepath.Join(root, "model", "scalav2"))
	scanned := 0
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if rel != "." && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "gen" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			if rel != "." {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		require.False(t, strings.HasPrefix(filepath.ToSlash(rel), "model/"), "Go source inside model: %s", rel)
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		scanned++
		for _, item := range parsed.Imports {
			imported, err := strconv.Unquote(item.Path.Value)
			if err != nil {
				return err
			}
			require.Empty(t, modelImportProblem(filepath.ToSlash(rel), imported), "%s imports %s", rel, imported)
		}
		return nil
	}))
	require.Greater(t, scanned, 100)
}

func TestModelDependencyGraphRejectsCrossedOwners(t *testing.T) {
	for _, test := range []struct {
		file, dependency string
		allowed          bool
	}{
		{"tools/umpire/model/load.go", "api/modelir/v1", true},
		{"tools/umpire/model/load.go", "api/testpilot/v1", false},
		{"tools/umpire/model/load_test.go", "common/testing/testpilot", false},
		{"tools/umpire/model/internal/checker/table.go", "tools/umpire/model", false},
		{"tools/umpire/lower/internal/producer/producer.go", "tools/umpire/model/internal/checker", false},
		{"tools/umpire/lower/internal/producer/producer.go", "tools/umpire/model", true},
		{"tools/umpire/lower/lower.go", "tools/umpire/explore", false},
		{"tools/umpire/lower/migration_test.go", "tools/umpire/explore", true},
		{"tools/umpire/conformance/conformance.go", "tools/umpire/lower", false},
		{"tools/umpire/export/export.go", "common/testing/testpilot", false},
		{"tools/umpire/lower/lower.go", "common/testing/testpilot/campaign", false},
		{"tools/umpire/lower/internal/producer/producer_test.go", "common/testing/testpilot/recordedrun", false},
		{"tools/umpire/explore/explore.go", "common/testing/testpilot/temporal", false},
		{"tools/umpire/internal/cli/cli.go", "tools/umpire/internal/golden", false},
		{"tools/umpire/explore/explore.go", "tools/umpire/lower", true},
		{"tools/umpire/cmd/x/main_test.go", "tests/testcore/testpilot", false},
		{"tools/umpire/model/load.go", "model0/go/umpire", false},
		{"tests/testcore/testpilot/case.go", "tools/umpire0/recordedrun", false},
	} {
		t.Run(test.file+"->"+test.dependency, func(t *testing.T) {
			require.Equal(t, test.allowed, modelImportProblem(test.file, "go.temporal.io/server/"+test.dependency) == "")
		})
	}
}
