package act

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

const (
	DefaultWorkerRPCBudget    uint64 = 4096
	DefaultWorkerCaptureBytes uint64 = 1 << 30
	workerMaxPayloadSize             = 128 << 20
	workerMaxStatusBytes      uint64 = 32 << 20
	workerCaptureMetadata     uint64 = 8192
)

var (
	ErrWorkerCaptureInvocationLimit = errors.New("WorkerCaptureInvocationLimit")
	ErrWorkerCaptureByteLimit       = errors.New("WorkerCaptureByteLimit")
)

type WorkerRPCStage uint8

const (
	WorkerRPCStarted WorkerRPCStage = iota + 1
	WorkerRPCCompleted
)

// WorkerRPC carries one immutable actual transport capture, independently of program policy.
type WorkerRPC struct {
	Invocation     uint64
	Sequence       uint64
	Stage          WorkerRPCStage
	Method         string
	Request        *anypb.Any
	Response       *anypb.Any
	ServerError    *realizationv1.Error
	CallError      *anypb.Any
	TransportError string
	Sent           bool
	Answered       bool
}

// WorkerBinding owns one Driver's executor and its mechanical transport observer.
type WorkerBinding struct {
	Follow  func(context.Context, client.Client, []byte) error
	Observe func(context.Context, WorkerRPC) error
	Close   func() error
}

func (o Options) validateWorker() error {
	if o.Program == nil {
		return nil
	}
	if o.ValidateProgram == nil || o.BindWorker == nil {
		return errors.New("programmed executor is not configured")
	}
	if err := o.ValidateProgram(o.Program); err != nil {
		return err
	}
	return o.withDefaults().validateWorkerLimits()
}

func (o Options) validateWorkerLimits() error {
	if o.WorkerRPCBudget == 0 || o.WorkerRPCBudget > math.MaxUint64/2 {
		return fmt.Errorf("%w: positive invocation budget required", ErrWorkerCaptureInvocationLimit)
	}
	minimum := minimumWorkerCaptureBytes()
	if o.WorkerCaptureBytes < minimum {
		return fmt.Errorf("%w: %d bytes cannot reserve one maximum worker invocation (%d required)", ErrWorkerCaptureByteLimit, o.WorkerCaptureBytes, minimum)
	}
	return nil
}

func minimumWorkerCaptureBytes() uint64 {
	var maximum uint64
	for _, request := range []proto.Message{
		new(workflowservice.PollActivityTaskQueueRequest), new(workflowservice.RecordActivityTaskHeartbeatRequest),
		new(workflowservice.RespondActivityTaskCompletedRequest), new(workflowservice.RespondActivityTaskFailedRequest), new(workflowservice.RespondActivityTaskCanceledRequest),
	} {
		typeURL := "type.googleapis.com/" + string(request.ProtoReflect().Descriptor().FullName())
		encoded := uint64(protowire.SizeTag(1) + protowire.SizeBytes(len(typeURL)) + protowire.SizeTag(2) + protowire.SizeBytes(workerMaxPayloadSize))
		maximum = max(maximum, encoded)
	}
	return 2*maximum + workerMaxPayloadSize + workerMaxStatusBytes + workerCaptureMetadata
}

type workerCapture struct {
	mu          sync.Mutex
	budget      uint64
	byteLimit   uint64
	occupied    uint64
	invocations uint64
	sequence    uint64
	calls       []WorkerRPC
	observe     func(context.Context, WorkerRPC) error
	active      uint64
	sealed      bool
	drained     chan struct{}
}

type workerInvocation struct {
	started     WorkerRPC
	reservation uint64
}

func newWorkerCapture(opts Options, observe func(context.Context, WorkerRPC) error) (*workerCapture, error) {
	if err := opts.validateWorkerLimits(); err != nil {
		return nil, err
	}
	return &workerCapture{budget: opts.WorkerRPCBudget, byteLimit: opts.WorkerCaptureBytes, observe: observe, drained: make(chan struct{})}, nil
}

func (c *workerCapture) enter() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sealed {
		return false
	}
	c.active++
	return true
}

func (c *workerCapture) leave() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active--
	if c.sealed && c.active == 0 {
		close(c.drained)
	}
}

func (c *workerCapture) join() {
	c.mu.Lock()
	if !c.sealed {
		c.sealed = true
		if c.active == 0 {
			close(c.drained)
		}
	}
	c.mu.Unlock()
	<-c.drained
}

func cloneWorkerRPC(r WorkerRPC) WorkerRPC {
	r.Request = proto.CloneOf(r.Request)
	r.Response = proto.CloneOf(r.Response)
	r.ServerError = proto.CloneOf(r.ServerError)
	r.CallError = proto.CloneOf(r.CallError)
	return r
}

func workerTrace(r WorkerRPC) *temporalv1.WorkerRPCTrace {
	result := &temporalv1.WorkerRPCTrace{Invocation: r.Invocation, Sequence: r.Sequence, Stage: uint32(r.Stage), Method: r.Method, Request: r.Request, Response: r.Response, TransportError: r.TransportError, Sent: r.Sent, Answered: r.Answered, CallError: r.CallError}
	if r.ServerError != nil {
		result.StatusCode, result.StatusMessage = r.ServerError.GetCode(), r.ServerError.GetMessage()
	}
	return result
}

func workerCaptureSize(r WorkerRPC) uint64 { return uint64(proto.Size(workerTrace(r))) + 16 }

func (c *workerCapture) publishLocked(ctx context.Context, rpc WorkerRPC) error {
	c.calls = append(c.calls, cloneWorkerRPC(rpc))
	return c.observe(ctx, cloneWorkerRPC(rpc))
}

func (c *workerCapture) begin(ctx context.Context, method string, request proto.Message) (*workerInvocation, error) {
	packed, err := pack(request)
	if err != nil {
		return nil, err
	}
	if len(packed.GetValue()) > workerMaxPayloadSize {
		return nil, fmt.Errorf("%w: worker request exceeds SDK message allowance", ErrWorkerCaptureByteLimit)
	}
	requestBytes := uint64(proto.Size(packed))
	reservation := 2*requestBytes + workerMaxPayloadSize + workerMaxStatusBytes + workerCaptureMetadata
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.invocations == c.budget || c.invocations == math.MaxUint64 {
		return nil, ErrWorkerCaptureInvocationLimit
	}
	if reservation > c.byteLimit-c.occupied {
		return nil, ErrWorkerCaptureByteLimit
	}
	c.invocations++
	c.occupied += reservation
	c.sequence++
	invocation := &workerInvocation{started: WorkerRPC{Invocation: c.invocations, Sequence: c.sequence, Stage: WorkerRPCStarted, Method: method, Request: packed}, reservation: reservation}
	err = c.publishLocked(ctx, invocation.started)
	if err != nil {
		c.occupied -= reservation - workerCaptureSize(invocation.started)
	}
	return invocation, err
}

func (c *workerCapture) complete(ctx context.Context, invocation *workerInvocation, response proto.Message, reached *progress, actualErr error) error {
	rpc := cloneWorkerRPC(invocation.started)
	rpc.Stage, rpc.Sent, rpc.Answered = WorkerRPCCompleted, reached.sent.Load(), reached.answered.Load()
	var captureErr error
	if proto.Size(response) > workerMaxPayloadSize {
		captureErr = fmt.Errorf("%w: actual worker response exceeds SDK message allowance", ErrWorkerCaptureByteLimit)
		rpc.TransportError = captureErr.Error()
	} else {
		var err error
		rpc.Response, err = pack(response)
		if err != nil {
			captureErr = err
			rpc.TransportError = err.Error()
		}
	}
	if actualErr != nil {
		var err error
		rpc.CallError, err = pack(status.Convert(actualErr).Proto())
		if err != nil {
			captureErr = errors.Join(captureErr, err)
		}
		code := status.Code(actualErr)
		if rpc.Answered && code != codes.Internal && code != codes.Canceled && code != codes.DeadlineExceeded {
			if code > codes.OK && code <= codes.Unauthenticated {
				rpc.ServerError = &realizationv1.Error{Code: int32(code), Message: status.Convert(actualErr).Message()}
			} else {
				rpc.TransportError = actualErr.Error()
			}
		} else {
			rpc.TransportError = actualErr.Error()
		}
	} else if !rpc.Sent || !rpc.Answered {
		captureErr = errors.New("worker RPC returned without actual server completion")
		rpc.TransportError = captureErr.Error()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sequence++
	rpc.Sequence = c.sequence
	used := workerCaptureSize(invocation.started) + workerCaptureSize(rpc)
	if used > invocation.reservation {
		captureErr = fmt.Errorf("%w: actual worker completion exceeds reserved capture", ErrWorkerCaptureByteLimit)
		rpc.Response, rpc.ServerError, rpc.CallError = nil, nil, nil
		rpc.TransportError = captureErr.Error()
		used = workerCaptureSize(invocation.started) + workerCaptureSize(rpc)
	}
	if used > invocation.reservation {
		c.occupied -= invocation.reservation - workerCaptureSize(invocation.started)
		return captureErr
	}
	c.occupied -= invocation.reservation - used
	return errors.Join(captureErr, c.publishLocked(ctx, rpc))
}

func capturedWorkerMethod(method string) bool {
	switch method {
	case workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName,
		workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName:
		return true
	default:
		return false
	}
}

func (d *Driver) workerFault(err error) {
	if err == nil {
		return
	}
	d.workerMu.Lock()
	d.workerErr = errors.Join(d.workerErr, err)
	d.workerMu.Unlock()
	d.workerCancel()
}

func (d *Driver) captureWorker(ctx context.Context, method string, request, response any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if !capturedWorkerMethod(method) {
		return invoke(ctx, method, request, response, conn, opts...)
	}
	if !d.workerCapture.enter() {
		if err := d.workerContext.Err(); err != nil {
			return err
		}
		return errors.New("worker transport capture is closed")
	}
	defer d.workerCapture.leave()
	var callCtx context.Context
	var cancel context.CancelFunc
	if deadline, ok := d.workerContext.Deadline(); ok {
		callCtx, cancel = context.WithDeadline(ctx, deadline)
	} else {
		callCtx, cancel = context.WithCancel(ctx)
	}
	joined := make(chan struct{})
	stop := context.AfterFunc(d.workerContext, func() { cancel(); close(joined) })
	defer func() {
		cancel()
		if !stop() {
			<-joined
		}
	}()
	ctx = callCtx
	if err := ctx.Err(); err != nil {
		return err
	}
	input, inputOK := request.(proto.Message)
	output, outputOK := response.(proto.Message)
	if !inputOK || !outputOK {
		err := errors.New("worker transport requires protobuf messages")
		d.workerFault(err)
		return err
	}
	invocation, err := d.workerCapture.begin(ctx, method, input)
	if err != nil {
		d.workerFault(err)
		return err
	}
	reached := new(progress)
	actualErr := invoke(context.WithValue(ctx, progressKey{}, reached), method, request, response, conn, opts...)
	d.workerFault(d.workerCapture.complete(ctx, invocation, output, reached, actualErr))
	return actualErr
}

func (d *Driver) workerFaultEvidence(controller *anypb.Any) (*anypb.Any, error) {
	if d.workerCapture == nil {
		return proto.CloneOf(controller), nil
	}
	d.workerCapture.mu.Lock()
	defer d.workerCapture.mu.Unlock()
	if len(d.workerCapture.calls) == 0 {
		return proto.CloneOf(controller), nil
	}
	fault := &temporalv1.WorkerTransportFault{Controller: proto.CloneOf(controller)}
	for _, rpc := range d.workerCapture.calls {
		fault.Calls = append(fault.Calls, workerTrace(cloneWorkerRPC(rpc)))
	}
	return pack(fault)
}

type workerEvidenceError struct {
	cause    error
	evidence *anypb.Any
}

func (e *workerEvidenceError) Error() string        { return e.cause.Error() }
func (e *workerEvidenceError) Unwrap() error        { return e.cause }
func (e *workerEvidenceError) Evidence() *anypb.Any { return proto.CloneOf(e.evidence) }
