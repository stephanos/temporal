package runner

import (
	"context"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	runv1 "umpire.local/model/gen/umpire/run/v1"
)

func TestRunWorkerUsesFreshWireTag(t *testing.T) {
	run := &Run{
		ModelHash: "h", ProtocolVersion: 1, World: &anypb.Any{TypeUrl: "w"}, Seed: 2,
		Events:  []*runv1.Event{{Kind: &runv1.Event_Answer{Answer: &runv1.Answer{Call: 3}}}},
		Verdict: &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{Termination: runv1.Termination_TERMINATION_QUIESCENT}}},
	}
	vectors, err := os.ReadFile("../../../protocol/proto/tests/run-wire.txt")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(vectors)), "\n") {
		name, wire, ok := strings.Cut(line, " ")
		if !ok || seen[name] {
			t.Fatalf("invalid or duplicate vector: %s", line)
		}
		seen[name] = true
		switch name {
		case "legacy":
			run.Worker = nil
		case "worker":
			run.Worker = &runv1.Worker{Kind: "k", Version: "v", Namespace: "n", TaskQueue: "q"}
		default:
			t.Fatalf("unknown vector: %s", name)
		}
		encoded := mustMarshal(t, run)
		if hex.EncodeToString(encoded) != wire {
			t.Fatalf("%s: %x, want %s", name, encoded, wire)
		}
		var decoded Run
		if err := proto.Unmarshal(encoded, &decoded); err != nil || !proto.Equal(run, &decoded) {
			t.Fatalf("%s round trip: %v, %v", name, &decoded, err)
		}
	}
	if len(seen) != 2 {
		t.Fatalf("got %d vectors, want 2", len(seen))
	}
}

func TestWorkerMetadataIsRecordedAndReplayedWithoutDriver(t *testing.T) {
	wasm := statefulGuest(t)
	h := newHarness(t, wasm, driverFor(good))
	run := h.run(ModeExplore, configured(2))
	if got := run.GetWorker(); got == nil || got.GetKind() != "fake" || got.GetVersion() != "1" {
		t.Fatalf("worker = %v, want fake version 1", got)
	}
	requireReplays(t, wasm, run)
	replayed, err := New(wasm, nil).Run(context.Background(), ModeReplay, Options{Recorded: run})
	if err != nil {
		t.Fatal(err)
	}
	run.GetWorker().Kind = "recorded-executor"
	if replayed.GetWorker().GetKind() != "fake" {
		t.Fatal("recording mutation changed the replayed Run")
	}
	requireReplays(t, wasm, run)
}

func TestRunOwnsDriverWorkerMetadata(t *testing.T) {
	wasm := statefulGuest(t)
	metadata := &runv1.Worker{Kind: "fake", Version: "1", Namespace: "space", TaskQueue: "queue"}
	h := newHarness(t, wasm, func(int) *fakeDriver {
		return &fakeDriver{script: good, worker: metadata}
	})
	run := h.run(ModeExplore, configured(2))
	if !proto.Equal(run.GetWorker(), metadata) {
		t.Fatalf("worker = %v, want %v", run.GetWorker(), metadata)
	}
	metadata.Kind = "changed"
	if run.GetWorker().GetKind() != "fake" {
		t.Fatal("Driver mutation changed the finished Run")
	}
	requireReplays(t, wasm, run)
}

func TestRefusedVerdictNamesTheRecordedReason(t *testing.T) {
	event := &runv1.Event{Sequence: 4, Kind: &runv1.Event_Refusal{Refusal: &runv1.Refusal{Method: modelv1.Method_METHOD_JUDGE, Refusal: &modelv1.Refusal{Code: modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED, Message: "unprogrammed worker task"}}}}
	verdict := terminal(event)
	if !strings.Contains(verdict.GetRefused().GetReason(), event.GetRefusal().GetRefusal().GetMessage()) {
		t.Fatalf("verdict lost refusal reason: %v", verdict)
	}
}
