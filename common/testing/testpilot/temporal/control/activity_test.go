//go:build test_dep

package control

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/server/api/historyservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/chasm"
	serviceerrors "go.temporal.io/server/common/serviceerror"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/testing/testhooks"
)

const scope = namespace.ID("namespace")

type server struct {
	hooks    testhooks.TestHooks
	dispatch func(context.Context, testhooks.ActivityDelivery) error
	answer   func(request, response any, err error)
}

func newServer(t *testing.T) (*server, *Deliveries) {
	s := &server{hooks: testhooks.NewTestHooks()}
	d, err := NewDeliveries(func(h testhooks.Hook) func() { return h.Apply(s.hooks, scope) })
	require.NoError(t, err)
	t.Cleanup(d.Close)
	var ok bool
	s.dispatch, ok = testhooks.Get(s.hooks, testhooks.ActivityDispatch, scope)
	require.True(t, ok)
	respond, ok := testhooks.Get(s.hooks, testhooks.GRPCResponseFaultGeneratorByNamespaceID, scope)
	require.True(t, ok)
	s.answer = func(request, response any, err error) {
		require.Nil(t, respond(t.Context(), "/temporal.server.api.historyservice.v1.HistoryService/RecordActivityTaskStarted", request, response, err))
	}
	return s, d
}

func started(t *testing.T, key chasm.ExecutionKey, stamp int32) *historyservice.RecordActivityTaskStartedRequest {
	component := chasm.NewComponentRefByArchetypeID(key, 1)
	ref, err := component.Serialize(nil)
	require.NoError(t, err)
	return &historyservice.RecordActivityTaskStartedRequest{ComponentRef: ref, Stamp: stamp}
}

// The held dispatch is released only by Release, and what admission decided for that delivery is
// the release's result. Answers for another run, another stamp, an answer that decides nothing,
// and one given before the release are not the released delivery's decision.
func TestReleaseReportsAdmissionsDecisionForTheHeldDelivery(t *testing.T) {
	key := chasm.ExecutionKey{NamespaceID: string(scope), BusinessID: "activity", RunID: "run"}
	obsolete := serviceerrors.NewObsoleteMatchingTask("activity attempt stamp mismatch")
	for _, test := range []struct {
		name     string
		response any
		err      error
		want     *testpilotspb.DeliveryAdmission
	}{
		{"rejected", nil, obsolete, &testpilotspb.DeliveryAdmission{ActivityId: "activity", ActivityRunId: "run", DeliveryId: "7",
			Decision: testpilotspb.DELIVERY_ADMISSION_DECISION_REJECTED}},
		{"admitted", &historyservice.RecordActivityTaskStartedResponse{Attempt: 1}, nil, &testpilotspb.DeliveryAdmission{ActivityId: "activity",
			ActivityRunId: "run", DeliveryId: "7", Decision: testpilotspb.DELIVERY_ADMISSION_DECISION_ADMITTED, Attempt: 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, d := newServer(t)
			held, err := d.Hold("activity")
			require.NoError(t, err)
			dispatched := make(chan error, 1)
			go func() { dispatched <- s.dispatch(t.Context(), testhooks.ActivityDelivery{Execution: key, Stamp: 7}) }()
			delivery, err := held.Await(t.Context())
			require.NoError(t, err)
			require.Equal(t, testhooks.ActivityDelivery{Execution: key, Stamp: 7}, delivery)
			s.answer(started(t, key, 7), nil, obsolete)
			select {
			case err := <-dispatched:
				t.Fatalf("the dispatch crossed the hold before release: %v", err)
			default:
			}
			admission, err := held.Release(t.Context(), func(ctx context.Context) error {
				require.NoError(t, <-dispatched)
				crossed := key
				crossed.RunID = "other"
				s.answer(started(t, crossed, 7), nil, obsolete)
				s.answer(started(t, key, 8), nil, obsolete)
				s.answer(started(t, key, 7), nil, serviceerror.NewUnavailable("lost"))
				s.answer(&historyservice.RecordActivityTaskHeartbeatRequest{}, nil, obsolete)
				s.answer(started(t, key, 7), test.response, test.err)
				<-ctx.Done()
				return ctx.Err()
			})
			require.NoError(t, err)
			require.Equal(t, test.want, admission)
		})
	}
}

func TestReleaseFailsWithoutAHeldDeliveryOrADecision(t *testing.T) {
	key := chasm.ExecutionKey{NamespaceID: string(scope), BusinessID: "activity", RunID: "run"}
	s, d := newServer(t)
	held, err := d.Hold("activity")
	require.NoError(t, err)
	_, err = d.Hold("activity")
	require.ErrorIs(t, err, ErrConflict)
	_, err = held.Release(t.Context(), func(context.Context) error { return nil })
	require.ErrorIs(t, err, ErrNotHeld)

	go func() { _ = s.dispatch(t.Context(), testhooks.ActivityDelivery{Execution: key, Stamp: 7}) }()
	_, err = held.Await(t.Context())
	require.NoError(t, err)
	failed := errors.New("poll failed")
	_, err = held.Release(t.Context(), func(context.Context) error { return failed })
	require.ErrorIs(t, err, failed)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	other, err := d.Hold("other")
	require.NoError(t, err)
	go func() { _ = s.dispatch(t.Context(), testhooks.ActivityDelivery{Execution: chasm.ExecutionKey{BusinessID: "other"}, Stamp: 1}) }()
	_, err = other.Await(t.Context())
	require.NoError(t, err)
	_, err = other.Release(ctx, func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
	require.ErrorIs(t, err, context.Canceled)
}

// A dispatch of an activity nothing holds passes at once, and closing cancels a held one rather
// than letting cleanup dispatch work nobody released.
func TestUnheldDispatchPassesAndCloseCancelsAHeldOne(t *testing.T) {
	s, d := newServer(t)
	require.NoError(t, s.dispatch(t.Context(), testhooks.ActivityDelivery{Execution: chasm.ExecutionKey{BusinessID: "free"}, Stamp: 1}))
	held, err := d.Hold("activity")
	require.NoError(t, err)
	dispatched := make(chan error, 1)
	go func() {
		dispatched <- s.dispatch(t.Context(), testhooks.ActivityDelivery{Execution: chasm.ExecutionKey{BusinessID: "activity"}, Stamp: 1})
	}()
	_, err = held.Await(t.Context())
	require.NoError(t, err)
	held.Close()
	require.ErrorIs(t, <-dispatched, ErrClosed)
	again, err := d.Hold("activity")
	require.NoError(t, err)
	d.Close()
	_, err = d.Hold("next")
	require.ErrorIs(t, err, ErrClosed)
	require.ErrorIs(t, again.gate.Arrive(t.Context(), "activity"), ErrClosed)
}
