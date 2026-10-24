package temporal_test

import (
	"bytes"
	"context"
	"testing"

	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	model "umpire.local/model/host"
	annotationsv1 "umpire.local/model/gen/umpire/annotations/v1"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
)

func TestAnnotationsPreserveFrozenModelAndActualResponseBytes(t *testing.T) {
	h := guest(t)
	frozen := judge(t, h, observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0))
	unexplained(t, frozen, "Client::Cancel", "Worker::Poll(1)", "Worker::Poll(2)", "Worker::Poll(3)", "Deadline::ScheduleToStart")
	message, err := proto.Marshal(&workflowservice.DescribeActivityExecutionResponse{
		Info: &activitypb.ActivityExecutionInfo{Status: 999},
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := model.Annotate(context.Background(), h, &annotationsv1.AnnotationRequest{
		Method: describeMethod, Response: true, Message: message,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(response.Message, message) || len(response.Violations) == 0 {
		t.Fatalf("response bytes changed or finding missing: %v", response.Violations)
	}
	response, err = model.Annotate(context.Background(), h, &annotationsv1.AnnotationRequest{
		Method: "/temporal.api.workflowservice.v1.WorkflowService/PollActivityTaskQueue", Response: true,
	})
	if err != nil || len(response.GetViolations()) != 0 {
		t.Fatalf("empty successful poll: %v, %v", response.GetViolations(), err)
	}
	after := judge(t, h, scheduled(t))
	if !proto.Equal(frozen, after) {
		t.Fatal("annotation changed the frozen Model judgment")
	}
}

func TestAnnotationRefusalsLeaveWasmHostUsable(t *testing.T) {
	h := guest(t)
	for _, tc := range []struct {
		method string
		bytes  []byte
		code   modelv1.RefusalCode
	}{
		{"UnknownRpc", []byte("private-task-token"), modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"PollActivityTaskQueue", []byte{255}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
	} {
		_, err := model.Annotate(context.Background(), h, &annotationsv1.AnnotationRequest{
			Method: tc.method, Response: true, Message: tc.bytes,
		})
		refusal(t, err, tc.code)
	}
	_, err := h.Annotate(context.Background(), nil)
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST)
	explained(t, judge(t, h, scheduled(t)))
}
