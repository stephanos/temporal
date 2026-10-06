package ir

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckerAndProducerHaveOneLiveOwner(t *testing.T) {
	root, err := filepath.Abs(repoRoot)
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
			if strings.HasSuffix(name, "/internal/engine") {
				rel, err := filepath.Rel(base, path)
				if err != nil {
					return err
				}
				rel = filepath.ToSlash(rel)
				require.True(t, readerPackages[filepath.Dir(rel)] || strings.HasPrefix(rel, "internal/engine/"), "%s imports the private engine", path)
			}
			if strings.HasSuffix(name, "/internal/producer") {
				require.True(t, strings.HasPrefix(path, filepath.Join(base, "lower")+string(filepath.Separator)), "%s imports the private producer", path)
			}
		}
		return nil
	}))
}

func TestQuintOwnsBackendExport(t *testing.T) {
	for _, file := range []string{"p" + ".go", "p" + "_test.go"} {
		require.NoFileExists(t, filepath.Join(repoRoot, "tools", "umpire", "export", file))
	}
	for _, file := range []string{"Makefile", "tools/umpire/export/tool.go", "tools/umpire/export/tools_test.go"} {
		text, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(file)))
		require.NoError(t, err)
		for _, retired := range []string{"UMPIRE_" + "P", "UMPIRE_BACKEND_" + "TOOLS", "DOT" + "NET_ROOT"} {
			require.NotRegexp(t, `\b`+regexp.QuoteMeta(retired)+`\b`, string(text), file)
		}
	}
}

// readerPackages are the reader's packages, which the rules below call "model", the package they were
// split from (fn-124.8).
var readerPackages = map[string]bool{"ir": true, "interp": true, "check": true, "realization": true}

// modelImportProblem says why file may not import imported, or "" when it may. external is whether
// file declares an external test package, the only place the lowerer may reach the modules that
// consume its Cases.
func modelImportProblem(file string, external bool, imported string) string {
	const module = "go.temporal.io/server/"
	test := strings.HasSuffix(file, "_test.go")
	if !strings.HasPrefix(imported, module) {
		return ""
	}
	name := strings.TrimPrefix(imported, module)
	if strings.HasPrefix(name, "model/") {
		return "retired model import"
	}
	if strings.HasPrefix(file, "tools/umpire/") && (strings.HasPrefix(name, "tests/") || strings.HasPrefix(name, "tools/canary")) {
		return "tooling imports a consumer"
	}
	if !strings.HasPrefix(file, "tools/umpire/") {
		return ""
	}
	part := strings.TrimPrefix(file, "tools/umpire/")
	owner := strings.Split(part, "/")[0]
	if readerPackages[owner] {
		owner = "model"
	}
	if strings.HasPrefix(name, "tools/umpire/internal/engine") && owner != "model" && !strings.HasPrefix(part, "internal/engine/") {
		return "engine is private to reader"
	}
	if strings.HasPrefix(name, "tools/umpire/lower/internal/producer") && owner != "lower" {
		return "producer is private to lowering"
	}
	if name == "common/testing/testpilot" && owner == "internal" {
		return "unapproved Testpilot helper dependency"
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
			// The lowerer's own tests admit a lowered Case under the Driver's catalog; only its
			// external test package follows a Case into a recorded Run.
			allowed = test && !strings.Contains(part, "/internal/") && (helper == "temporal" || external && helper == "recordedrun")
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
	if strings.HasPrefix(part, "internal/engine/") && strings.HasPrefix(name, "tools/umpire/") && !strings.HasPrefix(name, "tools/umpire/internal/engine") {
		return "engine imports a higher layer"
	}
	if strings.HasPrefix(name, "tools/umpire/internal/engine") {
		return ""
	}
	if strings.HasPrefix(name, "tools/umpire/") {
		dependency := strings.TrimPrefix(name, "tools/umpire/")
		dependency = strings.Split(dependency, "/")[0]
		if readerPackages[dependency] {
			dependency = "model"
		}
		if dependency == owner || owner == "cmd" {
			return ""
		}
		// Lint reads lowering only through what its command hands it, so the reader is its one edge.
		allowed := map[string][]string{"lower": {"model"}, "conformance": {"model"}, "export": {"model"}, "explore": {"model", "lower"},
			"lint": {"model"}, "model": {}}
		if test {
			if owner == "lower" && external && !strings.Contains(part, "/internal/") {
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
	root, err := filepath.Abs(repoRoot)
	require.NoError(t, err)
	require.NoDirExists(t, filepath.Join(root, "model", "scalav2"))
	scanned := 0
	require.NoError(t, liveGoFiles(root, func(rel, pkg string, imports []string) {
		require.False(t, strings.HasPrefix(rel, "model/"), "Go source inside model: %s", rel)
		scanned++
		for _, imported := range imports {
			require.Empty(t, modelImportProblem(rel, strings.HasSuffix(pkg, "_test"), imported), "%s imports %s", rel, imported)
		}
	}))
	require.Greater(t, scanned, 100)
}

// liveGoFiles visits every Go file the main module builds, by its path from root, with the package
// it declares and what it imports. A directory with a go.mod of its own is another module, which
// leaves the build.
func liveGoFiles(root string, visit func(rel, pkg string, imports []string)) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// model/build is the Scala build's output; generated Go elsewhere is live code.
			if rel != "." && (strings.HasPrefix(entry.Name(), ".") || filepath.ToSlash(rel) == "model/build" || entry.Name() == "vendor") {
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
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		imports := make([]string, 0, len(parsed.Imports))
		for _, item := range parsed.Imports {
			imported, err := strconv.Unquote(item.Path.Value)
			if err != nil {
				return err
			}
			imports = append(imports, imported)
		}
		visit(filepath.ToSlash(rel), parsed.Name.Name, imports)
		return nil
	})
}

// toolingCallerProblem says why the package in directory has no caller, or "" when it has one: a
// live file outside the directory imports it, or the text of the Makefile and the CI workflow names
// the directory itself in a line that is no comment. A command is called only by being run.
func toolingCallerProblem(directory, name string, importers []string, commands string) string {
	commands = regexp.MustCompile(`(?m)^\s*#.*$`).ReplaceAllString(commands, "")
	run := regexp.MustCompile(`\./` + regexp.QuoteMeta(directory) + `/?(?:\s|$)`).MatchString(commands)
	switch {
	case run:
		return ""
	case name == "main":
		return "no Makefile target or CI step runs the command"
	case len(importers) == 0:
		return "no live file imports the package and no Makefile target or CI step runs it"
	default:
		return ""
	}
}

func TestEveryToolingPackageHasALiveCaller(t *testing.T) {
	root, err := filepath.Abs(repoRoot)
	require.NoError(t, err)
	var commands strings.Builder
	for _, file := range []string{"Makefile", ".github/workflows/umpire.yml"} {
		text, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		require.NoError(t, err)
		commands.Write(text)
	}
	const tooling, module = "tools/umpire/", "go.temporal.io/server/"
	packages := map[string]string{}
	importers := map[string][]string{}
	require.NoError(t, liveGoFiles(root, func(rel, pkg string, imports []string) {
		directory := filepath.ToSlash(filepath.Dir(rel))
		if strings.HasPrefix(rel, tooling) && !strings.HasSuffix(rel, "_test.go") {
			packages[directory] = pkg
		}
		for _, imported := range imports {
			if dependency := strings.TrimPrefix(imported, module); strings.HasPrefix(dependency, tooling) && dependency != directory {
				importers[dependency] = append(importers[dependency], rel)
			}
		}
	}))
	require.Greater(t, len(packages), 10)
	var problems []string
	for directory, name := range packages {
		if problem := toolingCallerProblem(directory, name, importers[directory], commands.String()); problem != "" {
			problems = append(problems, directory+": "+problem)
		}
	}
	slices.Sort(problems)
	require.Empty(t, problems)
}

func TestToolingCallerProblemRejectsAnUncalledPackage(t *testing.T) {
	commands := "\tgo build -o ./.build/umpire-run ./tools/umpire/cmd/umpire-run\n\tgo test ./tools/umpire/export -run X\n\tgo test ./tools/umpire/...\n" +
		"# formerly: go run ./tools/umpire/cmd/umpire-retired\n      # - run: go test ./tools/umpire/retired\n"
	for _, test := range []struct {
		directory, name string
		importers       []string
		called          bool
	}{
		{directory: "tools/umpire/lower", name: "lower", importers: []string{"tools/umpire/explore/explore.go"}, called: true},
		{directory: "tools/umpire/export", name: "export", called: true},
		{directory: "tools/umpire/cmd/umpire-run", name: "main", called: true},
		{directory: "tools/umpire/unused", name: "unused"},
		{directory: "tools/umpire/cmd/umpire-old", name: "main"},
		// A command is run, not imported, and a prefix of a name that is run is another command.
		{directory: "tools/umpire/cmd/umpire", name: "main", importers: []string{"tests/testpilot_test.go"}},
		{directory: "tools/umpire/cmd", name: "main"},
		// A comment of the Makefile or the workflow runs nothing.
		{directory: "tools/umpire/cmd/umpire-retired", name: "main"},
		{directory: "tools/umpire/retired", name: "retired"},
	} {
		t.Run(test.directory, func(t *testing.T) {
			problem := toolingCallerProblem(test.directory, test.name, test.importers, commands)
			require.Equal(t, test.called, problem == "", problem)
		})
	}
}

func TestModelDependencyGraphRejectsCrossedOwners(t *testing.T) {
	const module = "go.temporal.io/server/"
	for _, test := range []struct {
		file, dependency string
		external         bool
		allowed          bool
	}{
		{file: "tools/umpire/model/load.go", dependency: module + "api/umpire/v1", allowed: true},
		{file: "tools/umpire/model/load.go", dependency: module + "api/testpilot/v1"},
		{file: "tools/umpire/model/load_test.go", dependency: module + "common/testing/testpilot"},
		{file: "tools/umpire/model/load_test.go", dependency: module + "common/testing/testpilot", external: true},
		{file: "tools/umpire/model/load.go", dependency: module + "tools/umpire/lower"},
		{file: "tools/umpire/internal/engine/table.go", dependency: module + "tools/umpire/model"},
		{file: "tools/umpire/lower/internal/producer/producer.go", dependency: module + "tools/umpire/internal/engine"},
		{file: "tools/umpire/lower/internal/producer/producer.go", dependency: module + "tools/umpire/model", allowed: true},
		{file: "tools/umpire/lower/lower.go", dependency: module + "common/testing/testpilot", allowed: true},
		{file: "tools/umpire/lower/lower.go", dependency: module + "tools/umpire/explore"},
		{file: "tools/umpire/lower/lower.go", dependency: module + "tools/umpire/conformance"},
		{file: "tools/umpire/lower/lower.go", dependency: module + "common/testing/testpilot/recordedrun"},
		{file: "tools/umpire/lower/lower.go", dependency: module + "common/testing/testpilot/temporal"},
		{file: "tools/umpire/lower/lower.go", dependency: module + "common/testing/testpilot/campaign"},
		{file: "tools/umpire/lower/migration_test.go", dependency: module + "tools/umpire/explore", external: true, allowed: true},
		{file: "tools/umpire/lower/migration_test.go", dependency: module + "tools/umpire/conformance", external: true, allowed: true},
		{file: "tools/umpire/lower/migration_test.go", dependency: module + "common/testing/testpilot/recordedrun", external: true, allowed: true},
		{file: "tools/umpire/lower/lower_test.go", dependency: module + "tools/umpire/explore"},
		{file: "tools/umpire/lower/lower_test.go", dependency: module + "tools/umpire/conformance"},
		{file: "tools/umpire/lower/lower_test.go", dependency: module + "common/testing/testpilot/recordedrun"},
		{file: "tools/umpire/lower/lower_test.go", dependency: module + "common/testing/testpilot/temporal", allowed: true},
		{file: "tools/umpire/lower/migration_test.go", dependency: module + "common/testing/testpilot/campaign", external: true},
		{file: "tools/umpire/lower/internal/producer/producer_test.go", dependency: module + "common/testing/testpilot/recordedrun"},
		{file: "tools/umpire/lower/internal/producer/producer_test.go", dependency: module + "common/testing/testpilot/recordedrun", external: true},
		{file: "tools/umpire/lower/internal/producer/producer_test.go", dependency: module + "tools/umpire/explore", external: true},
		{file: "tools/umpire/conformance/conformance.go", dependency: module + "tools/umpire/lower"},
		{file: "tools/umpire/conformance/conformance_test.go", dependency: module + "tools/umpire/lower", allowed: true},
		{file: "tools/umpire/conformance/conformance.go", dependency: module + "tools/umpire/explore"},
		{file: "tools/umpire/lint/lint.go", dependency: module + "tools/umpire/model", allowed: true},
		{file: "tools/umpire/lint/lint.go", dependency: module + "tools/umpire/lower"},
		{file: "tools/umpire/lint/lint.go", dependency: module + "tools/umpire/explore"},
		{file: "tools/umpire/lint/lint_test.go", dependency: module + "tools/umpire/lower"},
		{file: "tools/umpire/cmd/umpire-lint/main.go", dependency: module + "tools/umpire/lint", allowed: true},
		{file: "tools/umpire/cmd/umpire-lint/main.go", dependency: module + "tools/umpire/lower", allowed: true},
		{file: "tools/umpire/cmd/umpire-run/run.go", dependency: module + "tools/umpire/conformance", allowed: true},
		{file: "tools/umpire/cmd/umpire-run/run.go", dependency: module + "tools/umpire/lower", allowed: true},
		{file: "tools/umpire/cmd/umpire-run/run.go", dependency: module + "tools/umpire/model", allowed: true},
		{file: "tools/umpire/cmd/umpire-assess/run.go", dependency: module + "tools/umpire/conformance", allowed: true},
		{file: "tools/umpire/cmd/umpire-assess/run.go", dependency: module + "tools/umpire/lower", allowed: true},
		{file: "tools/umpire/cmd/umpire-assess/run.go", dependency: module + "tools/umpire/model", allowed: true},
		{file: "tools/umpire/cmd/umpire-assess/run.go", dependency: module + "common/testing/testpilot/temporal", allowed: true},
		{file: "tools/umpire/export/slice.go", dependency: module + "tools/umpire/model", allowed: true},
		{file: "tools/umpire/export/slice.go", dependency: module + "api/umpire/v1", allowed: true},
		{file: "tools/umpire/export/slice.go", dependency: module + "common/testing/testpilot"},
		{file: "tools/umpire/export/slice.go", dependency: module + "api/testpilot/v1"},
		{file: "tools/umpire/export/slice.go", dependency: module + "tools/umpire/lower"},
		{file: "tools/umpire/export/verify_test.go", dependency: module + "tools/umpire/conformance"},
		{file: "tools/umpire/export/verify_test.go", dependency: module + "common/testing/testpilot/recordedrun"},
		{file: "tools/umpire/explore/explore.go", dependency: module + "common/testing/testpilot/temporal"},
		{file: "tools/umpire/explore/explore.go", dependency: module + "tools/umpire/lower", allowed: true},
		{file: "tools/umpire/explore/explore.go", dependency: module + "tools/umpire/model", allowed: true},
		{file: "tools/umpire/explore/explore.go", dependency: module + "common/testing/testpilot/campaign", allowed: true},
		{file: "tools/umpire/explore/explore.go", dependency: module + "common/testing/testpilot/replay", allowed: true},
		{file: "tools/umpire/explore/explore.go", dependency: module + "common/testing/testpilot/recordedrun", allowed: true},
		{file: "tools/umpire/explore/explore.go", dependency: module + "common/testing/testpilot/evaluation"},
		{file: "tools/umpire/explore/explore.go", dependency: module + "tools/umpire/conformance"},
		{file: "tools/umpire/explore/explore.go", dependency: module + "tools/umpire/export"},
		{file: "tools/umpire/internal/cli/cli.go", dependency: module + "common/testing/testpilot"},
		{file: "tools/umpire/cmd/x/main_test.go", dependency: module + "tests/testcore/testpilot"},
		{file: "service/history/handler.go", dependency: module + "model/go/umpire"},
	} {
		t.Run(test.file+"->"+test.dependency, func(t *testing.T) {
			problem := modelImportProblem(test.file, test.external, test.dependency)
			require.Equal(t, test.allowed, problem == "", problem)
		})
	}
}
