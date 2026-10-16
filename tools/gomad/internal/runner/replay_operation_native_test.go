//go:build gomad_native

package runner

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	execution "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/native"
	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/world"
	worldprocess "go.temporal.io/server/tools/gomad/internal/model/world/process"
)

func replayNativeExecutionProvider(t *testing.T) *native.Provider {
	t.Helper()
	return native.New(native.Config{ToolchainRoot: toolchainRoot(t),
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"}})
}

func recordReplayIOTranscript(t *testing.T) []byte {
	t.Helper()
	targetPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	targetBytes, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	argv := []string{"gomad-target", "-test.run=^TestWorldReplayMatchingTarget$"}
	profile := transcript.Default()
	provider := native.New(native.Config{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
	})
	result, err := provider.Execute(context.Background(), execution.ExecutionRequest{
		Target: target.Prepared{Path: targetPath, SHA256: string(record.HashBytes(targetBytes)), Size: uint64(len(targetBytes)), Argv: argv, CapabilityMode: target.CapabilityModeClosure},
		Seed:   7, RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Command: targetPath, Argv0: argv[0], Args: argv[1:], Dir: t.TempDir(),
		Env:              []string{"GOMAD_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"},
		ExecutionTimeout: 5 * time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64,
		World: execution.WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &execution.IOCapability{Profile: record.IOProfile{Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256())}, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IOTranscript.Complete || result.IOTranscript.Records == 0 {
		t.Fatalf("recorded I/O transcript = %#v", result.IOTranscript)
	}
	return result.IOTranscript.Bytes
}

func TestReplayRejectsFirstDivergentWorldTransitionBeforeTargetMutation(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Register(world.Request{Kind: "read", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: "expected"}}); err != nil {
		t.Fatal(err)
	}
	bundle, err := execution.ComposeRecording(world.Recording{
		Initial: initial,
		Final:   core.Snapshot(),
		Terminal: world.Terminal{
			Kind:   world.TerminalReplayDivergence,
			Detail: "recorded replay divergence",
		},
	}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath, _ := publishReplayArtifactForTarget(t, &bundle, replayArtifactTarget{
		Argv:          []string{"gomad-target", "-test.run=^TestWorldReplayDivergentTarget$"},
		OutcomeReason: "world_replay_divergence",
		IOTranscript:  recordReplayIOTranscript(t),
	})
	result, err := replayWith(context.Background(), ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t),
		Backend: native.New(native.Config{ToolchainRoot: toolchainRoot(t),
			SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
			BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"}}),
	}, executionDependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Divergence != "world.terminal" {
		t.Fatalf("divergent transition reached target-visible mutation: %#v", result)
	}
}

func TestReplayExecutesMatchingWorldPlanThroughChildTransport(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Register(world.Request{Kind: "read", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: "expected"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	bundle, err := execution.ComposeRecording(world.Recording{Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalDeadlock}}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath, _ := publishReplayArtifactForTarget(t, &bundle, replayArtifactTarget{
		Argv:          []string{"gomad-target", "-test.run=^TestWorldReplayMatchingTarget$"},
		OutcomeReason: "world_deadlock",
		IOTranscript:  recordReplayIOTranscript(t),
	})
	result, err := replayWith(context.Background(), ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t),
		Backend: native.New(native.Config{ToolchainRoot: toolchainRoot(t),
			SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
			BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"}}),
	}, executionDependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match || result.Divergence != "" {
		t.Fatalf("replay result = %#v", result)
	}
}

func TestWorldReplayDivergentTarget(_ *testing.T) {
	if !worldReplayTargetSelected("TestWorldReplayDivergentTarget") {
		return
	}
	runWorldReplayTarget("diverge")
}

func TestWorldReplayMatchingTarget(_ *testing.T) {
	if !worldReplayTargetSelected("TestWorldReplayMatchingTarget") {
		return
	}
	runWorldReplayTarget("match")
}

func worldReplayTargetSelected(name string) bool {
	want := "-test.run=^" + name + "$"
	for _, argument := range os.Args {
		if argument == want {
			return true
		}
	}
	return false
}

func runWorldReplayTarget(mode string) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		os.Exit(10) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	session, err := worldprocess.Open(core)
	if err != nil {
		os.Exit(11) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	key := "expected"
	stdout := "recorded stdout"
	if mode == "diverge" {
		key = "actual"
	}
	_, transitionErr := session.Model().Register(world.Request{Kind: "read", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: key}})
	if transitionErr == nil {
		if mode == "diverge" {
			stdout = "target-visible mutation"
			transitionErr = fmt.Errorf("%w: replay plan was not attached", world.ErrReplayDivergence)
		} else {
			_, transitionErr = session.Model().Quiesce()
		}
	}
	if transitionErr != nil {
		if err := session.FinishError(transitionErr); err != nil {
			os.Exit(13) //nolint:revive // This subprocess helper reports failure by exit status.
		}
	} else if err := session.Finish(); err != nil {
		os.Exit(14) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	if _, err := net.Dial("tcp", "127.0.0.1:1"); err == nil {
		os.Exit(15) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	if _, err := os.Stdout.WriteString(stdout); err != nil {
		os.Exit(16) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	if _, err := os.Stderr.WriteString("recorded stderr"); err != nil {
		os.Exit(17) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	os.Exit(2) //nolint:revive // This subprocess target intentionally returns a nonzero status.
}
