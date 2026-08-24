package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/agentworkflow/internal/agentworkflow"
	projectconfig "go.temporal.io/server/tools/agentworkflow/internal/project"
	"go.temporal.io/server/tools/agentworkflow/internal/recipe"
	"gopkg.in/yaml.v3"
)

func TestInitAndConfigExplainUserFlow(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run(context.Background(), []string{"init", "--project", root}, &stdout, &stderr); code != exitOK {
		t.Fatalf("init exit = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	config := filepath.Join(root, ".agentworkflow", "config.yml")
	if _, err := os.Stat(config); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"config", "explain", "--project", root}, &stdout, &stderr); code != exitOK {
		t.Fatalf("explain exit = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "schema: agentworkflow.resolved-config/v1") || !strings.Contains(stdout.String(), "name: test") || strings.HasPrefix(strings.TrimSpace(stdout.String()), "{") {
		t.Fatalf("resolved configuration = %s", stdout.String())
	}
	if code := Run(context.Background(), []string{"init", "--project", root}, &stdout, &stderr); code != exitFailure {
		t.Fatalf("second init exit = %d, want failure", code)
	}
}

func TestCLIUsageAndStableOutcomeCategories(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run(context.Background(), nil, &stdout, &stderr); code != exitUsage {
		t.Fatalf("empty CLI exit = %d", code)
	}
	if code := Run(context.Background(), []string{"unknown"}, &stdout, &stderr); code != exitUsage {
		t.Fatalf("unknown CLI exit = %d", code)
	}
	cases := map[agentworkflow.Outcome]int{
		agentworkflow.OutcomeSucceeded:    exitOK,
		agentworkflow.OutcomeNeedsChanges: exitNeedsChange,
		agentworkflow.OutcomeUnsupported:  exitUnsupported,
		agentworkflow.OutcomeTimedOut:     exitInterrupted,
		agentworkflow.OutcomeCorrupt:      exitFailure,
	}
	for outcome, expected := range cases {
		if actual := outcomeExit(outcome); actual != expected {
			t.Fatalf("outcome %s exit = %d, want %d", outcome, actual, expected)
		}
	}
}

func TestCLIHelpAliasesExposeCommandTree(t *testing.T) {
	cases := []struct {
		name      string
		arguments []string
		contains  []string
	}{
		{name: "help command", arguments: []string{"help"}, contains: []string{"Usage:", "init", "doctor", "run", "resume", "inspect", "report", "diff", "apply", "config"}},
		{name: "short help flag", arguments: []string{"-h"}, contains: []string{"Usage:"}},
		{name: "legacy long help flag", arguments: []string{"-help"}, contains: []string{"Usage:"}},
		{name: "long help flag", arguments: []string{"--help"}, contains: []string{"Usage:"}},
		{name: "command help flag", arguments: []string{"run", "--help"}, contains: []string{"Usage:", "--criterion", "--backend-arg", "--model"}},
		{name: "nested help command", arguments: []string{"help", "config", "explain"}, contains: []string{"Usage:", "agentworkflow config explain", "--project", "--config", "--target"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(context.Background(), testCase.arguments, &stdout, &stderr)

			require.Equal(t, exitOK, code)
			require.Empty(t, stderr.String())
			for _, expected := range testCase.contains {
				require.Contains(t, stdout.String(), expected)
			}
		})
	}
}

func TestCLIAcceptsLegacySingleDashLongFlags(t *testing.T) {
	projectRoot := t.TempDir()
	configPath := writeCLIConfig(t, projectRoot, true, "")
	initRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(initRoot, "go.mod"), []byte("module example.com/init\n"), 0o600))
	storeRoot := t.TempDir()
	cases := []struct {
		name           string
		arguments      []string
		wantCode       int
		stdoutContains string
		stderrContains string
	}{
		{
			name: "init local flags",
			arguments: []string{
				"init", "-project=" + initRoot, "-config", ".agentworkflow/custom.yml",
			},
			wantCode:       exitOK,
			stdoutContains: "wrote",
		},
		{
			name: "doctor local and common flags",
			arguments: []string{
				"doctor", "-project", projectRoot, "-config=" + configPath, "-target=",
				"-qualified", "-backend-command=wrapper",
			},
			wantCode:       exitUnsupported,
			stderrContains: "--qualified cannot be combined",
		},
		{
			name: "run local and common flags",
			arguments: []string{
				"run", "-project", projectRoot, "-config=" + configPath, "-target=",
				"-task", "objective", "-task-file=task.txt", "-assurance=fast", "-apply",
				"-json=false", "-criterion=done", "-store=" + storeRoot, "-backend=codex",
				"-backend-command=wrapper", "-backend-arg=value", "-model=test-model", "-qualified",
			},
			wantCode:       exitUsage,
			stderrContains: "--task and --task-file are mutually exclusive",
		},
		{
			name: "resume flags",
			arguments: []string{
				"resume", "-json=false", "-store=" + storeRoot, "-qualified", "-backend-command=wrapper", "run-id",
			},
			wantCode:       exitUnsupported,
			stderrContains: "--qualified cannot be combined",
		},
		{
			name: "inspect flags",
			arguments: []string{
				"inspect", "-json=false", "-store=" + storeRoot, "-qualified", "-backend-command=wrapper", "run-id",
			},
			wantCode:       exitUnsupported,
			stderrContains: "--qualified cannot be combined",
		},
		{
			name: "report flags",
			arguments: []string{
				"report", "-json=true", "-store=" + storeRoot, "-qualified", "-backend-command=wrapper", "run-id",
			},
			wantCode:       exitUnsupported,
			stderrContains: "--qualified cannot be combined",
		},
		{
			name: "diff flags",
			arguments: []string{
				"diff", "-json", "-store=" + storeRoot, "-qualified", "-backend-command=wrapper", "run-id",
			},
			wantCode:       exitUnsupported,
			stderrContains: "--qualified cannot be combined",
		},
		{
			name: "apply flags",
			arguments: []string{
				"apply", "-store=" + storeRoot, "-qualified", "-backend-command=wrapper", "run-id",
			},
			wantCode:       exitUnsupported,
			stderrContains: "--qualified cannot be combined",
		},
		{
			name: "nested config flags",
			arguments: []string{
				"config", "explain", "-project=" + projectRoot, "-config", configPath, "-target=",
			},
			wantCode:       exitOK,
			stdoutContains: "schema: agentworkflow.resolved-config/v1",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(context.Background(), testCase.arguments, &stdout, &stderr)

			require.Equal(t, testCase.wantCode, code, "stdout=%q stderr=%q", stdout.String(), stderr.String())
			if testCase.stdoutContains == "" {
				require.Empty(t, stdout.String())
			} else {
				require.Contains(t, stdout.String(), testCase.stdoutContains)
			}
			if testCase.stderrContains == "" {
				require.Empty(t, stderr.String())
			} else {
				require.Contains(t, stderr.String(), testCase.stderrContains)
			}
		})
	}
}

func TestCLILegacyFlagNormalizationPreservesValuesAndSeparator(t *testing.T) {
	cases := []struct {
		name      string
		arguments []string
		wantCode  int
		contains  string
	}{
		{
			name:      "legacy flag value begins with dash",
			arguments: []string{"run", "-task", "-project", "-task-file=task.txt"},
			wantCode:  exitUsage,
			contains:  "--task and --task-file are mutually exclusive",
		},
		{
			name:      "double dash flag value begins with dash",
			arguments: []string{"run", "--task", "-project", "--task-file=task.txt"},
			wantCode:  exitUsage,
			contains:  "--task and --task-file are mutually exclusive",
		},
		{
			name:      "separator retains positional single dash value",
			arguments: []string{"run", "--", "-task"},
			wantCode:  exitFailure,
			contains:  ".agentworkflow/config.yml",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(context.Background(), testCase.arguments, &stdout, &stderr)

			require.Equal(t, testCase.wantCode, code, "stdout=%q stderr=%q", stdout.String(), stderr.String())
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(), testCase.contains)
		})
	}
}

func TestCLIUsageErrorsStayOnStderr(t *testing.T) {
	projectRoot := t.TempDir()
	cases := []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "missing command", contains: "Usage:"},
		{name: "unknown command", arguments: []string{"unknown"}, contains: "unknown command"},
		{name: "unknown flag", arguments: []string{"resume", "--unknown"}, contains: "unknown flag"},
		{name: "unknown legacy long flag", arguments: []string{"resume", "-unknown"}, contains: "unknown"},
		{name: "missing positional", arguments: []string{"resume"}, contains: "accepts 1 arg(s)"},
		{name: "extra positional", arguments: []string{"init", "--project", projectRoot, "extra"}, contains: "accepts 0 arg(s)"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(context.Background(), testCase.arguments, &stdout, &stderr)

			require.Equal(t, exitUsage, code)
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(), testCase.contains)
			require.Contains(t, stderr.String(), "Usage:")
		})
	}
}

func TestCLIClassifiesOutputFailure(t *testing.T) {
	t.Run("stdout", func(t *testing.T) {
		code := Run(context.Background(), []string{"help"}, failingWriter{}, io.Discard)
		require.Equal(t, exitFailure, code)
	})
	t.Run("stderr", func(t *testing.T) {
		code := Run(context.Background(), []string{"unknown"}, io.Discard, failingWriter{})
		require.Equal(t, exitFailure, code)
	})
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("fixture output failure")
}

func TestDefaultStoreHonorsExplicitAgentworkflowHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AGENTWORKFLOW_HOME", root)
	if actual := defaultStore(); actual != root {
		t.Fatalf("default store = %q, want %q", actual, root)
	}
}

func TestCLIEndToEndRunInspectDiffAndApply(t *testing.T) {
	projectRoot := t.TempDir()
	storeRoot := t.TempDir()
	configPath := writeCLIConfig(t, projectRoot, true, "configured-model")
	providerFlags := []string{
		"--backend", "codex", "--backend-command", os.Args[0],
		"--backend-arg=-test.run=TestCLIProvider", "--backend-arg=--", "--backend-arg=codex-model=override-model",
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	doctorArgs := append([]string{"doctor", "--project", projectRoot, "--config", configPath}, providerFlags...)
	if code := Run(context.Background(), doctorArgs, &stdout, &stderr); code != exitOK {
		t.Fatalf("doctor exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	runArgs := []string{"run", "--project", projectRoot, "--config", configPath, "--store", storeRoot, "--json", "--model", "override-model"}
	runArgs = append(runArgs, providerFlags...)
	runArgs = append(runArgs, "write", "result.txt")
	if code := Run(context.Background(), runArgs, &stdout, &stderr); code != exitOK {
		t.Fatalf("run exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	require.Empty(t, stderr.String())
	var result agentworkflow.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Outcome != agentworkflow.OutcomeSucceeded {
		t.Fatalf("result = %#v", result)
	}
	stdout.Reset()
	if code := Run(context.Background(), []string{"inspect", "--store", storeRoot, "--json", string(result.RunID)}, &stdout, &stderr); code != exitOK {
		t.Fatalf("inspect exit=%d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	if code := Run(context.Background(), []string{"diff", "--store", storeRoot, string(result.RunID)}, &stdout, &stderr); code != exitOK || !strings.Contains(stdout.String(), "A  result.txt") {
		t.Fatalf("diff exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	if code := Run(context.Background(), []string{"apply", "--store", storeRoot, string(result.RunID)}, &stdout, &stderr); code != exitOK {
		t.Fatalf("apply exit=%d stderr=%q", code, stderr.String())
	}
	if data, err := os.ReadFile(filepath.Join(projectRoot, "result.txt")); err != nil || string(data) != "good" {
		t.Fatalf("applied file = %q, %v", data, err)
	}
}

func TestRunApplyRejectsDisabledAdmittedStageBeforeBackendProbe(t *testing.T) {
	projectRoot := t.TempDir()
	configPath := writeCLIConfig(t, projectRoot, false, "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"run", "--project", projectRoot, "--config", configPath, "--task", "write result.txt", "--apply",
		"--backend", "codex", "--backend-command", "missing-agentworkflow-provider",
	}, &stdout, &stderr)
	if code != exitUnsupported || !strings.Contains(stderr.String(), "apply stage is disabled") || strings.Contains(stderr.String(), "missing-agentworkflow-provider") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestDoctorReportsMissingConfigurationBeforeBackendProbe(t *testing.T) {
	projectRoot := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"doctor", "--project", projectRoot, "--backend-command", "missing-agentworkflow-provider",
	}, &stdout, &stderr)
	if code != exitFailure || !strings.Contains(stderr.String(), ".agentworkflow/config.yml") || strings.Contains(stderr.String(), "missing-agentworkflow-provider") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestQualifiedBackendRejectsExecutableAndArgumentOverrides(t *testing.T) {
	name, model, qualified := "codex", "", true
	command := "wrapper"
	arguments := stringList{}
	if _, err := makeBackend(&name, &command, &arguments, &model, &qualified); !errors.Is(err, agentworkflow.ErrUnsupported) {
		t.Fatalf("qualified command override error = %v", err)
	}
	command = ""
	arguments = stringList{"--dangerously-bypass-approvals-and-sandbox"}
	if _, err := makeBackend(&name, &command, &arguments, &model, &qualified); !errors.Is(err, agentworkflow.ErrUnsupported) {
		t.Fatalf("qualified argument override error = %v", err)
	}
}

func TestResumeModelOverrideChangesBackendIdentity(t *testing.T) {
	name, command, qualified := "codex", os.Args[0], false
	arguments := stringList{"-test.run=TestCLIProvider", "--", "codex"}
	oldModel, newModel := "old-model", "new-model"
	oldBackend, err := makeBackend(&name, &command, &arguments, &oldModel, &qualified)
	require.NoError(t, err)
	newBackend, err := makeBackend(&name, &command, &arguments, &newModel, &qualified)
	require.NoError(t, err)

	oldInfo, err := oldBackend.Describe(context.Background())
	require.NoError(t, err)
	newInfo, err := newBackend.Describe(context.Background())
	require.NoError(t, err)
	require.NotEqual(t, oldInfo.ConfigurationDigest, newInfo.ConfigurationDigest)
}

//nolint:errcheck,revive // Writes and exits intentionally model a real provider subprocess.
func TestCLIProvider(t *testing.T) {
	separator := testArgumentIndex(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	arguments := os.Args[separator+2:]
	if testHasArgument(arguments, "--version") {
		fmt.Fprintln(os.Stdout, "codex-cli cli-test")
		os.Exit(0)
	}
	expectedModel := strings.TrimPrefix(os.Args[separator+1], "codex-model=")
	if expectedModel != os.Args[separator+1] && testArgumentValue(arguments, "--model") != expectedModel {
		fmt.Fprintln(os.Stderr, "configured stage model was not forwarded")
		os.Exit(23)
	}
	promptData, _ := io.ReadAll(os.Stdin)
	prompt := string(promptData)
	output, err := cliStructuredOutput(prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(20)
	}
	if strings.HasPrefix(prompt, "Agentworkflow phase: implement\n") {
		if err := os.WriteFile("result.txt", []byte("good"), 0o600); err != nil {
			os.Exit(21)
		}
	}
	outputPath := testArgumentValue(arguments, "--output-last-message")
	if err := os.WriteFile(outputPath, output, 0o600); err != nil {
		os.Exit(22)
	}
	fmt.Fprintln(os.Stdout, `{"type":"thread.started","thread_id":"session-cli"}`)
	fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
	fmt.Fprintln(os.Stdout, `{"type":"item.completed","item":{"type":"agent_message","text":"done"}}`)
	fmt.Fprintln(os.Stdout, `{"type":"turn.completed","usage":{}}`)
	os.Exit(0)
}

//nolint:errcheck,revive // Writes and exits intentionally model a direct-check subprocess.
func TestCLICheck(t *testing.T) {
	separator := testArgumentIndex(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	data, err := os.ReadFile(os.Args[separator+1])
	if err != nil || string(data) != "good" {
		os.Exit(1)
	}
	os.Exit(0)
}

func cliStructuredOutput(prompt string) ([]byte, error) {
	switch {
	case strings.HasPrefix(prompt, "Agentworkflow phase: discover\n"):
		return json.Marshal(map[string]any{"summary": "fixture", "languages": []string{}, "build_systems": []string{}, "architecture": []string{}, "risks": []string{}})
	case strings.HasPrefix(prompt, "Agentworkflow phase: plan\n"):
		return json.Marshal(map[string]any{
			"understanding": "write result", "assumptions": []string{}, "risks": []string{}, "tradeoffs": []string{},
			"steps": []any{map[string]any{"description": "write", "files": []string{"result.txt"}, "criteria": []int{1}, "verification": []string{"test"}}},
		})
	case strings.HasPrefix(prompt, "Agentworkflow phase: implement\n"):
		return json.Marshal(map[string]any{"summary": "done", "files": []string{"result.txt"}, "tests": []string{}})
	case strings.HasPrefix(prompt, "Agentworkflow phase: review\n"):
		lens := "correctness"
		begin := strings.Index(prompt, "BEGIN_CONTEXT\n")
		end := strings.Index(prompt, "\nEND_CONTEXT")
		if begin >= 0 && end > begin {
			var value map[string]any
			_ = json.Unmarshal([]byte(prompt[begin+len("BEGIN_CONTEXT\n"):end]), &value)
			if resolved, ok := value["lens"].(string); ok {
				lens = resolved
			}
		}
		return json.Marshal(map[string]any{"lens": lens, "summary": "clean", "findings": []any{}})
	default:
		return nil, fmt.Errorf("unexpected prompt %.80s", prompt)
	}
}

func writeCLIConfig(t *testing.T, root string, applyEnabled bool, codexModel string) string {
	t.Helper()
	profile, err := projectconfig.Starter(root)
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	profile.Source.Exclude = nil
	profile.Checks = []projectconfig.ProfileCheck{{
		Name: "test", Command: []string{os.Args[0], "-test.run=TestCLICheck", "--", "result.txt"},
		Timeout: projectconfig.Duration{Duration: time.Minute}, Required: true, Enabled: &enabled,
	}}
	profile.Environment = projectconfig.Environment{Allow: []string{"PATH"}}
	for index := range profile.Workflow.Stages {
		stage := &profile.Workflow.Stages[index]
		if string(stage.Kind) == string(agentworkflow.StageApply) {
			stage.Enabled = applyEnabled
		}
		if codexModel != "" && stage.Kind != recipe.Check && stage.Kind != recipe.Apply {
			stage.Models = recipe.Models{"codex": codexModel}
		}
	}
	data, err := yaml.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".agentworkflow", "config.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func testArgumentIndex(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func testHasArgument(values []string, target string) bool {
	return testArgumentIndex(values, target) >= 0
}

func testArgumentValue(values []string, name string) string {
	index := testArgumentIndex(values, name)
	if index < 0 || index+1 >= len(values) {
		return ""
	}
	return values[index+1]
}
