//go:build test_dep && integration

package tests

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"go.temporal.io/server/tests/testcore"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
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

// runRecording runs the bound Case once and records the closed Run, with the identity of caseBytes
// (the Case it was bound from) and the Profile identity it was prepared under, at path, in the
// recorded-Run shape a replay reads; the control test alone uses it, since a Run recorded under
// switch configuration is stale to a replay by design.
func (live testpilotLiveCase) runRecording(t *testing.T, ctx context.Context, caseBytes []byte, path string) (*testpilotpb.Run, *testpilotpb.Verdict) {
	t.Helper()
	run, verdict, err := live.prepared.Run(ctx, live.driver)
	require.NoError(t, err)
	recorded, err := machineFreeRun(run)
	require.NoError(t, err)
	require.NoError(t, recordedrun.Write(path, caseBytes, live.prepared.Identity(), recorded))
	return run, verdict
}

// recordedHost stands in for the recording machine's hostname, which the SDK worker writes into
// its default identity (pid@host@) and its sticky task queue name (host:uuid), so a pinned record
// re-recorded on any machine names none.
const recordedHost = "vm"

// machineFreeRun is a copy of run with the SDK worker's hostname replaced by recordedHost in every
// string, the ones inside an Any's message included. No Query reads a worker identity or a sticky
// queue name, so the recorded Run replays to the Verdict the live one reached.
func machineFreeRun(run *testpilotpb.Run) (*testpilotpb.Run, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	scrubbed := proto.CloneOf(run)
	if host == recordedHost {
		return scrubbed, nil
	}
	replace := func(value string) string {
		value = strings.ReplaceAll(value, "@"+host+"@", "@"+recordedHost+"@")
		if rest, ok := strings.CutPrefix(value, host+":"); ok {
			value = recordedHost + ":" + rest
		}
		return value
	}
	if _, err := scrubHost(scrubbed.ProtoReflect(), replace); err != nil {
		return nil, err
	}
	return scrubbed, nil
}

// scrubHost rewrites message's strings in place with replace and reports whether any changed.
func scrubHost(message protoreflect.Message, replace func(string) string) (bool, error) {
	if packed, ok := message.Interface().(*anypb.Any); ok {
		return scrubAny(packed, replace)
	}
	changed := false
	var err error
	// scrub rewrites one value of field, a string it returns changed or a message in place.
	scrub := func(field protoreflect.FieldDescriptor, value protoreflect.Value) (protoreflect.Value, bool) {
		if field.Kind() == protoreflect.StringKind {
			next := replace(value.String())
			return protoreflect.ValueOfString(next), next != value.String()
		}
		if field.Message() != nil && err == nil {
			var nested bool
			nested, err = scrubHost(value.Message(), replace)
			changed = changed || nested
		}
		return value, false
	}
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch {
		case field.IsList():
			list := value.List()
			for index := range list.Len() {
				if next, ok := scrub(field, list.Get(index)); ok {
					list.Set(index, next)
					changed = true
				}
			}
		case field.IsMap():
			entries := value.Map()
			entries.Range(func(key protoreflect.MapKey, entry protoreflect.Value) bool {
				if next, ok := scrub(field.MapValue(), entry); ok {
					entries.Set(key, next)
					changed = true
				}
				return true
			})
		default:
			if next, ok := scrub(field, value); ok {
				message.Set(field, next)
				changed = true
			}
		}
		return err == nil
	})
	return changed, err
}

// scrubAny rewrites the message an Any carries and re-packs it only when a string changed, so an
// Any with nothing to replace keeps its bytes.
func scrubAny(packed *anypb.Any, replace func(string) string) (bool, error) {
	inner, err := packed.UnmarshalNew()
	if err != nil {
		return false, err
	}
	changed, err := scrubHost(inner.ProtoReflect(), replace)
	if err != nil || !changed {
		return false, err
	}
	value, err := proto.MarshalOptions{Deterministic: true}.Marshal(inner)
	if err != nil {
		return false, err
	}
	packed.Value = value
	return true, nil
}

// requireRepeatedRuns runs every bound Case runsPerCase times, concurrently when concurrent is set,
// and hands each closed Run to check with the index of the Case it ran under. Every closed Run is
// captured before any is asserted on, so a failing one leaves the others' timing behind as well.
// It then asserts what every repeated-run live test shows: the Run IDs are distinct, each Run's
// workflow exists in its own namespace and in no other bound one, and the Case bytes, the prepared
// snapshots and the frozen bindings are unchanged.
func requireRepeatedRuns(
	t *testing.T,
	env *testcore.TestEnv,
	fixture string,
	caseSource *testpilotpb.Case,
	lives []testpilotLiveCase,
	runsPerCase int,
	concurrent bool,
	check func(index int, run *testpilotpb.Run, verdict *testpilotpb.Verdict),
) {
	t.Helper()
	results := make(chan testpilotLiveRunResult, len(lives)*runsPerCase)
	var pending sync.WaitGroup
	for index, live := range lives {
		for range runsPerCase {
			execute := func() {
				run, verdict, err := live.prepared.Run(env.Context(), live.driver)
				results <- testpilotLiveRunResult{environment: index, run: run, verdict: verdict, err: err}
			}
			if concurrent {
				pending.Go(execute)
			} else {
				execute()
			}
		}
	}
	pending.Wait()
	close(results)

	collected := make([]testpilotLiveRunResult, 0, len(lives)*runsPerCase)
	for result := range results {
		captureRun(t, fixture, lives[result.environment], result.run)
		collected = append(collected, result)
	}
	runIDs := make(map[string]struct{}, len(collected))
	for _, result := range collected {
		require.NoError(t, result.err)
		check(result.environment, result.run, result.verdict)
		require.NotContains(t, runIDs, result.run.GetRunId())
		runIDs[result.run.GetRunId()] = struct{}{}

		for index, live := range lives {
			_, err := live.client.DescribeWorkflowExecution(env.Context(), result.run.GetRunId(), "")
			if index == result.environment {
				require.NoError(t, err)
				continue
			}
			var notFound *serviceerror.NotFound
			require.ErrorAs(t, err, &notFound)
		}
	}
	require.Len(t, runIDs, len(lives)*runsPerCase)

	caseSnapshot := loadTestpilotCase(t, fixture)
	require.True(t, proto.Equal(caseSnapshot, caseSource))
	for _, live := range lives {
		require.True(t, proto.Equal(caseSnapshot, live.prepared.Snapshot()))
		require.Equal(t, live.profile.EnvironmentBindings, live.driver.Snapshot().EnvironmentBindings)
	}
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
		require.Equal(t, nexusEvidenceKinds[historyEvent.GetEventType()], evidence.GetKind())
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

// nexusEvidenceKinds is the Case-local name of the evidence kind each Nexus operation history event
// kind lifts into.
var nexusEvidenceKinds = map[enumspb.EventType]string{
	enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED: "scheduled",
	enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED:   "started",
	enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED: "completed",
	enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED:    "failed",
	enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT: "timedOut",
}

// nexusOperationReference is how every Nexus operation event after the scheduled one names its
// operation.
type nexusOperationReference interface {
	GetScheduledEventId() int64
	GetRequestId() string
}

// nexusOperationCoordinates reads the scheduled event id and the request id one Nexus history event
// carries for its operation; the scheduled event names itself.
func nexusOperationCoordinates(t testing.TB, event *historypb.HistoryEvent) (int64, string) {
	t.Helper()
	_, isNexus := nexusEvidenceKinds[event.GetEventType()]
	require.True(t, isNexus, "history event is not a Nexus operation event: %s", event.GetEventType())
	if scheduled := event.GetNexusOperationScheduledEventAttributes(); scheduled != nil {
		return event.GetEventId(), scheduled.GetRequestId()
	}
	message := event.ProtoReflect()
	field := message.WhichOneof(message.Descriptor().Oneofs().ByName("attributes"))
	require.NotNil(t, field, "Nexus operation event without attributes: %s", event.GetEventType())
	reference, ok := message.Get(field).Message().Interface().(nexusOperationReference)
	require.True(t, ok, "Nexus operation event attributes name no operation: %s", event.GetEventType())
	return reference.GetScheduledEventId(), reference.GetRequestId()
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
