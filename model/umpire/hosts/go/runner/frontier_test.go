package runner

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"
	runv1 "umpire.local/model/gen/umpire/run/v1"
)

func TestFrontierChangesCanonicalPastWithoutChangingEarlierRunReports(t *testing.T) {
	wasm := statefulGuest(t)
	script := map[byte][]step{51: {{kind: EventObservation, value: 50}, {kind: EventObservation, value: 51}, {kind: EventDone}}}
	opts := Options{World: control(50), Seed: 1}
	run := newHarness(t, wasm, driverFor(script)).run(ModeExplore, opts)
	requireExplained(t, run, runv1.Termination_TERMINATION_QUIESCENT)
	var judgments []int
	for i, event := range run.Events {
		if event.Sequence != uint64(i) {
			t.Fatalf("event %d has sequence %d", i, event.Sequence)
		}
		if event.GetJudgment() != nil {
			judgments = append(judgments, i)
		}
	}
	if len(judgments) != 2 {
		t.Fatalf("judgments = %v, events = %v", judgments, kinds(run))
	}
	first, last := judgments[0], judgments[1]
	if got := run.Events[first].GetJudgment().GetResponse().GetExplained().GetSteps(); !slices.Equal(got, []string{"Renew"}) {
		t.Fatalf("first canonical witness = %v", got)
	}
	if got := run.Events[last].GetJudgment().GetResponse().GetExplained().GetSteps(); !slices.Equal(got, []string{"Tick", "Renew", "Tick", "Tick"}) {
		t.Fatalf("later canonical witness = %v", got)
	}
	immutable := mustMarshal(t, run)
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
		t.Fatal("publication changed immutable frontier reports")
	}
	requireReplays(t, wasm, loaded)
	for _, terminal := range []struct {
		value byte
		steps []string
	}{{51, []string{"Tick", "Renew", "Tick", "Tick"}}, {52, []string{"Renew", "Tick", "Tick"}}} {
		question, err := New(wasm, nil).Run(t.Context(), ModeQuestion, Options{Recorded: run, Prefix: first + 1, Observation: control(terminal.value)})
		if err != nil {
			t.Fatal(err)
		}
		latest := question.Events[len(question.Events)-1]
		if !slices.Equal(latest.GetJudgment().GetResponse().GetExplained().GetSteps(), terminal.steps) {
			t.Fatalf("question %d selected %v", terminal.value, latest)
		}
		requireReplays(t, wasm, question)
	}
	for name, alter := range map[string]func(*Run){
		"greedy past": func(run *Run) {
			report := run.Events[last].GetJudgment().GetResponse().GetExplained()
			report.Steps = []string{"Renew", "Tick", "Tick"}
			report.Outcomes = []string{"Renewed", "Advanced", "PendingClosed"}
		},
		"latest only": func(run *Run) {
			report := run.Events[last].GetJudgment().GetResponse().GetExplained()
			report.Steps = []string{"Tick", "Tick"}
			report.Outcomes = []string{"Advanced", "GlobalClosed"}
		},
		"wrong earlier report": func(run *Run) {
			report := run.Events[first].GetJudgment().GetResponse().GetExplained()
			report.Steps = []string{"Tick", "Renew"}
			report.Outcomes = []string{"Advanced", "Renewed"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			mutant := proto.CloneOf(run)
			alter(mutant)
			replayed, err := New(wasm, func(context.Context) (Driver, error) { t.Fatal("replay contacted Driver"); return nil, nil }).Run(t.Context(), ModeReplay, Options{Recorded: mutant})
			if err != nil {
				t.Fatal(err)
			}
			event := last
			if name == "wrong earlier report" {
				event = first
			}
			requireRefused(t, replayed, uint64(event), fmt.Sprintf("replay: event %d differs", event))
		})
	}
	if string(immutable) != string(mustMarshal(t, run)) {
		t.Fatal("question/replay/publication changed earlier Run reports")
	}
}

func TestFrontierReduceReproducesAnUnexplainedContinuationAfterCanonicalPastChanges(t *testing.T) {
	wasm := statefulGuest(t)
	script := map[byte][]step{51: {{kind: EventObservation, value: 50}, {kind: EventObservation, value: 53}, {kind: EventObservation, value: 52}, {kind: EventDone}}}
	original := newHarness(t, wasm, driverFor(script)).run(ModeExplore, Options{World: control(50), Seed: 1})
	if original.GetVerdict().GetUnexplained() == nil {
		t.Fatalf("original verdict = %v", original.GetVerdict())
	}
	if got := original.Events[3].GetJudgment().GetResponse().GetExplained().GetSteps(); !slices.Equal(got, []string{"Tick", "Renew", "Tick", "Tick"}) {
		t.Fatalf("changed canonical witness = %v", got)
	}
	immutable := mustMarshal(t, original)
	requireReplays(t, wasm, original)
	reduced := newHarness(t, wasm, driverFor(script)).run(ModeReduce, Options{Recorded: original})
	if reduced.GetVerdict().GetUnexplained() == nil {
		t.Fatalf("reduced verdict = %v", reduced.GetVerdict())
	}
	requireReplays(t, wasm, reduced)
	for i, event := range reduced.Events {
		if event.Sequence != uint64(i) {
			t.Fatalf("reduced event %d has sequence %d", i, event.Sequence)
		}
	}
	if string(immutable) != string(mustMarshal(t, original)) {
		t.Fatal("reduce changed original immutable reports")
	}
}

func TestFrontierClosedCanonicalCannotTerminateAnActiveCompatibleAlternative(t *testing.T) {
	wasm := statefulGuest(t)
	for _, configuration := range []byte{54, 55} {
		t.Run(fmt.Sprint(configuration), func(t *testing.T) {
			script := map[byte][]step{51: {{kind: EventObservation, value: 54}, {kind: EventDone}}}
			run := newHarness(t, wasm, driverFor(script)).run(ModeExplore, Options{World: control(configuration), Seed: 1})
			if got := run.Events[2].GetJudgment().GetResponse().GetExplained().GetSteps(); !slices.Equal(got, []string{"Tick", "Tick"}) {
				t.Fatalf("closed canonical witness = %v", got)
			}
			requireRefused(t, run, 3, "REFUSAL_CODE_STATE")
			requireReplays(t, wasm, run)
		})
	}
}
