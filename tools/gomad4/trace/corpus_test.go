package trace

import (
	"bytes"
	"testing"

	"go.temporal.io/server/tools/gomad4/virtualtime"
)

func TestFinalizeProducesStableSemanticAndCorpusDigests(t *testing.T) {
	first := testCorpus(t)
	second := testCorpus(t)
	second.Coverage[0], second.Coverage[1] = second.Coverage[1], second.Coverage[0]
	second.Traces[0], second.Traces[1] = second.Traces[1], second.Traces[0]

	if err := Finalize(&first); err != nil {
		t.Fatal(err)
	}
	if err := Finalize(&second); err != nil {
		t.Fatal(err)
	}
	firstBytes, err := Encode(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := Encode(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatalf("canonical corpus differs:\n%s\n%s", firstBytes, secondBytes)
	}
	if first.SemanticDigest == "" || first.CorpusDigest == "" || first.SemanticDigest == first.CorpusDigest {
		t.Fatalf("unexpected digests: semantic=%q corpus=%q", first.SemanticDigest, first.CorpusDigest)
	}
}

func TestDecodeRejectsSemanticPayloadWithStaleDigest(t *testing.T) {
	corpus := testCorpus(t)
	if err := Finalize(&corpus); err != nil {
		t.Fatal(err)
	}
	encoded, err := Encode(corpus)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(encoded, []byte(`"timer-a"`), []byte(`"timer-z"`), 1)
	if _, err := Decode(changed); err == nil {
		t.Fatal("Decode() accepted changed semantic payload")
	}
}

func testCorpus(t *testing.T) Corpus {
	t.Helper()
	state := virtualtime.NewState(0)
	first, err := virtualtime.Step(state, virtualtime.ScheduleTimer("timer-a", 2))
	if err != nil {
		t.Fatal(err)
	}
	second, err := virtualtime.Step(first.PostState, virtualtime.AdvanceTime())
	if err != nil {
		t.Fatal(err)
	}
	return Corpus{
		Schema: "gomadv4.virtual-time-corpus/v1",
		Generation: GenerationContract{
			Schema: "gomadv4.virtual-time-generation/v1", ModelIdentity: "sha256:model", BoundsIdentity: "sha256:bounds",
		},
		BaselineIdentity: "sha256:baseline",
		Coverage:         []string{"action.advance_time", "action.schedule_timer"},
		Traces: []BehaviorTrace{
			{
				Name: "schedule-and-advance", InitialState: state.Snapshot(),
				Steps: []StepRecord{record(0, first), record(1, second)},
			},
			{Name: "empty", InitialState: virtualtime.NewState(5).Snapshot(), Steps: []StepRecord{}},
		},
		Rejections: []RejectionCase{{
			Name: "advance-empty", InitialState: state.Snapshot(), Action: virtualtime.AdvanceTime(),
			PreStateIdentity: state.Identity(), Code: virtualtime.RejectionNoPendingTimer,
		}},
	}
}

func record(ordinal int, transition virtualtime.Transition) StepRecord {
	return StepRecord{
		Ordinal: ordinal, Action: transition.Action, PreStateIdentity: transition.PreStateIdentity,
		PostStateIdentity: transition.PostStateIdentity, ObservableDelta: transition.Delta,
	}
}
