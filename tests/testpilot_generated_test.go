//go:build test_dep && integration

package tests

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/chasm/lib/activity"
	"go.temporal.io/server/chasm/lib/nexusoperation"
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

// requiredSettingOptions is the typed setting of each dynamic-configuration key a generated Case may
// require, by its lower-case key, as the server reads the key. dynamicconfig has no public lookup
// from a key to its setting, so a Case that requires a key missing here fails the suite rather than
// running against a server that does not set it.
var requiredSettingOptions = map[string]func(value string) (testcore.TestOption, error){
	strings.ToLower(nexusoperation.Enabled.Key().String()): boolSettingOption(nexusoperation.Enabled),
}

func boolSettingOption(setting dynamicconfig.GenericSetting) func(string) (testcore.TestOption, error) {
	return func(value string) (testcore.TestOption, error) {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		return testcore.WithDynamicConfig(setting, parsed), nil
	}
}

// generatedRequiredSettings is the union of the settings the lowered Cases require, by lower-case
// key: one environment runs them all, so two Cases that require one key at two values fail the suite.
func generatedRequiredSettings(t *testing.T, entries []lower.GeneratedCase) map[string]string {
	t.Helper()
	settings := map[string]string{}
	for _, entry := range entries {
		if entry.Standing != lower.Lowered {
			continue
		}
		encoded, err := os.ReadFile(filepath.Join(generatedCaseDirectory, entry.File))
		require.NoError(t, err)
		source, err := testpilot.DecodeCaseProtoJSON(encoded)
		require.NoError(t, err)
		for _, setting := range source.GetProgram().GetRequiredSettings() {
			key := strings.ToLower(setting.GetKey())
			if previous, ok := settings[key]; ok {
				require.Equal(t, previous, setting.GetValue(), "%s requires %s at another value than an earlier Case", entry.File, setting.GetKey())
			}
			settings[key] = setting.GetValue()
		}
	}
	return settings
}

func TestTestpilotGeneratedCases(t *testing.T) {
	entries, err := testpilotcore.GeneratedCases(generatedCaseDirectory)
	require.NoError(t, err)
	required := generatedRequiredSettings(t, entries)
	for _, value := range testpilotcore.NexusImplementationSwitch() {
		t.Run(value.Name, func(t *testing.T) {
			options := []testcore.TestOption{
				testcore.WithDynamicConfig(activity.Enabled, true),
				testcore.WithDynamicConfig(activity.EnableStandaloneActivityOperatorCommands, true),
			}
			// The server runs under every setting a Case requires, and the derived Profile records each,
			// so preparation checks a Case against what the server actually runs with.
			for _, key := range slices.Sorted(maps.Keys(required)) {
				option, ok := requiredSettingOptions[key]
				require.True(t, ok, "a generated Case requires %s, which the suite cannot set", key)
				applied, err := option(required[key])
				require.NoError(t, err, "a generated Case requires %s=%s", key, required[key])
				options = append(options, applied)
			}
			for _, setting := range value.Settings {
				options = append(options, testcore.WithDynamicConfig(setting.Setting, setting.Value))
			}
			// Standalone activity needs CHASM independently of the Nexus implementation switch.
			options = append(options, testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true))
			env := newTestpilotTestEnvironment(t, options...)
			for _, entry := range entries {
				if entry.Standing != lower.Lowered {
					continue
				}
				t.Run(entry.File, func(t *testing.T) {
					fixture, err := testpilotcore.LoadGeneratedCase(generatedCaseDirectory, entry)
					require.NoError(t, err)
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
						binding.DynamicConfig = value.Configuration()
						binding.DynamicConfig[dynamicconfig.EnableChasm.Key().String()] = "true"
						maps.Copy(binding.DynamicConfig, required)
						catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
						require.NoError(t, err)
						handlerQueue := ""
						if testpilotdriver.HandlerTaskQueueBindingID(fixture.Source.GetProgram()) != "" {
							handlerQueue = binding.TaskQueue + "-handler"
						}
						profile, err := testpilotdriver.DeriveProfile(fixture.Source, catalog, testpilotdriver.Environment{
							Identity: binding.Identity, Namespace: binding.Namespace, TaskQueue: binding.TaskQueue,
							HandlerTaskQueue: handlerQueue, NexusEndpoint: binding.NexusEndpoint, DeliveryControl: needsHold,
							DynamicConfig: binding.DynamicConfig,
						})
						require.NoError(t, err)
						if _, err := testpilot.Prepare(fixture.Source, profile); err != nil {
							var rejection *testpilot.PreparationError
							if errors.As(err, &rejection) && rejection.Category == testpilot.PreparationUnsupported {
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
					ids := map[string]bool{}
					for round := range 2 {
						for i, result := range runGeneratedCases(t, env, fixture, lives) {
							requireGeneratedAssessment(t, fixture, lives[i], result)
							if round == 0 && i == 0 && os.Getenv("UMPIRE_EXPLORATION_DIR") != "" {
								model, err := umpiremodel.Load(filepath.Join(generatedCaseDirectory, "..", "ir", entry.Model))
								require.NoError(t, err)
								identity, err := recordedrun.CaseIdentity(fixture.Bytes)
								require.NoError(t, err)
								root, err := filepath.Abs("..")
								require.NoError(t, err)
								trace, err := explore.RenderTrace(&explore.Candidate{Model: model, Case: fixture.Source, Identity: identity, Digest: "generated:" + entry.Model + "/" + entry.Query.Name}, entry.Query.Name, result.run, result.assessment, root)
								require.NoError(t, err)
								writeExplorationArtifact(t, value.Name+"-"+entry.File+".html", trace)
							}
							require.NotContains(t, ids, result.run.GetRunId())
							ids[result.run.GetRunId()] = true
							if len(fixture.Durable) != 0 {
								requireInconclusiveWithoutDurableEvidence(t, fixture, lives[i], result.run)
							}
						}
					}
				})
			}
		})
	}
}
