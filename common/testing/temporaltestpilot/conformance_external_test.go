package temporaltestpilot_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/temporaltestpilot"
	"go.temporal.io/server/common/testing/testpilot"
)

var (
	_ testpilot.Profile = (*temporaltestpilot.Driver)(nil)
	_ testpilot.Driver  = (*temporaltestpilot.Driver)(nil)

	_ func(temporaltestpilot.Options) (*temporaltestpilot.Driver, error) = temporaltestpilot.New
	_ temporaltestpilot.Endpoint                                         = temporaltestpilot.Endpoint{}
	_ temporaltestpilot.RoleBinding                                      = temporaltestpilot.RoleBinding{}
	_                                                                    = temporaltestpilot.Options{
		Profile:               testpilot.ProfileSpec{},
		ServerEndpoints:       map[string]temporaltestpilot.Endpoint{"workflow-service": {}},
		SystemCallbackBaseURL: "http://127.0.0.1",
		HTTPClient:            nil,
		SDKClient:             nil,
		Namespace:             "default",
		WorkerRoleID:          "worker",
		TaskQueues:            []temporaltestpilot.RoleBinding{{RoleID: "task-queue", Value: "queue"}},
		NexusEndpoints:        []temporaltestpilot.RoleBinding{{RoleID: "nexus-endpoint", Value: "endpoint"}},
		WorkerStopTimeout:     time.Second,
	}
)

func TestWorkflowServiceCatalogPublicAPI(t *testing.T) {
	descriptors := temporaltestpilot.WorkflowServiceDescriptorSet()
	require.NotEmpty(t, descriptors.GetFile())

	catalog, err := temporaltestpilot.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	require.NotEmpty(t, catalog.Identity())
}
