package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
	simulationrecord "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulationrecord"
	"go.temporal.io/server/tools/gomad3/runner/internal/minimizer"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/world"
)

const maximumMinimizationAttempts = uint64(1_000_000)

type MinimizeSpec struct {
	ArtifactPath      string
	OutputRoot        string
	AttemptBudget     uint64
	MaximumBytes      uint64
	ToolchainRoot     string
	SupervisorCommand []string
	BootstrapCommand  []string
	Replayer          ArtifactReplayer
	// Resume continues from the state an interrupted run left under OutputRoot.
	Resume bool
}

type MinimizeResult struct {
	Artifact      artifact.Artifact              `json:"artifact"`
	Changed       bool                           `json:"changed"`
	Attempts      uint64                         `json:"attempts"`
	AttemptBudget uint64                         `json:"attempt_budget"`
	Accepted      []record.MinimizationReduction `json:"accepted"`
	StopReason    string                         `json:"stop_reason"`
}

type minimizationSession struct {
	config          MinimizeSpec
	dependencies    executionDependencies
	opened          artifact.Artifact
	workDirectory   string
	prepared        target.Prepared
	campaign        CampaignSpec
	baseEnvironment []record.Environment
	profile         deterministicio.Spec
	mappings        []readonlymount.Mapping
	mountLimits     readonlymount.Limits
	mountSnapshot   *readonlymount.Snapshot
	mountArtifact   *readonlymount.CapturedInputs
	choiceIdentity  choice.ExecutionIdentity
	exactChoiceTape *choice.ReplayPlan
	executor        executionRunner
	replayer        ArtifactReplayer
	temporaryRoot   string
	workspace       *minimizer.Workspace
}

type minimizationTrial struct {
	input    artifact.ArtifactInput
	accepted bool
	replay   ReplayResult
}

func Minimize(ctx context.Context, config MinimizeSpec) (MinimizeResult, error) {
	return minimizeWith(ctx, config, executionDependencies{})
}

func minimizeWith(ctx context.Context, config MinimizeSpec, dependencies executionDependencies) (result MinimizeResult, retErr error) {
	session, err := openMinimizationSession(ctx, config, dependencies)
	if err != nil {
		return MinimizeResult{}, err
	}
	defer func() {
		retErr = errors.Join(retErr, session.close())
		if retErr != nil {
			result = MinimizeResult{}
		}
	}()
	state, err := session.reduce(ctx)
	if err != nil {
		return MinimizeResult{}, err
	}
	checkpoint := session.workspace.Checkpoint()
	result = MinimizeResult{
		Artifact: session.opened.Detached(), Changed: checkpoint.Accepted != nil, Attempts: state.Attempts,
		AttemptBudget: state.AttemptBudget, Accepted: projectMinimizationReductions(state.Accepted), StopReason: string(state.StopReason),
	}
	if checkpoint.Accepted != nil {
		published, err := session.publishMinimized(ctx, checkpoint)
		if err != nil {
			return MinimizeResult{}, err
		}
		result.Artifact = published
	}
	if err := session.workspace.Complete(); err != nil {
		return MinimizeResult{}, err
	}
	return result, nil
}

// reduce evaluates every attempt the persisted state has left and checkpoints
// each one.
func (session *minimizationSession) reduce(ctx context.Context) (minimizer.State, error) {
	state := session.workspace.Checkpoint().State
	for {
		attempt, ok, err := minimizer.Next(state)
		if err != nil {
			return minimizer.State{}, err
		}
		if !ok {
			return state, nil
		}
		trial, err := session.evaluate(ctx, state.Config, attempt.Candidate)
		if err != nil {
			return minimizer.State{}, err
		}
		state, err = minimizer.Commit(state, attempt, trial.accepted)
		if err != nil {
			return minimizer.State{}, err
		}
		var accepted *minimizer.AcceptedArtifact
		if trial.accepted {
			accepted, err = session.retainAccepted(ctx, trial)
			if err != nil {
				return minimizer.State{}, err
			}
		}
		if err := session.workspace.Commit(state, accepted); err != nil {
			return minimizer.State{}, err
		}
	}
}

func (session *minimizationSession) retainAccepted(ctx context.Context, trial minimizationTrial) (*minimizer.AcceptedArtifact, error) {
	retained, err := artifact.PublishArtifact(artifact.Store{
		Root: session.workspace.AcceptedRoot(), Context: ctx, MaximumBytes: defaultMinimizedArtifactBytes(session.opened.StoredBytes), Key: artifact.StoreKeyRecord,
		TargetPool: artifact.TargetPool(session.config.OutputRoot),
	}, trial.input)
	if err != nil {
		return nil, fmt.Errorf("retain accepted minimization candidate: %w", err)
	}
	return &minimizer.AcceptedArtifact{
		Directory: filepath.Base(retained.Path), RecordHash: retained.Manifest.RecordHash, ChoiceReplayStatus: trial.replay.ChoiceReplayStatus,
	}, nil
}

func (session *minimizationSession) publishMinimized(ctx context.Context, checkpoint minimizer.Checkpoint) (artifact.Artifact, error) {
	published, err := session.publishedArtifact(ctx, checkpoint)
	if err != nil {
		return artifact.Artifact{}, err
	}
	finalReplay, err := session.replay(ctx, published.Path)
	if err != nil {
		return artifact.Artifact{}, fmt.Errorf("replay minimized artifact: %w", err)
	}
	if err := validateMinimizationReplay(published.Manifest, finalReplay); err != nil {
		return artifact.Artifact{}, err
	}
	if published.Manifest.Outcome.FailureSignature != session.opened.Manifest.Outcome.FailureSignature {
		return artifact.Artifact{}, errors.New("minimized artifact changed the normalized failure signature")
	}
	return published, nil
}

// publishedArtifact publishes the final artifact once: a run that died after
// publishing finds the publication in its checkpoint and reopens it.
func (session *minimizationSession) publishedArtifact(ctx context.Context, checkpoint minimizer.Checkpoint) (artifact.Artifact, error) {
	if reference := checkpoint.Published; reference != nil {
		published, err := openRetainedMinimizationArtifact(filepath.Join(session.config.OutputRoot, reference.Directory), reference.RecordHash)
		if err != nil {
			return artifact.Artifact{}, fmt.Errorf("reopen published minimized artifact: %w", err)
		}
		return published.Detached(), published.Close()
	}
	published, err := session.publishAccepted(ctx, checkpoint)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if err := session.workspace.RecordPublication(minimizer.PublishedArtifact{
		Directory: filepath.Base(published.Path), RecordHash: published.Manifest.RecordHash,
	}); err != nil {
		return artifact.Artifact{}, err
	}
	return published, nil
}

// publishAccepted publishes the final artifact into the output root. A run
// that died before recording the publication republishes the same record, and
// the record-keyed store returns the artifact that is already there.
func (session *minimizationSession) publishAccepted(ctx context.Context, checkpoint minimizer.Checkpoint) (artifact.Artifact, error) {
	input, err := session.acceptedInput(*checkpoint.Accepted)
	if err != nil {
		return artifact.Artifact{}, err
	}
	input.Manifest.Minimization = minimizationEvidence(session.opened.Manifest, checkpoint.State, checkpoint.Accepted.ChoiceReplayStatus)
	maximumBytes := session.config.MaximumBytes
	if maximumBytes == 0 {
		maximumBytes = defaultMinimizedArtifactBytes(session.opened.StoredBytes)
	}
	published, err := artifact.PublishArtifact(artifact.Store{
		Root: session.config.OutputRoot, Context: ctx, MaximumBytes: maximumBytes, Key: artifact.StoreKeyRecord,
		TargetPool: artifact.TargetPool(session.config.OutputRoot),
	}, input)
	if err != nil {
		return artifact.Artifact{}, fmt.Errorf("publish minimized artifact: %w", err)
	}
	return published, nil
}

// acceptedInput rebuilds the publication input of the last accepted candidate
// from its retained artifact, so an uninterrupted and a resumed run publish
// from the same bytes.
func (session *minimizationSession) acceptedInput(reference minimizer.AcceptedArtifact) (_ artifact.ArtifactInput, retErr error) {
	accepted, err := openRetainedMinimizationArtifact(filepath.Join(session.workspace.AcceptedRoot(), reference.Directory), reference.RecordHash)
	if err != nil {
		return artifact.ArtifactInput{}, fmt.Errorf("open accepted minimization artifact: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, accepted.Close())
	}()
	manifest := accepted.Manifest
	if manifest.Outcome.FailureSignature != session.opened.Manifest.Outcome.FailureSignature || manifest.SimulationProfile == nil {
		return artifact.ArtifactInput{}, errors.New("accepted minimization artifact does not reproduce the parent failure")
	}
	input := artifact.ArtifactInput{Manifest: manifest, TargetPath: session.prepared.Path, ReadOnlyMounts: session.mountArtifact, Simulation: &artifact.SimulationPayloads{}}
	payloads := map[string]*[]byte{
		manifest.Streams.Stdout.File:           &input.Stdout,
		manifest.Streams.Stderr.File:           &input.Stderr,
		manifest.SimulationProfile.Plan.File:   &input.Simulation.Plan,
		manifest.SimulationProfile.Record.File: &input.Simulation.Record,
	}
	if transcript := manifest.IOProfile.Transcript; transcript != nil {
		payloads[transcript.File] = &input.IOTranscript
	}
	if choices := manifest.ChoiceProfile; choices != nil {
		payloads[choices.Trace.File] = &input.ChoiceTrace
	}
	for file, destination := range payloads {
		*destination, err = readRetainedMinimizationPayload(accepted, file)
		if err != nil {
			return artifact.ArtifactInput{}, fmt.Errorf("read accepted minimization artifact: %w", err)
		}
	}
	input.World, err = readWorldPayloads(accepted)
	if err != nil {
		return artifact.ArtifactInput{}, fmt.Errorf("read accepted minimization artifact: %w", err)
	}
	return input, nil
}

func openRetainedMinimizationArtifact(path string, recordHash record.SHA256) (artifact.Artifact, error) {
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if opened.Manifest.RecordHash != recordHash {
		return artifact.Artifact{}, errors.Join(errors.New("artifact record hash does not match the minimizer state"), opened.Close())
	}
	return opened, nil
}

func readRetainedMinimizationPayload(opened artifact.Artifact, file string) ([]byte, error) {
	for _, listed := range opened.Manifest.Files {
		if listed.Path == file {
			return artifact.ReadPayload(opened, file, uint64(listed.Size))
		}
	}
	return nil, fmt.Errorf("artifact payload %q is not listed", file)
}

func openMinimizationSession(ctx context.Context, config MinimizeSpec, dependencies executionDependencies) (_ *minimizationSession, retErr error) {
	if config.OutputRoot == "" {
		return nil, errors.New("minimized artifact output root is required")
	}
	if config.AttemptBudget == 0 || config.AttemptBudget > maximumMinimizationAttempts {
		return nil, fmt.Errorf("minimization attempt budget must be between 1 and %d", maximumMinimizationAttempts)
	}
	opened, err := preflight(ReplaySpec{ArtifactPath: config.ArtifactPath, ToolchainRoot: config.ToolchainRoot})
	if err != nil {
		return nil, &ReplayPreflightError{Err: err}
	}
	session := &minimizationSession{config: config, dependencies: dependencies, opened: opened, profile: deterministicio.Default()}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, session.close())
		}
	}()
	manifest := opened.Manifest
	if manifest.ArtifactKind != record.ArtifactTargetFailure || manifest.ReplayMode != record.ReplayExact || manifest.SimulationProfile == nil || manifest.SimulationProfile.FailureSHA256 == "" {
		return nil, errors.New("minimization requires an exact simulation target-failure artifact")
	}
	choiceCapability, unavailable, err := choiceCapabilityForArtifact(opened)
	if err != nil {
		return nil, err
	}
	if unavailable {
		return nil, errors.New("minimization requires an exact retained choice tape")
	}
	if choiceCapability == nil {
		return nil, errors.New("minimization requires an exact retained choice tape")
	}
	session.choiceIdentity = choiceCapability.ExecutionIdentity
	tape := *choiceCapability.ReplayPlan
	session.exactChoiceTape = &tape
	plan, err := artifact.ReadPayload(opened, manifest.SimulationProfile.Plan.File, uint64(manifest.SimulationProfile.Plan.Bytes))
	if err != nil {
		return nil, fmt.Errorf("read minimization simulation plan: %w", err)
	}
	explorationConfig, candidate, err := simulationrecord.CandidateForArtifact(*manifest.SimulationProfile, plan, session.exactChoiceTape)
	if err != nil {
		return nil, fmt.Errorf("reconstruct minimization candidate: %w", err)
	}
	initial, err := minimizer.New(explorationConfig, candidate, config.AttemptBudget)
	if err != nil {
		return nil, err
	}
	if err := session.prepareWorkspace(); err != nil {
		return nil, err
	}
	session.workspace, err = minimizer.OpenWorkspace(ctx, config.OutputRoot, minimizer.Binding{
		ParentRecordHash: manifest.RecordHash, ImplementationSHA256: minimizer.ImplementationSHA256(), ToolchainBuildKey: manifest.Toolchain.BuildKey,
	}, initial, config.Resume)
	if err != nil {
		return nil, err
	}
	if accepted := session.workspace.Checkpoint().Accepted; accepted != nil {
		if _, err := session.acceptedInput(*accepted); err != nil {
			return nil, err
		}
	}
	return session, nil
}

func (session *minimizationSession) prepareWorkspace() error {
	workDirectory, err := os.MkdirTemp("", "gomad3-minimize-")
	if err != nil {
		return fmt.Errorf("create minimization working directory: %w", err)
	}
	session.workDirectory = workDirectory
	if err := os.Chmod(workDirectory, 0o700); err != nil {
		return fmt.Errorf("make minimization working directory private: %w", err)
	}
	session.temporaryRoot = filepath.Join(workDirectory, "candidates")
	manifest := session.opened.Manifest
	targetPath := filepath.Join(workDirectory, "target")
	if err := artifact.CopyPayload(session.opened, manifest.Target.File, targetPath, 0o500); err != nil {
		return fmt.Errorf("copy verified minimization target: %w", err)
	}
	if err := validateTargetBuildInfo(targetPath, manifest.Target.BuildInfo); err != nil {
		return err
	}
	session.prepared = preparedTargetFromArtifact(targetPath, manifest)
	if session.prepared.CapabilityMode != target.CapabilityModeClosure {
		capabilities, err := target.ReadCapabilityManifest(targetPath, target.ToolchainIdentity{
			GoVersion: manifest.Toolchain.GoVersion, BuildKey: manifest.Toolchain.BuildKey,
			TargetGOOS: manifest.Toolchain.TargetGOOS, TargetGOARCH: manifest.Toolchain.TargetGOARCH,
		})
		if err != nil {
			return fmt.Errorf("read minimized target capability manifest: %w", err)
		}
		session.prepared.CapabilityManifest = capabilities
	}
	runTimeout, err := duration(manifest.Limits.ExecutionTimeoutNanos)
	if err != nil {
		return err
	}
	overallTimeout, err := duration(manifest.Limits.OverallTimeoutNanos)
	if err != nil {
		return err
	}
	terminateGrace, err := duration(manifest.Limits.TerminateGraceNanos)
	if err != nil {
		return err
	}
	session.baseEnvironment = minimizationBaseEnvironment(manifest.Environment)
	session.mountLimits = readonlymount.DefaultLimits()
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts != nil {
		descriptor, readErr := artifact.ReadPayload(session.opened, mounts.File, uint64(mounts.Bytes))
		if readErr != nil {
			return fmt.Errorf("read minimized target mounts: %w", readErr)
		}
		mappings, limits, snapshot, readErr := readonlymount.DecodeCapturedInputs(replayCapturedInputs(*mounts), descriptor, func(name string, maximum uint64) ([]byte, error) {
			return artifact.ReadPayload(session.opened, name, maximum)
		})
		if readErr != nil {
			return fmt.Errorf("decode minimized target mounts: %w", readErr)
		}
		captured, encodeErr := readonlymount.EncodeCapturedInputs(mappings, limits, snapshot)
		if encodeErr != nil {
			return fmt.Errorf("encode minimized target mounts: %w", encodeErr)
		}
		session.mappings, session.mountLimits, session.mountSnapshot, session.mountArtifact = mappings, limits, &snapshot, &captured
	}
	session.campaign = CampaignSpec{
		ExecutionTimeout: runTimeout, OverallTimeout: overallTimeout, TerminateGrace: terminateGrace,
		OutputLimit: uint64(manifest.Limits.OutputBytes), WorldTransitionLimit: uint64(manifest.Limits.WorldTransitionBytes),
		ChoiceTraceLimit: uint64(manifest.Limits.ChoiceTraceBytes), RunnerBuild: manifest.Runner.RunnerBuild,
		IOROMountLimits: session.mountLimits, SupervisorCommand: append([]string(nil), session.config.SupervisorCommand...),
	}
	session.executor = session.dependencies.executor
	if session.executor == nil {
		if len(session.config.SupervisorCommand) == 0 {
			return errors.New("supervisor command is required")
		}
		session.executor = processExecutor{}
	}
	session.replayer = session.config.Replayer
	return nil
}

func (session *minimizationSession) evaluate(ctx context.Context, explorationConfig simulationengine.Config, candidate simulationengine.Candidate) (minimizationTrial, error) {
	executionForCandidate, err := simulationrecord.ExecutionForCandidate(explorationConfig, candidate, session.choiceIdentity)
	if err != nil {
		return minimizationTrial{}, err
	}
	manifest := session.opened.Manifest
	ioConfig, err := session.profile.BootstrapFrame(session.prepared, manifest.Runner.RunnerBuild, uint64(manifest.Seed))
	if err != nil {
		return minimizationTrial{}, err
	}
	choiceCapability := &execution.ChoiceCapability{
		Mode: executionForCandidate.ChoiceMode, Profile: choice.Profile,
		ImplementationSHA256: session.choiceIdentity.ImplementationSHA256, ExecutionIdentity: session.choiceIdentity,
		Limit: uint64(manifest.Limits.ChoiceTraceBytes), ReplayPlan: executionForCandidate.ChoiceReplayPlan,
	}
	startedAt := time.Now().UTC()
	request := execution.Spec{
		SupervisorCommand: append([]string(nil), session.config.SupervisorCommand...), Command: session.prepared.Path,
		BootstrapCommand: append([]string(nil), session.config.BootstrapCommand...),
		Args:             append([]string(nil), session.prepared.Argv[1:]...), Argv0: session.prepared.Argv[0], Dir: session.workDirectory,
		Env:              environmentStrings(environmentForSeed(session.baseEnvironment, uint64(manifest.Seed))),
		ExecutionTimeout: session.campaign.ExecutionTimeout, TerminateGrace: session.campaign.TerminateGrace, OutputLimit: session.campaign.OutputLimit,
		World: execution.WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: session.campaign.WorldTransitionLimit, Seed: uint64(manifest.Seed)},
		IO: &execution.IOCapability{
			Config: ioConfig, Transcript: &execution.IOTranscriptCapability{Limit: uint64(manifest.Limits.IOTranscriptBytes)},
			ReadOnlyMount: &execution.ReadOnlyMountCapability{Mappings: session.mappings, Limits: session.mountLimits, Replay: session.mountSnapshot},
		},
		Choice: choiceCapability,
		Simulation: &execution.SimulationCapability{
			Role: execution.SimulationRoleCoordinator, ExplorationPlan: executionForCandidate.SimulationPlan,
			ExplorationRecordLimit: uint64(manifest.SimulationProfile.Record.Limit), ExplorationRecordCount: 1,
		},
	}
	if len(request.BootstrapCommand) == 0 && len(session.config.SupervisorCommand) != 0 {
		request.BootstrapCommand = []string{session.config.SupervisorCommand[0], "__target_bootstrap"}
	}
	observed, err := session.executor.Run(ctx, request)
	if err != nil {
		return minimizationTrial{}, fmt.Errorf("execute minimization candidate: %w", err)
	}
	if err := validateObservedChoiceTrace(session.campaign.ChoiceTraceLimit, choiceCapability, &observed.ChoiceTrace); err != nil {
		return minimizationTrial{}, err
	}
	completion := runCompletion{
		job: runJob{seed: uint64(manifest.Seed)}, startedAt: startedAt, finishedAt: time.Now().UTC(), result: observed,
	}
	worldBundle, err := recordedWorldForMinimization(observed.WorldRecord, uint64(manifest.Seed), session.campaign.WorldTransitionLimit)
	if err != nil {
		return minimizationTrial{}, err
	}
	outcome := execution.Classify(observed, false, worldBundle.Manifest.Terminal)
	if outcome.ArtifactKind != record.ArtifactTargetFailure {
		return minimizationTrial{accepted: false}, nil
	}
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace.Trace, session.choiceIdentity)
	if err != nil {
		return minimizationTrial{}, fmt.Errorf("derive minimized choice tape: %w", err)
	}
	completion.result.ChoiceTrace.TapeSHA256 = tape.SHA256
	completion.result.ChoiceTrace.Decisions = uint64(len(tape.Decisions))
	runtimeDecisions, err := simulationrecord.RuntimeDecisions(tape)
	if err != nil {
		return minimizationTrial{}, err
	}
	if len(observed.SimulationRecords) != 1 {
		return minimizationTrial{}, fmt.Errorf("minimization simulation records = %d, want 1", len(observed.SimulationRecords))
	}
	simulationProfile, err := simulationrecord.ProjectArtifact(
		explorationConfig, candidate, executionForCandidate.SimulationPlan, observed.SimulationRecords[0], runtimeDecisions,
		uint64(manifest.SimulationProfile.Record.Limit),
	)
	if err != nil {
		return minimizationTrial{}, err
	}
	retained, err := manifestForRun(campaignRequestFromSpec(session.campaign), session.prepared, session.baseEnvironment, completion, outcome, manifest.CampaignID, worldBundle.Manifest, session.mountArtifact)
	if err != nil {
		return minimizationTrial{}, err
	}
	retained.SimulationProfile = &simulationProfile
	input := executionArtifactInput(retained, session.prepared, observed, session.mountArtifact, worldBundle)
	input.Simulation = &artifact.SimulationPayloads{Plan: executionForCandidate.SimulationPlan, Record: observed.SimulationRecords[0]}
	published, err := artifact.PublishArtifact(artifact.Store{
		Root: session.temporaryRoot, Context: ctx, MaximumBytes: defaultMinimizedArtifactBytes(session.opened.StoredBytes), Key: artifact.StoreKeyRecord,
		TargetPool: artifact.TargetPool(session.workDirectory),
	}, input)
	if err != nil {
		return minimizationTrial{}, fmt.Errorf("publish minimization candidate: %w", err)
	}
	trial := minimizationTrial{input: input}
	if published.Manifest.Outcome.FailureSignature != manifest.Outcome.FailureSignature || !sameReplayOutcome(published.Manifest.Outcome, manifest.Outcome) {
		return trial, nil
	}
	replay, err := session.replay(ctx, published.Path)
	if err != nil {
		return minimizationTrial{}, fmt.Errorf("replay minimization candidate: %w", err)
	}
	trial.replay = replay
	trial.accepted = validateMinimizationReplay(published.Manifest, replay) == nil
	return trial, nil
}

func (session *minimizationSession) replay(ctx context.Context, artifactPath string) (ReplayResult, error) {
	config := ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: session.config.ToolchainRoot,
		SupervisorCommand: append([]string(nil), session.config.SupervisorCommand...),
		BootstrapCommand:  append([]string(nil), session.config.BootstrapCommand...),
	}
	if session.replayer != nil {
		return session.replayer.Replay(ctx, config)
	}
	return replayWith(ctx, config, session.dependencies)
}

func validateMinimizationReplay(manifest record.ExecutionRecord, replay ReplayResult) error {
	if !replay.Match || replay.Divergence != "" {
		return fmt.Errorf("minimization candidate exact replay diverged at %s", replay.Divergence)
	}
	if manifest.ChoiceProfile != nil && replay.ChoiceReplayStatus != ChoiceReplayExact {
		return errors.New("minimization candidate choice replay was not exact")
	}
	return nil
}

func recordedWorldForMinimization(encoded []byte, seed, limit uint64) (execution.Bundle, error) {
	if len(encoded) == 0 {
		return noneWorldBundle(), nil
	}
	recording, err := world.DecodeRecording(encoded)
	if err != nil {
		return execution.Bundle{}, fmt.Errorf("decode minimization World record: %w", err)
	}
	bundle, err := execution.ComposeRecording(recording, limit)
	if err != nil {
		return execution.Bundle{}, err
	}
	initial, _, err := execution.Validate(bundle.Manifest, bundle.Payloads)
	if err != nil {
		return execution.Bundle{}, err
	}
	if bundle.Manifest.Initial.Schema != "gomad3.world.snapshot/v1" || uint64(initial.Config.Seed) != seed {
		return execution.Bundle{}, errors.New("minimization World record seed or schema changed")
	}
	return bundle, nil
}

func preparedTargetFromArtifact(path string, manifest record.ExecutionRecord) target.Prepared {
	return target.Prepared{
		Path: path, Kind: target.Kind(manifest.Target.Kind), Source: manifest.Target.Source,
		SHA256: string(manifest.Target.SHA256), Size: uint64(manifest.Target.Size), Argv: append([]string(nil), manifest.Target.Argv...),
		BuildTags: append([]string(nil), manifest.Target.BuildTags...), Adapters: cloneAdapters(manifest.Target.Adapters),
		Compatibility: cloneCompatibility(manifest.Target.Compatibility), BuildInfo: manifest.Target.BuildInfo,
		GoVersion: manifest.Toolchain.GoVersion, BuildKey: manifest.Toolchain.BuildKey,
		TargetGOOS: manifest.Toolchain.TargetGOOS, TargetGOARCH: manifest.Toolchain.TargetGOARCH,
		CapabilityMode: target.CapabilityMode(manifest.Target.CapabilityMode),
	}
}

func minimizationBaseEnvironment(recorded []record.Environment) []record.Environment {
	base := make([]record.Environment, 0, len(recorded))
	for _, entry := range recorded {
		if entry.Name != "GOMADSEED" && entry.Name != "TZ" {
			base = append(base, entry)
		}
	}
	sort.Slice(base, func(left, right int) bool { return base[left].Name < base[right].Name })
	return base
}

func sameReplayOutcome(left, right record.Outcome) bool {
	return left.Domain == right.Domain && left.Reason == right.Reason && left.Termination == right.Termination
}

func minimizationEvidence(parent record.ExecutionRecord, state minimizer.State, acceptedChoiceReplayStatus string) *record.Minimization {
	choiceReplay := "not_present"
	if parent.ChoiceProfile != nil {
		choiceReplay = acceptedChoiceReplayStatus
	}
	return &record.Minimization{
		Schema: "gomad3.minimization/v1", ImplementationSHA256: minimizer.ImplementationSHA256(),
		ParentRecordHash: parent.RecordHash, ParentFailureSignature: parent.Outcome.FailureSignature,
		OriginalCandidateSHA256: state.Original.SHA256, FinalCandidateSHA256: state.Current.SHA256,
		AttemptBudget: record.Uint64String(state.AttemptBudget), Attempts: record.Uint64String(state.Attempts),
		OriginalForcedDecisions: record.Uint64String(len(state.Original.Overrides)), FinalForcedDecisions: record.Uint64String(len(state.Current.Overrides)),
		Accepted: projectMinimizationReductions(state.Accepted),
		Predicate: record.MinimizationPredicate{
			FailureSignature: parent.Outcome.FailureSignature, Domain: parent.Outcome.Domain, Reason: parent.Outcome.Reason,
			Termination: parent.Outcome.Termination, ReplayMatch: true,
			ChoiceReplay: choiceReplay, SimulationReplay: "exact",
		},
	}
}

func projectMinimizationReductions(reductions []minimizer.Reduction) []record.MinimizationReduction {
	result := make([]record.MinimizationReduction, len(reductions))
	for index, reduction := range reductions {
		removed := make([]record.MinimizationDecision, len(reduction.Removed))
		for decisionIndex, decision := range reduction.Removed {
			removed[decisionIndex] = record.MinimizationDecision{
				Dimension: string(decision.Dimension), Ordinal: record.Uint64String(decision.Ordinal), Identity: decision.Identity,
			}
		}
		result[index] = record.MinimizationReduction{
			Kind: string(reduction.Kind), BeforeSHA256: reduction.BeforeSHA256, AfterSHA256: reduction.AfterSHA256, Removed: removed,
		}
	}
	return result
}

// defaultMinimizedArtifactBytes bounds one minimized artifact by its parent.
// Both sides are the stored bytes of a single artifact, which count the target
// whether or not a store shares it (artifact.RetainedBytes), so the bound is
// the same for a parent that owns its target and a result linked to a pool.
func defaultMinimizedArtifactBytes(parent uint64) uint64 {
	const metadataAllowance = uint64(1 << 20)
	if parent > ^uint64(0)-metadataAllowance {
		return ^uint64(0)
	}
	return parent + metadataAllowance
}

func (session *minimizationSession) close() error {
	var err error
	if session.opened.Path != "" {
		err = session.opened.Close()
		session.opened = artifact.Artifact{}
	}
	if session.workDirectory != "" {
		err = errors.Join(err, os.RemoveAll(session.workDirectory))
		session.workDirectory = ""
	}
	return errors.Join(err, session.workspace.Close())
}
