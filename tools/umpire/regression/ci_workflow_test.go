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
	packageLocalTestCommand  = "mise exec -- go test -count=1 -tags test_dep ./tools/umpire/..."
	liveTestCommand          = "mise exec -- go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpire'"
	liveTestTargetCommand    = "make umpire-check-live-tests"
	conformanceTargetCommand = "./tools/umpire/cmd/umpire-gen-case-runtime-conformance"
	retiredGeneratedTestPath = "tests/umpire4_caller_closure_generated_test.go"
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
					{Name: "Run package-local Umpire tests", Run: packageLocalTestCommand},
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
	for _, identity := range inheritedLiveFailures {
		require.Contains(t, normalizedDryRun, identity)
	}
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
			"applies a Contract to a Program and its Run",
		},
		"tools/umpire/temporal/README.md": {
			"The composite adds no Case or scenario interpretation",
			"`Open` creates the server Session first",
			"NewWorkflowServiceCatalog",
		},
		"tools/umpire/internal/execution/README.md": {
			"Raw payloads and Slots are not evidence",
			"`Run` owns actual Host/bridge closure",
			"private `scheduler`",
		},
		"tools/umpire/verification/README.md": {
			"`Observe` processes an appended event synchronously",
			"a PreparedContract supports concurrent independent Runs",
		},
		".plans/UMPIRE_CASE_RUNTIME_DESIGN.md": {
			"No Nexus lifecycle checker or callback-closure adapter is implemented in Go",
			"The root facade owns the public Profile",
		},
		"model/README.md": {
			"PrepareCase(case, Profile)",
			"Temporal authority remains split",
			"complete twelve-file tree under one physical temporary root",
		},
		"model/ARCHITECTURE.md": {
			"Scheduling, recording, effect ownership, private Slot storage, and Monitor factories are internal",
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

func TestCaseRuntimeFacadeAndExecutionImportBoundary(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	root := filepath.Join(repositoryRoot, "tools", "umpire")
	forbiddenDeclarations := map[string]bool{
		"Scheduler": true, "Recorder": true, "Slot": true, "MonitorFactory": true,
		"NewScheduler": true, "NewRecorder": true, "NewSlot": true, "NewMonitorFactory": true,
	}
	facadeCalls := map[string]bool{"PrepareCase": false, "PreparedCase.Run": false}
	var preparedCaseExports []string
	var runtimeBoundaryCalls []string
	var violations []string
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		require.NoError(t, err)
		for _, declaration := range parsed.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					if named, ok := spec.(*ast.TypeSpec); ok && named.Name.IsExported() {
						if forbiddenDeclarations[named.Name.Name] {
							violations = append(violations, entry.Name()+": public "+named.Name.Name)
						}
						if exportedTypeExposesInternalExecution(named) {
							violations = append(violations, entry.Name()+": "+named.Name.Name+" exposes internal execution construction")
						}
					}
				}
			case *ast.FuncDecl:
				if declaration.Name.IsExported() && forbiddenDeclarations[declaration.Name.Name] {
					violations = append(violations, entry.Name()+": public "+declaration.Name.Name)
				}
				if declaration.Recv == nil && declaration.Name.Name == "PrepareCase" {
					facadeCalls["PrepareCase"] = true
				}
				if receiverTypeName(declaration.Recv) == "PreparedCase" && declaration.Name.Name == "Run" {
					facadeCalls["PreparedCase.Run"] = true
				}
				if receiverTypeName(declaration.Recv) == "PreparedCase" && declaration.Name.IsExported() {
					preparedCaseExports = append(preparedCaseExports, "PreparedCase."+declaration.Name.Name)
				}
				if declaration.Name.IsExported() && signatureExposesInternalExecution(declaration.Type) {
					violations = append(violations, entry.Name()+": "+declaration.Name.Name+" exposes internal execution construction")
				}
				if declaration.Name.IsExported() && signatureMentionsRuntimeBoundary(declaration) {
					name := declaration.Name.Name
					if receiver := receiverTypeName(declaration.Recv); receiver != "" {
						name = receiver + "." + name
					}
					runtimeBoundaryCalls = append(runtimeBoundaryCalls, name)
				}
			default:
				continue
			}
		}
	}
	require.Empty(t, violations)
	require.Equal(t, map[string]bool{"PrepareCase": true, "PreparedCase.Run": true}, facadeCalls)
	slices.Sort(preparedCaseExports)
	require.Equal(t, []string{"PreparedCase.Identity", "PreparedCase.Run", "PreparedCase.Snapshot"}, preparedCaseExports)
	slices.Sort(runtimeBoundaryCalls)
	require.Equal(t, []string{"PrepareCase", "PreparedCase.Run"}, runtimeBoundaryCalls)

	conformanceSource, err := os.ReadFile(filepath.Join(root, "conformance_test.go"))
	require.NoError(t, err)
	require.NotContains(t, string(conformanceSource), "os/"+"exec")
	require.NotContains(t, string(conformanceSource), "-"+"rewrite")
	require.NotContains(t, string(conformanceSource), "lake "+"build")

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !strings.Contains(relative, "/") || strings.HasPrefix(relative, "internal/execution/") || strings.HasPrefix(relative, "verification/") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range parsed.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if importPath == "go.temporal.io/server/tools/umpire/internal/execution" {
				violations = append(violations, relative+": imports internal execution")
			}
		}
		return nil
	})
	require.NoError(t, err)
	slices.Sort(violations)
	require.Empty(t, violations)
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

func receiverTypeName(receiver *ast.FieldList) string {
	if receiver == nil || len(receiver.List) != 1 {
		return ""
	}
	typeExpression := receiver.List[0].Type
	if pointer, ok := typeExpression.(*ast.StarExpr); ok {
		typeExpression = pointer.X
	}
	if named, ok := typeExpression.(*ast.Ident); ok {
		return named.Name
	}
	return ""
}

func signatureMentionsRuntimeBoundary(declaration *ast.FuncDecl) bool {
	mentionsBoundary := false
	isTopLevel := declaration.Recv == nil
	for _, fields := range []*ast.FieldList{declaration.Type.Params, declaration.Type.Results} {
		if fields == nil {
			continue
		}
		ast.Inspect(fields, func(node ast.Node) bool {
			switch expression := node.(type) {
			case *ast.Ident:
				mentionsBoundary = mentionsBoundary || expression.Name == "Profile" ||
					expression.Name == "PreparedCase" || expression.Name == "Host"
			case *ast.SelectorExpr:
				mentionsBoundary = mentionsBoundary || expression.Sel.Name == "Run" ||
					expression.Sel.Name == "Verdict" || isTopLevel && expression.Sel.Name == "Case"
			default:
				return !mentionsBoundary
			}
			return !mentionsBoundary
		})
	}
	return mentionsBoundary
}

func exportedTypeExposesInternalExecution(named *ast.TypeSpec) bool {
	if structure, ok := named.Type.(*ast.StructType); ok && !named.Assign.IsValid() {
		for _, field := range structure.Fields.List {
			if len(field.Names) != 0 && field.Names[0].IsExported() && expressionExposesInternalExecution(field.Type) {
				return true
			}
		}
		return false
	}
	return expressionExposesInternalExecution(named.Type)
}

func signatureExposesInternalExecution(function *ast.FuncType) bool {
	return expressionExposesInternalExecution(function.Params) || expressionExposesInternalExecution(function.Results)
}

func expressionExposesInternalExecution(node ast.Node) bool {
	exposed := false
	ast.Inspect(node, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		packageName, ok := selector.X.(*ast.Ident)
		exposed = exposed || ok && packageName.Name == "execution" && map[string]bool{
			"Scheduler": true, "Recorder": true, "Slot": true, "MonitorFactory": true,
		}[selector.Sel.Name]
		return !exposed
	})
	return exposed
}
