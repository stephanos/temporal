package runner

import (
	"context"
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"
	runv1 "umpire.local/model/gen/umpire/run/v1"
)

func TestWholePrefixRecordingWithCallbacksAndCurrent(t *testing.T) {
	wasm := statefulGuest(t)
	script := map[byte][]step{
		0: good[0],
		1: {{kind: EventObservation, value: 20}, {kind: EventCallback, value: 4, call: 7}, {kind: EventObservation, value: 20}, {kind: EventObservation, value: 2}, {kind: EventDone}},
		3: {{kind: EventObservation, value: 20}, {kind: EventObservation, value: 3}, {kind: EventDone}},
	}
	run := newHarness(t, wasm, driverFor(script)).run(ModeExplore, configured(1))
	requireExplained(t, run, runv1.Termination_TERMINATION_QUIESCENT)
	expected := map[int][]string{
		2: {}, 4: {}, 6: {"Deliver"}, 7: {"Deliver", "Respond"},
		9: {"Deliver", "Respond"}, 10: {"Deliver", "Respond", "Finish"},
	}
	for i, steps := range expected {
		explanation := run.Events[i].GetJudgment().GetResponse().GetExplained()
		if explanation == nil || !slices.Equal(explanation.GetSteps(), steps) || len(explanation.GetOutcomes()) != len(steps) {
			t.Fatalf("event %d: whole-prefix witness = %v, want %v", i, explanation, steps)
		}
	}
	if run.Events[5].GetAnswer().GetCall() != 7 {
		t.Fatal("lost callback correlation")
	}
	before := mustMarshal(t, run)
	requireReplays(t, wasm, run)
	publication, err := Publish(t.Context(), t.TempDir(), run)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(publication.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(run, loaded) {
		t.Fatal("publication changed witnesses")
	}
	requireReplays(t, wasm, loaded)
	for _, prefix := range []int{7, 8} {
		question, err := New(wasm, nil).Run(t.Context(), ModeQuestion, Options{Recorded: run, Prefix: prefix, Observation: control(20)})
		if err != nil {
			t.Fatal(err)
		}
		want := expected[6]
		if prefix == 8 {
			want = expected[7]
		}
		last := question.Events[len(question.Events)-1]
		if !slices.Equal(last.GetJudgment().GetResponse().GetExplained().GetSteps(), want) {
			t.Fatalf("question prefix %d: %v", prefix, last)
		}
		requireReplays(t, wasm, question)
	}
	for name, alter := range map[string]func(*Run){
		"latest observation only": func(run *Run) {
			run.Events[10].GetJudgment().GetResponse().GetExplained().Steps = []string{"Finish"}
			run.Events[10].GetJudgment().GetResponse().GetExplained().Outcomes = []string{"Outcome(3)"}
		},
		"wrong earlier prefix": func(run *Run) {
			run.Events[6].GetJudgment().GetResponse().GetExplained().Steps = nil
			run.Events[6].GetJudgment().GetResponse().GetExplained().Outcomes = nil
		},
		"concatenated deltas": func(run *Run) {
			run.Events[10].GetJudgment().GetResponse().GetExplained().Steps = []string{"Deliver", "Deliver", "Respond", "Deliver", "Respond", "Finish"}
			run.Events[10].GetJudgment().GetResponse().GetExplained().Outcomes = []string{"Outcome(1)", "Outcome(1)", "Outcome(2)", "Outcome(1)", "Outcome(2)", "Outcome(3)"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			mutant := proto.CloneOf(run)
			alter(mutant)
			replayed, err := New(wasm, func(context.Context) (Driver, error) { t.Fatal("replay contacted Driver"); return nil, nil }).Run(t.Context(), ModeReplay, Options{Recorded: mutant})
			if err != nil {
				t.Fatal(err)
			}
			if replayed.GetVerdict().GetRefused() == nil {
				t.Fatalf("mutant replayed: %v", replayed)
			}
		})
	}
	if string(before) != string(mustMarshal(t, run)) {
		t.Fatal("later replay/question/publication mutated prior reports")
	}
}
