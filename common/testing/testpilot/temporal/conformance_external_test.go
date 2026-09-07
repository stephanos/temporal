package temporal_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

var (
	_ testpilot.Profile = (*temporal.Driver)(nil)
	_ testpilot.Driver  = (*temporal.Driver)(nil)

	_ func(temporal.Options) (*temporal.Driver, error) = temporal.New
	_ temporal.Endpoint                                = temporal.Endpoint{}
	_                                                  = temporal.Options{
		Profile:               testpilot.ProfileSpec{},
		ServerEndpoints:       map[string]temporal.Endpoint{"workflow-service": {}},
		SystemCallbackBaseURL: "http://127.0.0.1",
		HTTPClient:            nil,
		SDKClient:             nil,
		WorkerRoleID:          "worker",
		WorkerStopTimeout:     time.Second,
	}
)

func TestWorkflowServiceCatalogPublicAPI(t *testing.T) {
	descriptors := temporal.WorkflowServiceDescriptorSet()
	require.NotEmpty(t, descriptors.GetFile())

	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	require.NotEmpty(t, catalog.Identity())
}
