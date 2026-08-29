package nexus

import (
	"context"
	"sync"
	"time"

	nexussdk "github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	callerWorkflowName = "umpire.temporal.nexus.caller-closure"
	nexusServiceName   = "umpire-caller-closure"
	nexusOperationName = "wait-for-caller-closure"
)

type callerWorkflowInput struct {
	EndpointName      string
	OperationIdentity string
}

func callerWorkflow(ctx workflow.Context, input callerWorkflowInput) error {
	client := workflow.NewNexusClient(input.EndpointName, nexusServiceName)
	future := client.ExecuteOperation(
		ctx,
		nexusOperationName,
		input.OperationIdentity,
		workflow.NexusOperationOptions{
			ScheduleToCloseTimeout: 30 * time.Second,
			CancellationType:       workflow.NexusOperationCancellationTypeWaitRequested,
		},
	)
	return future.Get(ctx, nil)
}

type callerClosureOperation struct {
	nexussdk.UnimplementedOperation[string, string]

	identity string
	started  chan struct{}
	canceled chan struct{}

	mu          sync.Mutex
	startCount  uint64
	cancelCount uint64
}

func newCallerClosureOperation(identity string) *callerClosureOperation {
	return &callerClosureOperation{
		identity: identity,
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
	}
}

func (*callerClosureOperation) Name() string { return nexusOperationName }

func (o *callerClosureOperation) Start(
	_ context.Context,
	input string,
	_ nexussdk.StartOperationOptions,
) (nexussdk.HandlerStartOperationResult[string], error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if input != o.identity {
		return nil, nonRetryableHandlerError(nexussdk.HandlerErrorTypeBadRequest, "unsupported input")
	}
	if o.startCount != 0 {
		return nil, nonRetryableHandlerError(nexussdk.HandlerErrorTypeConflict, "duplicate start")
	}
	o.startCount = 1
	close(o.started)
	return &nexussdk.HandlerStartOperationResultAsync{OperationToken: o.identity}, nil
}

func (o *callerClosureOperation) Cancel(
	_ context.Context,
	token string,
	_ nexussdk.CancelOperationOptions,
) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if token != o.identity {
		return nonRetryableHandlerError(nexussdk.HandlerErrorTypeBadRequest, "unsupported token")
	}
	if o.cancelCount != 0 {
		return nonRetryableHandlerError(nexussdk.HandlerErrorTypeConflict, "duplicate cancel")
	}
	o.cancelCount = 1
	close(o.canceled)
	return nil
}

func (o *callerClosureOperation) counts() (uint64, uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.startCount, o.cancelCount
}

func nonRetryableHandlerError(
	kind nexussdk.HandlerErrorType,
	message string,
) error {
	return &nexussdk.HandlerError{
		Type: kind, Message: message,
		RetryBehavior: nexussdk.HandlerErrorRetryBehaviorNonRetryable,
	}
}

type callerClosureRegistration struct {
	operation *callerClosureOperation
}

func (r callerClosureRegistration) Register(registry worker.Registry) {
	service := nexussdk.NewService(nexusServiceName)
	service.MustRegister(&guardedCallerClosureOperation{delegate: r.operation})
	registry.RegisterWorkflowWithOptions(
		callerWorkflow,
		workflow.RegisterOptions{Name: callerWorkflowName},
	)
	registry.RegisterNexusService(service)
}

type guardedCallerClosureOperation struct {
	nexussdk.UnimplementedOperation[string, string]
	delegate nexussdk.Operation[string, string]
}

func (o *guardedCallerClosureOperation) Name() string { return o.delegate.Name() }

func (o *guardedCallerClosureOperation) Start(
	ctx context.Context,
	input string,
	options nexussdk.StartOperationOptions,
) (result nexussdk.HandlerStartOperationResult[string], err error) {
	defer func() {
		if recover() != nil {
			result = nil
			err = nonRetryableHandlerError(nexussdk.HandlerErrorTypeInternal, "handler failed")
		}
	}()
	return o.delegate.Start(ctx, input, options)
}

func (o *guardedCallerClosureOperation) Cancel(
	ctx context.Context,
	token string,
	options nexussdk.CancelOperationOptions,
) (err error) {
	defer func() {
		if recover() != nil {
			err = nonRetryableHandlerError(nexussdk.HandlerErrorTypeInternal, "handler failed")
		}
	}()
	return o.delegate.Cancel(ctx, token, options)
}
