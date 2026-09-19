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
	Namespace      string
	TaskQueue      string
	NexusEndpoint  string
	NexusTaskQueue string
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
		Namespace:      resources.Namespace,
		TaskQueue:      resources.TaskQueue,
		NexusEndpoint:  resources.NexusEndpoint,
		NexusTaskQueue: resources.NexusTaskQueue,
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

// requireCorrelatedNexusHistoryEvidence reads the supporting evidence back out of the Run's history
// read. Each supporting event carries both the history event it projected and the
// CorrelatedEvidence the same projection lifted from it, and the two agree: the operation key every
// value carries is the scheduled event every event of the operation names, and the events are the
// ones the Query's claim supports, in the order the operation recorded them. It returns the
// scheduled event's id, which the reads the Run lifted before the history read are keyed by.
func requireCorrelatedNexusHistoryEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64, endpoint string, supporting []enumspb.EventType) int64 {
	t.Helper()
	require.Len(t, sequences, len(supporting))
	require.NotEmpty(t, sequences)
	var scheduledID int64
	var requestID string
	for index, sequence := range sequences {
		event := runEventAt(t, run, sequence)
		require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
		require.Equal(t, "history", event.GetCoordinates().GetInstructionId())
		var historyEvent historypb.HistoryEvent
		require.NoError(t, observationValue(t, event, "history-event").GetMessageValue().UnmarshalTo(&historyEvent))
		var evidence testpilotpb.CorrelatedEvidence
		require.NoError(t, observationValue(t, event, "correlated-evidence").GetMessageValue().UnmarshalTo(&evidence))
		require.Equal(t, supporting[index], historyEvent.GetEventType())
		// The lift names each evidence kind by the Case-local name the Case's provenance maps to
		// its Definition ID, which is the event kind's own last segment.
		require.Equal(t, nexusEvidenceKind(historyEvent.GetEventType()), evidence.GetKind())
		key, request := nexusOperationCoordinates(t, &historyEvent)
		require.Positive(t, key)
		if index == 0 {
			scheduledID, requestID = key, request
		}
		require.Equal(t, scheduledID, key)
		require.Equal(t, requestID, request)
		require.Equal(t, strconv.FormatInt(scheduledID, 10), evidence.GetOperation())
	}
	require.NotEmpty(t, requestID)
	requireScheduledNexusEndpoint(t, run, scheduledID, requestID, endpoint)
	return scheduledID
}

// nexusEvidenceKind is the Case-local name of the evidence kind one history event kind lifts into.
func nexusEvidenceKind(eventType enumspb.EventType) string {
	switch eventType {
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED:
		return "scheduled"
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED:
		return "started"
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED:
		return "completed"
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED:
		return "failed"
	case enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT:
		return "timedOut"
	default:
		return ""
	}
}

// nexusOperationCoordinates reads the scheduled event id and the request id one Nexus history event
// carries for its operation; the scheduled event names itself.
func nexusOperationCoordinates(t testing.TB, event *historypb.HistoryEvent) (int64, string) {
	t.Helper()
	switch attributes := event.GetAttributes().(type) {
	case *historypb.HistoryEvent_NexusOperationScheduledEventAttributes:
		return event.GetEventId(), attributes.NexusOperationScheduledEventAttributes.GetRequestId()
	case *historypb.HistoryEvent_NexusOperationStartedEventAttributes:
		return attributes.NexusOperationStartedEventAttributes.GetScheduledEventId(), attributes.NexusOperationStartedEventAttributes.GetRequestId()
	case *historypb.HistoryEvent_NexusOperationCompletedEventAttributes:
		return attributes.NexusOperationCompletedEventAttributes.GetScheduledEventId(), attributes.NexusOperationCompletedEventAttributes.GetRequestId()
	case *historypb.HistoryEvent_NexusOperationFailedEventAttributes:
		return attributes.NexusOperationFailedEventAttributes.GetScheduledEventId(), attributes.NexusOperationFailedEventAttributes.GetRequestId()
	case *historypb.HistoryEvent_NexusOperationTimedOutEventAttributes:
		return attributes.NexusOperationTimedOutEventAttributes.GetScheduledEventId(), attributes.NexusOperationTimedOutEventAttributes.GetRequestId()
	default:
		require.FailNow(t, "history event is not a Nexus operation event", "%s", event.GetEventType())
		return 0, ""
	}
}

// requireNexusHistoryEvent finds one recorded history event of the type among the Run's history
// observations, whether or not a clause supported it.
func requireNexusHistoryEvent(t testing.TB, run *testpilotpb.Run, eventType enumspb.EventType) {
	t.Helper()
	for _, event := range run.GetEvents() {
		for _, observation := range event.GetObservations() {
			if observation.GetObservationId() != "history-event" {
				continue
			}
			var historyEvent historypb.HistoryEvent
			require.NoError(t, observation.GetValue().GetMessageValue().UnmarshalTo(&historyEvent))
			if historyEvent.GetEventType() == eventType {
				return
			}
		}
	}
	require.FailNow(t, "history event not recorded", "%s", eventType)
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
