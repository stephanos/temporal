package act

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func workerReceive[T any](ctx context.Context, t *testing.T, events <-chan T) T {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-ctx.Done():
		t.Fatal(ctx.Err())
		var zero T
		return zero
	}
}

func workerTestOptions(follow func(context.Context, client.Client, []byte) error, observe func(context.Context, WorkerRPC) error) Options {
	return Options{Program: []byte{1}, ValidateProgram: func([]byte) error { return nil }, BindWorker: func(context.Context) (WorkerBinding, error) {
		return WorkerBinding{Follow: follow, Observe: observe, Close: func() error { return nil }}, nil
	}}
}

func TestWorkerCaptureReservationsAndBounds(t *testing.T) {
	request := &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: "namespace", TaskToken: []byte("token")}
	for _, test := range []struct {
		name      string
		budget    uint64
		byteLimit uint64
		admitted  int
		bound     error
	}{
		{name: "count", budget: 1, byteLimit: DefaultWorkerCaptureBytes, admitted: 1, bound: ErrWorkerCaptureInvocationLimit},
		{name: "bytes", budget: DefaultWorkerRPCBudget, byteLimit: minimumWorkerCaptureBytes(), admitted: 2, bound: ErrWorkerCaptureByteLimit},
	} {
		t.Run(test.name, func(t *testing.T) {
			var published []WorkerRPC
			capture, err := newWorkerCapture(Options{WorkerRPCBudget: test.budget, WorkerCaptureBytes: test.byteLimit}, func(_ context.Context, rpc WorkerRPC) error { published = append(published, rpc); return nil })
			require.NoError(t, err)
			var first *workerInvocation
			for i := 0; i < test.admitted; i++ {
				invocation, err := capture.begin(t.Context(), workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, request)
				require.NoError(t, err)
				if i == 0 {
					first = invocation
				}
			}
			_, err = capture.begin(t.Context(), workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, request)
			require.ErrorIs(t, err, test.bound)
			require.Len(t, published, test.admitted)
			require.Len(t, capture.calls, test.admitted)
			for i, rpc := range published {
				require.Equal(t, uint64(i+1), rpc.Invocation)
				require.Equal(t, uint64(i+1), rpc.Sequence)
				require.Equal(t, WorkerRPCStarted, rpc.Stage)
			}
			before := capture.occupied
			reached := new(progress)
			reached.sent.Store(true)
			reached.answered.Store(true)
			require.NoError(t, capture.complete(t.Context(), first, &workflowservice.RecordActivityTaskHeartbeatResponse{}, reached, nil))
			require.Less(t, capture.occupied, before)
			if test.name == "bytes" {
				_, err := capture.begin(t.Context(), workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, request)
				require.NoError(t, err, "completed response releases unused reservation")
			}
		})
	}
}

func TestWorkerCaptureLimitsRefuseBeforeBinding(t *testing.T) {
	for _, test := range []struct {
		name  string
		count uint64
		bytes uint64
		bound error
	}{
		{name: "capture cannot reserve SDK response", bytes: 64 << 20, bound: ErrWorkerCaptureByteLimit},
		{name: "sequence arithmetic", count: math.MaxUint64, bound: ErrWorkerCaptureInvocationLimit},
	} {
		t.Run(test.name, func(t *testing.T) {
			var bound atomic.Bool
			opts := Options{Address: "127.0.0.1:1", Namespace: "namespace", TaskQueue: "queue", Program: []byte{1}, ValidateProgram: func([]byte) error { return nil }, WorkerRPCBudget: test.count, WorkerCaptureBytes: test.bytes,
				BindWorker: func(context.Context) (WorkerBinding, error) { bound.Store(true); return WorkerBinding{}, nil },
			}
			_, err := NewFactory(opts)
			require.ErrorIs(t, err, test.bound)
			require.False(t, bound.Load())
		})
	}
}

type workerStatsObserver struct{ ended atomic.Int32 }

func (*workerStatsObserver) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}
func (s *workerStatsObserver) HandleRPC(_ context.Context, event stats.RPCStats) {
	if _, ok := event.(*stats.End); ok {
		s.ended.Add(1)
	}
}
func (*workerStatsObserver) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}
func (*workerStatsObserver) HandleConn(context.Context, stats.ConnStats) {}

func TestWorkerCaptureIsInsideMutatorsAndDoesNotFaultSuccessfulWork(t *testing.T) {
	response := &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}
	server := newFakeServer(t, func(string) ([]byte, error) { return marshal(t, response), nil })
	events := make(chan WorkerRPC, 4)
	result := make(chan error, 1)
	statsObserver := new(workerStatsObserver)
	opts := workerTestOptions(func(ctx context.Context, conn client.Client, _ []byte) error {
		_, err := conn.WorkflowService().RecordActivityTaskHeartbeat(ctx, &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: "namespace", TaskToken: []byte("token"), Identity: "before"})
		result <- err
		<-ctx.Done()
		return nil
	}, func(_ context.Context, rpc WorkerRPC) error {
		events <- cloneWorkerRPC(rpc)
		rpc.Request.Value = []byte("mutated-observer-copy")
		return nil
	})
	opts.Transport.DialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(statsObserver), grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, request, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, callOptions ...grpc.CallOption) error {
		if request, ok := request.(*workflowservice.RecordActivityTaskHeartbeatRequest); ok {
			request.Identity = "actual-mutated"
		}
		return invoke(ctx, method, request, reply, conn, callOptions...)
	})}
	driver := testDriver(t, server.address, opts)
	require.NoError(t, workerReceive(t.Context(), t, result))
	require.Positive(t, statsObserver.ended.Load(), "caller stats handler remains installed")
	started := workerReceive(t.Context(), t, events)
	completed := workerReceive(t.Context(), t, events)
	require.Equal(t, WorkerRPCStarted, started.Stage)
	require.Equal(t, WorkerRPCCompleted, completed.Stage)
	require.Equal(t, started.Invocation, completed.Invocation)
	require.Equal(t, uint64(1), started.Sequence)
	require.Equal(t, uint64(2), completed.Sequence)
	require.True(t, completed.Sent)
	require.True(t, completed.Answered)
	require.Empty(t, completed.TransportError)
	require.Nil(t, completed.CallError)
	actual := new(workflowservice.RecordActivityTaskHeartbeatRequest)
	require.NoError(t, completed.Request.UnmarshalTo(actual))
	require.Equal(t, "actual-mutated", actual.GetIdentity())
	require.Equal(t, marshal(t, actual), server.calls(completed.Method)[0])
	controller := packed(t, cancelRequest)
	evidence, err := driver.workerFaultEvidence(controller)
	require.NoError(t, err)
	fault := new(temporalv1.WorkerTransportFault)
	require.NoError(t, evidence.UnmarshalTo(fault))
	require.True(t, proto.Equal(controller, fault.GetController()))
	require.Len(t, fault.GetCalls(), 2)
	require.True(t, proto.Equal(completed.Request, fault.GetCalls()[1].GetRequest()))
	require.NoError(t, driver.Close(), "retained captures alone do not create a DriverFault")
}

func TestWorkerCaptureObserverFaultPreservesActualSuccess(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return nil, nil })
	result := make(chan error, 1)
	opts := workerTestOptions(func(ctx context.Context, conn client.Client, _ []byte) error {
		_, err := conn.WorkflowService().RecordActivityTaskHeartbeat(ctx, &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: "namespace", TaskToken: []byte("token")})
		result <- err
		<-ctx.Done()
		return nil
	}, func(_ context.Context, rpc WorkerRPC) error {
		if rpc.Stage == WorkerRPCCompleted {
			return errors.New("observer-completion-fault")
		}
		return nil
	})
	driver := testDriver(t, server.address, opts)
	require.NoError(t, workerReceive(t.Context(), t, result), "the SDK receives actual invoke success unchanged")
	err := driver.Close()
	require.ErrorContains(t, err, "observer-completion-fault")
	var carrier interface{ Evidence() *anypb.Any }
	require.ErrorAs(t, err, &carrier)
	fault := new(temporalv1.WorkerTransportFault)
	require.NoError(t, carrier.Evidence().UnmarshalTo(fault))
	require.Len(t, fault.GetCalls(), 2)
	require.NotNil(t, fault.GetCalls()[1].GetResponse())
}

func TestWorkerCaptureAmbiguousInternalErrorsKeepNeutralStatus(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		t.Run(map[bool]string{false: "server-internal", true: "malformed-protobuf"}[malformed], func(t *testing.T) {
			server := newFakeServer(t, func(string) ([]byte, error) {
				if malformed {
					return []byte{0xff}, nil
				}
				failure, err := status.New(codes.Internal, "actual-server-internal").WithDetails(&commonpb.Payload{Data: []byte("actual-status-detail")})
				if err != nil {
					return nil, err
				}
				return nil, failure.Err()
			})
			events := make(chan WorkerRPC, 4)
			result := make(chan error, 1)
			opts := workerTestOptions(func(ctx context.Context, conn client.Client, _ []byte) error {
				_, err := conn.WorkflowService().RecordActivityTaskHeartbeat(ctx, &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: "namespace", TaskToken: []byte("token")})
				result <- err
				return err
			}, func(_ context.Context, rpc WorkerRPC) error { events <- rpc; return nil })
			driver := testDriver(t, server.address, opts)
			require.Error(t, workerReceive(t.Context(), t, result))
			workerReceive(t.Context(), t, events)
			completed := workerReceive(t.Context(), t, events)
			require.Nil(t, completed.ServerError)
			require.NotEmpty(t, completed.TransportError)
			require.NotNil(t, completed.CallError)
			neutral := new(statuspb.Status)
			require.NoError(t, completed.CallError.UnmarshalTo(neutral))
			require.Equal(t, int32(codes.Internal), neutral.GetCode())
			if malformed {
				require.Contains(t, neutral.GetMessage(), "unmarshal")
			} else {
				require.Equal(t, "actual-server-internal", neutral.GetMessage())
				require.Len(t, neutral.GetDetails(), 1)
				detail := new(commonpb.Payload)
				require.NoError(t, neutral.GetDetails()[0].UnmarshalTo(detail))
				require.Equal(t, []byte("actual-status-detail"), detail.GetData())
			}
			workerReceive(t.Context(), t, driver.workerDone)
			require.Error(t, driver.Close())
			require.Empty(t, server.calls(workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName))
			require.Empty(t, server.calls(workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName))
		})
	}
}

func TestWorkerBindingStateIsPerDriverAndIdleShutdownIsClean(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return nil, nil })
	var bound, closed atomic.Int32
	opts := Options{Address: server.address, Namespace: "namespace", TaskQueue: "queue", Program: []byte{1}, ValidateProgram: func(data []byte) error {
		if len(data) != 1 || data[0] != 1 {
			return errors.New("factory program was mutated")
		}
		return nil
	}, BindWorker: func(context.Context) (WorkerBinding, error) {
		bound.Add(1)
		return WorkerBinding{Follow: func(ctx context.Context, _ client.Client, _ []byte) error { <-ctx.Done(); return nil }, Observe: func(context.Context, WorkerRPC) error { return nil }, Close: func() error { closed.Add(1); return nil }}, nil
	}}
	factory, err := NewFactory(opts)
	require.NoError(t, err)
	opts.Program[0] = 2
	for range 2 {
		driver, err := factory.New(t.Context())
		require.NoError(t, err)
		require.NoError(t, driver.(interface{ Close() error }).Close())
	}
	require.Equal(t, int32(2), bound.Load())
	require.Equal(t, int32(2), closed.Load())
}

func TestWorkerCaptureCanceledEmptyPollShutdownIsClean(t *testing.T) {
	entered := make(chan struct{}, 1)
	server := newFakeServerContext(t, func(ctx context.Context, method string) ([]byte, error) {
		if method == workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName {
			entered <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return nil, nil
	})
	events := make(chan WorkerRPC, 4)
	opts := workerTestOptions(func(ctx context.Context, conn client.Client, _ []byte) error {
		_, err := conn.WorkflowService().PollActivityTaskQueue(ctx, &workflowservice.PollActivityTaskQueueRequest{Namespace: "namespace"})
		if ctx.Err() != nil {
			return nil
		}
		return err
	}, func(_ context.Context, rpc WorkerRPC) error { events <- rpc; return nil })
	driver := testDriver(t, server.address, opts)
	_, err := driver.polls.prepare(false)
	require.NoError(t, err)
	workerReceive(t.Context(), t, entered)
	require.NoError(t, driver.Close())
	started := workerReceive(t.Context(), t, events)
	completed := workerReceive(t.Context(), t, events)
	require.Equal(t, WorkerRPCStarted, started.Stage)
	require.Equal(t, WorkerRPCCompleted, completed.Stage)
	require.NotEmpty(t, completed.TransportError)
	require.Nil(t, completed.ServerError)
	require.NotNil(t, completed.CallError)
	require.Equal(t, started.Invocation, completed.Invocation)
}

func TestWorkerCaptureBoundRetainsActualPartialFaultEvidence(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return nil, nil })
	results := make(chan error, 2)
	opts := workerTestOptions(func(ctx context.Context, conn client.Client, _ []byte) error {
		for range 2 {
			_, err := conn.WorkflowService().RecordActivityTaskHeartbeat(ctx, &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: "namespace", TaskToken: []byte("token")})
			results <- err
			if err != nil {
				return err
			}
		}
		<-ctx.Done()
		return nil
	}, func(context.Context, WorkerRPC) error { return nil })
	opts.WorkerRPCBudget = 1
	driver := testDriver(t, server.address, opts)
	require.NoError(t, workerReceive(t.Context(), t, results))
	require.ErrorContains(t, workerReceive(t.Context(), t, results), "WorkerCaptureInvocationLimit")
	workerReceive(t.Context(), t, driver.workerDone)
	err := driver.Close()
	require.ErrorIs(t, err, ErrWorkerCaptureInvocationLimit)
	var carrier interface{ Evidence() *anypb.Any }
	require.ErrorAs(t, err, &carrier)
	fault := new(temporalv1.WorkerTransportFault)
	require.NoError(t, carrier.Evidence().UnmarshalTo(fault))
	require.Len(t, fault.GetCalls(), 2)
	require.Equal(t, uint64(1), fault.GetCalls()[0].GetInvocation())
	require.Equal(t, uint64(1), fault.GetCalls()[1].GetInvocation())
	require.Len(t, server.calls(workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName), 1)
}

func TestWorkerCaptureRunLifetimeJoinsBackgroundTerminal(t *testing.T) {
	for _, failFollow := range []bool{false, true} {
		t.Run(map[bool]string{false: "Close", true: "Follow failure"}[failFollow], func(t *testing.T) {
			entered, left := make(chan struct{}), make(chan struct{})
			var enteredOnce sync.Once
			server := newFakeServerContext(t, func(ctx context.Context, method string) ([]byte, error) {
				if method == workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName {
					enteredOnce.Do(func() { close(entered) })
					<-ctx.Done()
					close(left)
					return nil, ctx.Err()
				}
				return nil, nil
			})
			events := make(chan WorkerRPC, 4)
			result := make(chan error, 1)
			connections := make(chan client.Client, 1)
			opts := workerTestOptions(func(ctx context.Context, conn client.Client, _ []byte) error {
				connections <- conn
				go func() {
					_, err := conn.WorkflowService().RespondActivityTaskCompleted(context.WithoutCancel(ctx), &workflowservice.RespondActivityTaskCompletedRequest{Namespace: "namespace", TaskToken: []byte("token")})
					result <- err
				}()
				select {
				case <-entered:
				case <-ctx.Done():
					return nil
				}
				if failFollow {
					return errors.New("Follow failed while terminal was outstanding")
				}
				<-ctx.Done()
				return nil
			}, func(_ context.Context, rpc WorkerRPC) error { events <- rpc; return nil })
			driver := testDriver(t, server.address, opts)
			connection := workerReceive(t.Context(), t, connections)
			workerReceive(t.Context(), t, entered)
			if failFollow {
				workerReceive(t.Context(), t, driver.workerDone)
			}
			err := driver.Close()
			if failFollow {
				require.ErrorContains(t, err, "Follow failed while terminal was outstanding")
			} else {
				require.NoError(t, err)
			}
			require.Error(t, workerReceive(t.Context(), t, result))
			workerReceive(t.Context(), t, left)
			workerReceive(t.Context(), t, events)
			completed := workerReceive(t.Context(), t, events)
			require.Equal(t, WorkerRPCCompleted, completed.Stage)
			require.NotEmpty(t, completed.TransportError)
			require.NotNil(t, completed.CallError)
			require.Zero(t, driver.workerCapture.active)
			require.True(t, driver.workerCapture.sealed)
			_, err = connection.WorkflowService().RespondActivityTaskCompleted(t.Context(), &workflowservice.RespondActivityTaskCompletedRequest{Namespace: "namespace", TaskToken: []byte("token")})
			require.Error(t, err)
			require.Empty(t, events, "no invocation can publish after Close sealed capture")
			require.Len(t, server.calls(workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName), 1)
			if failFollow {
				var carrier interface{ Evidence() *anypb.Any }
				require.ErrorAs(t, driver.Close(), &carrier)
				fault := new(temporalv1.WorkerTransportFault)
				require.NoError(t, carrier.Evidence().UnmarshalTo(fault))
				require.Len(t, fault.GetCalls(), 2)
				require.NotNil(t, fault.GetCalls()[1].GetCallError())
			}
		})
	}
}

func TestWorkerCaptureMinimumIncludesMaximumRequestAnyFraming(t *testing.T) {
	opts := Options{WorkerRPCBudget: 1, WorkerCaptureBytes: minimumWorkerCaptureBytes()}
	require.NoError(t, opts.validateWorkerLimits())
	opts.WorkerCaptureBytes--
	require.ErrorIs(t, opts.validateWorkerLimits(), ErrWorkerCaptureByteLimit)
	require.Greater(t, minimumWorkerCaptureBytes(), uint64(3*workerMaxPayloadSize)+workerMaxStatusBytes+workerCaptureMetadata)
}
