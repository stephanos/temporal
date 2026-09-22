package binding

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

const fixtureRoot = "../../../tests/testcore/testpilot/testdata"

func nexusCallerCase(t *testing.T) *testpilotspb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(fixtureRoot, "nexusCallerTests-asyncCompletion-case.json"))
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	return source
}

// unreachable is a deployment nothing listens on: the campaign binding dials lazily, so opening
// it without --create touches no server, and a candidate binding fails at the SDK client, after
// preparation and before any Driver.
func unreachable() Deployment {
	return Deployment{
		GRPCAddress: "127.0.0.1:1", HTTPAddress: "127.0.0.1:1",
		Namespace: "probe", TaskQueue: "probe-queue", NexusEndpoint: "probe-endpoint",
	}
}

func TestHandlerQueueForNamesTheHandlerQueueOnlyWhenTheProgramBindsOne(t *testing.T) {
	source := nexusCallerCase(t)
	deployment := unreachable()
	require.Equal(t, "probe-queue-handler", HandlerQueueFor(deployment, source.GetProgram()))
	deployment.HandlerTaskQueue = "named"
	require.Equal(t, "named", HandlerQueueFor(deployment, source.GetProgram()))
	require.Empty(t, HandlerQueueFor(deployment, &testpilotspb.Program{}))
}

func TestOpenWithoutCreateTouchesNoServerAndCloses(t *testing.T) {
	campaign, err := Open(t.Context(), unreachable(), "probe-queue-handler")
	require.NoError(t, err)
	require.NotNil(t, campaign.Catalog())
	require.Equal(t, "probe", campaign.Deployment().Namespace)
	require.NoError(t, campaign.Close(t.Context()))
}

// A Case whose Profile the deployment cannot derive rejects before preparation; a Case that
// Prepare rejects is the Case's own static rejection, decided before the SDK client dials; a Case
// that prepares but whose deployment is unreachable fails at the SDK client, after preparation and
// before any Driver opens.
func TestBindDecidesPreparationBeforeAnyDriverOpens(t *testing.T) {
	source := nexusCallerCase(t)
	malformed := nexusCallerCase(t)
	malformed.CaseId = ""
	campaignForMalformed, err := Open(t.Context(), unreachable(), "probe-queue-handler")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, campaignForMalformed.Close(context.Background())) })
	rejectedBound, err := campaignForMalformed.Bind(t.Context(), "probe.identity", malformed)
	require.Error(t, err)
	require.Nil(t, rejectedBound)
	rejection, rejected := IsPreparationRejection(err)
	require.True(t, rejected, "a Case Prepare rejects is a preparation rejection: %v", err)
	require.Equal(t, testpilot.PreparationMalformed, rejection.Category)
	require.NotContains(t, err.Error(), "open SDK client", "a rejected Case never dials")

	noEndpoint := unreachable()
	noEndpoint.NexusEndpoint = ""
	campaign, err := Open(t.Context(), noEndpoint, "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, campaign.Close(context.Background())) })
	bound, err := campaign.Bind(t.Context(), "probe.identity", source)
	require.Error(t, err)
	require.Nil(t, bound)
	require.ErrorContains(t, err, "derive Profile")
	_, rejected = IsPreparationRejection(err)
	require.False(t, rejected)

	reachableProfile, err := Open(t.Context(), unreachable(), "probe-queue-handler")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reachableProfile.Close(context.Background())) })
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	bound, err = reachableProfile.Bind(ctx, "probe.identity", source)
	require.Error(t, err)
	require.Nil(t, bound)
	require.ErrorContains(t, err, "open SDK client")
	_, rejected = IsPreparationRejection(err)
	require.False(t, rejected)
}

func TestIsPreparationRejectionFindsAWrappedRejection(t *testing.T) {
	rejection := &testpilot.PreparationError{Category: testpilot.PreparationUnsupported, Path: "program", Detail: "opcode"}
	found, ok := IsPreparationRejection(errors.Join(errors.New("released"), rejection))
	require.True(t, ok)
	require.Equal(t, rejection, found)
	_, ok = IsPreparationRejection(errors.New("plain"))
	require.False(t, ok)
	_, ok = IsPreparationRejection(nil)
	require.False(t, ok)
}

// Releases run in reverse order, every one runs whatever the earlier ones did, and each failure
// is reported on its own.
func TestReleaseAllRunsEveryReleaseInReverseOrder(t *testing.T) {
	var order []string
	err := ReleaseAll(t.Context(), []func(context.Context) error{
		func(context.Context) error { order = append(order, "connection"); return nil },
		func(context.Context) error {
			order = append(order, "namespace")
			return errors.New("delete namespace: in progress")
		},
		func(context.Context) error {
			order = append(order, "driver")
			return errors.New("stop worker: timed out")
		},
	})
	require.Equal(t, []string{"driver", "namespace", "connection"}, order)
	require.ErrorContains(t, err, "stop worker: timed out")
	require.ErrorContains(t, err, "delete namespace: in progress")
	require.NoError(t, ReleaseAll(t.Context(), nil))
}
