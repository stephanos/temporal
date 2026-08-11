package replay

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
	"go.temporal.io/server/tools/gomadv3/internal/worldrecord"
	"go.temporal.io/server/tools/gomadv3/world"
)

type Executor interface {
	Run(context.Context, process.Request) (process.Result, error)
}

type Config struct {
	ArtifactPath      string
	VerifyOnly        bool
	ToolchainRoot     string
	SupervisorCommand []string
	BootstrapCommand  []string
	Executor          Executor
}

type Result struct {
	Artifact   artifact.Artifact
	Verified   bool
	Match      bool
	Diagnostic bool
	Divergence string
}

type PreflightError struct {
	Err error
}

func (err *PreflightError) Error() string {
	return "incompatible replay artifact: " + err.Err.Error()
}

func (err *PreflightError) Unwrap() error {
	return err.Err
}

type processExecutor struct{}

func (processExecutor) Run(ctx context.Context, request process.Request) (process.Result, error) {
	return process.Run(ctx, request)
}

func Replay(ctx context.Context, config Config) (result Result, retErr error) {
	opened, err := preflight(config)
	if err != nil {
		return Result{}, &PreflightError{Err: err}
	}
	defer func() {
		if closeErr := opened.Close(); closeErr != nil && retErr == nil {
			result = Result{}
			retErr = fmt.Errorf("close replay artifact: %w", closeErr)
		}
	}()
	result = Result{Artifact: opened.Detached(), Verified: true, Diagnostic: opened.Manifest.ReplayMode == record.ReplayDiagnostic}
	if config.VerifyOnly {
		return result, nil
	}
	executor := config.Executor
	if executor == nil {
		if len(config.SupervisorCommand) == 0 {
			return Result{}, fmt.Errorf("supervisor command is required")
		}
		executor = processExecutor{}
	}
	workDirectory, err := os.MkdirTemp("", "gomadv3-replay-")
	if err != nil {
		return Result{}, fmt.Errorf("create replay working directory: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(workDirectory); cleanupErr != nil && retErr == nil {
			result = Result{}
			retErr = fmt.Errorf("remove replay working directory: %w", cleanupErr)
		}
	}()
	if err := os.Chmod(workDirectory, 0o700); err != nil {
		return Result{}, fmt.Errorf("make replay working directory private: %w", err)
	}
	manifest := opened.Manifest
	targetPath := filepath.Join(workDirectory, "target")
	if err := artifact.CopyPayload(opened, manifest.Target.File, targetPath, 0o500); err != nil {
		return Result{}, fmt.Errorf("copy verified replay target: %w", err)
	}
	if err := validateTargetBuildInfo(targetPath, manifest.Target.BuildInfo); err != nil {
		return Result{}, err
	}
	runTimeout, err := duration(manifest.Limits.RunTimeoutNanos)
	if err != nil {
		return Result{}, err
	}
	terminateGrace, err := duration(manifest.Limits.TerminateGraceNanos)
	if err != nil {
		return Result{}, err
	}
	environment := make([]string, len(manifest.Environment))
	for index, entry := range manifest.Environment {
		environment[index] = entry.Name + "=" + entry.Value
	}
	var expectedWorldInitial []byte
	if manifest.World.Initial.Schema == "gomadv3.world.snapshot/v1" {
		expectedWorldInitial, err = artifact.ReadPayload(opened, manifest.World.Initial.File, world.MaximumSnapshotJSONBytes)
		if err != nil {
			return Result{}, fmt.Errorf("read replay initial World snapshot: %w", err)
		}
	}
	var ioConfig []byte
	var ioTranscriptLimit uint64
	var expectedIOTranscript []byte
	if manifest.IOProfile.Name != "" {
		profile, profileErr := ioprofile.Resolve(manifest.IOProfile.Name)
		if profileErr != nil {
			return Result{}, profileErr
		}
		if string(profile.Inventory) != manifest.IOProfile.Inventory || profile.InventorySHA256 != manifest.IOProfile.InventorySHA256 || profile.ImplementationSHA256 != manifest.IOProfile.ImplementationSHA256 {
			return Result{}, errors.New("recorded I/O profile identity does not match this Runner")
		}
		ioConfig, err = profile.BootstrapFrame(target.Prepared{SHA256: string(manifest.Target.SHA256), Argv: append([]string(nil), manifest.Target.Argv...)}, manifest.Runner.RunnerBuild, uint64(manifest.Seed))
		if err != nil {
			return Result{}, err
		}
		ioTranscriptLimit = uint64(manifest.Limits.IOTranscriptBytes)
		if manifest.IOProfile.Transcript == nil {
			return Result{}, errors.New("recorded I/O profile has no complete transcript")
		}
		expectedIOTranscript, err = artifact.ReadPayload(opened, manifest.IOProfile.Transcript.File, ioTranscriptLimit)
		if err != nil {
			return Result{}, fmt.Errorf("read expected I/O transcript: %w", err)
		}
	}
	observed, err := executor.Run(ctx, process.Request{
		SupervisorCommand: append([]string(nil), config.SupervisorCommand...), Command: targetPath,
		BootstrapCommand: replayBootstrapCommand(config),
		Args:             append([]string(nil), manifest.Target.Argv[1:]...), Argv0: manifest.Target.Argv[0], Dir: workDirectory, Env: environment,
		RunTimeout: runTimeout, TerminateGrace: terminateGrace, OutputLimit: uint64(manifest.Limits.OutputBytes),
		WorldRecordLimit: world.MaximumRecordingBytes, WorldTransitionLimit: uint64(manifest.Limits.WorldTransitionBytes),
		WorldSeed: uint64(manifest.Seed), ExpectedWorldInitial: expectedWorldInitial,
		IOConfig: ioConfig, IOTranscriptLimit: ioTranscriptLimit, IOReplay: manifest.IOProfile.Name != "", ExpectedIOTranscript: expectedIOTranscript,
	})
	if err != nil {
		return Result{}, fmt.Errorf("execute replay target: %w", err)
	}
	var observedWorld *worldrecord.Bundle
	if len(observed.WorldRecord) != 0 {
		recording, worldErr := world.DecodeRecording(observed.WorldRecord)
		if worldErr != nil {
			return Result{}, fmt.Errorf("decode replay World record: %w", worldErr)
		}
		bundle, worldErr := worldrecord.ComposeRecording(recording, uint64(manifest.Limits.WorldTransitionBytes))
		if worldErr != nil {
			return Result{}, fmt.Errorf("validate replay World record: %w", worldErr)
		}
		observedWorld = &bundle
	}
	result.Divergence = firstDivergence(manifest, observed, observedWorld)
	result.Match = result.Divergence == ""
	return result, nil
}

func replayBootstrapCommand(config Config) []string {
	if len(config.BootstrapCommand) != 0 {
		return append([]string(nil), config.BootstrapCommand...)
	}
	return []string{config.SupervisorCommand[0], "__target_bootstrap"}
}

func preflight(config Config) (opened artifact.Artifact, retErr error) {
	if config.ArtifactPath == "" {
		return artifact.Artifact{}, fmt.Errorf("artifact path is required")
	}
	opened, err := artifact.Open(config.ArtifactPath)
	if err != nil {
		return artifact.Artifact{}, err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, opened.Close())
		}
	}()
	manifest := opened.Manifest
	if manifest.IOProfile.Name != "" {
		profile, profileErr := ioprofile.Resolve(manifest.IOProfile.Name)
		if profileErr != nil {
			return artifact.Artifact{}, profileErr
		}
		if string(profile.Inventory) != manifest.IOProfile.Inventory || profile.InventorySHA256 != manifest.IOProfile.InventorySHA256 || profile.ImplementationSHA256 != manifest.IOProfile.ImplementationSHA256 {
			return artifact.Artifact{}, errors.New("artifact I/O profile identity does not match this Runner")
		}
	}
	if manifest.ReplayMode != record.ReplayExact && manifest.ReplayMode != record.ReplayDiagnostic {
		return artifact.Artifact{}, fmt.Errorf("artifact replay mode %q cannot be executed", manifest.ReplayMode)
	}
	if manifest.Runner.HostOS != runtime.GOOS || manifest.Runner.HostArch != runtime.GOARCH || manifest.Toolchain.TargetGOOS != runtime.GOOS || manifest.Toolchain.TargetGOARCH != runtime.GOARCH {
		return artifact.Artifact{}, fmt.Errorf("artifact platform does not match host %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	identity, err := target.ReadToolchainIdentity(config.ToolchainRoot)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if identity.GoVersion != manifest.Toolchain.GoVersion || identity.BuildKey != manifest.Toolchain.BuildKey || identity.TargetGOOS != manifest.Toolchain.TargetGOOS || identity.TargetGOARCH != manifest.Toolchain.TargetGOARCH {
		return artifact.Artifact{}, fmt.Errorf("artifact toolchain identity does not match the pinned toolchain")
	}
	worldPayloads, err := readWorldPayloads(opened)
	if err != nil {
		return artifact.Artifact{}, err
	}
	initialWorld, _, err := worldrecord.Validate(manifest.World, worldPayloads)
	if err != nil {
		return artifact.Artifact{}, fmt.Errorf("validate World record: %w", err)
	}
	if manifest.World.Initial.Schema == "gomadv3.world.snapshot/v1" && uint64(initialWorld.Config.Seed) != uint64(manifest.Seed) {
		return artifact.Artifact{}, fmt.Errorf("World seed does not match target seed")
	}
	targetFile, err := artifact.OpenPayload(opened, manifest.Target.File, uint64(manifest.Target.Size))
	if err != nil {
		return artifact.Artifact{}, err
	}
	info, err := buildinfo.Read(targetFile)
	closeErr := targetFile.Close()
	if err != nil {
		return artifact.Artifact{}, errors.Join(fmt.Errorf("read stored target build info: %w", err), closeErr)
	}
	if closeErr != nil {
		return artifact.Artifact{}, fmt.Errorf("close stored target build info: %w", closeErr)
	}
	if err := validateBuildInfo(info, manifest.Target.BuildInfo); err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := duration(manifest.Limits.RunTimeoutNanos); err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := duration(manifest.Limits.TerminateGraceNanos); err != nil {
		return artifact.Artifact{}, err
	}
	return opened, nil
}

func readWorldPayloads(opened artifact.Artifact) (record.WorldPayloads, error) {
	initial, err := artifact.ReadPayload(opened, opened.Manifest.World.Initial.File, world.MaximumSnapshotJSONBytes)
	if err != nil {
		return record.WorldPayloads{}, err
	}
	transitions, err := artifact.ReadPayload(opened, opened.Manifest.World.Transitions.File, uint64(opened.Manifest.Limits.WorldTransitionBytes))
	if err != nil {
		return record.WorldPayloads{}, err
	}
	final, err := artifact.ReadPayload(opened, opened.Manifest.World.Final.File, world.MaximumSnapshotJSONBytes)
	if err != nil {
		return record.WorldPayloads{}, err
	}
	return record.WorldPayloads{Initial: initial, Transitions: transitions, Final: final}, nil
}

func validateTargetBuildInfo(path string, expected record.BuildInfo) error {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read stored target build info: %w", err)
	}
	return validateBuildInfo(info, expected)
}

func validateBuildInfo(info *debug.BuildInfo, expected record.BuildInfo) error {
	actual := projectBuildInfo(info)
	expectedBytes, err := record.CanonicalJSON(expected)
	if err != nil {
		return fmt.Errorf("encode recorded target build info: %w", err)
	}
	actualBytes, err := record.CanonicalJSON(actual)
	if err != nil {
		return fmt.Errorf("encode stored target build info: %w", err)
	}
	if string(expectedBytes) != string(actualBytes) {
		return fmt.Errorf("stored target build info does not match manifest")
	}
	return nil
}

func firstDivergence(manifest record.Manifest, observed process.Result, observedWorld *worldrecord.Bundle) string {
	if observed.IOTranscript.ReplayDivergence != nil {
		return fmt.Sprintf("io_profile.transcript.ordinal[%d]", *observed.IOTranscript.ReplayDivergence)
	}
	expected := manifest.Outcome
	if expected.Termination == "timeout" {
		if !observed.WatchdogTimeout {
			return "outcome.termination"
		}
	} else if string(observed.Termination) != expected.Termination {
		return "outcome.termination"
	}
	if expected.ExitCode != nil && observed.ExitCode != int(*expected.ExitCode) {
		return "outcome.exit_code"
	}
	if expected.Signal != nil && observed.Signal != *expected.Signal {
		return "outcome.signal"
	}
	if actualReason(observed, observedWorld) != expected.Reason {
		return "outcome.reason"
	}
	if sha256Record(observed.Stdout.FullSHA256) != manifest.Streams.Stdout.FullSHA256 {
		return "stdout.full_sha256"
	}
	if sha256Record(observed.Stderr.FullSHA256) != manifest.Streams.Stderr.FullSHA256 {
		return "stderr.full_sha256"
	}
	if manifest.IOProfile.Transcript != nil {
		if !observed.IOTranscript.Complete {
			return "io_profile.transcript.complete"
		}
		if sha256Record(observed.IOTranscript.SHA256) != manifest.IOProfile.Transcript.SHA256 {
			return "io_profile.transcript.sha256"
		}
		if record.Uint64String(observed.IOTranscript.Records) != manifest.IOProfile.Transcript.Records {
			return "io_profile.transcript.records"
		}
	} else if observed.IOTranscript.Complete {
		return "io_profile.transcript"
	}
	if manifest.World.Initial.Schema == "gomadv3.world.snapshot/v1" {
		if observedWorld == nil {
			return "world.record"
		}
		if observedWorld.Manifest.Initial.SemanticDigest != manifest.World.Initial.SemanticDigest {
			return "world.initial.semantic_digest"
		}
		if observedWorld.Manifest.Terminal != manifest.World.Terminal {
			return "world.terminal"
		}
		if observedWorld.Manifest.Initial.RawSHA256 != manifest.World.Initial.RawSHA256 {
			return "world.initial.raw_sha256"
		}
		if observedWorld.Manifest.Transitions.TranscriptDigest != manifest.World.Transitions.TranscriptDigest {
			return "world.transitions.transcript_digest"
		}
		if observedWorld.Manifest.Transitions.Count != manifest.World.Transitions.Count {
			return "world.transitions.count"
		}
		if observedWorld.Manifest.Transitions.RawSHA256 != manifest.World.Transitions.RawSHA256 {
			return "world.transitions.raw_sha256"
		}
		if observedWorld.Manifest.Final.SemanticDigest != manifest.World.Final.SemanticDigest {
			return "world.final.semantic_digest"
		}
		if observedWorld.Manifest.Final.RawSHA256 != manifest.World.Final.RawSHA256 {
			return "world.final.raw_sha256"
		}
	} else if observedWorld != nil {
		return "world.record"
	}
	return ""
}

func actualReason(result process.Result, observedWorld *worldrecord.Bundle) string {
	if result.WatchdogTimeout {
		return "watchdog_timeout"
	}
	if observedWorld != nil {
		if reason := worldFailureReason(observedWorld.Manifest.Terminal.Kind); reason != "" {
			return reason
		}
	}
	if result.Termination == process.TerminationSignal {
		return "external_signal"
	}
	diagnostic := string(result.Stderr.Bytes)
	switch {
	case len(diagnostic) >= len("runtime: GOMADSEED does not support cgo or external linking") && diagnostic[:len("runtime: GOMADSEED does not support cgo or external linking")] == "runtime: GOMADSEED does not support cgo or external linking":
		return "unsupported_deterministic_mode"
	case len(diagnostic) >= len("fatal error: all goroutines are asleep - deadlock!") && diagnostic[:len("fatal error: all goroutines are asleep - deadlock!")] == "fatal error: all goroutines are asleep - deadlock!":
		return "deterministic_deadlock"
	case len(diagnostic) >= len("panic: test timed out after") && diagnostic[:len("panic: test timed out after")] == "panic: test timed out after":
		return "logical_test_timeout"
	case len(diagnostic) >= len("panic:") && diagnostic[:len("panic:")] == "panic:":
		return "panic_or_runtime_fatal"
	case len(diagnostic) >= len("fatal error:") && diagnostic[:len("fatal error:")] == "fatal error:":
		return "panic_or_runtime_fatal"
	default:
		return "nonzero_exit"
	}
}

func worldFailureReason(kind string) string {
	switch world.TerminalKind(kind) {
	case world.TerminalDeadlock:
		return "world_deadlock"
	case world.TerminalCapacity:
		return "world_capacity"
	case world.TerminalReplayDivergence:
		return "world_replay_divergence"
	case world.TerminalInvalidInput:
		return "world_invalid_input"
	default:
		return ""
	}
}

func duration(value record.Uint64String) (time.Duration, error) {
	if uint64(value) > math.MaxInt64 {
		return 0, fmt.Errorf("recorded duration exceeds host representation")
	}
	return time.Duration(value), nil
}

func sha256Record(value [sha256.Size]byte) record.SHA256 {
	return record.SHA256("sha256:" + hex.EncodeToString(value[:]))
}

func projectBuildInfo(info *debug.BuildInfo) record.BuildInfo {
	settings := make([]record.BuildSetting, len(info.Settings))
	for index, setting := range info.Settings {
		settings[index] = record.BuildSetting{Key: setting.Key, Value: setting.Value}
	}
	sort.Slice(settings, func(i, j int) bool { return settings[i].Key < settings[j].Key })
	mainModule := info.Main.Path
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		mainModule += "@" + info.Main.Version
	}
	return record.BuildInfo{GoVersion: info.GoVersion, Path: info.Path, MainModule: mainModule, Settings: settings}
}
