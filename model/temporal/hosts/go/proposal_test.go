package temporal_test

import (
	"context"
	"errors"
	"testing"

	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	model "umpire.local/model/host"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func configuredGuest(t *testing.T, seed uint64) (*model.Host, *modelv1.ProposeResponse, *temporalv1.WorkerProgram) {
	t.Helper()
	h := unboundGuest(t)
	captured := new(realizationv1.Observation)
	if err := startObservation(t, 3).UnmarshalTo(captured); err != nil {
		t.Fatal(err)
	}
	reset, err := h.Reset(context.Background(), &modelv1.ResetRequest{Configuration: captured.GetRequest()})
	if err != nil {
		t.Fatal(err)
	}
	program := new(temporalv1.WorkerProgram)
	if err := reset.GetProgram().UnmarshalTo(program); err != nil {
		t.Fatal(err)
	}
	bootstrap, err := h.Propose(context.Background(), &modelv1.ProposeRequest{Seed: seed})
	if err != nil || bootstrap.GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_BOOTSTRAP || bootstrap.GetAction() != "" {
		t.Fatalf("bootstrap: %v %v", bootstrap, err)
	}
	captured.Request = bootstrap.GetRequest()
	explained(t, judge(t, h, pack(t, captured)))
	return h, bootstrap, program
}

func programResponse(t *testing.T, program *temporalv1.WorkerProgram, scenario string, attempt int32) *anypb.Any {
	t.Helper()
	for _, rule := range program.GetRules() {
		key := new(temporalv1.ActivityTaskKey)
		if err := rule.GetKey().UnmarshalTo(key); err != nil {
			t.Fatal(err)
		}
		if key.GetScenario() == scenario && key.GetAttempt() == attempt {
			if len(rule.GetRespond()) != 1 {
				t.Fatalf("program row: %v", rule)
			}
			return rule.GetRespond()[0]
		}
	}
	t.Fatalf("program has no %s attempt %d row", scenario, attempt)
	return nil
}

func configuredObservation(t *testing.T, request *anypb.Any, value *anypb.Any, program *temporalv1.WorkerProgram, scenario string, attempt int32) *anypb.Any {
	t.Helper()
	observed := new(realizationv1.Observation)
	if err := value.UnmarshalTo(observed); err != nil {
		t.Fatal(err)
	}
	response := new(workflowservice.DescribeActivityExecutionResponse)
	if err := observed.GetResponse().UnmarshalTo(response); err != nil {
		t.Fatal(err)
	}
	observed.Request = proto.Clone(request).(*anypb.Any)
	response.Info.Attempt = attempt
	if attempt > 1 {
		failed := new(workflowservice.RespondActivityTaskFailedRequest)
		if err := programResponse(t, program, scenario, attempt-1).UnmarshalTo(failed); err != nil {
			t.Fatal(err)
		}
		response.Info.LastFailure = failed.GetFailure()
	}
	if response.GetInfo().GetStatus() == enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED {
		completed := new(workflowservice.RespondActivityTaskCompletedRequest)
		if err := programResponse(t, program, scenario, attempt).UnmarshalTo(completed); err != nil {
			t.Fatal(err)
		}
		response.Outcome = &activitypb.ActivityExecutionOutcome{Value: &activitypb.ActivityExecutionOutcome_Result{Result: completed.GetResult()}}
	}
	observed.Result = &realizationv1.Observation_Response{Response: pack(t, response)}
	return pack(t, observed)
}

func TestConfiguredProposalNativeWasmParity(t *testing.T) {
	ctx := context.Background()
	h, bootstrap, program := configuredGuest(t, 0)
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
	if !read.GetIncludeOutcome() || !read.GetIncludeLastFailure() || read.GetRunId() != receiptRun || instruction.GetPollIntervalMs() != 10 || instruction.GetReadTimeoutMs() != 30000 || instruction.GetBackoffMs() != 10 {
		t.Fatalf("instruction: %v", instruction)
	}
	if result := judge(t, h, configuredObservation(t, instruction.GetRequest(), scheduled(t), program, "Complete", 1)); !result.GetContinuePolling() {
		t.Fatalf("scheduled read must continue: %v", result)
	}
	_, err = h.Propose(ctx, seed)
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
	result := judge(t, h, configuredObservation(t, instruction.GetRequest(), started(t), program, "Complete", 1))
	explained(t, result, "Worker::Poll(1)")
	if !result.GetContinuePolling() {
		t.Fatalf("Started progress must preserve pending read: %v", result)
	}
	_, err = h.Propose(ctx, seed)
	refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
	if current := judge(t, h, configuredObservation(t, instruction.GetRequest(), started(t), program, "Complete", 1)); !current.GetContinuePolling() {
		t.Fatalf("an unchanged Started read must continue: %v", current)
	}
	result = judge(t, h, configuredObservation(t, instruction.GetRequest(), observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0), program, "Complete", 1))
	explained(t, result, "Worker::Poll(1)", "Worker::RespondCompleted")
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
	again, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: 0})
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
			bootstrapSeed := uint64(0)
			if scenario == "retry before first Started read" {
				bootstrapSeed = 1
			}
			h, _, program := configuredGuest(t, bootstrapSeed)
			if scenario == "retry before first Started read" {
				read, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: 1})
				if err != nil {
					t.Fatal(err)
				}
				instruction := new(temporalv1.ObserveActivity)
				if err := read.GetRequest().UnmarshalTo(instruction); err != nil {
					t.Fatal(err)
				}
				observed := configuredObservation(t, instruction.GetRequest(), scheduled(t), program, "RetryThenComplete", 2)
				result := judge(t, h, observed)
				explained(t, result, "Worker::Poll(1)", "Worker::RespondFailed { retryable: true }")
				if !result.GetContinuePolling() {
					t.Fatalf("retry Scheduled progress must preserve pending read: %v", result)
				}
				_, err = h.Propose(ctx, &modelv1.ProposeRequest{Seed: 1})
				refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_STATE)
				result = judge(t, h, configuredObservation(t, instruction.GetRequest(), started(t), program, "RetryThenComplete", 2))
				explained(t, result, "Worker::Poll(1)", "Worker::RespondFailed { retryable: true }", "Worker::Poll(2)")
				if !result.GetContinuePolling() {
					t.Fatalf("retry Started progress must preserve pending read: %v", result)
				}
				result = judge(t, h, configuredObservation(t, instruction.GetRequest(), observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0), program, "RetryThenComplete", 2))
				explained(t, result, "Worker::Poll(1)", "Worker::RespondFailed { retryable: true }", "Worker::Poll(2)", "Worker::RespondCompleted")
				if result.GetContinuePolling() {
					t.Fatalf("terminal retry result must finish read: %v", result)
				}
				next, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: 1})
				if err != nil || next.GetAction() != "Client::Cancel" {
					t.Fatalf("post-terminal proposal: %v %v", next, err)
				}
			} else {
				cancel, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: 2})
				if err != nil || cancel.GetAction() != "Client::Cancel" {
					t.Fatalf("cancel: %v %v", cancel, err)
				}
				_, err = h.Judge(ctx, &modelv1.JudgeRequest{Observation: configuredObservation(t, pack(t, &workflowservice.DescribeActivityExecutionRequest{Namespace: receiptNamespace, ActivityId: receiptActivity, RunId: receiptRun, IncludeOutcome: true, IncludeLastFailure: true}), observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0), program, "Complete", 1)})
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

func TestOpenCancellationRequiresPollContainment(t *testing.T) {
	for _, seed := range []uint64{0, 2} {
		h, _, _ := configuredGuest(t, 0)
		proposal, err := h.Propose(context.Background(), &modelv1.ProposeRequest{Seed: seed})
		if seed == 0 {
			refusal(t, err, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED)
			var refused *model.RefusalError
			if proposal != nil || !errors.As(err, &refused) || refused.Message != "unheld cancellation of an open activity is outside the atomic cancellation Model" {
				t.Fatalf("unheld cancellation must refuse before issuing an instruction: %v %v", proposal, err)
			}
			continue
		}
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
