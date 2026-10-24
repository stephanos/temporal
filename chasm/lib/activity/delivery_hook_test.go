//go:build test_dep

package activity

import (
	"context"
	"sync"
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
	arrive, held, release := holdActivityDelivery(t, testhooks.ActivityDelivery{Execution: key, Stamp: 7})
	hooks := testhooks.NewTestHooks()
	t.Cleanup(testhooks.Set(hooks, testhooks.ActivityDispatch, arrive, namespace.ID(key.NamespaceID)))
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
	select {
	case <-held:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	select {
	case <-sent:
		t.Fatal("matching received the held delivery")
	default:
	}
	attempt.Stamp = 8
	release()
	require.NoError(t, <-done)
	require.Equal(t, int32(7), <-sent)
	select {
	case <-sent:
		t.Fatal("delivery was sent more than once")
	default:
	}
}

func holdActivityDelivery(t *testing.T, target testhooks.ActivityDelivery) (func(context.Context, testhooks.ActivityDelivery) error, <-chan struct{}, func()) {
	t.Helper()
	held, released := make(chan struct{}), make(chan struct{})
	var arriveOnce, releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(released) }) }
	t.Cleanup(release)
	arrive := func(ctx context.Context, delivery testhooks.ActivityDelivery) error {
		if delivery != target {
			return nil
		}
		arriveOnce.Do(func() { close(held) })
		select {
		case <-released:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return arrive, held, release
}
