//go:build test_dep && integration

package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testhooks"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/common/testing/testpilot/temporal/control"
	"go.temporal.io/server/tests/testcore"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
	"go.temporal.io/server/tools/umpire/replay"
	"google.golang.org/grpc/credentials/insecure"
)

// scalaControlledCases is the lowered Cases whose realization holds a delivery, by where the IR
// declares each. What a Run of one must show is its Contract and its Query's Property; nothing
// here says it again.
var scalaControlledCases = []struct{ model, family, owner, set, query string }{
	{"activity-race", "temporal.activity.standalone.system", "heldAdmission", "standaloneActivityRace", "heldAdmission.staleDelivery"},
}

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
		Deliveries:            deliveries,
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

// requireScalaClaim requires of a Run what its Case declares and no more: it completed and cleaned
// up, its Contract is satisfied, the model explains it, the Query's Property is satisfied on it and
// no claim is violated; it realized and recorded every fault the Case declares; and the recorded Run
// replays to the same Verdict and assessment.
func requireScalaClaim(t *testing.T, fixture *testpilotcore.ScalaCase, live testpilotLiveCase, result scalaRun) {
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
	require.Equal(t, testpilot.ConformanceConformant, result.assessment.Conformance.Status, result.assessment.Conformance.Detail)
	claim := claimOf(t, fixture, result.assessment)
	require.Equal(t, testpilot.PropertySatisfied, claim.Status, claim.Detail)
	for _, claim := range result.assessment.Properties {
		require.NotEqual(t, testpilot.PropertyViolated, claim.Status, claim.ID)
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

func claimOf(t *testing.T, fixture *testpilotcore.ScalaCase, assessment *testpilot.Assessment) testpilot.PropertyAssessment {
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
func requireInconclusiveWithoutDurableEvidence(t *testing.T, fixture *testpilotcore.ScalaCase, live testpilotLiveCase, run *testpilotspb.Run) {
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

func TestTestpilotScalaActivityControlled(t *testing.T) {
	env := scalaActivityEnvironment(t)
	for _, item := range scalaControlledCases {
		t.Run(item.query, func(t *testing.T) {
			fixture := scalaFixture(t, item.model, item.family, item.owner, item.set, item.query)
			live := bindControlledCase(t, env, fixture.Source, fmt.Sprintf("scala-controlled-%s", uuid.NewString()))

			// A Driver whose environment holds no delivery refuses the Case before it opens a Run.
			_, _, err := live.prepared.Run(t.Context(), live.uncontrolled)
			require.ErrorIs(t, err, testpilotdriver.ErrNoDeliveryControl)

			ids := map[string]bool{}
			for range 2 {
				result := runScalaCase(t, env, fixture, live.testpilotLiveCase)
				requireScalaClaim(t, fixture, live.testpilotLiveCase, result)
				require.NotContains(t, ids, result.run.GetRunId())
				ids[result.run.GetRunId()] = true
				requireInconclusiveWithoutDurableEvidence(t, fixture, live.testpilotLiveCase, result.run)
			}
		})
	}
}
