//go:build test_dep && integration

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/chasm/lib/activity"
	activitymodel "go.temporal.io/server/chasm/lib/activity/model"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/payloads"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/model/scalav2/goir"
	"go.temporal.io/server/tests/testcore"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
	"go.temporal.io/server/tools/umpire/replay"
)

func scalaFixture(t *testing.T, model, family, owner, set, query string) *testpilotcore.ScalaCase {
	t.Helper()
	fixture, err := testpilotcore.LoadScalaCase(filepath.Join("..", "model", "scalav2", "ir", model+".json"),
		goir.ClaimKey{Family: family, Owner: owner, Name: query}, set)
	require.NoError(t, err)
	return fixture
}

func scalaActivityFixture(t *testing.T, query string) *testpilotcore.ScalaCase {
	t.Helper()
	return scalaFixture(t, "activity", "temporal.activity.standalone", "activityProtocol", "standaloneActivityTests", query)
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

func requireScalaAssessment(t *testing.T, fixture *testpilotcore.ScalaCase, live testpilotLiveCase, result scalaRun, property string, status testpilot.PropertyStatus) {
	t.Helper()
	if dir := os.Getenv(umpireRepeatRunDirVariable); dir != "" && result.run != nil {
		require.NoError(t, replay.WriteRecordedRun(capturePath(t, dir), fixture.Bytes, live.prepared.Identity(), result.run))
	}
	require.NoError(t, result.err)
	require.NotNil(t, result.run)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, result.run.GetDisposition(), "%v", result.run.GetDiagnostics())
	require.Equal(t, testpilotspb.CLEANUP_STATUS_SUCCEEDED, result.run.GetCleanup().GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, result.verdict.GetStatus(), "%v", result.run.GetDiagnostics())
	require.NotNil(t, result.assessment)
	require.Nil(t, result.assessment.Failure)
	require.Equal(t, testpilot.ConformanceConformant, result.assessment.Conformance.Status)
	require.Len(t, result.assessment.Properties, 1)
	require.Equal(t, property, result.assessment.Properties[0].ID)
	require.Equal(t, status, result.assessment.Properties[0].Status)
	if status == testpilot.PropertyInconclusive {
		require.NotEmpty(t, result.assessment.Properties[0].Detail)
	}
	assessed, err := live.prepared.WithAssessment(fixture.Assessment)
	require.NoError(t, err)
	verdict, evaluation, err := assessed.Evaluate(t.Context(), result.run, result.assessment)
	require.NoError(t, err)
	protorequire.ProtoEqual(t, result.verdict, verdict)
	require.Equal(t, result.assessment, evaluation.Assessment)
	protorequire.ProtoEqual(t, fixture.Source, live.prepared.Snapshot())
}

func requireScalaActivity(t *testing.T, env *testcore.TestEnv, live testpilotLiveCase, result scalaRun, responses []testpilotspb.ActivityAttemptResponse) string {
	t.Helper()
	var attempts []*testpilotspb.ActivityAttempt
	for _, event := range result.run.GetEvents() {
		if attempt := event.GetOutcome().GetActivityAttempt(); attempt != nil {
			attempts = append(attempts, attempt)
		}
	}
	require.Len(t, attempts, len(responses))
	activityRun := attempts[0].GetActivityRunId()
	require.NotEmpty(t, activityRun)
	_, err := uuid.Parse(activityRun)
	require.NoError(t, err)
	deliveries := map[string]bool{}
	for i, attempt := range attempts {
		require.NotEmpty(t, attempt.GetDeliveryId())
		require.NotContains(t, deliveries, attempt.GetDeliveryId())
		deliveries[attempt.GetDeliveryId()] = true
		protorequire.ProtoEqual(t, &testpilotspb.ActivityAttempt{ActivityRunId: activityRun, SdkAttempt: int32(i + 1), DeliveryId: attempt.GetDeliveryId(), Response: responses[i]}, attempt)
	}
	namespace := scalaNamespace(live)
	described, err := env.FrontendClient().DescribeActivityExecution(t.Context(), &workflowservice.DescribeActivityExecutionRequest{
		Namespace: namespace, ActivityId: result.run.GetRunId(), RunId: activityRun, IncludeOutcome: true,
	})
	require.NoError(t, err)
	require.Equal(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, described.GetInfo().GetStatus())
	resultValue := &testpilotspb.Value{}
	require.NoError(t, payloads.Decode(described.GetOutcome().GetResult(), resultValue))
	protorequire.ProtoEqual(t, &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "done"}}, resultValue)
	return activityRun
}

func TestTestpilotScalaActivity(t *testing.T) {
	env := scalaActivityEnvironment(t)
	for _, query := range []string{"completion", "retry", "pauseResume"} {
		t.Run(query, func(t *testing.T) {
			fixture := scalaActivityFixture(t, query)
			lives := make([]testpilotLiveCase, 2)
			for i := range lives {
				name := fmt.Sprintf("scala-%s-%s", query, uuid.NewString())
				lives[i] = bindCase(t, env, fixture.Source, CaseBinding{Identity: name, Namespace: name, TaskQueue: name})
			}
			require.NotEqual(t, lives[0].prepared.Identity().Bindings, lives[1].prepared.Identity().Bindings)
			ids := map[string]bool{}
			learned := map[string]bool{}
			for range 2 {
				results := runScalaCases(t, env, fixture, lives)
				for i, result := range results {
					property, status := "completes", testpilot.PropertySatisfied
					responses := []testpilotspb.ActivityAttemptResponse{testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED}
					if query == "retry" {
						property, status = "retryCompletes", testpilot.PropertyInconclusive
						responses = []testpilotspb.ActivityAttemptResponse{testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED}
					}
					requireScalaAssessment(t, fixture, lives[i], result, property, status)
					activityRun := requireScalaActivity(t, env, lives[i], result, responses)
					require.NotContains(t, ids, result.run.GetRunId())
					require.NotContains(t, learned, activityRun)
					ids[result.run.GetRunId()], learned[activityRun] = true, true
					for other := range lives {
						if other == i {
							continue
						}
						_, err := env.FrontendClient().DescribeActivityExecution(t.Context(), &workflowservice.DescribeActivityExecutionRequest{Namespace: scalaNamespace(lives[other]), ActivityId: result.run.GetRunId(), RunId: activityRun})
						var missing *serviceerror.NotFound
						require.ErrorAs(t, err, &missing)
					}
					if query == "pauseResume" {
						requireScalaPauseBeforeAttempt(t, result.run)
					}
				}
			}
		})
	}
}

func scalaNamespace(live testpilotLiveCase) string {
	for _, binding := range live.profile.EnvironmentBindings {
		if binding.ID == "temporal.worker.namespace" {
			return binding.Value
		}
	}
	return ""
}

func requireScalaPauseBeforeAttempt(t *testing.T, run *testpilotspb.Run) {
	t.Helper()
	var paused, unpaused, delivered int64
	for _, event := range run.GetEvents() {
		if event.GetCoordinates().GetInstructionId() == "await-paused" {
			for _, observation := range event.GetObservations() {
				value := observation.GetValue().GetMessageValue()
				if value != nil && value.MessageIs(&testpilotspb.CorrelatedEvidence{}) {
					evidence := &testpilotspb.CorrelatedEvidence{}
					require.NoError(t, value.UnmarshalTo(evidence))
					require.Equal(t, "evidence.statusPaused", evidence.GetKind())
					require.Equal(t, run.GetRunId(), evidence.GetOperation())
					paused = event.GetSequence()
				}
			}
		}
		if event.GetCoordinates().GetInstructionId() == "unpause-activity" && event.GetOutcome().GetStatus() == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
			unpaused = event.GetSequence()
		}
		if event.GetOutcome().GetActivityAttempt() != nil {
			delivered = event.GetSequence()
		}
	}
	require.Positive(t, paused)
	require.Greater(t, unpaused, paused)
	require.Greater(t, delivered, unpaused)
}

func TestTestpilotScalaActivityWorkflowParity(t *testing.T) {
	env := scalaActivityEnvironment(t)
	fixture := scalaActivityFixture(t, "completion")
	name := "scala-parity-" + uuid.NewString()
	live := bindCase(t, env, fixture.Source, CaseBinding{Identity: name, Namespace: name, TaskQueue: name})
	result := runScalaCase(t, env, fixture, live)
	requireScalaAssessment(t, fixture, live, result, "completes", testpilot.PropertySatisfied)
	runID := requireScalaActivity(t, env, live, result, []testpilotspb.ActivityAttemptResponse{testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED})
	described, err := env.FrontendClient().DescribeActivityExecution(t.Context(), &workflowservice.DescribeActivityExecutionRequest{Namespace: name, ActivityId: result.run.GetRunId(), RunId: runID, IncludeOutcome: true})
	require.NoError(t, err)
	workflow := newWFADriver(t, env, activityConfig{MaxAttempts: 1}).driveTrace(t, []activitymodel.Event{activitymodel.Poll, activitymodel.Complete})
	require.Equal(t, activityTerminalOutcome{status: enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED}, workflow.terminalOutcome(t))
	workflowOutcome := workflow.terminalOutcome(t)
	standaloneOutcome := activityTerminalOutcome{status: described.GetInfo().GetStatus(), retryState: described.GetOutcome().GetRetryState()}
	require.Equal(t, workflowOutcome, standaloneOutcome)
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
