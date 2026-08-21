package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	"go.temporal.io/server/tools/gomadv3/runner/internal/simulationexploration"
	"go.temporal.io/server/tools/gomadv3/target"
	"go.temporal.io/server/tools/gomadv3/world"
)

const combinedExplorationRecordLimit = uint64(128 << 20)

type combinedFrontierRoundResult struct {
	result combinedfrontier.Result
	run    campaignstore.ExecutionRecord
}

func runCombinedFrontierLocal(
	ctx context.Context,
	config CampaignSpec,
	selection SeedSelection,
	baseEnvironment []evidence.Environment,
	readOnlyMounts []deterministicio.Mapping,
	prepared target.Prepared,
	profile deterministicio.Spec,
	journal *campaignstore.CampaignJournal,
	runID string,
	resuming bool,
	resumedRuns []campaignstore.ExecutionRecord,
	summary *CampaignResult,
	reportProgress func(CampaignPhase, int) error,
) error {
	baseSeed, ok := selection.SeedAt(0)
	if !ok || selection.Count() != 1 {
		return errors.New("combined-frontier base seed is unavailable")
	}
	choiceImplementation, err := choice.ImplementationIdentity(prepared.BuildKey)
	if err != nil {
		return &HostError{Reason: "combined_frontier_setup", Err: err}
	}
	choiceIdentity, err := choiceExecutionIdentity(prepared, choiceImplementation)
	if err != nil {
		return &HostError{Reason: "combined_frontier_setup", Err: err}
	}
	executionSHA256, err := combinedFrontierExecutionIdentity(config, prepared, profile, choiceImplementation)
	if err != nil {
		return &HostError{Reason: "combined_frontier_setup", Err: err}
	}
	failureBudget := config.FailureBudget
	if config.OnFailure == PolicyAll {
		failureBudget = config.MaxRuns
	}
	frontierConfig := combinedfrontier.Config{
		ExecutionSHA256: executionSHA256, ControllerSHA256: combinedfrontier.ImplementationSHA256(), BaseSeed: baseSeed,
		Parallel: config.Parallel, MaxRuns: config.MaxRuns, MaxForcedDecisions: config.MaxForcedDecisions,
		MaxFrontierBytes: config.MaxFrontierBytes, MaxResultBytes: config.MaxExplorationResultBytes,
		FailureBudget: failureBudget, Limits: config.CombinedDimensionLimits,
	}
	state, err := combinedfrontier.New(frontierConfig)
	if err != nil {
		return &HostError{Reason: "combined_frontier_setup", Err: err}
	}
	segmentBytes, err := combinedFrontierSegmentCapacity(config)
	if err != nil {
		return &HostError{Reason: "combined_frontier_setup", Err: err}
	}
	var frontierJournal *campaignstore.CombinedFrontierJournal
	if resuming {
		var recovery uint64
		frontierJournal, state, recovery, err = campaignstore.ResumeCombinedFrontierJournal(ctx, journal.Path(), frontierConfig, segmentBytes)
		if err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
		summary.RecoveryExecutions += recovery
		committedRuns := frontierJournal.CommittedExecutions()
		if err := reconcileFrontierExecutions(journal, resumedRuns, committedRuns); err != nil {
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
		frontierJournal, err = campaignstore.NewCombinedFrontierJournal(ctx, journal.Path(), state, segmentBytes)
		if err != nil {
			return &HostError{Reason: "combined_frontier_setup", Err: err}
		}
	}
	combinedSummary := state.Summary()
	summary.CombinedFrontier = &combinedSummary

	distinct := make(map[evidence.SHA256]string)
	semanticProbes := make(map[string]struct{})
	choiceFeatures := make(map[string]struct{})
	if len(frontierJournal.CommittedExecutions()) != 0 {
		restored, err := restoreResumeSummary(journal.Path(), selection, frontierJournal.CommittedExecutions())
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
		staged, err := frontierJournal.StageRound(round)
		if err != nil {
			return &HostError{Reason: "combined_frontier_stage", Err: err}
		}
		completions, err := executeCombinedFrontierRound(ctx, config, executor, prepared, baseEnvironment, profile, readOnlyMounts, staged, state, round, choiceIdentity, reportProgress)
		if err != nil {
			return err
		}
		roundSummary := cloneSummary(*summary)
		roundDistinct := cloneFailurePaths(distinct)
		roundProbes := cloneStringSet(semanticProbes)
		roundChoiceFeatures := cloneStringSet(choiceFeatures)
		results := make([]combinedfrontier.Result, len(round.Candidates))
		runs := make([]campaignstore.ExecutionRecord, len(round.Candidates))
		for index, completion := range completions {
			processed, err := processCombinedFrontierCompletion(
				ctx, config, prepared, baseEnvironment, readOnlyMounts, runID, journal.Path(), staged,
				state, round, index, completion, choiceIdentity, &roundSummary, roundDistinct, roundProbes, roundChoiceFeatures,
			)
			if err != nil {
				return err
			}
			results[index] = processed.result
			runs[index] = processed.run
			if err := staged.RecordExecution(index, processed.run); err != nil {
				return &HostError{Reason: "combined_frontier_stage", Err: err}
			}
		}
		next, segment, err := combinedfrontier.CommitRound(state, round, results)
		if err != nil {
			return &HostError{Reason: "combined_frontier_commit", Err: err}
		}
		if err := frontierJournal.CommitRound(staged, segment); err != nil {
			return &HostError{Reason: "combined_frontier_commit", Err: err}
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
		summary.CombinedFrontier = &combinedSummary
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
	failureSignatures := make([]evidence.SHA256, 0, len(distinct))
	for signature := range distinct {
		failureSignatures = append(failureSignatures, signature)
	}
	combinedSummary = state.Summary()
	summary.CombinedFrontier = &combinedSummary
	summary.StopReason = StopReason(combinedSummary.StopReason)
	if err := journal.Publish(campaignstore.CampaignSummary{
		Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures, Watchdogs: summary.Watchdogs,
		Cancelled: summary.Cancelled, DistinctFailures: summary.DistinctFailures, RetainedSuccesses: summary.RetainedSuccesses,
		RetainedSuccessBytes: summary.RetainedSuccessBytes, StopReason: string(summary.StopReason), FailureSignatures: failureSignatures,
		CombinedFrontier: &combinedSummary, CombinedFrontierImplementationSHA256: combinedfrontier.ImplementationSHA256(),
		CombinedFrontierChainSHA256: frontierJournal.ChainSHA256(), RecoveryExecutions: summary.RecoveryExecutions,
	}); err != nil {
		return &HostError{Reason: "batch_publish", Err: err}
	}
	if err := reportProgress(ProgressComplete, 0); err != nil {
		return &HostError{Reason: "progress_output", Err: err}
	}
	return nil
}

func executeCombinedFrontierRound(
	ctx context.Context,
	config CampaignSpec,
	executor Executor,
	prepared target.Prepared,
	baseEnvironment []evidence.Environment,
	profile deterministicio.Spec,
	readOnlyMounts []deterministicio.Mapping,
	staged *campaignstore.CombinedFrontierRoundJournal,
	state combinedfrontier.State,
	round combinedfrontier.Round,
	choiceIdentity choice.ExecutionIdentity,
	reportProgress func(CampaignPhase, int) error,
) ([]runCompletion, error) {
	roundCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	completionChannel := make(chan runCompletion, len(round.Candidates))
	startOrdinal := state.LogicalExecutions
	for index, candidate := range round.Candidates {
		candidateExecution, err := simulationexploration.ExecutionForCandidate(state.Config, candidate, choiceIdentity)
		if err != nil {
			return nil, &HostError{Reason: "combined_frontier_control", Err: err}
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
				return nil, &HostError{Reason: "combined_frontier_order", Err: errors.New("combined frontier completion ordinal is outside its round")}
			}
			index := int(completion.job.ordinal - startOrdinal)
			if seen[index] {
				cancel()
				return nil, &HostError{Reason: "combined_frontier_order", Err: errors.New("combined frontier completion ordinal is duplicated")}
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
			return nil, &HostError{Reason: "runner_cancelled", Err: errors.New("combined frontier candidate was cancelled")}
		}
	}
	return completions, nil
}

func processCombinedFrontierCompletion(
	ctx context.Context,
	config CampaignSpec,
	prepared target.Prepared,
	baseEnvironment []evidence.Environment,
	readOnlyMounts []deterministicio.Mapping,
	runID string,
	batchPath string,
	staged *campaignstore.CombinedFrontierRoundJournal,
	state combinedfrontier.State,
	round combinedfrontier.Round,
	index int,
	completion runCompletion,
	choiceIdentity choice.ExecutionIdentity,
	summary *CampaignResult,
	distinct map[evidence.SHA256]string,
	semanticProbes map[string]struct{},
	choiceFeatures map[string]struct{},
) (combinedFrontierRoundResult, error) {
	if err := ctx.Err(); err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: contextFailureReason(err), Err: err}
	}
	if err := prepared.Verify(); err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: "prepared_target_integrity", Err: err}
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
			return combinedFrontierRoundResult{}, &HostError{Reason: "world_record", Err: err}
		}
	}
	runCoverage, err := deterministicio.SummarizeSemanticProbes(nil)
	if err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: "semantic_coverage", Err: err}
	}
	if coverageHasSemantic(config.Coverage) {
		runCoverage, err = deterministicio.DecodeSemanticCoverage(completion.result.IOTranscript.Bytes)
		if err != nil {
			return combinedFrontierRoundResult{}, &HostError{Reason: "semantic_coverage", Err: err}
		}
	}
	runChoiceFeatures := []string{}
	var runChoiceProjection *choice.FeatureProjection
	if coverageHasChoice(config.Coverage) {
		projection, features, err := projectChoiceFeatures(completion.result.ChoiceTrace, prepared)
		if err != nil {
			return combinedFrontierRoundResult{}, &HostError{Reason: "choice_coverage", Err: err}
		}
		runChoiceProjection = &projection
		runChoiceFeatures = features
	}
	outcome := execution.Classify(completion.result, false, worldBundle.Manifest.Terminal)
	if outcome.Domain == "runner" {
		return combinedFrontierRoundResult{}, &HostError{Reason: outcome.Reason, Err: errors.New("combined-frontier controller result is not expandable")}
	}
	tape, err := choice.ProjectReplayPlan(completion.result.ChoiceTrace.Trace, choiceIdentity)
	if err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: "choice_trace_malformed", Err: err}
	}
	completion.result.ChoiceTrace.TapeSHA256 = tape.SHA256
	completion.result.ChoiceTrace.Decisions = uint64(len(tape.Decisions))
	summary.ChoiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
	runtimeDecisions, err := simulationexploration.RuntimeDecisions(tape)
	if err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: "combined_frontier_runtime", Err: err}
	}
	if len(completion.result.SimulationRecords) != 1 {
		return combinedFrontierRoundResult{}, &HostError{Reason: "combined_frontier_record", Err: fmt.Errorf("simulation exploration records = %d, want 1", len(completion.result.SimulationRecords))}
	}
	frontierResult, err := simulationexploration.ResultForRecord(state.Config, candidate, completion.result.SimulationRecords[0], runtimeDecisions)
	if err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: "combined_frontier_record", Err: err}
	}
	simulationPlan := []byte(completion.job.simulationPlan)
	simulationRecord := completion.result.SimulationRecords[0]
	simulationProfile, err := simulationexploration.ProjectArtifact(
		state.Config, candidate, simulationPlan, simulationRecord, runtimeDecisions, completion.job.simulationRecordLimit,
	)
	if err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: "combined_frontier_record", Err: err}
	}
	simulationPayloads := &campaignstore.SimulationPayloads{Plan: simulationPlan, Record: simulationRecord}
	mountArtifact, err := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
	if err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: "manifest", Err: err}
	}
	roundValue := evidence.Uint64String(round.Index)
	depthValue := evidence.Uint64String(len(candidate.Overrides))
	run := campaignstore.ExecutionRecord{
		Strategy: string(StrategyCombinedFrontier), Round: &roundValue, CandidateSHA256: candidate.SHA256,
		ParentCandidateSHA256: candidate.ParentSHA256, ForcedDepth: &depthValue, OutcomeSHA256: frontierResult.OutcomeSHA256,
		SelectionOrdinal: evidence.Uint64String(completion.job.ordinal), Seed: evidence.Uint64String(completion.job.seed),
		Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination,
		ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	run.SemanticProbes = append([]string(nil), runCoverage.Probes...)
	run.ChoiceFeatures = append([]string(nil), runChoiceFeatures...)

	if config.CollectRunEvidence {
		runRecord := runEvidence(config, prepared, baseEnvironment, completion, outcome, worldBundle.Manifest, mountArtifact, runCoverage, runChoiceProjection)
		summary.ExecutionEvidence = &runRecord
	}
	if outcome.Domain == "success" {
		if frontierResult.Failed {
			return combinedFrontierRoundResult{}, &HostError{Reason: "combined_frontier_outcome", Err: errors.New("simulation failure completed with a successful target outcome")}
		}
		novelProbes := novelSemanticProbes(runCoverage.Probes, semanticProbes)
		novelChoices := novelStrings(runChoiceFeatures, choiceFeatures)
		retain := config.KeepSuccesses == KeepSuccessesAll || config.KeepSuccesses == KeepSuccessesNovel && (len(novelProbes) != 0 || len(novelChoices) != 0)
		if retain {
			if !completion.result.IOTranscript.Complete {
				return combinedFrontierRoundResult{}, &HostError{Reason: "success_artifact_publication", Err: errors.New("retained success requires a complete I/O transcript for exact replay")}
			}
			if summary.RetainedSuccesses >= config.SuccessArtifactLimit || summary.RetainedSuccessBytes >= config.SuccessBytesLimit {
				return combinedFrontierRoundResult{}, &HostError{Reason: "success_retention_capacity", Err: errors.New("successful-run retention capacity is exhausted")}
			}
			manifest, err := manifestForRun(config, prepared, baseEnvironment, completion, outcome, runID, worldBundle.Manifest, mountArtifact)
			if err != nil {
				return combinedFrontierRoundResult{}, &HostError{Reason: "success_artifact_publication", Err: err}
			}
			manifest.SimulationProfile = &simulationProfile
			published, err := campaignstore.PublishArtifact(evidence.Store{Root: filepath.Join(staged.Path(), "successes"), Context: ctx, MaximumBytes: config.SuccessBytesLimit - summary.RetainedSuccessBytes}, campaignstore.ArtifactInput{
				Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
				IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads, Simulation: simulationPayloads,
			})
			if err != nil {
				reason := "success_artifact_publication"
				var capacity *evidence.CapacityError
				if errors.As(err, &capacity) {
					reason = "success_retention_capacity"
				}
				return combinedFrontierRoundResult{}, &HostError{Reason: reason, Err: err}
			}
			finalPath, relative, err := combinedFrontierPublishedPath(batchPath, round.Index, staged.Path(), published.Path)
			if err != nil {
				return combinedFrontierRoundResult{}, &HostError{Reason: "artifact_path", Err: err}
			}
			bytes := evidence.Uint64String(published.StoredBytes)
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
			return combinedFrontierRoundResult{}, &HostError{Reason: "manifest", Err: err}
		}
		manifest.SimulationProfile = &simulationProfile
		published, err := publishBoundedFailureArtifact(ctx, config, filepath.Join(staged.Path(), "failures"), manifest.Outcome.FailureSignature, distinct, &summary.failureArtifactBytes, campaignstore.ArtifactInput{
			Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
			IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads, Simulation: simulationPayloads,
		})
		if err != nil {
			return combinedFrontierRoundResult{}, &HostError{Reason: "artifact_publication", Err: err}
		}
		signature := published.Manifest.Outcome.FailureSignature
		artifactPath, found := distinct[signature]
		if !found {
			artifactPath, _, err = combinedFrontierPublishedPath(batchPath, round.Index, staged.Path(), published.Path)
			if err != nil {
				return combinedFrontierRoundResult{}, &HostError{Reason: "artifact_path", Err: err}
			}
			distinct[signature] = artifactPath
			summary.Artifacts = append(summary.Artifacts, artifactPath)
		} else {
			currentRound := filepath.Join(batchPath, "combined-frontier", "rounds", fmt.Sprintf("%020d", round.Index)) + string(filepath.Separator)
			if !strings.HasPrefix(artifactPath, currentRound) {
				if err := os.RemoveAll(published.Path); err != nil {
					return combinedFrontierRoundResult{}, &HostError{Reason: "artifact_publication", Err: fmt.Errorf("remove duplicate staged failure: %w", err)}
				}
				if err := syncFrontierDirectory(filepath.Dir(published.Path)); err != nil {
					return combinedFrontierRoundResult{}, &HostError{Reason: "artifact_publication", Err: fmt.Errorf("sync duplicate failure removal: %w", err)}
				}
			}
		}
		relative, err := filepath.Rel(batchPath, artifactPath)
		if err != nil {
			return combinedFrontierRoundResult{}, &HostError{Reason: "artifact_path", Err: err}
		}
		relative = filepath.ToSlash(relative)
		run.FailureSignature = &signature
		run.Artifact = &relative
		summary.Failures++
		if outcome.Domain == "watchdog" {
			summary.Watchdogs++
		}
		if outcome.Reason == "world_replay_divergence" || frontierResult.Diverged {
			summary.ReplayDivergences++
		}
		frontierResult.Failed = true
		frontierResult.FailureSHA256 = signature
	}
	summary.Attempted++
	summary.DistinctFailures = uint64(len(distinct))
	addSemanticProbes(semanticProbes, runCoverage.Probes)
	addStrings(choiceFeatures, runChoiceFeatures)
	if err := completion.journal.Transition(campaignstore.ExecutionClassified); err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: "partial_write", Err: err}
	}
	if err := completion.journal.Complete(); err != nil {
		return combinedFrontierRoundResult{}, &HostError{Reason: "partial_cleanup", Err: err}
	}
	return combinedFrontierRoundResult{result: frontierResult, run: run}, nil
}

func combinedFrontierExecutionIdentity(config CampaignSpec, prepared target.Prepared, profile deterministicio.Spec, choiceImplementation [32]byte) (evidence.SHA256, error) {
	encoded, err := evidence.CanonicalJSON(struct {
		TargetSHA256         evidence.SHA256          `json:"target_sha256"`
		ToolchainBuildKey    string                   `json:"toolchain_build_key"`
		GOOS                 string                   `json:"goos"`
		GOARCH               string                   `json:"goarch"`
		ChoiceImplementation evidence.SHA256          `json:"choice_implementation_sha256"`
		IOProfile            deterministicio.Contract `json:"io_profile"`
		RunnerBuild          string                   `json:"runner_build"`
	}{
		TargetSHA256: evidence.SHA256(prepared.SHA256), ToolchainBuildKey: prepared.BuildKey,
		GOOS: prepared.TargetGOOS, GOARCH: prepared.TargetGOARCH, ChoiceImplementation: evidence.SHA256FromSum(choiceImplementation),
		IOProfile: profile.Identity(), RunnerBuild: config.RunnerBuild,
	})
	if err != nil {
		return "", err
	}
	return evidence.DomainHash("gomadv3-simulation-exploration-execution/v1", encoded), nil
}

func combinedFrontierSegmentCapacity(config CampaignSpec) (uint64, error) {
	const maximum = uint64(1 << 30)
	const overhead = uint64(4 << 20)
	if config.MaxFrontierBytes > maximum-overhead {
		return 0, errors.New("combined-frontier round evidence exceeds the 1 GiB journal bound")
	}
	runsPerRound := uint64(config.Parallel)
	if runsPerRound > config.MaxRuns {
		runsPerRound = config.MaxRuns
	}
	if runsPerRound == 0 || config.MaxExplorationResultBytes > (maximum-config.MaxFrontierBytes)/runsPerRound {
		return 0, errors.New("combined-frontier round evidence exceeds the 1 GiB journal bound")
	}
	required := config.MaxFrontierBytes + runsPerRound*config.MaxExplorationResultBytes
	if required > maximum-overhead {
		return 0, errors.New("combined-frontier round evidence exceeds the 1 GiB journal bound")
	}
	return required + overhead, nil
}

func combinedFrontierPublishedPath(batchPath string, round uint64, stagedPath, publishedPath string) (string, string, error) {
	withinRound, err := filepath.Rel(stagedPath, publishedPath)
	if err != nil || withinRound == ".." || len(withinRound) >= 3 && withinRound[:3] == ".."+string(filepath.Separator) {
		return "", "", errors.Join(errors.New("combined frontier artifact escaped its staged round"), err)
	}
	finalPath := filepath.Join(batchPath, "combined-frontier", "rounds", fmt.Sprintf("%020d", round), withinRound)
	relative, err := filepath.Rel(batchPath, finalPath)
	if err != nil {
		return "", "", err
	}
	return finalPath, filepath.ToSlash(relative), nil
}
