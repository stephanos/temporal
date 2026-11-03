package worker

import (
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func programBytes(t *testing.T, p *temporalv1.WorkerProgram) []byte {
	t.Helper()
	a, err := anypb.New(p)
	if err != nil {
		t.Fatal(err)
	}
	b, err := proto.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestProgramValidationAndExactMatch(t *testing.T) {
	key, _ := anypb.New(&temporalv1.ActivityTaskKey{Scenario: "Complete", Attempt: 1, ActivityType: "test"})
	response, _ := anypb.New(&workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("first")}}}})
	second, _ := anypb.New(&workflowservice.RespondActivityTaskFailedRequest{})
	secondKey, _ := anypb.New(&temporalv1.ActivityTaskKey{Scenario: "Fail", Attempt: 1, ActivityType: "test"})
	p := &temporalv1.WorkerProgram{Namespace: "ns", TaskQueue: "q", Rules: []*temporalv1.Rule{{Key: key, Respond: []*anypb.Any{response}}, {Key: secondKey, Respond: []*anypb.Any{second}}}}
	table, err := Decode(programBytes(t, p))
	if err != nil {
		t.Fatal(err)
	}
	got := table.Match("Complete", 1, "test")
	if got == nil || !proto.Equal(got, response) {
		t.Fatalf("response = %v", got)
	}
	if table.Match("Complete", 2, "test") != nil || table.Match("Complete", 1, "other") != nil {
		t.Fatal("non-exact key matched")
	}
	p.Rules[1].Key = &anypb.Any{TypeUrl: "type.googleapis.com/unknown.Key"}
	if _, err := Decode(programBytes(t, p)); err == nil {
		t.Fatal("unknown key accepted at startup")
	}
	p.Rules = nil
	if _, err := Decode(programBytes(t, p)); err != nil {
		t.Fatalf("empty program: %v", err)
	}
}

func TestProgramRejectsDuplicateExactKey(t *testing.T) {
	key, err := anypb.New(&temporalv1.ActivityTaskKey{Scenario: "Complete", Attempt: 1, ActivityType: "test"})
	if err != nil {
		t.Fatal(err)
	}
	response, err := anypb.New(&workflowservice.RespondActivityTaskCompletedRequest{})
	if err != nil {
		t.Fatal(err)
	}
	p := &temporalv1.WorkerProgram{Namespace: "ns", TaskQueue: "q", Rules: []*temporalv1.Rule{{Key: key, Respond: []*anypb.Any{response}}, {Key: key, Respond: []*anypb.Any{response}}}}
	if _, err := Decode(programBytes(t, p)); err == nil {
		t.Fatal("duplicate exact key accepted")
	}
}
