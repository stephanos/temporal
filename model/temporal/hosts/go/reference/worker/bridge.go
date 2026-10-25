package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
)

// RPCStage distinguishes an actual invocation's registration from its completion.
type RPCStage uint8

const (
	RPCStarted RPCStage = iota + 1
	RPCCompleted
)

// RPC carries one immutable actual transport snapshot from the Driver capture owner.
type RPC struct {
	Invocation, Sequence uint64
	Stage                RPCStage
	Method               string
	Request, Response    *anypb.Any
	CallError            *anypb.Any
	ServerError          *realizationv1.Error
	TransportError       string
	Sent, Answered       bool
}

func (r RPC) clone() RPC {
	if r.Request != nil {
		r.Request = proto.CloneOf(r.Request)
	}
	if r.Response != nil {
		r.Response = proto.CloneOf(r.Response)
	}
	if r.CallError != nil {
		r.CallError = proto.CloneOf(r.CallError)
	}
	if r.ServerError != nil {
		r.ServerError = proto.CloneOf(r.ServerError)
	}
	return r
}

// TaskIdentity contains public poll/SDK assignment values, without decoding the token.
type TaskIdentity struct {
	Namespace, TaskQueue, ActivityID, ActivityRunID   string
	WorkflowID, WorkflowRunID, ActivityType, Scenario string
	Attempt                                           int32
	Token                                             []byte
}

func (i TaskIdentity) same(other TaskIdentity) bool {
	return i.Namespace == other.Namespace && i.TaskQueue == other.TaskQueue &&
		i.ActivityID == other.ActivityID && i.ActivityRunID == other.ActivityRunID &&
		i.WorkflowID == other.WorkflowID && i.WorkflowRunID == other.WorkflowRunID &&
		i.ActivityType == other.ActivityType && i.Scenario == other.Scenario &&
		i.Attempt == other.Attempt && bytes.Equal(i.Token, other.Token)
}

// IdentityFromPoll reads assignment identity only from the actual poll request/reply.
func IdentityFromPoll(namespace, taskQueue string, task *workflowservice.PollActivityTaskQueueResponse) TaskIdentity {
	return TaskIdentity{Namespace: namespace, TaskQueue: taskQueue,
		ActivityID: task.GetActivityId(), ActivityRunID: task.GetActivityRunId(),
		WorkflowID: task.GetWorkflowExecution().GetWorkflowId(), WorkflowRunID: task.GetWorkflowExecution().GetRunId(),
		ActivityType: task.GetActivityType().GetName(), Scenario: Scenario(task.GetHeader().GetFields()[ScenarioHeader]),
		Attempt: task.GetAttempt(), Token: bytes.Clone(task.GetTaskToken())}
}

type assignmentMarker struct{}
type assignmentKey struct{ namespace, token string }

type invocation struct {
	start      RPC
	assignment *Assignment
	operation  *operation
	completed  bool
}

type operation struct {
	method   string
	done     chan struct{}
	result   *RPC
	inFlight uint64
	waiting  bool
}

// Bridge correlates actual transport invocations with one Driver's assignments.
// It has no policy rows, heartbeat history or Model state.
type Bridge struct {
	ctx           context.Context
	mu            sync.Mutex
	seenSequences map[uint64]struct{}
	assignments   map[assignmentKey]*Assignment
	invocations   map[uint64]*invocation
	failed        chan struct{}
	closed        chan struct{}
	err           error
	isClosed      bool
}

// Assignment is a unique actual poll delivery, retained through one terminal choice.
type Assignment struct {
	bridge                 *Bridge
	identity               TaskIdentity
	claimed, retired       bool
	lastInvocationSequence uint64
	current                *operation
}

// ActualCompletion exposes copies of the correlated actual completion.
type ActualCompletion struct{ rpc RPC }

func (c ActualCompletion) RPC() RPC { return c.rpc.clone() }

type HeartbeatWait struct {
	assignment *Assignment
	operation  *operation
}
type TerminalWait struct {
	assignment *Assignment
	operation  *operation
}

func NewBridge(ctx context.Context) *Bridge {
	return &Bridge{ctx: ctx, assignments: make(map[assignmentKey]*Assignment), invocations: make(map[uint64]*invocation), seenSequences: make(map[uint64]struct{}), failed: make(chan struct{}), closed: make(chan struct{})}
}

func (b *Bridge) fail(err error) error {
	if b.err == nil {
		b.err = err
		close(b.failed)
	}
	return b.err
}

func (b *Bridge) Err() error              { b.mu.Lock(); defer b.mu.Unlock(); return b.err }
func (b *Bridge) Failed() <-chan struct{} { return b.failed }

// Done closes when the bridge owner explicitly closes its subscriptions.
func (b *Bridge) Done() <-chan struct{} { return b.closed }

// Observe never waits for a handler, another invocation or SDK cancellation.
func (b *Bridge) Observe(ctx context.Context, rpc RPC) error {
	rpc = rpc.clone()
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return b.err
	}
	if b.isClosed {
		return errors.New("worker bridge is closed")
	}
	if rpc.Invocation == 0 || rpc.Sequence == 0 || rpc.Request == nil {
		return b.fail(errors.New("worker RPC has missing or reordered invocation identity"))
	}
	if _, exists := b.seenSequences[rpc.Sequence]; exists {
		return b.fail(errors.New("duplicate worker RPC sequence identity"))
	}
	b.seenSequences[rpc.Sequence] = struct{}{}
	switch rpc.Stage {
	case RPCStarted:
		return b.observeStart(ctx, rpc)
	case RPCCompleted:
		return b.observeCompletion(rpc)
	default:
		return b.fail(errors.New("unknown worker RPC stage"))
	}
}

func (b *Bridge) observeStart(ctx context.Context, rpc RPC) error {
	if _, exists := b.invocations[rpc.Invocation]; exists {
		return b.fail(errors.New("duplicate worker RPC invocation"))
	}
	call := &invocation{start: rpc}
	b.invocations[rpc.Invocation] = call
	if rpc.Method == workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName {
		return nil
	}
	namespace, token, err := rpcAssignment(rpc)
	if err != nil {
		return b.fail(err)
	}
	a := b.assignments[assignmentKey{namespace, string(token)}]
	if b.ctx.Err() != nil {
		return nil
	}
	if a == nil || !a.claimed || a.retired {
		return b.fail(errors.New("worker RPC has no live claimed poll assignment"))
	}
	if rpc.Method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
		if marker, ok := ctx.Value(assignmentMarker{}).(*Assignment); !ok || marker != a || marker.bridge != b {
			return b.fail(errors.New("heartbeat context does not identify its actual poll assignment"))
		}
	} else if marker, ok := ctx.Value(assignmentMarker{}).(*Assignment); ok && marker != a {
		return b.fail(errors.New("terminal context crosses its actual poll assignment"))
	}
	if rpc.Sequence <= a.lastInvocationSequence {
		return b.fail(errors.New("worker RPC start reordered within assignment"))
	}
	op := a.current
	if op == nil || op.method != rpc.Method || op.result != nil || op.inFlight != 0 {
		return b.fail(errors.New("worker RPC has no unique matching outstanding operation"))
	}
	op.inFlight = rpc.Invocation
	call.assignment, call.operation = a, op
	return nil
}

func (b *Bridge) observeCompletion(rpc RPC) error {
	call := b.invocations[rpc.Invocation]
	if call == nil || call.completed || call.start.Method != rpc.Method || !proto.Equal(call.start.Request, rpc.Request) {
		return b.fail(errors.New("worker RPC completion is missing, duplicate or crossed"))
	}
	if rpc.Sequence <= call.start.Sequence {
		return b.fail(errors.New("worker RPC completion precedes its invocation start"))
	}
	call.completed = true
	if rpc.Method == workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName {
		return b.poll(rpc)
	}
	if call.operation == nil {
		return nil
	}
	op := call.operation
	if op.inFlight != rpc.Invocation {
		return b.fail(errors.New("worker RPC completion reordered within operation"))
	}
	if rpc.Sequence <= call.assignment.lastInvocationSequence {
		return b.fail(errors.New("worker RPC completion reordered within assignment"))
	}
	call.assignment.lastInvocationSequence = rpc.Sequence
	op.inFlight = 0
	if rpc.TransportError != "" || rpc.ServerError != nil || rpc.CallError != nil {
		return nil
	}
	if rpc.Response == nil || !rpc.Sent || !rpc.Answered {
		return b.fail(errors.New("worker RPC has no acknowledged response"))
	}
	if op.result != nil {
		return b.fail(errors.New("duplicate successful worker RPC completion"))
	}
	op.result = &rpc
	if rpc.Method != workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
		call.assignment.retired = true
	}
	close(op.done)
	return nil
}

func (b *Bridge) poll(rpc RPC) error {
	if rpc.TransportError != "" || rpc.ServerError != nil || rpc.CallError != nil {
		return nil
	}
	if rpc.Response == nil || !rpc.Sent || !rpc.Answered {
		return b.fail(errors.New("poll has no acknowledged response"))
	}
	request := new(workflowservice.PollActivityTaskQueueRequest)
	response := new(workflowservice.PollActivityTaskQueueResponse)
	if err := rpc.Request.UnmarshalTo(request); err != nil {
		return b.fail(fmt.Errorf("poll request: %w", err))
	}
	if err := rpc.Response.UnmarshalTo(response); err != nil {
		return b.fail(fmt.Errorf("poll response: %w", err))
	}
	if len(response.GetTaskToken()) == 0 {
		return nil
	}
	if response.GetWorkflowNamespace() != "" && response.GetWorkflowNamespace() != request.GetNamespace() {
		return b.fail(errors.New("poll response namespace crosses actual poll request"))
	}
	identity := IdentityFromPoll(request.GetNamespace(), request.GetTaskQueue().GetName(), response)
	if identity.Namespace == "" || identity.TaskQueue == "" || identity.ActivityType == "" || identity.Attempt <= 0 {
		return b.fail(errors.New("poll assignment identity is incomplete"))
	}
	key := assignmentKey{identity.Namespace, string(identity.Token)}
	if b.assignments[key] != nil {
		return b.fail(errors.New("duplicate or reused poll task token is ambiguous"))
	}
	b.assignments[key] = &Assignment{bridge: b, identity: identity, lastInvocationSequence: rpc.Sequence}
	return nil
}

func rpcAssignment(rpc RPC) (string, []byte, error) {
	var namespace string
	var token []byte
	var request proto.Message
	switch rpc.Method {
	case workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName:
		request = new(workflowservice.RecordActivityTaskHeartbeatRequest)
	case workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName:
		request = new(workflowservice.RespondActivityTaskCompletedRequest)
	case workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName:
		request = new(workflowservice.RespondActivityTaskFailedRequest)
	case workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName:
		request = new(workflowservice.RespondActivityTaskCanceledRequest)
	default:
		return "", nil, errors.New("unsupported worker RPC method")
	}
	if err := rpc.Request.UnmarshalTo(request); err != nil {
		return "", nil, fmt.Errorf("worker RPC request: %w", err)
	}
	fields, ok := request.(interface {
		GetNamespace() string
		GetTaskToken() []byte
	})
	if !ok {
		return "", nil, errors.New("worker request has no assignment identity")
	}
	namespace, token = fields.GetNamespace(), fields.GetTaskToken()
	if namespace == "" || len(token) == 0 {
		return "", nil, errors.New("worker request assignment identity is missing")
	}
	return namespace, token, nil
}

func (b *Bridge) Bind(identity TaskIdentity) (*Assignment, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.available(); err != nil {
		return nil, err
	}
	a := b.assignments[assignmentKey{identity.Namespace, string(identity.Token)}]
	if a == nil || a.claimed || a.retired || !a.identity.same(identity) {
		return nil, b.fail(errors.New("handler does not match one unclaimed actual poll assignment"))
	}
	a.claimed = true
	return a, nil
}

func (b *Bridge) available() error {
	if b.err != nil {
		return b.err
	}
	if b.isClosed {
		return errors.New("worker bridge is closed")
	}
	return b.ctx.Err()
}

func (a *Assignment) HeartbeatContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, assignmentMarker{}, a)
}

func (a *Assignment) begin(method string) (*operation, error) {
	b := a.bridge
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.available(); err != nil {
		return nil, err
	}
	if !a.claimed || a.retired || a.current != nil {
		return nil, b.fail(errors.New("assignment already has an outstanding or terminal operation"))
	}
	op := &operation{method: method, done: make(chan struct{})}
	a.current = op
	return op, nil
}

func (a *Assignment) BeginHeartbeat() (*HeartbeatWait, error) {
	op, err := a.begin(workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName)
	if err != nil {
		return nil, err
	}
	return &HeartbeatWait{assignment: a, operation: op}, nil
}

func (a *Assignment) BeginTerminal(method string) (*TerminalWait, error) {
	switch method {
	case workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName, workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName, workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName:
	default:
		return nil, errors.New("unsupported terminal worker method")
	}
	op, err := a.begin(method)
	if err != nil {
		return nil, err
	}
	return &TerminalWait{assignment: a, operation: op}, nil
}

func (w *HeartbeatWait) Wait(ctx context.Context) (ActualCompletion, error) {
	return w.assignment.wait(ctx, w.operation)
}
func (w *TerminalWait) Wait(ctx context.Context) (ActualCompletion, error) {
	return w.assignment.wait(ctx, w.operation)
}

func (a *Assignment) wait(ctx context.Context, op *operation) (ActualCompletion, error) {
	b := a.bridge
	b.mu.Lock()
	if op.waiting {
		err := b.fail(errors.New("operation completion was awaited more than once"))
		b.mu.Unlock()
		return ActualCompletion{}, err
	}
	op.waiting = true
	b.mu.Unlock()
	select {
	case <-op.done:
	case <-ctx.Done():
	case <-b.ctx.Done():
	case <-b.failed:
	case <-b.closed:
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.available(); err != nil {
		return ActualCompletion{}, err
	}
	if err := ctx.Err(); err != nil {
		return ActualCompletion{}, err
	}
	if op.result == nil {
		return ActualCompletion{}, errors.New("worker operation has no successful completion")
	}
	if a.current != op {
		return ActualCompletion{}, b.fail(errors.New("operation is no longer current for assignment"))
	}
	a.current = nil
	return ActualCompletion{rpc: op.result.clone()}, nil
}

// Close wakes waits; normal Run cancellation does not manufacture a worker fault.
func (b *Bridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.isClosed {
		b.isClosed = true
		close(b.closed)
	}
	if b.err != nil {
		return b.err
	}
	if b.ctx.Err() == nil {
		for _, a := range b.assignments {
			if a.current != nil && !a.retired {
				return b.fail(errors.New("worker bridge closed with an outstanding assignment operation"))
			}
		}
	}
	return nil
}
