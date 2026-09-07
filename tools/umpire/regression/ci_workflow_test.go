package regression

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
	"gopkg.in/yaml.v3"
)

const (
	packageLocalTestCommand  = "mise exec -- go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/..."
	liveTestCommand          = "mise exec -- go test -count=1 -tags 'test_dep integration' ./tests -run '^(TestUmpire|TestTestpilotAsyncNexusCase)'"
	liveTestTargetCommand    = "make umpire-check-live-tests"
	conformanceTargetCommand = "./tools/umpire/cmd/umpire-gen-case-runtime-conformance"
	retiredVocabularyTarget  = "umpire-check-retired-vocabulary"
	retiredGeneratedTestPath = "tests/umpire4_caller_closure_generated_test.go"
	testpilotLiveSuccess     = "TestTestpilotAsyncNexusCase"
)

var inheritedLiveFailures = []string{
	"TestUmpire2TestSuite",
	"TestUmpire2TestSuite/TestPlanAndDriveKitchenSinkNexusOperation",
	"TestUmpire2TestSuite/TestPlanAndDriveNexusOperationCHASM",
	"TestUmpire2TestSuite/TestProbeNexusDegraded",
	"TestUmpire2TestSuite/TestProbeNexusExploration",
	"TestUmpire2TestSuite/TestProbeNexusFlagged",
	"TestUmpire2TestSuite/TestProbeNexusRandomized",
	"TestUmpire2TestSuite/TestProbeNexusResilience",
	"TestUmpire3ParticipantProcessCrashAndRestartResumesRealSDKProgram",
}

type ciWorkflow struct {
	Name        string                       `yaml:"name"`
	On          map[string]ciWorkflowTrigger `yaml:"on"`
	Permissions map[string]string            `yaml:"permissions"`
	Concurrency ciWorkflowConcurrency        `yaml:"concurrency"`
	Jobs        map[string]ciWorkflowJob     `yaml:"jobs"`
}

type ciWorkflowTrigger struct {
	Branches []string `yaml:"branches"`
}

type ciWorkflowConcurrency struct {
	Group            string `yaml:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress"`
}

type ciWorkflowJob struct {
	RunsOn         string           `yaml:"runs-on"`
	TimeoutMinutes int              `yaml:"timeout-minutes"`
	Steps          []ciWorkflowStep `yaml:"steps"`
}

type ciWorkflowStep struct {
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses"`
	With map[string]any `yaml:"with"`
	Run  string         `yaml:"run"`
}

func TestUmpireCIWorkflowRunsSeparatedUnitAndLiveProofs(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	workflowBytes, err := os.ReadFile(filepath.Join(repositoryRoot, ".github", "workflows", "umpire.yml"))
	require.NoError(t, err)
	var workflow ciWorkflow
	decoder := yaml.NewDecoder(bytes.NewReader(workflowBytes))
	decoder.KnownFields(true)
	require.NoError(t, decoder.Decode(&workflow))

	require.Equal(t, ciWorkflow{
		Name: "Umpire",
		On: map[string]ciWorkflowTrigger{
			"pull_request": {},
			"push": {
				Branches: []string{"main", "stephanos/umpire"},
			},
		},
		Permissions: map[string]string{"contents": "read"},
		Concurrency: ciWorkflowConcurrency{
			Group:            "umpire-${{ github.head_ref || github.run_id }}",
			CancelInProgress: true,
		},
		Jobs: map[string]ciWorkflowJob{
			"portability": {
				RunsOn:         "ubuntu-24.04",
				TimeoutMinutes: 15,
				Steps: []ciWorkflowStep{
					{Uses: "actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10"},
					{
						Uses: "actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c",
						With: map[string]any{
							"go-version-file": "go.mod",
							"check-latest":    false,
							"cache":           false,
						},
					},
					{
						Uses: "jdx/mise-action@dba19683ed58901619b14f395a24841710cb4925",
						With: map[string]any{
							"version":           "2026.8.16",
							"sha256":            "cff4832ded79af2951e800bddcb5a22acac58630d765a2d062c1180680a0bb35",
							"working_directory": "model",
							"cache":             false,
						},
					},
					{Name: "Run package-local Testpilot and Umpire Producer tests", Run: packageLocalTestCommand},
					{Name: "Run the live Umpire tests", Run: liveTestTargetCommand},
				},
			},
		},
	}, workflow)

	command := exec.Command("make", "--no-print-directory", "-n", "umpire-check-regression")
	command.Dir = repositoryRoot
	dryRun, err := command.Output()
	require.NoError(t, err)
	normalizedDryRun := strings.Join(strings.Fields(strings.ReplaceAll(string(dryRun), "\\\n", " ")), " ")
	require.Equal(t, 1, strings.Count(normalizedDryRun, packageLocalTestCommand))
	require.Equal(t, 1, strings.Count(normalizedDryRun, liveTestCommand))
	require.Contains(t, normalizedDryRun, conformanceTargetCommand)
	require.Contains(t, normalizedDryRun, retiredVocabularyTarget)
	for _, identity := range inheritedLiveFailures {
		require.Contains(t, normalizedDryRun, identity)
	}
	require.Contains(t, normalizedDryRun, testpilotLiveSuccess)
	require.Contains(t, normalizedDryRun, "Live Umpire failure identities differ from the inherited exact set.")
	require.Contains(t, normalizedDryRun, "Live Umpire failure identities match the inherited exact set.")
	require.NotContains(t, normalizedDryRun, retiredGeneratedTestPath)
}

func TestUmpireDocumentationStatesAttachedOwnershipAndBoundedClaim(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	for path, expected := range map[string][]string{
		"tools/umpire/CONTEXT.md": {
			"A coherent pairing of one Program and one Contract",
			"Immutable, single-assignment typed operational data",
			"private Testpilot component that applies a Contract to a Program and its Run",
		},
		"tests/testcore/testpilot/README.md": {
			"owns the retained generated functional fixtures",
			"fixture admission and prepared-Case reuse tests",
			"Cluster provisioning",
		},
		"common/testing/testpilot/temporal/README.md": {
			"The composite adds no Case or scenario interpretation",
			"`Open` creates the server Session first",
			"NewWorkflowServiceCatalog",
		},
		"common/testing/testpilot/internal/execution/README.md": {
			"Raw payloads and Slots are not evidence",
			"`Run` owns actual Driver/bridge closure",
			"private `scheduler`",
		},
		"common/testing/testpilot/internal/verification/README.md": {
			"`Observe` processes an appended event synchronously",
			"a PreparedContract supports concurrent independent Runs",
		},
		".plans/UMPIRE_CASE_RUNTIME_DESIGN.md": {
			"Current ownership is `common/testing/testpilot`",
			"Temporal Driver under `common/testing/testpilot/temporal`",
		},
		"model/README.md": {
			"testpilot.Prepare(case, Profile)",
			"Temporal authority remains split",
			"complete twelve-file conformance tree",
		},
		"model/ARCHITECTURE.md": {
			"The Testpilot `.proto` files own the Case protocol",
			"`common/testing/testpilot` owns the Profile/Driver contract",
			"`common/testing/testpilot/temporal`",
			"checks horizon expiry before every transition",
		},
		"model/Umpire/ARCHITECTURE.md": {
			"Case, Program, Contract, and Run vocabularies are finite, versioned, and bounded",
			"Promotion remains generic and review-only",
		},
	} {
		documentation, err := os.ReadFile(filepath.Join(repositoryRoot, path))
		require.NoError(t, err)
		text := string(documentation)
		normalizedText := strings.Join(strings.Fields(text), " ")

		for _, fragment := range expected {
			require.Contains(t, normalizedText, fragment, path)
		}
	}

}

func TestUmpireSourcesCannotRegainRetiredTemporalAuthority(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	var violations []string
	for _, sourceRoot := range []string{
		filepath.Join(repositoryRoot, "tools", "umpire"),
		filepath.Join(repositoryRoot, "tests"),
	} {
		err := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if sourceRoot == filepath.Join(repositoryRoot, "tests") && path != sourceRoot {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}

			encoded, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), path, encoded, 0)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(repositoryRoot, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)

			for _, imported := range parsed.Imports {
				importPath, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}
				switch importPath {
				case "go.temporal.io/server/" + "temporaltest":
					violations = append(violations, relative+": imports deprecated temporaltest authority")
				case "go.temporal.io/server/tests/" + "testcore":
					if strings.HasPrefix(relative, "tools/umpire/") && !strings.HasSuffix(relative, "_test.go") {
						violations = append(violations, relative+": production Umpire imports tests/testcore")
					}
				default:
				}
			}
			return nil
		})
		require.NoError(t, err)
	}

	slices.Sort(violations)
	require.Empty(t, violations)
}

func TestGeneratedUmpireTestHasOnlyTheRelocatedDestination(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	require.NoFileExists(t, filepath.Join(repositoryRoot, filepath.FromSlash(retiredGeneratedTestPath)))
	require.NoFileExists(t, filepath.Join(
		repositoryRoot,
		"tools", "umpire", "temporal", "nexus", filepath.Base(retiredGeneratedTestPath),
	))
}

func TestTestpilotOwnsCaseProtocolAndRuntime(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	for _, former := range []string{
		"api/umpire",
		"api/umpire/v1",
		"proto/internal/temporal/server/api/umpire",
		"proto/internal/temporal/server/api/umpire/v1",
		"tools/umpire/internal/ir",
		"tools/umpire/internal/execution",
		"tools/umpire/verification",
		"tools/umpire/caseartifact",
		"tools/umpire/temporal",
		"tools/umpire/testdata",
		"tools/umpire/internal/legacy" + "vocabulary",
		"tools/umpire/cmd/umpire-check-legacy" + "-vocabulary",
	} {
		require.NoDirExists(t, filepath.Join(repositoryRoot, filepath.FromSlash(former)), former)
	}
	for _, former := range []string{
		"tools/umpire/prepare.go",
		"tools/umpire/prepared_case.go",
		"tools/umpire/profile.go",
		"tools/umpire/host.go",
		"tools/umpire/carrier_test.go",
		"common/testing/testpilot/protocol_compatibility_test.go",
		"tools/umpire/vocabulary/legacy" + "_gate_test.go",
	} {
		require.NoFileExists(t, filepath.Join(repositoryRoot, filepath.FromSlash(former)), former)
	}
	require.FileExists(t, filepath.Join(repositoryRoot, "common", "testing", "testpilot", "internal", "verification", "evaluator.go"))
	generatedTypes, err := os.ReadFile(filepath.Join(repositoryRoot, "model", "Temporal", "API", "Types.lean"))
	require.NoError(t, err)
	require.NotContains(t, string(generatedTypes), "Temporal.Server.Api."+"Umpire.V1")

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
	command := exec.CommandContext(ctx, "go", "list", "-tags", "test_dep", "-deps", "-test", "-json", "./common/testing/testpilot/temporal/...")
	command.Dir = repositoryRoot
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list shared Driver production/test dependency closure: %w: %s", err, strings.TrimSpace(string(output)))
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
				if relativeRoot == genericRoot && hasImportPrefix(importPath, moduleRoot+driverRoot) {
					violations = append(violations, relative+": imports Temporal Driver "+importPath)
				}
				if relativeRoot == genericRoot && forbiddenGenericTestpilotDependency(importPath) {
					violations = append(violations, relative+": imports forbidden dependency "+importPath)
				}
				if relativeRoot == driverRoot {
					if forbiddenTemporalDriverDependency(importPath) {
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
		if forbiddenTemporalDriverDependency(importPath) {
			violations = append(violations, "shared Driver dependency closure reaches "+importPath)
		}
	}
	slices.Sort(violations)
	return violations, nil
}

func forbiddenTemporalDriverDependency(importPath string) bool {
	return hasImportPrefix(importPath, "go.temporal.io/server/tests") ||
		strings.HasPrefix(importPath, "go.temporal.io/server/tools/umpire/cmd/umpire-gen-") ||
		hasImportPrefix(importPath, "go.temporal.io/server/tools/canary")
}

func forbiddenGenericTestpilotDependency(importPath string) bool {
	return hasImportPrefix(importPath, "go.temporal.io/server/tests/testcore") ||
		hasImportPrefix(importPath, "go.temporal.io/server/tools/umpire") ||
		hasImportPrefix(importPath, "go.temporal.io/server/tools/canary")
}

func hasImportPrefix(importPath string, prefix string) bool {
	return importPath == prefix || strings.HasPrefix(importPath, prefix+"/")
}

func TestMigrationLedgerAndGenericPromotionRemainClosed(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	ledger, err := os.ReadFile(filepath.Join(repositoryRoot, ".flow", "artifacts", "fn-64-umpire-case-runtime", "task8-migration-ledger.md"))
	require.NoError(t, err)
	ledgerText := string(ledger)
	for _, fragment := range []string{
		"Manifest count: `179`.",
		"All 307 deleted top-level Go Test/Fuzz entry points have a status",
		"All 10 inherited failure identities are preserved and named exactly.",
		"Official read-only ledger review verdict is `SHIP`.",
		"| 3 | `codex:gpt-5.6-sol:high` | SHIP |",
	} {
		require.Contains(t, ledgerText, fragment)
	}

	promotion, err := os.ReadFile(filepath.Join(repositoryRoot, "model", "Umpire", "Promotion.lean"))
	require.NoError(t, err)
	promotionTests, err := os.ReadFile(filepath.Join(repositoryRoot, "model", "Umpire", "PromotionTests.lean"))
	require.NoError(t, err)
	combined := string(promotion) + string(promotionTests)
	require.NotContains(t, combined, "Caller"+"Closure")
	require.NotContains(t, combined, "Temporal"+".System")
	require.Contains(t, string(promotion), "import Umpire.Planning.Engine")
}
