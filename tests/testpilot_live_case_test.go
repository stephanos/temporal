//go:build test_dep && integration

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"go.temporal.io/server/tests/testcore"
	"google.golang.org/grpc/credentials/insecure"
)

// testpilotLiveResources names the physical resources one Case's Program binds symbolically. A
// Case that declares no Nexus endpoint leaves NexusEndpoint empty and none is created.
type testpilotLiveResources struct {
	Namespace     string
	TaskQueue     string
	NexusEndpoint string
}

// testpilotLiveCase is one Case bound to those resources: the Profile snapshot taken before the
// Driver was built, a prepared Case over unchanged bytes, an SDK client, and the shared Driver.
type testpilotLiveCase struct {
	profile  testpilot.ProfileSpec
	prepared *testpilot.PreparedCase
	client   client.Client
	driver   *testpilotdriver.Driver
}

// newTestpilotLiveCase performs the binding every live Case needs: provision the namespace and the
// Nexus endpoint when one is named, freeze the Profile, prepare the unchanged Case bytes, and open
// one Driver over the frozen Profile. Provisioning is the shared package, over the public workflow
// and operator services only, so a live test and the umpire-run CLI create the same resources the
// same way. Every resource it creates is released by a registered cleanup, all under the one
// cleanupTimeout the caller chose. After the Driver exists it mutates
// the frozen bindings, so a Driver that read its environment lazily rather than from its own
// snapshot fails in each caller's binding assertion.
func newTestpilotLiveCase(
	t *testing.T,
	env *testcore.TestEnv,
	caseSource *testpilotpb.Case,
	profile testpilot.ProfileSpec,
	resources testpilotLiveResources,
	cleanupTimeout time.Duration,
) testpilotLiveCase {
	t.Helper()
	release, err := provision.Create(env.Context(), provision.Clients{
		Workflow: env.FrontendClient(), Operator: env.OperatorClient(),
	}, provision.Resources{
		Namespace:     resources.Namespace,
		TaskQueue:     resources.TaskQueue,
		NexusEndpoint: resources.NexusEndpoint,
		// The functional cluster is discarded wholesale after the suite, and deleting a namespace
		// is a server-side workflow that takes tens of seconds; waiting for it here buys nothing.
		RetainNamespace: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		require.NoError(t, release(ctx))
	})

	frozen := profile.Snapshot()
	expectedProfile := frozen.Snapshot()
	prepared, err := testpilot.Prepare(caseSource, frozen)
	require.NoError(t, err)
	caseClient, err := client.Dial(client.Options{HostPort: env.FrontendGRPCAddress(), Namespace: resources.Namespace})
	require.NoError(t, err)
	t.Cleanup(caseClient.Close)
	driver, err := testpilotdriver.New(testpilotdriver.Options{
		Profile: frozen,
		ServerEndpoints: map[string]testpilotdriver.Endpoint{
			"temporal.workflow-service": {Target: env.FrontendGRPCAddress(), Credentials: insecure.NewCredentials()},
		},
		SystemCallbackBaseURL: "http://" + env.HttpAPIAddress(),
		SDKClient:             caseClient,
		WorkerRoleID:          "temporal.worker",
		WorkerStopTimeout:     cleanupTimeout,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		require.NoError(t, driver.Close(ctx))
	})
	for index := range frozen.EnvironmentBindings {
		frozen.EnvironmentBindings[index].Value = "mutated-after-freeze"
	}
	return testpilotLiveCase{profile: expectedProfile, prepared: prepared, client: caseClient, driver: driver}
}

// runEventAt resolves one recorded Run Event by its one-based sequence, which is the only place
// that assumption lives.
func runEventAt(t testing.TB, run *testpilotpb.Run, sequence int64) *testpilotpb.RunEvent {
	t.Helper()
	require.Positive(t, sequence)
	require.LessOrEqual(t, sequence, int64(len(run.GetEvents())))
	return run.GetEvents()[sequence-1]
}
