package model

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/umpire/model/v1"
)

func TestProposalWireContract(t *testing.T) {
	if modelv1.Method_METHOD_PROPOSE != 6 || modelv1.Method_METHOD_ANNOTATE != 7 || modelv1.RefusalCode_REFUSAL_CODE_QUIESCENT != 9 {
		t.Fatal("protocol numbers changed")
	}
	opaque := &anypb.Any{TypeUrl: "x", Value: []byte{1, 2}}
	wants := map[string]proto.Message{
		"propose_request":     &modelv1.ProposeRequest{Seed: 150},
		"reset_configuration": &modelv1.ResetRequest{Configuration: opaque},
		"propose_action":      &modelv1.Response{Outcome: &modelv1.Response_Propose{Propose: &modelv1.ProposeResponse{Request: opaque, Action: "Act", Kind: modelv1.ProposalKind_PROPOSAL_KIND_ACTION}}},
		"propose_bootstrap":   &modelv1.Response{Outcome: &modelv1.Response_Propose{Propose: &modelv1.ProposeResponse{Request: opaque, Kind: modelv1.ProposalKind_PROPOSAL_KIND_BOOTSTRAP}}},
		"judge_poll":          &modelv1.Response{Outcome: &modelv1.Response_Judge{Judge: &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Explained{Explained: &modelv1.Explanation{}}, ContinuePolling: true}}},
		"refusal_quiescent":   &modelv1.Response{Outcome: &modelv1.Response_Refusal{Refusal: &modelv1.Refusal{Code: modelv1.RefusalCode_REFUSAL_CODE_QUIESCENT, Message: "x"}}},
	}
	wants["judge_poll_go"] = wants["judge_poll"]
	data, err := os.ReadFile("../../proto/tests/proposal-wire.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		name, wire, ok := strings.Cut(line, " ")
		if !ok || wants[name] == nil {
			t.Fatal(line)
		}
		if name == "judge_poll" {
			bytes, err := hex.DecodeString(wire)
			if err != nil {
				t.Fatal(err)
			}
			decoded := new(modelv1.Response)
			if err := proto.Unmarshal(bytes, decoded); err != nil || !proto.Equal(decoded, wants[name]) {
				t.Fatalf("Rust polling wire: %v", err)
			}
			continue
		}
		encoded, err := proto.Marshal(wants[name])
		if err != nil || hex.EncodeToString(encoded) != wire {
			t.Fatalf("%s: %x %v want %s", name, encoded, err, wire)
		}
		decoded := wants[name].ProtoReflect().New().Interface()
		if err := proto.Unmarshal(encoded, decoded); err != nil || !proto.Equal(decoded, wants[name]) {
			t.Fatalf("%s roundtrip: %v", name, err)
		}
	}
}
func TestProposalKindsFailClosed(t *testing.T) {
	for _, kind := range []modelv1.ProposalKind{modelv1.ProposalKind_PROPOSAL_KIND_UNSPECIFIED, 99} {
		if err := validateProposal(&modelv1.ProposeResponse{Request: controlAny(0), Kind: kind}); err == nil {
			t.Fatalf("accepted kind %v", kind)
		}
	}
}
func TestControlProposalBootstrapPollingAndCallbackParity(t *testing.T) {
	ctx := context.Background()
	host := statefulHost(t)
	if _, err := host.Reset(ctx, &modelv1.ResetRequest{Configuration: controlAny(0)}); err != nil {
		t.Fatal(err)
	}
	request := &modelv1.ProposeRequest{Seed: 7}
	bootstrap, err := host.Propose(ctx, request)
	if err != nil || bootstrap.GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_BOOTSTRAP || bootstrap.GetAction() != "" || !proto.Equal(bootstrap.GetRequest(), controlAny(0)) {
		t.Fatalf("bootstrap: %v %v", bootstrap, err)
	}
	_, err = host.Answer(ctx, &modelv1.AnswerRequest{Callback: controlAny(4)})
	var refusal *RefusalError
	if !errors.As(err, &refusal) || refusal.Code != modelv1.RefusalCode_REFUSAL_CODE_STATE {
		t.Fatalf("pre-admission callback: %v", err)
	}
	controlJudge(t, host, 0)
	proposal, err := host.Propose(ctx, request)
	if err != nil || proposal.GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_ACTION || proposal.GetAction() != "Deliver" || !proto.Equal(proposal.GetRequest(), controlAny(1)) {
		t.Fatalf("action: %v %v", proposal, err)
	}
	poll, err := host.Judge(ctx, &modelv1.JudgeRequest{Observation: controlAny(20)})
	if err != nil || !poll.GetContinuePolling() {
		t.Fatalf("intermediate: %v %v", poll, err)
	}
	if _, err := host.Answer(ctx, &modelv1.AnswerRequest{Callback: controlAny(4)}); err != nil {
		t.Fatal(err)
	}
	_, err = host.Propose(ctx, request)
	if !errors.As(err, &refusal) || refusal.Code != modelv1.RefusalCode_REFUSAL_CODE_STATE {
		t.Fatalf("pending: %v", err)
	}
	final, err := host.Judge(ctx, &modelv1.JudgeRequest{Observation: controlAny(2)})
	if err != nil || final.GetContinuePolling() || final.GetExplained() == nil {
		t.Fatalf("final: %v %v", final, err)
	}
	next, err := host.Propose(ctx, request)
	if err != nil || next.GetAction() != "Finish" {
		t.Fatalf("next: %v %v", next, err)
	}
	controlJudge(t, host, 3, "Finish")
	_, err = host.Propose(ctx, request)
	if !errors.As(err, &refusal) || refusal.Code != modelv1.RefusalCode_REFUSAL_CODE_QUIESCENT {
		t.Fatalf("quiescent: %v", err)
	}
	if _, err := host.Reset(ctx, &modelv1.ResetRequest{}); err != nil {
		t.Fatal(err)
	}
	again, err := host.Propose(ctx, request)
	if err != nil || !proto.Equal(again, bootstrap) {
		t.Fatalf("Reset bootstrap: %v %v", again, err)
	}
}
