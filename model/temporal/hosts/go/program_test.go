package temporal_test

import (
	"bytes"
	"context"
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestResetProgramRowsAndScenarioHeader(t *testing.T) {
	ctx := context.Background()
	for attempts := int32(1); attempts <= 3; attempts++ {
		h := unboundGuest(t)
		empty, err := h.Reset(ctx, &modelv1.ResetRequest{})
		if err != nil || empty.GetProgram() != nil {
			t.Fatalf("unconfigured Reset: %v %v", empty, err)
		}
		captured := new(realizationv1.Observation)
		if err := startObservation(t, attempts).UnmarshalTo(captured); err != nil {
			t.Fatal(err)
		}
		configured := new(workflowservice.StartActivityExecutionRequest)
		if err := captured.GetRequest().UnmarshalTo(configured); err != nil {
			t.Fatal(err)
		}
		configured.Input = &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("application input")}}}
		configuration := pack(t, configured)
		reset, err := h.Reset(ctx, &modelv1.ResetRequest{Configuration: configuration})
		if err != nil {
			t.Fatal(err)
		}
		program := new(temporalv1.WorkerProgram)
		if err := reset.GetProgram().UnmarshalTo(program); err != nil {
			t.Fatal(err)
		}
		if program.GetNamespace() != receiptNamespace || program.GetTaskQueue() != "queue" || len(program.GetRules()) != 3*int(attempts) {
			t.Fatalf("program: %v", program)
		}
		rows := map[string]map[int32]bool{}
		for _, rule := range program.GetRules() {
			key := new(temporalv1.ActivityTaskKey)
			if err := rule.GetKey().UnmarshalTo(key); err != nil {
				t.Fatal(err)
			}
			if key.GetActivityType() != "ModelSlice" || key.GetAttempt() < 1 || key.GetAttempt() > attempts || len(rule.GetRespond()) != 1 {
				t.Fatalf("row: %v", rule)
			}
			if rows[key.GetScenario()] == nil {
				rows[key.GetScenario()] = map[int32]bool{}
			}
			if rows[key.GetScenario()][key.GetAttempt()] {
				t.Fatalf("duplicate key %v", key)
			}
			rows[key.GetScenario()][key.GetAttempt()] = true
			response := rule.GetRespond()[0]
			if key.GetScenario() == "Fail" || key.GetScenario() == "RetryThenComplete" && key.GetAttempt() == 1 {
				request := new(workflowservice.RespondActivityTaskFailedRequest)
				if err := response.UnmarshalTo(request); err != nil {
					t.Fatal(err)
				}
				wantNonRetryable := key.GetScenario() == "Fail"
				wantMessage := "Fail"
				if !wantNonRetryable {
					wantMessage = "Retry"
				}
				if len(request.GetTaskToken()) != 0 || request.GetIdentity() != "" || request.GetFailure().GetMessage() != wantMessage || request.GetFailure().GetApplicationFailureInfo().GetType() != "MiniActivityFailure" || request.GetFailure().GetApplicationFailureInfo().GetNonRetryable() != wantNonRetryable {
					t.Fatalf("failure row %v: %v", key, request)
				}
			} else {
				request := new(workflowservice.RespondActivityTaskCompletedRequest)
				if err := response.UnmarshalTo(request); err != nil {
					t.Fatal(err)
				}
				want := &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Complete")}}}
				if len(request.GetTaskToken()) != 0 || request.GetIdentity() != "" || !proto.Equal(request.GetResult(), want) {
					t.Fatalf("completion row %v: %v", key, request)
				}
			}
		}
		for _, scenario := range []string{"Complete", "RetryThenComplete", "Fail"} {
			for attempt := int32(1); attempt <= attempts; attempt++ {
				if !rows[scenario][attempt] {
					t.Fatalf("missing %s attempt %d", scenario, attempt)
				}
			}
		}
		original, _ := proto.Marshal(reset)
		for seed := uint64(0); seed < 3; seed++ {
			again, err := h.Reset(ctx, &modelv1.ResetRequest{})
			if err != nil {
				t.Fatal(err)
			}
			repeated, _ := proto.Marshal(again)
			if !bytes.Equal(original, repeated) {
				t.Fatalf("same World changed program: %x -> %x", original, repeated)
			}
			proposal, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: seed})
			if err != nil {
				t.Fatal(err)
			}
			start := new(workflowservice.StartActivityExecutionRequest)
			if err := proposal.GetRequest().UnmarshalTo(start); err != nil {
				t.Fatal(err)
			}
			wantScenario := []string{"Complete", "RetryThenComplete", "Fail"}[seed]
			tag := start.GetHeader().GetFields()["umpire-scenario"]
			if tag == nil || string(tag.GetData()) != wantScenario || string(tag.GetMetadata()["encoding"]) != "binary/plain" || !proto.Equal(start.GetInput(), configured.GetInput()) {
				t.Fatalf("scenario header or application input: %v", start)
			}
		}
	}
}
