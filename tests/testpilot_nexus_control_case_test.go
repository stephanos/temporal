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
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	umpirebinding "go.temporal.io/server/tools/umpire/binding"
	"go.temporal.io/server/tools/umpire/replay"
	"google.golang.org/protobuf/proto"
)

// The negative control, the replay's early proof point: the control Case is bound under the
// cluster's default settings with no dynamic configuration in its Profile and run twice. Both Runs
// are in the admissible violated form, each admits as a replay subject under its own recorded
// identity, replays offline to its recorded Verdict, and the two share one Contract-relative key;
// the evidence core omits the Run's scaffolding events, named by instruction id, and neither the
// Run nor the Verdict is changed by reading it.
func TestTestpilotNexusControlForgedCompletionIsViolated(t *testing.T) {
	name := "nexusCallerControl-forgedCompletion"
	caseBytes, err := os.ReadFile(filepath.Join("testcore", "testpilot", "testdata", name+"-case.json"))
	require.NoError(t, err)
	caseSource := loadTestpilotCase(t, name)
	env := newTestpilotTestEnvironment(t)
	binding := CaseBinding{
		Identity: name + "-profile", Namespace: "umpire-control", TaskQueue: "umpire-control-queue",
		NexusEndpoint: "umpire-control-endpoint", CreateEndpoint: true,
	}
	live := bindCase(t, env, caseSource, binding)
	require.Empty(t, live.profile.Configuration, "the control is recorded without dynamic configuration")

	// The replay prepares under the recorded Profile name with the deployment's names through
	// binding.Prepare, the path umpire-replay takes, and must arrive at the identity the live
	// binding recorded.
	deployment := umpirebinding.Deployment{Namespace: binding.Namespace, TaskQueue: binding.TaskQueue, NexusEndpoint: binding.NexusEndpoint}
	prepare := func(identity string, source *testpilotpb.Case) (*testpilot.PreparedCase, error) {
		prepared, err := umpirebinding.Prepare(deployment, umpirebinding.HandlerQueue(deployment), identity, source)
		if err != nil {
			return nil, err
		}
		return prepared.Case, nil
	}

	// The first Run's record is the replay package's pin of the correlated key when
	// UMPIRE_CONTROL_RECORD names the file to write; the pin's test says where it lives.
	dir := t.TempDir()
	paths := map[string]string{"first": os.Getenv("UMPIRE_CONTROL_RECORD")}
	var keys []replay.ViolationKey
	var runIDs []string
	for _, attempt := range []string{"first", "second"} {
		path := paths[attempt]
		if path == "" {
			path = filepath.Join(dir, attempt+"-run.json")
		}
		run, verdict := live.runRecording(t, env.Context(), path)
		require.Equal(t, testpilotpb.RUN_DISPOSITION_STOPPED_BY_MONITOR, run.GetDisposition(), "diagnostics: %v", run.GetDiagnostics())
		require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
		require.Equal(t, testpilotpb.VERDICT_STATUS_VIOLATED, verdict.GetStatus())
		require.True(t, proto.Equal(verdict, run.GetVerdict()))
		before := proto.CloneOf(run)
		require.NotContains(t, runIDs, run.GetRunId(), "each Run is its own")
		runIDs = append(runIDs, run.GetRunId())

		recorded, err := os.ReadFile(path)
		require.NoError(t, err)
		subject, err := replay.Admit(env.Context(), caseBytes, recorded, prepare)
		require.NoError(t, err)
		require.Equal(t, live.prepared.Identity(), subject.Driver)
		require.True(t, proto.Equal(run.GetVerdict(), subject.Verdict))
		// The forged row's Property and the completed event's fact are both violated by the one
		// failed event, so the key names two rules with the same evidence and terminal state.
		require.Len(t, subject.Replay.Violations, 2)
		require.Len(t, subject.Key.Rules, len(subject.Replay.Violations))
		for index, violation := range subject.Replay.Violations {
			require.NotEmpty(t, violation.CorrelatedKind, "the violating evidence is the failed event's kind")
			require.Equal(t, subject.Replay.Violations[0].Sequence, violation.Sequence)
			require.Equal(t, replay.CorrelatedViolated, subject.Key.Rules[index].Terminal)
			require.Equal(t, subject.Key.Rules[0].Evidence, subject.Key.Rules[index].Evidence)
			require.NotEmpty(t, subject.Key.Rules[index].Evidence)
		}
		keys = append(keys, subject.Key)

		core := replay.EvidenceCore(verdict)
		require.NotEmpty(t, core)
		outside := replay.OutsideCore(run, core)
		require.NotEmpty(t, outside, "the realization's scaffolding supports no violated rule")
		for _, event := range outside {
			require.NotEmpty(t, event.InstructionID)
			require.NotContains(t, core, event.Sequence)
		}
		require.True(t, proto.Equal(before, run), "reading the core changes nothing")
	}
	require.True(t, keys[0].Equal(keys[1]), "two Runs of the control share one key: %s / %s", keys[0], keys[1])
	t.Logf("control key: %s", keys[0])
}

// The negative control end to end through the commands, the replay's live proof: the test
// provisions its own namespace, queues and endpoint with no Driver of its own, `umpire-run
// --record` records one Run against them, and `umpire-replay run` is invoked while they are still
// held, with the same names and without `--create`, so the recorded identity is the one the replay
// prepares under. The subject is admitted, replayed offline, reproduced on two fresh Runs, found
// irreducible (its one prefix step is the schedule), and its proposal is compiled and written under
// a scratch root -- proving the mechanism only, since the control's expected trace is the row the
// platform never takes. The Lean replay bridge is required: without it the test fails.
func TestTestpilotNexusControlReplaysThroughTheCommand(t *testing.T) {
	modelRoot, err := filepath.Abs(filepath.Join("..", "model"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(modelRoot, ".lake", "build", "bin", "umpire-replay-bridge"))
	require.NoError(t, err, "the replay bridge is not built; make umpire-check-live-tests builds it")
	runBinary := buildUmpireRun(t)
	replayBinary := buildUmpireReplay(t)

	name := "nexusCallerControl-forgedCompletion"
	casePath := filepath.Join("testcore", "testpilot", "testdata", name+"-case.json")
	env := newTestpilotTestEnvironment(t)
	namespace, taskQueue, endpoint := "umpire-control-replay", "umpire-control-replay-queue", "umpire-control-replay-endpoint"
	release, err := provision.Create(env.Context(), provision.Clients{
		Workflow: env.FrontendClient(), Operator: env.OperatorClient(),
	}, provision.Resources{
		Namespace: namespace, TaskQueue: taskQueue, NexusEndpoint: endpoint, NexusTaskQueue: taskQueue + "-handler",
		RetainNamespace: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), testpilotCleanupTimeout)
		defer cancel()
		require.NoError(t, release(ctx))
	})
	deployment := []string{
		"--grpc", env.FrontendGRPCAddress(), "--http", env.HttpAPIAddress(),
		"--namespace", namespace, "--task-queue", taskQueue, "--nexus-endpoint", endpoint,
	}

	ctx, cancel := context.WithTimeout(env.Context(), 5*time.Minute)
	defer cancel()
	runPath := filepath.Join(t.TempDir(), "run.json")
	record := exec.CommandContext(ctx, runBinary, append([]string{"--case", casePath, "--record", runPath, "--timeout", "2m"}, deployment...)...)
	output, err := record.CombinedOutput()
	require.Equal(t, 1, record.ProcessState.ExitCode(), "umpire-run records a violated Run: %v %s", err, output)
	require.FileExists(t, runPath)

	root := filepath.Join(t.TempDir(), "proposals")
	command := exec.CommandContext(ctx, replayBinary, append([]string{"run",
		"--case", casePath, "--run", runPath,
		"--set", "nexusCallerControl", "--query", "forgedCompletion",
		"--model-root", modelRoot, "--promotion-root", root, "--timeout", "5m",
	}, deployment...)...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	require.NoError(t, command.Run(), "umpire-replay exited %d: %s\n%s", command.ProcessState.ExitCode(), stderr.String(), stdout.String())

	var report replay.Report
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &report), stdout.String())
	require.Equal(t, replay.StatusAdmitted, report.Admission.Status)
	require.Equal(t, replay.StatusReproduced, report.SemanticReplay.Status)
	require.Contains(t, report.Key, "temporal.nexus.control.property.forgedSuccess")
	require.Len(t, report.Identity, 64)
	require.Equal(t, replay.ClassReproduced, report.Reproduction.Class, "reruns: %+v", report.Reproduction.Reruns)
	require.Equal(t, "irreducible", report.Reduction.Status)
	require.Equal(t, replay.ProposalWritten, report.Proposal.Status, report.Proposal.Error)
	require.FileExists(t, report.Proposal.Written)
	require.Equal(t, replay.StatusReleased, report.Cleanup.Status)
}

// buildUmpireReplay builds the replay command the way a developer would, so the live proof runs the
// real binary rather than an in-process call.
func buildUmpireReplay(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "umpire-replay")
	build := exec.Command("go", "build", "-o", binary, "go.temporal.io/server/tools/umpire/cmd/umpire-replay")
	build.Env = os.Environ()
	output, err := build.CombinedOutput()
	require.NoError(t, err, "build umpire-replay: %s", output)
	return binary
}
