package host

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
)

func TestProtocolResponseValidation(t *testing.T) {
	responses := []*modelv1.Response{
		{Outcome: &modelv1.Response_Explore{Explore: &modelv1.ExploreResponse{Report: "r"}}},
		{Outcome: &modelv1.Response_Rows{Rows: &modelv1.RowsResponse{Rows: []string{"x"}}}},
		{Outcome: &modelv1.Response_Reset_{Reset_: &modelv1.ResetResponse{}}},
		{Outcome: &modelv1.Response_Judge{Judge: &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Explained{Explained: &modelv1.Explanation{}}}}},
		{Outcome: &modelv1.Response_Answer{Answer: &modelv1.AnswerResponse{Reply: &anypb.Any{TypeUrl: "example.test/Reply", Value: []byte{1}}}}},
	}
	for method := modelv1.Method_METHOD_EXPLORE; method <= modelv1.Method_METHOD_ANSWER; method++ {
		for index, response := range responses {
			t.Run(fmt.Sprintf("%s/outcome%d", method, index+1), func(t *testing.T) {
				h := protocolControlHost(t, response)
				got, err := h.Call(context.Background(), method, nil)
				if int(method) == index+1 {
					if err != nil || !proto.Equal(got, response) {
						t.Fatalf("got %v, %v; want %v", got, err, response)
					}
				} else if err == nil || got != nil {
					t.Fatalf("mismatched method accepted: %v, %v", got, err)
				}
			})
		}
	}
	for _, tc := range []struct {
		name     string
		method   modelv1.Method
		response *modelv1.Response
	}{
		{"judge missing judgment", modelv1.Method_METHOD_JUDGE, &modelv1.Response{Outcome: &modelv1.Response_Judge{Judge: &modelv1.JudgeResponse{}}}},
		{"answer missing reply", modelv1.Method_METHOD_ANSWER, &modelv1.Response{Outcome: &modelv1.Response_Answer{Answer: &modelv1.AnswerResponse{}}}},
		{"answer empty reply type", modelv1.Method_METHOD_ANSWER, &modelv1.Response{Outcome: &modelv1.Response_Answer{Answer: &modelv1.AnswerResponse{Reply: &anypb.Any{}}}}},
		{"unknown refusal", modelv1.Method_METHOD_RESET, &modelv1.Response{Outcome: &modelv1.Response_Refusal{Refusal: &modelv1.Refusal{Code: modelv1.RefusalCode(99)}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := protocolControlHost(t, tc.response)
			got, err := h.Call(context.Background(), tc.method, nil)
			if err == nil || got != nil {
				t.Fatalf("invalid response accepted: %v, %v", got, err)
			}
			var refusal *RefusalError
			if tc.name == "unknown refusal" && errors.As(err, &refusal) {
				t.Fatalf("unknown enum exposed as valid refusal: %v", err)
			}
		})
	}
	for code := modelv1.RefusalCode(0); code <= modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED; code++ {
		t.Run(code.String(), func(t *testing.T) {
			h := protocolControlHost(t, &modelv1.Response{Outcome: &modelv1.Response_Refusal{Refusal: &modelv1.Refusal{Code: code, Message: "reason"}}})
			_, err := h.Call(context.Background(), modelv1.Method_METHOD_RESET, nil)
			var refusal *RefusalError
			if !errors.As(err, &refusal) || refusal.Code != code || refusal.Message != "reason" {
				t.Fatalf("refusal changed: %v", err)
			}
		})
	}
}

func TestProtocolWrappers(t *testing.T) {
	ctx := context.Background()
	opaque := &anypb.Any{TypeUrl: "example.test/Opaque", Value: []byte{0xff, 1}}
	for _, tc := range []struct {
		name     string
		request  proto.Message
		response *modelv1.Response
		call     func(*Host) (proto.Message, error)
		want     proto.Message
	}{
		{"reset", &modelv1.ResetRequest{}, &modelv1.Response{Outcome: &modelv1.Response_Reset_{Reset_: &modelv1.ResetResponse{}}},
			func(h *Host) (proto.Message, error) { return h.Reset(ctx, &modelv1.ResetRequest{}) }, &modelv1.ResetResponse{}},
		{"explained", &modelv1.JudgeRequest{Observation: opaque}, &modelv1.Response{Outcome: &modelv1.Response_Judge{Judge: &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Explained{Explained: &modelv1.Explanation{Steps: []string{"step"}}}}}},
			func(h *Host) (proto.Message, error) { return h.Judge(ctx, &modelv1.JudgeRequest{Observation: opaque}) }, &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Explained{Explained: &modelv1.Explanation{Steps: []string{"step"}}}}},
		{"unexplained", &modelv1.JudgeRequest{Observation: opaque}, &modelv1.Response{Outcome: &modelv1.Response_Judge{Judge: &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Unexplained{Unexplained: &modelv1.Unexplained{ExpectedActions: []string{"action"}}}}}},
			func(h *Host) (proto.Message, error) { return h.Judge(ctx, &modelv1.JudgeRequest{Observation: opaque}) }, &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Unexplained{Unexplained: &modelv1.Unexplained{ExpectedActions: []string{"action"}}}}},
		{"answer", &modelv1.AnswerRequest{Callback: opaque}, &modelv1.Response{Outcome: &modelv1.Response_Answer{Answer: &modelv1.AnswerResponse{Reply: opaque}}},
			func(h *Host) (proto.Message, error) { return h.Answer(ctx, &modelv1.AnswerRequest{Callback: opaque}) }, &modelv1.AnswerResponse{Reply: opaque}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := protocolControlHost(t, tc.response)
			got, err := tc.call(h)
			if err != nil || !proto.Equal(got, tc.want) {
				t.Fatalf("got %v, %v; want %v", got, err, tc.want)
			}
			wire, err := proto.Marshal(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			if len(wire) != 0 {
				written, ok := h.module.Memory().Read(1024, uint32(len(wire)))
				if !ok || string(written) != string(wire) {
					t.Fatalf("request bytes %x, want %x", written, wire)
				}
			}
		})
	}
	for _, tc := range []struct {
		name string
		call func(*Host) (proto.Message, error)
	}{
		{"nil reset", func(h *Host) (proto.Message, error) { return h.Reset(ctx, nil) }},
		{"nil judge", func(h *Host) (proto.Message, error) { return h.Judge(ctx, nil) }},
		{"missing observation", func(h *Host) (proto.Message, error) { return h.Judge(ctx, &modelv1.JudgeRequest{}) }},
		{"empty observation type", func(h *Host) (proto.Message, error) {
			return h.Judge(ctx, &modelv1.JudgeRequest{Observation: &anypb.Any{}})
		}},
		{"nil answer", func(h *Host) (proto.Message, error) { return h.Answer(ctx, nil) }},
		{"missing callback", func(h *Host) (proto.Message, error) { return h.Answer(ctx, &modelv1.AnswerRequest{}) }},
		{"empty callback type", func(h *Host) (proto.Message, error) {
			return h.Answer(ctx, &modelv1.AnswerRequest{Callback: &anypb.Any{}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.call(&Host{})
			var refusal *RefusalError
			if !errors.As(err, &refusal) || refusal.Code != modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST {
				t.Fatalf("invalid request did not refuse before calling guest: %v", err)
			}
		})
	}
}

func protocolControlHost(t *testing.T, response *modelv1.Response) *Host {
	t.Helper()
	payload, err := proto.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	h, err := New(context.Background(), controlModule(2048, uint32(len(payload)), payload, false, false, ""))
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
