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

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	annotationsv1 "umpire.local/model/gen/umpire/annotations/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	model "umpire.local/model/host"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

type modelSliceAnnotations struct {
	host     *model.Host
	mu       sync.Mutex
	coverage map[string]int
	failures []string
	seed     atomic.Uint64
}

// modelSliceCapture owns one invocation's actual after-fill request and raw result.
type modelSliceCapture struct {
	method   string
	request  proto.Message
	response proto.Message
	rpcErr   error
	localErr error
}

// modelSliceCaptures is a caller-owned sink for the attempts of one RPC call context.
type modelSliceCaptures struct {
	mu       sync.Mutex
	attempts []*modelSliceCapture
}

type modelSliceCaptureKey struct{}

func modelSliceCapturing(ctx context.Context) (context.Context, *modelSliceCaptures) {
	captures := new(modelSliceCaptures)
	return context.WithValue(ctx, modelSliceCaptureKey{}, captures), captures
}

// single returns the only attempt; callers that expect retries read attempts directly.
func (c *modelSliceCaptures) single(t *testing.T) *modelSliceCapture {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	require.Len(t, c.attempts, 1, "one captured invocation")
	return c.attempts[0]
}

// observation transports the owned pair; local validation errors never become RPC results.
func (c *modelSliceCapture) observation(t *testing.T) *realizationv1.Observation {
	t.Helper()
	observation := &realizationv1.Observation{Method: c.method, Request: modelSliceAny(t, c.request)}
	if c.rpcErr != nil {
		rpcStatus := status.Convert(c.rpcErr)
		observation.Result = &realizationv1.Observation_Error{Error: &realizationv1.Error{Code: int32(rpcStatus.Code()), Message: rpcStatus.Message()}}
	} else {
		observation.Result = &realizationv1.Observation_Response{Response: modelSliceAny(t, c.response)}
	}
	return observation
}

func (a *modelSliceAnnotations) intercept(ctx context.Context, method string, request, response any, connection *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
	if err := a.annotate(method, false, request); err != nil {
		return err
	}
	captures, _ := ctx.Value(modelSliceCaptureKey{}).(*modelSliceCaptures)
	var capture *modelSliceCapture
	if captures != nil {
		capture = &modelSliceCapture{method: method, request: proto.Clone(request.(proto.Message))}
		defer func() {
			captures.mu.Lock()
			defer captures.mu.Unlock()
			captures.attempts = append(captures.attempts, capture)
		}()
	}
	err := invoke(ctx, method, request, response, connection, options...)
	if capture != nil {
		// Record the raw result first so every later refusal keeps it.
		capture.rpcErr = err
		if message, ok := response.(proto.Message); ok && err == nil {
			capture.response = proto.Clone(message)
		}
		if !proto.Equal(capture.request, request.(proto.Message)) {
			capture.localErr = status.Error(codes.Internal, "annotation request changed after capture")
			a.record(fmt.Sprintf("method=%s failure=request changed after capture", method), true)
			return capture.localErr
		}
	}
	if err != nil {
		a.record(fmt.Sprintf("method=%s rpc_error=%s", method, status.Code(err)), false)
		return err
	}
	if capture == nil {
		return a.annotate(method, true, response)
	}
	capture.localErr = a.annotate(method, true, response)
	return capture.localErr
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
	output, err := model.Annotate(ctx, a.host, &annotationsv1.AnnotationRequest{Method: method, Response: response, Message: data, Seed: a.seed.Add(1)})
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

var modelSliceTransportMethods = []string{
	workflowservice.WorkflowService_StartActivityExecution_FullMethodName,
	workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName,
	workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName,
	workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName,
	workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName,
	workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName,
	workflowservice.WorkflowService_PollActivityExecution_FullMethodName,
}

func (a *modelSliceAnnotations) missingCoverage() []string {
	var missing []string
	for _, method := range modelSliceTransportMethods {
		for _, response := range []bool{false, true} {
			key := fmt.Sprintf("method=%s response=%t scope=validated", method, response)
			if a.coverage[key] == 0 {
				missing = append(missing, key)
			}
		}
	}
	return missing
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
	require.Empty(t, a.missingCoverage(), "missing validated transport coverage")

}

// modelSliceRunID derives a valid run ID that names the request which produced it.
func modelSliceRunID(requestID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(requestID)).String()
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
		ctx, captures := modelSliceCapturing(t.Context())
		err := annotations.intercept(ctx, workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName,
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
		capture := captures.single(t)
		require.NoError(t, capture.rpcErr, "local validation is not an RPC error")
		require.Equal(t, codes.InvalidArgument, status.Code(capture.localErr))
		require.True(t, proto.Equal(before, capture.response), "raw result is retained unrepaired")
		require.NotNil(t, capture.observation(t).GetResponse())
	})
	t.Run("NestedFailureNotRepaired", func(t *testing.T) {
		annotations := newAnnotations(t)
		response := &workflowservice.PollActivityExecutionResponse{
			Outcome: &activitypb.ActivityExecutionOutcome{Value: &activitypb.ActivityExecutionOutcome_Failure{
				Failure: &failurepb.Failure{StackTrace: "private-stack", Cause: &failurepb.Failure{
					FailureInfo: &failurepb.Failure_ActivityFailureInfo{ActivityFailureInfo: &failurepb.ActivityFailureInfo{RetryState: 999}},
				}},
			}},
		}
		before := proto.Clone(response)
		called := false
		err := annotations.intercept(t.Context(), workflowservice.WorkflowService_PollActivityExecution_FullMethodName,
			&workflowservice.PollActivityExecutionRequest{Namespace: "model-slice", ActivityId: "activity"}, response, nil,
			func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
				called = true
				return nil
			})
		require.Error(t, err)
		require.True(t, called)
		require.True(t, proto.Equal(before, response))
		require.Len(t, annotations.failures, 1)
		require.Contains(t, annotations.failures[0], "ActivityFailureInfo.retry_state")
		require.NotContains(t, annotations.failures[0], "private-stack")
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
		ctx, captures := modelSliceCapturing(t.Context())
		for range 2 {
			err := annotations.intercept(ctx, workflowservice.WorkflowService_StartActivityExecution_FullMethodName, request,
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
		require.Len(t, captures.attempts, 2)
		for _, capture := range captures.attempts {
			require.True(t, proto.Equal(request, capture.request))
			require.Equal(t, codes.Unavailable, status.Code(capture.rpcErr))
			require.Nil(t, capture.response, "an RPC error captures no result")
			require.NoError(t, capture.localErr)
			observation := capture.observation(t)
			require.Equal(t, int32(codes.Unavailable), observation.GetError().GetCode())
			require.Nil(t, observation.GetResponse())
		}
		captures.attempts[0].request.(*workflowservice.StartActivityExecutionRequest).RequestId = "changed"
		require.Equal(t, ids[1], captures.attempts[1].request.(*workflowservice.StartActivityExecutionRequest).GetRequestId(), "attempts own separate clones")
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
	started := func(request any, response any) {
		response.(*workflowservice.StartActivityExecutionResponse).RunId = modelSliceRunID(request.(*workflowservice.StartActivityExecutionRequest).GetRequestId())
		response.(*workflowservice.StartActivityExecutionResponse).Started = true
	}
	t.Run("CaptureIsTheInnermostAfterFillInvocation", func(t *testing.T) {
		annotations := newAnnotations(t)
		request := start()
		response := new(workflowservice.StartActivityExecutionResponse)
		var seen proto.Message
		ctx, captures := modelSliceCapturing(t.Context())
		err := annotations.intercept(ctx, workflowservice.WorkflowService_StartActivityExecution_FullMethodName, request, response, nil,
			func(_ context.Context, _ string, request, response any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
				seen = proto.Clone(request.(proto.Message))
				started(request, response)
				return nil
			})
		require.NoError(t, err)
		require.Empty(t, annotations.failures)
		capture := captures.single(t)
		require.True(t, proto.Equal(seen, capture.request), "capture is the message the innermost invoker saw")
		require.NotEmpty(t, capture.request.(*workflowservice.StartActivityExecutionRequest).GetRequestId(), "capture follows the request fill")
		raw := proto.Clone(response)
		request.Namespace, request.RequestId = "caller-alias", "caller-alias"
		response.RunId, response.Started = "caller-alias", false
		require.True(t, proto.Equal(seen, capture.request), "caller request aliases cannot alter the capture")
		require.True(t, proto.Equal(raw, capture.response), "caller response aliases cannot alter the capture")
		observation := capture.observation(t)
		actual, err := observation.GetRequest().UnmarshalNew()
		require.NoError(t, err)
		require.True(t, proto.Equal(seen, actual))
		result, err := observation.GetResponse().UnmarshalNew()
		require.NoError(t, err)
		require.True(t, proto.Equal(raw, result))
	})
	t.Run("LaterRequestMutationIsRefused", func(t *testing.T) {
		annotations := newAnnotations(t)
		ctx, captures := modelSliceCapturing(t.Context())
		err := annotations.intercept(ctx, workflowservice.WorkflowService_StartActivityExecution_FullMethodName, start(),
			new(workflowservice.StartActivityExecutionResponse), nil,
			func(_ context.Context, _ string, request, response any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
				request.(*workflowservice.StartActivityExecutionRequest).Identity = "later-interceptor"
				started(request, response)
				return nil
			})
		require.Equal(t, codes.Internal, status.Code(err))
		capture := captures.single(t)
		require.Empty(t, capture.request.(*workflowservice.StartActivityExecutionRequest).GetIdentity())
		require.NoError(t, capture.rpcErr)
		require.NotEmpty(t, capture.response.(*workflowservice.StartActivityExecutionResponse).GetRunId(), "refusal keeps the raw result")
		require.Error(t, capture.localErr)
		require.Len(t, annotations.failures, 1)
		require.Contains(t, annotations.failures[0], "request changed after capture")
		failing := newAnnotations(t)
		ctx, captures = modelSliceCapturing(t.Context())
		err = failing.intercept(ctx, workflowservice.WorkflowService_StartActivityExecution_FullMethodName, start(),
			new(workflowservice.StartActivityExecutionResponse), nil,
			func(_ context.Context, _ string, request, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
				request.(*workflowservice.StartActivityExecutionRequest).Identity = "later-interceptor"
				return status.Error(codes.Unavailable, "retry")
			})
		require.Equal(t, codes.Internal, status.Code(err))
		capture = captures.single(t)
		require.Equal(t, codes.Unavailable, status.Code(capture.rpcErr), "refusal keeps the raw RPC error")
		require.Nil(t, capture.response)
		require.Equal(t, int32(codes.Unavailable), capture.observation(t).GetError().GetCode())
	})
	t.Run("OverlappingReversedResponsesKeepTheirInvocation", func(t *testing.T) {
		annotations := newAnnotations(t)
		type call struct {
			captures *modelSliceCaptures
			sent     chan string
			release  chan struct{}
			done     chan error
		}
		calls := make([]call, 2)
		for i := range calls {
			ctx, captures := modelSliceCapturing(t.Context())
			calls[i] = call{captures, make(chan string, 1), make(chan struct{}), make(chan error, 1)}
			go func(c call) {
				c.done <- annotations.intercept(ctx, workflowservice.WorkflowService_StartActivityExecution_FullMethodName, start(),
					new(workflowservice.StartActivityExecutionResponse), nil,
					func(_ context.Context, _ string, request, response any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
						c.sent <- request.(*workflowservice.StartActivityExecutionRequest).GetRequestId()
						<-c.release
						started(request, response)
						return nil
					})
			}(calls[i])
		}
		ids := []string{<-calls[0].sent, <-calls[1].sent}
		require.NotEqual(t, ids[0], ids[1])
		for _, i := range []int{1, 0} {
			close(calls[i].release)
			require.NoError(t, <-calls[i].done)
		}
		for i, c := range calls {
			capture := c.captures.single(t)
			require.Equal(t, ids[i], capture.request.(*workflowservice.StartActivityExecutionRequest).GetRequestId())
			require.Equal(t, modelSliceRunID(ids[i]), capture.response.(*workflowservice.StartActivityExecutionResponse).GetRunId())
		}
		require.Empty(t, annotations.failures)
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

func TestTestpilotModelSliceCoverageRejectsMissingWorkerTraffic(t *testing.T) {
	a := &modelSliceAnnotations{coverage: map[string]int{}}
	for _, method := range modelSliceTransportMethods {
		for _, response := range []bool{false, true} {
			a.coverage[fmt.Sprintf("method=%s response=%t scope=validated", method, response)] = 1
		}
	}
	require.Empty(t, a.missingCoverage())
	for _, method := range []string{workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName} {
		key := fmt.Sprintf("method=%s response=true scope=validated", method)
		delete(a.coverage, key)
		a.coverage[fmt.Sprintf("method=%s response=true scope=out_of_scope:missing instrumentation", method)] = 1
		require.Contains(t, a.missingCoverage(), key)
	}
}

func (a *modelSliceAnnotations) actual(t *testing.T, evidence *anypb.Any) {
	t.Helper()
	if evidence.MessageIs(new(temporalv1.ReconciledObservation)) {
		compound := new(temporalv1.ReconciledObservation)
		require.NoError(t, evidence.UnmarshalTo(compound))
		a.actual(t, compound.Acknowledged)
		a.actual(t, compound.Current)
		return
	}
	observation := new(realizationv1.Observation)
	require.NoError(t, evidence.UnmarshalTo(observation))
	for _, response := range []bool{false, true} {
		value := observation.GetRequest()
		if response {
			value = observation.GetResponse()
		}
		if value == nil {
			continue
		}
		message, err := value.UnmarshalNew()
		require.NoError(t, err)
		require.NoError(t, a.annotate(observation.Method, response, message))
	}
}
