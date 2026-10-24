package act

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/await"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	"umpire.local/model/runner"
)

func task(token string) *workflowservice.PollActivityTaskQueueResponse {
	return &workflowservice.PollActivityTaskQueueResponse{TaskToken: []byte(token), ActivityId: "activity", Attempt: 1}
}

// pollWith plays the SDK worker's poll through the gate; invoke is what the server does once the
// gate admits it.
func pollWith(ctx context.Context, d *Driver, invoke grpc.UnaryInvoker) error {
	return d.gate.intercept(ctx, workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName,
		new(workflowservice.PollActivityTaskQueueRequest), new(workflowservice.PollActivityTaskQueueResponse), nil, invoke)
}

// poll is a poll the server answers at once, with the task or empty when it is nil.
func poll(ctx context.Context, d *Driver, answer *workflowservice.PollActivityTaskQueueResponse) error {
	return pollWith(ctx, d, func(_ context.Context, _ string, _, reply any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		if answer != nil {
			proto.Merge(reply.(*workflowservice.PollActivityTaskQueueResponse), answer)
		}
		return nil
	})
}

// hang is a poll the server never answers.
func hang(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
	<-ctx.Done()
	return status.FromContextError(ctx.Err()).Err()
}

func describing(t *testing.T) *fakeServer {
	return newFakeServer(t, func(string) ([]byte, error) { return describeResponse(t), nil })
}

func completedReplyFor(t *testing.T, payload *commonpb.Payload) runner.Decision {
	return runner.Decision{Reply: completedReply(t, payload)}
}

// requireClosed checks the exchange's stream ends.
func requireClosed(t *testing.T, x runner.Exchange) {
	t.Helper()
	for {
		select {
		case _, open := <-x.Events():
			if !open {
				return
			}
		case <-time.After(30 * time.Second):
			t.Fatal("the event stream did not close")
		}
	}
}

func requireReturns[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(30 * time.Second):
		t.Fatalf("%s did not return", what)
		var zero T
		return zero
	}
}

func requireBlocked[T any](t *testing.T, ch <-chan T, what string) {
	t.Helper()
	select {
	case got := <-ch:
		t.Fatalf("%s returned early: %v", what, got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestReadCannotInvokeAnswer(t *testing.T) {
	d := testDriver(t, describing(t).address, Options{})
	d.ensureWorker = func(context.Context) error {
		t.Error("a read attempted to start the live Answer worker")
		return nil
	}
	ctx, cancel := context.WithCancel(t.Context())
	polled := make(chan error, 1)
	go func() { polled <- poll(ctx, d, task("token")) }()
	instruction := describeInstruction(t, 30*time.Second)
	instruction.Value = append(instruction.Value, 0x28, 0x01)
	x := start(t, d, 1, instruction)
	for i := range 3 {
		event := next(t, x)
		require.NoError(t, x.Respond(t.Context(), event.ID, explained(i < 2)))
	}
	require.Equal(t, runner.EventDone, next(t, x).Kind)
	cancel()
	require.ErrorIs(t, requireReturns(t, polled, "the held poll"), context.Canceled)
}

// TestAnOutcomeCountsOnlyInTheWindowThatAdmittedItsPoll is the reviewer's race, made
// deterministic: a poll admitted in one window ends exactly as that window closes, and a later
// window must neither see its outcome nor admit while it is outstanding.
func TestAnOutcomeCountsOnlyInTheWindowThatAdmittedItsPoll(t *testing.T) {
	d := testDriver(t, unreachable(t), Options{})
	g := d.gate
	x := newExchange(d, 1)
	first := g.openFor(x)
	release := make(chan struct{})
	polled := make(chan error, 1)
	go func() {
		polled <- pollWith(t.Context(), d, func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
			<-release
			return nil
		})
	}()
	require.Equal(t, pollAdmitted, requireReturns(t, first.outcomes, "the first window's admission"))
	require.False(t, g.closeIdle(first), "the closing window admitted a poll")
	second := g.openFor(x)
	held := make(chan error, 1)
	go func() { held <- poll(t.Context(), d, task("token")) }()
	requireBlocked(t, held, "an admission while the first poll is outstanding")
	close(release)
	require.NoError(t, requireReturns(t, polled, "the first poll"))
	require.Equal(t, pollEmpty, requireReturns(t, first.outcomes, "the first window's outcome"))
	require.Equal(t, pollAdmitted, requireReturns(t, second.outcomes, "the second window's admission"))
	require.NoError(t, requireReturns(t, held, "the second poll"))
	require.Equal(t, pollTook, requireReturns(t, second.outcomes, "the second window's outcome"))
	select {
	case outcome := <-first.outcomes:
		t.Fatalf("the first window received the second poll's outcome %v", outcome)
	default:
	}
	owner, captured := g.claim([]byte("token"))
	require.Same(t, second, owner)
	require.Equal(t, []byte("token"), captured.GetTaskToken())
	g.finish(first)
	owner, _ = g.claim([]byte("token"))
	require.Same(t, second, owner, "a stale window cannot release the current one")
}

func workerExchange(t *testing.T, d *Driver, request uint64, timeout time.Duration) runner.Exchange {
	t.Helper()
	x := newExchange(d, request)
	d.mu.Lock()
	d.exchanges[x] = true
	d.mu.Unlock()
	go func() {
		defer close(x.done)
		defer close(x.events)
		ctx, cancel := context.WithTimeout(x.ctx, timeout)
		defer cancel()
		x.work(ctx, timeout)
	}()
	return x
}

func TestCloseRefusesHeldPollsAndCancelsTheAdmittedOne(t *testing.T) {
	d := testDriver(t, describing(t).address, Options{})
	x := workerExchange(t, d, 1, 30*time.Second)
	admitted := make(chan error, 1)
	go func() { admitted <- pollWith(t.Context(), d, hang) }()
	// Only once the hanging poll holds the gate does a second one start, so it can only be held.
	waitForGate(t, d)
	held := make(chan error, 1)
	go func() { held <- poll(t.Context(), d, nil) }()
	requireBlocked(t, held, "a poll while another is admitted")
	began := time.Now()
	require.NoError(t, d.Close())
	require.Less(t, time.Since(began), 5*time.Second)
	require.ErrorIs(t, requireReturns(t, held, "the held poll"), errStopped)
	require.Equal(t, codes.Canceled, status.Code(requireReturns(t, admitted, "the admitted poll")))
	requireClosed(t, x)
	require.ErrorIs(t, poll(t.Context(), d, nil), errStopped, "a poll after Close is refused at once")
}

func waitForGate(t *testing.T, d *Driver) {
	t.Helper()
	await.RequireTrue(t, func() bool { return holding(d) }, 30*time.Second, time.Millisecond)
}

func holding(d *Driver) bool {
	d.gate.mu.Lock()
	defer d.gate.mu.Unlock()
	return d.gate.held != nil
}

func TestEveryWaitEndsAtTheReadTimeoutAndAtClose(t *testing.T) {
	type wait struct {
		name  string
		setup func(t *testing.T, d *Driver, x runner.Exchange) (activity <-chan error)
		fault string
	}
	waits := []wait{
		{
			name: "an admitted poll that never returns",
			setup: func(t *testing.T, d *Driver, x runner.Exchange) <-chan error {
				done := make(chan error, 1)
				go func() { done <- pollWith(t.Context(), d, hang) }()
				waitForGate(t, d)
				return done
			},
			fault: "read timeout without Model acceptance",
		},
		{
			name: "a taken task with no callback",
			setup: func(t *testing.T, d *Driver, x runner.Exchange) <-chan error {
				done := make(chan error, 1)
				go func() { done <- poll(t.Context(), d, task("token")) }()
				waitForGate(t, d)
				require.NoError(t, requireReturns(t, done, "the poll"))
				return nil
			},
			fault: "the worker took a task and never called back",
		},
		{
			name: "a callback the Model never decides",
			setup: func(t *testing.T, d *Driver, x runner.Exchange) <-chan error {
				polled := make(chan error, 1)
				go func() { polled <- poll(t.Context(), d, task("token")) }()
				waitForGate(t, d)
				require.NoError(t, requireReturns(t, polled, "the poll"))
				done := make(chan error, 1)
				go func() {
					_, err := d.deliver(t.Context(), []byte("token"))
					done <- err
				}()
				require.Equal(t, runner.EventCallback, next(t, x).Kind)
				return done
			},
			fault: "read timeout without Model acceptance",
		},
	}
	for _, w := range waits {
		t.Run(w.name+"/read timeout", func(t *testing.T) {
			d := testDriver(t, describing(t).address, Options{})
			x := workerExchange(t, d, 1, 2*time.Second)
			activity := w.setup(t, d, x)
			event := next(t, x)
			require.Equal(t, runner.EventFault, event.Kind)
			require.Equal(t, w.fault, event.Fault)
			requireClosed(t, x)
			if activity != nil {
				requireBlocked(t, activity, "the wait's work before Close")
			}
			require.NoError(t, d.Close())
			if activity != nil {
				_ = requireReturns(t, activity, "the wait's work after Close")
			}
		})
		t.Run(w.name+"/close", func(t *testing.T) {
			d := testDriver(t, describing(t).address, Options{})
			x := workerExchange(t, d, 1, 30*time.Second)
			activity := w.setup(t, d, x)
			began := time.Now()
			require.NoError(t, d.Close())
			require.Less(t, time.Since(began), 5*time.Second)
			requireClosed(t, x)
			if activity != nil {
				_ = requireReturns(t, activity, "the wait's work")
			}
		})
	}
}

func TestTheDriverNeverAnswersATaskTheModelDidNot(t *testing.T) {
	t.Run("refused", func(t *testing.T) {
		d := testDriver(t, describing(t).address, Options{})
		polled := make(chan error, 1)
		go func() { polled <- poll(t.Context(), d, task("token")) }()
		x := workerExchange(t, d, 1, 30*time.Second)
		waitForGate(t, d)
		require.NoError(t, requireReturns(t, polled, "the poll"))
		delivered := make(chan error, 1)
		go func() {
			_, err := d.deliver(t.Context(), []byte("token"))
			delivered <- err
		}()
		callback := next(t, x)
		require.NoError(t, x.Respond(t.Context(), callback.ID, runner.Decision{Refusal: &modelv1.Refusal{Code: modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED}}))
		requireClosed(t, x)
		require.NoError(t, x.Close())
		requireBlocked(t, delivered, "a refused task before Close")
		require.NoError(t, d.Close())
		require.ErrorIs(t, requireReturns(t, delivered, "the refused task"), errReleased)
	})
	t.Run("unclaimed", func(t *testing.T) {
		d := testDriver(t, unreachable(t), Options{})
		delivered := make(chan error, 1)
		go func() {
			_, err := d.deliver(t.Context(), []byte("token"))
			delivered <- err
		}()
		requireBlocked(t, delivered, "an unclaimed task before Close")
		require.NoError(t, d.Close())
		require.ErrorIs(t, requireReturns(t, delivered, "the unclaimed task"), errReleased)
	})
}

func TestAnUncapturedTaskIsRefusedBeforeAnswer(t *testing.T) {
	d := testDriver(t, describing(t).address, Options{})
	polled := make(chan error, 1)
	go func() { polled <- poll(t.Context(), d, task("token")) }()
	x := workerExchange(t, d, 1, 30*time.Second)
	waitForGate(t, d)
	require.NoError(t, requireReturns(t, polled, "the poll"))
	go func() { _, _ = d.deliver(t.Context(), []byte("another token")) }()
	event := next(t, x)
	require.Equal(t, runner.EventFault, event.Kind)
	require.Contains(t, event.Fault, "malformed evidence")
}

// startServer answers Start with a started run and records Terminate; terminate is what it
// answers Terminate with.
func startServer(t *testing.T, startDelay time.Duration, terminate error) *fakeServer {
	return newFakeServer(t, func(method string) ([]byte, error) {
		switch method {
		case workflowservice.WorkflowService_StartActivityExecution_FullMethodName:
			<-time.After(startDelay)
			return marshal(t, &workflowservice.StartActivityExecutionResponse{RunId: "run", Started: true}), nil
		case workflowservice.WorkflowService_TerminateActivityExecution_FullMethodName:
			if terminate != nil {
				return nil, terminate
			}
			return marshal(t, &workflowservice.TerminateActivityExecutionResponse{}), nil
		default:
			return nil, nil
		}
	})
}

var startRequest = &workflowservice.StartActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity", RequestId: "request"}

func terminated(t *testing.T, server *fakeServer) []*workflowservice.TerminateActivityExecutionRequest {
	t.Helper()
	var requests []*workflowservice.TerminateActivityExecutionRequest
	for _, data := range server.calls(workflowservice.WorkflowService_TerminateActivityExecution_FullMethodName) {
		request := new(workflowservice.TerminateActivityExecutionRequest)
		require.NoError(t, proto.Unmarshal(data, request))
		requests = append(requests, request)
	}
	return requests
}

func TestCloseTerminatesWhatTheDriverStarted(t *testing.T) {
	t.Run("answered", func(t *testing.T) {
		server := startServer(t, 0, nil)
		d := testDriver(t, server.address, Options{})
		x := start(t, d, 1, packed(t, startRequest))
		require.NoError(t, x.Respond(t.Context(), next(t, x).ID, explained(false)))
		require.Equal(t, runner.EventDone, next(t, x).Kind)
		require.NoError(t, d.Close())
		got := terminated(t, server)
		require.Len(t, got, 1)
		require.Equal(t, "namespace", got[0].GetNamespace())
		require.Equal(t, "activity", got[0].GetActivityId())
		require.Equal(t, "run", got[0].GetRunId())
	})
	t.Run("answered too late", func(t *testing.T) {
		server := startServer(t, 500*time.Millisecond, nil)
		d := testDriver(t, server.address, Options{RPCTimeout: 50 * time.Millisecond})
		x := start(t, d, 1, packed(t, startRequest))
		require.Equal(t, runner.EventFault, next(t, x).Kind)
		require.NoError(t, d.Close())
		got := terminated(t, server)
		require.Len(t, got, 1, "an unanswered Start may still have started the activity")
		require.Equal(t, "activity", got[0].GetActivityId())
		require.Empty(t, got[0].GetRunId(), "terminated by its ID")
	})
	for _, tolerated := range []codes.Code{codes.NotFound, codes.FailedPrecondition} {
		t.Run("already closed "+tolerated.String(), func(t *testing.T) {
			server := startServer(t, 0, status.Error(tolerated, "closed"))
			d := testDriver(t, server.address, Options{})
			x := start(t, d, 1, packed(t, startRequest))
			require.NoError(t, x.Respond(t.Context(), next(t, x).ID, explained(false)))
			require.NoError(t, d.Close())
		})
	}
	t.Run("failed", func(t *testing.T) {
		server := startServer(t, 0, status.Error(codes.Internal, "broken"))
		d := testDriver(t, server.address, Options{})
		x := start(t, d, 1, packed(t, startRequest))
		require.NoError(t, x.Respond(t.Context(), next(t, x).ID, explained(false)))
		require.ErrorContains(t, d.Close(), "terminate activity activity")
	})
}

// TestOnlyAResponseToAnUnansweredTaskIsATeardownFault: a Run that ends right after an Answer may
// close while the SDK still sends the Model's reply; that response is the Model's own, however
// late. Only a response for a task held because the Model did not answer it is a fault.
func TestOnlyAResponseToAnUnansweredTaskIsATeardownFault(t *testing.T) {
	d := testDriver(t, describing(t).address, Options{})
	polled := make(chan error, 1)
	go func() { polled <- poll(t.Context(), d, task("answered")) }()
	x := workerExchange(t, d, 1, 30*time.Second)
	waitForGate(t, d)
	require.NoError(t, requireReturns(t, polled, "the poll"))
	delivered := make(chan error, 1)
	go func() {
		_, err := d.deliver(t.Context(), []byte("answered"))
		delivered <- err
	}()
	callback := next(t, x)
	require.NoError(t, x.Respond(t.Context(), callback.ID, completedReplyFor(t, &commonpb.Payload{Data: []byte("Complete")})))
	require.NoError(t, requireReturns(t, delivered, "the answered task"))
	unanswered := make(chan error, 1)
	go func() {
		_, err := d.deliver(t.Context(), []byte("unanswered"))
		unanswered <- err
	}()
	await.RequireTrue(t, func() bool { return d.wasUnanswered([]byte("unanswered")) }, 30*time.Second, time.Millisecond)
	respond := func(token string) {
		require.NoError(t, d.observeResponses(t.Context(), workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName,
			&workflowservice.RespondActivityTaskCompletedRequest{TaskToken: []byte(token)}, nil, nil,
			func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error { return nil }))
	}
	require.NoError(t, d.Close())
	require.ErrorIs(t, requireReturns(t, unanswered, "the unanswered task"), errReleased)
	faults := func() []error {
		d.mu.Lock()
		defer d.mu.Unlock()
		return append([]error(nil), d.teardowns...)
	}
	// Both responses complete after the release, as an in-flight reply can.
	respond("answered")
	require.Empty(t, faults(), "the Model's own reply completing after the Run is no fault")
	respond("unanswered")
	require.Len(t, faults(), 1)
	require.ErrorContains(t, faults()[0], "the SDK completed a task the Model did not answer with RespondActivityTaskCompleted")
}
