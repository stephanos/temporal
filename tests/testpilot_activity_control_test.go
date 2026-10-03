//go:build test_dep && integration

package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/testing/testhooks"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/common/testing/testpilot/temporal/control"
	"go.temporal.io/server/tests/testcore"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
	"google.golang.org/grpc/credentials/insecure"
)

// controlledCase is a Case bound for the in-process server: under a Profile whose environment
// supplies a delivery control, with the Driver that holds it and one that does not.
type controlledCase struct {
	testpilotLiveCase
	uncontrolled *testpilotdriver.Driver
}

func bindControlledCase(t *testing.T, env *testcore.TestEnv, source *testpilotspb.Case, name string) controlledCase {
	t.Helper()
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := testpilotdriver.DeriveProfile(source, catalog, testpilotdriver.Environment{
		Identity: name, Namespace: name, TaskQueue: name, DeliveryControl: true,
	})
	require.NoError(t, err)
	live := newTestpilotLiveCase(t, env, source, profile, testpilotLiveResources{Namespace: name, TaskQueue: name}, testpilotCleanupTimeout)
	described, err := env.FrontendClient().DescribeNamespace(t.Context(), &workflowservice.DescribeNamespaceRequest{Namespace: name})
	require.NoError(t, err)
	scope := namespace.ID(described.GetNamespaceInfo().GetId())
	deliveries, err := control.NewDeliveries(func(hook testhooks.Hook) func() { return env.GetTestCluster().InjectHook(t, hook, scope) })
	require.NoError(t, err)
	t.Cleanup(deliveries.Close)
	driver, err := testpilotdriver.New(testpilotdriver.Options{
		Profile: live.profile,
		ServerEndpoints: map[string]testpilotdriver.Endpoint{
			"temporal.workflow-service": {Target: env.FrontendGRPCAddress(), Credentials: insecure.NewCredentials()},
		},
		SystemCallbackBaseURL: "http://" + env.HttpAPIAddress(),
		SDKClient:             live.client,
		WorkerRoleID:          "temporal.worker",
		WorkerStopTimeout:     testpilotCleanupTimeout,
		Deliveries: func(activityID string) (testpilotdriver.HeldDelivery, error) {
			held, err := deliveries.Hold(activityID)
			if err != nil {
				return nil, err
			}
			return held, nil
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), testpilotCleanupTimeout)
		defer cancel()
		require.NoError(t, driver.Close(ctx))
	})
	bound := controlledCase{testpilotLiveCase: live, uncontrolled: live.driver}
	bound.driver = driver
	return bound
}

func claimOf(t *testing.T, fixture *testpilotcore.ModelCase, assessment *testpilot.Assessment) testpilot.PropertyAssessment {
	t.Helper()
	require.NotEmpty(t, fixture.Property)
	for _, claim := range assessment.Properties {
		if claim.ID == fixture.Property {
			return claim
		}
	}
	require.FailNow(t, "the assessment reads no claim "+fixture.Property)
	return testpilot.PropertyAssessment{}
}

// requireInconclusiveWithoutDurableEvidence is the ambiguity control: the same recorded Run with its
// durable-commit evidence taken out no longer satisfies the Contract, and leaves the Query's Property
// inconclusive, never satisfied and never violated.
func requireInconclusiveWithoutDurableEvidence(t *testing.T, fixture *testpilotcore.ModelCase, live testpilotLiveCase, run *testpilotspb.Run) {
	t.Helper()
	require.NotEmpty(t, fixture.Durable)
	stripped, err := fixture.WithoutDurableEvidence(run)
	require.NoError(t, err)
	require.NotEqual(t, len(run.String()), len(stripped.String()), "the Run recorded no durable-commit evidence to take out")
	assessed, err := live.prepared.WithAssessment(fixture.Assessment)
	require.NoError(t, err)
	verdict, evaluation, err := assessed.Evaluate(t.Context(), stripped, nil)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
	require.NotNil(t, evaluation.Assessment)
	require.Nil(t, evaluation.Assessment.Failure)
	require.NotEqual(t, testpilot.ConformanceNonconformant, evaluation.Assessment.Conformance.Status)
	claim := claimOf(t, fixture, evaluation.Assessment)
	require.Equal(t, testpilot.PropertyInconclusive, claim.Status)
	require.NotEmpty(t, claim.Detail)
}
