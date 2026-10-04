package delivery

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// activityFixture is one carried activity start. Its carrier reserves one activation per attempt
// the script declares; reservation and retained are the first attempt's.
type activityFixture struct {
	ledger       *Ledger
	origin       testpilot.Coordinate
	plan         testpilot.ReservationCarrierPlan
	binding      ActivityBinding
	reservation  *testsupport.Reservation
	retained     testpilot.ReservationHandle
	reservations []*testsupport.Reservation
	handles      []testpilot.ReservationHandle
	bundle       Bundle
}

func activityPlan() testpilot.ReservationCarrierPlan {
	return activityPlanOf(1)
}

func activityPlanOf(attempts int64) testpilot.ReservationCarrierPlan {
	return testpilot.ReservationCarrierPlan{
		EndpointRoleID: "temporal",
		Method:         StartActivityPath,
		Reservations:   []testpilot.ReservationTopology{{EntrypointID: "activity", Kind: testpilot.ActivityEntrypoint, Count: attempts}},
	}
}

func newActivityFixture(t *testing.T, runID, sessionID string) *activityFixture {
	t.Helper()
	return newActivityFixtureOf(t, runID, sessionID, 1)
}

// newActivityFixtureOf reserves attempts activations; the first keeps the single-attempt fixture's
// reservation identity and the others are numbered by their ordinal.
func newActivityFixtureOf(t *testing.T, runID, sessionID string, attempts int64) *activityFixture {
	t.Helper()
	ledger, err := New(Config{RunID: runID, SessionID: sessionID, Limits: Limits{MaxRoutes: 8, MaxHeaderBytes: 4096, MaxHandles: 8, MaxDiagnostics: 8}})
	require.NoError(t, err)
	origin := testpilot.Coordinate{RunID: runID, EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start-activity", Attempt: 1}
	f := &activityFixture{ledger: ledger, origin: origin, plan: activityPlanOf(attempts), binding: ActivityBinding{Namespace: "namespace", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "task-queue"}}
	for ordinal := int64(0); ordinal < attempts; ordinal++ {
		id := sessionID + "-activity"
		if ordinal > 0 {
			id = fmt.Sprintf("%s-activity-%d", sessionID, ordinal)
		}
		reservation := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "activity", Ordinal: ordinal, ID: id})
		retained, err := ledger.RetainReservation(context.Background(), reservation)
		require.NoError(t, err)
		f.reservations, f.handles = append(f.reservations, reservation), append(f.handles, retained)
	}
	f.reservation, f.retained = f.reservations[0], f.handles[0]
	// The handles arrive in no particular order; the bundle orders the attempts by ordinal.
	reversed := slices.Clone(f.handles)
	slices.Reverse(reversed)
	f.bundle, err = ledger.CreateActivityBundle(context.Background(), origin, f.plan, f.binding, reversed)
	require.NoError(t, err)
	return f
}

func startActivityMethod(t *testing.T) protoreflect.MethodDescriptor {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("temporal.api.workflowservice.v1.WorkflowService.StartActivityExecution")
	require.NoError(t, err)
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	return method
}

func activityRequest(f *activityFixture) *workflowservice.StartActivityExecutionRequest {
	return &workflowservice.StartActivityExecutionRequest{
		Namespace:    f.binding.Namespace,
		ActivityId:   f.binding.ActivityID,
		ActivityType: &commonpb.ActivityType{Name: f.binding.ActivityType},
		TaskQueue:    &taskqueuepb.TaskQueue{Name: f.binding.TaskQueue},
		RequestId:    "application-request-id",
	}
}

func activityHeader(t *testing.T, f *activityFixture) *commonpb.Header {
	t.Helper()
	prepared, err := f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startActivityMethod(t), activityRequest(f), 1<<20)
	require.NoError(t, err)
	return prepared.(*workflowservice.StartActivityExecutionRequest).Header
}

// delivery is the first SDK attempt of the activity run, under one delivery identity.
func (f *activityFixture) delivery(header *commonpb.Header, activityRunID string) ActivityDelivery {
	return f.attempt(header, activityRunID, 1, "delivery-1")
}

func (f *activityFixture) attempt(header *commonpb.Header, activityRunID string, attempt int32, deliveryID string) ActivityDelivery {
	return ActivityDelivery{Header: header, Namespace: f.binding.Namespace, ActivityID: f.binding.ActivityID, ActivityType: f.binding.ActivityType, TaskQueue: f.binding.TaskQueue, ActivityRunID: activityRunID, Attempt: attempt, DeliveryID: deliveryID}
}

// An activity route rides in the start request's header as the workflow route does, so its wire
// bytes are pinned beside theirs. It names the activity it starts and no workflow.
func TestActivityRouteWireBytesGolden(t *testing.T) {
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1}
	value := route{
		Version:     routeVersion,
		Kind:        activityRoute,
		SessionID:   "session",
		RunID:       "run",
		Origin:      origin,
		Reservation: testpilot.ReservationIdentity{Origin: origin, EntrypointID: "activity", ID: "activity-reservation"},
		Activity:    ActivityBinding{Namespace: "namespace", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "task-queue"},
	}
	want := `{"version":1,"kind":"activity","session_id":"session","run_id":"run","origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller.0","InstructionID":"start","Attempt":1},"reservation":{"Origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller.0","InstructionID":"start","Attempt":1},"EntrypointID":"activity","Ordinal":0,"ID":"activity-reservation"},"workflow_ordinal":0,"activity":{"namespace":"namespace","activity_id":"activity-id","activity_type":"activity-type","task_queue":"task-queue"}}`
	codec := routeCodec{maximumBytes: 2048}

	encoded, err := codec.encode(value)
	require.NoError(t, err)
	require.Equal(t, want, string(encoded))
	decoded, err := codec.decode([]byte(want), activityRoute)
	require.NoError(t, err)
	require.Equal(t, value, decoded)

	workflowBinding := WorkflowBinding{Namespace: "namespace", WorkflowID: "workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	for name, mutate := range map[string]func(*route){
		"an activity route that names a workflow": func(r *route) { r.Binding = workflowBinding },
		"an activity route without its activity":  func(r *route) { r.Activity.ActivityID = "" },
		"an activity route with a workflow run":   func(r *route) { r.WorkflowRunID = "temporal-run" },
		"an activity route with a source":         func(r *route) { r.SourceInstructionID = "start-nexus" },
		"a workflow route that names an activity": func(r *route) { r.Kind, r.Binding = workflowRoute, workflowBinding },
		"a Nexus route that names an activity": func(r *route) {
			r.Kind, r.Binding = nexusRoute, workflowBinding
			r.WorkflowReservation, r.WorkflowEntrypoint, r.WorkflowRunID, r.SourceInstructionID = "workflow-reservation", "workflow", "temporal-run", "start-nexus"
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := value
			mutate(&invalid)
			_, err := codec.encode(invalid)
			require.ErrorIs(t, err, ErrRouteMalformed)
			// The route is malformed for the binding it should not name, and is well formed without it.
			if invalid.Kind != activityRoute {
				invalid.Activity = ActivityBinding{}
				_, err = codec.encode(invalid)
				require.NoError(t, err)
			}
		})
	}
	_, err = codec.decode([]byte(want), workflowRoute)
	require.ErrorIs(t, err, ErrRouteCrossed)
}

func TestPrepareRPCInjectsOnlyTheActivityRoute(t *testing.T) {
	f := newActivityFixture(t, "run", "session")
	request := activityRequest(f)
	request.Header = &commonpb.Header{Fields: map[string]*commonpb.Payload{"application": {Data: []byte("kept")}}}
	request.Input = &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("input")}}}
	snapshot := proto.CloneOf(request)

	preparedMessage, err := f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startActivityMethod(t), request, 1<<20)
	require.NoError(t, err)
	prepared := preparedMessage.(*workflowservice.StartActivityExecutionRequest)
	require.NotSame(t, request, prepared)
	require.True(t, proto.Equal(snapshot, request))
	require.Contains(t, prepared.Header.Fields, reservedActivityHeader)
	require.NotContains(t, prepared.Header.Fields, reservedWorkflowHeader)
	delete(prepared.Header.Fields, reservedActivityHeader)
	require.True(t, proto.Equal(request, prepared))

	_, err = f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startActivityMethod(t), activityRequest(f), int64(proto.Size(preparedMessage)-1))
	require.ErrorIs(t, err, ErrCapacity)

	dynamicRequest := dynamicpb.NewMessage(startActivityMethod(t).Input())
	wire, err := proto.Marshal(activityRequest(f))
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(wire, dynamicRequest))
	dynamicPrepared, err := f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startActivityMethod(t), dynamicRequest, 1<<20)
	require.NoError(t, err)
	wire, err = proto.Marshal(dynamicPrepared)
	require.NoError(t, err)
	var decoded workflowservice.StartActivityExecutionRequest
	require.NoError(t, proto.Unmarshal(wire, &decoded))
	require.Contains(t, decoded.Header.Fields, reservedActivityHeader)
}

func TestPrepareRPCRejectsActivityCollisionBindingAndCrossedCarriers(t *testing.T) {
	for name, test := range map[string]struct {
		mutate func(*workflowservice.StartActivityExecutionRequest)
		err    error
	}{
		"reserved collision": {func(request *workflowservice.StartActivityExecutionRequest) {
			request.Header = &commonpb.Header{Fields: map[string]*commonpb.Payload{reservedActivityHeader: {Data: []byte("anything")}}}
		}, ErrReservedHeader},
		"namespace":        {func(request *workflowservice.StartActivityExecutionRequest) { request.Namespace = "other" }, ErrBindingMismatch},
		"activity id":      {func(request *workflowservice.StartActivityExecutionRequest) { request.ActivityId = "other" }, ErrBindingMismatch},
		"activity type":    {func(request *workflowservice.StartActivityExecutionRequest) { request.ActivityType.Name = "other" }, ErrBindingMismatch},
		"task queue":       {func(request *workflowservice.StartActivityExecutionRequest) { request.TaskQueue.Name = "other" }, ErrBindingMismatch},
		"no activity type": {func(request *workflowservice.StartActivityExecutionRequest) { request.ActivityType = nil }, ErrInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			f := newActivityFixture(t, "run", "session")
			request := activityRequest(f)
			test.mutate(request)
			_, err := f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startActivityMethod(t), request, 1<<20)
			require.ErrorIs(t, err, test.err)
		})
	}

	// A carrier prepares only the start its plan names: an activity carrier refuses a workflow start,
	// and a workflow carrier an activity start.
	activity := newActivityFixture(t, "run", "session")
	workflow := newFixture(t, "run", "session")
	_, err := activity.ledger.PrepareRPC(context.Background(), &activity.bundle, "temporal", startMethod(t), workflowRequest(workflow), 1<<20)
	require.ErrorIs(t, err, ErrRouteCrossed)
	_, err = workflow.ledger.PrepareRPC(context.Background(), &workflow.bundle, "temporal", startActivityMethod(t), activityRequest(activity), 1<<20)
	require.ErrorIs(t, err, ErrRouteCrossed)
	_, err = activity.ledger.PrepareRPC(context.Background(), &activity.bundle, "other-role", startActivityMethod(t), activityRequest(activity), 1<<20)
	require.ErrorIs(t, err, ErrRouteCrossed)
}

func TestActivityStartBindingReadsDynamicStartRequestsOnly(t *testing.T) {
	f := newActivityFixture(t, "run", "session")
	request := activityRequest(f)
	request.Header = &commonpb.Header{Fields: map[string]*commonpb.Payload{"application": {Data: []byte("kept")}}}
	wire, err := proto.Marshal(request)
	require.NoError(t, err)
	dynamicRequest := dynamicpb.NewMessage(startActivityMethod(t).Input())
	require.NoError(t, proto.Unmarshal(wire, dynamicRequest))
	binding, header, err := ActivityStartBinding(dynamicRequest)
	require.NoError(t, err)
	require.Equal(t, f.binding, binding)
	require.NotNil(t, header)

	for name, message := range map[string]proto.Message{
		"a workflow start":     &workflowservice.StartWorkflowExecutionRequest{},
		"no activity type":     &workflowservice.StartActivityExecutionRequest{TaskQueue: &taskqueuepb.TaskQueue{Name: "task-queue"}},
		"no task queue":        &workflowservice.StartActivityExecutionRequest{ActivityType: &commonpb.ActivityType{Name: "activity-type"}},
		"an unrelated message": &commonpb.Payload{},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := ActivityStartBinding(message.ProtoReflect())
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

// admission is what an Activation says of the delivery it admitted: the reservation that names the
// activation, the logical operation's run, the SDK attempt, the delivery, and whether it is a replay.
type admission struct {
	Coordinate    testpilot.Coordinate
	Reservation   string
	ActivityRunID string
	Attempt       int32
	DeliveryID    string
	Replay        bool
}

func admissionOf(activation Activation) admission {
	return admission{Coordinate: activation.Coordinate(), Reservation: activation.Reservation().ID, ActivityRunID: activation.TemporalRunID(), Attempt: activation.Attempt(), DeliveryID: activation.DeliveryID(), Replay: activation.Replay()}
}

// The script's Nth instruction is the server's Nth attempt, so attempt N is admitted under the
// reservation of ordinal N-1 whatever order the attempts arrive in. The operation, the attempt and
// the delivery are three identities: the run names the operation across attempts, the attempt
// number is the server's, and the reservation names the activation the attempt runs as. A delivery
// of an admitted attempt is a replay of it. An attempt the script does not declare is refused as
// that, and is never answered as another attempt was.
func TestAdmitActivityBindsEachAttemptToTheReservationOfItsNumber(t *testing.T) {
	f := newActivityFixtureOf(t, "run", "session", 3)
	header := activityHeader(t, f)
	admittedAs := func(ordinal int, attempt int32, deliveryID string) admission {
		id := "session-activity"
		if ordinal > 0 {
			id = fmt.Sprintf("session-activity-%d", ordinal)
		}
		return admission{Coordinate: testpilot.Coordinate{RunID: "run", EntrypointID: "activity", ActivationID: id}, Reservation: id, ActivityRunID: "activity-run", Attempt: attempt, DeliveryID: deliveryID}
	}
	replayOf := func(a admission) admission {
		a.Replay = true
		return a
	}
	consumed := func() []int64 {
		return []int64{f.reservations[0].Consumes(), f.reservations[1].Consumes(), f.reservations[2].Consumes()}
	}

	for _, step := range []struct {
		name     string
		delivery ActivityDelivery
		want     admission
		wantUsed []int64
	}{
		// The second attempt overtakes the first: each still gets the reservation of its number.
		{"the second attempt, first to arrive", f.attempt(header, "activity-run", 2, "delivery-2"), admittedAs(1, 2, "delivery-2"), []int64{0, 1, 0}},
		{"the first attempt, late", f.attempt(header, "activity-run", 1, "delivery-1"), admittedAs(0, 1, "delivery-1"), []int64{1, 1, 0}},
		{"the same delivery again", f.attempt(header, "activity-run", 1, "delivery-1"), replayOf(admittedAs(0, 1, "delivery-1")), []int64{1, 1, 0}},
		{"the same attempt redelivered", f.attempt(header, "activity-run", 2, "redelivery-2"), replayOf(admittedAs(1, 2, "delivery-2")), []int64{1, 1, 0}},
		{"the third attempt", f.attempt(header, "activity-run", 3, "delivery-3"), admittedAs(2, 3, "delivery-3"), []int64{1, 1, 1}},
	} {
		activation, err := f.ledger.AdmitActivity(context.Background(), step.delivery)
		require.NoError(t, err, step.name)
		require.Equal(t, step.want, admissionOf(activation), step.name)
		require.Equal(t, step.wantUsed, consumed(), step.name)
	}

	for name, test := range map[string]struct {
		delivery ActivityDelivery
		err      error
	}{
		"an attempt past the script":  {f.attempt(header, "activity-run", 4, "delivery-4"), ErrAttemptUndeclared},
		"another run":                 {f.attempt(header, "other-run", 2, "delivery-2"), ErrRouteConflict},
		"another run's later attempt": {f.attempt(header, "other-run", 4, "delivery-4"), ErrRouteConflict},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := f.ledger.AdmitActivity(context.Background(), test.delivery)
			require.ErrorIs(t, err, test.err)
			require.Equal(t, []int64{1, 1, 1}, consumed())
		})
	}
}

// Once the activity is closed the server sends no other attempt, so the reservations after the
// last attempt admitted are released as not needed: each is named once, admits nothing afterwards,
// and the attempts already admitted are untouched and still replay.
func TestReleaseActivityAttemptsReleasesTheAttemptsAfterTheLastAdmitted(t *testing.T) {
	f := newActivityFixtureOf(t, "run", "session", 3)
	header := activityHeader(t, f)
	first, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "delivery-1"))
	require.NoError(t, err)

	released, err := f.ledger.ReleaseActivityAttempts(context.Background(), first)
	require.NoError(t, err)
	require.Equal(t, []testpilot.ReservationIdentity{f.reservations[1].Identity(), f.reservations[2].Identity()}, released)
	again, err := f.ledger.ReleaseActivityAttempts(context.Background(), first)
	require.NoError(t, err)
	require.Empty(t, again)

	for _, attempt := range []int32{2, 3} {
		_, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", attempt, "delivery"))
		require.ErrorIs(t, err, ErrRouteStale)
	}
	replay, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "delivery-1"))
	require.NoError(t, err)
	require.True(t, replay.Replay())
	require.Equal(t, []int64{1, 0, 0}, []int64{f.reservations[0].Consumes(), f.reservations[1].Consumes(), f.reservations[2].Consumes()})
	// The ledger only names the released reservations; settling them is their Session's.
	require.Zero(t, f.reservations[1].Cancels())

	// What is released is what follows the last attempt admitted, whichever activation of the
	// activity asks: with the third attempt admitted, the second, which the worker never saw, is
	// not released and still admits.
	g := newActivityFixtureOf(t, "run", "session", 3)
	header = activityHeader(t, g)
	_, err = g.ledger.AdmitActivity(context.Background(), g.attempt(header, "activity-run", 3, "delivery-3"))
	require.NoError(t, err)
	first, err = g.ledger.AdmitActivity(context.Background(), g.attempt(header, "activity-run", 1, "delivery-1"))
	require.NoError(t, err)
	released, err = g.ledger.ReleaseActivityAttempts(context.Background(), first)
	require.NoError(t, err)
	require.Empty(t, released)
	_, err = g.ledger.AdmitActivity(context.Background(), g.attempt(header, "activity-run", 2, "delivery-2"))
	require.NoError(t, err)

	// An earlier attempt the worker never saw was not made needless by the activity closing; it
	// stays reserved and admissible.
	h := newActivityFixtureOf(t, "run", "session", 3)
	header = activityHeader(t, h)
	second, err := h.ledger.AdmitActivity(context.Background(), h.attempt(header, "activity-run", 2, "delivery-2"))
	require.NoError(t, err)
	released, err = h.ledger.ReleaseActivityAttempts(context.Background(), second)
	require.NoError(t, err)
	require.Equal(t, []testpilot.ReservationIdentity{h.reservations[2].Identity()}, released)
	_, err = h.ledger.AdmitActivity(context.Background(), h.attempt(header, "activity-run", 1, "delivery-1"))
	require.NoError(t, err)

	// Only an activity attempt's activation releases, and only in its own ledger.
	_, err = f.ledger.ReleaseActivityAttempts(context.Background(), second)
	require.ErrorIs(t, err, ErrRouteCrossed)
	workflow := newFixture(t, "run", "session")
	_, err = workflow.ledger.ReleaseActivityAttempts(context.Background(), admitWorkflow(t, workflow, "temporal-run"))
	require.ErrorIs(t, err, ErrRouteCrossed)
}

// The scheduler observes each attempt's reservation as it ends. The operation outlives them: a
// later attempt is still admitted, and a settled one still replays, until its carrier is done.
func TestActivityOperationOutlivesItsSettledAttempts(t *testing.T) {
	f := newActivityFixtureOf(t, "run", "session", 2)
	header := activityHeader(t, f)
	_, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "delivery-1"))
	require.NoError(t, err)
	f.reservations[0].Complete()
	_, err = f.handles[0].Wait(context.Background())
	require.NoError(t, err)

	replay, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "delivery-1"))
	require.NoError(t, err)
	require.True(t, replay.Replay())
	next, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 2, "delivery-2"))
	require.NoError(t, err)
	require.Equal(t, "session-activity-1", next.Reservation().ID)
	require.False(t, next.Replay())

	// Once the start is final and every attempt's reservation has ended, the route is gone.
	require.NoError(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartActivityExecutionResponse{RunId: "activity-run"}))
	_, err = f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerSucceeded)
	require.NoError(t, err)
	f.reservations[1].Complete()
	_, err = f.handles[1].Wait(context.Background())
	require.NoError(t, err)
	_, err = f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 2, "delivery-2"))
	require.ErrorIs(t, err, ErrRouteStale)
	require.Equal(t, []int64{1, 1}, []int64{f.reservations[0].Consumes(), f.reservations[1].Consumes()})
}

// Deliveries of one attempt that race admit it once: one of them consumes the reservation and
// every other is a replay of that admission.
func TestConcurrentDeliveriesOfOneAttemptAdmitItOnce(t *testing.T) {
	f := newActivityFixtureOf(t, "run", "session", 2)
	header := activityHeader(t, f)
	const racers = 8
	results := make(chan admission, racers)
	failures := make(chan error, racers)
	start := make(chan struct{})
	for range racers {
		go func() {
			<-start
			activation, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "delivery-1"))
			if err != nil {
				failures <- err
				return
			}
			results <- admissionOf(activation)
		}()
	}
	close(start)
	fresh := 0
	for range racers {
		select {
		case err := <-failures:
			require.NoError(t, err)
		case result := <-results:
			if !result.Replay {
				fresh++
			}
			result.Replay = false
			require.Equal(t, admission{Coordinate: testpilot.Coordinate{RunID: "run", EntrypointID: "activity", ActivationID: "session-activity"}, Reservation: "session-activity", ActivityRunID: "activity-run", Attempt: 1, DeliveryID: "delivery-1"}, result)
		}
	}
	require.Equal(t, 1, fresh)
	require.Equal(t, []int64{1, 0}, []int64{f.reservations[0].Consumes(), f.reservations[1].Consumes()})
}

func TestInvalidActivityDeliveriesRejectBeforeReservationConsumption(t *testing.T) {
	f := newActivityFixture(t, "run", "session")
	valid := activityHeader(t, f)
	payload := valid.Fields[reservedActivityHeader]
	foreign := newActivityFixture(t, "run", "other-session")
	workflow := newFixture(t, "run", "session")
	workflowPayload := workflowHeader(t, workflow).Fields[reservedWorkflowHeader]

	for name, test := range map[string]struct {
		delivery ActivityDelivery
		err      error
	}{
		"no header":               {f.delivery(nil, "activity-run"), ErrRouteMissing},
		"no route":                {f.delivery(&commonpb.Header{Fields: map[string]*commonpb.Payload{"application": {Data: []byte("kept")}}}, "activity-run"), ErrRouteMissing},
		"a workflow header":       {f.delivery(&commonpb.Header{Fields: map[string]*commonpb.Payload{reservedWorkflowHeader: payload}}, "activity-run"), ErrRouteMissing},
		"malformed":               {f.delivery(&commonpb.Header{Fields: map[string]*commonpb.Payload{reservedActivityHeader: {Metadata: map[string][]byte{"encoding": []byte("wrong")}, Data: []byte("route")}}}, "activity-run"), ErrRouteMalformed},
		"oversized":               {f.delivery(&commonpb.Header{Fields: map[string]*commonpb.Payload{reservedActivityHeader: {Metadata: map[string][]byte{"encoding": []byte(workflowRouteEncoding)}, Data: make([]byte, f.ledger.config.Limits.MaxHeaderBytes+1)}}}, "activity-run"), ErrRouteOversized},
		"a workflow route":        {f.delivery(&commonpb.Header{Fields: map[string]*commonpb.Payload{reservedActivityHeader: workflowPayload}}, "activity-run"), ErrRouteCrossed},
		"another session's route": {f.delivery(activityHeader(t, foreign), "activity-run"), ErrRouteCrossed},
		"a route its reservation was not carried under": {f.delivery(&commonpb.Header{Fields: map[string]*commonpb.Payload{reservedActivityHeader: {Metadata: payload.Metadata, Data: bytes.ReplaceAll(payload.Data, []byte(`"InstructionID":"start-activity"`), []byte(`"InstructionID":"another-start"`))}}}, "activity-run"), ErrRouteCrossed},
		"no activity run":                {f.delivery(valid, ""), ErrInvalid},
		"no SDK attempt":                 {f.attempt(valid, "activity-run", 0, "delivery-1"), ErrInvalid},
		"no delivery identity":           {f.attempt(valid, "activity-run", 1, ""), ErrInvalid},
		"an unbounded delivery identity": {f.attempt(valid, "activity-run", 1, strings.Repeat("d", 257)), ErrInvalid},
		"another namespace":              {ActivityDelivery{Header: valid, Namespace: "other", ActivityID: f.binding.ActivityID, ActivityType: f.binding.ActivityType, TaskQueue: f.binding.TaskQueue, ActivityRunID: "activity-run", Attempt: 1, DeliveryID: "delivery-1"}, ErrBindingMismatch},
		"another activity":               {ActivityDelivery{Header: valid, Namespace: f.binding.Namespace, ActivityID: "other", ActivityType: f.binding.ActivityType, TaskQueue: f.binding.TaskQueue, ActivityRunID: "activity-run", Attempt: 1, DeliveryID: "delivery-1"}, ErrBindingMismatch},
		"another activity type":          {ActivityDelivery{Header: valid, Namespace: f.binding.Namespace, ActivityID: f.binding.ActivityID, ActivityType: "other", TaskQueue: f.binding.TaskQueue, ActivityRunID: "activity-run", Attempt: 1, DeliveryID: "delivery-1"}, ErrBindingMismatch},
		"another task queue":             {ActivityDelivery{Header: valid, Namespace: f.binding.Namespace, ActivityID: f.binding.ActivityID, ActivityType: f.binding.ActivityType, TaskQueue: "other", ActivityRunID: "activity-run", Attempt: 1, DeliveryID: "delivery-1"}, ErrBindingMismatch},
		"no activity id":                 {ActivityDelivery{Header: valid, Namespace: f.binding.Namespace, ActivityType: f.binding.ActivityType, TaskQueue: f.binding.TaskQueue, ActivityRunID: "activity-run", Attempt: 1, DeliveryID: "delivery-1"}, ErrInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := f.ledger.AdmitActivity(context.Background(), test.delivery)
			require.ErrorIs(t, err, test.err)
			require.Zero(t, f.reservation.Consumes())
		})
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := f.ledger.AdmitActivity(canceled, f.delivery(valid, "activity-run"))
	requireContextError(t, err)
	require.Zero(t, f.reservation.Consumes())
}

// The run a carried start answers with and the run the worker is delivered are one logical
// operation, whichever is seen first.
func TestActivityStartResponseMustAgreeWithItsDelivery(t *testing.T) {
	f := newActivityFixture(t, "run", "session")
	header := activityHeader(t, f)
	require.NoError(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartActivityExecutionResponse{RunId: "activity-run"}))
	require.ErrorIs(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartActivityExecutionResponse{RunId: "crossed"}), ErrRouteConflict)
	_, err := f.ledger.AdmitActivity(context.Background(), f.delivery(header, "crossed"))
	require.ErrorIs(t, err, ErrRouteConflict)
	require.Zero(t, f.reservation.Consumes())
	_, err = f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
	require.NoError(t, err)

	other := newActivityFixture(t, "other-run", "other-session")
	_, err = other.ledger.AdmitActivity(context.Background(), other.delivery(activityHeader(t, other), "delivered-first"))
	require.NoError(t, err)
	require.ErrorIs(t, other.ledger.PinStartResponse(context.Background(), other.bundle, &workflowservice.StartActivityExecutionResponse{RunId: "crossed"}), ErrRouteConflict)
	require.NoError(t, other.ledger.PinStartResponse(context.Background(), other.bundle, &workflowservice.StartActivityExecutionResponse{RunId: "delivered-first"}))
	require.ErrorIs(t, other.ledger.PinStartResponse(context.Background(), other.bundle, &workflowservice.StartActivityExecutionResponse{}), ErrInvalid)
}

func TestActivityTriggerFailuresAndStopRetireTheRoute(t *testing.T) {
	for _, disposition := range []TriggerStatus{TriggerRejected, TriggerCanceled, TriggerNonSuccess, TriggerUncertain} {
		t.Run("before delivery/"+disposition.String(), func(t *testing.T) {
			f := newActivityFixture(t, "run", "session")
			header := activityHeader(t, f)
			release, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, disposition)
			require.NoError(t, err)
			require.Equal(t, 1, release.Unused())
			require.Equal(t, int64(1), f.reservation.Cancels())
			_, err = f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
			require.ErrorIs(t, err, ErrRouteStale)
			require.Zero(t, f.reservation.Consumes())
		})
		t.Run("after delivery/"+disposition.String(), func(t *testing.T) {
			f := newActivityFixture(t, "run", "session")
			header := activityHeader(t, f)
			_, err := f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
			require.NoError(t, err)
			release, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, disposition)
			require.NoError(t, err)
			require.Zero(t, release.Unused())
			require.Equal(t, int64(1), f.reservation.Cancels())
			// The attempt was admitted, but its start did not stand, so its redelivery replays nothing.
			_, err = f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
			require.ErrorIs(t, err, ErrRouteStale)
		})
	}

	t.Run("succeeded", func(t *testing.T) {
		f := newActivityFixture(t, "run", "session")
		header := activityHeader(t, f)
		_, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerSucceeded)
		require.ErrorIs(t, err, ErrRouteConflict)
		require.NoError(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartActivityExecutionResponse{RunId: "activity-run"}))
		release, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerSucceeded)
		require.NoError(t, err)
		require.Zero(t, release.Unused())
		require.Zero(t, f.reservation.Cancels())
		_, err = f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
		require.NoError(t, err)
	})

	t.Run("stop", func(t *testing.T) {
		f := newActivityFixture(t, "run", "session")
		header := activityHeader(t, f)
		release, err := f.ledger.Stop(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, release.Unused())
		require.Equal(t, int64(1), f.reservation.Cancels())
		_, err = f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
		require.ErrorIs(t, err, ErrRouteStale)
		_, err = f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startActivityMethod(t), activityRequest(f), 1<<20)
		require.ErrorIs(t, err, ErrRouteStale)
	})

	t.Run("stop after delivery", func(t *testing.T) {
		f := newActivityFixture(t, "run", "session")
		header := activityHeader(t, f)
		_, err := f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
		require.NoError(t, err)
		_, err = f.ledger.Stop(context.Background())
		require.NoError(t, err)
		_, err = f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
		require.ErrorIs(t, err, ErrRouteStale)
	})
}

// A reservation that cannot be consumed, because its Run canceled it while the task was in flight,
// admits nothing and is canceled once.
func TestAdmitActivityRejectsAReservationItCannotConsume(t *testing.T) {
	f := newActivityFixture(t, "run", "session")
	header := activityHeader(t, f)
	f.reservation.OnConsume = func(context.Context) (testpilot.Coordinate, error) {
		return testpilot.Coordinate{}, context.Canceled
	}
	_, err := f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
	require.ErrorIs(t, err, ErrLifecycle)
	require.Equal(t, int64(1), f.reservation.Cancels())
	_, err = f.ledger.AdmitActivity(context.Background(), f.delivery(header, "activity-run"))
	require.ErrorIs(t, err, ErrRouteStale)
}

// Two Runs that start the same physical activity binding keep their own routes: each admits only
// the delivery that carries its own route, in either order.
func TestConcurrentRunsRouteActivitiesByIdentity(t *testing.T) {
	first := newActivityFixture(t, "run-a", "session-a")
	second := newActivityFixture(t, "run-b", "session-b")
	firstHeader, secondHeader := activityHeader(t, first), activityHeader(t, second)

	_, err := first.ledger.AdmitActivity(context.Background(), first.delivery(secondHeader, "activity-run-b"))
	require.ErrorIs(t, err, ErrRouteCrossed)
	admittedSecond, err := second.ledger.AdmitActivity(context.Background(), second.delivery(secondHeader, "activity-run-b"))
	require.NoError(t, err)
	admittedFirst, err := first.ledger.AdmitActivity(context.Background(), first.delivery(firstHeader, "activity-run-a"))
	require.NoError(t, err)
	require.Equal(t, testpilot.Coordinate{RunID: "run-a", EntrypointID: "activity", ActivationID: "session-a-activity"}, admittedFirst.Coordinate())
	require.Equal(t, testpilot.Coordinate{RunID: "run-b", EntrypointID: "activity", ActivationID: "session-b-activity"}, admittedSecond.Coordinate())
	require.Equal(t, int64(1), first.reservation.Consumes())
	require.Equal(t, int64(1), second.reservation.Consumes())
}

func TestCreateBundleAdmitsOnlyTheReservationsItsStartActivates(t *testing.T) {
	binding := ActivityBinding{Namespace: "namespace", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "task-queue"}
	activity := reservationKey{entrypoint: "activity"}
	// Each case supplies exactly the handles its plan expects, so only the plan's shape rejects.
	for name, test := range map[string]struct {
		mutate  func(*testpilot.ReservationCarrierPlan, *ActivityBinding)
		handles []reservationKey
	}{
		"a workflow start": {func(plan *testpilot.ReservationCarrierPlan, _ *ActivityBinding) {
			plan.Method = primitive.StartWorkflowPath
		}, []reservationKey{activity}},
		"a workflow reservation": {func(plan *testpilot.ReservationCarrierPlan, _ *ActivityBinding) {
			plan.Reservations[0].Kind = testpilot.WorkflowEntrypoint
		}, []reservationKey{activity}},
		"no activation of the activity": {func(plan *testpilot.ReservationCarrierPlan, _ *ActivityBinding) {
			plan.Reservations[0].Count = 0
		}, nil},
		"a second reservation": {func(plan *testpilot.ReservationCarrierPlan, _ *ActivityBinding) {
			plan.Reservations = append(plan.Reservations, testpilot.ReservationTopology{EntrypointID: "handler", Kind: testpilot.NexusHandlerEntrypoint, Count: 1})
		}, []reservationKey{activity, {entrypoint: "handler"}}},
		"a second activity": {func(plan *testpilot.ReservationCarrierPlan, _ *ActivityBinding) {
			plan.Reservations = append(plan.Reservations, testpilot.ReservationTopology{EntrypointID: "other", Kind: testpilot.ActivityEntrypoint, Count: 1})
		}, []reservationKey{activity, {entrypoint: "other"}}},
		"a Nexus route": {func(plan *testpilot.ReservationCarrierPlan, _ *ActivityBinding) {
			plan.Routes = []testpilot.ReservationRoute{{WorkflowEntrypointID: "activity", SourceInstructionID: "start", HandlerEntrypointID: "activity"}}
		}, []reservationKey{activity}},
		"no activity id": {func(_ *testpilot.ReservationCarrierPlan, binding *ActivityBinding) { binding.ActivityID = "" }, []reservationKey{activity}},
	} {
		t.Run(name, func(t *testing.T) {
			ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 8, MaxHeaderBytes: 4096, MaxHandles: 8, MaxDiagnostics: 8}})
			require.NoError(t, err)
			origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start-activity", Attempt: 1}
			handles := make([]testpilot.ReservationHandle, 0, len(test.handles))
			for _, key := range test.handles {
				retained, err := ledger.RetainReservation(context.Background(), testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: key.entrypoint, Ordinal: key.ordinal, ID: fmt.Sprintf("%s-%d", key.entrypoint, key.ordinal)}))
				require.NoError(t, err)
				handles = append(handles, retained)
			}
			plan, bound := activityPlan(), binding
			test.mutate(&plan, &bound)
			_, err = ledger.CreateActivityBundle(context.Background(), origin, plan, bound, handles)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}

	// A workflow start carries workflows and the Nexus handlers and activities they reach, and its
	// activity's attempts are routed by the workflow, never started as a standalone activity.
	f := newFixture(t, "existing-run", "existing-session")
	ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 8, MaxHeaderBytes: 4096, MaxHandles: 8, MaxDiagnostics: 8}})
	require.NoError(t, err)
	origin := f.origin
	origin.RunID = "run"
	plan := f.plan
	plan.Reservations = append([]testpilot.ReservationTopology{}, plan.Reservations...)
	plan.Reservations[1].Kind = testpilot.ActivityEntrypoint
	var handles []testpilot.ReservationHandle
	for _, entrypoint := range []string{"workflow", "handler"} {
		retained, err := ledger.RetainReservation(context.Background(), testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: entrypoint, ID: entrypoint}))
		require.NoError(t, err)
		handles = append(handles, retained)
	}
	_, err = ledger.CreateBundle(context.Background(), origin, plan, f.binding, handles)
	require.NoError(t, err)
	require.Empty(t, ledger.operations)
}
