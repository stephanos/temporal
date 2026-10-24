package temporal_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	model "umpire.local/model/host"
	temporalmodel "umpire.local/temporal"
)

const describeMethod = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"
const startMethod = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"
const receiptRun = "22222222-2222-4222-8222-222222222222"
const receiptNamespace = "model-slice"
const receiptActivity = "activity"

const cancelMethod = "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"

func cancelObservation(t *testing.T, code int32) *anypb.Any {
	t.Helper()
	value := &realizationv1.Observation{Method: cancelMethod, Request: pack(t, &workflowservice.RequestCancelActivityExecutionRequest{Namespace: receiptNamespace, ActivityId: receiptActivity, RunId: receiptRun})}
	if code == 0 {
		value.Result = &realizationv1.Observation_Response{Response: pack(t, &workflowservice.RequestCancelActivityExecutionResponse{})}
	} else {
		value.Result = &realizationv1.Observation_Error{Error: &realizationv1.Error{Code: code, Message: "actual server status"}}
	}
	return pack(t, value)
}

func TestCancelSuccessAndRejectedSelfLoop(t *testing.T) {
	for _, tc := range []struct {
		policy string
		status enumspb.ActivityExecutionStatus
	}{
		{"Complete", enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED},
		{"Fail", enumspb.ACTIVITY_EXECUTION_STATUS_FAILED},
		{"", enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED},
	} {
		t.Run(tc.status.String(), func(t *testing.T) {
			h := guest(t)
			if tc.policy == "" {
				explained(t, judge(t, h, cancelObservation(t, 0)), "Client::Cancel")
			} else {
				deliver(t, h)
				judge(t, h, observation(t, tc.status, 0))
			}
			for range 2 {
				result := judge(t, h, cancelObservation(t, 5)).GetExplained()
				if result == nil || !slices.Equal(result.Steps, []string{"Client::Cancel"}) || !slices.Equal(result.Outcomes, []string{"Rejected(NotFound)"}) {
					t.Fatalf("rejected cancel = %v", result)
				}
				explained(t, judge(t, h, observation(t, tc.status, 0)))
			}
			unexplained(t, judge(t, h, cancelObservation(t, 0)), "Client::Cancel")
			unexplained(t, judge(t, h, cancelObservation(t, 5)), "Client::Cancel")
		})
	}
}

func TestCancelStatusContradictionFreezesAndMalformedEnvelopeRefuses(t *testing.T) {
	for _, closed := range []bool{false, true} {
		for _, code := range []int32{5, 6, 9, 3, 7, 14, 999} {
			if closed && code == 5 {
				continue
			}
			h := guest(t)
			expected := []string{"Client::Cancel", "Worker::Poll(1)", "Worker::Poll(2)", "Worker::Poll(3)", "Worker::RespondCanceled(2)", "Worker::RespondCanceled(3)"}
			if closed {
				explained(t, judge(t, h, cancelObservation(t, 0)), "Client::Cancel")
				expected = []string{"Client::Cancel"}
			}
			unexplained(t, judge(t, h, cancelObservation(t, code)), expected...)
			unexplained(t, judge(t, h, cancelObservation(t, 0)), expected...)
			_, err := h.Answer(context.Background(), &modelv1.AnswerRequest{Callback: callback(t, "Complete")})
			refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED)
			reset(t, h)
			explained(t, judge(t, h, cancelObservation(t, 0)), "Client::Cancel")
		}
	}
	for _, tc := range []struct {
		value *realizationv1.Observation
		code  modelv1.RefusalCode
	}{
		{&realizationv1.Observation{Method: "unknown", Result: &realizationv1.Observation_Error{Error: &realizationv1.Error{Code: 5}}}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{&realizationv1.Observation{Method: cancelMethod}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{&realizationv1.Observation{Method: cancelMethod, Result: &realizationv1.Observation_Response{Response: pack(t, &workflowservice.DescribeActivityExecutionResponse{})}}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
	} {
		h := guest(t)
		_, err := h.Judge(context.Background(), &modelv1.JudgeRequest{Observation: pack(t, tc.value)})
		refusal(t, err, tc.code)
		explained(t, judge(t, h, scheduled(t)))
	}
}

func TestObservationResultWireCompatibility(t *testing.T) {
	for _, tc := range []struct {
		value *realizationv1.Observation
		wire  []byte
	}{
		{&realizationv1.Observation{Method: "m", Result: &realizationv1.Observation_Response{Response: &anypb.Any{TypeUrl: "x", Value: []byte{1, 2}}}}, []byte{10, 1, 'm', 18, 7, 10, 1, 'x', 18, 2, 1, 2}},
		{&realizationv1.Observation{Method: "m", Result: &realizationv1.Observation_Error{Error: &realizationv1.Error{Code: 5, Message: "x"}}}, []byte{10, 1, 'm', 26, 5, 8, 5, 18, 1, 'x'}},
		// Deliberate protobuf-go upgrade tripwire: this release emits the oneof after regular
		// fields, an order protobuf does not promise. If an upgrade changes it, re-pin these
		// bytes; the decode equality with Rust's number order below is the compatibility claim.
		{&realizationv1.Observation{Method: "m", Result: &realizationv1.Observation_Response{Response: &anypb.Any{TypeUrl: "x", Value: []byte{1, 2}}}, Request: &anypb.Any{TypeUrl: "r", Value: []byte{3}}}, []byte{10, 1, 'm', 34, 6, 10, 1, 'r', 18, 1, 3, 18, 7, 10, 1, 'x', 18, 2, 1, 2}},
	} {
		encoded, err := proto.Marshal(tc.value)
		if err != nil || !bytes.Equal(encoded, tc.wire) {
			t.Fatalf("encoded %x, %v; want %x", encoded, err, tc.wire)
		}
		var decoded realizationv1.Observation
		if err := proto.Unmarshal(tc.wire, &decoded); err != nil || !proto.Equal(tc.value, &decoded) {
			t.Fatalf("decoded %v, %v; want %v", &decoded, err, tc.value)
		}
	}
	want := &realizationv1.Observation{Method: "m", Result: &realizationv1.Observation_Response{Response: &anypb.Any{TypeUrl: "x", Value: []byte{1, 2}}}, Request: &anypb.Any{TypeUrl: "r", Value: []byte{3}}}
	var decoded realizationv1.Observation
	rust := []byte{10, 1, 'm', 18, 7, 10, 1, 'x', 18, 2, 1, 2, 34, 6, 10, 1, 'r', 18, 1, 3}
	if err := proto.Unmarshal(rust, &decoded); err != nil || !proto.Equal(want, &decoded) {
		t.Fatalf("decoded Rust order %v, %v; want %v", &decoded, err, want)
	}
}

func TestActualRequestIsRequiredForLiveBinding(t *testing.T) {
	h := guest(t)
	value := scheduled(t)
	observation := new(realizationv1.Observation)
	if err := value.UnmarshalTo(observation); err != nil {
		t.Fatal(err)
	}
	observation.Request = nil
	_, err := h.Judge(context.Background(), &modelv1.JudgeRequest{Observation: pack(t, observation)})
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST)
	explained(t, judge(t, h, value))
}

func startObservation(t *testing.T, attempts int32) *anypb.Any {
	t.Helper()
	return pack(t, &realizationv1.Observation{Method: startMethod,
		Request: pack(t, &workflowservice.StartActivityExecutionRequest{
			Namespace: receiptNamespace, ActivityId: receiptActivity, RequestId: "11111111-1111-4111-8111-111111111111",
			ActivityType: &commonpb.ActivityType{Name: "ModelSlice"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"},
			StartToCloseTimeout: durationpb.New(20 * time.Second), RetryPolicy: &commonpb.RetryPolicy{MaximumAttempts: attempts},
		}), Result: &realizationv1.Observation_Response{Response: pack(t, &workflowservice.StartActivityExecutionResponse{RunId: receiptRun, Started: true})}})
}

func unboundGuest(t *testing.T) *model.Host {
	t.Helper()
	h, err := model.New(context.Background(), temporalmodel.Guest())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := h.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return h
}

func guest(t *testing.T) *model.Host {
	t.Helper()
	h := unboundGuest(t)
	explained(t, judge(t, h, startObservation(t, 3)))
	return h
}

func pack(t *testing.T, message proto.Message) *anypb.Any {
	t.Helper()
	a, err := anypb.New(message)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func observation(t *testing.T, status enumspb.ActivityExecutionStatus, run enumspb.PendingActivityState) *anypb.Any {
	t.Helper()
	return pack(t, &realizationv1.Observation{
		Method:  describeMethod,
		Request: pack(t, &workflowservice.DescribeActivityExecutionRequest{Namespace: receiptNamespace, ActivityId: receiptActivity, RunId: receiptRun, IncludeLastFailure: true}),
		Result: &realizationv1.Observation_Response{Response: pack(t, &workflowservice.DescribeActivityExecutionResponse{RunId: receiptRun, Info: &activitypb.ActivityExecutionInfo{
			ActivityId: receiptActivity, RunId: receiptRun, Attempt: 1, Status: status, RunState: run,
		}})},
	})
}

func scheduled(t *testing.T) *anypb.Any {
	t.Helper()
	return observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, enumspb.PENDING_ACTIVITY_STATE_SCHEDULED)
}

func started(t *testing.T) *anypb.Any {
	t.Helper()
	return observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, enumspb.PENDING_ACTIVITY_STATE_STARTED)
}

func callback(t *testing.T, policy string) *anypb.Any {
	t.Helper()
	return pack(t, &realizationv1.Callback{Task: pack(t, &workflowservice.PollActivityTaskQueueResponse{
		Attempt: 1, WorkflowNamespace: receiptNamespace, ActivityId: receiptActivity, ActivityRunId: receiptRun,
		Input: &commonpb.Payloads{Payloads: []*commonpb.Payload{{
			Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte(policy),
		}}},
	})})
}

func judge(t *testing.T, h *model.Host, value *anypb.Any) *modelv1.JudgeResponse {
	t.Helper()
	response, err := h.Judge(context.Background(), &modelv1.JudgeRequest{Observation: value})
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func explained(t *testing.T, response *modelv1.JudgeResponse, steps ...string) {
	t.Helper()
	if response.GetExplained() == nil || !slices.Equal(response.GetExplained().Steps, steps) {
		t.Fatalf("judgment = %v, want explicit Explanation %v", response, steps)
	}
	outcomes := make([]string, len(steps))
	for i := range outcomes {
		outcomes[i] = "Accepted"
	}
	if !slices.Equal(response.GetExplained().Outcomes, outcomes) {
		t.Fatalf("outcomes = %v, want %v", response.GetExplained().Outcomes, outcomes)
	}
}

func unexplained(t *testing.T, response *modelv1.JudgeResponse, actions ...string) {
	t.Helper()
	if response.GetUnexplained() == nil || !slices.Equal(response.GetUnexplained().ExpectedActions, actions) {
		t.Fatalf("judgment = %v, want Unexplained %v", response, actions)
	}
}

func refusal(t *testing.T, err error, code modelv1.RefusalCode) {
	t.Helper()
	var typed *model.RefusalError
	if !errors.As(err, &typed) || typed.Code != code {
		t.Fatalf("error = %v, want typed refusal %s", err, code)
	}
}

func deliver(t *testing.T, h *model.Host) {
	t.Helper()
	explained(t, judge(t, h, started(t)), "Worker::Poll(1)")
}

func reset(t *testing.T, h *model.Host) {
	t.Helper()
	response, err := h.Reset(context.Background(), &modelv1.ResetRequest{})
	if err != nil || response == nil {
		t.Fatalf("Reset = %v, %v", response, err)
	}
}

func TestObservedDeliveryAndTerminal(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status enumspb.ActivityExecutionStatus
		step   string
	}{
		{"Complete", enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, "Worker::RespondCompleted"},
		{"Fail", enumspb.ACTIVITY_EXECUTION_STATUS_FAILED, "Worker::RespondFailed { retryable: false }"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, skipStarted := range []bool{false, true} {
				h := guest(t)
				explained(t, judge(t, h, scheduled(t)))
				explained(t, judge(t, h, scheduled(t)))
				want := []string{tc.step}
				if skipStarted {
					want = append([]string{"Worker::Poll(1)"}, want...)
				} else {
					deliver(t, h)
					explained(t, judge(t, h, started(t)))
				}
				explained(t, judge(t, h, observation(t, tc.status, 0)), want...)
				explained(t, judge(t, h, observation(t, tc.status, 0)))
			}
		})
	}
}

func TestImpossibleObservationFreezesUntilReset(t *testing.T) {
	for _, tc := range []struct {
		name     string
		before   func(*testing.T, *model.Host)
		bad      func(*testing.T) *anypb.Any
		expected []string
	}{
		{"stale scheduled after delivery", func(t *testing.T, h *model.Host) { deliver(t, h) }, scheduled, []string{"Client::Cancel", "Worker::Poll(2)", "Worker::Poll(3)", "Worker::RespondCompleted", "Worker::RespondFailed { retryable: false }", "Worker::RespondFailed { retryable: true }", "Worker::Heartbeat", "Worker::RespondCanceled(2)", "Worker::RespondCanceled(3)"}},
		{"opposite terminal after closure", func(t *testing.T, h *model.Host) {
			deliver(t, h)
			explained(t, judge(t, h, observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0)), "Worker::RespondCompleted")
		}, func(t *testing.T) *anypb.Any { return observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_FAILED, 0) }, []string{"Client::Cancel"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := guest(t)
			if tc.before != nil {
				tc.before(t, h)
			}
			first := judge(t, h, tc.bad(t))
			unexplained(t, first, tc.expected...)
			if next := judge(t, h, scheduled(t)); !proto.Equal(first, next) {
				t.Fatalf("frozen result changed: %v -> %v", first, next)
			}
			_, err := h.Answer(context.Background(), &modelv1.AnswerRequest{Callback: callback(t, "Complete")})
			refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED)
			reset(t, h)
			explained(t, judge(t, h, scheduled(t)))
			deliver(t, h)
			explained(t, judge(t, h, started(t)))
		})
	}
}

func TestObservationRefusalPreservesScheduled(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value func(*testing.T) *anypb.Any
		code  modelv1.RefusalCode
	}{
		{"wrong outer type", func(t *testing.T) *anypb.Any { return callback(t, "Complete") }, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"alternate outer URL", func(t *testing.T) *anypb.Any {
			a := scheduled(t)
			a.TypeUrl = "other/umpire.realization.v1.Observation"
			return a
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"malformed envelope", func(t *testing.T) *anypb.Any { a := scheduled(t); a.Value = []byte{255}; return a }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"unsupported method", func(t *testing.T) *anypb.Any {
			return pack(t, &realizationv1.Observation{Method: "DescribeActivityExecution", Result: &realizationv1.Observation_Response{Response: pack(t, &workflowservice.DescribeActivityExecutionResponse{})}})
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"missing method", func(t *testing.T) *anypb.Any {
			return pack(t, &realizationv1.Observation{Result: &realizationv1.Observation_Response{Response: pack(t, &workflowservice.DescribeActivityExecutionResponse{})}})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"missing response", func(t *testing.T) *anypb.Any { return pack(t, &realizationv1.Observation{Method: describeMethod}) }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"missing response URL", func(t *testing.T) *anypb.Any {
			return pack(t, &realizationv1.Observation{Method: describeMethod, Result: &realizationv1.Observation_Response{Response: &anypb.Any{}}})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"wrong inner type", func(t *testing.T) *anypb.Any {
			return pack(t, &realizationv1.Observation{Method: describeMethod, Result: &realizationv1.Observation_Response{Response: pack(t, &workflowservice.PollActivityTaskQueueResponse{})}})
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"alternate inner URL", func(t *testing.T) *anypb.Any {
			return pack(t, &realizationv1.Observation{Method: describeMethod, Result: &realizationv1.Observation_Response{Response: &anypb.Any{TypeUrl: "other/temporal.api.workflowservice.v1.DescribeActivityExecutionResponse"}}})
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"malformed response", func(t *testing.T) *anypb.Any {
			return pack(t, &realizationv1.Observation{Method: describeMethod, Result: &realizationv1.Observation_Response{Response: &anypb.Any{TypeUrl: "type.googleapis.com/temporal.api.workflowservice.v1.DescribeActivityExecutionResponse", Value: []byte{255}}}})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"missing info", func(t *testing.T) *anypb.Any {
			return pack(t, &realizationv1.Observation{Method: describeMethod, Result: &realizationv1.Observation_Response{Response: pack(t, &workflowservice.DescribeActivityExecutionResponse{})}})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"unknown status", func(t *testing.T) *anypb.Any { return observation(t, 999, 1) }, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"unknown run state", func(t *testing.T) *anypb.Any { return observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, 999) }, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"unspecified status", func(t *testing.T) *anypb.Any { return observation(t, 0, 1) }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"unspecified running state", func(t *testing.T) *anypb.Any { return observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, 0) }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"paused", func(t *testing.T) *anypb.Any {
			return observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_PAUSED, enumspb.PENDING_ACTIVITY_STATE_PAUSED)
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"closed with running state", func(t *testing.T) *anypb.Any {
			return observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, enumspb.PENDING_ACTIVITY_STATE_STARTED)
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := guest(t)
			_, err := h.Judge(context.Background(), &modelv1.JudgeRequest{Observation: tc.value(t)})
			refusal(t, err, tc.code)
			explained(t, judge(t, h, scheduled(t)))
			deliver(t, h)
			explained(t, judge(t, h, started(t)))
		})
	}
}

func TestCancelAndTimeoutDoNotInventEvidence(t *testing.T) {
	for _, tc := range []struct {
		name      string
		delivered bool
		status    enumspb.ActivityExecutionStatus
	}{
		{"cancel scheduled", false, enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED},
		{"cancel started", true, enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED},
		{"schedule to start timeout", false, enumspb.ACTIVITY_EXECUTION_STATUS_TIMED_OUT},
		{"start to close timeout", true, enumspb.ACTIVITY_EXECUTION_STATUS_TIMED_OUT},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := guest(t)
			if tc.delivered {
				deliver(t, h)
			}
			want := []string{"Client::Cancel", "Worker::Poll(1)", "Worker::Poll(2)", "Worker::Poll(3)", "Worker::RespondCanceled(2)", "Worker::RespondCanceled(3)"}
			if tc.delivered {
				want = []string{"Client::Cancel", "Worker::Poll(2)", "Worker::Poll(3)", "Worker::RespondCompleted", "Worker::RespondFailed { retryable: false }", "Worker::RespondFailed { retryable: true }", "Worker::Heartbeat", "Worker::RespondCanceled(2)", "Worker::RespondCanceled(3)"}
			}
			unexplained(t, judge(t, h, observation(t, tc.status, 0)), want...)
			_, err := h.Answer(context.Background(), &modelv1.AnswerRequest{Callback: callback(t, "Complete")})
			refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED)
		})
	}
}

func TestRepeatedDeliveryAndInstanceIsolation(t *testing.T) {
	first, second := guest(t), guest(t)
	deliver(t, first)
	_, err := first.Answer(context.Background(), &modelv1.AnswerRequest{Callback: callback(t, "Fail")})
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED)
	explained(t, judge(t, first, started(t)))
	explained(t, judge(t, second, scheduled(t)))
	deliver(t, second)
	explained(t, judge(t, second, observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_FAILED, 0)), "Worker::RespondFailed { retryable: false }")
	reset(t, second)
	explained(t, judge(t, second, scheduled(t)))
	explained(t, judge(t, first, started(t)))
	reset(t, first)
	explained(t, judge(t, first, scheduled(t)))
}

func TestStartReceiptBindsExactCapturedBytesAndRunInWasm(t *testing.T) {
	h := unboundGuest(t)
	ctx := context.Background()
	captured := startObservation(t, 2)
	value := new(realizationv1.Observation)
	if err := captured.UnmarshalTo(value); err != nil {
		t.Fatal(err)
	}
	response := new(workflowservice.StartActivityExecutionResponse)
	if err := value.GetResponse().UnmarshalTo(response); err != nil {
		t.Fatal(err)
	}
	response.Started = false
	value.Result = &realizationv1.Observation_Response{Response: pack(t, response)}
	replay := pack(t, value)
	_, err := h.Judge(ctx, &modelv1.JudgeRequest{Observation: replay})
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
	explained(t, judge(t, h, captured))
	deliver(t, h)
	explained(t, judge(t, h, replay))
	for _, kind := range []string{"unknown wire field", "private changed input", "different execution", "crossed run"} {
		t.Run(kind, func(t *testing.T) {
			value := new(realizationv1.Observation)
			if err := captured.UnmarshalTo(value); err != nil {
				t.Fatal(err)
			}
			request := new(workflowservice.StartActivityExecutionRequest)
			if err := value.GetRequest().UnmarshalTo(request); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "unknown wire field":
				value.Request.Value = append(value.Request.Value, 0xf8, 0x07, 0x01)
			case "private changed input":
				request.Input = &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("private-admission-payload")}}}
				value.Request = pack(t, request)
			case "different execution":
				request.ActivityId = "another-activity"
				value.Request = pack(t, request)
			case "crossed run":
				changed := proto.Clone(response).(*workflowservice.StartActivityExecutionResponse)
				changed.RunId = "33333333-3333-4333-8333-333333333333"
				value.Result = &realizationv1.Observation_Response{Response: pack(t, changed)}
			default:
				t.Fatalf("unknown mutation %s", kind)
			}
			_, err := h.Judge(ctx, &modelv1.JudgeRequest{Observation: pack(t, value)})
			refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
			if err != nil && bytes.Contains([]byte(err.Error()), []byte("private-admission-payload")) {
				t.Fatal("private bytes appeared in refusal")
			}
			explained(t, judge(t, h, started(t)))
		})
	}
	reset(t, h)
	explained(t, judge(t, h, replay))
	explained(t, judge(t, h, scheduled(t)))
}

func TestTemporalAnswerUnsupported(t *testing.T) {
	h := guest(t)
	_, err := h.Answer(context.Background(), &modelv1.AnswerRequest{Callback: callback(t, "Complete")})
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED)
	explained(t, judge(t, h, scheduled(t)))
}
