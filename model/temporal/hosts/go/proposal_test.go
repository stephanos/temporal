package temporal_test

import (
	"context"
	"testing"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/umpire/model/v1"
	realizationv1 "umpire.local/model/umpire/realization/v1"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestConfiguredProposalNativeWasmParity(t *testing.T) {
	ctx := context.Background()
	h := guest(t)
	captured := new(realizationv1.Observation)
	if err := startObservation(t, 3).UnmarshalTo(captured); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Reset(ctx, &modelv1.ResetRequest{Configuration: captured.GetRequest()}); err != nil {
		t.Fatal(err)
	}
	_, err := h.Answer(ctx, &modelv1.AnswerRequest{Callback: callback(t, "Complete")})
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
	seed := &modelv1.ProposeRequest{Seed: 1}
	bootstrap, err := h.Propose(ctx, seed)
	if err != nil || bootstrap.GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_BOOTSTRAP || bootstrap.GetAction() != "" || !proto.Equal(bootstrap.GetRequest(), captured.GetRequest()) {
		t.Fatalf("bootstrap: %v %v", bootstrap, err)
	}
	_, err = h.Propose(ctx, seed)
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
	explained(t, judge(t, h, startObservation(t, 3)))
	poll, err := h.Propose(ctx, seed)
	if err != nil || poll.GetAction() != "Worker::Poll(1)" {
		t.Fatalf("poll: %v %v", poll, err)
	}
	instruction := new(temporalv1.ObserveActivity)
	if err := poll.GetRequest().UnmarshalTo(instruction); err != nil {
		t.Fatal(err)
	}
	read := new(workflowservice.DescribeActivityExecutionRequest)
	if err := instruction.GetRequest().UnmarshalTo(read); err != nil {
		t.Fatal(err)
	}
	if read.GetRunId() != receiptRun || instruction.GetPollIntervalMs() != 10 || instruction.GetReadTimeoutMs() != 30000 || instruction.GetBackoffMs() != 10 {
		t.Fatalf("instruction: %v", instruction)
	}
	if result := judge(t, h, scheduled(t)); !result.GetContinuePolling() {
		t.Fatalf("scheduled read must continue: %v", result)
	}
	answer(t, h, "Complete")
	if result := judge(t, h, started(t)); result.GetContinuePolling() {
		t.Fatalf("delivery read is final: %v", result)
	}
	response, err := h.Propose(ctx, seed)
	if err != nil || response.GetAction() != "Worker::RespondCompleted" {
		t.Fatalf("response: %v %v", response, err)
	}
	if err := response.GetRequest().UnmarshalTo(new(temporalv1.ObserveActivity)); err != nil {
		t.Fatal(err)
	}
	if result := judge(t, h, started(t)); !result.GetContinuePolling() {
		t.Fatalf("response read must continue: %v", result)
	}
	if result := judge(t, h, observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0)); result.GetContinuePolling() {
		t.Fatalf("response is final: %v", result)
	}
	cancel, err := h.Propose(ctx, seed)
	if err != nil || cancel.GetAction() != "Client::Cancel" {
		t.Fatalf("enabled rejection: %v %v", cancel, err)
	}
	result := judge(t, h, cancelObservation(t, 5))
	if result.GetContinuePolling() || result.GetExplained() == nil {
		t.Fatalf("rejection: %v", result)
	}
	if _, err := h.Reset(ctx, &modelv1.ResetRequest{}); err != nil {
		t.Fatal(err)
	}
	again, err := h.Propose(ctx, seed)
	if err != nil || !proto.Equal(again, bootstrap) {
		t.Fatalf("reused configuration: %v %v", again, err)
	}
	wrong := proto.Clone(captured).(*realizationv1.Observation)
	wrong.Request = &anypb.Any{TypeUrl: captured.Request.TypeUrl, Value: []byte{255}}
	_, err = h.Judge(ctx, &modelv1.JudgeRequest{Observation: pack(t, wrong)})
	if err == nil {
		t.Fatal("malformed bootstrap admitted")
	}
}

func TestProposalReviewedRaces(t *testing.T) {
	for _, scenario := range []string{"retry before first Started read", "Describe cannot finish Cancel"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			h := unboundGuest(t)
			start := new(realizationv1.Observation)
			if err := startObservation(t, 3).UnmarshalTo(start); err != nil {
				t.Fatal(err)
			}
			if _, err := h.Reset(ctx, &modelv1.ResetRequest{Configuration: start.GetRequest()}); err != nil {
				t.Fatal(err)
			}
			if _, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: 1}); err != nil {
				t.Fatal(err)
			}
			judge(t, h, startObservation(t, 3))
			if scenario == "retry before first Started read" {
				if _, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: 1}); err != nil {
					t.Fatal(err)
				}
				answer(t, h, "RetryThenComplete")
				observed := new(realizationv1.Observation)
				if err := scheduled(t).UnmarshalTo(observed); err != nil {
					t.Fatal(err)
				}
				response := new(workflowservice.DescribeActivityExecutionResponse)
				if err := observed.GetResponse().UnmarshalTo(response); err != nil {
					t.Fatal(err)
				}
				response.Info.Attempt = 2
				observed.Result = &realizationv1.Observation_Response{Response: pack(t, response)}
				if result := judge(t, h, pack(t, observed)); result.GetContinuePolling() {
					t.Fatalf("validated retry must finish prior delivery: %v", result)
				}
				next, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: 1})
				if err != nil || next.GetAction() != "Worker::Poll(2)" {
					t.Fatalf("next attempt: %v %v", next, err)
				}
			} else {
				answer(t, h, "Complete")
				cancel, err := h.Propose(ctx, &modelv1.ProposeRequest{})
				if err != nil || cancel.GetAction() != "Client::Cancel" {
					t.Fatalf("cancel: %v %v", cancel, err)
				}
				_, err = h.Judge(ctx, &modelv1.JudgeRequest{Observation: observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0)})
				refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
				_, err = h.Propose(ctx, &modelv1.ProposeRequest{})
				refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
				result := judge(t, h, cancelObservation(t, 0))
				if result.GetContinuePolling() {
					t.Fatalf("cancellation result must finish proposal: %v", result)
				}
				explained(t, result, "Client::Cancel")
			}
		})
	}
}
