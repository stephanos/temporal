package formalbridge

import (
	"testing"

	"go.temporal.io/server/tools/gomad/conformance"
	"go.temporal.io/server/tools/gomad/trace"
)

func TestGenerateCoversEveryVirtualTimeActionAndRejection(t *testing.T) {
	corpus, err := Generate(Inputs{
		ModelIdentity: "sha256:model", BoundsIdentity: "sha256:bounds", BaselineIdentity: "sha256:baseline",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"action.advance_time", "action.cancel_timer", "action.fire_timer", "action.schedule_timer", "action.set_runnable",
		"rejection.deadline_before_now", "rejection.no_pending_timer", "rejection.ready_timer", "rejection.runnable_work",
		"rejection.runnable_unchanged", "rejection.timer_exists", "rejection.timer_not_ready", "rejection.timer_terminal", "rejection.unknown_timer",
	} {
		if !contains(corpus.Coverage, required) {
			t.Fatalf("coverage does not contain %q: %v", required, corpus.Coverage)
		}
	}
	if _, err := conformance.Replay(corpus, conformance.Limits{MaxTraces: 10, MaxSteps: 100, MaxRejections: 100}); err != nil {
		t.Fatal(err)
	}
	encoded, err := trace.Encode(corpus)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := trace.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.CorpusDigest != corpus.CorpusDigest {
		t.Fatalf("decoded corpus digest = %q, want %q", decoded.CorpusDigest, corpus.CorpusDigest)
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
