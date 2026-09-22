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
	require.Equal(t, "named", HandlerQueue(deployment))
}

func TestOpenWithoutCreateTouchesNoServerAndCloses(t *testing.T) {
	campaign, err := Open(t.Context(), unreachable(), "probe-queue-handler")
	require.NoError(t, err)
	require.NotNil(t, campaign.Catalog())
	require.NoError(t, campaign.Close(t.Context()))
}

// Prepare needs no deployment: it prepares a Case under the deployment's names with the catalog it
// builds itself, refuses a handler-queue mismatch and a Case Prepare rejects, and returns the
// prepared Case with the identity Bind would run it under, all against an address nothing answers.
func TestPrepareTouchesNoDeployment(t *testing.T) {
	source := nexusCallerCase(t)
	prepared, err := Prepare(unreachable(), "probe-queue-handler", "probe.identity", source)
	require.NoError(t, err)
	require.NotNil(t, prepared.Case)
	require.Equal(t, "probe.identity", prepared.Case.Identity().Profile)
	require.NotEmpty(t, prepared.Case.Identity().Catalog)
	require.NotEmpty(t, prepared.Case.Identity().Bindings)
	require.Equal(t, "probe.identity", prepared.Profile.Identity)
	again, err := Prepare(unreachable(), "probe-queue-handler", "probe.identity", source)
	require.NoError(t, err)
	require.Equal(t, prepared.Case.Identity(), again.Case.Identity(), "the identity is a function of the Case and the names")
	other := unreachable()
	other.Namespace = "elsewhere"
	elsewhere, err := Prepare(other, "probe-queue-handler", "probe.identity", source)
	require.NoError(t, err)
	require.NotEqual(t, prepared.Case.Identity().Bindings, elsewhere.Case.Identity().Bindings, "other names are another binding fingerprint")

	malformed := nexusCallerCase(t)
	malformed.CaseId = ""
	_, err = Prepare(unreachable(), "probe-queue-handler", "probe.identity", malformed)
	_, rejected := IsPreparationRejection(err)
	require.True(t, rejected, "a Case Prepare rejects is a preparation rejection: %v", err)
	_, err = Prepare(unreachable(), "", "probe.identity", source)
	require.ErrorContains(t, err, "handler queue")
	_, err = Prepare(unreachable(), "probe-queue-handler", "probe.identity", nil)
	require.Error(t, err)
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

	// A Case that binds a handler queue of its own under a campaign opened without one would poll
	// one queue while the endpoint routes to another; it is refused before preparation.
	mixed, err := Open(t.Context(), unreachable(), "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, mixed.Close(context.Background())) })
	mixedBound, err := mixed.Bind(t.Context(), "probe.identity", source)
	require.Error(t, err)
	require.Nil(t, mixedBound)
	require.ErrorContains(t, err, "handler queue")

	noEndpoint := unreachable()
	noEndpoint.NexusEndpoint = ""
	campaign, err := Open(t.Context(), noEndpoint, "probe-queue-handler")
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
