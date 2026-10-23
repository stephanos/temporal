//go:build test_dep && integration

package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/replay"
	umpirebinding "go.temporal.io/server/common/testing/testpilot/temporal/binding"
	"google.golang.org/protobuf/proto"
)

// The negative control, the replay's early proof point: the control Case is bound under the
// cluster's default settings with no dynamic configuration in its Profile and run twice. Both Runs
// are in the admissible violated form, each admits as a replay subject under its own recorded
// identity, replays offline to its recorded Verdict, and the two share one Contract-relative key;
// the evidence core omits the Run's scaffolding events, named by instruction id, and neither the
// Run nor the Verdict is changed by reading it.
func TestTestpilotNexusControlForgedCompletionIsViolated(t *testing.T) {
	name := "nexus-workflow-control-forgedCompletion"
	caseBytes, err := os.ReadFile(testpilotFixturePath(name))
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

	// UMPIRE_CONTROL_RECORD writes the first Run for the current generated control Case's
	// offline Model-assessment probe; the two Runs still share the replay's correlated key.
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
		require.NotContains(t, runIDs, run.GetRunId(), "each Run is its own")
		runIDs = append(runIDs, run.GetRunId())

		recorded, err := os.ReadFile(path)
		require.NoError(t, err)
		subject, err := replay.Admit(env.Context(), caseBytes, recorded, prepare)
		require.NoError(t, err)
		require.Equal(t, live.prepared.Identity(), subject.Driver)
		require.True(t, proto.Equal(run.GetVerdict(), subject.Verdict))
		// Every correlated rule the Contract lowers the forged row's Property into is violated by the
		// one failed event, so the key names each of them with the same evidence and terminal state.
		require.Len(t, subject.Replay.Violations, len(caseSource.GetContract().GetCorrelated().GetRules()))
		require.Len(t, subject.Key.Rules, len(subject.Replay.Violations))
		for index, violation := range subject.Replay.Violations {
			require.NotEmpty(t, violation.CorrelatedKind, "the violating evidence is the failed event's kind")
			require.Equal(t, subject.Replay.Violations[0].Sequence, violation.Sequence)
			require.Equal(t, replay.CorrelatedViolated, subject.Key.Rules[index].Terminal)
			require.Equal(t, subject.Key.Rules[0].Evidence, subject.Key.Rules[index].Evidence)
			require.NotEmpty(t, subject.Key.Rules[index].Evidence)
		}
		keys = append(keys, subject.Key)
	}
	require.True(t, keys[0].Equal(keys[1]), "two Runs of the control share one key: %s / %s", keys[0], keys[1])
	t.Logf("control key: %s", keys[0])
}
