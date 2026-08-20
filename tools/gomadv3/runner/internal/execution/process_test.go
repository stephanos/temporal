package execution

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/choice"
	romount "go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/world"
	worldtarget "go.temporal.io/server/tools/gomadv3/world/target"
)

func TestRunCapturesTargetExitAndBothStreams(t *testing.T) {
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0],
		Args:              []string{"-test.run=TestTargetHelper"},
		Argv0:             "gomadv3-target",
		Dir:               t.TempDir(),
		Env:               []string{"GOMADV3_PROCESS_HELPER=output"},
		RunTimeout:        5 * time.Second,
		TerminateGrace:    time.Second,
		OutputLimit:       64,
		World:             WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 7 || result.Signal != "" || result.WatchdogTimeout || result.Cancelled {
		t.Fatalf("result status = %#v", result)
	}
	if got, want := string(result.Stdout.Bytes), "target stdout\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := string(result.Stderr.Bytes), "target stderr\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if result.PID <= 0 || result.PGID != result.PID || !result.GroupGone {
		t.Fatalf("process identity = pid %d pgid %d gone %v", result.PID, result.PGID, result.GroupGone)
	}
}

func TestRunInstallsBoundedIOConfigurationDescriptor(t *testing.T) {
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=io-config"}, IO: &IOCapability{Config: []byte("profile-frame")},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 || string(result.Stdout.Bytes) != "profile-frame" {
		t.Fatalf("result = %#v, stdout = %q", result, result.Stdout.Bytes)
	}
}

func TestRunTransportsCompleteChoiceTrace(t *testing.T) {
	limit := uint64(1 << 20)
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-trace", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || result.ChoiceTrace.Profile != choice.Profile || result.ChoiceTrace.Limit != limit || result.ChoiceTrace.Trace.Summary.Records == 0 || result.ChoiceTrace.Trace.Summary.Branching == 0 {
		t.Fatalf("result = %#v", result)
	}
	if result.ChoiceTrace.ImplementationSHA256 != testChoiceImplementationSHA256 {
		t.Fatalf("choice implementation identity = %x", result.ChoiceTrace.ImplementationSHA256)
	}
}

func TestRunReturnsValidatedOverflowChoiceTrace(t *testing.T) {
	limit := uint64(MinimumChoiceTraceBytes)
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-trace", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if !errors.Is(err, ErrChoiceTraceOverflow) {
		t.Fatalf("Run() error = %v", err)
	}
	payloadBytes, payloadErr := choice.TracePayloadBytes(1)
	if payloadErr != nil {
		t.Fatal(payloadErr)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 || result.ChoiceTrace.Profile != choice.Profile || result.ChoiceTrace.Limit != limit || result.ChoiceTrace.Trace.Summary.Terminal != choice.TerminalOverflow || result.ChoiceTrace.Trace.Summary.Records != 1 || uint64(len(result.ChoiceTrace.Trace.Bytes)) != payloadBytes {
		t.Fatalf("overflow result = %#v", result)
	}
}

func TestRunReplaysCompleteChoiceTape(t *testing.T) {
	limit := uint64(1 << 20)
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("process choice target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, ImplementationSHA256: testChoiceImplementationSHA256,
	}
	request := Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-trace", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	recorded, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(recorded.ChoiceTrace.Trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	request.Choice = &ChoiceCapability{
		Mode: choice.ModeReplay, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
		ExecutionIdentity: identity, Limit: limit, ReplayPlan: &tape,
	}
	replayed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ExitCode != 0 || replayed.ChoiceTrace.Trace.Summary.Terminal != choice.TerminalComplete {
		t.Fatalf("replayed result = %#v", replayed)
	}
	if replayed.ChoiceTrace.Trace.SHA256 != recorded.ChoiceTrace.Trace.SHA256 {
		t.Fatalf("replayed trace = %x, want %x", replayed.ChoiceTrace.Trace.SHA256, recorded.ChoiceTrace.Trace.SHA256)
	}
}

func TestRunReplaysLogicalChoiceAcrossPhysicalRunQueueOrder(t *testing.T) {
	limit := uint64(1 << 20)
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("process reordered choice target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, ImplementationSHA256: testChoiceImplementationSHA256,
	}
	request := Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-reorder", "GOMADV3_CHOICE_REORDER=ab", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	recorded, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(recorded.ChoiceTrace.Trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	request.Env = []string{"GOMADV3_PROCESS_HELPER=choice-reorder", "GOMADV3_CHOICE_REORDER=ba", "GOMADSEED=7"}
	request.Choice = &ChoiceCapability{
		Mode: choice.ModeReplay, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
		ExecutionIdentity: identity, Limit: limit, ReplayPlan: &tape,
	}
	replayed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if string(recorded.Stdout.Bytes) != string(replayed.Stdout.Bytes) || recorded.ChoiceTrace.Trace.SHA256 != replayed.ChoiceTrace.Trace.SHA256 {
		t.Fatalf("reordered replay = %q/%x, want %q/%x", replayed.Stdout.Bytes, replayed.ChoiceTrace.Trace.SHA256, recorded.Stdout.Bytes, recorded.ChoiceTrace.Trace.SHA256)
	}
}

func TestRunReplaysSelectPermutationAcrossSeededPhysicalOrder(t *testing.T) {
	limit := uint64(1 << 20)
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("process select permutation target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, ImplementationSHA256: testChoiceImplementationSHA256,
	}
	request := Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-select", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	baseline, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	baselineTape, err := choice.ProjectReplayPlan(baseline.ChoiceTrace.Trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	repeatedTape, err := choice.ProjectReplayPlan(repeated.ChoiceTrace.Trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(repeatedTape.Decisions) != len(baselineTape.Decisions) {
		t.Fatalf("same-seed fixture recorded %d decisions, want %d", len(repeatedTape.Decisions), len(baselineTape.Decisions))
	}
	if difference := firstChoiceDecisionDifference(repeatedTape.Decisions, baselineTape.Decisions); difference >= 0 {
		t.Fatalf("same-seed fixture decision %d was not repeatable", difference)
	}
	var alternateSeed int
	for seed := 8; seed < 128; seed++ {
		request.Env = []string{"GOMADV3_PROCESS_HELPER=choice-select", fmt.Sprintf("GOMADSEED=%d", seed)}
		alternate, err := Run(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		tape, err := choice.ProjectReplayPlan(alternate.ChoiceTrace.Trace, identity)
		if err != nil {
			t.Fatal(err)
		}
		if compatibleSelectDifference(baselineTape.Decisions, tape.Decisions) {
			alternateSeed = seed
			break
		}
	}
	if alternateSeed == 0 {
		t.Fatal("no alternate seed changed a compatible select permutation")
	}
	request.Env = []string{"GOMADV3_PROCESS_HELPER=choice-select", fmt.Sprintf("GOMADSEED=%d", alternateSeed)}
	request.Choice = &ChoiceCapability{
		Mode: choice.ModeReplay, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
		ExecutionIdentity: identity, Limit: limit, ReplayPlan: &baselineTape,
	}
	replayed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if string(replayed.Stdout.Bytes) != string(baseline.Stdout.Bytes) || replayed.ChoiceTrace.Trace.SHA256 != baseline.ChoiceTrace.Trace.SHA256 {
		t.Fatalf("select replay = %q/%x, want %q/%x", replayed.Stdout.Bytes, replayed.ChoiceTrace.Trace.SHA256, baseline.Stdout.Bytes, baseline.ChoiceTrace.Trace.SHA256)
	}
}

func TestRunPreservesChoicePrefixRNGPosition(t *testing.T) {
	limit := uint64(1 << 20)
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("process prefix RNG target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, ImplementationSHA256: testChoiceImplementationSHA256,
	}
	request := Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-prefix-rng", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	baseline, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(baseline.ChoiceTrace.Trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(tape.Decisions) < 2 {
		t.Fatalf("prefix RNG fixture recorded %d decisions, want at least 2", len(tape.Decisions))
	}
	prefix, err := tape.Prefix(uint64(len(tape.Decisions) / 2))
	if err != nil {
		t.Fatal(err)
	}
	request.Choice = &ChoiceCapability{
		Mode: choice.ModePrefix, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
		ExecutionIdentity: identity, Limit: limit, ReplayPlan: &prefix,
	}
	prefixed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if string(prefixed.Stdout.Bytes) != string(baseline.Stdout.Bytes) || prefixed.ChoiceTrace.Trace.SHA256 != baseline.ChoiceTrace.Trace.SHA256 {
		t.Fatalf("prefix RNG result = %q/%x, want %q/%x", prefixed.Stdout.Bytes, prefixed.ChoiceTrace.Trace.SHA256, baseline.Stdout.Bytes, baseline.ChoiceTrace.Trace.SHA256)
	}
}

func TestRunForcesCanonicalRankAtFinalPrefixDecision(t *testing.T) {
	limit := uint64(1 << 20)
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("process rank prefix target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, ImplementationSHA256: testChoiceImplementationSHA256,
	}
	request := Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-prefix-rng", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	baseline, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(baseline.ChoiceTrace.Trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	var ordinal uint64
	var rank uint32
	found := false
	for index, decision := range tape.Decisions {
		if decision.Kind == choice.KindSelectPoll && decision.Alternatives > 1 {
			ordinal = uint64(index)
			rank = (decision.Selected + 1) % decision.Alternatives
			found = true
			break
		}
	}
	if !found {
		t.Fatal("prefix fixture recorded no branching select decision")
	}
	prefix, err := choice.BuildRankPrefix(tape, ordinal, rank)
	if err != nil {
		t.Fatal(err)
	}
	request.Choice = &ChoiceCapability{
		Mode: choice.ModePrefix, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
		ExecutionIdentity: identity, Limit: limit, ReplayPlan: &prefix,
	}
	forced, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	forcedTape, err := choice.ProjectReplayPlan(forced.ChoiceTrace.Trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	decision := forcedTape.Decisions[ordinal]
	if decision.Selected != rank || decision.SelectedIdentity == ([sha256.Size]byte{}) || decision.RankOverride {
		t.Fatalf("forced decision = %#v, want canonical rank %d", decision, rank)
	}
}

func TestRunTargetInheritsChoiceTapeReadOnly(t *testing.T) {
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("process read-only choice target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, ImplementationSHA256: testChoiceImplementationSHA256,
	}
	tape, err := choice.ProjectReplayPlan(choice.Trace{
		Version: choice.Version2, Bytes: []byte{}, SHA256: sha256.Sum256(nil), Records: []choice.Record{},
		Summary: choice.Summary{Terminal: choice.TerminalComplete},
	}, identity)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-tape-readonly", "GOMADSEED=7"}, Choice: &ChoiceCapability{
			Mode: choice.ModePrefix, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
			ExecutionIdentity: identity, Limit: 1 << 20, ReplayPlan: &tape,
		},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Stdout.Bytes) != "choice tape read-only\n" {
		t.Fatalf("target result = %#v", result)
	}
}

func TestRunRejectsExhaustedChoiceTapeBeforeTargetMarker(t *testing.T) {
	limit := uint64(1 << 20)
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("process choice marker target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, ImplementationSHA256: testChoiceImplementationSHA256,
	}
	request := Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-marker", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	recorded, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(recorded.ChoiceTrace.Trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	emptyTape, err := tape.Prefix(0)
	if err != nil {
		t.Fatal(err)
	}
	request.Choice = &ChoiceCapability{
		Mode: choice.ModeReplay, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
		ExecutionIdentity: identity, Limit: limit, ReplayPlan: &emptyTape,
	}
	diverged, err := Run(context.Background(), request)
	var replayDivergence *ChoiceReplayDivergenceError
	if !errors.As(err, &replayDivergence) || replayDivergence.Divergence.Reason != choice.DivergenceTapeExhausted {
		t.Fatalf("Run() error = %#v", err)
	}
	if strings.Contains(string(diverged.Stdout.Bytes), "post-choice-marker") || diverged.ChoiceTrace.Trace.Summary.Terminal != choice.TerminalDiverged {
		t.Fatalf("diverged result = %#v", diverged)
	}
	request.Choice.Mode = choice.ModePrefix
	prefixed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(prefixed.Stdout.Bytes), "post-choice-marker") || prefixed.ChoiceTrace.Trace.Summary.Terminal != choice.TerminalComplete {
		t.Fatalf("prefixed result = %#v", prefixed)
	}
	request.Choice.Mode = choice.ModeReplay
	request.IO = &IOCapability{Config: []byte("profile-frame"), Transcript: &IOTranscriptCapability{Limit: 1 << 20}}
	diverged, err = Run(context.Background(), request)
	if !errors.As(err, &replayDivergence) || replayDivergence.Divergence.Reason != choice.DivergenceTapeExhausted {
		t.Fatalf("Run() error with incomplete I/O terminal = %#v", err)
	}
	if strings.Contains(string(diverged.Stdout.Bytes), "post-choice-marker") || diverged.ChoiceTrace.Trace.Summary.Terminal != choice.TerminalDiverged {
		t.Fatalf("combined diverged result = %#v", diverged)
	}
}

func TestRunRejectsChoiceMetadataMismatchBeforeTargetMarker(t *testing.T) {
	limit := uint64(1 << 20)
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("process choice mismatch target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, ImplementationSHA256: testChoiceImplementationSHA256,
	}
	request := Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-marker", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	recorded, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	recordIndex, decisionOrdinal := firstBranchingChoiceDecision(t, recorded.ChoiceTrace.Trace)
	for _, test := range []struct {
		name   string
		reason choice.DivergenceReason
		mutate func(*choice.Record)
	}{
		{name: "kind", reason: choice.DivergenceKind, mutate: func(record *choice.Record) {
			if record.Kind == choice.KindRunnable {
				record.Kind = choice.KindSelectPoll
			} else {
				record.Kind = choice.KindRunnable
			}
		}},
		{name: "site", reason: choice.DivergenceSite, mutate: func(record *choice.Record) {
			if record.Flags&choice.FlagSiteMissing != 0 {
				record.Flags &^= choice.FlagSiteMissing
				record.SiteOffset = 1
			} else {
				record.Flags |= choice.FlagSiteMissing
				record.SiteOffset = 0
			}
		}},
		{name: "alternatives", reason: choice.DivergenceAlternatives, mutate: func(record *choice.Record) {
			record.Alternatives++
		}},
		{name: "selected", reason: choice.DivergenceSelected, mutate: func(record *choice.Record) {
			record.Selected = (record.Selected + 1) % record.Alternatives
		}},
		{name: "alternative set", reason: choice.DivergenceAlternativeSet, mutate: func(record *choice.Record) {
			record.AlternativeSetDigest[0] ^= 1
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			tape := projectEditedChoiceTape(t, recorded.ChoiceTrace.Trace, identity, func(records []choice.Record) []choice.Record {
				test.mutate(&records[recordIndex])
				return records
			})
			request.Choice = &ChoiceCapability{
				Mode: choice.ModeReplay, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
				ExecutionIdentity: identity, Limit: limit, ReplayPlan: &tape,
			}
			diverged, err := Run(context.Background(), request)
			var replayDivergence *ChoiceReplayDivergenceError
			if !errors.As(err, &replayDivergence) || replayDivergence.Divergence.Reason != test.reason || replayDivergence.Divergence.Ordinal != decisionOrdinal {
				t.Fatalf("Run() error = %#v, want %s at %d", err, choice.DivergenceReasonName(test.reason), decisionOrdinal)
			}
			if strings.Contains(string(diverged.Stdout.Bytes), "post-choice-marker") || diverged.ChoiceTrace.Trace.Summary.Terminal != choice.TerminalDiverged {
				t.Fatalf("diverged result = %#v", diverged)
			}
		})
	}
}

func TestRunRejectsUnconsumedChoiceTape(t *testing.T) {
	limit := uint64(1 << 20)
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("process unconsumed choice target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, ImplementationSHA256: testChoiceImplementationSHA256,
	}
	request := Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=choice-marker", "GOMADSEED=7"}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: limit},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	recorded, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	tape := projectEditedChoiceTape(t, recorded.ChoiceTrace.Trace, identity, func(records []choice.Record) []choice.Record {
		for index := len(records) - 1; index >= 0; index-- {
			if records[index].Flags&choice.FlagDecision != 0 {
				extra := records[index]
				extra.Ordinal = uint64(len(records))
				return append(records, extra)
			}
		}
		t.Fatal("recorded trace contains no choice decision")
		return nil
	})
	for _, mode := range []choice.Mode{choice.ModeReplay, choice.ModePrefix} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			request.Choice = &ChoiceCapability{
				Mode: mode, Profile: choice.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
				ExecutionIdentity: identity, Limit: limit, ReplayPlan: &tape,
			}
			diverged, err := Run(context.Background(), request)
			var replayDivergence *ChoiceReplayDivergenceError
			if !errors.As(err, &replayDivergence) || replayDivergence.Divergence.Reason != choice.DivergenceTapeUnconsumed {
				t.Fatalf("Run() error = %#v", err)
			}
			if !strings.Contains(string(diverged.Stdout.Bytes), "post-choice-marker") || diverged.ChoiceTrace.Trace.Summary.Terminal != choice.TerminalDiverged {
				t.Fatalf("diverged result = %#v", diverged)
			}
		})
	}
}

func TestRunInstallsReadOnlyMountBrokerDescriptors(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "file"), []byte("contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=io-ro-mount"}, IO: &IOCapability{
			Config: []byte("profile-frame"), Transcript: &IOTranscriptCapability{Limit: 1 << 20},
			ReadOnlyMount: &ReadOnlyMountCapability{Mappings: []romount.Mapping{{Source: source, Target: "/mounted"}}, Limits: romount.DefaultLimits()},
		},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || string(result.Stdout.Bytes) != "contents" || len(result.IOROMounts.Entries) != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunInstallsEmptyReadOnlyMountBrokerDescriptors(t *testing.T) {
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=io-ro-mount-unmounted"}, IO: &IOCapability{
			Config: []byte("profile-frame"), Transcript: &IOTranscriptCapability{Limit: 1 << 20},
			ReadOnlyMount: &ReadOnlyMountCapability{Limits: romount.DefaultLimits()},
		},
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || string(result.Stdout.Bytes) != "unmounted" || len(result.IOROMounts.Entries) != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestValidateRequestRejectsExpectedIOTranscriptOutsideReplay(t *testing.T) {
	request := Spec{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		RunTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
		IO:    &IOCapability{Config: []byte("profile-frame"), Transcript: &IOTranscriptCapability{Limit: 1 << 20}},
	}
	expectedBytes, err := romount.ExpectedTranscriptBytes(1)
	if err != nil {
		t.Fatal(err)
	}
	request.IO.Transcript.Expected = make([]byte, expectedBytes)
	if err := validateSpec(request); err == nil || !strings.Contains(err.Error(), "requires replay mode") {
		t.Fatalf("validateSpec() error = %v", err)
	}
	request.IO.Transcript.Replay = true
	request.IO.Transcript.Expected = make([]byte, expectedBytes-1)
	if err := validateSpec(request); err == nil || !strings.Contains(err.Error(), "invalid expected I/O transcript length") {
		t.Fatalf("validateSpec() error = %v", err)
	}
}

func TestValidateRequestAcceptsNestedExecutionCapabilities(t *testing.T) {
	request := Spec{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		RunTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO: &IOCapability{
			Config:        []byte("profile-frame"),
			Transcript:    &IOTranscriptCapability{Limit: 1 << 20},
			ReadOnlyMount: &ReadOnlyMountCapability{Limits: romount.DefaultLimits()},
		},
	}
	if err := validateSpec(request); err != nil {
		t.Fatal(err)
	}
	request.IO.ReadOnlyMount = nil
	request.IO.Transcript = nil
	if err := validateSpec(request); err != nil {
		t.Fatal(err)
	}
}

func TestRunTimesOutAndRemovesTermIgnoringProcessGroup(t *testing.T) {
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           "/bin/sh",
		Args:              []string{"-c", `trap '' TERM; (trap '' TERM; while :; do :; done) & while :; do :; done`},
		Argv0:             "gomadv3-target",
		Dir:               t.TempDir(),
		Env:               []string{"TZ=UTC"},
		RunTimeout:        750 * time.Millisecond,
		TerminateGrace:    250 * time.Millisecond,
		OutputLimit:       64,
		World:             WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.WatchdogTimeout || result.Termination != TerminationSignal || result.Signal != "killed" || !result.GroupGone {
		t.Fatalf("timeout result = %#v", result)
	}
}

func TestRunBoundsFloodedOutputWithoutBlocking(t *testing.T) {
	stdoutHead, err := os.CreateTemp(t.TempDir(), "stdout-head")
	if err != nil {
		t.Fatal(err)
	}
	defer stdoutHead.Close()
	stderrHead, err := os.CreateTemp(t.TempDir(), "stderr-head")
	if err != nil {
		t.Fatal(err)
	}
	defer stderrHead.Close()
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0],
		Args:              []string{"-test.run=TestTargetHelper"},
		Argv0:             "gomadv3-target",
		Dir:               t.TempDir(),
		Env:               []string{"GOMADV3_PROCESS_HELPER=flood"},
		RunTimeout:        5 * time.Second,
		TerminateGrace:    time.Second,
		OutputLimit:       128,
		World:             WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
		StdoutHead:        stdoutHead,
		StderrHead:        stderrHead,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 {
		t.Fatalf("result status = %#v", result)
	}
	for name, output := range map[string]Output{"stdout": result.Stdout, "stderr": result.Stderr} {
		if !output.Truncated || output.TotalBytes != 1<<20 || output.RetainedBytes != 128 || output.DiscardedBytes != 1<<20-128 {
			t.Fatalf("%s accounting = %#v", name, output)
		}
		if !strings.Contains(string(output.Bytes), "gomadv3 output truncated") {
			t.Fatalf("%s retained bytes omitted marker", name)
		}
	}
	for name, file := range map[string]*os.File{"stdout": stdoutHead, "stderr": stderrHead} {
		info, statErr := file.Stat()
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Size() != 96 {
			t.Fatalf("%s partial head size = %d, want 96", name, info.Size())
		}
	}
}

func TestRunBoundsUnresponsiveSupervisor(t *testing.T) {
	started := time.Now()
	_, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestUnresponsiveSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=output"}, RunTimeout: 300 * time.Millisecond, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if err == nil || !strings.Contains(err.Error(), "supervisor") {
		t.Fatalf("Run() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("unresponsive supervisor took %v", elapsed)
	}
}

func TestRunGivesDescendantsGraceAfterLeaderExit(t *testing.T) {
	directory := t.TempDir()
	readyPath := filepath.Join(directory, "ready")
	gracefulPath := filepath.Join(directory, "graceful")
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: directory,
		Env:        []string{"GOMADV3_PROCESS_HELPER=spawn-child", "GOMADV3_HELPER_EXE=" + os.Args[0], "GOMADV3_READY_PATH=" + readyPath, "GOMADV3_GRACEFUL_PATH=" + gracefulPath},
		RunTimeout: 3 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 || !result.GroupGone {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(gracefulPath); err != nil {
		t.Fatalf("descendant did not exit during TERM grace: %v", err)
	}
}

func TestRunCapturesWorldRecordFromExecutingChild(t *testing.T) {
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADSEED=9", "GOMADV3_PROCESS_HELPER=world-record"}, RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: 1 << 20, Seed: 9},
	})
	if err != nil {
		t.Fatal(err)
	}
	recording, err := world.DecodeRecording(result.WorldRecord)
	if err != nil {
		t.Fatal(err)
	}
	if recording.Initial.Config.Seed != 9 || len(recording.Final.Transitions)-len(recording.Initial.Transitions) != 2 || recording.Terminal.Kind != world.TerminalDeadlock {
		t.Fatalf("child World recording = %#v", recording)
	}
}

func TestRunPreservesPrematureWorldProducerMarker(t *testing.T) {
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADSEED=9", "GOMADV3_PROCESS_HELPER=world-open-only"}, RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: 1 << 20, Seed: 9},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.WorldRecord) == 0 {
		t.Fatal("premature World producer was indistinguishable from an unconnected target")
	}
	if _, err := world.DecodeRecording(result.WorldRecord); err == nil {
		t.Fatal("premature World producer emitted a complete record")
	}
}

func TestRunPreservesWorldSeedMismatchMarker(t *testing.T) {
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADSEED=8", "GOMADV3_PROCESS_HELPER=world-record"}, RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: 1 << 20, Seed: 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 15 || len(result.WorldRecord) == 0 {
		t.Fatalf("seed-mismatch result = %#v", result)
	}
	if _, err := world.DecodeRecording(result.WorldRecord); err == nil {
		t.Fatal("World seed mismatch emitted a complete record")
	}
}

func TestRunInstallsWorldInitialReplayInputBeforeModeledWork(t *testing.T) {
	limits := world.Limits{MaxRequests: 11, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}
	core, err := world.New(world.Config{Seed: 9, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	expectedInitial, err := world.EncodeSnapshot(core.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADSEED=9", "GOMADV3_PROCESS_HELPER=world-record"}, RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: 1 << 20, Seed: 9, ExpectedInitial: expectedInitial},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("trusted-initial result = %#v", result)
	}
	recording, err := world.DecodeRecording(result.WorldRecord)
	if err != nil {
		t.Fatal(err)
	}
	if recording.Initial.Config.Limits.MaxRequests != 11 {
		t.Fatalf("initial max requests = %d, want 11", recording.Initial.Config.Limits.MaxRequests)
	}
}

func TestSupervisorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	if err := SupervisorMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestTargetBootstrapHelper(t *testing.T) {
	if os.Getenv("GOMADV3_TARGET_BOOTSTRAP") != "1" {
		t.Skip("target bootstrap subprocess only")
	}
	if err := BootstrapMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestTargetHelper(t *testing.T) {
	switch os.Getenv("GOMADV3_PROCESS_HELPER") {
	case "output":
		fmt.Fprintln(os.Stdout, "target stdout")
		fmt.Fprintln(os.Stderr, "target stderr")
		os.Exit(7)
	case "flood":
		if _, err := os.Stdout.Write(make([]byte, 1<<20)); err != nil {
			os.Exit(8)
		}
		if _, err := os.Stderr.Write(make([]byte, 1<<20)); err != nil {
			os.Exit(9)
		}
		os.Exit(0)
	case "io-config":
		configuration := os.NewFile(5, "gomadv3-io-config")
		if configuration == nil {
			os.Exit(21)
		}
		if _, err := io.Copy(os.Stdout, configuration); err != nil {
			os.Exit(22)
		}
		if err := configuration.Close(); err != nil {
			os.Exit(23)
		}
		os.Exit(0)
	case "choice-trace":
		os.Exit(0)
	case "choice-marker":
		fmt.Fprintln(os.Stdout, "post-choice-marker")
		os.Exit(0)
	case "choice-reorder":
		runChoiceReorderTarget()
		os.Exit(0)
	case "choice-select":
		runChoiceSelectTarget()
		os.Exit(0)
	case "choice-prefix-rng":
		runChoicePrefixRNGTarget()
		os.Exit(0)
	case "choice-tape-readonly":
		descriptor, err := strconv.Atoi(os.Getenv(choiceTapeFDEnvironmentName))
		if err != nil {
			os.Exit(32)
		}
		if _, err := syscall.Pwrite(descriptor, []byte{1}, 0); err == nil {
			os.Exit(34)
		}
		if err := syscall.Ftruncate(descriptor, 1); err == nil {
			os.Exit(35)
		}
		fmt.Fprintln(os.Stdout, "choice tape read-only")
		os.Exit(0)
	case "io-ro-mount":
		request := os.NewFile(9, "gomadv3-io-ro-mount-request")
		response := os.NewFile(10, "gomadv3-io-ro-mount-response")
		if request == nil || response == nil || romount.WriteLookupRequest(request, 0, "/mounted/file") != nil {
			os.Exit(24)
		}
		entry, err := romount.ReadResponse(response, romount.DefaultLimits())
		if err != nil || entry.Status != romount.StatusOK {
			os.Exit(25)
		}
		if _, err := os.Stdout.Write(entry.Entry.Data); err != nil {
			os.Exit(26)
		}
		if err := writeEmptyIOTranscriptTerminal(); err != nil {
			os.Exit(27)
		}
		os.Exit(0)
	case "io-ro-mount-unmounted":
		request := os.NewFile(9, "gomadv3-io-ro-mount-request")
		response := os.NewFile(10, "gomadv3-io-ro-mount-response")
		if request == nil || response == nil || romount.WriteLookupRequest(request, 0, "/outside") != nil {
			os.Exit(28)
		}
		entry, err := romount.ReadResponse(response, romount.DefaultLimits())
		if err != nil || entry.Status != romount.StatusUnmounted {
			os.Exit(29)
		}
		if _, err := os.Stdout.WriteString("unmounted"); err != nil {
			os.Exit(30)
		}
		if err := writeEmptyIOTranscriptTerminal(); err != nil {
			os.Exit(31)
		}
		os.Exit(0)
	case "spawn-child":
		command := exec.Command(os.Getenv("GOMADV3_HELPER_EXE"), "-test.run=TestTargetHelper")
		command.Env = []string{
			"GOMADV3_PROCESS_HELPER=graceful-child",
			"GOMADV3_READY_PATH=" + os.Getenv("GOMADV3_READY_PATH"),
			"GOMADV3_GRACEFUL_PATH=" + os.Getenv("GOMADV3_GRACEFUL_PATH"),
		}
		if err := command.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(10)
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			if _, err := os.Stat(os.Getenv("GOMADV3_READY_PATH")); err == nil {
				os.Exit(0)
			}
			if time.Now().After(deadline) {
				os.Exit(11)
			}
			runtime.Gosched()
		}
	case "graceful-child":
		signal.Ignore(syscall.SIGTERM)
		if err := os.WriteFile(os.Getenv("GOMADV3_READY_PATH"), []byte("ready"), 0o600); err != nil {
			os.Exit(12)
		}
		<-time.After(250 * time.Millisecond)
		if err := os.WriteFile(os.Getenv("GOMADV3_GRACEFUL_PATH"), []byte("graceful"), 0o600); err != nil {
			os.Exit(13)
		}
		os.Exit(0)
	case "world-record":
		core, err := world.New(world.Config{Seed: 9, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(14)
		}
		session, err := worldtarget.Open(core)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(15)
		}
		core = session.Model()
		if _, err := core.Register(world.Request{Kind: "wait", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(16)
		}
		if _, err := core.Quiesce(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(17)
		}
		if err := session.Finish(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(20)
		}
		os.Exit(0)
	case "world-open-only":
		core, err := world.New(world.Config{Seed: 9, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
		if err != nil {
			os.Exit(18)
		}
		if _, err := worldtarget.Open(core); err != nil {
			os.Exit(19)
		}
		os.Exit(0)
	case "simulation-coordinator":
		runSimulationProcessHelper()
		os.Exit(0)
	default:
		t.Skip("target subprocess only")
	}
}

func runSimulationProcessHelper() {
	requestDescriptor, err := strconv.Atoi(os.Getenv(simulationRequestFDEnvironmentName))
	if err != nil {
		os.Exit(40)
	}
	responseDescriptor, err := strconv.Atoi(os.Getenv(simulationResponseFDEnvironmentName))
	if err != nil {
		os.Exit(41)
	}
	request := os.NewFile(uintptr(requestDescriptor), "simulation-request")
	response := os.NewFile(uintptr(responseDescriptor), "simulation-response")
	if request == nil || response == nil {
		os.Exit(42)
	}
	if os.Getenv(simulationRoleEnvironmentName) == string(SimulationRoleNode) {
		bootstrapDescriptor, bootstrapErr := strconv.Atoi(os.Getenv(simulationBootstrapFDEnvironmentName))
		if bootstrapErr != nil {
			os.Exit(43)
		}
		bootstrap := os.NewFile(uintptr(bootstrapDescriptor), "simulation-bootstrap")
		if bootstrap == nil {
			os.Exit(44)
		}
		payload, readErr := io.ReadAll(io.LimitReader(bootstrap, maximumSimulationBootstrapBytes+1))
		wantBootstrap := "node-bootstrap"
		if os.Getenv("GOMADV3_SIMULATION_CASE") == "crash" {
			wantBootstrap = "crash-bootstrap"
		}
		if readErr != nil || string(payload) != wantBootstrap {
			os.Exit(45)
		}
		frames := []simulationFrame{{Kind: simulationFrameReady, Request: 1, Node: "node", Incarnation: 1}, {Kind: simulationFrameActivated, Request: 2, Node: "node", Incarnation: 1}}
		if os.Getenv("GOMADV3_SIMULATION_CASE") != "crash" {
			frames = append(frames, simulationFrame{Kind: simulationFrameTerminal, Request: 3, Node: "node", Incarnation: 1, Payload: []byte("node-terminal")})
		}
		for _, frame := range frames {
			frame.Profile = simulationProtocol
			if writeErr := writeSimulationFrame(request, frame); writeErr != nil {
				os.Exit(46)
			}
			if _, readErr := readSimulationFrame(response); readErr != nil {
				os.Exit(47)
			}
		}
		if os.Getenv("GOMADV3_SIMULATION_CASE") == "crash" {
			select {}
		}
		return
	}
	frames := []simulationFrame{{Kind: simulationFrameStart, Request: 1, Node: "node", Incarnation: 1, Payload: []byte("node-bootstrap")}, {Kind: simulationFrameActivate, Request: 2, Node: "node", Incarnation: 1}, {Kind: simulationFrameWait, Request: 3, Node: "node", Incarnation: 1}}
	if os.Getenv("GOMADV3_SIMULATION_CASE") == "crash" {
		frames = []simulationFrame{{Kind: simulationFrameStart, Request: 1, Node: "node", Incarnation: 1, Payload: []byte("crash-bootstrap")}, {Kind: simulationFrameActivate, Request: 2, Node: "node", Incarnation: 1}, {Kind: simulationFrameCrash, Request: 3, Node: "node", Incarnation: 1}}
	}
	for _, frame := range frames {
		frame.Profile = simulationProtocol
		if writeErr := writeSimulationFrame(request, frame); writeErr != nil {
			os.Exit(48)
		}
		answer, readErr := readSimulationFrame(response)
		if readErr != nil || answer.Error != "" {
			os.Exit(49)
		}
		if frame.Kind == simulationFrameWait && string(answer.Payload) != "node-terminal" {
			os.Exit(50)
		}
	}
}

func runChoiceReorderTarget() {
	ready := make(chan struct{}, 3)
	firstStart := make(chan struct{})
	secondStart := make(chan struct{})
	sentinelStart := make(chan struct{})
	results := make(chan string, 2)
	go func() {
		ready <- struct{}{}
		<-firstStart
		results <- "a"
	}()
	go func() {
		ready <- struct{}{}
		<-secondStart
		results <- "b"
	}()
	go func() {
		ready <- struct{}{}
		<-sentinelStart
	}()
	for range 3 {
		<-ready
	}
	if os.Getenv("GOMADV3_CHOICE_REORDER") == "ba" {
		close(secondStart)
		close(firstStart)
	} else {
		close(firstStart)
		close(secondStart)
	}
	close(sentinelStart)
	fmt.Fprintln(os.Stdout, <-results+<-results)
}

func runChoiceSelectTarget() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	fmt.Fprintln(os.Stdout, runChoiceSelectSequence(8))
}

func runChoicePrefixRNGTarget() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	fmt.Fprintln(os.Stdout, runChoiceSelectSequence(8))
}

func runChoiceSelectSequence(iterations int) string {
	left := make(chan struct{}, 1)
	right := make(chan struct{}, 1)
	left <- struct{}{}
	right <- struct{}{}
	var output strings.Builder
	for range iterations {
		select {
		case <-left:
			output.WriteByte('a')
			left <- struct{}{}
		case <-right:
			output.WriteByte('b')
			right <- struct{}{}
		}
	}
	return output.String()
}

func compatibleSelectDifference(baseline, alternate []choice.Decision) bool {
	if len(baseline) != len(alternate) {
		return false
	}
	for index := range baseline {
		left := baseline[index]
		right := alternate[index]
		if left.Kind != right.Kind || left.SiteOffset != right.SiteOffset || left.SiteMissing != right.SiteMissing || left.Alternatives != right.Alternatives || left.Data != right.Data || left.AlternativeSetDigest != right.AlternativeSetDigest {
			return false
		}
		if left.Kind == choice.KindSelectPoll && left.Alternatives > 1 && index+1 < len(baseline) && (left.Selected != right.Selected || left.SelectedIdentity != right.SelectedIdentity) {
			return true
		}
	}
	return false
}

func firstChoiceDecisionDifference(left, right []choice.Decision) int {
	if len(left) != len(right) {
		return min(len(left), len(right))
	}
	for index := range left {
		if left[index] != right[index] {
			return index
		}
	}
	return -1
}

func firstBranchingChoiceDecision(t *testing.T, trace choice.Trace) (int, uint64) {
	t.Helper()
	var ordinal uint64
	for index, record := range trace.Records {
		if record.Flags&choice.FlagDecision == 0 {
			continue
		}
		if record.Alternatives > 1 {
			return index, ordinal
		}
		ordinal++
	}
	t.Fatal("recorded trace contains no branching choice decision")
	return 0, 0
}

func projectEditedChoiceTape(t *testing.T, trace choice.Trace, identity choice.ExecutionIdentity, edit func([]choice.Record) []choice.Record) choice.ReplayPlan {
	t.Helper()
	records := edit(append([]choice.Record(nil), trace.Records...))
	trace, err := choice.BuildTrace(records, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	return tape
}

func writeEmptyIOTranscriptTerminal() error {
	file := os.NewFile(7, "gomadv3-io-terminal")
	if file == nil {
		return errors.New("terminal descriptor unavailable")
	}
	if err := romount.WriteCompletion(file, romount.Transcript{Complete: true, SHA256: sha256.Sum256(nil)}); err != nil {
		return err
	}
	return file.Close()
}

func TestUnresponsiveSupervisorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	for {
	}
}
