package replay

import (
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	executionoutcome "go.temporal.io/server/tools/gomadv3/internal/outcome"
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
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
	Artifact           artifact.Artifact
	Verified           bool
	Match              bool
	Diagnostic         bool
	Divergence         string
	ChoiceReplayStatus string
}

const (
	ChoiceReplayNone        = "none"
	ChoiceReplayAvailable   = "available"
	ChoiceReplayExact       = "exact"
	ChoiceReplayDiverged    = "diverged"
	ChoiceReplayUnavailable = "unavailable"
)

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
	if err := target.VerifyCompatibility(opened.Manifest.Target.Compatibility); err != nil {
		return Result{}, &PreflightError{Err: fmt.Errorf("verify replay compatibility: %w", err)}
	}
	if err := ioprofile.Default().VerifyAdapters(opened.Manifest.Target.Adapters); err != nil {
		return Result{}, &PreflightError{Err: fmt.Errorf("verify replay adapters: %w", err)}
	}
	result = Result{Artifact: opened.Detached(), Verified: true, Diagnostic: opened.Manifest.ReplayMode == record.ReplayDiagnostic, ChoiceReplayStatus: ChoiceReplayNone}
	choiceCapability, choiceUnavailable, err := choiceCapabilityForArtifact(opened)
	if err != nil {
		return Result{}, &PreflightError{Err: err}
	}
	if choiceUnavailable {
		result.ChoiceReplayStatus = ChoiceReplayUnavailable
		result.Divergence = "choice_profile.replay_unavailable"
		return result, nil
	}
	if choiceCapability != nil {
		result.ChoiceReplayStatus = ChoiceReplayAvailable
	}
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
	environment := replayEnvironment(manifest.Environment)
	var expectedWorldInitial []byte
	var worldReplayPlan []byte
	if manifest.World.Initial.Schema == "gomadv3.world.snapshot/v1" {
		worldPayloads, readErr := readWorldPayloads(opened)
		if readErr != nil {
			return Result{}, fmt.Errorf("read replay World payloads: %w", readErr)
		}
		initial, final, validateErr := worldrecord.Validate(manifest.World, worldPayloads)
		if validateErr != nil {
			return Result{}, fmt.Errorf("validate replay World plan: %w", validateErr)
		}
		expectedWorldInitial = worldPayloads.Initial
		worldReplayPlan, err = record.CanonicalJSON(world.ReplayPlan{
			SchemaVersion: world.SchemaVersion,
			InitialDigest: initial.StateDigest,
			Transitions:   final.Transitions[len(initial.Transitions):],
			FinalDigest:   final.StateDigest,
		})
		if err != nil {
			return Result{}, fmt.Errorf("encode replay World plan: %w", err)
		}
	}
	var ioConfig []byte
	var ioTranscriptLimit uint64
	var expectedIOTranscript []byte
	var readOnlyMounts []romount.Mapping
	var readOnlyMountLimits romount.Limits
	var readOnlyMountSnapshot *romount.Snapshot
	profile := ioprofile.Default()
	ioConfig, err = profile.BootstrapFrame(target.Prepared{SHA256: string(manifest.Target.SHA256), Argv: append([]string(nil), manifest.Target.Argv...)}, manifest.Runner.RunnerBuild, uint64(manifest.Seed))
	if err != nil {
		return Result{}, err
	}
	ioTranscriptLimit = uint64(manifest.Limits.IOTranscriptBytes)
	if manifest.IOProfile.Transcript == nil {
		return Result{}, errors.New("recorded I/O profile has no complete transcript")
	}
	readOnlyMountLimits = romount.DefaultLimits()
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts != nil {
		descriptor, readErr := artifact.ReadPayload(opened, mounts.File, uint64(mounts.Bytes))
		if readErr != nil {
			return Result{}, fmt.Errorf("read read-only mount descriptor: %w", readErr)
		}
		var snapshot romount.Snapshot
		readOnlyMounts, readOnlyMountLimits, snapshot, readErr = romount.DecodeArtifact(*mounts, descriptor, func(name string, maximum uint64) ([]byte, error) {
			return artifact.ReadPayload(opened, name, maximum)
		})
		if readErr != nil {
			return Result{}, fmt.Errorf("decode read-only mount artifact: %w", readErr)
		}
		readOnlyMountSnapshot = &snapshot
	}
	expectedIOTranscript, err = artifact.ReadPayload(opened, manifest.IOProfile.Transcript.File, ioTranscriptLimit)
	if err != nil {
		return Result{}, fmt.Errorf("read expected I/O transcript: %w", err)
	}
	ioCapability := &process.IOCapability{
		Config:     ioConfig,
		Transcript: &process.IOTranscriptCapability{Limit: ioTranscriptLimit, Replay: true, Expected: expectedIOTranscript},
		ReadOnlyMount: &process.ReadOnlyMountCapability{
			Mappings: readOnlyMounts, Limits: readOnlyMountLimits, Replay: readOnlyMountSnapshot,
		},
	}
	observed, err := executor.Run(ctx, process.Request{
		SupervisorCommand: append([]string(nil), config.SupervisorCommand...), Command: targetPath,
		BootstrapCommand: replayBootstrapCommand(config),
		Args:             append([]string(nil), manifest.Target.Argv[1:]...), Argv0: manifest.Target.Argv[0], Dir: workDirectory, Env: environment,
		RunTimeout: runTimeout, TerminateGrace: terminateGrace, OutputLimit: uint64(manifest.Limits.OutputBytes),
		World: process.WorldCapability{
			RecordLimit: world.MaximumRecordingBytes, TransitionLimit: uint64(manifest.Limits.WorldTransitionBytes),
			Seed: uint64(manifest.Seed), ExpectedInitial: expectedWorldInitial, ReplayPlan: worldReplayPlan,
		},
		IO: ioCapability, Choice: choiceCapability,
	})
	if err != nil {
		if divergence, controlled := onlyChoiceReplayDivergence(err); controlled {
			result.ChoiceReplayStatus = ChoiceReplayDiverged
			result.Divergence = fmt.Sprintf("choice_profile.divergence.ordinal[%d].%s", divergence.Divergence.Ordinal, choicewire.DivergenceReasonName(divergence.Divergence.Reason))
			return result, nil
		}
		return Result{}, fmt.Errorf("execute replay target: %w", err)
	}
	var observedWorld *worldrecord.Bundle
	if len(observed.WorldRecord) != 0 {
		recording, worldErr := world.DecodeRecording(observed.WorldRecord)
		if worldErr != nil {
			return Result{}, fmt.Errorf("decode replay World record: %w", worldErr)
		}
		if recording.Final.Replay.Expected != 0 && recording.Final.Replay.Cursor != recording.Final.Replay.Expected && recording.Terminal.Kind != world.TerminalReplayDivergence {
			return Result{}, errors.New("validate replay World record: final World replay is incomplete")
		}
		recording.Initial.Replay = world.ReplayProgress{}
		recording.Final.Replay = world.ReplayProgress{}
		bundle, worldErr := worldrecord.ComposeRecording(recording, uint64(manifest.Limits.WorldTransitionBytes))
		if worldErr != nil {
			return Result{}, fmt.Errorf("validate replay World record: %w", worldErr)
		}
		observedWorld = &bundle
	}
	result.Divergence = firstDivergence(manifest, observed, observedWorld)
	result.Match = result.Divergence == ""
	if choiceCapability != nil {
		if choiceTraceDivergence(manifest, observed) == "" {
			result.ChoiceReplayStatus = ChoiceReplayExact
		} else {
			result.ChoiceReplayStatus = ChoiceReplayDiverged
		}
	}
	return result, nil
}

func onlyChoiceReplayDivergence(err error) (*process.ChoiceReplayDivergenceError, bool) {
	var found *process.ChoiceReplayDivergenceError
	var visit func(error) bool
	visit = func(current error) bool {
		if current == nil {
			return true
		}
		if divergence, ok := current.(*process.ChoiceReplayDivergenceError); ok {
			if found != nil && found != divergence {
				return false
			}
			found = divergence
			return true
		}
		if joined, ok := current.(interface{ Unwrap() []error }); ok {
			for _, child := range joined.Unwrap() {
				if !visit(child) {
					return false
				}
			}
			return true
		}
		if wrapped := errors.Unwrap(current); wrapped != nil {
			return visit(wrapped)
		}
		return false
	}
	return found, visit(err) && found != nil
}

func choiceCapabilityForArtifact(opened artifact.Artifact) (*process.ChoiceCapability, bool, error) {
	manifest := opened.Manifest
	choices := manifest.ChoiceProfile
	if choices == nil {
		return nil, false, nil
	}
	if choices.Name == choicewire.LegacyProfile {
		return nil, true, nil
	}
	implementation, err := choicewire.ImplementationIdentity(manifest.Toolchain.BuildKey)
	if err != nil || choices.Name != choicewire.Profile || choices.ImplementationSHA256 != record.SHA256FromSum(implementation) {
		return nil, false, errors.New("artifact choice profile identity does not match this Runner")
	}
	payload, err := artifact.ReadPayload(opened, choices.Trace.File, uint64(choices.Trace.Limit))
	if err != nil {
		return nil, false, fmt.Errorf("read choice trace: %w", err)
	}
	digest, err := choices.Trace.SHA256.Bytes()
	if err != nil {
		return nil, false, fmt.Errorf("decode choice trace identity: %w", err)
	}
	trace, err := choicewire.DecodeStoredTrace(choices.Name, payload, choicewire.TerminalMetadata{
		State: choicewire.TerminalComplete, Limit: uint64(choices.Trace.Limit), Records: uint64(choices.Trace.Records), SHA256: digest,
	})
	if err != nil {
		return nil, false, fmt.Errorf("validate choice trace: %w", err)
	}
	targetSHA256, err := manifest.Target.SHA256.Bytes()
	if err != nil {
		return nil, false, fmt.Errorf("decode choice target identity: %w", err)
	}
	executionIdentity := choicewire.ExecutionIdentity{
		TargetSHA256: targetSHA256, ToolchainBuildKey: manifest.Toolchain.BuildKey,
		GOOS: manifest.Toolchain.TargetGOOS, GOARCH: manifest.Toolchain.TargetGOARCH, ImplementationSHA256: implementation,
	}
	tape, err := choicewire.ProjectDecisionTape(trace, executionIdentity)
	if err != nil {
		return nil, false, fmt.Errorf("derive choice replay tape: %w", err)
	}
	if choices.Trace.TapeSHA256 != record.SHA256FromSum(tape.SHA256) || uint64(choices.Trace.Decisions) != uint64(len(tape.Decisions)) {
		return nil, false, errors.New("artifact choice tape identity does not match its trace")
	}
	return &process.ChoiceCapability{
		Mode: choicewire.ModeReplay, Profile: choices.Name, ImplementationSHA256: implementation,
		ExecutionIdentity: executionIdentity, Limit: uint64(choices.Trace.Limit), Tape: &tape,
	}, false, nil
}

func replayEnvironment(recorded []record.Environment) []string {
	environment := make([]string, 0, len(recorded))
	for _, entry := range recorded {
		if entry.Name == "GOMADV3_CHOICE_PROFILE" {
			continue
		}
		environment = append(environment, entry.Name+"="+entry.Value)
	}
	return environment
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
	profile := ioprofile.Default()
	if !profile.MatchesRecord(manifest.IOProfile) {
		return artifact.Artifact{}, errors.New("artifact I/O profile identity does not match this Runner")
	}
	if manifest.ReplayMode != record.ReplayExact && manifest.ReplayMode != record.ReplayDiagnostic {
		return artifact.Artifact{}, fmt.Errorf("artifact replay mode %q cannot be executed", manifest.ReplayMode)
	}
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts != nil {
		descriptor, readErr := artifact.ReadPayload(opened, mounts.File, uint64(mounts.Bytes))
		if readErr != nil {
			return artifact.Artifact{}, fmt.Errorf("read read-only mount descriptor: %w", readErr)
		}
		if _, _, _, readErr = romount.DecodeArtifact(*mounts, descriptor, func(name string, maximum uint64) ([]byte, error) {
			return artifact.ReadPayload(opened, name, maximum)
		}); readErr != nil {
			return artifact.Artifact{}, fmt.Errorf("validate read-only mount artifact: %w", readErr)
		}
	}
	if _, _, choiceErr := choiceCapabilityForArtifact(opened); choiceErr != nil {
		return artifact.Artifact{}, choiceErr
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
	actual := target.ProjectBuildInfo(info)
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
	if record.SHA256FromSum(observed.Stdout.FullSHA256) != manifest.Streams.Stdout.FullSHA256 {
		return "stdout.full_sha256"
	}
	if record.SHA256FromSum(observed.Stderr.FullSHA256) != manifest.Streams.Stderr.FullSHA256 {
		return "stderr.full_sha256"
	}
	if manifest.IOProfile.Transcript != nil {
		if !observed.IOTranscript.Complete {
			return "io_profile.transcript.complete"
		}
		if record.SHA256FromSum(observed.IOTranscript.SHA256) != manifest.IOProfile.Transcript.SHA256 {
			return "io_profile.transcript.sha256"
		}
		if record.Uint64String(observed.IOTranscript.Records) != manifest.IOProfile.Transcript.Records {
			return "io_profile.transcript.records"
		}
	} else if observed.IOTranscript.Complete {
		return "io_profile.transcript"
	}
	if divergence := choiceTraceDivergence(manifest, observed); divergence != "" {
		return divergence
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

func choiceTraceDivergence(manifest record.Manifest, observed process.Result) string {
	if choices := manifest.ChoiceProfile; choices != nil {
		trace := observed.ChoiceTrace
		if trace.Profile != choices.Name || trace.Limit != uint64(choices.Trace.Limit) || trace.Trace.Summary.Terminal != choicewire.TerminalComplete {
			return "choice_profile.trace.complete"
		}
		if record.SHA256FromSum(trace.Trace.SHA256) != choices.Trace.SHA256 {
			return "choice_profile.trace.sha256"
		}
		if record.Uint64String(trace.Trace.Summary.Records) != choices.Trace.Records {
			return "choice_profile.trace.records"
		}
		if record.Uint64String(trace.Trace.Summary.Branching) != choices.Trace.BranchingRecords {
			return "choice_profile.trace.branching_records"
		}
	} else if observed.ChoiceTrace.Profile != "" {
		return "choice_profile.trace"
	}
	return ""
}

func actualReason(result process.Result, observedWorld *worldrecord.Bundle) string {
	terminal := record.WorldTerminal{}
	if observedWorld != nil {
		terminal = observedWorld.Manifest.Terminal
	}
	return executionoutcome.Classify(result, false, terminal).Reason
}

func duration(value record.Uint64String) (time.Duration, error) {
	if uint64(value) > math.MaxInt64 {
		return 0, fmt.Errorf("recorded duration exceeds host representation")
	}
	return time.Duration(value), nil
}
