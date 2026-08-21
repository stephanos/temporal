package runner

import (
	"bytes"
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

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	"go.temporal.io/server/tools/gomadv3/runner/internal/simulationexploration"
	"go.temporal.io/server/tools/gomadv3/target"
	"go.temporal.io/server/tools/gomadv3/world"
)

type ReplayExecutor interface {
	Run(context.Context, execution.Spec) (execution.Result, error)
}

type ReplaySpec struct {
	ArtifactPath      string
	VerifyOnly        bool
	ToolchainRoot     string
	SupervisorCommand []string
	BootstrapCommand  []string
	Executor          ReplayExecutor
}

type ReplayResult struct {
	Artifact           evidence.Artifact
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

type ReplayPreflightError struct {
	Err error
}

func (err *ReplayPreflightError) Error() string {
	return "incompatible replay artifact: " + err.Err.Error()
}

func (err *ReplayPreflightError) Unwrap() error {
	return err.Err
}

type replayProcessExecutor struct{}

func (replayProcessExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	return execution.Run(ctx, request)
}

func Replay(ctx context.Context, config ReplaySpec) (result ReplayResult, retErr error) {
	opened, err := preflight(config)
	if err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: err}
	}
	defer func() {
		if closeErr := opened.Close(); closeErr != nil && retErr == nil {
			result = ReplayResult{}
			retErr = fmt.Errorf("close replay artifact: %w", closeErr)
		}
	}()
	if err := target.VerifyCompatibility(opened.Manifest.Target.Compatibility); err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: fmt.Errorf("verify replay compatibility: %w", err)}
	}
	if err := deterministicio.Default().VerifyAdapters(replayAdapters(opened.Manifest.Target.Adapters)); err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: fmt.Errorf("verify replay adapters: %w", err)}
	}
	result = ReplayResult{Artifact: opened.Detached(), Verified: true, Diagnostic: opened.Manifest.ReplayMode == evidence.ReplayDiagnostic, ChoiceReplayStatus: ChoiceReplayNone}
	choiceCapability, choiceUnavailable, err := choiceCapabilityForArtifact(opened)
	if err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: err}
	}
	if choiceUnavailable {
		result.ChoiceReplayStatus = ChoiceReplayUnavailable
		result.Divergence = "choice_profile.replay_unavailable"
		return result, nil
	}
	if choiceCapability != nil {
		result.ChoiceReplayStatus = ChoiceReplayAvailable
	}
	simulationReplay, err := simulationCapabilityForArtifact(opened)
	if err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: err}
	}
	if config.VerifyOnly {
		return result, nil
	}
	executor := config.Executor
	if executor == nil {
		if len(config.SupervisorCommand) == 0 {
			return ReplayResult{}, fmt.Errorf("supervisor command is required")
		}
		executor = replayProcessExecutor{}
	}
	workDirectory, err := os.MkdirTemp("", "gomadv3-replay-")
	if err != nil {
		return ReplayResult{}, fmt.Errorf("create replay working directory: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(workDirectory); cleanupErr != nil && retErr == nil {
			result = ReplayResult{}
			retErr = fmt.Errorf("remove replay working directory: %w", cleanupErr)
		}
	}()
	if err := os.Chmod(workDirectory, 0o700); err != nil {
		return ReplayResult{}, fmt.Errorf("make replay working directory private: %w", err)
	}
	manifest := opened.Manifest
	targetPath := filepath.Join(workDirectory, "target")
	if err := evidence.CopyPayload(opened, manifest.Target.File, targetPath, 0o500); err != nil {
		return ReplayResult{}, fmt.Errorf("copy verified replay target: %w", err)
	}
	if err := validateTargetBuildInfo(targetPath, manifest.Target.BuildInfo); err != nil {
		return ReplayResult{}, err
	}
	runTimeout, err := duration(manifest.Limits.RunTimeoutNanos)
	if err != nil {
		return ReplayResult{}, err
	}
	terminateGrace, err := duration(manifest.Limits.TerminateGraceNanos)
	if err != nil {
		return ReplayResult{}, err
	}
	environment := replayEnvironment(manifest.Environment)
	var expectedWorldInitial []byte
	var worldReplayPlan []byte
	if manifest.World.Initial.Schema == "gomadv3.world.snapshot/v1" {
		worldPayloads, readErr := readWorldPayloads(opened)
		if readErr != nil {
			return ReplayResult{}, fmt.Errorf("read replay World payloads: %w", readErr)
		}
		initial, final, validateErr := execution.Validate(manifest.World, worldPayloads)
		if validateErr != nil {
			return ReplayResult{}, fmt.Errorf("validate replay World plan: %w", validateErr)
		}
		expectedWorldInitial = worldPayloads.Initial
		worldReplayPlan, err = evidence.CanonicalJSON(world.ReplayPlan{
			SchemaVersion: world.SchemaVersion,
			InitialDigest: initial.StateDigest,
			Transitions:   final.Transitions[len(initial.Transitions):],
			FinalDigest:   final.StateDigest,
		})
		if err != nil {
			return ReplayResult{}, fmt.Errorf("encode replay World plan: %w", err)
		}
	}
	var ioConfig []byte
	var ioTranscriptLimit uint64
	var expectedIOTranscript []byte
	var readOnlyMounts []deterministicio.Mapping
	var readOnlyMountLimits deterministicio.Limits
	var readOnlyMountSnapshot *deterministicio.Snapshot
	profile := deterministicio.Default()
	ioConfig, err = profile.BootstrapFrame(target.Prepared{SHA256: string(manifest.Target.SHA256), Argv: append([]string(nil), manifest.Target.Argv...)}, manifest.Runner.RunnerBuild, uint64(manifest.Seed))
	if err != nil {
		return ReplayResult{}, err
	}
	ioTranscriptLimit = uint64(manifest.Limits.IOTranscriptBytes)
	if manifest.IOProfile.Transcript == nil {
		return ReplayResult{}, errors.New("recorded I/O profile has no complete transcript")
	}
	readOnlyMountLimits = deterministicio.DefaultLimits()
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts != nil {
		descriptor, readErr := evidence.ReadPayload(opened, mounts.File, uint64(mounts.Bytes))
		if readErr != nil {
			return ReplayResult{}, fmt.Errorf("read read-only mount descriptor: %w", readErr)
		}
		var snapshot deterministicio.Snapshot
		readOnlyMounts, readOnlyMountLimits, snapshot, readErr = deterministicio.DecodeCapturedInputs(replayCapturedInputs(*mounts), descriptor, func(name string, maximum uint64) ([]byte, error) {
			return evidence.ReadPayload(opened, name, maximum)
		})
		if readErr != nil {
			return ReplayResult{}, fmt.Errorf("decode read-only mount artifact: %w", readErr)
		}
		readOnlyMountSnapshot = &snapshot
	}
	expectedIOTranscript, err = evidence.ReadPayload(opened, manifest.IOProfile.Transcript.File, ioTranscriptLimit)
	if err != nil {
		return ReplayResult{}, fmt.Errorf("read expected I/O transcript: %w", err)
	}
	ioCapability := &execution.IOCapability{
		Config:     ioConfig,
		Transcript: &execution.IOTranscriptCapability{Limit: ioTranscriptLimit, Replay: true, Expected: expectedIOTranscript},
		ReadOnlyMount: &execution.ReadOnlyMountCapability{
			Mappings: readOnlyMounts, Limits: readOnlyMountLimits, Replay: readOnlyMountSnapshot,
		},
	}
	request := execution.Spec{
		SupervisorCommand: append([]string(nil), config.SupervisorCommand...), Command: targetPath,
		BootstrapCommand: replayBootstrapCommand(config),
		Args:             append([]string(nil), manifest.Target.Argv[1:]...), Argv0: manifest.Target.Argv[0], Dir: workDirectory, Env: environment,
		RunTimeout: runTimeout, TerminateGrace: terminateGrace, OutputLimit: uint64(manifest.Limits.OutputBytes),
		World: execution.WorldCapability{
			RecordLimit: world.MaximumRecordingBytes, TransitionLimit: uint64(manifest.Limits.WorldTransitionBytes),
			Seed: uint64(manifest.Seed), ExpectedInitial: expectedWorldInitial, ReplayPlan: worldReplayPlan,
		},
		IO: ioCapability, Choice: choiceCapability,
	}
	request.Simulation = simulationReplay.capability
	if request.Simulation == nil {
		if _, ok := executor.(replayProcessExecutor); ok {
			request.Simulation = &execution.SimulationCapability{Role: execution.SimulationRoleCoordinator}
		}
	}
	observed, err := executor.Run(ctx, request)
	if err != nil {
		if divergence, controlled := onlyChoiceReplayDivergence(err); controlled {
			result.ChoiceReplayStatus = ChoiceReplayDiverged
			result.Divergence = fmt.Sprintf("choice_profile.divergence.ordinal[%d].%s", divergence.Divergence.Ordinal, choice.DivergenceReasonName(divergence.Divergence.Reason))
			return result, nil
		}
		return ReplayResult{}, fmt.Errorf("execute replay target: %w", err)
	}
	var observedWorld *execution.Bundle
	if len(observed.WorldRecord) != 0 {
		recording, worldErr := world.DecodeRecording(observed.WorldRecord)
		if worldErr != nil {
			return ReplayResult{}, fmt.Errorf("decode replay World record: %w", worldErr)
		}
		if recording.Final.Replay.Expected != 0 && recording.Final.Replay.Cursor != recording.Final.Replay.Expected && recording.Terminal.Kind != world.TerminalReplayDivergence {
			return ReplayResult{}, errors.New("validate replay World record: final World replay is incomplete")
		}
		recording.Initial.Replay = world.ReplayProgress{}
		recording.Final.Replay = world.ReplayProgress{}
		bundle, worldErr := execution.ComposeRecording(recording, uint64(manifest.Limits.WorldTransitionBytes))
		if worldErr != nil {
			return ReplayResult{}, fmt.Errorf("validate replay World record: %w", worldErr)
		}
		observedWorld = &bundle
	}
	result.Divergence = simulationReplay.divergence(manifest, observed)
	if result.Divergence == "" {
		result.Divergence = replayDivergence(manifest, observed, observedWorld)
	}
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

type simulationArtifactReplay struct {
	capability     *execution.SimulationCapability
	expectedRecord []byte
}

func simulationCapabilityForArtifact(opened evidence.Artifact) (simulationArtifactReplay, error) {
	profile := opened.Manifest.SimulationProfile
	if profile == nil {
		return simulationArtifactReplay{}, nil
	}
	plan, err := evidence.ReadPayload(opened, profile.Plan.File, uint64(profile.Plan.Bytes))
	if err != nil {
		return simulationArtifactReplay{}, fmt.Errorf("read simulation exploration plan: %w", err)
	}
	record, err := evidence.ReadPayload(opened, profile.Record.File, uint64(profile.Record.Bytes))
	if err != nil {
		return simulationArtifactReplay{}, fmt.Errorf("read simulation exploration record: %w", err)
	}
	if err := simulationexploration.ValidateArtifact(*profile, plan, record); err != nil {
		return simulationArtifactReplay{}, fmt.Errorf("validate simulation exploration artifact: %w", err)
	}
	return simulationArtifactReplay{
		capability: &execution.SimulationCapability{
			Role: execution.SimulationRoleCoordinator, ExplorationPlan: plan,
			ExplorationRecordLimit: uint64(profile.Record.Limit), ExplorationRecordCount: 1,
		},
		expectedRecord: record,
	}, nil
}

func (replay simulationArtifactReplay) divergence(manifest evidence.ExecutionRecord, observed execution.Result) string {
	if manifest.SimulationProfile == nil {
		if len(observed.SimulationRecords) != 0 {
			return "simulation_profile"
		}
		return ""
	}
	if len(observed.SimulationRecords) != 1 {
		return "simulation_profile.record.count"
	}
	if !bytes.Equal(observed.SimulationRecords[0], replay.expectedRecord) {
		return "simulation_profile.record.sha256"
	}
	return ""
}

func onlyChoiceReplayDivergence(err error) (*execution.ChoiceReplayDivergenceError, bool) {
	var found *execution.ChoiceReplayDivergenceError
	var visit func(error) bool
	visit = func(current error) bool {
		if current == nil {
			return true
		}
		if divergence, ok := current.(*execution.ChoiceReplayDivergenceError); ok {
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

func choiceCapabilityForArtifact(opened evidence.Artifact) (*execution.ChoiceCapability, bool, error) {
	manifest := opened.Manifest
	choices := manifest.ChoiceProfile
	if choices == nil {
		return nil, false, nil
	}
	if choices.Name == choice.LegacyProfile {
		return nil, true, nil
	}
	implementation, err := choice.ImplementationIdentity(manifest.Toolchain.BuildKey)
	if err != nil || choices.Name != choice.Profile || choices.ImplementationSHA256 != evidence.SHA256FromSum(implementation) {
		return nil, false, errors.New("artifact choice profile identity does not match this Runner")
	}
	payload, err := evidence.ReadPayload(opened, choices.Trace.File, uint64(choices.Trace.Limit))
	if err != nil {
		return nil, false, fmt.Errorf("read choice trace: %w", err)
	}
	digest, err := choices.Trace.SHA256.Bytes()
	if err != nil {
		return nil, false, fmt.Errorf("decode choice trace identity: %w", err)
	}
	trace, err := choice.DecodeStoredTrace(choices.Name, payload, choice.TerminalMetadata{
		State: choice.TerminalComplete, Limit: uint64(choices.Trace.Limit), Records: uint64(choices.Trace.Records), SHA256: digest,
	})
	if err != nil {
		return nil, false, fmt.Errorf("validate choice trace: %w", err)
	}
	targetSHA256, err := manifest.Target.SHA256.Bytes()
	if err != nil {
		return nil, false, fmt.Errorf("decode choice target identity: %w", err)
	}
	executionIdentity := choice.ExecutionIdentity{
		TargetSHA256: targetSHA256, ToolchainBuildKey: manifest.Toolchain.BuildKey,
		GOOS: manifest.Toolchain.TargetGOOS, GOARCH: manifest.Toolchain.TargetGOARCH, ImplementationSHA256: implementation,
	}
	tape, err := choice.ProjectReplayPlan(trace, executionIdentity)
	if err != nil {
		return nil, false, fmt.Errorf("derive choice replay tape: %w", err)
	}
	if choices.Trace.TapeSHA256 != evidence.SHA256FromSum(tape.SHA256) || uint64(choices.Trace.Decisions) != uint64(len(tape.Decisions)) {
		return nil, false, errors.New("artifact choice tape identity does not match its trace")
	}
	return &execution.ChoiceCapability{
		Mode: choice.ModeReplay, Profile: choices.Name, ImplementationSHA256: implementation,
		ExecutionIdentity: executionIdentity, Limit: uint64(choices.Trace.Limit), ReplayPlan: &tape,
	}, false, nil
}

func replayEnvironment(recorded []evidence.Environment) []string {
	environment := make([]string, 0, len(recorded))
	for _, entry := range recorded {
		if entry.Name == "GOMADV3_CHOICE_PROFILE" {
			continue
		}
		environment = append(environment, entry.Name+"="+entry.Value)
	}
	return environment
}

func replayBootstrapCommand(config ReplaySpec) []string {
	if len(config.BootstrapCommand) != 0 {
		return append([]string(nil), config.BootstrapCommand...)
	}
	return []string{config.SupervisorCommand[0], "__target_bootstrap"}
}

func preflight(config ReplaySpec) (opened evidence.Artifact, retErr error) {
	if config.ArtifactPath == "" {
		return evidence.Artifact{}, fmt.Errorf("artifact path is required")
	}
	opened, err := evidence.OpenArtifact(config.ArtifactPath)
	if err != nil {
		return evidence.Artifact{}, err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, opened.Close())
		}
	}()
	manifest := opened.Manifest
	profile := deterministicio.Default()
	if !profile.MatchesRecorded(manifest.IOProfile.Name, string(manifest.IOProfile.ImplementationSHA256), string(manifest.IOProfile.InventorySHA256), manifest.IOProfile.Inventory) {
		return evidence.Artifact{}, errors.New("artifact I/O profile identity does not match this Runner")
	}
	if manifest.ReplayMode != evidence.ReplayExact && manifest.ReplayMode != evidence.ReplayDiagnostic {
		return evidence.Artifact{}, fmt.Errorf("artifact replay mode %q cannot be executed", manifest.ReplayMode)
	}
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts != nil {
		descriptor, readErr := evidence.ReadPayload(opened, mounts.File, uint64(mounts.Bytes))
		if readErr != nil {
			return evidence.Artifact{}, fmt.Errorf("read read-only mount descriptor: %w", readErr)
		}
		if _, _, _, readErr = deterministicio.DecodeCapturedInputs(replayCapturedInputs(*mounts), descriptor, func(name string, maximum uint64) ([]byte, error) {
			return evidence.ReadPayload(opened, name, maximum)
		}); readErr != nil {
			return evidence.Artifact{}, fmt.Errorf("validate read-only mount artifact: %w", readErr)
		}
	}
	if _, _, choiceErr := choiceCapabilityForArtifact(opened); choiceErr != nil {
		return evidence.Artifact{}, choiceErr
	}
	if manifest.Runner.HostOS != runtime.GOOS || manifest.Runner.HostArch != runtime.GOARCH || manifest.Toolchain.TargetGOOS != runtime.GOOS || manifest.Toolchain.TargetGOARCH != runtime.GOARCH {
		return evidence.Artifact{}, fmt.Errorf("artifact platform does not match host %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	identity, err := target.ReadToolchainIdentity(config.ToolchainRoot)
	if err != nil {
		return evidence.Artifact{}, err
	}
	if identity.GoVersion != manifest.Toolchain.GoVersion || identity.BuildKey != manifest.Toolchain.BuildKey || identity.TargetGOOS != manifest.Toolchain.TargetGOOS || identity.TargetGOARCH != manifest.Toolchain.TargetGOARCH {
		return evidence.Artifact{}, fmt.Errorf("artifact toolchain identity does not match the pinned toolchain")
	}
	worldPayloads, err := readWorldPayloads(opened)
	if err != nil {
		return evidence.Artifact{}, err
	}
	initialWorld, _, err := execution.Validate(manifest.World, worldPayloads)
	if err != nil {
		return evidence.Artifact{}, fmt.Errorf("validate World record: %w", err)
	}
	if manifest.World.Initial.Schema == "gomadv3.world.snapshot/v1" && uint64(initialWorld.Config.Seed) != uint64(manifest.Seed) {
		return evidence.Artifact{}, fmt.Errorf("World seed does not match target seed")
	}
	targetFile, err := evidence.OpenPayload(opened, manifest.Target.File, uint64(manifest.Target.Size))
	if err != nil {
		return evidence.Artifact{}, err
	}
	if err := verifyReplayCapabilityManifest(opened, targetFile, identity); err != nil {
		return evidence.Artifact{}, errors.Join(err, targetFile.Close())
	}
	info, err := buildinfo.Read(targetFile)
	closeErr := targetFile.Close()
	if err != nil {
		return evidence.Artifact{}, errors.Join(fmt.Errorf("read stored target build info: %w", err), closeErr)
	}
	if closeErr != nil {
		return evidence.Artifact{}, fmt.Errorf("close stored target build info: %w", closeErr)
	}
	if err := validateBuildInfo(info, manifest.Target.BuildInfo); err != nil {
		return evidence.Artifact{}, err
	}
	if _, err := duration(manifest.Limits.RunTimeoutNanos); err != nil {
		return evidence.Artifact{}, err
	}
	if _, err := duration(manifest.Limits.TerminateGraceNanos); err != nil {
		return evidence.Artifact{}, err
	}
	return opened, nil
}

func verifyReplayCapabilityManifest(opened evidence.Artifact, targetFile *os.File, identity target.ToolchainIdentity) error {
	recorded := opened.Manifest.Target.CapabilityManifest
	switch opened.Manifest.Target.CapabilityMode {
	case "closure":
		if recorded != nil {
			return errors.New("closure replay target contains a linked capability manifest")
		}
		return nil
	case "linked":
		if recorded == nil {
			return errors.New("linked replay target capability manifest is missing")
		}
		actual, err := target.ReadCapabilityManifestFile(targetFile, identity)
		if err != nil {
			return fmt.Errorf("extract stored target capability manifest: %w", err)
		}
		if *actual.Record() != *recorded {
			return errors.New("stored target embedded capability manifest does not match the run record")
		}
		payload, err := evidence.ReadPayload(opened, recorded.File, uint64(recorded.Bytes))
		if err != nil {
			return fmt.Errorf("read stored target capability manifest: %w", err)
		}
		if !bytes.Equal(payload, actual.Payload) {
			return errors.New("stored target capability manifest payload does not match its embedded record")
		}
		return nil
	default:
		return fmt.Errorf("unknown replay target capability mode %q", opened.Manifest.Target.CapabilityMode)
	}
}

func replayAdapters(adapters []evidence.TargetAdapter) []deterministicio.Adapter {
	result := make([]deterministicio.Adapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = deterministicio.Adapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}

func replayCapturedInputs(manifest evidence.ReadOnlyMounts) deterministicio.CapturedInputsManifest {
	return deterministicio.CapturedInputsManifest{
		Schema: manifest.Schema, File: manifest.File, SHA256: deterministicio.Digest(manifest.SHA256), Bytes: uint64(manifest.Bytes),
		Entries: uint64(manifest.Entries), NotExist: uint64(manifest.NotExist), TotalBytes: uint64(manifest.TotalBytes), Mappings: append([]string(nil), manifest.Mappings...),
		Limits: deterministicio.CapturedInputLimits{
			PathBytes: uint64(manifest.Limits.PathBytes), Requests: uint64(manifest.Limits.Requests), Files: uint64(manifest.Limits.Files),
			DirectoryEntries: uint64(manifest.Limits.DirectoryEntries), SingleFileBytes: uint64(manifest.Limits.SingleFileBytes), TotalBytes: uint64(manifest.Limits.TotalBytes),
		},
	}
}

func readWorldPayloads(opened evidence.Artifact) (evidence.WorldPayloads, error) {
	initial, err := evidence.ReadPayload(opened, opened.Manifest.World.Initial.File, world.MaximumSnapshotJSONBytes)
	if err != nil {
		return evidence.WorldPayloads{}, err
	}
	transitions, err := evidence.ReadPayload(opened, opened.Manifest.World.Transitions.File, uint64(opened.Manifest.Limits.WorldTransitionBytes))
	if err != nil {
		return evidence.WorldPayloads{}, err
	}
	final, err := evidence.ReadPayload(opened, opened.Manifest.World.Final.File, world.MaximumSnapshotJSONBytes)
	if err != nil {
		return evidence.WorldPayloads{}, err
	}
	return evidence.WorldPayloads{Initial: initial, Transitions: transitions, Final: final}, nil
}

func validateTargetBuildInfo(path string, expected evidence.BuildInfo) error {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read stored target build info: %w", err)
	}
	return validateBuildInfo(info, expected)
}

func validateBuildInfo(info *debug.BuildInfo, expected evidence.BuildInfo) error {
	actual := target.ProjectBuildInfo(info)
	expectedBytes, err := evidence.CanonicalJSON(expected)
	if err != nil {
		return fmt.Errorf("encode recorded target build info: %w", err)
	}
	actualBytes, err := evidence.CanonicalJSON(actual)
	if err != nil {
		return fmt.Errorf("encode stored target build info: %w", err)
	}
	if string(expectedBytes) != string(actualBytes) {
		return fmt.Errorf("stored target build info does not match manifest")
	}
	return nil
}

func replayDivergence(manifest evidence.ExecutionRecord, observed execution.Result, observedWorld *execution.Bundle) string {
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
	if evidence.SHA256FromSum(observed.Stdout.FullSHA256) != manifest.Streams.Stdout.FullSHA256 {
		return "stdout.full_sha256"
	}
	if evidence.SHA256FromSum(observed.Stderr.FullSHA256) != manifest.Streams.Stderr.FullSHA256 {
		return "stderr.full_sha256"
	}
	if manifest.IOProfile.Transcript != nil {
		if !observed.IOTranscript.Complete {
			return "io_profile.transcript.complete"
		}
		if evidence.SHA256FromSum(observed.IOTranscript.SHA256) != manifest.IOProfile.Transcript.SHA256 {
			return "io_profile.transcript.sha256"
		}
		if evidence.Uint64String(observed.IOTranscript.Records) != manifest.IOProfile.Transcript.Records {
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

func choiceTraceDivergence(manifest evidence.ExecutionRecord, observed execution.Result) string {
	if choices := manifest.ChoiceProfile; choices != nil {
		trace := observed.ChoiceTrace
		if trace.Profile != choices.Name || trace.Limit != uint64(choices.Trace.Limit) || trace.Trace.Summary.Terminal != choice.TerminalComplete {
			return "choice_profile.trace.complete"
		}
		if evidence.SHA256FromSum(trace.Trace.SHA256) != choices.Trace.SHA256 {
			return "choice_profile.trace.sha256"
		}
		if evidence.Uint64String(trace.Trace.Summary.Records) != choices.Trace.Records {
			return "choice_profile.trace.records"
		}
		if evidence.Uint64String(trace.Trace.Summary.Branching) != choices.Trace.BranchingRecords {
			return "choice_profile.trace.branching_records"
		}
	} else if observed.ChoiceTrace.Profile != "" {
		return "choice_profile.trace"
	}
	return ""
}

func actualReason(result execution.Result, observedWorld *execution.Bundle) string {
	terminal := evidence.WorldTerminal{}
	if observedWorld != nil {
		terminal = observedWorld.Manifest.Terminal
	}
	return execution.Classify(result, false, terminal).Reason
}

func duration(value evidence.Uint64String) (time.Duration, error) {
	if uint64(value) > math.MaxInt64 {
		return 0, fmt.Errorf("recorded duration exceeds host representation")
	}
	return time.Duration(value), nil
}
