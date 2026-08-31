package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderGeneratedRunnerTestMatchesTheCheckedInOrdinaryGoTest(t *testing.T) {
	packageRoot := filepath.Join("..", "..", "temporal", "nexus")
	manifestPath := filepath.Join(
		packageRoot,
		"testdata",
		"caller-closure-input-set",
		"manifest.json",
	)
	input, err := loadGenerationInput(manifestPath, packageRoot)
	require.NoError(t, err)

	generated, err := renderGeneratedTest(input)
	require.NoError(t, err)
	require.Contains(t, string(generated), "context.WithTimeout(context.Background(), 315*time.Second)")
	want, err := os.ReadFile(filepath.Join(packageRoot, generatedTestFileName))
	require.NoError(t, err)
	require.Equal(t, want, generated)
}

func TestRenderGeneratedRunnerTestPinsHermeticSubjectBeforeRuntimeIO(t *testing.T) {
	packageRoot := filepath.Join("..", "..", "temporal", "nexus")
	manifestPath := filepath.Join(
		packageRoot,
		"testdata",
		"caller-closure-input-set",
		"manifest.json",
	)
	input, err := loadGenerationInput(manifestPath, packageRoot)
	require.NoError(t, err)

	generated, err := renderGeneratedTest(input)
	require.NoError(t, err)
	encoded := string(generated)
	require.Contains(t, encoded, "func TestHermeticCIPortability(t *testing.T)")
	require.Contains(t, encoded, "runevaluation.CheckSubject")
	require.Contains(t, encoded, "ExperimentSHA256:")
	require.Contains(t, encoded, `"sha256:528c23e7807ee9833af65baeb32a8ec2d38ffacc1fae829600692d3d3eb93fd1"`)
	require.Contains(t, encoded, "ImplementationLinkID:")
	require.Contains(t, encoded, `"temporal.system.nexus.caller-closure.implementation-link"`)

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, "generated_test.go", generated, 0)
	require.NoError(t, err)
	var portabilityTest *ast.FuncDecl
	var pathRunner *ast.FuncDecl
	var pathFactory *ast.FuncDecl
	var evaluationHelper *ast.FuncDecl
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		require.NotEqual(t,
			"TestGeneratedWorkflowNexusQueryExactActionCallerClosureExecutesLocally",
			function.Name.Name,
		)
		if function.Name.Name == "TestHermeticCIPortability" {
			portabilityTest = function
		}
		if function.Name.Name == "runCallerClosurePath" {
			pathRunner = function
		}
		if function.Name.Name == "newCallerClosurePath" {
			pathFactory = function
		}
		if function.Name.Name == "runCallerClosureEvaluation" {
			evaluationHelper = function
		}
	}
	require.NotNil(t, portabilityTest)
	require.NotNil(t, pathRunner)
	require.NotNil(t, pathFactory)
	require.NotNil(t, evaluationHelper)
	runnerCalls := 0
	evaluationCalls := 0
	ast.Inspect(pathFactory.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if identifier, ok := call.Fun.(*ast.Ident); ok &&
			identifier.Name == "runCallerClosureEvaluation" {
			evaluationCalls++
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Run" {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if ok && identifier.Name == "runner" {
			runnerCalls++
		}
		return true
	})
	require.Equal(t, 1, runnerCalls)
	require.Equal(t, 1, evaluationCalls)
	commandCalls := 0
	ast.Inspect(evaluationHelper.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "CommandContext" {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if ok && identifier.Name == "exec" {
			commandCalls++
		}
		return true
	})
	require.Equal(t, 1, commandCalls)
	bodyStart := fileSet.Position(pathFactory.Body.Pos()).Offset
	bodyEnd := fileSet.Position(pathFactory.Body.End()).Offset
	body := encoded[bodyStart:bodyEnd]
	require.Less(t,
		strings.Index(body, "runevaluation.CheckSubject"),
		strings.Index(body, "runner.Run("),
	)
	require.Greater(t,
		strings.Index(body, "runCallerClosureEvaluation("),
		strings.Index(body, "runner.Run("),
	)
	helperStart := fileSet.Position(evaluationHelper.Body.Pos()).Offset
	helperEnd := fileSet.Position(evaluationHelper.Body.End()).Offset
	helperBody := encoded[helperStart:helperEnd]
	require.Contains(t, helperBody, `"umpire-check-local-run-evaluation"`)
	require.NotContains(t, helperBody, "runevaluation.Check(")
	require.Contains(t, helperBody, "exitError.ExitCode() != 2")
	require.NotContains(t, encoded, "artifact.PublishSet(")
	require.NotContains(t, encoded, "EvaluationProfile")
	require.NotContains(t, encoded, "EvaluationReceipt")
	require.NotContains(t, encoded, "ClaimAssessment")
	portabilityStart := fileSet.Position(portabilityTest.Body.Pos()).Offset
	portabilityEnd := fileSet.Position(portabilityTest.Body.End()).Offset
	portabilityBody := encoded[portabilityStart:portabilityEnd]
	require.Contains(t, portabilityBody, `require.Equal(t, "complete", run.Cleanup.Status)`)
	require.Contains(t, portabilityBody, `require.EqualValues(t, "0", run.Cleanup.OpenHandleCount)`)
}

func TestRunRegeneratesOnlyTheDeterministicGoTest(t *testing.T) {
	packageRoot := filepath.Join(hostTempDir(t), "nexus")
	fixtureRoot := filepath.Join(packageRoot, "testdata", "caller-closure-input-set")
	copyInputSet(t, fixtureRoot)
	manifestPath := filepath.Join(fixtureRoot, "manifest.json")

	require.NoError(t, run([]string{manifestPath, "--output", packageRoot}))
	first, err := os.ReadFile(filepath.Join(packageRoot, generatedTestFileName))
	require.NoError(t, err)
	require.NoError(t, run([]string{manifestPath, "--output", packageRoot}))
	second, err := os.ReadFile(filepath.Join(packageRoot, generatedTestFileName))
	require.NoError(t, err)
	require.Equal(t, first, second)

	entries, err := os.ReadDir(packageRoot)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, generatedTestFileName, entries[0].Name())
	require.Equal(t, "testdata", entries[1].Name())
}

func TestRenderGeneratedRunnerTestQuotesWhitespaceInEmbedPaths(t *testing.T) {
	packageRoot := filepath.Join(hostTempDir(t), "nexus")
	fixtureRoot := filepath.Join(packageRoot, "fixture with space")
	copyInputSet(t, fixtureRoot)
	input, err := loadGenerationInput(filepath.Join(fixtureRoot, "manifest.json"), packageRoot)
	require.NoError(t, err)

	generated, err := renderGeneratedTest(input)
	require.NoError(t, err)
	require.Contains(t, string(generated), `//go:embed "fixture with space/manifest.json"`)
}

func TestValidateEmbedRootRejectsPatternMetacharactersAndControls(t *testing.T) {
	for _, embedRoot := range []string{
		"fixture*",
		"fixture?",
		"fixture[one]",
		`fixture\one`,
		"fixture\nnext",
	} {
		t.Run(strings.ReplaceAll(embedRoot, "/", "-"), func(t *testing.T) {
			require.ErrorContains(t, validateEmbedRoot(embedRoot), "unsafe generated test fixture path")
		})
	}
}

func hostTempDir(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "..", "..", ".flow", "tmp")
	require.NoError(t, os.MkdirAll(root, 0o755))
	temporary, err := os.MkdirTemp(root, "fn19-8-generator-")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(temporary))
	})
	return temporary
}

func TestRunRejectsAnythingButTheGenerationGrammar(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"manifest.json"},
		{"manifest.json", "--run", "nexus"},
		{"manifest.json", "--output", "nexus", "--run"},
	} {
		require.ErrorContains(t, run(args), "expected <manifest> --output <package>")
	}
}

func copyInputSet(t *testing.T, destination string) {
	t.Helper()
	source := filepath.Join(
		"..", "..", "temporal", "nexus", "testdata", "caller-closure-input-set",
	)
	for _, relative := range []string{
		"manifest.json",
		filepath.Join("artifacts", "experiment.json"),
		filepath.Join("artifacts", "runtime-configuration.json"),
	} {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(destination, relative)), 0o755))
		encoded, err := os.ReadFile(filepath.Join(source, relative))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(destination, relative), encoded, 0o600))
	}
}
