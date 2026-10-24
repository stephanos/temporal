package act

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
)

type pollHold struct {
	mu         sync.Mutex
	held       bool
	timeout    time.Duration
	generation uint64
	changed    chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
}

func newPollHold() *pollHold {
	ctx, cancel := context.WithCancel(context.Background())
	return &pollHold{held: true, changed: make(chan struct{}), ctx: ctx, cancel: cancel}
}

func (h *pollHold) prepare(hold bool) (uint64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if hold && !h.held {
		return 0, errors.New("cannot establish worker poll hold after polls were released")
	}
	h.generation++
	if !hold && h.held {
		h.held = false
		close(h.changed)
	}
	return h.generation, nil
}

func (h *pollHold) acknowledged(generation uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.held && generation == h.generation {
		h.held = false
		close(h.changed)
	}
}

func (h *pollHold) close() { h.cancel() }

func (h *pollHold) intercept(ctx context.Context, method string, req, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if method != workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName {
		return invoke(ctx, method, req, reply, conn, opts...)
	}
	h.mu.Lock()
	held, changed := h.held, h.changed
	h.mu.Unlock()
	if held {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.ctx.Done():
			return h.ctx.Err()
		case <-changed:
		}
	}
	pollCtx, cancel := context.WithCancel(ctx)
	if h.timeout > 0 {
		cancel()
		pollCtx, cancel = context.WithTimeout(ctx, h.timeout)
	}
	stop := context.AfterFunc(h.ctx, cancel)
	defer cancel()
	defer stop()
	return invoke(pollCtx, method, req, reply, conn, opts...)
}
