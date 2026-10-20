package measure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner"
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

func TestStressCohortBarrierReleasesAllMembersAndCancelledPreparation(t *testing.T) {
	t.Parallel()
	for _, cancelEarly := range []bool{false, true} {
		ctx, stop := context.WithTimeout(t.Context(), time.Second)
		cohort, cancel := stressCohortContext(ctx, 2)
		wait := cohort.Value(stressCohortKey{}).(func())
		started, done := make(chan struct{}), make(chan struct{})
		go func() { close(started); wait(); close(done) }()
		<-started
		select {
		case <-done:
			t.Fatal("cohort began before its peer was prepared")
		default:
		}
		if cancelEarly {
			cancel()
		} else {
			wait()
		}
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatal("cohort preparation barrier did not release")
		}
		cancel()
		stop()
	}
}

func TestStressConcurrentRepetitionsAreFullCohorts(t *testing.T) {
	t.Parallel()
	for _, parallel := range []int{2, 5} {
		cell := stressNewCell(stressCase{Workload: "runtime"}, "concurrent", 3, parallel)
		if cell.Rounds != 3 || cell.Fanout != parallel || cell.Requested != 3*parallel || cell.Requested%cell.Fanout != 0 {
			t.Fatalf("partial or unrealized fan-out: %+v", cell)
		}
	}
	cell := stressNewCell(stressCase{}, "cpu", 3, 5)
	if cell.Requested != 3 || cell.Fanout != 1 {
		t.Fatalf("non-concurrent repetitions changed: %+v", cell)
	}
}

func TestStressRuntimeCanaryIsOnlyAnExpectedApplicationExit(t *testing.T) {
	t.Parallel()
	zero, seventeen := uint32(0), uint32(17)
	for _, result := range []wasi.Result{{Termination: "exit", ExitCode: &zero, Reaped: true}, {Termination: "exit", ExitCode: &seventeen, Reaped: true, Stdout: []byte("canary 2\n")}} {
		if err := stressRequireExit(t.Context(), "runtime", result); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		workload string
		result   wasi.Result
	}{
		{"runtime", wasi.Result{Termination: "exit", ExitCode: &seventeen, Reaped: true}},
		{"workflow", wasi.Result{Termination: "exit", ExitCode: &seventeen, Reaped: true, Stdout: []byte("canary 2\n")}},
		{"runtime", wasi.Result{Termination: "capacity", ExitCode: &seventeen, Reaped: true, Stdout: []byte("canary 2\n")}},
		{"runtime", wasi.Result{Termination: "infrastructure", ExitCode: &seventeen, Reaped: true, Stdout: []byte("canary 2\n")}},
		{"runtime", wasi.Result{Termination: "exit", ExitCode: &zero, Reaped: true, Cancelled: true}},
		{"runtime", wasi.Result{Termination: "exit", ExitCode: &zero, Reaped: true, WatchdogTimeout: true}},
	} {
		if err := stressRequireExit(t.Context(), test.workload, test.result); err == nil {
			t.Fatalf("accepted unexpected outcome: %+v", test)
		}
	}
}

func TestStressCanaryRetainsNormalFailureArtifact(t *testing.T) {
	t.Parallel()
	success := runner.CampaignResult{Attempted: 1, Succeeded: 1, SuccessArtifacts: []string{"success"}}
	if path, err := stressArtifactPath(success, false); err != nil || path != "success" {
		t.Fatalf("success artifact lost: %s %v", path, err)
	}
	zero := record.Uint64String(0)
	if err := stressArtifactOutcome(record.ExecutionRecord{ArtifactKind: record.ArtifactSuccess, Outcome: record.Outcome{Domain: "success", Termination: "exit", ExitCode: &zero}}, false); err != nil {
		t.Fatal(err)
	}
	failure := runner.CampaignResult{Attempted: 1, Failures: 1, Artifacts: []string{"failure"}}
	if path, err := stressArtifactPath(failure, true); err != nil || path != "failure" {
		t.Fatalf("canary failure artifact lost: %s %v", path, err)
	}
	if _, err := stressArtifactPath(failure, false); err == nil {
		t.Fatal("ordinary workload accepted a failure")
	}
	seventeen := record.Uint64String(17)
	manifest := record.ExecutionRecord{ArtifactKind: record.ArtifactTargetFailure, Outcome: record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &seventeen}}
	if err := stressArtifactOutcome(manifest, true); err != nil {
		t.Fatal(err)
	}
	for _, domain := range []string{"runner", "watchdog", "capacity"} {
		manifest.Outcome.Domain = domain
		if err := stressArtifactOutcome(manifest, true); err == nil {
			t.Fatalf("accepted %s artifact as a canary", domain)
		}
	}
	failure.Watchdogs = 1
	if _, err := stressArtifactPath(failure, true); err == nil {
		t.Fatal("accepted watchdog outcome as a canary")
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
