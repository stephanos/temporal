//go:build test_dep && integration

package tests

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/chasm/lib/activity"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tests/testcore"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
	"go.temporal.io/server/tools/umpire/explore"
	"go.temporal.io/server/tools/umpire/lower"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

var generatedCaseDirectory = filepath.Join("..", "model", "cases")

func generatedFixture(t *testing.T, file string) *testpilotcore.ModelCase {
	t.Helper()
	entries, err := testpilotcore.GeneratedCases(generatedCaseDirectory)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.File == file {
			fixture, err := testpilotcore.LoadGeneratedCase(generatedCaseDirectory, entry)
			require.NoError(t, err)
			return fixture
		}
	}
	t.Fatalf("no generated Case %s", file)
	return nil
}

func activityEnvironment(t *testing.T) *testcore.TestEnv {
	t.Helper()
	return newTestpilotTestEnvironment(t,
		testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true),
		testcore.WithDynamicConfig(activity.Enabled, true),
		testcore.WithDynamicConfig(activity.EnableStandaloneActivityOperatorCommands, true))
}

type generatedRun struct {
	run        *testpilotspb.Run
	verdict    *testpilotspb.Verdict
	assessment *testpilot.Assessment
	err        error
}

func runGeneratedCase(t *testing.T, env *testcore.TestEnv, fixture *testpilotcore.ModelCase, live testpilotLiveCase) generatedRun {
	t.Helper()
	assessed, err := live.prepared.WithAssessment(fixture.Assessment)
	if err != nil {
		return generatedRun{err: err}
	}
	run, verdict, assessment, err := assessed.Run(t.Context(), live.driver)
	return generatedRun{run, verdict, assessment, err}
}

func requireGeneratedAssessment(t *testing.T, fixture *testpilotcore.ModelCase, live testpilotLiveCase, result generatedRun) {
	t.Helper()
	if dir := os.Getenv(umpireRepeatRunDirVariable); dir != "" && result.run != nil {
		require.NoError(t, recordedrun.Write(capturePath(t, dir), fixture.Bytes, live.prepared.Identity(), result.run))
	}
	require.NoError(t, result.err)
	require.NotNil(t, result.run)
	require.NotNil(t, fixture.Expected)
	disposition, status := testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.VERDICT_STATUS_SATISFIED
	if fixture.Expected.Contract == "violated" {
		disposition, status = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, testpilotspb.VERDICT_STATUS_VIOLATED
	}
	require.Equal(t, disposition, result.run.GetDisposition(), "%v", result.run.GetDiagnostics())
	require.Equal(t, testpilotspb.CLEANUP_STATUS_SUCCEEDED, result.run.GetCleanup().GetStatus())
	require.Equal(t, status, result.verdict.GetStatus(), "%v", result.run.GetDiagnostics())
	require.NotNil(t, result.assessment)
	require.Nil(t, result.assessment.Failure)
	require.NotNil(t, fixture.Expected)
	require.Equal(t, fixture.Expected.Conformance, string(result.assessment.Conformance.Status), result.assessment.Conformance.Detail)
	require.Len(t, result.assessment.Properties, len(fixture.Expected.Properties))
	for _, expected := range fixture.Expected.Properties {
		var found bool
		for _, actual := range result.assessment.Properties {
			if actual.ID != expected.ID {
				continue
			}
			found = true
			require.Equal(t, expected.Status, string(actual.Status), "%s: %s", expected.ID, actual.Detail)
			if expected.Reason == "" {
				require.Empty(t, actual.Detail)
			} else {
				require.True(t, strings.HasSuffix(actual.Detail, ": "+expected.Reason), actual.Detail)
			}
		}
		require.True(t, found, "assessment omitted %s", expected.ID)
	}
	// Every fault the Case declares was realized, and each is on the Run's record.
	var declared, realized []testpilotspb.FaultKind
	for _, entrypoint := range fixture.Source.GetProgram().GetEntrypoints() {
		for _, node := range entrypoint.GetInstructions() {
			if fault := node.GetInstruction().GetInjectFault(); fault != nil {
				declared = append(declared, fault.GetKind())
			}
		}
	}
	for _, event := range result.run.GetEvents() {
		if event.GetKind() == testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED {
			realized = append(realized, event.GetFaultInjected().GetKind())
		}
	}
	require.Equal(t, declared, realized)
	assessed, err := live.prepared.WithAssessment(fixture.Assessment)
	require.NoError(t, err)
	verdict, evaluation, err := assessed.Evaluate(t.Context(), result.run, result.assessment)
	require.NoError(t, err)
	protorequire.ProtoEqual(t, result.verdict, verdict)
	require.Equal(t, result.assessment, evaluation.Assessment)
	protorequire.ProtoEqual(t, fixture.Source, live.prepared.Snapshot())
}

func runGeneratedCases(t *testing.T, env *testcore.TestEnv, fixture *testpilotcore.ModelCase, lives []testpilotLiveCase) []generatedRun {
	t.Helper()
	results := make([]generatedRun, len(lives))
	var pending sync.WaitGroup
	for i := range lives {
		pending.Go(func() { results[i] = runGeneratedCase(t, env, fixture, lives[i]) })
	}
	pending.Wait()
	return results
}

// TestTestpilotGeneratedCases runs every lowered Case as its own depth-2 subtest, the unit the
// functional job shards and the salt optimizer times, each against a dedicated cluster. A Case that
// binds a Nexus endpoint runs once per value of the Nexus implementation switch, a cluster each, and
// the two Verdicts must agree; any other Case runs once with no switch below it.
func TestTestpilotGeneratedCases(t *testing.T) {
	entries, err := testpilotcore.GeneratedCases(generatedCaseDirectory)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Standing != lower.Lowered {
			continue
		}
		name := testpilotcore.GeneratedCaseName(entry)
		t.Run(name, func(t *testing.T) {
			// First, so a Case outside the running shard skips as a whole: a Nexus Case constructs its
			// clusters below this level, where only the prefix places them.
			testcore.CheckTestShard(t)
			fixture, err := testpilotcore.LoadGeneratedCase(generatedCaseDirectory, entry)
			require.NoError(t, err)
			if !bindsNexusEndpoint(fixture.Source) {
				runGeneratedCaseOnCluster(t, entry, fixture, "", []testpilotcore.SwitchSetting{
					{Setting: activity.Enabled, Value: true},
					{Setting: activity.EnableStandaloneActivityOperatorCommands, Value: true},
					{Setting: dynamicconfig.EnableChasm, Value: true},
				})
				return
			}
			// Each value appends only the Verdict it produced, and nothing here counts them: the test
			// runner retries a failed value alone, by its anchored name, and that retry must be judged on
			// the one value it runs.
			var results []testpilotcore.SwitchVerdict
			for _, value := range testpilotcore.NexusImplementationSwitch() {
				t.Run(value.Name, func(t *testing.T) {
					// Standalone activity needs CHASM independently of the Nexus implementation switch.
					settings := append(slices.Clone(value.Settings), testpilotcore.SwitchSetting{Setting: dynamicconfig.EnableChasm, Value: true})
					verdict := runGeneratedCaseOnCluster(t, entry, fixture, value.Name, settings)
					results = append(results, testpilotcore.SwitchVerdict{Value: value.Name, Verdict: verdict})
				})
			}
			require.NoError(t, testpilotcore.CheckSwitchAgreement(testpilotcore.NexusImplementationSwitchName, results))
		})
	}
}

// runGeneratedCaseOnCluster constructs one dedicated cluster under settings, binds the Case to two
// isolated lives on it and runs both twice, requiring every Run to be assessed as the manifest
// expects. It returns the first life's first Verdict, the one a switch agreement compares. value is
// the switch value the cluster runs under, empty for a Case with no switch: under a value, a Profile
// that rejects the Case as unsupported fails naming the value, since the other value running and this
// one not is a finding; with no switch it skips, since no agreement depends on it.
func runGeneratedCaseOnCluster(t *testing.T, entry lower.GeneratedCase, fixture *testpilotcore.ModelCase, value string, settings []testpilotcore.SwitchSetting) *testpilotspb.Verdict {
	t.Helper()
	options := make([]testcore.TestOption, 0, len(settings))
	for _, setting := range settings {
		options = append(options, testcore.WithDynamicConfig(setting.Setting, setting.Value))
	}
	env := newTestpilotTestEnvironment(t, options...)
	// The Profile records the settings this cluster was constructed with, and only those; a setting
	// repeated in the list takes its last value, as at construction.
	configuration := testpilotcore.SwitchValue{Settings: settings}.Configuration()
	needsHold := false
	for _, script := range fixture.Source.GetProgram().GetEntrypoints() {
		for _, node := range script.GetInstructions() {
			if node.GetInstruction().GetInjectFault().GetKind() == testpilotspb.FAULT_KIND_DELIVERY_HOLD {
				needsHold = true
			}
		}
	}
	lives := make([]testpilotLiveCase, 2)
	for i := range lives {
		name := "scala-" + uuid.NewString()
		binding := defaultBinding(name, fixture.Source)
		binding.CreateEndpoint = bindsNexusEndpoint(fixture.Source)
		binding.DynamicConfig = maps.Clone(configuration)
		catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
		require.NoError(t, err)
		handlerQueue := ""
		if testpilotdriver.HandlerTaskQueueBindingID(fixture.Source.GetProgram()) != "" {
			handlerQueue = binding.TaskQueue + "-handler"
		}
		profile, err := testpilotdriver.DeriveProfile(fixture.Source, catalog, testpilotdriver.Environment{
			Identity: binding.Identity, Namespace: binding.Namespace, TaskQueue: binding.TaskQueue,
			HandlerTaskQueue: handlerQueue, NexusEndpoint: binding.NexusEndpoint, DeliveryControl: needsHold,
		})
		require.NoError(t, err)
		if _, err := testpilot.Prepare(fixture.Source, profile); err != nil {
			var rejection *testpilot.PreparationError
			if errors.As(err, &rejection) && rejection.Category == testpilot.PreparationUnsupported {
				if value != "" {
					t.Fatalf("%s=%s rejected the Case at preparation: %v", testpilotcore.NexusImplementationSwitchName, value, err)
				}
				t.Skipf("skipped: %v", err)
			}
			require.NoError(t, err)
		}
		if needsHold {
			controlled := bindControlledCase(t, env, fixture.Source, name)
			_, _, err := controlled.prepared.Run(t.Context(), controlled.uncontrolled)
			require.ErrorIs(t, err, testpilotdriver.ErrNoDeliveryControl)
			lives[i] = controlled.testpilotLiveCase
		} else {
			lives[i] = bindCase(t, env, fixture.Source, binding)
		}
	}
	require.NotEqual(t, lives[0].prepared.Identity().Bindings, lives[1].prepared.Identity().Bindings)
	var verdict *testpilotspb.Verdict
	ids := map[string]bool{}
	for round := range 2 {
		for i, result := range runGeneratedCases(t, env, fixture, lives) {
			requireGeneratedAssessment(t, fixture, lives[i], result)
			if round == 0 && i == 0 {
				verdict = result.verdict
				if os.Getenv("UMPIRE_EXPLORATION_DIR") != "" {
					model, err := umpiremodel.Load(filepath.Join(generatedCaseDirectory, "..", "ir", entry.Model))
					require.NoError(t, err)
					identity, err := recordedrun.CaseIdentity(fixture.Bytes)
					require.NoError(t, err)
					root, err := filepath.Abs("..")
					require.NoError(t, err)
					trace, err := explore.RenderTrace(&explore.Candidate{Model: model, Case: fixture.Source, Identity: identity, Digest: "generated:" + entry.Model + "/" + entry.Query.Name}, entry.Query.Name, result.run, result.assessment, root)
					require.NoError(t, err)
					artifact := testpilotcore.GeneratedCaseName(entry)
					if value != "" {
						artifact += "-" + value
					}
					writeExplorationArtifact(t, artifact+".html", trace)
				}
			}
			require.NotContains(t, ids, result.run.GetRunId())
			ids[result.run.GetRunId()] = true
			if len(fixture.Durable) != 0 {
				requireInconclusiveWithoutDurableEvidence(t, fixture, lives[i], result.run)
			}
		}
	}
	return verdict
}
