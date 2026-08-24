package regenerate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad/trace"
	"go.temporal.io/server/tools/gomad/virtualtime"
)

func TestVerifyRegeneratesTwiceAndComparesCompleteOutput(t *testing.T) {
	corpus := generatedCorpus(t)
	generator := &fakeGenerator{corpus: corpus}

	evidence, err := Verify(context.Background(), generator, Config{
		ExpectedSemanticDigest: corpus.SemanticDigest,
		RequiredCoverage:       []string{"action.schedule_timer"},
		MaxFiles:               4,
		MaxBytes:               1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if generator.calls != 2 {
		t.Fatalf("generator calls = %d, want 2", generator.calls)
	}
	if evidence.FirstCorpusDigest != corpus.CorpusDigest || evidence.FirstCorpusDigest != evidence.SecondCorpusDigest {
		t.Fatalf("evidence = %#v", evidence)
	}
}

func TestVerifyRejectsNondeterministicIncidentalOutput(t *testing.T) {
	corpus := generatedCorpus(t)
	generator := &fakeGenerator{corpus: corpus, addIncidentalFile: true}

	if _, err := Verify(context.Background(), generator, Config{
		ExpectedSemanticDigest: corpus.SemanticDigest, MaxFiles: 4, MaxBytes: 1 << 20,
	}); err == nil {
		t.Fatal("Verify() accepted nondeterministic generated output")
	}
}

type fakeGenerator struct {
	corpus            trace.Corpus
	calls             int
	addIncidentalFile bool
}

func (generator *fakeGenerator) Generate(_ context.Context, output string) error {
	generator.calls++
	encoded, err := trace.Encode(generator.corpus)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, CorpusFile), encoded, 0o600); err != nil {
		return err
	}
	if generator.addIncidentalFile {
		return os.WriteFile(filepath.Join(output, "incidental.txt"), []byte{byte(generator.calls)}, 0o600)
	}
	return nil
}

func generatedCorpus(t *testing.T) trace.Corpus {
	t.Helper()
	state := virtualtime.NewState(0)
	transition, err := virtualtime.Step(state, virtualtime.ScheduleTimer("timer", 1))
	if err != nil {
		t.Fatal(err)
	}
	corpus := trace.Corpus{
		Schema:           "gomad.virtual-time-corpus/v1",
		Generation:       trace.GenerationContract{Schema: "gomad.virtual-time-generation/v1", ModelIdentity: "sha256:model", BoundsIdentity: "sha256:bounds"},
		BaselineIdentity: "sha256:baseline",
		Coverage:         []string{"action.schedule_timer"},
		Traces: []trace.BehaviorTrace{{
			Name: "schedule", InitialState: state.Snapshot(),
			Steps: []trace.StepRecord{{
				Ordinal: 0, Action: transition.Action, PreStateIdentity: transition.PreStateIdentity,
				PostStateIdentity: transition.PostStateIdentity, ObservableDelta: transition.Delta,
			}},
		}},
	}
	if err := trace.Finalize(&corpus); err != nil {
		t.Fatal(err)
	}
	return corpus
}
