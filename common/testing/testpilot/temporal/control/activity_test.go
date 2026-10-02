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
	"go.temporal.io/server/common/namespace"
	serviceerrors "go.temporal.io/server/common/serviceerror"
	"go.temporal.io/server/common/testing/protorequire"
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

// answer is one answer of history's admission, for a delivery of the held activity by its stamp.
type answer struct {
	stamp    int32
	response any
	err      error
}

// The held dispatch is released only by Release, and what admission decided is the release's result.
// Answers for another run, the rejection of another stamp, another method's answer, and an answer
// given before the release decide nothing of the released delivery.
//
// A rejection is read only while every answer so far is accounted for. The server answers a delivery
// as obsolete for two reasons: its stamp is no longer the activity's, which commits nothing, or the
// activity has already started, which a redelivery meets after an admission that committed and whose
// answer was lost (chasm/lib/activity/activity.go, HandleStarted). So once an answer for the activity
// decided nothing that can be told, a later obsolete answer is no rejection: the release has no
// decision. An admission the server then reports, as it does for the retry of the same request, is
// still a committed one.
func TestReleaseReportsAdmissionsDecisionForTheHeldDelivery(t *testing.T) {
	key := chasm.ExecutionKey{NamespaceID: string(scope), BusinessID: "activity", RunID: "run"}
	obsolete := serviceerrors.NewObsoleteMatchingTask("activity attempt stamp mismatch")
	lost := serviceerror.NewUnavailable("lost")
	admitted := func(attempt int32) any { return &historyservice.RecordActivityTaskStartedResponse{Attempt: attempt} }
	decision := func(delivery string, decision testpilotspb.DeliveryAdmissionDecision, attempt int32) *testpilotspb.DeliveryAdmission {
		return &testpilotspb.DeliveryAdmission{ActivityId: "activity", ActivityRunId: "run", DeliveryId: delivery, Decision: decision, Attempt: attempt}
	}
	for _, test := range []struct {
		name    string
		answers []answer
		want    *testpilotspb.DeliveryAdmission
	}{
		{"rejected", []answer{{7, nil, obsolete}}, decision("7", testpilotspb.DELIVERY_ADMISSION_DECISION_REJECTED, 0)},
		{"admitted", []answer{{7, admitted(1), nil}}, decision("7", testpilotspb.DELIVERY_ADMISSION_DECISION_ADMITTED, 1)},
		// The hold lets no dispatch of the activity pass, so an attempt admission commits for any
		// delivery of it is one the release let through, and is reported under that delivery.
		{"admitted by another delivery", []answer{{9, admitted(2), nil}}, decision("9", testpilotspb.DELIVERY_ADMISSION_DECISION_ADMITTED, 2)},
		{"a lost answer, then the admission it committed", []answer{{7, nil, lost}, {7, admitted(1), nil}},
			decision("7", testpilotspb.DELIVERY_ADMISSION_DECISION_ADMITTED, 1)},
		{"a lost answer, then the redelivery refused", []answer{{7, nil, lost}, {7, nil, obsolete}}, nil},
		{"a lost answer of another delivery, then the held one refused", []answer{{9, nil, lost}, {7, nil, obsolete}}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, d := newServer(t)
			held, err := d.Hold("activity")
			require.NoError(t, err)
			dispatched := make(chan error, 1)
			go func() { dispatched <- s.dispatch(t.Context(), testhooks.ActivityDelivery{Execution: key, Stamp: 7}) }()
			_, arrived := held.Delivery()
			require.False(t, arrived)
			require.NoError(t, held.Await(t.Context()))
			delivery, arrived := held.Delivery()
			require.True(t, arrived)
			require.Equal(t, testhooks.ActivityDelivery{Execution: key, Stamp: 7}, delivery)
			s.answer(started(t, key, 7), nil, obsolete)
			select {
			case err := <-dispatched:
				t.Fatalf("the dispatch crossed the hold before release: %v", err)
			default:
			}
			ctx, expire := context.WithCancel(t.Context())
			defer expire()
			admission, err := held.Release(ctx, func(polling context.Context) error {
				require.NoError(t, <-dispatched)
				crossed := key
				crossed.RunID = "other"
				s.answer(started(t, crossed, 7), nil, obsolete)
				s.answer(started(t, crossed, 7), nil, lost)
				s.answer(started(t, key, 8), nil, obsolete)
				s.answer(&historyservice.RecordActivityTaskHeartbeatRequest{}, nil, lost)
				for _, a := range test.answers {
					s.answer(started(t, key, a.stamp), a.response, a.err)
				}
				if test.want == nil {
					// The instruction's deadline is what ends a release that has no decision.
					expire()
				}
				<-polling.Done()
				return polling.Err()
			})
			if test.want == nil {
				require.ErrorIs(t, err, context.Canceled)
				require.Nil(t, admission)
				return
			}
			require.NoError(t, err)
			protorequire.ProtoEqual(t, test.want, admission)
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
	require.NoError(t, held.Await(t.Context()))
	failed := errors.New("poll failed")
	_, err = held.Release(t.Context(), func(context.Context) error { return failed })
	require.ErrorIs(t, err, failed)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	other, err := d.Hold("other")
	require.NoError(t, err)
	go func() {
		_ = s.dispatch(t.Context(), testhooks.ActivityDelivery{Execution: chasm.ExecutionKey{BusinessID: "other"}, Stamp: 1})
	}()
	require.NoError(t, other.Await(t.Context()))
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
	require.NoError(t, held.Await(t.Context()))
	held.Close()
	require.ErrorIs(t, <-dispatched, ErrClosed)
	again, err := d.Hold("activity")
	require.NoError(t, err)
	d.Close()
	_, err = d.Hold("next")
	require.ErrorIs(t, err, ErrClosed)
	require.ErrorIs(t, again.gate.Arrive(t.Context(), "activity"), ErrClosed)
}
