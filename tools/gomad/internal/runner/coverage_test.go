package runner

import (
	"bytes"
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestWatchdogDiagnosticReplayRejectsUnsupportedChoiceEvidence(t *testing.T) {
	path, observed := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true})
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	})
	choices, err := opened.ReadPayload(opened.Manifest().ChoiceProfile.Trace.File, uint64(opened.Manifest().ChoiceProfile.Trace.Limit))
	if err != nil {
		t.Fatal(err)
	}
	targetPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	manifest := opened.Manifest()
	manifest.ArtifactKind, manifest.ReplayMode = record.ArtifactWatchdogTimeout, record.ReplayDiagnostic
	manifest.IOProfile.Transcript = nil
	deadline := "execution_timeout"
	manifest.Outcome = record.Outcome{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline}
	_, payloads := record.NoneWorld()
	path = publishWatchdogReplayInput(t, artifact.ArtifactInput{Manifest: manifest, TargetPath: targetPath, Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes, ChoiceTrace: choices, World: payloads})
	executor := &fakeReplayExecutor{result: observed}
	config := watchdogReplaySpec(t, path)
	configDependencies := executionDependencies{}
	configDependencies.executor = executor
	result, err := replayWith(t.Context(), config, configDependencies)
	if err == nil || !strings.Contains(err.Error(), "diagnostic replay without an I/O transcript cannot replay recorded choices") || executor.calls != 0 || result.Match || result.ChoiceReplayStatus == ChoiceReplayExact {
		t.Fatalf("unsupported diagnostic replay = %#v, %v, calls = %d", result, err, executor.calls)
	}
}

// watchdogReplayTimeouts are the execution timeouts a real watchdog run may use, shortest first.
// The watchdog is a host wall-clock deadline that starts with the supervisor, so a loaded host can
// fire it before the target has started. Such a run observed nothing of the target and is repeated
// under the next timeout instead of being compared.
var watchdogReplayTimeouts = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 32 * time.Second}

func publishWatchdogReplayInput(t *testing.T, input artifact.ArtifactInput) string {
	t.Helper()
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, input)
	if err != nil {
		t.Fatal(err)
	}
	return published.Path
}

const watchdogReplayMainSource = `package main
import ("fmt"; "os"; "runtime")
func main() {
 data, err := os.ReadFile("/mounted/input")
 if err != nil { panic(err) }
 fmt.Print(string(data))
 for { runtime.Gosched() }
}
`

func TestWatchdogReplaySourceParsesWithStandardImports(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", watchdogReplayMainSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	wantImports := []string{`"fmt"`, `"os"`, `"runtime"`}
	if file.Name.Name != "main" || len(file.Imports) != len(wantImports) || len(file.Decls) != 2 {
		t.Fatalf("watchdog source package/imports/declarations = %s/%v/%v", file.Name.Name, file.Imports, file.Decls)
	}
	for index, spec := range file.Imports {
		if spec.Name != nil || spec.Path.Value != wantImports[index] {
			t.Fatalf("watchdog source import %d = %#v, want %s without alias", index, spec, wantImports[index])
		}
	}
	main, ok := file.Decls[1].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("watchdog source main declaration = %T", file.Decls[1])
	}
	var body bytes.Buffer
	if err := format.Node(&body, token.NewFileSet(), main); err != nil {
		t.Fatal(err)
	}
	const wantMain = `func main() {
	data, err := os.ReadFile("/mounted/input")
	if err != nil {
		panic(err)
	}
	fmt.Print(string(data))
	for {
		runtime.Gosched()
	}
}`
	if body.String() != wantMain {
		t.Fatalf("watchdog source main = %s, want %s", body.String(), wantMain)
	}
}

func TestChoiceTraceObservedExcludesWatchdogAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name   string
		result backend.ExecutionResult
		want   bool
	}{
		{name: "exited", result: backend.ExecutionResult{}, want: true},
		{name: "watchdog", result: backend.ExecutionResult{WatchdogTimeout: true}, want: false},
		{name: "cancelled", result: backend.ExecutionResult{Cancelled: true}, want: false},
	} {
		if got := choiceTraceObserved(test.result); got != test.want {
			t.Fatalf("%s: choiceTraceObserved() = %v, want %v", test.name, got, test.want)
		}
	}
}

// A target the watchdog kills never writes its terminal choice frame; the run
// is the watchdog outcome, retained without a choice profile, not an
// unterminated-trace runner failure that discards the evidence.
func TestRunClassifiesWatchdogTimeoutBeforeUnterminatedChoiceTrace(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	result := processResult(0, "", "")
	result.Termination = backend.TerminationSignal
	result.Signal = "SIGKILL"
	result.WatchdogTimeout = true
	result.ChoiceTrace = backend.ChoiceTrace{Profile: choice.Profile, Limit: limit}
	config, configDependencies := testConfig(t, preparer, terminalErrorExecutor{result: result}, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = limit
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatalf("exploreWith() error = %v", err)
	}
	if summary.Watchdogs != 1 || len(summary.Artifacts) != 1 {
		t.Fatalf("exploreWith() summary = %#v", summary)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if opened.Manifest().ArtifactKind != record.ArtifactWatchdogTimeout || opened.Manifest().Outcome.Reason != "watchdog_timeout" || opened.Manifest().ChoiceProfile != nil {
		t.Fatalf("watchdog manifest = %#v", opened.Manifest())
	}
}

func watchdogReplaySpec(t *testing.T, path string) ReplaySpec {
	t.Helper()
	return ReplaySpec{ArtifactPath: path, ToolchainRoot: toolchainRoot(t),
		Backend: replayNativeExecutionProvider(t)}
}
