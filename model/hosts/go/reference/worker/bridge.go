package worker

import (
	"context"
	"errors"

	genericworker "github.com/temporalio/umpire/hosts/go/worker"
	"go.temporal.io/api/workflowservice/v1"
)

// RPCStage distinguishes an actual invocation's registration from its completion.
type RPCStage = genericworker.Stage

const (
	RPCStarted   = genericworker.Started
	RPCCompleted = genericworker.Completed
)

// RPC carries one immutable actual transport snapshot from the Driver capture owner.
type RPC = genericworker.Capture

// Bridge correlates actual transport invocations with one Driver's assignments.
// It has no policy rows, heartbeat history or Model state.
type Bridge struct {
	coordinator *genericworker.Coordinator[TaskIdentity]
	initErr     error
	unavailable chan struct{}
}

// Assignment is a unique actual poll delivery, retained through one terminal choice.
type Assignment struct {
	assignment *genericworker.Assignment[TaskIdentity]
}

// ActualCompletion exposes copies of the correlated actual completion.
type ActualCompletion struct{ completion genericworker.Completion }

func (c ActualCompletion) RPC() RPC { return c.completion.Capture() }

type HeartbeatWait struct{ wait *genericworker.Wait }
type TerminalWait struct{ wait *genericworker.Wait }

func NewBridge(ctx context.Context) *Bridge {
	coordinator, err := genericworker.NewCoordinator(ctx, activityProtocol())
	if err != nil {
		unavailable := make(chan struct{})
		close(unavailable)
		return &Bridge{initErr: err, unavailable: unavailable}
	}
	return &Bridge{coordinator: coordinator}
}
func (b *Bridge) Observe(ctx context.Context, rpc RPC) error {
	if b.initErr != nil {
		return b.initErr
	}
	return b.coordinator.Observe(ctx, rpc)
}
func (b *Bridge) Err() error {
	if b.initErr != nil {
		return b.initErr
	}
	return b.coordinator.Err()
}
func (b *Bridge) Failed() <-chan struct{} {
	if b.initErr != nil {
		return b.unavailable
	}
	return b.coordinator.Failed()
}

// Done closes when the bridge owner explicitly closes its subscriptions.
func (b *Bridge) Done() <-chan struct{} {
	if b.initErr != nil {
		return b.unavailable
	}
	return b.coordinator.Done()
}

// Close wakes waits; normal Run cancellation does not manufacture a worker fault.
func (b *Bridge) Close() error {
	if b.initErr != nil {
		return b.initErr
	}
	return b.coordinator.Close()
}
func (b *Bridge) Bind(identity TaskIdentity) (*Assignment, error) {
	if b.initErr != nil {
		return nil, b.initErr
	}
	a, err := b.coordinator.Bind(genericworker.Address{Scope: identity.Namespace, Handle: string(identity.Token)}, identity)
	if err != nil {
		return nil, err
	}
	return &Assignment{assignment: a}, nil
}
func (a *Assignment) HeartbeatContext(ctx context.Context) context.Context {
	return a.assignment.Context(ctx)
}
func (a *Assignment) BeginHeartbeat() (*HeartbeatWait, error) {
	w, err := a.assignment.Begin(workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName)
	if err != nil {
		return nil, err
	}
	return &HeartbeatWait{wait: w}, nil
}
func (a *Assignment) BeginTerminal(method string) (*TerminalWait, error) {
	switch method {
	case workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName, workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName, workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName:
	default:
		return nil, errors.New("unsupported terminal worker method")
	}
	w, err := a.assignment.Begin(method)
	if err != nil {
		return nil, err
	}
	return &TerminalWait{wait: w}, nil
}
func (w *HeartbeatWait) Wait(ctx context.Context) (ActualCompletion, error) {
	c, err := w.wait.Wait(ctx)
	return ActualCompletion{completion: c}, err
}
func (w *TerminalWait) Wait(ctx context.Context) (ActualCompletion, error) {
	c, err := w.wait.Wait(ctx)
	return ActualCompletion{completion: c}, err
}
