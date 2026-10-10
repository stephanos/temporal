package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
)

// sdkActivityInfo is what the SDK's activity test environment calls the first activity a fresh
// environment runs. The environment chooses the namespace, queue, activity and run itself, so the
// fixture binds the Program to what it chooses rather than the other way round.
func sdkActivityInfo(t *testing.T) activity.Info {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestActivityEnvironment().SetExecuteActivitiesInWorkflow(false)
	var info activity.Info
	environment.RegisterActivityWithOptions(func(ctx context.Context) error {
		info = activity.GetInfo(ctx)
		return nil
	}, activity.RegisterOptions{Name: "activity-type"})
	_, err := environment.ExecuteActivity("activity-type")
	require.NoError(t, err)
	require.NotEmpty(t, info.ActivityRunID)
	return info
}

// sdkActivityEnvironment is a fresh SDK activity environment whose worker runs the Driver's
// interceptor and whose registered activity is the Driver's dynamic activity under the Program's
// activity type. The environment runs the activity as one a client started, or as one a workflow
// scheduled.
func sdkActivityEnvironment(host *Driver, definition programDefinition, queue string, standalone bool) *testsuite.TestActivityEnvironment {
	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestActivityEnvironment().SetExecuteActivitiesInWorkflow(!standalone)
	environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{host: host, queue: queue, registration: definition.registrations[0]}}})
	environment.RegisterActivityWithOptions(func(ctx context.Context) (*celpb.Value, error) {
		return host.dynamicActivity(ctx, nil)
	}, activity.RegisterOptions{Name: "activity-type"})
	return environment
}

func TestSDKActivityInterpretsItsScriptUnderTheCarriedRoute(t *testing.T) {
	info := sdkActivityInfo(t)
	prepared := preparedActivityFixture(t, standaloneActivity, func(profile *testpilot.ProfileSpec) {
		for index, binding := range profile.EnvironmentBindings {
			switch binding.ID {
			case "namespace":
				profile.EnvironmentBindings[index].Value = info.Namespace
			case "task-queue":
				profile.EnvironmentBindings[index].Value = info.TaskQueue
			default:
			}
		}
	})
	host, definition := runtimeTestDriver(t, prepared)
	binding := delivery.ActivityBinding{Namespace: info.Namespace, ActivityID: info.ActivityID, ActivityType: "activity-type", TaskQueue: info.TaskQueue}
	session, _, request := activityTestSession(t, host, definition, prepared, "run", binding, info.ActivityRunID, delivery.TriggerSucceeded)

	// A task of the registered type that carries no route, or one a workflow scheduled, which names
	// no activity run, is refused and consumes nothing.
	unrouted := sdkActivityEnvironment(host, definition, info.TaskQueue, true)
	scheduled := sdkActivityEnvironment(host, definition, info.TaskQueue, false)
	scheduled.SetHeader(request.GetHeader())
	for name, refusing := range map[string]*testsuite.TestActivityEnvironment{"no route": unrouted, "scheduled by a workflow": scheduled} {
		t.Run(name, func(t *testing.T) {
			_, err := refusing.ExecuteActivity("activity-type")
			var refused *temporal.ApplicationError
			require.ErrorAs(t, err, &refused)
			require.Equal(t, "umpire_worker", refused.Type())
			require.True(t, refused.NonRetryable())
			require.False(t, reservationForEntrypoint(t, session, "activity").consumed)
		})
	}

	environment := sdkActivityEnvironment(host, definition, info.TaskQueue, true)
	environment.SetHeader(request.GetHeader())
	encoded, err := environment.ExecuteActivity("activity-type")
	require.NoError(t, err)
	var result celpb.Value
	require.NoError(t, encoded.Get(&result))
	require.True(t, proto.Equal(textResult("done"), &result), &result)
	// The recorded outcome names the run, the attempt and the delivery the SDK handed the
	// interceptor: the delivery is the digest of the task token, never the token.
	token := sha256.Sum256(info.TaskToken)
	require.NotEmpty(t, info.TaskToken)
	require.Equal(t, int32(1), info.Attempt)
	settled, err := settledActivity(t, session)
	require.NoError(t, err)
	requireOutcome(t, answered(info.ActivityRunID, 1, hex.EncodeToString(token[:]), completed), settled)
}
