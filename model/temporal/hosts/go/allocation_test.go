package temporal_test

import (
	"context"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	enumspb "go.temporal.io/api/enums/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	annotationsv1 "umpire.local/model/umpire/annotations/v1"
	modelv1 "umpire.local/model/umpire/model/v1"
	temporalmodel "umpire.local/temporal"
)

type rawGuest struct {
	module api.Module
}

func newRawGuest(t *testing.T) *rawGuest {
	t.Helper()
	ctx := context.Background()
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter().
		WithCoreFeatures(api.CoreFeaturesV2|experimental.CoreFeaturesExceptionHandling).
		WithMemoryLimitPages(1024))
	t.Cleanup(func() {
		if err := runtime.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		t.Fatal(err)
	}
	module, err := runtime.InstantiateWithConfig(ctx, temporalmodel.Guest(), wazero.NewModuleConfig().WithStartFunctions())
	if err != nil {
		t.Fatal(err)
	}
	return &rawGuest{module: module}
}

func (g *rawGuest) invoke(t *testing.T, name string, args ...uint64) []uint64 {
	t.Helper()
	function := g.module.ExportedFunction(name)
	if function == nil {
		t.Fatalf("missing ABI export %s", name)
	}
	result, err := function.Call(context.Background(), args...)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return result
}

func (g *rawGuest) call(t *testing.T, method modelv1.Method, request proto.Message, exhausted bool) *modelv1.Response {
	t.Helper()
	data, err := proto.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var allocations [][2]uint64
	defer func() {
		for _, allocation := range allocations {
			g.invoke(t, "umpire_buffer_release", allocation[0], allocation[1])
		}
	}()
	allocate := func(length uint64) uint64 {
		t.Helper()
		result := g.invoke(t, "umpire_buffer_alloc", length)
		if len(result) != 1 || result[0] == 0 {
			t.Fatalf("alloc(%d) = %v", length, result)
		}
		allocations = append(allocations, [2]uint64{result[0], length})
		return result[0]
	}
	var pointer uint64
	if len(data) != 0 {
		pointer = allocate(uint64(len(data)))
		if !g.module.Memory().Write(uint32(pointer), data) {
			t.Fatal("request allocation is outside guest memory")
		}
	}
	if exhausted {
		for remaining := uint64(16<<20) - uint64(len(data)); remaining != 0; {
			length := min(remaining, 1<<20)
			allocate(length)
			remaining -= length
		}
	}
	result := g.invoke(t, "umpire_call", uint64(method), pointer, uint64(len(data)))
	if len(result) != 1 {
		t.Fatalf("call result = %v", result)
	}
	if exhausted {
		if result[0] != 0 {
			t.Fatalf("response allocation succeeded with all 16 MiB live: %v", result)
		}
		return nil
	}
	responsePointer, responseLength := uint32(result[0]), uint32(result[0]>>32)
	if responsePointer == 0 || responseLength == 0 || responseLength > 1<<20 {
		t.Fatalf("invalid response allocation: %v", result)
	}
	allocations = append(allocations, [2]uint64{uint64(responsePointer), uint64(responseLength)})
	encoded, ok := g.module.Memory().Read(responsePointer, responseLength)
	if !ok {
		t.Fatal("response allocation is outside guest memory")
	}
	response := new(modelv1.Response)
	if err := proto.Unmarshal(encoded, response); err != nil {
		t.Fatal(err)
	}
	if response.GetRefusal() != nil {
		t.Fatalf("unexpected refusal: %v", response.GetRefusal())
	}
	return response
}

func (g *rawGuest) deliver(t *testing.T) {
	t.Helper()
	response := g.call(t, modelv1.Method_METHOD_ANSWER, &modelv1.AnswerRequest{Callback: callback(t, "Complete")}, false)
	if response.GetAnswer().GetReply() == nil {
		t.Fatalf("delivery reply = %v", response)
	}
}

func (g *rawGuest) judge(t *testing.T, observation *anypb.Any) *modelv1.JudgeResponse {
	t.Helper()
	response := g.call(t, modelv1.Method_METHOD_JUDGE, &modelv1.JudgeRequest{Observation: observation}, false)
	if response.GetJudge() == nil {
		t.Fatalf("judgment = %v", response)
	}
	return response.GetJudge()
}

func TestTemporalResponseExhaustionRollsBackState(t *testing.T) {
	for _, tc := range []struct {
		name              string
		delivered, frozen bool
		method            modelv1.Method
		request           func(*testing.T) proto.Message
	}{
		{"answer delivery", false, false, modelv1.Method_METHOD_ANSWER, func(t *testing.T) proto.Message {
			return &modelv1.AnswerRequest{Callback: callback(t, "Complete")}
		}},
		{"judge terminal", true, false, modelv1.Method_METHOD_JUDGE, func(t *testing.T) proto.Message {
			return &modelv1.JudgeRequest{Observation: observation(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, 0)}
		}},
		{"reset started", true, false, modelv1.Method_METHOD_RESET, func(*testing.T) proto.Message { return &modelv1.ResetRequest{} }},
		{"reset frozen", true, true, modelv1.Method_METHOD_RESET, func(*testing.T) proto.Message { return &modelv1.ResetRequest{} }},
		{"judge freeze", true, false, modelv1.Method_METHOD_JUDGE, func(t *testing.T) proto.Message {
			return &modelv1.JudgeRequest{Observation: scheduled(t)}
		}},
		{"annotate started", true, false, modelv1.Method_METHOD_ANNOTATE, annotationPoll},
		{"annotate frozen", true, true, modelv1.Method_METHOD_ANNOTATE, annotationPoll},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newRawGuest(t)
			matching := scheduled(t)
			if tc.delivered {
				g.deliver(t)
				matching = started(t)
			}
			var frozen *modelv1.JudgeResponse
			if tc.frozen {
				frozen = g.judge(t, scheduled(t))
				unexplained(t, frozen, "Client::Cancel", "Worker::RespondCompleted", "Worker::RespondFailed", "Deadline::StartToClose")
			}
			g.call(t, tc.method, tc.request(t), true)
			after := g.judge(t, matching)
			if tc.frozen {
				if !proto.Equal(frozen, after) {
					t.Fatalf("failed reset changed frozen judgment: %v -> %v", frozen, after)
				}
			} else {
				explained(t, after)
			}
		})
	}
}

func annotationPoll(t *testing.T) proto.Message {
	t.Helper()
	return &modelv1.AnnotateRequest{Input: pack(t, &annotationsv1.AnnotationRequest{
		Method: "/temporal.api.workflowservice.v1.WorkflowService/PollActivityTaskQueue", Response: true,
	})}
}
