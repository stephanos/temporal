package raw

import (
	"context"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"testing"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

type service struct {
	workflowservice.WorkflowServiceClient
	cancel    context.CancelFunc
	task      *workflowservice.PollActivityTaskQueueResponse
	completed *workflowservice.RespondActivityTaskCompletedRequest
	failed    *workflowservice.RespondActivityTaskFailedRequest
}

func (s *service) PollActivityTaskQueue(ctx context.Context, _ *workflowservice.PollActivityTaskQueueRequest, _ ...grpc.CallOption) (*workflowservice.PollActivityTaskQueueResponse, error) {
	if s.task != nil {
		task := s.task
		s.task = nil
		return task, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s *service) RespondActivityTaskCompleted(_ context.Context, r *workflowservice.RespondActivityTaskCompletedRequest, _ ...grpc.CallOption) (*workflowservice.RespondActivityTaskCompletedResponse, error) {
	s.completed = r
	s.cancel()
	return &workflowservice.RespondActivityTaskCompletedResponse{}, nil
}
func (s *service) RespondActivityTaskFailed(_ context.Context, r *workflowservice.RespondActivityTaskFailedRequest, _ ...grpc.CallOption) (*workflowservice.RespondActivityTaskFailedResponse, error) {
	s.failed = r
	s.cancel()
	return &workflowservice.RespondActivityTaskFailedResponse{}, nil
}
func TestFollowExactResponseAndUnprogrammed(t *testing.T) {
	for _, matched := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		task := &workflowservice.PollActivityTaskQueueResponse{TaskToken: []byte("token"), Attempt: 1, ActivityType: &commonpb.ActivityType{Name: "activity"}, Header: &commonpb.Header{Fields: map[string]*commonpb.Payload{"umpire-scenario": {Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Complete")}}}}
		result := &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("exact")}}}
		key, _ := anypb.New(&temporalv1.ActivityTaskKey{Scenario: "Complete", Attempt: 1, ActivityType: "activity"})
		response, _ := anypb.New(&workflowservice.RespondActivityTaskCompletedRequest{Result: result})
		p := &temporalv1.WorkerProgram{Namespace: "ns", TaskQueue: "q"}
		if matched {
			p.Rules = []*temporalv1.Rule{{Key: key, Respond: []*anypb.Any{response}}}
		}
		envelope, _ := anypb.New(p)
		data, _ := proto.Marshal(envelope)
		svc := &service{cancel: cancel, task: task}
		err := Follow(ctx, svc, data)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		if matched {
			if !proto.Equal(svc.completed.GetResult(), result) || string(svc.completed.GetTaskToken()) != "token" {
				t.Fatalf("completion = %v", svc.completed)
			}
		} else {
			failure := svc.failed.GetFailure().GetApplicationFailureInfo()
			if failure.GetType() != "umpire.unprogrammed" || !failure.GetNonRetryable() {
				t.Fatalf("failure = %v", svc.failed)
			}
		}
	}
}
