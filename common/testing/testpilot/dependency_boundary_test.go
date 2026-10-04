package testpilot_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTestpilotDependencyBoundary(t *testing.T) {
	repositoryRoot, err := filepath.Abs("../../..")
	require.NoError(t, err)
	dependencies, err := listTestpilotDependencies(t.Context(), repositoryRoot)
	require.NoError(t, err)
	violations, err := checkTestpilotDependencyBoundary(repositoryRoot, dependencies)
	require.NoError(t, err)
	require.Empty(t, violations)
}

func TestTestpilotDependencyBoundaryRejectsForbiddenEdges(t *testing.T) {
	tests := []struct {
		name       string
		relative   string
		source     string
		packages   []testpilotDependency
		unreadable bool
		want       string
		wantError  string
	}{
		{
			name:     "shared Driver imports repository tests",
			relative: "common/testing/testpilot/temporal/driver.go",
			source:   "package temporal\nimport _ \"go.temporal.io/server/tests/testcore\"\n",
			want:     "common/testing/testpilot/temporal/driver.go: imports forbidden dependency go.temporal.io/server/tests/testcore",
		},
		{
			name:     "shared Driver imports Umpire generator",
			relative: "common/testing/testpilot/temporal/driver.go",
			source:   "package temporal\nimport _ \"go.temporal.io/server/tools/umpire/cmd/umpire-gen-regression-views\"\n",
			want:     "common/testing/testpilot/temporal/driver.go: imports forbidden dependency go.temporal.io/server/tools/umpire/cmd/umpire-gen-regression-views",
		},
		{
			name:     "shared Driver imports canary orchestration",
			relative: "common/testing/testpilot/temporal/driver.go",
			source:   "package temporal\nimport _ \"go.temporal.io/server/tools/canary\"\n",
			want:     "common/testing/testpilot/temporal/driver.go: imports forbidden dependency go.temporal.io/server/tools/canary",
		},
		{
			name:     "shared Driver imports private IR",
			relative: "common/testing/testpilot/temporal/driver.go",
			source:   "package temporal\nimport _ \"go.temporal.io/server/common/testing/testpilot/internal/ir\"\n",
			want:     "common/testing/testpilot/temporal/driver.go: imports generic Testpilot private package go.temporal.io/server/common/testing/testpilot/internal/ir",
		},
		{
			name:     "shared Driver imports private execution",
			relative: "common/testing/testpilot/temporal/driver.go",
			source:   "package temporal\nimport _ \"go.temporal.io/server/common/testing/testpilot/internal/execution\"\n",
			want:     "common/testing/testpilot/temporal/driver.go: imports generic Testpilot private package go.temporal.io/server/common/testing/testpilot/internal/execution",
		},
		{
			name:     "shared Driver imports private verification",
			relative: "common/testing/testpilot/temporal/driver.go",
			source:   "package temporal\nimport _ \"go.temporal.io/server/common/testing/testpilot/internal/verification\"\n",
			want:     "common/testing/testpilot/temporal/driver.go: imports generic Testpilot private package go.temporal.io/server/common/testing/testpilot/internal/verification",
		},
		{
			name:     "generic Testpilot imports Temporal Driver",
			relative: "common/testing/testpilot/driver.go",
			source:   "package testpilot\nimport _ \"go.temporal.io/server/common/testing/testpilot/temporal\"\n",
			want:     "common/testing/testpilot/driver.go: imports Temporal Driver go.temporal.io/server/common/testing/testpilot/temporal",
		},
		{
			name:     "generic Testpilot imports repository tests",
			relative: "common/testing/testpilot/driver.go",
			source:   "package testpilot\nimport _ \"go.temporal.io/server/tests/testcore\"\n",
			want:     "common/testing/testpilot/driver.go: imports forbidden dependency go.temporal.io/server/tests/testcore",
		},
		{
			name:     "generic Testpilot imports Umpire tooling",
			relative: "common/testing/testpilot/driver.go",
			source:   "package testpilot\nimport _ \"go.temporal.io/server/tools/umpire/regression\"\n",
			want:     "common/testing/testpilot/driver.go: imports forbidden dependency go.temporal.io/server/tools/umpire/regression",
		},
		{
			name:     "generic Testpilot imports canary orchestration",
			relative: "common/testing/testpilot/driver.go",
			source:   "package testpilot\nimport _ \"go.temporal.io/server/tools/canary\"\n",
			want:     "common/testing/testpilot/driver.go: imports forbidden dependency go.temporal.io/server/tools/canary",
		},
		{
			name:     "server imports SDK worker",
			relative: "common/testing/testpilot/temporal/server/driver.go",
			source:   "package server\nimport _ \"go.temporal.io/server/common/testing/testpilot/temporal/worker\"\n",
			want:     "common/testing/testpilot/temporal/server/driver.go: crosses adapter authority through go.temporal.io/server/common/testing/testpilot/temporal/worker",
		},
		{
			name:     "SDK worker imports server",
			relative: "common/testing/testpilot/temporal/worker/driver.go",
			source:   "package worker\nimport _ \"go.temporal.io/server/common/testing/testpilot/temporal/server\"\n",
			want:     "common/testing/testpilot/temporal/worker/driver.go: crosses adapter authority through go.temporal.io/server/common/testing/testpilot/temporal/server",
		},
		{
			name:     "delivery imports server",
			relative: "common/testing/testpilot/temporal/internal/delivery/ledger.go",
			source:   "package delivery\nimport _ \"go.temporal.io/server/common/testing/testpilot/temporal/server\"\n",
			want:     "common/testing/testpilot/temporal/internal/delivery/ledger.go: crosses adapter authority through go.temporal.io/server/common/testing/testpilot/temporal/server",
		},
		{
			name:     "delivery imports SDK worker",
			relative: "common/testing/testpilot/temporal/internal/delivery/ledger.go",
			source:   "package delivery\nimport _ \"go.temporal.io/server/common/testing/testpilot/temporal/worker\"\n",
			want:     "common/testing/testpilot/temporal/internal/delivery/ledger.go: crosses adapter authority through go.temporal.io/server/common/testing/testpilot/temporal/worker",
		},
		{
			name:     "closure reaches repository tests",
			packages: []testpilotDependency{{ImportPath: "go.temporal.io/server/tests/testcore"}},
			want:     "shared Driver dependency closure reaches go.temporal.io/server/tests/testcore",
		},
		{
			name:     "closure reaches Umpire generator",
			packages: []testpilotDependency{{ImportPath: "go.temporal.io/server/tools/umpire/cmd/umpire-gen-case-runtime-conformance"}},
			want:     "shared Driver dependency closure reaches go.temporal.io/server/tools/umpire/cmd/umpire-gen-case-runtime-conformance",
		},
		{
			name:     "closure reaches canary orchestration",
			packages: []testpilotDependency{{ImportPath: "go.temporal.io/server/tools/canary [go.temporal.io/server/tools/canary.test]"}},
			want:     "shared Driver dependency closure reaches go.temporal.io/server/tools/canary",
		},
		{
			name:      "malformed source",
			relative:  "common/testing/testpilot/temporal/driver.go",
			source:    "package temporal\nimport (\n",
			wantError: "parse common/testing/testpilot/temporal/driver.go",
		},
		{
			name:       "unreadable source",
			relative:   "common/testing/testpilot/temporal/driver.go",
			unreadable: true,
			wantError:  "inspect common/testing/testpilot/temporal/driver.go",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repositoryRoot := t.TempDir()
			for _, relative := range []string{
				"common/testing/testpilot",
				"common/testing/testpilot/temporal/server",
				"common/testing/testpilot/temporal/worker",
				"common/testing/testpilot/temporal/internal/delivery",
			} {
				require.NoError(t, os.MkdirAll(filepath.Join(repositoryRoot, filepath.FromSlash(relative)), 0o755))
			}
			if test.relative != "" {
				path := filepath.Join(repositoryRoot, filepath.FromSlash(test.relative))
				if test.unreadable {
					require.NoError(t, os.Symlink(filepath.Join(repositoryRoot, "missing.go"), path))
				} else {
					require.NoError(t, os.WriteFile(path, []byte(test.source), 0o600))
				}
			}

			violations, err := checkTestpilotDependencyBoundary(repositoryRoot, test.packages)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			require.Contains(t, violations, test.want)
		})
	}
}

type testpilotDependency struct {
	ImportPath string
}

func listTestpilotDependencies(ctx context.Context, repositoryRoot string) ([]testpilotDependency, error) {
	command := exec.CommandContext(ctx, "go", "list", "-tags", "test_dep", "-deps", "-test", "-json", "./common/testing/testpilot/...")
	command.Dir = repositoryRoot
	// Only stdout carries the JSON stream: a cold module cache reports its downloads on stderr, which
	// mixed into the same reader would make the decode fail on progress text.
	var progress bytes.Buffer
	command.Stderr = &progress
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list shared Driver production/test dependency closure: %w: %s", err, strings.TrimSpace(progress.String()))
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var dependencies []testpilotDependency
	for decoder.More() {
		var dependency testpilotDependency
		if err := decoder.Decode(&dependency); err != nil {
			return nil, fmt.Errorf("decode shared Driver dependency closure: %w", err)
		}
		dependencies = append(dependencies, dependency)
	}
	return dependencies, nil
}

func checkTestpilotDependencyBoundary(repositoryRoot string, dependencies []testpilotDependency) ([]string, error) {
	const (
		genericRoot = "common/testing/testpilot"
		driverRoot  = "common/testing/testpilot/temporal"
		moduleRoot  = "go.temporal.io/server/"
	)
	var violations []string
	for _, relativeRoot := range []string{genericRoot, driverRoot} {
		sourceRoot := filepath.Join(repositoryRoot, filepath.FromSlash(relativeRoot))
		err := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if relativeRoot == genericRoot && path == filepath.Join(repositoryRoot, filepath.FromSlash(driverRoot)) {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			relative, err := filepath.Rel(repositoryRoot, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			encoded, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("inspect %s: %w", relative, err)
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), path, encoded, parser.ImportsOnly)
			if err != nil {
				return fmt.Errorf("parse %s: %w", relative, err)
			}
			for _, imported := range parsed.Imports {
				importPath, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return fmt.Errorf("inspect import in %s: %w", relative, err)
				}
				if relativeRoot == genericRoot && hasImportPrefix(importPath, moduleRoot+driverRoot) && !permittedHelperDriverImport(relative, importPath) {
					violations = append(violations, relative+": imports Temporal Driver "+importPath)
				}
				if relativeRoot == genericRoot && forbiddenTestpilotDependency(importPath) {
					violations = append(violations, relative+": imports forbidden dependency "+importPath)
				}
				if relativeRoot == driverRoot {
					if forbiddenTestpilotDependency(importPath) {
						violations = append(violations, relative+": imports forbidden dependency "+importPath)
					}
					if hasImportPrefix(importPath, moduleRoot+genericRoot+"/internal/ir") ||
						hasImportPrefix(importPath, moduleRoot+genericRoot+"/internal/execution") ||
						hasImportPrefix(importPath, moduleRoot+genericRoot+"/internal/verification") {
						violations = append(violations, relative+": imports generic Testpilot private package "+importPath)
					}
					server := moduleRoot + driverRoot + "/server"
					worker := moduleRoot + driverRoot + "/worker"
					if strings.HasPrefix(relative, driverRoot+"/server/") && hasImportPrefix(importPath, worker) ||
						strings.HasPrefix(relative, driverRoot+"/worker/") && hasImportPrefix(importPath, server) ||
						strings.HasPrefix(relative, driverRoot+"/internal/delivery/") && (hasImportPrefix(importPath, server) || hasImportPrefix(importPath, worker)) {
						violations = append(violations, relative+": crosses adapter authority through "+importPath)
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	for _, dependency := range dependencies {
		importPath := strings.SplitN(dependency.ImportPath, " [", 2)[0]
		if forbiddenTestpilotDependency(importPath) {
			violations = append(violations, "shared Driver dependency closure reaches "+importPath)
		}
	}
	slices.Sort(violations)
	return violations, nil
}

func permittedHelperDriverImport(relative, importPath string) bool {
	const root = "common/testing/testpilot/"
	const driver = "go.temporal.io/server/common/testing/testpilot/temporal"
	if importPath == driver+"/binding" && (strings.HasPrefix(relative, root+"campaign/") || strings.HasPrefix(relative, root+"replay/")) {
		return true
	}
	if importPath == driver {
		switch relative {
		case root + "evaluation/admission_test.go", root + "replay/driver_test.go", root + "replay/key_test.go":
			return true
		}
	}
	return false
}

func forbiddenTestpilotDependency(importPath string) bool {
	for _, prefix := range []string{
		"go.temporal.io/server/tests",
		"go.temporal.io/server/tools/umpire",
		"go.temporal.io/server/tools/canary",
		"go.temporal.io/server/model",
		"go.temporal.io/server/api/umpire",
	} {
		if hasImportPrefix(importPath, prefix) {
			return true
		}
	}
	return false
}

func hasImportPrefix(importPath string, prefix string) bool {
	return importPath == prefix || strings.HasPrefix(importPath, prefix+"/")
}
func TestTestpilotHelperDriverBoundary(t *testing.T) {
	const root = "common/testing/testpilot/"
	const driver = "go.temporal.io/server/common/testing/testpilot/temporal"
	for _, test := range []struct {
		file       string
		dependency string
		permitted  bool
	}{
		{"campaign/run.go", driver + "/binding", true},
		{"campaign/run_test.go", driver + "/binding", true},
		{"replay/report.go", driver + "/binding", true},
		{"evaluation/admission_test.go", driver, true},
		{"replay/driver_test.go", driver, true},
		{"replay/key_test.go", driver, true},
		{"evaluation/admission.go", driver, false},
		{"evaluation/other_test.go", driver, false},
		{"campaign/run.go", driver, false},
		{"replay/report.go", driver + "/server", false},
		{"recordedrun/recordedrun.go", driver + "/binding", false},
		{"prepare.go", driver + "/binding", false},
		{"campaign/run.go", "go.temporal.io/server/tools/umpire/model", false},
		{"replay/report.go", "go.temporal.io/server/tools/umpire/replay", false},
		{"temporal/binding/binding.go", "go.temporal.io/server/model/go/umpire", false},
		{"replay/driver_test.go", "go.temporal.io/server/api/umpire/v1", false},
	} {
		t.Run(test.file+":"+test.dependency, func(t *testing.T) {
			repositoryRoot := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(repositoryRoot, root, "temporal"), 0o755))
			path := filepath.Join(repositoryRoot, root, test.file)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte("package fixture\nimport _ "+strconv.Quote(test.dependency)+"\n"), 0o600))
			violations, err := checkTestpilotDependencyBoundary(repositoryRoot, nil)
			require.NoError(t, err)
			if test.permitted {
				require.Empty(t, violations)
			} else {
				require.NotEmpty(t, violations)
				require.Contains(t, strings.Join(violations, "\n"), root+test.file)
				require.Contains(t, strings.Join(violations, "\n"), test.dependency)
			}
		})
	}
}
