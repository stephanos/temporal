package regression

import (
	"bytes"
	"go/ast"
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
	packageLocalTestCommand  = "mise exec -- go test -count=1 -tags test_dep ./tools/umpire/temporal/local/... ./tools/umpire/runner/... ./tools/umpire/temporal/nexus/... ./tools/umpire/runevaluation/... ./tools/umpire/cmd/umpire-gen-tests-go/..."
	relocatedLiveTestCommand = "mise exec -- go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpireCallerClosurePortability$'"
	generatedGoTestPath      = "tests/umpire4_caller_closure_generated_test.go"
)

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
					{Name: "Run package-local Umpire tests", Run: packageLocalTestCommand},
					{Name: "Run the relocated Umpire test", Run: relocatedLiveTestCommand},
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
	require.Equal(t, 1, strings.Count(normalizedDryRun, relocatedLiveTestCommand))
	require.Contains(t, normalizedDryRun, "--output \"$temporary/tests\"")
	require.Contains(t, normalizedDryRun, "diff -u "+generatedGoTestPath)
}

func TestUmpireDocumentationStatesAttachedOwnershipAndBoundedClaim(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	for path, expected := range map[string][]string{
		"tools/umpire/runtime/README.md": {
			relocatedLiveTestCommand,
			"--output tests",
			"TestEnv owns the Temporal cluster and SDK client",
			"Umpire owns the per-run environment wrapper, SDK worker, Nexus endpoints, workflows, and run resources",
			"local.NewAttachedFactory",
		},
		"tools/umpire/runevaluation/README.md": {
			"TestUmpireCallerClosureRunEvaluation",
			"TestUmpireDuplicateDeliveryRunEvaluation",
			"testcore.NewEnv",
			"local.NewAttachedFactory",
		},
		"tools/umpire/portableevaluation/README.md": {
			"`testcore.NewEnv` retains ownership of the borrowed cluster and client",
			"Umpire owns only resources created for one run",
		},
		".plans/UMPIRE4_COMPONENTS.md": {
			relocatedLiveTestCommand,
			generatedGoTestPath,
			"TestEnv owns the Temporal cluster and SDK client",
			"local.NewAttachedFactory",
		},
		".plans/UMPIRE4_ORDER.md": {
			"local.NewAttachedFactory",
			"`TestEnv` owns cluster/client lifecycle",
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

	for _, path := range []string{
		"tools/umpire/runtime/README.md",
		".plans/UMPIRE4_COMPONENTS.md",
	} {
		documentation, err := os.ReadFile(filepath.Join(repositoryRoot, path))
		require.NoError(t, err)
		normalizedText := strings.Join(strings.Fields(string(documentation)), " ")

		require.Contains(t, normalizedText, "make umpire-check-regression")
		require.Contains(t, normalizedText, "byte-identical canonical v2 `ExperimentSpec`")
		require.Contains(t, normalizedText, "stable typed semantic meaning")
		require.Contains(t, normalizedText, "runtime-scoped transport identities")
		require.Contains(t, normalizedText, "Evaluation Profiles, Evaluation Receipts, provenance schemas, new artifact-set versions, Claim Assessment, remote, canary, and release work are excluded")
	}
}

func TestUmpireSourcesCannotRegainLegacyTemporalAuthority(t *testing.T) {
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

			localAliases := make(map[string]struct{})
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
				case "go.temporal.io/server/tools/umpire/temporal/local":
					alias := "local"
					if imported.Name != nil {
						alias = imported.Name.Name
					}
					if alias == "." {
						violations = append(violations, relative+": dot-imports the local authority package")
					} else if alias != "_" {
						localAliases[alias] = struct{}{}
					}
				default:
				}
			}

			localPackage := filepath.Dir(relative) == "tools/umpire/temporal/local"
			ast.Inspect(parsed, func(node ast.Node) bool {
				switch value := node.(type) {
				case *ast.FuncDecl:
					if localPackage && value.Name.Name == "New"+"Factory" {
						violations = append(violations, relative+": declares deprecated local.NewFactory")
					}
				case *ast.SelectorExpr:
					identifier, ok := value.X.(*ast.Ident)
					if !ok || value.Sel.Name != "New"+"Factory" {
						return true
					}
					if _, ok := localAliases[identifier.Name]; ok {
						violations = append(violations, relative+": calls deprecated local.NewFactory")
					}
				default:
				}
				return true
			})
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

	require.FileExists(t, filepath.Join(repositoryRoot, filepath.FromSlash(generatedGoTestPath)))
	require.NoFileExists(t, filepath.Join(
		repositoryRoot,
		"tools", "umpire", "temporal", "nexus", filepath.Base(generatedGoTestPath),
	))
}
