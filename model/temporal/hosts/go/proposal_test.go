package temporal_test

import (
	"context"
	"testing"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	model "umpire.local/model/host"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func configuredGuest(t *testing.T, seed uint64) (*model.Host, *modelv1.ProposeResponse) {
	t.Helper()
	h := unboundGuest(t)
	captured := new(realizationv1.Observation)
	if err := startObservation(t, 3).UnmarshalTo(captured); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Reset(context.Background(), &modelv1.ResetRequest{Configuration: captured.GetRequest()}); err != nil {
		t.Fatal(err)
	}
	bootstrap, err := h.Propose(context.Background(), &modelv1.ProposeRequest{Seed: seed})
	if err != nil || bootstrap.GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_BOOTSTRAP || bootstrap.GetAction() != "" {
		t.Fatalf("bootstrap: %v %v", bootstrap, err)
	}
	captured.Request = bootstrap.GetRequest()
	explained(t, judge(t, h, pack(t, captured)))
	return h, bootstrap
}

func TestConfiguredProposalNativeWasmParity(t *testing.T) {
	ctx := context.Background()
	h, bootstrap := configuredGuest(t, 1)
	seed := &modelv1.ProposeRequest{Seed: 1}
	poll, err := h.Propose(ctx, seed)
	if err != nil || poll.GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_READ || poll.GetAction() != "" {
		t.Fatalf("read: %v %v", poll, err)
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
	_, err = h.Propose(ctx, seed)
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
	result := judge(t, h, started(t))
	explained(t, result, "Worker::Poll(1)")
	if result.GetContinuePolling() {
		t.Fatalf("delivery read is final: %v", result)
	}
	response, err := h.Propose(ctx, seed)
	if err != nil || response.GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_READ || response.GetAction() != "" {
		t.Fatalf("response read: %v %v", response, err)
	}
	if current := judge(t, h, started(t)); !current.GetContinuePolling() {
		t.Fatalf("an unchanged Started read must continue: %v", current)
	}
	result = judge(t, h, observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0))
	explained(t, result, "Worker::RespondCompleted")
	if result.GetContinuePolling() {
		t.Fatalf("response is final: %v", result)
	}
	cancel, err := h.Propose(ctx, seed)
	if err != nil || cancel.GetAction() != "Client::Cancel" {
		t.Fatalf("enabled rejection: %v %v", cancel, err)
	}
	result = judge(t, h, cancelObservation(t, 5))
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
	captured := new(realizationv1.Observation)
	if err := startObservation(t, 3).UnmarshalTo(captured); err != nil {
		t.Fatal(err)
	}
	captured.Request = &anypb.Any{TypeUrl: bootstrap.GetRequest().TypeUrl, Value: []byte{255}}
	_, err = h.Judge(ctx, &modelv1.JudgeRequest{Observation: pack(t, captured)})
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST)
}

func TestProposalReviewedRaces(t *testing.T) {
	for _, scenario := range []string{"retry before first Started read", "Describe cannot finish Cancel"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			h, _ := configuredGuest(t, 1)
			if scenario == "retry before first Started read" {
				if _, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: 1}); err != nil {
					t.Fatal(err)
				}
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
				result := judge(t, h, pack(t, observed))
				explained(t, result, "Worker::Poll(1)", "Worker::RespondFailed { retryable: true }")
				if result.GetContinuePolling() {
					t.Fatalf("validated retry must finish prior read: %v", result)
				}
				next, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: 1})
				if err != nil || next.GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_READ || next.GetAction() != "" {
					t.Fatalf("next attempt: %v %v", next, err)
				}
			} else {
				deliver(t, h)
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

func TestCancelHoldIsSeededWithoutChangingTheAction(t *testing.T) {
	for _, seed := range []uint64{0, 2} {
		h, _ := configuredGuest(t, 0)
		proposal, err := h.Propose(context.Background(), &modelv1.ProposeRequest{Seed: seed})
		if err != nil || proposal.GetAction() != "Client::Cancel" || proposal.GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_ACTION {
			t.Fatalf("cancel: %v %v", proposal, err)
		}
		instruction := new(temporalv1.TemporalInstruction)
		if err := proposal.GetRequest().UnmarshalTo(instruction); err != nil {
			t.Fatal(err)
		}
		request := new(workflowservice.RequestCancelActivityExecutionRequest)
		if err := instruction.GetRequest().UnmarshalTo(request); err != nil {
			t.Fatal(err)
		}
		if instruction.GetHoldWorkerPolls() != (seed == 2) || request.GetNamespace() != receiptNamespace || request.GetActivityId() != receiptActivity || request.GetRunId() != receiptRun {
			t.Fatalf("cancel instruction: %v", instruction)
		}
	}
}
