package measure

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestStressMatrix(t *testing.T) {
	t.Parallel()
	cases := stressCases()
	if len(cases) != 41 || len(stressAxes) != 4 {
		t.Fatalf("matrix: %d seeds x %d axes", len(cases), len(stressAxes))
	}
	counts := map[string]int{}
	for _, test := range cases {
		counts[test.Workload]++
	}
	if !maps.Equal(counts, map[string]int{"runtime": 32, "two-cluster": 3, "workflow": 3, "activity": 3}) {
		t.Fatal(counts)
	}
	for index := range 32 {
		if cases[index].Seed != uint64(index)*1000003 {
			t.Fatal(cases[index])
		}
	}
}

func TestStressRetainsInterruptedChoicesAndDiagnostics(t *testing.T) {
	t.Parallel()
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("module")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 19, false, [][32]byte{a, b}, a, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	diagnostic, err := choice.BuildDiagnosticTrace([]choice.DiagnosticRecord{{Ordinal: 0}}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	observation := stressFixture()
	root := t.TempDir()
	result := wasi.Result{RuntimeInterrupted: true, InterruptedChoiceBytes: trace.Bytes, InterruptedDiagnosticBytes: diagnostic.Bytes, EnvironmentIdentity: "declared"}
	if err := stressCollect(t.Context(), root, "active", 1, result, &backends.ChoiceRequest{ExecutionIdentity: identity, Limit: 8192}, &observation); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(observation.Run.Incarnations[0].Diagnostic, diagnostic.Bytes) {
		t.Fatal("crashed incarnation lost diagnostic counters")
	}
	stored, err := os.ReadFile(filepath.Join(root, "active-1-choice-trace.bin"))
	if err != nil || !bytes.Equal(stored, trace.Bytes) {
		t.Fatalf("interrupted choice prefix lost: %v", err)
	}
	tape, err := os.ReadFile(filepath.Join(root, "active-1-choice-tape.bin"))
	if err != nil || len(tape) == 0 {
		t.Fatalf("interrupted choices lack projected tape: %v", err)
	}
}

func stressFixture() stressObservation {
	return stressObservation{Fresh: true, Coordinates: true, Digests: map[string]string{"choice": "same"}, Run: determinism.Run{Workload: "fixture", Seed: 7, Identities: map[string]string{"module": "same"}, Command: []string{"wasmmeasure", "stress"}}}
}

func TestStressRefusesReplayAndChangedIdentities(t *testing.T) {
	t.Parallel()
	for _, change := range []func(*stressObservation){
		func(run *stressObservation) { run.Fresh = false },
		func(run *stressObservation) { run.Run.Identities = map[string]string{"module": "different"} },
		func(run *stressObservation) { run.Run.Seed++ },
	} {
		reference, actual := stressFixture(), stressFixture()
		change(&actual)
		if _, err := stressCompare(reference, actual); !errors.Is(err, ErrRefused) {
			t.Fatalf("accepted inadmissible cell: %v", err)
		}
	}
}

func TestStressShortOrUncomparedCellNeverPasses(t *testing.T) {
	t.Parallel()
	for _, cell := range []stressCell{
		{Requested: 3, Runs: []stressObservation{stressFixture()}, Comparisons: []determinism.Report{{Status: "identical"}}},
		{Requested: 1, Runs: []stressObservation{stressFixture()}},
	} {
		cell.finish()
		if cell.Status != "skipped" {
			t.Fatal(cell.Status)
		}
	}
	cell := stressCell{Requested: 2, Runs: []stressObservation{stressFixture()}, Comparisons: []determinism.Report{{Status: "diverged"}}}
	cell.finish()
	if cell.Status != "divergence" {
		t.Fatal(cell.Status)
	}
}

func TestStressCanonicalDifferenceWithoutCoordinatesIsUndiagnosable(t *testing.T) {
	t.Parallel()
	reference, actual := stressFixture(), stressFixture()
	actual.Coordinates = false
	actual.Digests = map[string]string{"choice": "changed"}
	report, err := stressCompare(reference, actual)
	if err != nil || report.Status != "undiagnosable" {
		t.Fatalf("%+v %v", report, err)
	}
}
