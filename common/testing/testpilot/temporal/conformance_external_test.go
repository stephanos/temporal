package temporal_test

import (
	"os"
	"path/filepath"
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

// TestDefaultCeilingsAdmitTheConformanceCorpus prepares every admitted Case of the generic conformance
// corpus under the Profile DeriveProfile gives it. The generic facade conformance test may not import
// this Driver, so it spells these ceilings itself; this is where the one default set is shown to admit
// that corpus.
func TestDefaultCeilingsAdmitTheConformanceCorpus(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	for _, class := range []string{"satisfied", "violated", "inconclusive", "cleanup-failure-after-proved-violation", "cross-run-isolation"} {
		t.Run(class, func(t *testing.T) {
			encoded, err := os.ReadFile(filepath.Join("..", "testdata", "case-runtime-conformance", class, "case.json"))
			require.NoError(t, err)
			source, err := testpilot.DecodeCaseProtoJSON(encoded)
			require.NoError(t, err)
			profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{Identity: "conformance"})
			require.NoError(t, err)
			_, err = testpilot.Prepare(source, profile)
			require.NoError(t, err)
		})
	}
}
