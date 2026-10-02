//go:build test_dep && integration

package tests

import (
	"errors"
	"os"
	"path/filepath"
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

var scalaCaseDirectory = filepath.Join("..", "model", "scalav2", "cases")

func generatedScalaFixture(t *testing.T, file string) *testpilotcore.ScalaCase {
	t.Helper()
	entries, err := testpilotcore.ScalaManifest(scalaCaseDirectory)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.File == file {
			fixture, err := testpilotcore.LoadGeneratedScalaCase(scalaCaseDirectory, entry)
			require.NoError(t, err)
			return fixture
		}
	}
	t.Fatalf("no generated Case %s", file)
	return nil
}

func scalaActivityEnvironment(t *testing.T) *testcore.TestEnv {
	t.Helper()
	return newTestpilotTestEnvironment(t,
		testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true),
		testcore.WithDynamicConfig(activity.Enabled, true),
		testcore.WithDynamicConfig(activity.EnableStandaloneActivityOperatorCommands, true))
}

type scalaRun struct {
	run        *testpilotspb.Run
	verdict    *testpilotspb.Verdict
	assessment *testpilot.Assessment
	err        error
}

func runScalaCase(t *testing.T, env *testcore.TestEnv, fixture *testpilotcore.ScalaCase, live testpilotLiveCase) scalaRun {
	t.Helper()
	assessed, err := live.prepared.WithAssessment(fixture.Assessment)
	if err != nil {
		return scalaRun{err: err}
	}
	run, verdict, assessment, err := assessed.Run(t.Context(), live.driver)
	return scalaRun{run, verdict, assessment, err}
}

func requireScalaAssessment(t *testing.T, fixture *testpilotcore.ScalaCase, live testpilotLiveCase, result scalaRun) {
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

func runScalaCases(t *testing.T, env *testcore.TestEnv, fixture *testpilotcore.ScalaCase, lives []testpilotLiveCase) []scalaRun {
	t.Helper()
	results := make([]scalaRun, len(lives))
	var pending sync.WaitGroup
	for i := range lives {
		pending.Go(func() { results[i] = runScalaCase(t, env, fixture, lives[i]) })
	}
	pending.Wait()
	return results
}

func TestTestpilotScalaGeneratedCases(t *testing.T) {
	entries, err := testpilotcore.ScalaManifest(scalaCaseDirectory)
	require.NoError(t, err)
	for _, value := range testpilotcore.NexusImplementationSwitch() {
		t.Run(value.Name, func(t *testing.T) {
			options := []testcore.TestOption{
				testcore.WithDynamicConfig(activity.Enabled, true),
				testcore.WithDynamicConfig(activity.EnableStandaloneActivityOperatorCommands, true),
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
					fixture, err := testpilotcore.LoadGeneratedScalaCase(scalaCaseDirectory, entry)
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
						for i, result := range runScalaCases(t, env, fixture, lives) {
							requireScalaAssessment(t, fixture, lives[i], result)
							if round == 0 && i == 0 && os.Getenv("UMPIRE_EXPLORATION_DIR") != "" {
								model, err := umpiremodel.Load(filepath.Join(scalaCaseDirectory, "..", "ir", entry.Model))
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
