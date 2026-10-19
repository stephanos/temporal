package temporal_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	model "umpire.local/model"
	modelv1 "umpire.local/model/umpire/model/v1"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

const describeMethod = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"

func guestBytes(t *testing.T) []byte {
	t.Helper()
	path := os.Getenv("UMPIRE_TEMPORAL_WASM")
	if path == "" {
		target := os.Getenv("CARGO_TARGET_DIR")
		if target == "" {
			target = "../../../target"
		}
		path = filepath.Join(target, "wasm32-wasip1", "release", "umpire_temporal_guest.wasm")
	}
	wasm, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Temporal guest is required (%s): %v", path, err)
	}
	return wasm
}

func guest(t *testing.T) *model.Host {
	t.Helper()
	h, err := model.New(context.Background(), guestBytes(t))
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
	return pack(t, &temporalv1.Observation{
		Method: describeMethod,
		Response: pack(t, &workflowservice.DescribeActivityExecutionResponse{Info: &activitypb.ActivityExecutionInfo{
			Status: status, RunState: run,
		}}),
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
	return pack(t, &temporalv1.Callback{Task: pack(t, &workflowservice.PollActivityTaskQueueResponse{
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

func answer(t *testing.T, h *model.Host, policy string) *temporalv1.Reply {
	t.Helper()
	response, err := h.Answer(context.Background(), &modelv1.AnswerRequest{Callback: callback(t, policy)})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetReply().GetTypeUrl() != "type.googleapis.com/umpire.temporal.v1.Reply" {
		t.Fatalf("reply envelope = %v", response)
	}
	reply := new(temporalv1.Reply)
	if err := response.GetReply().UnmarshalTo(reply); err != nil {
		t.Fatal(err)
	}
	return reply
}

func reset(t *testing.T, h *model.Host) {
	t.Helper()
	response, err := h.Reset(context.Background(), &modelv1.ResetRequest{})
	if err != nil || response == nil {
		t.Fatalf("Reset = %v, %v", response, err)
	}
}

func TestAnswerDeliveryAndObservedTerminal(t *testing.T) {
	for _, tc := range []struct {
		policy string
		status enumspb.ActivityExecutionStatus
		step   string
	}{
		{"Complete", enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, "Worker::RespondCompleted"},
		{"Fail", enumspb.ACTIVITY_EXECUTION_STATUS_FAILED, "Worker::RespondFailed"},
	} {
		t.Run(tc.policy, func(t *testing.T) {
			h := guest(t)
			explained(t, judge(t, h, scheduled(t)))
			explained(t, judge(t, h, scheduled(t)))
			reply := answer(t, h, tc.policy)
			if tc.policy == "Complete" {
				if reply.Request.GetTypeUrl() != "type.googleapis.com/temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest" {
					t.Fatalf("completion request = %v", reply)
				}
				request := new(workflowservice.RespondActivityTaskCompletedRequest)
				if err := reply.Request.UnmarshalTo(request); err != nil {
					t.Fatal(err)
				}
				want := &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Complete")}}}
				if !proto.Equal(request.Result, want) || len(request.TaskToken) != 0 || request.Identity != "" {
					t.Fatalf("partial completion = %v", request)
				}
			} else {
				if reply.Request.GetTypeUrl() != "type.googleapis.com/temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest" {
					t.Fatalf("failure request = %v", reply)
				}
				request := new(workflowservice.RespondActivityTaskFailedRequest)
				if err := reply.Request.UnmarshalTo(request); err != nil {
					t.Fatal(err)
				}
				failure := request.GetFailure()
				if failure.GetMessage() != "Fail" || failure.GetApplicationFailureInfo().GetType() != "MiniActivityFailure" || !failure.GetApplicationFailureInfo().GetNonRetryable() || len(request.TaskToken) != 0 || request.Identity != "" {
					t.Fatalf("partial non-retryable failure = %v", request)
				}
			}
			explained(t, judge(t, h, started(t)))
			explained(t, judge(t, h, started(t)))
			explained(t, judge(t, h, observation(t, tc.status, enumspb.PENDING_ACTIVITY_STATE_UNSPECIFIED)), tc.step)
			explained(t, judge(t, h, observation(t, tc.status, enumspb.PENDING_ACTIVITY_STATE_UNSPECIFIED)))
			_, err := h.Answer(context.Background(), &modelv1.AnswerRequest{Callback: callback(t, tc.policy)})
			refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
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
		{"started without delivery", nil, started, []string{"Client::Cancel", "Worker::Poll", "Deadline::ScheduleToStart"}},
		{"terminal without delivery", nil, func(t *testing.T) *anypb.Any { return observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0) }, []string{"Client::Cancel", "Worker::Poll", "Deadline::ScheduleToStart"}},
		{"stale scheduled after delivery", func(t *testing.T, h *model.Host) { answer(t, h, "Complete") }, scheduled, []string{"Client::Cancel", "Worker::RespondCompleted", "Worker::RespondFailed", "Deadline::StartToClose"}},
		{"opposite terminal after closure", func(t *testing.T, h *model.Host) {
			answer(t, h, "Complete")
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
			refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
			reset(t, h)
			explained(t, judge(t, h, scheduled(t)))
			answer(t, h, "Complete")
			explained(t, judge(t, h, started(t)))
		})
	}
}

func TestCallbackRefusalPreservesScheduled(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, *temporalv1.Callback)
		code   modelv1.RefusalCode
	}{
		{"missing task", func(_ *testing.T, c *temporalv1.Callback) { c.Task = nil }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"missing task URL", func(_ *testing.T, c *temporalv1.Callback) { c.Task.TypeUrl = "" }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"wrong task type", func(_ *testing.T, c *temporalv1.Callback) {
			c.Task.TypeUrl = "type.googleapis.com/temporal.api.workflowservice.v1.DescribeActivityExecutionResponse"
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"alternate task URL", func(_ *testing.T, c *temporalv1.Callback) {
			c.Task.TypeUrl = "other/temporal.api.workflowservice.v1.PollActivityTaskQueueResponse"
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"malformed task", func(_ *testing.T, c *temporalv1.Callback) { c.Task.Value = []byte{255} }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"missing input", func(t *testing.T, c *temporalv1.Callback) {
			c.Task = pack(t, &workflowservice.PollActivityTaskQueueResponse{})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"zero payloads", func(t *testing.T, c *temporalv1.Callback) {
			c.Task = pack(t, &workflowservice.PollActivityTaskQueueResponse{Input: &commonpb.Payloads{}})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"two payloads", func(t *testing.T, c *temporalv1.Callback) {
			c.Task = pack(t, &workflowservice.PollActivityTaskQueueResponse{Input: &commonpb.Payloads{Payloads: []*commonpb.Payload{{}, {}}}})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"unknown policy", func(t *testing.T, c *temporalv1.Callback) {
			value := new(temporalv1.Callback)
			if err := callback(t, "Retry").UnmarshalTo(value); err != nil {
				t.Fatal(err)
			}
			c.Task = value.Task
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"wrong encoding", func(t *testing.T, c *temporalv1.Callback) {
			c.Task = pack(t, &workflowservice.PollActivityTaskQueueResponse{Input: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte("Complete")}}}})
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := guest(t)
			value := new(temporalv1.Callback)
			if err := callback(t, "Complete").UnmarshalTo(value); err != nil {
				t.Fatal(err)
			}
			tc.mutate(t, value)
			_, err := h.Answer(context.Background(), &modelv1.AnswerRequest{Callback: pack(t, value)})
			refusal(t, err, tc.code)
			explained(t, judge(t, h, scheduled(t)))
			answer(t, h, "Complete")
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
			a.TypeUrl = "other/umpire.temporal.v1.Observation"
			return a
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"malformed envelope", func(t *testing.T) *anypb.Any { a := scheduled(t); a.Value = []byte{255}; return a }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"unsupported method", func(t *testing.T) *anypb.Any {
			return pack(t, &temporalv1.Observation{Method: "DescribeActivityExecution", Response: pack(t, &workflowservice.DescribeActivityExecutionResponse{})})
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"missing method", func(t *testing.T) *anypb.Any {
			return pack(t, &temporalv1.Observation{Response: pack(t, &workflowservice.DescribeActivityExecutionResponse{})})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"missing response", func(t *testing.T) *anypb.Any { return pack(t, &temporalv1.Observation{Method: describeMethod}) }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"missing response URL", func(t *testing.T) *anypb.Any {
			return pack(t, &temporalv1.Observation{Method: describeMethod, Response: &anypb.Any{}})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"wrong inner type", func(t *testing.T) *anypb.Any {
			return pack(t, &temporalv1.Observation{Method: describeMethod, Response: pack(t, &workflowservice.PollActivityTaskQueueResponse{})})
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"alternate inner URL", func(t *testing.T) *anypb.Any {
			return pack(t, &temporalv1.Observation{Method: describeMethod, Response: &anypb.Any{TypeUrl: "other/temporal.api.workflowservice.v1.DescribeActivityExecutionResponse"}})
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"malformed response", func(t *testing.T) *anypb.Any {
			return pack(t, &temporalv1.Observation{Method: describeMethod, Response: &anypb.Any{TypeUrl: "type.googleapis.com/temporal.api.workflowservice.v1.DescribeActivityExecutionResponse", Value: []byte{255}}})
		}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"missing info", func(t *testing.T) *anypb.Any {
			return pack(t, &temporalv1.Observation{Method: describeMethod, Response: pack(t, &workflowservice.DescribeActivityExecutionResponse{})})
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
			answer(t, h, "Complete")
			explained(t, judge(t, h, started(t)))
		})
	}
}

func TestCallbackEnvelopeRefusalPreservesScheduled(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value func(*testing.T) *anypb.Any
		code  modelv1.RefusalCode
	}{
		{"wrong outer type", scheduled, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"alternate outer URL", func(t *testing.T) *anypb.Any {
			a := callback(t, "Complete")
			a.TypeUrl = "other/umpire.temporal.v1.Callback"
			return a
		}, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"malformed envelope", func(t *testing.T) *anypb.Any { a := callback(t, "Complete"); a.Value = []byte{255}; return a }, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := guest(t)
			_, err := h.Answer(context.Background(), &modelv1.AnswerRequest{Callback: tc.value(t)})
			refusal(t, err, tc.code)
			explained(t, judge(t, h, scheduled(t)))
			answer(t, h, "Complete")
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
				answer(t, h, "Complete")
			}
			want := []string{"Client::Cancel", "Worker::Poll", "Deadline::ScheduleToStart"}
			if tc.delivered {
				want = []string{"Client::Cancel", "Worker::RespondCompleted", "Worker::RespondFailed", "Deadline::StartToClose"}
			}
			unexplained(t, judge(t, h, observation(t, tc.status, 0)), want...)
			_, err := h.Answer(context.Background(), &modelv1.AnswerRequest{Callback: callback(t, "Complete")})
			refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
		})
	}
}

func TestRepeatedDeliveryAndInstanceIsolation(t *testing.T) {
	first, second := guest(t), guest(t)
	answer(t, first, "Complete")
	_, err := first.Answer(context.Background(), &modelv1.AnswerRequest{Callback: callback(t, "Fail")})
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
	explained(t, judge(t, first, started(t)))
	explained(t, judge(t, second, scheduled(t)))
	answer(t, second, "Fail")
	explained(t, judge(t, second, observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_FAILED, 0)), "Worker::RespondFailed")
	reset(t, second)
	explained(t, judge(t, second, scheduled(t)))
	explained(t, judge(t, first, started(t)))
	reset(t, first)
	explained(t, judge(t, first, scheduled(t)))
}
