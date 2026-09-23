//go:build test_dep && integration

package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
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
