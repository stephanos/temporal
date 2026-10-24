package act

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"umpire.local/model/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestPollHoldContainsAlreadyWaitingAndConcurrentPolls(t *testing.T) {
	h := newPollHold()
	var sent atomic.Int32
	done := make(chan error, 3)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	for range 3 {
		go func() {
			done <- h.intercept(ctx, workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, nil, nil, nil,
				func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
					sent.Add(1)
					return nil
				})
		}()
	}
	require.Never(t, func() bool { return sent.Load() != 0 }, 20*time.Millisecond, time.Millisecond)
	earlier, err := h.prepare(true)
	require.NoError(t, err)
	later, err := h.prepare(true)
	require.NoError(t, err)
	h.acknowledged(earlier)
	require.Zero(t, sent.Load())
	h.acknowledged(later)
	for range 3 {
		require.NoError(t, <-done)
	}
	require.EqualValues(t, 3, sent.Load())
	_, err = h.prepare(true)
	require.ErrorContains(t, err, "cannot establish")
}

func TestClosingPollHoldCancelsHeldPolls(t *testing.T) {
	h := newPollHold()
	done := make(chan error, 1)
	go func() {
		done <- h.intercept(t.Context(), workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, nil, nil, nil,
			func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
				t.Error("closed poll reached server")
				return nil
			})
	}()
	h.close()
	require.Error(t, <-done)
}

func TestHoldReleasesOnlyOnServerAcknowledgment(t *testing.T) {
	for _, answered := range []bool{false, true} {
		t.Run(map[bool]string{false: "local-timeout", true: "server-status"}[answered], func(t *testing.T) {
			address := unreachable(t)
			if answered {
				address = newFakeServer(t, func(string) ([]byte, error) { return nil, status.Error(codes.NotFound, "acknowledged") }).address
			}
			d := testDriver(t, address, Options{RPCTimeout: 50 * time.Millisecond})
			d.polls = newPollHold()
			x := start(t, d, 1, packed(t, &temporalv1.TemporalInstruction{Request: packed(t, cancelRequest), HoldWorkerPolls: true}))
			e := next(t, x)
			d.polls.mu.Lock()
			held := d.polls.held
			d.polls.mu.Unlock()
			require.Equal(t, !answered, held)
			if answered {
				observation(t, e)
				require.NoError(t, x.Respond(t.Context(), e.ID, explained(false)))
			} else {
				require.Equal(t, runner.EventFault, e.Kind)
			}
		})
	}
}

func TestPollHoldCloseCancelsAnOutstandingPoll(t *testing.T) {
	h := newPollHold()
	_, err := h.prepare(false)
	require.NoError(t, err)
	entered := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- h.intercept(t.Context(), workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, nil, nil, nil,
			func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
				close(entered)
				<-ctx.Done()
				return ctx.Err()
			})
	}()
	<-entered
	h.close()
	require.ErrorIs(t, <-done, context.Canceled)
}
