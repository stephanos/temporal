//go:build test_dep && integration

package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tests/testcore"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
	"google.golang.org/protobuf/proto"
)

const testpilotCleanupTimeout = 5 * time.Second

// switchRun is one Case bound and run under one value of a switch: the environment that set the
// value's configuration, the bound Case, and what the Run produced.
type switchRun struct {
	value   testpilotcore.SwitchValue
	binding CaseBinding
	live    testpilotLiveCase
	run     *testpilotpb.Run
	verdict *testpilotpb.Verdict
}

// TestTestpilotAsyncNexusCase runs the one Case once per value of the Nexus implementation switch,
// each under its own environment constructed with that value's settings, the way the upstream Nexus
// suites run under HSM and CHASM. The Case bytes are the same under every value; the Profile is
// not, because it records the configuration it ran under; and a Verdict that differs between the
// values fails naming both, because that is a finding about the implementations, not a flake.
func TestTestpilotAsyncNexusCase(t *testing.T) {
	caseSource := loadTestpilotCase(t, "async-nexus")
	caseSnapshot := proto.CloneOf(caseSource)

	runs := make([]switchRun, 0, 2)
	for _, value := range testpilotcore.NexusImplementationSwitch() {
		// A subtest per value: each has its own dedicated cluster, constructed with the value's
		// settings, and its own resources named after the value.
		t.Run(value.Name, func(t *testing.T) {
			options := make([]testcore.TestOption, 0, len(value.Settings))
			for _, setting := range value.Settings {
				options = append(options, testcore.WithDynamicConfig(setting.Setting, setting.Value))
			}
			env := newTestpilotTestEnvironment(t, options...)
			binding := CaseBinding{
				Identity:  "async-nexus-profile-" + value.Name,
				Namespace: "umpire-async-nexus-" + value.Name, TaskQueue: "umpire-async-nexus-queue-" + value.Name,
				NexusEndpoint: "umpire-async-nexus-endpoint-" + value.Name, CreateEndpoint: true,
				DynamicConfig: value.Configuration(),
			}
			live := bindCase(t, env, caseSource, binding)
			require.True(t, proto.Equal(caseSnapshot, caseSource))
			require.Len(t, live.profile.Configuration, len(value.Settings))

			run, verdict, err := live.prepared.Run(env.Context(), live.driver)
			require.NoError(t, err)
			require.Equal(t, testpilotpb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
			require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
			require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
			require.True(t, proto.Equal(verdict, run.GetVerdict()))
			// One rule verdict per scoped clause the checked Property lowered into, each answered by
			// the two recorded Nexus events the projection admitted as this operation's semantic
			// steps. Two clauses: the Model records no Fact, because every step reaches a state
			// named after what happened.
			require.Len(t, verdict.GetRules(), 2)
			for _, rule := range verdict.GetRules() {
				require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus())
				require.Equal(t, verdict.GetSupportingEventSequences(), rule.GetSupportingEventSequences())
			}
			requireCorrelatedNexusHistoryEvidence(t, run, verdict.GetSupportingEventSequences(), binding.NexusEndpoint)
			_, err = live.client.DescribeWorkflowExecution(env.Context(), run.GetRunId(), "")
			require.NoError(t, err)
			require.Equal(t, live.profile.EnvironmentBindings, live.driver.Snapshot().EnvironmentBindings)
			require.True(t, proto.Equal(caseSnapshot, live.prepared.Snapshot()))
			runs = append(runs, switchRun{value: value, binding: binding, live: live, run: run, verdict: verdict})
		})
	}
	require.Len(t, runs, 2)

	// Identical Case bytes under every value: the same Case, Program, Contract and provenance; and
	// two Profiles, because each records the configuration it ran under.
	first, second := runs[0].live.prepared, runs[1].live.prepared
	require.True(t, proto.Equal(first.Snapshot(), second.Snapshot()))
	require.True(t, proto.Equal(first.Snapshot().GetContract(), second.Snapshot().GetContract()))
	require.True(t, proto.Equal(first.Snapshot().GetProvenance(), second.Snapshot().GetProvenance()))
	require.Equal(t, first.Snapshot().GetCaseId(), second.Snapshot().GetCaseId())
	require.Equal(t, first.Snapshot().GetProgram().GetProgramId(), second.Snapshot().GetProgram().GetProgramId())
	require.Equal(t, first.Snapshot().GetContract().GetContractId(), second.Snapshot().GetContract().GetContractId())
	require.NotEqual(t, first.Identity().Bindings, second.Identity().Bindings)
	require.NotEqual(t, runs[0].live.profile.Configuration, runs[1].live.profile.Configuration)
	require.NotEqual(t, runs[0].run.GetRunId(), runs[1].run.GetRunId())

	// Both Verdicts, reported together: a divergence names the switch, both values and both.
	results := make([]testpilotcore.SwitchVerdict, 0, len(runs))
	for _, run := range runs {
		results = append(results, testpilotcore.SwitchVerdict{Value: run.value.Name, Verdict: run.verdict})
	}
	require.NoError(t, testpilotcore.CheckSwitchAgreement(testpilotcore.NexusImplementationSwitchName, results))
	require.True(t, proto.Equal(caseSnapshot, caseSource))
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
	require.Equal(t, testpilotpb.RUN_DISPOSITION_INCOMPLETE, run.GetDisposition())
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

	require.Equal(t, testpilotpb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
	require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Len(t, verdict.GetRules(), 2)
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
