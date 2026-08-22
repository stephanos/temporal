package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomadv3/artifact"
	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	simulationengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulation"
	simulationrecord "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulationrecord"
	"go.temporal.io/server/tools/gomadv3/target"
	"go.temporal.io/server/tools/gomadv3/world"
)

const combinedExplorationRecordLimit = uint64(128 << 20)

type simulationExplorationRoundResult struct {
	result simulationengine.Result
	run    campaign.ExecutionRecord
}

func runSimulationExplorationLocal(
	ctx context.Context,
	config CampaignSpec,
	selection SeedSelection,
	baseEnvironment []record.Environment,
	readOnlyMounts []readonlymount.Mapping,
	prepared target.Prepared,
	profile deterministicio.Spec,
	journal *campaign.CampaignJournal,
	runID string,
	resuming bool,
	resumedRuns []campaign.ExecutionRecord,
	summary *CampaignResult,
	reportProgress func(CampaignPhase, int) error,
) error {
	baseSeed, ok := selection.SeedAt(0)
	if !ok || selection.Count() != 1 {
		return errors.New("simulation-exploration base seed is unavailable")
	}
	choiceImplementation, err := choice.ImplementationIdentity(prepared.BuildKey)
	if err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	choiceIdentity, err := choiceExecutionIdentity(prepared, choiceImplementation)
	if err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	executionSHA256, err := simulationExplorationExecutionIdentity(config, prepared, profile, choiceImplementation)
	if err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	failureBudget := config.FailureBudget
	if config.OnFailure == PolicyAll {
		failureBudget = config.MaxExecutions
	}
	explorationConfig := simulationengine.Config{
		ExecutionSHA256: executionSHA256, ControllerSHA256: simulationengine.ImplementationSHA256(), BaseSeed: baseSeed,
		Parallel: config.Parallel, MaxExecutions: config.MaxExecutions, MaxForcedDecisions: config.MaxForcedDecisions,
		MaxExplorationBytes: config.MaxExplorationBytes, MaxResultBytes: config.MaxExplorationResultBytes,
		FailureBudget: failureBudget, Limits: simulationengine.DimensionLimits(config.SimulationDimensionLimits),
	}
	state, err := simulationengine.New(explorationConfig)
	if err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	segmentBytes, err := simulationExplorationSegmentCapacity(config)
	if err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	var explorationJournal *campaign.SimulationExplorationJournal
	if resuming {
		var recovery uint64
		explorationJournal, state, recovery, err = campaign.ResumeSimulationExplorationJournal(ctx, journal.Path(), explorationConfig, segmentBytes)
		if err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
		summary.RecoveryExecutions += recovery
		committedRuns := explorationJournal.CommittedExecutions()
		if err := reconcileExplorationExecutions(journal, resumedRuns, committedRuns); err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
		restored, err := restoreResumeSummary(journal.Path(), selection, committedRuns)
		if err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
		recoveryExecutions := summary.RecoveryExecutions
		*summary = restored.summary
		summary.RecoveryExecutions = recoveryExecutions
	} else {
		explorationJournal, err = campaign.NewSimulationExplorationJournal(ctx, journal.Path(), state, segmentBytes)
		if err != nil {
			return &HostError{Reason: "simulation_exploration_setup", Err: err}
		}
	}
	combinedSummary := state.Summary()
	projectedSummary := projectSimulationExplorationSummary(combinedSummary)
	summary.SimulationExploration = &projectedSummary

	distinct := make(map[record.SHA256]string)
	semanticProbes := make(map[string]struct{})
	choiceFeatures := make(map[string]struct{})
	if len(explorationJournal.CommittedExecutions()) != 0 {
		restored, err := restoreResumeSummary(journal.Path(), selection, explorationJournal.CommittedExecutions())
		if err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
		distinct = restored.distinct
		semanticProbes = restored.probes
		choiceFeatures = restored.choiceFeatures
	}
	executor := config.Executor
	if executor == nil {
		executor = processExecutor{}
	}

	for {
		round, available := state.NextRound()
		if !available {
			break
		}
		staged, err := explorationJournal.StageRound(round)
		if err != nil {
			return &HostError{Reason: "simulation_exploration_stage", Err: err}
		}
		completions, err := executeSimulationExplorationRound(ctx, config, executor, prepared, baseEnvironment, profile, readOnlyMounts, staged, state, round, choiceIdentity, reportProgress)
		if err != nil {
			return err
		}
		roundSummary := cloneSummary(*summary)
		roundDistinct := cloneFailurePaths(distinct)
		roundProbes := cloneStringSet(semanticProbes)
		roundChoiceFeatures := cloneStringSet(choiceFeatures)
		results := make([]simulationengine.Result, len(round.Candidates))
		runs := make([]campaign.ExecutionRecord, len(round.Candidates))
		for index, completion := range completions {
			processed, err := processSimulationExplorationCompletion(
				ctx, config, prepared, baseEnvironment, readOnlyMounts, runID, journal.Path(), staged,
				state, round, index, completion, choiceIdentity, &roundSummary, roundDistinct, roundProbes, roundChoiceFeatures,
			)
			if err != nil {
				return err
			}
			results[index] = processed.result
			runs[index] = processed.run
			if err := staged.RecordExecution(index, processed.run); err != nil {
				return &HostError{Reason: "simulation_exploration_stage", Err: err}
			}
		}
		next, segment, err := simulationengine.CommitRound(state, round, results)
		if err != nil {
			return &HostError{Reason: "simulation_exploration_commit", Err: err}
		}
		if err := explorationJournal.CommitRound(staged, segment); err != nil {
			return &HostError{Reason: "simulation_exploration_commit", Err: err}
		}
		for _, run := range runs {
			if err := journal.AppendExecution(run); err != nil {
				return &HostError{Reason: "runs_append", Err: err}
			}
		}
		state = next
		distinct = roundDistinct
		semanticProbes = roundProbes
		choiceFeatures = roundChoiceFeatures
		*summary = roundSummary
		combinedSummary := state.Summary()
		projectedSummary := projectSimulationExplorationSummary(combinedSummary)
		summary.SimulationExploration = &projectedSummary
		summary.StopReason = StopReason(combinedSummary.StopReason)
		if err := reportProgress(ProgressRunning, 0); err != nil {
			return &HostError{Reason: "progress_output", Err: err}
		}
	}

	if coverageHasSemantic(config.Coverage) {
		coverage, err := deterministicio.SummarizeSemanticProbes(sortedProbeList(semanticProbes))
		if err != nil {
			return &HostError{Reason: "semantic_coverage", Err: err}
		}
		summary.SemanticCoverage = &coverage
		missing, err := deterministicio.MissingRequiredSemanticProbes(coverage, config.RequiredSemanticProbes)
		if err != nil {
			return err
		}
		if len(missing) != 0 {
			return &deterministicio.MissingSemanticProbesError{Probes: missing}
		}
	}
	if err := prepared.Verify(); err != nil {
		return &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	if err := ctx.Err(); err != nil {
		return &HostError{Reason: contextFailureReason(err), Err: err}
	}
	failureSignatures := make([]record.SHA256, 0, len(distinct))
	for signature := range distinct {
		failureSignatures = append(failureSignatures, signature)
	}
	combinedSummary = state.Summary()
	projectedSummary = projectSimulationExplorationSummary(combinedSummary)
	summary.SimulationExploration = &projectedSummary
	summary.StopReason = StopReason(combinedSummary.StopReason)
	if err := journal.Publish(campaign.CampaignSummary{
		Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures, Watchdogs: summary.Watchdogs,
		Cancelled: summary.Cancelled, DistinctFailures: summary.DistinctFailures, RetainedSuccesses: summary.RetainedSuccesses,
		RetainedSuccessBytes: summary.RetainedSuccessBytes, StopReason: string(summary.StopReason), FailureSignatures: failureSignatures,
		SimulationExploration: &combinedSummary, SimulationExplorationImplementationSHA256: simulationengine.ImplementationSHA256(),
		SimulationExplorationChainSHA256: explorationJournal.ChainSHA256(), RecoveryExecutions: summary.RecoveryExecutions,
	}); err != nil {
		return &HostError{Reason: "campaign_publish", Err: err}
	}
	if err := reportProgress(ProgressComplete, 0); err != nil {
		return &HostError{Reason: "progress_output", Err: err}
	}
	return nil
}

func executeSimulationExplorationRound(
	ctx context.Context,
	config CampaignSpec,
	executor Executor,
	prepared target.Prepared,
	baseEnvironment []record.Environment,
	profile deterministicio.Spec,
	readOnlyMounts []readonlymount.Mapping,
	staged *campaign.SimulationExplorationRoundJournal,
	state simulationengine.State,
	round simulationengine.Round,
	choiceIdentity choice.ExecutionIdentity,
	reportProgress func(CampaignPhase, int) error,
) ([]runCompletion, error) {
	roundCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	completionChannel := make(chan runCompletion, len(round.Candidates))
	startOrdinal := state.LogicalExecutions
	for index, candidate := range round.Candidates {
		candidateExecution, err := simulationrecord.ExecutionForCandidate(state.Config, candidate, choiceIdentity)
		if err != nil {
			return nil, &HostError{Reason: "simulation_exploration_control", Err: err}
		}
		job := runJob{
			ordinal: startOrdinal + uint64(index), seed: state.Config.BaseSeed,
			choiceMode: candidateExecution.ChoiceMode, choiceReplayPlan: candidateExecution.ChoiceReplayPlan,
			simulationPlan: string(candidateExecution.SimulationPlan), simulationRecordLimit: combinedExplorationRecordLimit, simulationRecordCount: 1,
		}
		readiness := newRunReadiness()
		go runSeed(roundCtx, config, executor, prepared, baseEnvironment, profile, readOnlyMounts, staged, job, readiness, completionChannel)
		readiness.wait()
	}
	progressErr := reportProgress(ProgressRunning, len(round.Candidates))
	if progressErr != nil {
		cancel()
	}
	completions := make([]runCompletion, len(round.Candidates))
	seen := make([]bool, len(round.Candidates))
	received := 0
	var contextErr error
	for received < len(round.Candidates) {
		select {
		case completion := <-completionChannel:
			if completion.job.ordinal < startOrdinal || completion.job.ordinal >= startOrdinal+uint64(len(completions)) {
				cancel()
				return nil, &HostError{Reason: "simulation_exploration_order", Err: errors.New("simulation exploration completion ordinal is outside its round")}
			}
			index := int(completion.job.ordinal - startOrdinal)
			if seen[index] {
				cancel()
				return nil, &HostError{Reason: "simulation_exploration_order", Err: errors.New("simulation exploration completion ordinal is duplicated")}
			}
			completions[index] = completion
			seen[index] = true
			received++
		case <-ctx.Done():
			contextErr = ctx.Err()
			cancel()
			ctx = context.WithoutCancel(ctx)
		}
	}
	if contextErr != nil {
		return nil, &HostError{Reason: contextFailureReason(contextErr), Err: contextErr}
	}
	if progressErr != nil {
		return nil, &HostError{Reason: "progress_output", Err: progressErr}
	}
	for _, completion := range completions {
		if completion.err != nil {
			return nil, &HostError{Reason: supervisionFailureReason(completion.err), Err: completion.err}
		}
		if completion.result.Cancelled {
			return nil, &HostError{Reason: "runner_cancelled", Err: errors.New("simulation exploration candidate was cancelled")}
		}
	}
	return completions, nil
}

func processSimulationExplorationCompletion(
	ctx context.Context,
	config CampaignSpec,
	prepared target.Prepared,
	baseEnvironment []record.Environment,
	readOnlyMounts []readonlymount.Mapping,
	runID string,
	batchPath string,
	staged *campaign.SimulationExplorationRoundJournal,
	state simulationengine.State,
	round simulationengine.Round,
	index int,
	completion runCompletion,
	choiceIdentity choice.ExecutionIdentity,
	summary *CampaignResult,
	distinct map[record.SHA256]string,
	semanticProbes map[string]struct{},
	choiceFeatures map[string]struct{},
) (simulationExplorationRoundResult, error) {
	if err := ctx.Err(); err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: contextFailureReason(err), Err: err}
	}
	if err := prepared.Verify(); err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	candidate := round.Candidates[index]
	worldBundle := noneWorldBundle()
	if len(completion.result.WorldRecord) != 0 {
		recording, err := world.DecodeRecording(completion.result.WorldRecord)
		if err == nil {
			worldBundle, err = execution.ComposeRecording(recording, config.WorldTransitionLimit)
		}
		if err == nil {
			initialWorld, _, validateErr := execution.Validate(worldBundle.Manifest, worldBundle.Payloads)
			if validateErr != nil {
				err = validateErr
			} else if worldBundle.Manifest.Initial.Schema != "gomadv3.world.snapshot/v1" || uint64(initialWorld.Config.Seed) != completion.job.seed {
				err = fmt.Errorf("World record seed or schema does not match seed %d", completion.job.seed)
			}
		}
		if err != nil {
			return simulationExplorationRoundResult{}, &HostError{Reason: "world_record", Err: err}
		}
	}
	runCoverage, err := deterministicio.SummarizeSemanticProbes(nil)
	if err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: "semantic_coverage", Err: err}
	}
	if coverageHasSemantic(config.Coverage) {
		runCoverage, err = deterministicio.DecodeSemanticCoverage(completion.result.IOTranscript.Bytes)
		if err != nil {
			return simulationExplorationRoundResult{}, &HostError{Reason: "semantic_coverage", Err: err}
		}
	}
	runChoiceFeatures := []string{}
	var runChoiceProjection *choice.FeatureProjection
	if coverageHasChoice(config.Coverage) {
		projection, features, err := projectChoiceFeatures(completion.result.ChoiceTrace, prepared)
		if err != nil {
			return simulationExplorationRoundResult{}, &HostError{Reason: "choice_coverage", Err: err}
		}
		runChoiceProjection = &projection
		runChoiceFeatures = features
	}
	outcome := execution.Classify(completion.result, false, worldBundle.Manifest.Terminal)
	if outcome.Domain == "runner" {
		return simulationExplorationRoundResult{}, &HostError{Reason: outcome.Reason, Err: errors.New("simulation-exploration controller result is not expandable")}
	}
	tape, err := choice.ProjectReplayPlan(completion.result.ChoiceTrace.Trace, choiceIdentity)
	if err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: "choice_trace_malformed", Err: err}
	}
	completion.result.ChoiceTrace.TapeSHA256 = tape.SHA256
	completion.result.ChoiceTrace.Decisions = uint64(len(tape.Decisions))
	summary.ChoiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
	runtimeDecisions, err := simulationrecord.RuntimeDecisions(tape)
	if err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: "simulation_exploration_runtime", Err: err}
	}
	if len(completion.result.SimulationRecords) != 1 {
		return simulationExplorationRoundResult{}, &HostError{Reason: "simulation_exploration_record", Err: fmt.Errorf("simulation exploration records = %d, want 1", len(completion.result.SimulationRecords))}
	}
	explorationResult, err := simulationrecord.ResultForRecord(state.Config, candidate, completion.result.SimulationRecords[0], runtimeDecisions)
	if err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: "simulation_exploration_record", Err: err}
	}
	simulationPlan := []byte(completion.job.simulationPlan)
	simulationRecord := completion.result.SimulationRecords[0]
	simulationProfile, err := simulationrecord.ProjectArtifact(
		state.Config, candidate, simulationPlan, simulationRecord, runtimeDecisions, completion.job.simulationRecordLimit,
	)
	if err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: "simulation_exploration_record", Err: err}
	}
	simulationPayloads := &artifact.SimulationPayloads{Plan: simulationPlan, Record: simulationRecord}
	mountArtifact, err := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
	if err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: "manifest", Err: err}
	}
	roundValue := record.Uint64String(round.Index)
	depthValue := record.Uint64String(len(candidate.Overrides))
	run := campaign.ExecutionRecord{
		Strategy: string(StrategySimulationExploration), Round: &roundValue, CandidateSHA256: candidate.SHA256,
		ParentCandidateSHA256: candidate.ParentSHA256, ForcedDepth: &depthValue, OutcomeSHA256: explorationResult.OutcomeSHA256,
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination,
		ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	run.SemanticProbes = append([]string(nil), runCoverage.Probes...)
	run.ChoiceFeatures = append([]string(nil), runChoiceFeatures...)

	if config.CollectExecutionEvidence {
		runRecord := executionEvidence(config, prepared, baseEnvironment, completion, outcome, worldBundle.Manifest, mountArtifact, runCoverage, runChoiceProjection)
		summary.ExecutionEvidence = &runRecord
	}
	if outcome.Domain == "success" {
		if explorationResult.Failed {
			return simulationExplorationRoundResult{}, &HostError{Reason: "simulation_exploration_outcome", Err: errors.New("simulation failure completed with a successful target outcome")}
		}
		novelProbes := novelSemanticProbes(runCoverage.Probes, semanticProbes)
		novelChoices := novelStrings(runChoiceFeatures, choiceFeatures)
		retain := config.KeepSuccesses == KeepSuccessesAll || config.KeepSuccesses == KeepSuccessesNovel && (len(novelProbes) != 0 || len(novelChoices) != 0)
		if retain {
			if !completion.result.IOTranscript.Complete {
				return simulationExplorationRoundResult{}, &HostError{Reason: "success_artifact_publication", Err: errors.New("retained success requires a complete I/O transcript for exact replay")}
			}
			if summary.RetainedSuccesses >= config.SuccessArtifactLimit || summary.RetainedSuccessBytes >= config.SuccessBytesLimit {
				return simulationExplorationRoundResult{}, &HostError{Reason: "success_retention_capacity", Err: errors.New("successful-execution retention capacity is exhausted")}
			}
			manifest, err := manifestForRun(config, prepared, baseEnvironment, completion, outcome, runID, worldBundle.Manifest, mountArtifact)
			if err != nil {
				return simulationExplorationRoundResult{}, &HostError{Reason: "success_artifact_publication", Err: err}
			}
			manifest.SimulationProfile = &simulationProfile
			published, err := artifact.PublishArtifact(artifact.Store{Root: filepath.Join(staged.Path(), "successes"), Context: ctx, MaximumBytes: config.SuccessBytesLimit - summary.RetainedSuccessBytes}, artifact.ArtifactInput{
				Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
				IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads, Simulation: simulationPayloads,
			})
			if err != nil {
				reason := "success_artifact_publication"
				var capacity *artifact.CapacityError
				if errors.As(err, &capacity) {
					reason = "success_retention_capacity"
				}
				return simulationExplorationRoundResult{}, &HostError{Reason: reason, Err: err}
			}
			finalPath, relative, err := simulationExplorationPublishedPath(batchPath, round.Index, staged.Path(), published.Path)
			if err != nil {
				return simulationExplorationRoundResult{}, &HostError{Reason: "artifact_path", Err: err}
			}
			bytes := record.Uint64String(published.StoredBytes)
			run.SuccessArtifact = &relative
			run.SuccessArtifactBytes = &bytes
			if config.KeepSuccesses == KeepSuccessesNovel {
				run.NovelSemanticProbes = append([]string(nil), novelProbes...)
				run.NovelChoiceFeatures = append([]string(nil), novelChoices...)
			}
			summary.SuccessArtifacts = append(summary.SuccessArtifacts, finalPath)
			summary.RetainedSuccesses++
			summary.RetainedSuccessBytes += published.StoredBytes
		}
		summary.Succeeded++
	} else {
		manifest, err := manifestForRun(config, prepared, baseEnvironment, completion, outcome, runID, worldBundle.Manifest, mountArtifact)
		if err != nil {
			return simulationExplorationRoundResult{}, &HostError{Reason: "manifest", Err: err}
		}
		manifest.SimulationProfile = &simulationProfile
		published, err := publishBoundedFailureArtifact(ctx, config, filepath.Join(staged.Path(), "failures"), manifest.Outcome.FailureSignature, distinct, &summary.failureArtifactBytes, artifact.ArtifactInput{
			Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
			IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads, Simulation: simulationPayloads,
		})
		if err != nil {
			return simulationExplorationRoundResult{}, &HostError{Reason: "artifact_publication", Err: err}
		}
		signature := published.Manifest.Outcome.FailureSignature
		artifactPath, found := distinct[signature]
		if !found {
			artifactPath, _, err = simulationExplorationPublishedPath(batchPath, round.Index, staged.Path(), published.Path)
			if err != nil {
				return simulationExplorationRoundResult{}, &HostError{Reason: "artifact_path", Err: err}
			}
			distinct[signature] = artifactPath
			summary.Artifacts = append(summary.Artifacts, artifactPath)
		} else {
			currentRound := filepath.Join(batchPath, "simulation-exploration", "rounds", fmt.Sprintf("%020d", round.Index)) + string(filepath.Separator)
			if !strings.HasPrefix(artifactPath, currentRound) {
				if err := os.RemoveAll(published.Path); err != nil {
					return simulationExplorationRoundResult{}, &HostError{Reason: "artifact_publication", Err: fmt.Errorf("remove duplicate staged failure: %w", err)}
				}
				if err := syncExplorationDirectory(filepath.Dir(published.Path)); err != nil {
					return simulationExplorationRoundResult{}, &HostError{Reason: "artifact_publication", Err: fmt.Errorf("sync duplicate failure removal: %w", err)}
				}
			}
		}
		relative, err := filepath.Rel(batchPath, artifactPath)
		if err != nil {
			return simulationExplorationRoundResult{}, &HostError{Reason: "artifact_path", Err: err}
		}
		relative = filepath.ToSlash(relative)
		run.FailureSignature = &signature
		run.Artifact = &relative
		summary.Failures++
		if outcome.Domain == "watchdog" {
			summary.Watchdogs++
		}
		if outcome.Reason == "world_replay_divergence" || explorationResult.Diverged {
			summary.ReplayDivergences++
		}
		explorationResult.Failed = true
		explorationResult.FailureSHA256 = signature
	}
	summary.Attempted++
	summary.DistinctFailures = uint64(len(distinct))
	addSemanticProbes(semanticProbes, runCoverage.Probes)
	addStrings(choiceFeatures, runChoiceFeatures)
	if err := completion.journal.Transition(campaign.ExecutionClassified); err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: "partial_write", Err: err}
	}
	if err := completion.journal.Complete(); err != nil {
		return simulationExplorationRoundResult{}, &HostError{Reason: "partial_cleanup", Err: err}
	}
	return simulationExplorationRoundResult{result: explorationResult, run: run}, nil
}

func simulationExplorationExecutionIdentity(config CampaignSpec, prepared target.Prepared, profile deterministicio.Spec, choiceImplementation [32]byte) (record.SHA256, error) {
	encoded, err := canonicaljson.CanonicalJSON(struct {
		TargetSHA256         record.SHA256            `json:"target_sha256"`
		ToolchainBuildKey    string                   `json:"toolchain_build_key"`
		GOOS                 string                   `json:"goos"`
		GOARCH               string                   `json:"goarch"`
		ChoiceImplementation record.SHA256            `json:"choice_implementation_sha256"`
		IOProfile            deterministicio.Contract `json:"io_profile"`
		RunnerBuild          string                   `json:"runner_build"`
	}{
		TargetSHA256: record.SHA256(prepared.SHA256), ToolchainBuildKey: prepared.BuildKey,
		GOOS: prepared.TargetGOOS, GOARCH: prepared.TargetGOARCH, ChoiceImplementation: record.SHA256FromSum(choiceImplementation),
		IOProfile: profile.Identity(), RunnerBuild: config.RunnerBuild,
	})
	if err != nil {
		return "", err
	}
	return record.DomainHash("gomadv3-simulation-exploration-execution/v1", encoded), nil
}

func simulationExplorationSegmentCapacity(config CampaignSpec) (uint64, error) {
	const maximum = uint64(1 << 30)
	const overhead = uint64(4 << 20)
	if config.MaxExplorationBytes > maximum-overhead {
		return 0, errors.New("simulation-exploration round evidence exceeds the 1 GiB journal bound")
	}
	runsPerRound := uint64(config.Parallel)
	if runsPerRound > config.MaxExecutions {
		runsPerRound = config.MaxExecutions
	}
	if runsPerRound == 0 || config.MaxExplorationResultBytes > (maximum-config.MaxExplorationBytes)/runsPerRound {
		return 0, errors.New("simulation-exploration round evidence exceeds the 1 GiB journal bound")
	}
	required := config.MaxExplorationBytes + runsPerRound*config.MaxExplorationResultBytes
	if required > maximum-overhead {
		return 0, errors.New("simulation-exploration round evidence exceeds the 1 GiB journal bound")
	}
	return required + overhead, nil
}

func simulationExplorationPublishedPath(batchPath string, round uint64, stagedPath, publishedPath string) (string, string, error) {
	withinRound, err := filepath.Rel(stagedPath, publishedPath)
	if err != nil || withinRound == ".." || len(withinRound) >= 3 && withinRound[:3] == ".."+string(filepath.Separator) {
		return "", "", errors.Join(errors.New("simulation exploration artifact escaped its staged round"), err)
	}
	finalPath := filepath.Join(batchPath, "simulation-exploration", "rounds", fmt.Sprintf("%020d", round), withinRound)
	relative, err := filepath.Rel(batchPath, finalPath)
	if err != nil {
		return "", "", err
	}
	return finalPath, filepath.ToSlash(relative), nil
}
