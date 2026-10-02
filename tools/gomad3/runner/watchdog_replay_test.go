package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestWatchdogDiagnosticReplayUsesCapturedInputs(t *testing.T) {
	input, observed, source := watchdogReplayInput(t)
	if err := os.RemoveAll(source); err != nil {
		t.Fatal(err)
	}
	t.Run("captured input after host removal", func(t *testing.T) {
		result, err := replayWatchdogInput(t, input)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Verified || !result.Diagnostic || !result.Match || result.ChoiceReplayStatus != ChoiceReplayNone {
			t.Fatalf("diagnostic replay = %#v", result)
		}
	})
	t.Run("uncaptured input fails", func(t *testing.T) {
		missing := input
		mounts, err := readonlymount.EncodeCapturedInputs([]readonlymount.Mapping{{Target: "/mounted"}}, readonlymount.DefaultLimits(), readonlymount.Snapshot{})
		if err != nil {
			t.Fatal(err)
		}
		missing.ReadOnlyMounts = &mounts
		missing.Manifest.IOProfile.ReadOnlyMounts = pointerToReadOnlyMounts(replayRecordedCapturedInputs(mounts.Manifest))
		result, err := replayWatchdogInput(t, missing)
		if !errors.Is(err, readonlymount.ErrReplayDivergence) || result.Match {
			t.Fatalf("uncaptured input replay = %#v, %v", result, err)
		}
	})
	t.Run("exact replay requires transcript", func(t *testing.T) {
		exact := input
		exact.Manifest.ArtifactKind = record.ArtifactTargetFailure
		exact.Manifest.ReplayMode = record.ReplayExact
		exitCode := record.Uint64String(2)
		exact.Manifest.Outcome = record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode}
		path := publishWatchdogReplayInput(t, exact)
		executor := &fakeReplayExecutor{result: observed}
		config := watchdogReplaySpec(t, path)
		config.Executor = executor
		_, err := Replay(t.Context(), config)
		if err == nil || !strings.Contains(err.Error(), "no complete transcript") || executor.calls != 0 {
			t.Fatalf("exact replay error = %v, calls = %d", err, executor.calls)
		}
	})
	for _, corruption := range []string{"changed", "missing"} {
		t.Run(corruption+" retained input fails before execution", func(t *testing.T) {
			path := publishWatchdogReplayInput(t, input)
			for name := range input.ReadOnlyMounts.Payloads {
				var err error
				if corruption == "missing" {
					err = os.Remove(filepath.Join(path, name))
				} else {
					err = os.WriteFile(filepath.Join(path, name), []byte("changed input"), 0o600)
				}
				if err != nil {
					t.Fatal(err)
				}
				break
			}
			executor := &fakeReplayExecutor{result: observed}
			config := watchdogReplaySpec(t, path)
			config.Executor = executor
			if _, err := Replay(t.Context(), config); err == nil || executor.calls != 0 {
				t.Fatalf("corrupt replay error = %v, calls = %d", err, executor.calls)
			}
		})
	}
	t.Run("malformed retained descriptor fails before execution", func(t *testing.T) {
		malformed := input
		mounts := *input.ReadOnlyMounts
		mounts.Descriptor = []byte("{}")
		mounts.Manifest.SHA256 = record.HashBytes(mounts.Descriptor)
		mounts.Manifest.Bytes = uint64(len(mounts.Descriptor))
		malformed.ReadOnlyMounts = &mounts
		malformed.Manifest.IOProfile.ReadOnlyMounts = pointerToReadOnlyMounts(replayRecordedCapturedInputs(mounts.Manifest))
		path := publishWatchdogReplayInput(t, malformed)
		executor := &fakeReplayExecutor{result: observed}
		config := watchdogReplaySpec(t, path)
		config.Executor = executor
		if _, err := Replay(t.Context(), config); err == nil || executor.calls != 0 {
			t.Fatalf("malformed replay error = %v, calls = %d", err, executor.calls)
		}
	})
	t.Run("cancelled replay does not match", func(t *testing.T) {
		path := publishWatchdogReplayInput(t, input)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		result, err := Replay(ctx, watchdogReplaySpec(t, path))
		if result.Match || err == nil && result.Divergence == "" {
			t.Fatalf("cancelled replay = %#v, %v", result, err)
		}
	})
	for _, test := range []struct {
		name       string
		change     func(*execution.Result)
		divergence string
	}{
		{name: "stdout", change: func(result *execution.Result) { result.Stdout = replayOutput("changed") }, divergence: "stdout.full_sha256"},
		{name: "stderr", change: func(result *execution.Result) { result.Stderr = replayOutput("changed") }, divergence: "stderr.full_sha256"},
		{name: "outcome", change: func(result *execution.Result) { result.WatchdogTimeout = false }, divergence: "outcome.termination"},
	} {
		t.Run(test.name+" divergence", func(t *testing.T) {
			path := publishWatchdogReplayInput(t, input)
			changed := observed
			test.change(&changed)
			config := watchdogReplaySpec(t, path)
			config.Executor = &fakeReplayExecutor{result: changed}
			result, err := Replay(t.Context(), config)
			if err != nil || result.Match || !result.Diagnostic || result.Divergence != test.divergence {
				t.Fatalf("divergent diagnostic replay = %#v, %v", result, err)
			}
		})
	}
}

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
	choices, err := artifact.ReadPayload(opened, opened.Manifest.ChoiceProfile.Trace.File, uint64(opened.Manifest.ChoiceProfile.Trace.Limit))
	if err != nil {
		t.Fatal(err)
	}
	targetPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	manifest := opened.Manifest
	manifest.ArtifactKind, manifest.ReplayMode = record.ArtifactWatchdogTimeout, record.ReplayDiagnostic
	manifest.IOProfile.Transcript = nil
	deadline := "execution_timeout"
	manifest.Outcome = record.Outcome{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline}
	_, payloads := record.NoneWorld()
	path = publishWatchdogReplayInput(t, artifact.ArtifactInput{Manifest: manifest, TargetPath: targetPath, Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes, ChoiceTrace: choices, World: payloads})
	executor := &fakeReplayExecutor{result: observed}
	config := watchdogReplaySpec(t, path)
	config.Executor = executor
	result, err := Replay(t.Context(), config)
	if err == nil || !strings.Contains(err.Error(), "diagnostic replay without an I/O transcript cannot replay recorded choices") || executor.calls != 0 || result.Match || result.ChoiceReplayStatus == ChoiceReplayExact {
		t.Fatalf("unsupported diagnostic replay = %#v, %v, calls = %d", result, err, executor.calls)
	}
}

func watchdogReplaySpec(t *testing.T, path string) ReplaySpec {
	t.Helper()
	return ReplaySpec{ArtifactPath: path, ToolchainRoot: toolchainRoot(t),
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"}}
}

// watchdogReplayTimeouts are the execution timeouts a real watchdog run may use, shortest first.
// The watchdog is a host wall-clock deadline that starts with the supervisor, so a loaded host can
// fire it before the target has started. Such a run observed nothing of the target and is repeated
// under the next timeout instead of being compared.
var watchdogReplayTimeouts = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 32 * time.Second}

func replayWatchdogInput(t *testing.T, input artifact.ArtifactInput) (ReplayResult, error) {
	t.Helper()
	var result ReplayResult
	var err error
	for _, timeout := range watchdogReplayTimeouts {
		if record.Uint64String(timeout) < input.Manifest.Limits.ExecutionTimeoutNanos {
			continue
		}
		input.Manifest.Limits.ExecutionTimeoutNanos = record.Uint64String(timeout)
		config := watchdogReplaySpec(t, publishWatchdogReplayInput(t, input))
		config.ObservedDir = t.TempDir()
		result, err = Replay(t.Context(), config)
		if err != nil || result.Divergence != "stdout.full_sha256" {
			return result, err
		}
		stdout, readErr := os.ReadFile(filepath.Join(config.ObservedDir, "stdout"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if len(stdout) != 0 {
			return result, err
		}
		t.Logf("replay watchdog fired after %v before the target wrote output", timeout)
	}
	return result, err
}

func publishWatchdogReplayInput(t *testing.T, input artifact.ArtifactInput) string {
	t.Helper()
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, input)
	if err != nil {
		t.Fatal(err)
	}
	return published.Path
}

func watchdogReplayInput(t *testing.T) (artifact.ArtifactInput, execution.Result, string) {
	t.Helper()
	module := t.TempDir()
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/watchdogreplay\n\ngo 1.27.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "main.go"), []byte(`package main
import ("fmt"; "os"; "runtime")
func main() {
 data, err := os.ReadFile("/mounted/input")
 if err != nil { panic(err) }
 fmt.Print(string(data))
 for { runtime.Gosched() }
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(t.Context(), target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	const runnerBuild = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	frame, err := profile.BootstrapFrame(prepared, runnerBuild, 7)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "input"), []byte("captured input\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mappings := []readonlymount.Mapping{{Source: source, Target: "/mounted"}}
	limits := readonlymount.DefaultLimits()
	var observed execution.Result
	var timeout time.Duration
	for _, timeout = range watchdogReplayTimeouts {
		observed, err = execution.Run(t.Context(), execution.Spec{
			SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
			Command: prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: []string{"GOMAD3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"},
			ExecutionTimeout: timeout, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1 << 20,
			World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
			IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}, ReadOnlyMount: &execution.ReadOnlyMountCapability{Mappings: mappings, Limits: limits}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !observed.WatchdogTimeout || observed.Stdout.TotalBytes != 0 {
			break
		}
		t.Logf("fixture watchdog fired after %v before the target wrote output", timeout)
	}
	if !observed.WatchdogTimeout || observed.IOTranscript.Complete || string(observed.Stdout.Bytes) != "captured input\n" || len(observed.IOROMounts.Entries) != 1 {
		t.Fatalf("watchdog fixture = %#v", observed)
	}
	mounts, err := readonlymount.EncodeCapturedInputs(mappings, limits, observed.IOROMounts)
	if err != nil {
		t.Fatal(err)
	}
	worldRecord, worldPayloads := record.NoneWorld()
	deadline := "execution_timeout"
	return artifact.ArtifactInput{TargetPath: prepared.Path, Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes, ReadOnlyMounts: &mounts, World: worldPayloads,
		Manifest: record.ExecutionRecord{
			SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactWatchdogTimeout, CreatedAt: "2026-10-02T12:00:00Z", CampaignID: "watchdog-replay-test", Seed: 7, ReplayMode: record.ReplayDiagnostic,
			Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: runnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
			Toolchain: prepared.RecordToolchain(), Target: prepared.RecordTarget(),
			IOProfile:   record.IOProfile{Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256()), ReadOnlyMounts: pointerToReadOnlyMounts(replayRecordedCapturedInputs(mounts.Manifest))},
			Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: profile.Name()}, {Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
			Limits:      record.Limits{ExecutionTimeoutNanos: record.Uint64String(timeout), OverallTimeoutNanos: record.Uint64String(time.Minute), TerminateGraceNanos: record.Uint64String(100 * time.Millisecond), OutputBytes: 1 << 20, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20},
			World:       worldRecord, Outcome: record.Outcome{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline},
			Streams: record.Streams{Stdout: replayStream(observed.Stdout), Stderr: replayStream(observed.Stderr)},
			Host:    record.Host{StartedAt: "2026-10-02T12:00:00Z", FinishedAt: "2026-10-02T12:00:01Z", ElapsedNanos: record.Uint64String(time.Second)},
		}}, observed, source
}
