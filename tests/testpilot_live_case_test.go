//go:build test_dep && integration

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
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

// newTestpilotLiveCase performs the binding every live Case needs: register the namespace, create
// the Nexus endpoint when one is named, freeze the Profile, prepare the unchanged Case bytes, and
// open one Driver over the frozen Profile. Every resource it creates is released by a registered
// cleanup, all under the one cleanupTimeout the caller chose. After the Driver exists it mutates
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
	_, err := env.RegisterNamespace(namespace.Name(resources.Namespace), 1, enumspb.ARCHIVAL_STATE_DISABLED, "", "")
	require.NoError(t, err)
	if resources.NexusEndpoint != "" {
		created, err := env.OperatorClient().CreateNexusEndpoint(env.Context(), &operatorservice.CreateNexusEndpointRequest{
			Spec: &nexus.EndpointSpec{
				Name: resources.NexusEndpoint,
				Target: &nexus.EndpointTarget{Variant: &nexus.EndpointTarget_Worker_{
					Worker: &nexus.EndpointTarget_Worker{Namespace: resources.Namespace, TaskQueue: resources.TaskQueue},
				}},
			},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
			defer cancel()
			_, err := env.OperatorClient().DeleteNexusEndpoint(ctx, &operatorservice.DeleteNexusEndpointRequest{
				Id: created.GetEndpoint().GetId(), Version: created.GetEndpoint().GetVersion(),
			})
			require.NoError(t, err)
		})
	}

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
