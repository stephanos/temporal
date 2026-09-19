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
	Executor          Executor
	Replayer          ArtifactReplayer
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
	executor        Executor
	replayer        ArtifactReplayer
	temporaryRoot   string
}

type minimizationTrial struct {
	input    artifact.ArtifactInput
	accepted bool
	replay   ReplayResult
}

func Minimize(ctx context.Context, config MinimizeSpec) (result MinimizeResult, retErr error) {
	session, state, err := openMinimizationSession(ctx, config)
	if err != nil {
		return MinimizeResult{}, err
	}
	defer func() {
		retErr = errors.Join(retErr, session.close())
		if retErr != nil {
			result = MinimizeResult{}
		}
	}()
	var accepted *minimizationTrial
	for {
		attempt, ok, err := minimizer.Next(state)
		if err != nil {
			return MinimizeResult{}, err
		}
		if !ok {
			break
		}
		trial, err := session.evaluate(ctx, state.Config, attempt.Candidate)
		if err != nil {
			return MinimizeResult{}, err
		}
		state, err = minimizer.Commit(state, attempt, trial.accepted)
		if err != nil {
			return MinimizeResult{}, err
		}
		if trial.accepted {
			accepted = &trial
		}
	}
	result = MinimizeResult{
		Artifact: session.opened.Detached(), Changed: accepted != nil, Attempts: state.Attempts,
		AttemptBudget: state.AttemptBudget, Accepted: projectMinimizationReductions(state.Accepted), StopReason: string(state.StopReason),
	}
	if accepted == nil {
		return result, nil
	}
	accepted.input.Manifest.Minimization = minimizationEvidence(session.opened.Manifest, state, accepted.replay)
	maximumBytes := config.MaximumBytes
	if maximumBytes == 0 {
		maximumBytes = defaultMinimizedArtifactBytes(session.opened.StoredBytes)
	}
	published, err := artifact.PublishArtifact(artifact.Store{
		Root: config.OutputRoot, Context: ctx, MaximumBytes: maximumBytes, Key: artifact.StoreKeyRecord,
	}, accepted.input)
	if err != nil {
		return MinimizeResult{}, fmt.Errorf("publish minimized artifact: %w", err)
	}
	finalReplay, err := session.replay(ctx, published.Path)
	if err != nil {
		return MinimizeResult{}, fmt.Errorf("replay minimized artifact: %w", err)
	}
	if err := validateMinimizationReplay(published.Manifest, finalReplay); err != nil {
		return MinimizeResult{}, err
	}
	if published.Manifest.Outcome.FailureSignature != session.opened.Manifest.Outcome.FailureSignature {
		return MinimizeResult{}, errors.New("minimized artifact changed the normalized failure signature")
	}
	result.Artifact = published
	return result, nil
}

func openMinimizationSession(ctx context.Context, config MinimizeSpec) (_ *minimizationSession, state minimizer.State, retErr error) {
	if config.OutputRoot == "" {
		return nil, minimizer.State{}, errors.New("minimized artifact output root is required")
	}
	if config.AttemptBudget == 0 || config.AttemptBudget > maximumMinimizationAttempts {
		return nil, minimizer.State{}, fmt.Errorf("minimization attempt budget must be between 1 and %d", maximumMinimizationAttempts)
	}
	opened, err := preflight(ReplaySpec{ArtifactPath: config.ArtifactPath, ToolchainRoot: config.ToolchainRoot})
	if err != nil {
		return nil, minimizer.State{}, &ReplayPreflightError{Err: err}
	}
	session := &minimizationSession{config: config, opened: opened, profile: deterministicio.Default()}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, session.close())
		}
	}()
	manifest := opened.Manifest
	if manifest.ArtifactKind != record.ArtifactTargetFailure || manifest.ReplayMode != record.ReplayExact || manifest.SimulationProfile == nil || manifest.SimulationProfile.FailureSHA256 == "" {
		return nil, minimizer.State{}, errors.New("minimization requires an exact simulation target-failure artifact")
	}
	choiceCapability, unavailable, err := choiceCapabilityForArtifact(opened)
	if err != nil {
		return nil, minimizer.State{}, err
	}
	if unavailable {
		return nil, minimizer.State{}, errors.New("minimization requires an exact retained choice tape")
	}
	if choiceCapability == nil {
		return nil, minimizer.State{}, errors.New("minimization requires an exact retained choice tape")
	}
	session.choiceIdentity = choiceCapability.ExecutionIdentity
	tape := *choiceCapability.ReplayPlan
	session.exactChoiceTape = &tape
	plan, err := artifact.ReadPayload(opened, manifest.SimulationProfile.Plan.File, uint64(manifest.SimulationProfile.Plan.Bytes))
	if err != nil {
		return nil, minimizer.State{}, fmt.Errorf("read minimization simulation plan: %w", err)
	}
	explorationConfig, candidate, err := simulationrecord.CandidateForArtifact(*manifest.SimulationProfile, plan, session.exactChoiceTape)
	if err != nil {
		return nil, minimizer.State{}, fmt.Errorf("reconstruct minimization candidate: %w", err)
	}
	state, err = minimizer.New(explorationConfig, candidate, config.AttemptBudget)
	if err != nil {
		return nil, minimizer.State{}, err
	}
	if err := session.prepareWorkspace(); err != nil {
		return nil, minimizer.State{}, err
	}
	return session, state, nil
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
	session.executor = session.config.Executor
	if session.executor == nil {
		if len(session.config.SupervisorCommand) == 0 {
			return errors.New("supervisor command is required")
		}
		session.executor = processExecutor{}
	}
	session.replayer = session.config.Replayer
	if session.replayer == nil {
		session.replayer = artifactReplayer{}
	}
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
	retained, err := manifestForRun(session.campaign, session.prepared, session.baseEnvironment, completion, outcome, manifest.CampaignID, worldBundle.Manifest, session.mountArtifact)
	if err != nil {
		return minimizationTrial{}, err
	}
	retained.SimulationProfile = &simulationProfile
	input := artifact.ArtifactInput{
		Manifest: retained, TargetPath: session.prepared.Path, Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes,
		IOTranscript: observed.IOTranscript.Bytes, ChoiceTrace: observed.ChoiceTrace.Trace.Bytes,
		ReadOnlyMounts: session.mountArtifact, World: worldBundle.Payloads,
		Simulation: &artifact.SimulationPayloads{Plan: executionForCandidate.SimulationPlan, Record: observed.SimulationRecords[0]},
	}
	published, err := artifact.PublishArtifact(artifact.Store{
		Root: session.temporaryRoot, Context: ctx, MaximumBytes: defaultMinimizedArtifactBytes(session.opened.StoredBytes), Key: artifact.StoreKeyRecord,
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
	return session.replayer.Replay(ctx, ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: session.config.ToolchainRoot,
		SupervisorCommand: append([]string(nil), session.config.SupervisorCommand...),
		BootstrapCommand:  append([]string(nil), session.config.BootstrapCommand...), Executor: replayExecutor(session.config.Executor),
	})
}

func replayExecutor(executor Executor) ReplayExecutor {
	if executor == nil {
		return nil
	}
	return executor
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

func minimizationEvidence(parent record.ExecutionRecord, state minimizer.State, replay ReplayResult) *record.Minimization {
	choiceReplay := "not_present"
	if parent.ChoiceProfile != nil {
		choiceReplay = replay.ChoiceReplayStatus
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
			Termination: parent.Outcome.Termination, ReplayMatch: replay.Match,
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
	return err
}
