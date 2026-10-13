package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	execution "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/captured"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/exploration"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/world"
)

type ReplaySpec struct {
	Backend       execution.Provider `json:"-"`
	ArtifactPath  string
	VerifyOnly    bool
	ToolchainRoot string
	// ObservedDir, when set, receives the replayed target's stdout and stderr
	// so a divergence can be diffed against the artifact's retained streams.
	ObservedDir string
}

type ReplayResult struct {
	Artifact           artifact.Artifact
	Verified           bool
	Match              bool
	Diagnostic         bool
	Divergence         string
	ChoiceReplayStatus string
	ObservedRepetition bool
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

func Replay(ctx context.Context, config ReplaySpec) (ReplayResult, error) {
	return replayWith(ctx, config, executionDependencies{})
}

func replayWith(ctx context.Context, config ReplaySpec, dependencies executionDependencies) (result ReplayResult, retErr error) {
	opened, prepared, err := preflight(ctx, config)
	if err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: err}
	}
	defer func() {
		if closeErr := opened.Close(); closeErr != nil && retErr == nil {
			result = ReplayResult{}
			retErr = fmt.Errorf("close replay artifact: %w", closeErr)
		}
	}()
	manifest := opened.Manifest()
	if manifest.Target.Backend != nil && manifest.World.Initial.Schema != "gomad3.world.snapshot/none" {
		return ReplayResult{}, &ReplayPreflightError{Err: errors.New("external observed profile does not support a World replay plan")}
	}
	result = ReplayResult{Artifact: opened.Snapshot(), Verified: true, Diagnostic: manifest.ReplayMode == record.ReplayDiagnostic, ObservedRepetition: manifest.ReplayMode == record.ReplayObserved, ChoiceReplayStatus: ChoiceReplayNone}
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
	workDirectory, err := os.MkdirTemp("", "gomad3-replay-")
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
	targetPath := filepath.Join(workDirectory, "target")
	if err := opened.CopyPayload(manifest.Target.File, targetPath, 0o500); err != nil {
		return ReplayResult{}, fmt.Errorf("copy verified replay target: %w", err)
	}
	prepared.Path = targetPath
	runTimeout, err := duration(manifest.Limits.ExecutionTimeoutNanos)
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
	if manifest.World.Initial.Schema == "gomad3.world.snapshot/v1" {
		worldPayloads, readErr := readWorldPayloads(opened)
		if readErr != nil {
			return ReplayResult{}, fmt.Errorf("read replay World payloads: %w", readErr)
		}
		initial, final, validateErr := execution.Validate(manifest.World, worldPayloads)
		if validateErr != nil {
			return ReplayResult{}, fmt.Errorf("validate replay World plan: %w", validateErr)
		}
		expectedWorldInitial = worldPayloads.Initial
		worldReplayPlan, err = world.EncodeReplayPlan(world.ReplayPlan{
			SchemaVersion: world.SchemaVersion,
			InitialDigest: initial.StateDigest,
			Transitions:   final.Transitions[len(initial.Transitions):],
			FinalDigest:   final.StateDigest,
		})
		if err != nil {
			return ReplayResult{}, fmt.Errorf("encode replay World plan: %w", err)
		}
	}
	var ioTranscriptLimit uint64
	var expectedIOTranscript []byte
	var readOnlyMounts []captured.Mapping
	var readOnlyMountLimits captured.Limits
	readOnlyMountSnapshot := &captured.Snapshot{}
	ioTranscriptLimit = uint64(manifest.Limits.IOTranscriptBytes)
	if config.Backend.Capabilities().IOTranscript && manifest.IOProfile.Transcript == nil {
		if manifest.ReplayMode != record.ReplayDiagnostic {
			return ReplayResult{}, errors.New("recorded I/O profile has no complete transcript")
		}
		if choiceCapability != nil {
			return ReplayResult{}, errors.New("diagnostic replay without an I/O transcript cannot replay recorded choices")
		}
	}
	readOnlyMountLimits = captured.DefaultLimits()
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts != nil {
		descriptor, readErr := opened.ReadPayload(mounts.File, uint64(mounts.Bytes))
		if readErr != nil {
			return ReplayResult{}, fmt.Errorf("read read-only mount descriptor: %w", readErr)
		}
		var snapshot captured.Snapshot
		readOnlyMounts, readOnlyMountLimits, snapshot, readErr = captured.DecodeCapturedInputs(replayCapturedInputs(*mounts), descriptor, func(name string, maximum uint64) ([]byte, error) {
			return opened.ReadPayload(name, maximum)
		})
		if readErr != nil {
			return ReplayResult{}, fmt.Errorf("decode read-only mount artifact: %w", readErr)
		}
		readOnlyMountSnapshot = &snapshot
	}
	if transcript := manifest.IOProfile.Transcript; transcript != nil {
		expectedIOTranscript, err = opened.ReadPayload(transcript.File, ioTranscriptLimit)
		if err != nil {
			return ReplayResult{}, fmt.Errorf("read expected I/O transcript: %w", err)
		}
	}
	ioCapability := &execution.IOCapability{
		Profile:    manifest.IOProfile,
		Transcript: &execution.IOTranscriptCapability{Limit: ioTranscriptLimit, Replay: manifest.IOProfile.Transcript != nil, Expected: expectedIOTranscript},
		ReadOnlyMount: &execution.ReadOnlyMountCapability{
			Mappings: readOnlyMounts, Limits: readOnlyMountLimits, Replay: readOnlyMountSnapshot,
		},
	}
	request := execution.ExecutionRequest{
		Target: prepared, Seed: uint64(manifest.Seed), RunnerBuild: manifest.Runner.RunnerBuild, Command: targetPath,
		Args: append([]string(nil), manifest.Target.Argv[1:]...), Argv0: manifest.Target.Argv[0], Dir: workDirectory, Env: environment,
		ExecutionTimeout: runTimeout, TerminateGrace: terminateGrace, OutputLimit: uint64(manifest.Limits.OutputBytes),
		World: execution.WorldCapability{
			RecordLimit: world.MaximumRecordingBytes, TransitionLimit: uint64(manifest.Limits.WorldTransitionBytes),
			Seed: uint64(manifest.Seed), ExpectedInitial: expectedWorldInitial, ReplayPlan: worldReplayPlan,
		},
		IO: ioCapability, Choice: choiceCapability,
	}
	request.Simulation = simulationReplay.capability
	if request.Simulation == nil && dependencies.executor == nil && config.Backend.Capabilities().Simulation {
		request.Simulation = &execution.SimulationCapability{Role: execution.SimulationRoleCoordinator}
	}
	var observed execution.ExecutionResult
	if dependencies.executor != nil {
		observed, err = dependencies.executor.Run(ctx, request)
	} else {
		observed, err = config.Backend.Execute(ctx, request)
	}
	if config.ObservedDir != "" {
		if dumpErr := dumpObservedStreams(config.ObservedDir, observed); dumpErr != nil {
			return ReplayResult{}, dumpErr
		}
		if manifest.Target.Backend != nil {
			if dumpErr := os.WriteFile(filepath.Join(config.ObservedDir, "backend-evidence.bin"), observed.BackendEvidence, 0o600); dumpErr != nil {
				return ReplayResult{}, dumpErr
			}
		}
	}
	if err != nil {
		if divergence, controlled := onlyChoiceReplayDivergence(err); controlled {
			result.ChoiceReplayStatus = ChoiceReplayDiverged
			result.Divergence = fmt.Sprintf("choice_profile.divergence.ordinal[%d].%s", divergence.Divergence.Ordinal, choice.DivergenceReasonName(divergence.Divergence.Reason))
			return result, nil
		}
		if choiceCapability != nil && backendChoiceReplayDivergence(err) {
			result.ChoiceReplayStatus = ChoiceReplayDiverged
			result.Divergence = "choice_profile.divergence"
			var failure *execution.FailureError
			if errors.As(err, &failure) && failure.ChoiceDivergence != nil {
				result.Divergence = fmt.Sprintf("choice_profile.divergence.ordinal[%d].%s", failure.ChoiceDivergence.Ordinal, choice.DivergenceReasonName(failure.ChoiceDivergence.Reason))
			}
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
	if result.Divergence == "" && manifest.Target.Backend != nil && (manifest.Target.Backend.Evidence == nil || record.HashBytes(observed.BackendEvidence) != manifest.Target.Backend.Evidence.SHA256) {
		result.Divergence = "backend.evidence.sha256"
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

func simulationCapabilityForArtifact(opened *artifact.Opened) (simulationArtifactReplay, error) {
	profile := opened.Manifest().SimulationProfile
	if profile == nil {
		return simulationArtifactReplay{}, nil
	}
	plan, err := opened.ReadPayload(profile.Plan.File, uint64(profile.Plan.Bytes))
	if err != nil {
		return simulationArtifactReplay{}, fmt.Errorf("read simulation exploration plan: %w", err)
	}
	record, err := opened.ReadPayload(profile.Record.File, uint64(profile.Record.Bytes))
	if err != nil {
		return simulationArtifactReplay{}, fmt.Errorf("read simulation exploration record: %w", err)
	}
	if err := exploration.ValidateArtifact(*profile, plan, record); err != nil {
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

func (replay simulationArtifactReplay) divergence(manifest record.ExecutionRecord, observed execution.ExecutionResult) string {
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

func choiceCapabilityForArtifact(opened *artifact.Opened) (*execution.ChoiceCapability, bool, error) {
	manifest := opened.Manifest()
	choices := manifest.ChoiceProfile
	if choices == nil {
		return nil, false, nil
	}
	if choices.Name == choice.LegacyProfile {
		return nil, true, nil
	}
	implementation, err := choice.ImplementationIdentity(manifest.Toolchain.BuildKey)
	if err != nil || choices.Name != choice.Profile || choices.ImplementationSHA256 != record.SHA256FromSum(implementation) {
		return nil, false, errors.New("artifact choice profile identity does not match this Runner")
	}
	payload, err := opened.ReadPayload(choices.Trace.File, uint64(choices.Trace.Limit))
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
	if choices.Trace.TapeSHA256 != record.SHA256FromSum(tape.SHA256) || uint64(choices.Trace.Decisions) != uint64(len(tape.Decisions)) {
		return nil, false, errors.New("artifact choice tape identity does not match its trace")
	}
	return &execution.ChoiceCapability{
		Mode: choice.ModeReplay, Profile: choices.Name, ImplementationSHA256: implementation,
		ExecutionIdentity: executionIdentity, Limit: uint64(choices.Trace.Limit), ReplayPlan: &tape,
	}, false, nil
}

func replayEnvironment(recorded []record.Environment) []string {
	environment := make([]string, 0, len(recorded))
	for _, entry := range recorded {
		if entry.Name == "GOMAD3_CHOICE_PROFILE" || entry.Name == choice.DiagnosticProfileEnvironment {
			continue
		}
		environment = append(environment, entry.Name+"="+entry.Value)
	}
	return environment
}

func preflight(ctx context.Context, config ReplaySpec) (_ *artifact.Opened, _ target.Prepared, retErr error) {
	if config.ArtifactPath == "" {
		return nil, target.Prepared{}, fmt.Errorf("artifact path is required")
	}
	if config.Backend == nil {
		return nil, target.Prepared{}, errors.New("replay requires a backend provider")
	}
	opened, err := artifact.OpenArtifact(config.ArtifactPath)
	if err != nil {
		return nil, target.Prepared{}, err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, opened.Close())
		}
	}()
	prepared, err := config.Backend.ValidateReplay(ctx, opened)
	if err != nil {
		return nil, target.Prepared{}, err
	}
	manifest := opened.Manifest()
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts != nil {
		descriptor, readErr := opened.ReadPayload(mounts.File, uint64(mounts.Bytes))
		if readErr != nil {
			return nil, target.Prepared{}, fmt.Errorf("read read-only mount descriptor: %w", readErr)
		}
		if _, _, _, readErr = captured.DecodeCapturedInputs(replayCapturedInputs(*mounts), descriptor, func(name string, maximum uint64) ([]byte, error) {
			return opened.ReadPayload(name, maximum)
		}); readErr != nil {
			return nil, target.Prepared{}, fmt.Errorf("validate read-only mount artifact: %w", readErr)
		}
	}
	if _, _, choiceErr := choiceCapabilityForArtifact(opened); choiceErr != nil {
		return nil, target.Prepared{}, choiceErr
	}
	worldPayloads, err := readWorldPayloads(opened)
	if err != nil {
		return nil, target.Prepared{}, err
	}
	initialWorld, _, err := execution.Validate(manifest.World, worldPayloads)
	if err != nil {
		return nil, target.Prepared{}, fmt.Errorf("validate World record: %w", err)
	}
	if manifest.World.Initial.Schema == "gomad3.world.snapshot/v1" && uint64(initialWorld.Config.Seed) != uint64(manifest.Seed) {
		//nolint:staticcheck // World is the model's proper name; preserve the preflight diagnostic.
		return nil, target.Prepared{}, fmt.Errorf("World seed does not match target seed")
	}
	if _, err := duration(manifest.Limits.ExecutionTimeoutNanos); err != nil {
		return nil, target.Prepared{}, err
	}
	if _, err := duration(manifest.Limits.TerminateGraceNanos); err != nil {
		return nil, target.Prepared{}, err
	}
	return opened, prepared, nil
}

func replayCapturedInputs(manifest record.ReadOnlyMounts) captured.CapturedInputsManifest {
	return captured.CapturedInputsManifest{
		Schema: manifest.Schema, File: manifest.File, SHA256: manifest.SHA256, Bytes: uint64(manifest.Bytes),
		Entries: uint64(manifest.Entries), NotExist: uint64(manifest.NotExist), TotalBytes: uint64(manifest.TotalBytes), Mappings: append([]string(nil), manifest.Mappings...),
		Limits: captured.CapturedInputLimits{
			PathBytes: uint64(manifest.Limits.PathBytes), Requests: uint64(manifest.Limits.Requests), Files: uint64(manifest.Limits.Files),
			DirectoryEntries: uint64(manifest.Limits.DirectoryEntries), SingleFileBytes: uint64(manifest.Limits.SingleFileBytes), TotalBytes: uint64(manifest.Limits.TotalBytes),
		},
	}
}

func readWorldPayloads(opened *artifact.Opened) (record.WorldPayloads, error) {
	manifest := opened.Manifest()
	initial, err := opened.ReadPayload(manifest.World.Initial.File, world.MaximumSnapshotJSONBytes)
	if err != nil {
		return record.WorldPayloads{}, err
	}
	transitions, err := opened.ReadPayload(manifest.World.Transitions.File, uint64(manifest.Limits.WorldTransitionBytes))
	if err != nil {
		return record.WorldPayloads{}, err
	}
	final, err := opened.ReadPayload(manifest.World.Final.File, world.MaximumSnapshotJSONBytes)
	if err != nil {
		return record.WorldPayloads{}, err
	}
	return record.WorldPayloads{Initial: initial, Transitions: transitions, Final: final}, nil
}

func replayDivergence(manifest record.ExecutionRecord, observed execution.ExecutionResult, observedWorld *execution.Bundle) string {
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
	if manifest.World.Initial.Schema == "gomad3.world.snapshot/v1" {
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

func choiceTraceDivergence(manifest record.ExecutionRecord, observed execution.ExecutionResult) string {
	if choices := manifest.ChoiceProfile; choices != nil {
		trace := observed.ChoiceTrace
		if trace.Profile != choices.Name || trace.Limit != uint64(choices.Trace.Limit) || trace.Trace.Summary.Terminal != choice.TerminalComplete {
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

func actualReason(result execution.ExecutionResult, observedWorld *execution.Bundle) string {
	terminal := record.WorldTerminal{}
	if observedWorld != nil {
		terminal = observedWorld.Manifest.Terminal
	}
	return execution.Classify(result, false, terminal).Reason
}

func duration(value record.Uint64String) (time.Duration, error) {
	if uint64(value) > math.MaxInt64 {
		return 0, fmt.Errorf("recorded duration exceeds host representation")
	}
	return time.Duration(value), nil
}

func dumpObservedStreams(directory string, observed execution.ExecutionResult) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create replay observation directory: %w", err)
	}
	for name, data := range map[string][]byte{"stdout": observed.Stdout.Bytes, "stderr": observed.Stderr.Bytes} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o644); err != nil {
			return fmt.Errorf("write replayed %s: %w", name, err)
		}
	}
	return nil
}
