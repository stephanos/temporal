//go:build test_dep && integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/evaluation"
)

// assessSummary is the part of umpire-assess's summary the live proof reads.
type assessSummary struct {
	Status      string   `json:"status"`
	Reasons     []string `json:"reasons"`
	Receipt     string   `json:"receipt"`
	Publication string   `json:"publication"`
	Path        string   `json:"path"`
}

// assess runs umpire-assess once and returns its exit code and summary.
func assess(t *testing.T, ctx context.Context, binary, casePath, runPath, root, modelRoot string) (int, assessSummary) {
	t.Helper()
	command := exec.CommandContext(ctx, binary, "run",
		"--case", casePath, "--run", runPath, "--profile", "local-ephemeral",
		"--receipt-root", root, "--model-root", modelRoot)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	require.NotNil(t, command.ProcessState, "umpire-assess did not start: %v", err)
	var result assessSummary
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result), "stdout %q, stderr %q", stdout.String(), stderr.String())
	return command.ProcessState.ExitCode(), result
}

// The qualification's live proof: the caller Model's asyncCompletion Case is recorded by
// `umpire-run --record` against the test cluster, then assessed twice by `umpire-assess run` under
// local-ephemeral, a separate process given only the two files, a Profile name and a receipt root.
// It is accepted with one receipt, and the second assessment finds it already published. The
// negative control's pinned record is assessed the same way and rejected. Neither assessment is
// given an address, so neither can create or replay a Run, and the recorded Run is read, never
// changed.
func TestTestpilotAssessRecordedRuns(t *testing.T) {
	modelRoot, err := filepath.Abs(filepath.Join("..", "model"))
	require.NoError(t, err)
	runBinary := buildUmpireRun(t)
	assessBinary := buildUmpireCommand(t, "umpire-assess")
	env := newTestpilotTestEnvironment(t)
	ctx, cancel := context.WithTimeout(env.Context(), 5*time.Minute)
	defer cancel()

	casePath := filepath.Join("testcore", "testpilot", "testdata", "nexusCallerTests-asyncCompletion-case.json")
	runPath := filepath.Join(t.TempDir(), "run.json")
	record := exec.CommandContext(ctx, runBinary,
		"--case", casePath, "--record", runPath,
		"--grpc", env.FrontendGRPCAddress(), "--http", env.HttpAPIAddress(),
		"--namespace", "umpire-assess-caller", "--task-queue", "umpire-assess-caller-queue",
		"--nexus-endpoint", "umpire-assess-caller-endpoint", "--create", "--timeout", "2m")
	output, err := record.CombinedOutput()
	require.NotNil(t, record.ProcessState, "umpire-run did not start: %v", err)
	require.Equal(t, 0, record.ProcessState.ExitCode(), "umpire-run records a satisfied Run: %v %s", err, output)
	recorded, err := os.ReadFile(runPath)
	require.NoError(t, err)

	root := t.TempDir()
	code, first := assess(t, ctx, assessBinary, casePath, runPath, root, modelRoot)
	require.Equal(t, 0, code, "%+v", first)
	require.Equal(t, evaluation.DecisionAccepted, first.Status)
	require.Empty(t, first.Reasons)
	require.Equal(t, "published", first.Publication)
	published, err := os.ReadFile(first.Path)
	require.NoError(t, err)
	receipt, err := evaluation.DecodeReceipt(published)
	require.NoError(t, err)
	require.Equal(t, evaluation.DecisionAccepted, receipt.Decision)
	require.Equal(t, "RUN_DISPOSITION_COMPLETED", receipt.Run.Disposition)
	require.Equal(t, "VERDICT_STATUS_SATISFIED", receipt.Verdict.Status)

	code, second := assess(t, ctx, assessBinary, casePath, runPath, root, modelRoot)
	require.Equal(t, 0, code)
	require.Equal(t, "already-published", second.Publication)
	require.Equal(t, first.Receipt, second.Receipt)
	listed, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, listed, 1, "one receipt for one subject under one Profile")
	unchanged, err := os.ReadFile(runPath)
	require.NoError(t, err)
	require.Equal(t, recorded, unchanged, "assessment reads the recorded Run and never changes it")

	controlCase := filepath.Join("testcore", "testpilot", "testdata", "nexusCallerControl-forgedCompletion-case.json")
	controlRun := filepath.Join("..", "tools", "umpire", "replay", "testdata", "nexusCallerControl-forgedCompletion-run.json")
	code, control := assess(t, ctx, assessBinary, controlCase, controlRun, root, modelRoot)
	require.Equal(t, 1, code, "%+v", control)
	require.Equal(t, evaluation.DecisionRejected, control.Status)
	require.Equal(t, []string{"verdict-violated", "monitor-stopped"}, control.Reasons)
	require.NotEqual(t, first.Receipt, control.Receipt)
}
