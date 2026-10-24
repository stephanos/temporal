package host

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
)

func statefulHost(t *testing.T) *Host {
	t.Helper()
	path := os.Getenv("UMPIRE_STATEFUL_WASM")
	if path == "" {
		target := os.Getenv("CARGO_TARGET_DIR")
		if target == "" {
			target = "../../../../target"
		}
		path = filepath.Join(target, "wasm32-wasip1", "guest", "stateful_control.wasm")
	}
	wasm, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	host, err := New(context.Background(), wasm)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := host.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return host
}

func controlAny(value byte) *anypb.Any {
	return &anypb.Any{TypeUrl: "type.googleapis.com/umpire.control.Value", Value: []byte{value}}
}

func controlJudge(t *testing.T, host *Host, value byte, steps ...string) {
	t.Helper()
	response, err := host.Judge(context.Background(), &modelv1.JudgeRequest{Observation: controlAny(value)})
	if err != nil || response.GetExplained() == nil || !slices.Equal(response.GetExplained().Steps, steps) {
		t.Fatalf("Judge(%d): %v %v, want explained steps %v", value, response, err, steps)
	}
	outcomes := make([]string, len(steps))
	for i, step := range steps {
		switch step {
		case "Deliver":
			outcomes[i] = "Outcome(1)"
		case "Respond":
			outcomes[i] = "Outcome(2)"
		case "Finish":
			outcomes[i] = "Outcome(3)"
		default:
			t.Fatalf("unknown control step %q", step)
		}
	}
	if !slices.Equal(response.GetExplained().Outcomes, outcomes) {
		t.Fatalf("outcomes = %v, want %v", response.GetExplained().Outcomes, outcomes)
	}
}

func TestStatefulGuestReplayFreezeResetAndIsolation(t *testing.T) {
	ctx := context.Background()
	host, other := statefulHost(t), statefulHost(t)
	for range 2 {
		request, err := proto.Marshal(&modelv1.JudgeRequest{Observation: controlAny(0)})
		if err != nil {
			t.Fatal(err)
		}
		response, err := host.Call(ctx, modelv1.Method_METHOD_JUDGE, request)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := proto.Marshal(response)
		if err != nil || !bytes.Equal(encoded, []byte{0x2a, 0x02, 0x0a, 0x00}) {
			t.Fatalf("positive zero-step wire response: %x %v", encoded, err)
		}
	}
	for _, tc := range []struct {
		callback byte
		code     modelv1.RefusalCode
	}{
		{5, modelv1.RefusalCode_REFUSAL_CODE_STATE},
		{6, modelv1.RefusalCode_REFUSAL_CODE_ALLOCATION},
		{7, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{8, modelv1.RefusalCode_REFUSAL_CODE_STATE},
	} {
		_, err := host.Answer(ctx, &modelv1.AnswerRequest{Callback: controlAny(tc.callback)})
		var refusal *RefusalError
		if !errors.As(err, &refusal) || refusal.Code != tc.code {
			t.Fatalf("Answer(%d): %v, want %v", tc.callback, err, tc.code)
		}
		controlJudge(t, host, 0)
	}
	answer, err := host.Answer(ctx, &modelv1.AnswerRequest{Callback: controlAny(4)})
	if err != nil || !proto.Equal(answer.GetReply(), controlAny(42)) {
		t.Fatalf("Answer: %v %v", answer, err)
	}
	controlJudge(t, host, 1)
	controlJudge(t, host, 2, "Respond")
	for _, value := range []byte{1, 3} {
		response, err := host.Judge(ctx, &modelv1.JudgeRequest{Observation: controlAny(value)})
		if err != nil || response.GetUnexplained() == nil || !slices.Equal(response.GetUnexplained().ExpectedActions, []string{"Finish"}) {
			t.Fatalf("frozen Judge: %v %v", response, err)
		}
	}
	unknown, err := proto.Marshal(&modelv1.JudgeRequest{Observation: &anypb.Any{TypeUrl: "unknown", Value: []byte{1}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method  modelv1.Method
		request []byte
		code    modelv1.RefusalCode
	}{
		{modelv1.Method_METHOD_JUDGE, []byte{255}, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{modelv1.Method(999), nil, modelv1.RefusalCode_REFUSAL_CODE_UNKNOWN_METHOD},
		{modelv1.Method_METHOD_JUDGE, unknown, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
	} {
		_, err := host.Call(ctx, tc.method, tc.request)
		var refusal *RefusalError
		if !errors.As(err, &refusal) || refusal.Code != tc.code {
			t.Fatalf("frozen refusal: %v, want %v", err, tc.code)
		}
	}
	frozen, err := host.Judge(ctx, &modelv1.JudgeRequest{Observation: controlAny(3)})
	if err != nil || frozen.GetUnexplained() == nil || !slices.Equal(frozen.GetUnexplained().ExpectedActions, []string{"Finish"}) {
		t.Fatalf("refusal changed frozen marker: %v %v", frozen, err)
	}
	if _, err := host.Reset(ctx, &modelv1.ResetRequest{}); err != nil {
		t.Fatal(err)
	}
	controlJudge(t, host, 0)
	controlJudge(t, other, 0)
	controlJudge(t, other, 10, "Deliver", "Respond")
	controlJudge(t, host, 0)
	controlJudge(t, other, 3, "Finish")
	closed, err := other.Judge(ctx, &modelv1.JudgeRequest{Observation: controlAny(0)})
	if err != nil || closed.GetUnexplained() == nil || len(closed.GetUnexplained().ExpectedActions) != 0 {
		t.Fatalf("closed state expected actions: %v %v", closed, err)
	}
}

func TestStatefulResponseExhaustionPreservesRunAndOwnership(t *testing.T) {
	for _, method := range []modelv1.Method{modelv1.Method_METHOD_JUDGE, modelv1.Method_METHOD_ANSWER, modelv1.Method_METHOD_RESET} {
		t.Run(method.String(), func(t *testing.T) {
			ctx := context.Background()
			host := statefulHost(t)
			var request proto.Message
			var current byte
			switch method {
			case modelv1.Method_METHOD_JUDGE:
				request = &modelv1.JudgeRequest{Observation: controlAny(1)}
			case modelv1.Method_METHOD_ANSWER:
				request = &modelv1.AnswerRequest{Callback: controlAny(4)}
			case modelv1.Method_METHOD_RESET:
				request = &modelv1.ResetRequest{}
				controlJudge(t, host, 1, "Deliver")
				current = 1
			}
			data, err := proto.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			var allocations [][2]uint64
			allocate := func(size uint64) uint64 {
				t.Helper()
				result, err := host.alloc.Call(ctx, size)
				if err != nil || len(result) != 1 || result[0] == 0 {
					t.Fatalf("alloc(%d): %v %v", size, result, err)
				}
				allocations = append(allocations, [2]uint64{result[0], size})
				return result[0]
			}
			var pointer uint64
			if len(data) != 0 {
				pointer = allocate(uint64(len(data)))
				if !host.module.Memory().Write(uint32(pointer), data) {
					t.Fatal("write request")
				}
			}
			for range 15 {
				allocate(maxBuffer)
			}
			allocate(maxBuffer - uint64(len(data)))
			result, err := host.call.Call(ctx, uint64(method), pointer, uint64(len(data)))
			if err != nil || len(result) != 1 || result[0] != 0 {
				t.Fatalf("exhausted call: %v %v", result, err)
			}
			for _, allocation := range allocations {
				if _, err := host.free.Call(ctx, allocation[0], allocation[1]); err != nil {
					t.Fatal(err)
				}
			}
			controlJudge(t, host, current)
			allocations = nil
			for range 16 {
				allocate(maxBuffer)
			}
			for _, allocation := range allocations {
				if _, err := host.free.Call(ctx, allocation[0], allocation[1]); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
