//go:build test_dep && integration

package tests

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
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

// requireCorrelatedNexusHistoryEvidence reads the supporting evidence back out of the Run. Each
// supporting event carries both the history event it projected and the CorrelatedEvidence the same
// projection lifted from it, and the two agree: the operation key every value carries is the
// scheduled event both the started and the completed event name.
func requireCorrelatedNexusHistoryEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64, endpoint string) {
	t.Helper()
	require.Len(t, sequences, 2)
	events := make([]*historypb.HistoryEvent, 0, len(sequences))
	keys := make([]string, 0, len(sequences))
	kinds := make([]string, 0, len(sequences))
	for _, sequence := range sequences {
		event := runEventAt(t, run, sequence)
		require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
		require.Equal(t, "history", event.GetCoordinates().GetInstructionId())
		var historyEvent historypb.HistoryEvent
		require.NoError(t, observationValue(t, event, "history-event").GetMessageValue().UnmarshalTo(&historyEvent))
		events = append(events, &historyEvent)
		var evidence testpilotpb.CorrelatedEvidence
		require.NoError(t, observationValue(t, event, "correlated-evidence").GetMessageValue().UnmarshalTo(&evidence))
		keys = append(keys, evidence.GetOperation())
		kinds = append(kinds, evidence.GetKind())
	}
	require.Equal(t, []enumspb.EventType{
		enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
	}, []enumspb.EventType{events[0].GetEventType(), events[1].GetEventType()})
	require.Equal(t, []string{
		"temporal.nexus.success.evidence.started", "temporal.nexus.success.evidence.completed",
	}, kinds)

	scheduledID := events[0].GetNexusOperationStartedEventAttributes().GetScheduledEventId()
	require.Positive(t, scheduledID)
	require.Equal(t, scheduledID, events[1].GetNexusOperationCompletedEventAttributes().GetScheduledEventId())
	require.Equal(t, []string{strconv.FormatInt(scheduledID, 10), strconv.FormatInt(scheduledID, 10)}, keys)
	requestID := events[0].GetNexusOperationStartedEventAttributes().GetRequestId()
	require.NotEmpty(t, requestID)
	require.Equal(t, requestID, events[1].GetNexusOperationCompletedEventAttributes().GetRequestId())
	requireScheduledNexusEndpoint(t, run, scheduledID, requestID, endpoint)
}

// requireScheduledNexusEndpoint finds the scheduled event the two supporting events name. It
// supports no clause of its own -- the model's operation is already scheduled when it starts -- so
// it is read from the Run rather than from the Verdict.
func requireScheduledNexusEndpoint(t testing.TB, run *testpilotpb.Run, scheduledID int64, requestID, endpoint string) {
	t.Helper()
	for _, event := range run.GetEvents() {
		for _, observation := range event.GetObservations() {
			if observation.GetObservationId() != "history-event" {
				continue
			}
			var historyEvent historypb.HistoryEvent
			require.NoError(t, observation.GetValue().GetMessageValue().UnmarshalTo(&historyEvent))
			if historyEvent.GetEventId() != scheduledID {
				continue
			}
			scheduled := historyEvent.GetNexusOperationScheduledEventAttributes()
			require.Equal(t, endpoint, scheduled.GetEndpoint())
			require.Equal(t, requestID, scheduled.GetRequestId())
			return
		}
	}
	require.FailNow(t, "scheduled Nexus event not recorded")
}

func requireRunHasOutcome(t testing.TB, run *testpilotpb.Run, instructionID string, status testpilotpb.InstructionOutcomeStatus) {
	t.Helper()
	for _, event := range run.GetEvents() {
		if event.GetCoordinates().GetInstructionId() == instructionID && event.GetOutcome().GetStatus() == status {
			return
		}
	}
	require.Fail(t, "Run does not contain expected instruction outcome", "instruction: %s, status: %s", instructionID, status)
}
