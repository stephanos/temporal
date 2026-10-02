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
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/common/testing/testpilot/replay"
	umpirebinding "go.temporal.io/server/common/testing/testpilot/temporal/binding"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"go.temporal.io/server/tools/umpire/explore"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
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
		run, verdict := live.runRecording(t, env.Context(), caseBytes, path)
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
// minimized by removing inspections while retaining the async handle dependency, and its proposal is compiled and written under
// a scratch root -- proving the mechanism only, since the control's expected trace is the row the
// platform never takes. The IR replay bridge re-answers the proposal before its Case is rerun.
func TestTestpilotNexusControlReplaysThroughTheCommand(t *testing.T) {
	modelRoot, err := filepath.Abs(filepath.Join("..", "model"))
	require.NoError(t, err)
	bridgeBinary := buildUmpireCommand(t, "umpire-ir-bridge")
	runBinary := buildUmpireRun(t)
	replayBinary := buildUmpireCommand(t, "umpire-replay")

	model, err := umpiremodel.Load(filepath.Join(modelRoot, "ir", "nexus-control.json"))
	require.NoError(t, err)
	plan, err := explore.New(model, "nexusControl")
	require.NoError(t, err)
	candidate := plan.Candidates[0]
	require.Equal(t, "twice", candidate.Key)
	require.Empty(t, candidate.Rejection)
	casePath := filepath.Join(t.TempDir(), "control-case.json")
	require.NoError(t, os.WriteFile(casePath, candidate.Bytes, 0600))
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
	if bytes, readErr := os.ReadFile(runPath); readErr == nil {
		writeExplorationArtifact(t, "control-run.json", bytes)
	}
	writeExplorationArtifact(t, "control-case.json", candidate.Bytes)
	require.NotNil(t, record.ProcessState, "umpire-run did not start: %v", err)
	require.Equal(t, 1, record.ProcessState.ExitCode(), "umpire-run records a violated Run: %v %s", err, output)
	require.FileExists(t, runPath)

	root := filepath.Join(t.TempDir(), "proposals")
	command := exec.CommandContext(ctx, replayBinary, append([]string{"run",
		"--case", casePath, "--run", runPath,
		"--set", "nexusControl", "--target", candidate.Key, "--bridge", bridgeBinary,
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
	require.Equal(t, "minimized", report.Reduction.Status)
	require.Equal(t, replay.ProposalWritten, report.Proposal.Status, report.Proposal.Error)
	require.FileExists(t, report.Proposal.Written)
	require.Equal(t, replay.StatusReleased, report.Cleanup.Status)
	proposal, err := os.ReadFile(report.Proposal.Written)
	require.NoError(t, err)
	retained, err := explore.ReadProposal(proposal)
	require.NoError(t, err)
	require.Less(t, len(retained.Actions), len(candidate.Actions))
	recovered := exec.CommandContext(ctx, bridgeBinary, "proposal", report.Proposal.Written)
	recoveredBytes, err := recovered.Output()
	require.NoError(t, err)
	require.Equal(t, append(bytes.Clone(retained.Bytes), '\n'), recoveredBytes)
	retainedPath := filepath.Join(t.TempDir(), "retained-case.json")
	require.NoError(t, os.WriteFile(retainedPath, retained.Bytes, 0600))
	retainedRun := filepath.Join(t.TempDir(), "retained-run.json")
	replayProposal := exec.CommandContext(ctx, runBinary, append([]string{"--case", retainedPath, "--record", retainedRun, "--timeout", "2m"}, deployment...)...)
	output, err = replayProposal.CombinedOutput()
	require.NotNil(t, replayProposal.ProcessState, "proposal did not start: %v", err)
	require.Equal(t, 1, replayProposal.ProcessState.ExitCode(), "%v %s", err, output)
	recorded, err := os.ReadFile(runPath)
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(recorded)
	require.NoError(t, err)
	sourceRoot, err := filepath.Abs("..")
	require.NoError(t, err)
	trace, err := explore.RenderTrace(candidate, plan.Query, decoded.Run, nil, sourceRoot)
	require.NoError(t, err)
	writeExplorationArtifact(t, "control-case.json", candidate.Bytes)
	writeExplorationArtifact(t, "control-run.json", recorded)
	writeExplorationArtifact(t, "control-trace.html", trace)
	writeExplorationArtifact(t, "reduction.json", stdout.Bytes())
	writeExplorationArtifact(t, "regression.json", proposal)
	writeExplorationArtifact(t, "retained-case.json", retained.Bytes)
	recorded, err = os.ReadFile(retainedRun)
	require.NoError(t, err)
	writeExplorationArtifact(t, "retained-run.json", recorded)
	resolved := umpirebinding.Deployment{GRPCAddress: env.FrontendGRPCAddress(), HTTPAddress: env.HttpAPIAddress(), Namespace: namespace, TaskQueue: taskQueue, NexusEndpoint: endpoint}
	admitted, err := replay.Admit(ctx, retained.Bytes, recorded, func(identity string, source *testpilotpb.Case) (*testpilot.PreparedCase, error) {
		prepared, err := umpirebinding.Prepare(resolved, taskQueue+"-handler", identity, source)
		if err != nil {
			return nil, err
		}
		return prepared.Case, nil
	})
	require.NoError(t, err)
	require.Equal(t, report.Key, admitted.Key.String())
}
