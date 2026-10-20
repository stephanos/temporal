//go:build test_dep && integration

package tests

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"umpire.local/model"
	temporalmodel "umpire.local/temporal"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

type modelSliceAnnotations struct {
	host     *model.Host
	mu       sync.Mutex
	coverage map[string]int
	failures []string
	seed     atomic.Uint64
}

func (a *modelSliceAnnotations) intercept(ctx context.Context, method string, request, response any, connection *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
	if err := a.annotate(method, false, request); err != nil {
		return err
	}
	err := invoke(ctx, method, request, response, connection, options...)
	if err != nil {
		a.record(fmt.Sprintf("method=%s rpc_error=%s", method, status.Code(err)), false)
		return err
	}
	return a.annotate(method, true, response)
}

func (a *modelSliceAnnotations) record(event string, failed bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.coverage == nil {
		a.coverage = make(map[string]int)
	}
	a.coverage[event]++
	if failed {
		a.failures = append(a.failures, event)
	}
}

func (a *modelSliceAnnotations) annotate(method string, response bool, value any) error {
	event := fmt.Sprintf("method=%s response=%t", method, response)
	fail := func(reason string) error {
		a.record(event+" failure="+reason, true)
		return status.Error(codes.InvalidArgument, "annotation "+reason)
	}
	message, ok := value.(proto.Message)
	if !ok {
		return fail("non-protobuf message")
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return fail("marshal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := temporalmodel.Annotate(ctx, a.host, &temporalv1.AnnotationRequest{Method: method, Response: response, Message: data, Seed: a.seed.Add(1)})
	if err != nil {
		return fail("guest refused")
	}
	if len(output.GetViolations()) != 0 {
		return fail(fmt.Sprintf("violations=%v", output.GetViolations()))
	}
	if !response {
		if err := proto.Unmarshal(output.GetMessage(), message); err != nil {
			return fail("filled request decode")
		}
	}
	scope := "validated"
	if output.GetOutOfScope() != "" {
		scope = "out_of_scope:" + output.GetOutOfScope()
	}
	a.record(event+" scope="+scope, false)
	return nil
}

func (a *modelSliceAnnotations) assert(t *testing.T) {
	t.Helper()
	a.mu.Lock()
	defer a.mu.Unlock()
	keys := make([]string, 0, len(a.coverage))
	for key := range a.coverage {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	t.Log("transport scope=validated covers declared Rust catalog constraints and nested consumed fields; ignored opaque subtrees remain unvalidated for their catalog reasons")
	for _, key := range keys {
		t.Logf("actual transport coverage %s attempts=%d", key, a.coverage[key])
	}
	require.Empty(t, a.failures, "annotation failures retained through shutdown")
	for _, method := range []string{
		workflowservice.WorkflowService_StartActivityExecution_FullMethodName,
		workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName,
		workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName,
		workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName,
		workflowservice.WorkflowService_PollActivityExecution_FullMethodName,
	} {
		for _, response := range []bool{false, true} {
			prefix := fmt.Sprintf("method=%s response=%t scope=validated", method, response)
			require.Contains(t, keys, prefix, "missing validated transport coverage")
		}
	}
}

func modelSliceAnnotationControls(t *testing.T, wasm []byte) {
	newAnnotations := func(t *testing.T) *modelSliceAnnotations {
		host, err := model.New(t.Context(), wasm)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, host.Close(context.Background())) })
		return &modelSliceAnnotations{host: host}
	}
	start := func() *workflowservice.StartActivityExecutionRequest {
		return &workflowservice.StartActivityExecutionRequest{
			Namespace: "model-slice", ActivityId: "activity", ActivityType: &commonpb.ActivityType{Name: "ModelSlice"},
			TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"}, StartToCloseTimeout: durationpb.New(20 * time.Second),
			RetryPolicy: &commonpb.RetryPolicy{MaximumAttempts: 1},
		}
	}
	t.Run("InvalidRequestNeverSent", func(t *testing.T) {
		annotations := newAnnotations(t)
		request := start()
		request.ActivityId = ""
		called := false
		err := annotations.intercept(t.Context(), workflowservice.WorkflowService_StartActivityExecution_FullMethodName, request,
			new(workflowservice.StartActivityExecutionResponse), nil,
			func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
				called = true
				return nil
			})
		require.Error(t, err)
		require.False(t, called)
		require.Len(t, annotations.failures, 1)
	})
	t.Run("ResponseNotRepaired", func(t *testing.T) {
		annotations := newAnnotations(t)
		response := new(workflowservice.DescribeActivityExecutionResponse)
		before := proto.Clone(response)
		called := false
		err := annotations.intercept(t.Context(), workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName,
			&workflowservice.DescribeActivityExecutionRequest{Namespace: "model-slice", ActivityId: "activity"}, response, nil,
			func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
				called = true
				return nil
			})
		require.Error(t, err)
		require.True(t, called)
		require.True(t, proto.Equal(before, response))
		require.Len(t, annotations.failures, 1)
		require.Contains(t, annotations.failures[0], "response=true")
	})
	t.Run("UnknownTrafficRefusesEveryAttempt", func(t *testing.T) {
		annotations := newAnnotations(t)
		var sent atomic.Int32
		results := make(chan error, 2)
		for range 2 {
			go func() {
				results <- annotations.intercept(t.Context(), "/temporal.api.workflowservice.v1.WorkflowService/Unknown", start(),
					new(workflowservice.StartActivityExecutionResponse), nil,
					func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
						sent.Add(1)
						return nil
					})
			}()
		}
		errors := []error{<-results, <-results}
		for _, err := range errors {
			require.Error(t, err)
		}
		require.Zero(t, sent.Load())
		require.Len(t, annotations.failures, 2)
	})
	t.Run("RetryKeepsFilledRequestID", func(t *testing.T) {
		annotations := newAnnotations(t)
		request := start()
		var ids []string
		for range 2 {
			err := annotations.intercept(t.Context(), workflowservice.WorkflowService_StartActivityExecution_FullMethodName, request,
				new(workflowservice.StartActivityExecutionResponse), nil,
				func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
					ids = append(ids, request.GetRequestId())
					return status.Error(codes.Unavailable, "retry")
				})
			require.Equal(t, codes.Unavailable, status.Code(err))
		}
		require.Len(t, ids, 2)
		require.NotEmpty(t, ids[0])
		require.Equal(t, ids[0], ids[1])
		require.Equal(t, "model-slice", request.GetNamespace())
		require.Equal(t, "activity", request.GetActivityId())
		require.Equal(t, "queue", request.GetTaskQueue().GetName())
		require.Empty(t, annotations.failures)
		require.Equal(t, 2, annotations.coverage["method="+workflowservice.WorkflowService_StartActivityExecution_FullMethodName+" rpc_error=Unavailable"])
		for event := range annotations.coverage {
			require.False(t, strings.Contains(event, "response=true"), "RPC errors do not validate allocated response")
		}
		independent := start()
		err := annotations.intercept(t.Context(), workflowservice.WorkflowService_StartActivityExecution_FullMethodName, independent,
			new(workflowservice.StartActivityExecutionResponse), nil,
			func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
				return status.Error(codes.Unavailable, "retry")
			})
		require.Equal(t, codes.Unavailable, status.Code(err))
		require.NotEmpty(t, independent.GetRequestId())
		require.NotEqual(t, ids[0], independent.GetRequestId())
	})
	t.Run("CanceledTransportDoesNotPoisonGuest", func(t *testing.T) {
		annotations := newAnnotations(t)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		for _, attempt := range []struct {
			ctx  context.Context
			code codes.Code
		}{{ctx, codes.Canceled}, {t.Context(), codes.Unavailable}} {
			request := start()
			err := annotations.intercept(attempt.ctx, workflowservice.WorkflowService_StartActivityExecution_FullMethodName, request,
				new(workflowservice.StartActivityExecutionResponse), nil,
				func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
					if ctx.Err() != nil {
						return status.Error(codes.Canceled, "canceled")
					}
					return status.Error(codes.Unavailable, "retry")
				})
			require.Equal(t, attempt.code, status.Code(err))
			require.NotEmpty(t, request.GetRequestId())
		}
	})
}
