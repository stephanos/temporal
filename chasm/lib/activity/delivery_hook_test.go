//go:build test_dep

package activity

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/api/matchingservicemock/v1"
	"go.temporal.io/server/chasm"
	"go.temporal.io/server/chasm/lib/activity/gen/activitypb/v1"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/testing/testhooks"
	"go.temporal.io/server/common/testing/testpilot/temporal/control"
	"go.uber.org/mock/gomock"
)

func TestDispatchHoldPreservesTheValidatedMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	controller := gomock.NewController(t)
	engine := chasm.NewMockEngine(controller)
	matching := matchingservicemock.NewMockMatchingServiceClient(controller)
	key := chasm.ExecutionKey{NamespaceID: "namespace", BusinessID: "activity", RunID: "run"}
	ref := chasm.NewComponentRef[*Activity](key)
	mutable := &chasm.MockMutableContext{MockContext: chasm.MockContext{
		HandleRef: func(chasm.Component) ([]byte, error) { return []byte("reference"), nil },
	}}
	attempt := &activitypb.ActivityAttemptState{Stamp: 7}
	a := &Activity{
		ActivityState: &activitypb.ActivityState{TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"}},
		LastAttempt:   chasm.NewDataField(mutable, attempt),
	}
	engine.EXPECT().ReadComponent(gomock.Any(), ref, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ chasm.ComponentRef, read func(chasm.Context, chasm.Component) error, _ ...chasm.TransitionOption) error {
			return read(mutable, a)
		})
	gate := control.NewDeliveryGate(testhooks.ActivityDelivery{Execution: key, Stamp: 7})
	defer gate.Close()
	hooks := testhooks.NewTestHooks()
	t.Cleanup(testhooks.Set(hooks, testhooks.ActivityDispatch, gate.Arrive, namespace.ID(key.NamespaceID)))
	sent := make(chan int32, 1)
	matching.EXPECT().AddActivityTask(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *matchingservice.AddActivityTaskRequest, _ ...any) (*matchingservice.AddActivityTaskResponse, error) {
			sent <- req.GetStamp()
			return &matchingservice.AddActivityTaskResponse{}, nil
		}).AnyTimes()
	handler := newActivityDispatchTaskHandler(activityDispatchTaskHandlerOptions{MatchingClient: matching, TestHooks: hooks})
	done := make(chan error, 1)
	go func() {
		done <- handler.Execute(chasm.NewEngineContext(ctx, engine), ref, chasm.TaskAttributes{}, &activitypb.ActivityDispatchTask{Stamp: 7})
	}()
	require.NoError(t, gate.WaitHeld(ctx))
	select {
	case <-sent:
		t.Fatal("matching received the held delivery")
	default:
	}
	attempt.Stamp = 8
	require.NoError(t, gate.Release())
	require.NoError(t, <-done)
	require.Equal(t, int32(7), <-sent)
	select {
	case <-sent:
		t.Fatal("delivery was sent more than once")
	default:
	}
}
