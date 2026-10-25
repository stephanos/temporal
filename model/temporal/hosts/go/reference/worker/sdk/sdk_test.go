package sdk

import (
	"context"
	"strings"
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestCompletionPreservesPayloadAndRefusesMultiple(t *testing.T) {
	payload := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("exact")}
	response, _ := anypb.New(&workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{payload}}})
	got, err := convert(response)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(payload, got.value.Payload()) {
		t.Fatal("payload changed")
	}
	response, _ = anypb.New(&workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{payload, payload}}})
	if _, err := convert(response); err == nil {
		t.Fatal("multiple payloads accepted")
	}
}

func TestUnknownKeyRefusedBeforeSDKStartup(t *testing.T) {
	p := &temporalv1.WorkerProgram{Namespace: "ns", TaskQueue: "q", Rules: []*temporalv1.Rule{{Key: &anypb.Any{TypeUrl: "type.googleapis.com/unknown.Key"}}}}
	a, err := anypb.New(p)
	if err != nil {
		t.Fatal(err)
	}
	data, err := proto.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	err = Follow(context.Background(), nil, data, nil)
	if err == nil || !strings.Contains(err.Error(), "key") {
		t.Fatalf("startup error = %v", err)
	}
}

type header struct{ payload *commonpb.Payload }

func (h header) Get(string) (*commonpb.Payload, bool)                   { return h.payload, true }
func (h header) ForEachKey(func(string, *commonpb.Payload) error) error { return nil }
func TestScenarioUsesHeader(t *testing.T) {
	p := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Fail")}
	ctx, err := ScenarioPropagator().Extract(context.Background(), header{payload: p})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := ctx.Value(scenarioKey{}).(string); got != "Fail" {
		t.Fatalf("scenario = %q", got)
	}
}
