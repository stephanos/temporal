package testpilot

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/model/scalav2/goir"
)

func TestScalaCasesLowerForExistingConsumers(t *testing.T) {
	for _, item := range []struct{ model, family, owner, set, query string }{
		{"activity", "temporal.activity.standalone", "activityProtocol", "standaloneActivityTests", "completion"},
		{"activity", "temporal.activity.standalone", "activityProtocol", "standaloneActivityTests", "retry"},
		{"activity", "temporal.activity.standalone", "activityProtocol", "standaloneActivityTests", "pauseResume"},
		{"nexus-caller", "temporal.nexus.caller", "nexusProtocol", "nexusCallerTests", "syncCompletion"},
		{"nexus-caller", "temporal.nexus.caller", "nexusProtocol", "nexusCallerTests", "asyncCompletion"},
	} {
		t.Run(item.query, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "model", "scalav2", "ir", item.model+".json")
			key := goir.ClaimKey{Family: item.family, Owner: item.owner, Name: item.query}
			fixture, err := LoadScalaCase(path, key, item.set)
			require.NoError(t, err)
			again, err := LoadScalaCase(path, key, item.set)
			require.NoError(t, err)
			require.Equal(t, fixture.Bytes, again.Bytes)
			decoded, err := testpilot.DecodeCaseProtoJSON(fixture.Bytes)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, fixture.Source, decoded)
			catalog, err := temporal.NewWorkflowServiceCatalog()
			require.NoError(t, err)
			profile, err := temporal.DeriveProfile(fixture.Source, catalog, temporal.Environment{Identity: "scala", Namespace: "ns", TaskQueue: "q", HandlerTaskQueue: "h", NexusEndpoint: "e"})
			require.NoError(t, err)
			prepared, err := testpilot.Prepare(fixture.Source, profile)
			require.NoError(t, err)
			_, err = prepared.WithAssessment(fixture.Assessment)
			require.NoError(t, err)
		})
	}
}
