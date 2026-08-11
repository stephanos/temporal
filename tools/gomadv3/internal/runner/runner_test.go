package runner

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
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
	"go.temporal.io/server/tools/gomadv3/world"
)

func TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{result: func(seed uint64) process.Result {
		if seed%2 == 0 {
			return processResult(2, "same failure", "")
		}
		return processResult(0, "success", "")
	}}
	summary, err := Run(context.Background(), testConfig(t, preparer, executor, "1-6", PolicyAll, 2))
	if err != nil {
		t.Fatal(err)
	}
	if preparer.calls != 1 {
		t.Fatalf("preparation calls = %d, want 1", preparer.calls)
	}
	if executor.maximumActive > 2 {
		t.Fatalf("maximum active = %d, want <= 2", executor.maximumActive)
	}
	if summary.Attempted != 6 || summary.Succeeded != 3 || summary.Failures != 3 || summary.DistinctFailures != 1 || summary.StopReason != StopSeedsExhausted {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.Artifacts) != 1 {
		t.Fatalf("artifacts = %v", summary.Artifacts)
	}
	if _, err := artifact.Open(summary.Artifacts[0]); err != nil {
		t.Fatal(err)
	}
	opened, err := artifact.Open(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.Target.BuildTags == nil {
		t.Fatal("empty target build tags encoded as null")
	}
	if _, err := os.Stat(filepath.Join(summary.BatchPath, "batch.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(summary.BatchPath, ".prepared")); !os.IsNotExist(err) {
		t.Fatalf("prepared target retained after publication: %v", err)
	}
	if got := executor.environments(); len(got) != 6 {
		t.Fatalf("target environments = %v", got)
	} else {
		for _, environment := range got {
			if len(environment) != 3 || environment[1] != "MODE=test" || environment[2] != "TZ=UTC" || !strings.HasPrefix(environment[0], "GOMADSEED=") {
				t.Fatalf("target environment = %v", environment)
			}
		}
	}
	if !allUnique(executor.directories()) {
		t.Fatalf("working directories were reused: %v", executor.directories())
	}
}

func TestRunResolvesRelativeArtifactRootBeforeTargetPreparation(t *testing.T) {
	workingDirectory := t.TempDir()
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workingDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDirectory); err != nil {
			t.Error(err)
		}
	})
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{}
	config := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Artifacts = "artifacts"
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(summary.BatchPath) {
		t.Fatalf("batch path = %q, want absolute path", summary.BatchPath)
	}
	for _, directory := range executor.directories() {
		if !filepath.IsAbs(directory) {
			t.Fatalf("target working directory = %q, want absolute path", directory)
		}
	}
}

func TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := newFirstFailureExecutor(3)
	config := testConfig(t, preparer, executor, "1-10", PolicyFirst, 3)
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 3 || summary.Failures != 1 || summary.Cancelled != 2 || summary.DistinctFailures != 1 || summary.StopReason != StopFirstFailure {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.Artifacts) != 1 {
		t.Fatalf("artifacts = %v", summary.Artifacts)
	}
	partials, err := os.ReadDir(filepath.Join(summary.BatchPath, ".partial"))
	if err != nil {
		t.Fatal(err)
	}
	if len(partials) != 2 {
		t.Fatalf("cancelled target partials = %v, want 2", partials)
	}
}

func TestRunBudgetCountsDistinctSignatures(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{result: func(seed uint64) process.Result {
		output := "same"
		if seed == 4 {
			output = "different"
		}
		return processResult(1, output, "")
	}}
	config := testConfig(t, preparer, executor, "1-10", PolicyBudget, 1)
	config.FailureBudget = 2
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 4 || summary.Failures != 4 || summary.DistinctFailures != 2 || summary.StopReason != StopFailureBudget {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestRunPublishesConnectedWorldBundle(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	recording, err := world.EncodeRecording(world.Recording{Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalIdle}})
	if err != nil {
		t.Fatal(err)
	}
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(1, "failure", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := artifact.Open(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.World.Initial.Schema != "gomadv3.world.snapshot/v1" || opened.Manifest.World.Transitions.Count != 1 || opened.Manifest.World.Terminal.Kind != "idle" {
		t.Fatalf("recorded World = %#v", opened.Manifest.World)
	}
}

func TestRunClassifiesConnectedWorldDeadlock(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Register(world.Request{Kind: "wait", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}}); err != nil {
		t.Fatal(err)
	}
	if result, err := core.Quiesce(); err != nil || result.Kind != world.QuiescenceDeadlock {
		t.Fatalf("Quiesce() = %#v, %v", result, err)
	}
	recording, err := world.EncodeRecording(world.Recording{Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalDeadlock}})
	if err != nil {
		t.Fatal(err)
	}
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Failures != 1 || len(summary.Artifacts) != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	opened, err := artifact.Open(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.Outcome.Reason != "world_deadlock" || opened.Manifest.World.Terminal.Kind != "deadlock" {
		t.Fatalf("World deadlock manifest = %#v", opened.Manifest)
	}
}

func TestRunRejectsInvalidConnectedWorldBeforePublication(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	recording, err := world.EncodeRecording(world.Recording{Initial: initial, Final: initial, Terminal: world.Terminal{Kind: world.TerminalInvalidInput, Detail: "fixture"}})
	if err != nil {
		t.Fatal(err)
	}
	recording[len(recording)-1] ^= 1
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(1, "failure", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "world_record" {
		t.Fatalf("Run() error = %#v", err)
	}
	if len(summary.Artifacts) != 1 {
		t.Fatalf("Runner failure artifacts = %v, want 1", summary.Artifacts)
	}
	opened, openErr := artifact.Open(summary.Artifacts[0])
	if openErr != nil {
		t.Fatal(openErr)
	}
	if opened.Manifest.ArtifactKind != record.ArtifactRunnerFailure || opened.Manifest.Outcome.Reason != "world_record" || opened.Manifest.ReplayMode != record.ReplayNone {
		t.Fatalf("Runner failure manifest = %#v", opened.Manifest)
	}
}

func TestRunRejectsPreparedTargetMutationBeforeFailurePublication(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), mutatingExecutor{}, "1", PolicyAll, 1)
	summary, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "prepared_target_integrity" {
		t.Fatalf("Run() error = %#v", err)
	}
	if len(summary.Artifacts) != 0 {
		t.Fatalf("target mutation published replayable artifacts: %v", summary.Artifacts)
	}
}

func TestRunRejectsIOProfileWhenPreparedTargetDoesNotMatch(t *testing.T) {
	preparer := newFakePreparer(t)
	config := testConfig(t, preparer, &fakeExecutor{}, "1", PolicyFirst, 1)
	config.IOProfile = "temporal-activity-api-batch-cancel/v1"
	config.Environment = nil
	config.Target = target.Spec{
		Kind: target.KindGoTest, Source: "./tests", Args: []string{"-test.run=^TestActivityAPIBatchCancelClientTestSuite$"},
	}
	_, err := Run(context.Background(), config)
	if err == nil || !strings.Contains(err.Error(), "requires a go-test target") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestValidateConfigRejectsUnknownIOProfileAndProfileEnvironment(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyFirst, 1)
	config.IOProfile = "unknown/v1"
	if _, _, err := validateConfig(config); err == nil || !strings.Contains(err.Error(), "unknown I/O profile") {
		t.Fatalf("validateConfig(unknown profile) error = %v", err)
	}

	config.IOProfile = "temporal-activity-api-batch-cancel/v1"
	if _, _, err := validateConfig(config); err == nil || !strings.Contains(err.Error(), "does not accept target environment") {
		t.Fatalf("validateConfig(profile environment) error = %v", err)
	}
}

func TestRunRejectsReservedDuplicateAndInvalidEnvironment(t *testing.T) {
	for name, environment := range map[string][]string{
		"reserved":  {"GOMAXPROCS=2"},
		"duplicate": {"A=1", "A=2"},
		"invalid":   {"NOT-VALID=1"},
		"nul":       {"A=value\x00tail"},
	} {
		t.Run(name, func(t *testing.T) {
			config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
			config.Environment = environment
			if _, err := Run(context.Background(), config); err == nil {
				t.Fatal("Run() succeeded")
			}
		})
	}
}

func TestRunOverallTimeoutIsAHostFailure(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), blockingExecutor{}, "1", PolicyAll, 1)
	config.OverallTimeout = 50 * time.Millisecond
	config.TerminateGrace = 10 * time.Millisecond
	summary, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("Run() error = %#v", err)
	}
	if summary.Failures != 0 || len(summary.Artifacts) != 0 {
		t.Fatalf("overall timeout summary = %#v", summary)
	}
	partials, readErr := os.ReadDir(filepath.Join(summary.BatchPath, ".partial"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(partials) != 2 {
		t.Fatalf("overall-timeout partials = %v, want batch and target", partials)
	}
	if _, err := os.Stat(filepath.Join(summary.BatchPath, ".partial", "batch", "partial.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRunPreparationFailureLeavesExplicitPartial(t *testing.T) {
	config := testConfig(t, errorPreparer{err: errors.New("build failed")}, &fakeExecutor{}, "1", PolicyAll, 1)
	summary, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "target_preparation" {
		t.Fatalf("Run() error = %#v", err)
	}
	partial, readErr := os.ReadFile(filepath.Join(summary.BatchPath, ".partial", "preparation", "partial.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(partial), `"state":"failed"`) || !strings.Contains(string(partial), `"reason":"target_preparation"`) {
		t.Fatalf("preparation partial = %s", partial)
	}
}

func TestRunPreparationOverallTimeoutIsClassifiedSeparately(t *testing.T) {
	config := testConfig(t, waitingPreparer{}, &fakeExecutor{}, "1", PolicyAll, 1)
	config.OverallTimeout = 25 * time.Millisecond
	config.TerminateGrace = 10 * time.Millisecond
	summary, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("Run() error = %#v", err)
	}
	partial, readErr := os.ReadFile(filepath.Join(summary.BatchPath, ".partial", "preparation", "partial.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(partial), `"reason":"overall_timeout"`) {
		t.Fatalf("preparation partial = %s", partial)
	}
}

func TestClassifyStableTargetDiagnostics(t *testing.T) {
	for name, stderr := range map[string]string{
		"panic_or_runtime_fatal":         "panic: broken\n",
		"deterministic_deadlock":         "fatal error: all goroutines are asleep - deadlock!\n",
		"logical_test_timeout":           "panic: test timed out after 1m0s\n",
		"unsupported_deterministic_mode": "runtime: GOMADSEED does not support cgo or external linking\n",
	} {
		t.Run(name, func(t *testing.T) {
			outcome := classify(processResult(2, "", stderr), false, record.WorldTerminal{Kind: "none"})
			if outcome.Domain != "target" || outcome.Reason != name {
				t.Fatalf("outcome = %#v", outcome)
			}
		})
	}
}

func TestManifestForRunBindsIOProfileIdentity(t *testing.T) {
	preparer := newFakePreparer(t)
	config := testConfig(t, preparer, &fakeExecutor{}, "1", PolicyFirst, 1)
	config.IOProfile = "temporal-activity-api-batch-cancel/v1"
	manifest, err := manifestForRun(config, preparer.prepared, nil, runCompletion{
		job: runJob{seed: 1}, startedAt: time.Unix(1, 0), finishedAt: time.Unix(2, 0), result: processResult(1, "", ""),
	}, classifiedOutcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}, "run", record.World{})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.IOProfile.Name != config.IOProfile || manifest.IOProfile.Inventory == "" || manifest.IOProfile.InventorySHA256 == "" || manifest.IOProfile.ImplementationSHA256 == "" {
		t.Fatalf("manifest I/O profile = %#v", manifest.IOProfile)
	}
}

func TestClassifyStructuredWorldFailures(t *testing.T) {
	for kind, reason := range map[string]string{
		"deadlock": "world_deadlock", "capacity": "world_capacity", "replay-divergence": "world_replay_divergence", "invalid-input": "world_invalid_input",
	} {
		t.Run(kind, func(t *testing.T) {
			outcome := classify(processResult(0, "", ""), false, record.WorldTerminal{Kind: kind, Detail: "detail"})
			if outcome.Domain != "target" || outcome.Reason != reason || outcome.Termination != "exit" || outcome.ExitCode == nil || *outcome.ExitCode != 0 {
				t.Fatalf("classify() = %#v", outcome)
			}
		})
	}
}

func TestIsolatedRunnerKillsAStuckCoordinatorInsideOverallDeadline(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.OverallTimeout = 200 * time.Millisecond
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestBlockingCoordinatorHelper"}
	started := time.Now()
	_, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("Run() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 350*time.Millisecond {
		t.Fatalf("Run() elapsed = %v", elapsed)
	}
}

func TestBlockingCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	for {
		runtime.Gosched()
	}
}

func TestIsolatedRunnerBoundsCoordinatorOutput(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestOversizedCoordinatorHelper"}
	_, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "coordinator_decode" {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestOversizedCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	if _, err := os.Stdout.Write(make([]byte, maximumCoordinatorMessageBytes+1)); err != nil {
		t.Fatal(err)
	}
}

func TestIsolatedRunnerRemovesCoordinatorProcessGroup(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "descendant-survived")
	t.Setenv("GOMADV3_COORDINATOR_DESCENDANT_MARKER", marker)
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.OverallTimeout = 250 * time.Millisecond
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestCoordinatorTreeHelper"}
	_, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("Run() error = %v", err)
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(marker); statErr == nil {
			t.Fatal("coordinator descendant survived cleanup")
		} else if !os.IsNotExist(statErr) {
			t.Fatal(statErr)
		}
		runtime.Gosched()
	}
}

func TestCoordinatorTreeHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	command := exec.Command(os.Args[0], "-test.run=TestCoordinatorDescendantHelper")
	command.Env = append(os.Environ(), "GOMADV3_COORDINATOR_DESCENDANT=1")
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	for {
		runtime.Gosched()
	}
}

func TestCoordinatorDescendantHelper(t *testing.T) {
	if os.Getenv("GOMADV3_COORDINATOR_DESCENDANT") != "1" {
		t.Skip("coordinator descendant subprocess only")
	}
	signal.Ignore(syscall.SIGTERM)
	<-time.After(350 * time.Millisecond)
	if err := os.WriteFile(os.Getenv("GOMADV3_COORDINATOR_DESCENDANT_MARKER"), []byte("survived"), 0o600); err != nil {
		t.Fatal(err)
	}
}

type fakePreparer struct {
	prepared target.Prepared
	calls    int
}

type errorPreparer struct {
	err error
}

func (preparer errorPreparer) Prepare(context.Context, target.Spec) (target.Prepared, error) {
	return target.Prepared{}, preparer.err
}

type waitingPreparer struct{}

func (waitingPreparer) Prepare(ctx context.Context, _ target.Spec) (target.Prepared, error) {
	<-ctx.Done()
	return target.Prepared{}, ctx.Err()
}

func newFakePreparer(t *testing.T) *fakePreparer {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target")
	data := []byte("fake prepared target")
	if err := os.WriteFile(path, data, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o500); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return &fakePreparer{prepared: target.Prepared{
		Path: path, Kind: target.KindGoRun, Source: ".", SHA256: fmt.Sprintf("sha256:%x", digest), Size: uint64(len(data)),
		Argv: []string{"gomadv3-target"}, BuildTags: []string{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
		GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e",
		TargetGOOS: "darwin", TargetGOARCH: "arm64",
	}}
}

func (preparer *fakePreparer) Prepare(_ context.Context, spec target.Spec) (target.Prepared, error) {
	preparer.calls++
	if err := os.MkdirAll(spec.PreparationRoot, 0o700); err != nil {
		return target.Prepared{}, err
	}
	data, err := os.ReadFile(preparer.prepared.Path)
	if err != nil {
		return target.Prepared{}, err
	}
	path := filepath.Join(spec.PreparationRoot, "target")
	if err := os.WriteFile(path, data, 0o500); err != nil {
		return target.Prepared{}, err
	}
	if err := os.Chmod(path, 0o500); err != nil {
		return target.Prepared{}, err
	}
	prepared := preparer.prepared
	prepared.Path = path
	return prepared, nil
}

type fakeExecutor struct {
	mu            sync.Mutex
	active        int
	maximumActive int
	requests      []process.Request
	result        func(uint64) process.Result
}

func (executor *fakeExecutor) Run(_ context.Context, request process.Request) (process.Result, error) {
	executor.mu.Lock()
	executor.active++
	if executor.active > executor.maximumActive {
		executor.maximumActive = executor.active
	}
	executor.requests = append(executor.requests, request)
	executor.mu.Unlock()
	seed := seedFromEnvironment(request.Env)
	result := processResult(0, "", "")
	if executor.result != nil {
		result = executor.result(seed)
	}
	executor.mu.Lock()
	executor.active--
	executor.mu.Unlock()
	return result, nil
}

func (executor *fakeExecutor) environments() [][]string {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	result := make([][]string, len(executor.requests))
	for index, request := range executor.requests {
		result[index] = append([]string(nil), request.Env...)
	}
	return result
}

func (executor *fakeExecutor) directories() []string {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	result := make([]string, len(executor.requests))
	for index, request := range executor.requests {
		result[index] = request.Dir
	}
	return result
}

type firstFailureExecutor struct {
	started chan struct{}
	once    sync.Once
	want    int
	mu      sync.Mutex
	count   int
}

type blockingExecutor struct{}

type mutatingExecutor struct{}

func (mutatingExecutor) Run(_ context.Context, request process.Request) (process.Result, error) {
	if err := os.Chmod(request.Command, 0o700); err != nil {
		return process.Result{}, err
	}
	file, err := os.OpenFile(request.Command, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return process.Result{}, err
	}
	if _, err := file.Write([]byte("mutation")); err != nil {
		file.Close()
		return process.Result{}, err
	}
	if err := file.Close(); err != nil {
		return process.Result{}, err
	}
	return processResult(1, "failure", ""), nil
}

func (blockingExecutor) Run(ctx context.Context, _ process.Request) (process.Result, error) {
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = process.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func newFirstFailureExecutor(want int) *firstFailureExecutor {
	return &firstFailureExecutor{started: make(chan struct{}), want: want}
}

func (executor *firstFailureExecutor) Run(ctx context.Context, request process.Request) (process.Result, error) {
	seed := seedFromEnvironment(request.Env)
	executor.mu.Lock()
	executor.count++
	if executor.count == executor.want {
		executor.once.Do(func() { close(executor.started) })
	}
	executor.mu.Unlock()
	<-executor.started
	if seed == 1 {
		return processResult(1, "first failure", ""), nil
	}
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = process.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func testConfig(t *testing.T, preparer Preparer, executor Executor, seeds string, policy FailurePolicy, parallel int) Config {
	t.Helper()
	return Config{
		Seeds: seeds, Parallel: parallel, RunTimeout: time.Second, OverallTimeout: 10 * time.Second, TerminateGrace: 100 * time.Millisecond,
		OnFailure: policy, FailureBudget: 1, OutputLimit: 64, WorldTransitionLimit: 64, Artifacts: t.TempDir(),
		Environment: []string{"MODE=test"}, Target: target.Spec{Kind: target.KindGoRun, Source: "."}, SupervisorCommand: []string{"unused"},
		RunnerBuild: "test", Preparer: preparer, Executor: executor,
	}
}

func processResult(exitCode int, stdout, stderr string) process.Result {
	return process.Result{
		Captured: true, Termination: process.TerminationExit, ExitCode: exitCode, GroupGone: true,
		Stdout: output(stdout), Stderr: output(stderr),
	}
}

func output(value string) process.Output {
	bytes := []byte(value)
	digest := sha256.Sum256(bytes)
	return process.Output{Bytes: bytes, FullSHA256: digest, RetainedSHA256: digest, TotalBytes: uint64(len(bytes)), RetainedBytes: uint64(len(bytes))}
}

func seedFromEnvironment(environment []string) uint64 {
	for _, entry := range environment {
		if value, found := strings.CutPrefix(entry, "GOMADSEED="); found {
			seed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				panic(err)
			}
			return seed
		}
	}
	panic("missing GOMADSEED")
}

func allUnique(values []string) bool {
	sort.Strings(values)
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return false
		}
	}
	return true
}
