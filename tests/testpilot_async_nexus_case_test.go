//go:build test_dep && integration

package tests

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

const testpilotCleanupTimeout = 5 * time.Second

type testpilotLiveBinding struct {
	binding CaseBinding
	live    testpilotLiveCase
}

type testpilotLiveRunResult struct {
	environment int
	run         *testpilotpb.Run
	verdict     *testpilotpb.Verdict
	err         error
}

func TestTestpilotAsyncNexusCase(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, "async-nexus")
	caseSnapshot := proto.CloneOf(caseSource)

	environments := []CaseBinding{
		{Identity: "async-nexus-profile", Namespace: "umpire-async-nexus-a", TaskQueue: "umpire-async-nexus-queue-a", NexusEndpoint: "umpire-async-nexus-endpoint-a", CreateEndpoint: true},
		{Identity: "async-nexus-profile", Namespace: "umpire-async-nexus-b", TaskQueue: "umpire-async-nexus-queue-b", NexusEndpoint: "umpire-async-nexus-endpoint-b", CreateEndpoint: true},
	}
	bindings := make([]testpilotLiveBinding, len(environments))
	for index, environment := range environments {
		live := bindCase(t, env, caseSource, environment)
		bindings[index] = testpilotLiveBinding{binding: environment, live: live}
	}

	require.True(t, proto.Equal(caseSnapshot, caseSource))
	require.True(t, proto.Equal(bindings[0].live.prepared.Snapshot(), bindings[1].live.prepared.Snapshot()))
	require.True(t, proto.Equal(bindings[0].live.prepared.Snapshot().GetContract(), bindings[1].live.prepared.Snapshot().GetContract()))
	require.True(t, proto.Equal(bindings[0].live.prepared.Snapshot().GetProvenance(), bindings[1].live.prepared.Snapshot().GetProvenance()))
	require.Equal(t, bindings[0].live.prepared.Snapshot().GetCaseId(), bindings[1].live.prepared.Snapshot().GetCaseId())
	require.Equal(t, bindings[0].live.prepared.Snapshot().GetProgram().GetProgramId(), bindings[1].live.prepared.Snapshot().GetProgram().GetProgramId())
	require.Equal(t, bindings[0].live.prepared.Snapshot().GetContract().GetContractId(), bindings[1].live.prepared.Snapshot().GetContract().GetContractId())
	require.NotEqual(t, bindings[0].live.prepared.Identity().Bindings, bindings[1].live.prepared.Identity().Bindings)

	results := make(chan testpilotLiveRunResult, len(bindings)*2)
	var runs sync.WaitGroup
	for index, binding := range bindings {
		for range 2 {
			runs.Go(func() {
				run, verdict, err := binding.live.prepared.Run(env.Context(), binding.live.driver)
				results <- testpilotLiveRunResult{environment: index, run: run, verdict: verdict, err: err}
			})
		}
	}
	runs.Wait()
	close(results)

	runIDs := make(map[string]struct{}, len(bindings)*2)
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, result.run.GetStatus())
		require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, result.run.GetCleanup().GetStatus())
		require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, result.verdict.GetStatus())
		require.True(t, proto.Equal(result.verdict, result.run.GetVerdict()))
		// One rule verdict per scoped clause the checked Property lowered into, each answered by the
		// two recorded Nexus events the projection admitted as this operation's semantic steps.
		require.Len(t, result.verdict.GetRules(), 3)
		for _, rule := range result.verdict.GetRules() {
			require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus())
			require.Equal(t, result.verdict.GetSupportingEventSequences(), rule.GetSupportingEventSequences())
		}
		requireCorrelatedNexusHistoryEvidence(t, result.run, result.verdict.GetSupportingEventSequences(), bindings[result.environment].binding.NexusEndpoint)
		require.NotContains(t, runIDs, result.run.GetRunId())
		runIDs[result.run.GetRunId()] = struct{}{}

		_, err := bindings[result.environment].live.client.DescribeWorkflowExecution(env.Context(), result.run.GetRunId(), "")
		require.NoError(t, err)
		_, err = bindings[1-result.environment].live.client.DescribeWorkflowExecution(env.Context(), result.run.GetRunId(), "")
		var notFound *serviceerror.NotFound
		require.ErrorAs(t, err, &notFound)
	}
	require.True(t, proto.Equal(caseSnapshot, caseSource))
	for _, binding := range bindings {
		require.Equal(t, binding.live.profile.EnvironmentBindings, binding.live.driver.Snapshot().EnvironmentBindings)
		require.True(t, proto.Equal(caseSnapshot, binding.live.prepared.Snapshot()))
	}
}

func TestTestpilotAsyncNexusCaseMissingRemoteEndpoint(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, "async-nexus")
	// The endpoint the Case binds is deliberately not created, so the Nexus operation never
	// completes and the Run closes incomplete and inconclusive.
	live := bindCase(t, env, caseSource, CaseBinding{
		Identity: "async-nexus-profile", Namespace: "umpire-async-nexus-missing",
		TaskQueue: "umpire-async-nexus-queue-missing", NexusEndpoint: "umpire-async-nexus-endpoint-missing",
		CreateEndpoint: false,
	})

	run, verdict, err := live.prepared.Run(env.Context(), live.driver)
	require.NoError(t, err)
	require.Equal(t, testpilotpb.RUN_STATUS_INCOMPLETE, run.GetStatus())
	require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
	requireRunHasOutcome(t, run, "await-completion-authority", testpilotpb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT)
	_, err = live.client.DescribeWorkflowExecution(env.Context(), run.GetRunId(), "")
	require.NoError(t, err)
}

// TestTestpilotAsyncNexusCaseRunsFromItsFixtureNameAlone is what a new live test costs after this
// spec: name the fixture, assert the Verdict. Everything else -- the namespace, the queue, the
// Nexus endpoint, the derived Profile, the Driver -- follows from the Case's own bytes.
func TestTestpilotAsyncNexusCaseRunsFromItsFixtureNameAlone(t *testing.T) {
	env := newTestpilotTestEnvironment(t)

	run, verdict := runCase(t, env, "async-nexus")

	require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, run.GetStatus())
	require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Len(t, verdict.GetRules(), 3)
	for _, rule := range verdict.GetRules() {
		require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus())
	}
	requireCorrelatedNexusHistoryEvidence(t, run, verdict.GetSupportingEventSequences(),
		"umpire-async-nexus-endpoint")
}

func loadTestpilotCase(t testing.TB, name string) *testpilotpb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testcore", "testpilot", "testdata", name+"-case.json"))
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}
