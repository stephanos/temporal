package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/choicefrontier"
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

type frontierRoundResult struct {
	completion runCompletion
	result     choicefrontier.Result
	run        artifact.RunRecord
}

func runChoiceFrontierLocal(
	ctx context.Context,
	config Config,
	selection SeedSelection,
	baseEnvironment []record.Environment,
	readOnlyMounts []romount.Mapping,
	prepared target.Prepared,
	profile ioprofile.ProfileSpec,
	journal *artifact.BatchJournal,
	runID string,
	resuming bool,
	resumedRuns []artifact.RunRecord,
	summary *Summary,
	reportProgress func(ProgressPhase, int) error,
) error {
	baseSeed, ok := selection.SeedAt(0)
	if !ok || selection.Count() != 1 {
		return errors.New("choice-frontier base seed is unavailable")
	}
	implementation, err := choicewire.ImplementationIdentity(prepared.BuildKey)
	if err != nil {
		return &HostError{Reason: "choice_frontier_setup", Err: err}
	}
	executionIdentity, err := choiceExecutionIdentity(prepared, implementation)
	if err != nil {
		return &HostError{Reason: "choice_frontier_setup", Err: err}
	}
	frontierConfig := choicefrontier.Config{
		Execution: executionIdentity, ControllerSHA256: choicefrontier.ImplementationSHA256(), BaseSeed: baseSeed, Parallel: config.Parallel,
		MaxRuns: config.MaxRuns, MaxChoiceDepth: config.MaxChoiceDepth, MaxFrontierBytes: config.MaxFrontierBytes,
		FailurePolicy: choicefrontier.FailurePolicy(config.OnFailure), FailureBudget: config.FailureBudget,
	}
	state, err := choicefrontier.New(frontierConfig)
	if err != nil {
		return &HostError{Reason: "choice_frontier_setup", Err: err}
	}
	segmentBytes, err := frontierSegmentCapacity(config)
	if err != nil {
		return &HostError{Reason: "choice_frontier_setup", Err: err}
	}
	var frontierJournal *artifact.FrontierJournal
	if resuming {
		var recovery uint64
		frontierJournal, state, recovery, err = artifact.ResumeFrontierJournal(ctx, journal.Path(), frontierConfig, segmentBytes)
		if err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
		summary.RecoveryExecutions += recovery
		committedRuns := frontierJournal.CommittedRuns()
		if err := reconcileFrontierRuns(journal, resumedRuns, committedRuns); err != nil {
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
		frontierJournal, err = artifact.NewFrontierJournal(ctx, journal.Path(), state, segmentBytes)
		if err != nil {
			return &HostError{Reason: "choice_frontier_setup", Err: err}
		}
	}
	frontierSummary := state.Summary()
	summary.Frontier = &frontierSummary

	distinct := make(map[record.SHA256]string)
	semanticProbes := make(map[string]struct{})
	choiceFeatures := make(map[string]struct{})
	if len(frontierJournal.CommittedRuns()) != 0 {
		restored, err := restoreResumeSummary(journal.Path(), selection, frontierJournal.CommittedRuns())
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
			return &HostError{Reason: "choice_frontier_stage", Err: err}
		}
		completions, err := executeFrontierRound(ctx, config, executor, prepared, baseEnvironment, profile, readOnlyMounts, staged, state, round, reportProgress)
		if err != nil {
			return err
		}
		roundSummary := cloneSummary(*summary)
		roundDistinct := cloneFailurePaths(distinct)
		roundProbes := cloneStringSet(semanticProbes)
		roundChoiceFeatures := cloneStringSet(choiceFeatures)
		results := make([]choicefrontier.Result, len(round.Candidates))
		runs := make([]artifact.RunRecord, len(round.Candidates))
		for index, completion := range completions {
			processed, err := processFrontierCompletion(ctx, config, prepared, baseEnvironment, readOnlyMounts, runID, journal.Path(), staged, state, round, index, completion, &roundSummary, roundDistinct, roundProbes, roundChoiceFeatures)
			if err != nil {
				return err
			}
			results[index] = processed.result
			runs[index] = processed.run
			if err := staged.RecordRun(index, processed.run); err != nil {
				return &HostError{Reason: "choice_frontier_stage", Err: err}
			}
		}
		next, segment, err := choicefrontier.CommitRound(state, round, results)
		if err != nil {
			return &HostError{Reason: "choice_frontier_commit", Err: err}
		}
		if err := frontierJournal.CommitRound(staged, segment); err != nil {
			return &HostError{Reason: "choice_frontier_commit", Err: err}
		}
		for _, run := range runs {
			if err := journal.AppendRun(run); err != nil {
				return &HostError{Reason: "runs_append", Err: err}
			}
		}
		state = next
		distinct = roundDistinct
		semanticProbes = roundProbes
		choiceFeatures = roundChoiceFeatures
		*summary = roundSummary
		frontierSummary := state.Summary()
		summary.Frontier = &frontierSummary
		summary.StopReason = StopReason(frontierSummary.StopReason)
		if err := reportProgress(ProgressRunning, 0); err != nil {
			return &HostError{Reason: "progress_output", Err: err}
		}
	}

	if coverageHasSemantic(config.Coverage) {
		coverage, err := ioprofile.SummarizeSemanticProbes(sortedProbeList(semanticProbes))
		if err != nil {
			return &HostError{Reason: "semantic_coverage", Err: err}
		}
		summary.SemanticCoverage = &coverage
		missing, err := ioprofile.MissingRequiredSemanticProbes(coverage, config.RequiredSemanticProbes)
		if err != nil {
			return err
		}
		if len(missing) != 0 {
			return &ioprofile.MissingSemanticProbesError{Probes: missing}
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
	frontierSummary = state.Summary()
	summary.Frontier = &frontierSummary
	summary.StopReason = StopReason(frontierSummary.StopReason)
	if err := journal.Publish(artifact.BatchSummary{
		Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures, Watchdogs: summary.Watchdogs,
		Cancelled: summary.Cancelled, DistinctFailures: summary.DistinctFailures, RetainedSuccesses: summary.RetainedSuccesses,
		RetainedSuccessBytes: summary.RetainedSuccessBytes, StopReason: string(summary.StopReason), FailureSignatures: failureSignatures,
		Frontier: &frontierSummary, FrontierImplementationSHA256: choicefrontier.ImplementationSHA256(), FrontierChainSHA256: frontierJournal.ChainSHA256(), RecoveryExecutions: summary.RecoveryExecutions,
	}); err != nil {
		return &HostError{Reason: "batch_publish", Err: err}
	}
	if err := reportProgress(ProgressComplete, 0); err != nil {
		return &HostError{Reason: "progress_output", Err: err}
	}
	return nil
}

func executeFrontierRound(
	ctx context.Context,
	config Config,
	executor Executor,
	prepared target.Prepared,
	baseEnvironment []record.Environment,
	profile ioprofile.ProfileSpec,
	readOnlyMounts []romount.Mapping,
	staged *artifact.FrontierRoundJournal,
	state choicefrontier.State,
	round choicefrontier.Round,
	reportProgress func(ProgressPhase, int) error,
) ([]runCompletion, error) {
	roundCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	completionChannel := make(chan runCompletion, len(round.Candidates))
	startOrdinal := state.LogicalExecutions
	for index, candidate := range round.Candidates {
		var tape *choicewire.Tape
		mode := choicewire.ModeRecord
		if candidate.ForcedDepth != 0 {
			prefix, err := candidate.PrefixTape(state.Config.Execution)
			if err != nil {
				return nil, &HostError{Reason: "choice_frontier_prefix", Err: err}
			}
			tape = &prefix
			mode = choicewire.ModePrefix
		}
		job := runJob{ordinal: startOrdinal + uint64(index), seed: state.Config.BaseSeed, choiceMode: mode, choiceTape: tape}
		go runSeed(roundCtx, config, executor, prepared, baseEnvironment, profile, readOnlyMounts, staged, job, completionChannel)
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
				return nil, &HostError{Reason: "choice_frontier_order", Err: errors.New("frontier completion ordinal is outside its round")}
			}
			index := int(completion.job.ordinal - startOrdinal)
			if seen[index] {
				cancel()
				return nil, &HostError{Reason: "choice_frontier_order", Err: errors.New("frontier completion ordinal is duplicated")}
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
			return nil, &HostError{Reason: "runner_cancelled", Err: errors.New("choice-frontier candidate was cancelled")}
		}
	}
	return completions, nil
}

func processFrontierCompletion(
	ctx context.Context,
	config Config,
	prepared target.Prepared,
	baseEnvironment []record.Environment,
	readOnlyMounts []romount.Mapping,
	runID string,
	batchPath string,
	staged *artifact.FrontierRoundJournal,
	state choicefrontier.State,
	round choicefrontier.Round,
	index int,
	completion runCompletion,
	summary *Summary,
	distinct map[record.SHA256]string,
	semanticProbes map[string]struct{},
	choiceFeatures map[string]struct{},
) (frontierRoundResult, error) {
	if err := ctx.Err(); err != nil {
		return frontierRoundResult{}, &HostError{Reason: contextFailureReason(err), Err: err}
	}
	if err := prepared.Verify(); err != nil {
		return frontierRoundResult{}, &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	candidate := round.Candidates[index]
	worldBundle := noneWorldBundle()
	if len(completion.result.WorldRecord) != 0 {
		recording, err := world.DecodeRecording(completion.result.WorldRecord)
		if err == nil {
			worldBundle, err = worldrecord.ComposeRecording(recording, config.WorldTransitionLimit)
		}
		if err == nil {
			initialWorld, _, validateErr := worldrecord.Validate(worldBundle.Manifest, worldBundle.Payloads)
			if validateErr != nil {
				err = validateErr
			} else if worldBundle.Manifest.Initial.Schema != "gomadv3.world.snapshot/v1" || uint64(initialWorld.Config.Seed) != completion.job.seed {
				err = fmt.Errorf("World record seed or schema does not match seed %d", completion.job.seed)
			}
		}
		if err != nil {
			return frontierRoundResult{}, &HostError{Reason: "world_record", Err: err}
		}
	}
	runCoverage, err := ioprofile.SummarizeSemanticProbes(nil)
	if err != nil {
		return frontierRoundResult{}, &HostError{Reason: "semantic_coverage", Err: err}
	}
	if coverageHasSemantic(config.Coverage) {
		runCoverage, err = ioprofile.DecodeSemanticCoverage(completion.result.IOTranscript.Bytes)
		if err != nil {
			return frontierRoundResult{}, &HostError{Reason: "semantic_coverage", Err: err}
		}
	}
	runChoiceFeatures := []string{}
	var runChoiceProjection *choicewire.FeatureProjection
	if coverageHasChoice(config.Coverage) {
		projection, features, err := projectChoiceFeatures(completion.result.ChoiceTrace, prepared)
		if err != nil {
			return frontierRoundResult{}, &HostError{Reason: "choice_coverage", Err: err}
		}
		runChoiceProjection = &projection
		runChoiceFeatures = features
	}
	outcome := executionoutcome.Classify(completion.result, false, worldBundle.Manifest.Terminal)
	if outcome.Domain == "runner" {
		return frontierRoundResult{}, &HostError{Reason: outcome.Reason, Err: errors.New("choice-frontier controller result is not expandable")}
	}
	tape, err := choicewire.ProjectDecisionTape(completion.result.ChoiceTrace.Trace, state.Config.Execution)
	if err != nil {
		return frontierRoundResult{}, &HostError{Reason: "choice_trace_malformed", Err: err}
	}
	completion.result.ChoiceTrace.TapeSHA256 = tape.SHA256
	completion.result.ChoiceTrace.Decisions = uint64(len(tape.Decisions))
	summary.ChoiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
	mountArtifact, err := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
	if err != nil {
		return frontierRoundResult{}, &HostError{Reason: "manifest", Err: err}
	}
	outcomeSHA256, err := frontierOutcomeSHA256(outcome, completion.result, worldBundle.Manifest)
	if err != nil {
		return frontierRoundResult{}, &HostError{Reason: "choice_frontier_outcome", Err: err}
	}
	roundValue := record.Uint64String(round.Index)
	depthValue := record.Uint64String(candidate.ForcedDepth)
	run := artifact.RunRecord{
		Strategy: string(StrategyChoiceFrontier), Round: &roundValue, CandidateSHA256: candidate.SHA256,
		ParentCandidateSHA256: candidate.ParentSHA256, PrefixSHA256: candidate.PrefixSHA256, ForcedDepth: &depthValue, OutcomeSHA256: outcomeSHA256,
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination,
		ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	run.SemanticProbes = append([]string(nil), runCoverage.Probes...)
	run.ChoiceFeatures = append([]string(nil), runChoiceFeatures...)

	if config.CollectRunEvidence {
		evidence := runEvidence(config, prepared, baseEnvironment, completion, outcome, worldBundle.Manifest, mountArtifact, runCoverage, runChoiceProjection)
		evidence.Frontier = &FrontierRunEvidence{
			ImplementationSHA256: choicefrontier.ImplementationSHA256(), Round: record.Uint64String(round.Index), CandidateSHA256: candidate.SHA256,
			ParentSHA256: candidate.ParentSHA256, PrefixSHA256: candidate.PrefixSHA256, ForcedDepth: record.Uint64String(candidate.ForcedDepth), OutcomeSHA256: outcomeSHA256,
		}
		summary.RunEvidence = &evidence
	}
	if outcome.Domain == "success" {
		novelProbes := novelSemanticProbes(runCoverage.Probes, semanticProbes)
		novelChoices := novelStrings(runChoiceFeatures, choiceFeatures)
		retain := config.KeepSuccesses == KeepSuccessesAll || config.KeepSuccesses == KeepSuccessesNovel && (len(novelProbes) != 0 || len(novelChoices) != 0)
		if retain {
			if !completion.result.IOTranscript.Complete {
				return frontierRoundResult{}, &HostError{Reason: "success_artifact_publication", Err: errors.New("retained success requires a complete I/O transcript for exact replay")}
			}
			if summary.RetainedSuccesses >= config.SuccessArtifactLimit || summary.RetainedSuccessBytes >= config.SuccessBytesLimit {
				return frontierRoundResult{}, &HostError{Reason: "success_retention_capacity", Err: errors.New("successful-run retention capacity is exhausted")}
			}
			manifest, err := manifestForRun(config, prepared, baseEnvironment, completion, outcome, runID, worldBundle.Manifest, mountArtifact)
			if err != nil {
				return frontierRoundResult{}, &HostError{Reason: "success_artifact_publication", Err: err}
			}
			published, err := (artifact.Store{Root: filepath.Join(staged.Path(), "successes"), Context: ctx, MaximumBytes: config.SuccessBytesLimit - summary.RetainedSuccessBytes}).Publish(artifact.Input{
				Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
				IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
			})
			if err != nil {
				reason := "success_artifact_publication"
				var capacity *artifact.CapacityError
				if errors.As(err, &capacity) {
					reason = "success_retention_capacity"
				}
				return frontierRoundResult{}, &HostError{Reason: reason, Err: err}
			}
			finalPath, relative, err := frontierPublishedPath(batchPath, round.Index, staged.Path(), published.Path)
			if err != nil {
				return frontierRoundResult{}, &HostError{Reason: "artifact_path", Err: err}
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
			return frontierRoundResult{}, &HostError{Reason: "manifest", Err: err}
		}
		published, err := (artifact.Store{Root: filepath.Join(staged.Path(), "failures"), Context: ctx}).Publish(artifact.Input{
			Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
			IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
		})
		if err != nil {
			return frontierRoundResult{}, &HostError{Reason: "artifact_publication", Err: err}
		}
		signature := published.Manifest.Outcome.FailureSignature
		artifactPath, found := distinct[signature]
		if !found {
			artifactPath, _, err = frontierPublishedPath(batchPath, round.Index, staged.Path(), published.Path)
			if err != nil {
				return frontierRoundResult{}, &HostError{Reason: "artifact_path", Err: err}
			}
			distinct[signature] = artifactPath
			summary.Artifacts = append(summary.Artifacts, artifactPath)
		} else {
			currentRound := filepath.Join(batchPath, "frontier", "rounds", fmt.Sprintf("%020d", round.Index)) + string(filepath.Separator)
			if !strings.HasPrefix(artifactPath, currentRound) {
				if err := os.RemoveAll(published.Path); err != nil {
					return frontierRoundResult{}, &HostError{Reason: "artifact_publication", Err: fmt.Errorf("remove duplicate staged failure: %w", err)}
				}
				if err := syncFrontierDirectory(filepath.Dir(published.Path)); err != nil {
					return frontierRoundResult{}, &HostError{Reason: "artifact_publication", Err: fmt.Errorf("sync duplicate failure removal: %w", err)}
				}
			}
		}
		relative, err := filepath.Rel(batchPath, artifactPath)
		if err != nil {
			return frontierRoundResult{}, &HostError{Reason: "artifact_path", Err: err}
		}
		relative = filepath.ToSlash(relative)
		run.FailureSignature = &signature
		run.Artifact = &relative
		summary.Failures++
		if outcome.Domain == "watchdog" {
			summary.Watchdogs++
		}
		if outcome.Reason == "world_replay_divergence" {
			summary.ReplayDivergences++
		}
	}
	summary.Attempted++
	summary.DistinctFailures = uint64(len(distinct))
	addSemanticProbes(semanticProbes, runCoverage.Probes)
	addStrings(choiceFeatures, runChoiceFeatures)
	if err := completion.journal.Transition(artifact.RunClassified); err != nil {
		return frontierRoundResult{}, &HostError{Reason: "partial_write", Err: err}
	}
	if err := completion.journal.Complete(); err != nil {
		return frontierRoundResult{}, &HostError{Reason: "partial_cleanup", Err: err}
	}
	failed := outcome.Domain == "target" || outcome.Domain == "watchdog"
	frontierResult := choicefrontier.Result{CandidateSHA256: candidate.SHA256, OutcomeSHA256: outcomeSHA256, Failed: failed, Trace: &tape}
	if run.FailureSignature != nil {
		frontierResult.FailureSHA256 = *run.FailureSignature
	}
	return frontierRoundResult{completion: completion, result: frontierResult, run: run}, nil
}

func reconcileFrontierRuns(journal *artifact.BatchJournal, projected, committed []artifact.RunRecord) error {
	if len(projected) > len(committed) {
		return errors.New("frontier run projection exceeds committed rounds")
	}
	for index := range projected {
		left, err := record.CanonicalJSON(projected[index])
		if err != nil {
			return err
		}
		right, err := record.CanonicalJSON(committed[index])
		if err != nil {
			return err
		}
		if string(left) != string(right) {
			return fmt.Errorf("frontier run projection diverges at ordinal %d", index)
		}
	}
	for _, run := range committed[len(projected):] {
		if err := journal.AppendRun(run); err != nil {
			return fmt.Errorf("restore committed frontier run projection: %w", err)
		}
	}
	return nil
}

func frontierSegmentCapacity(config Config) (uint64, error) {
	const maximum = uint64(1 << 30)
	const overhead = uint64(4 << 20)
	if config.MaxFrontierBytes > maximum-overhead {
		return 0, errors.New("choice-frontier round evidence exceeds the 1 GiB journal bound")
	}
	runsPerRound := uint64(config.Parallel)
	if runsPerRound > config.MaxRuns {
		runsPerRound = config.MaxRuns
	}
	if runsPerRound == 0 || config.ChoiceTraceLimit > (maximum-config.MaxFrontierBytes)/(runsPerRound) {
		return 0, errors.New("choice-frontier round evidence exceeds the 1 GiB journal bound")
	}
	required := config.MaxFrontierBytes + runsPerRound*config.ChoiceTraceLimit
	if required > maximum-overhead {
		return 0, errors.New("choice-frontier round evidence exceeds the 1 GiB journal bound")
	}
	return required + overhead, nil
}

func frontierPublishedPath(batchPath string, round uint64, stagedPath, publishedPath string) (string, string, error) {
	withinRound, err := filepath.Rel(stagedPath, publishedPath)
	if err != nil || withinRound == ".." || len(withinRound) >= 3 && withinRound[:3] == ".."+string(filepath.Separator) {
		return "", "", errors.Join(errors.New("frontier artifact escaped its staged round"), err)
	}
	finalPath := filepath.Join(batchPath, "frontier", "rounds", fmt.Sprintf("%020d", round), withinRound)
	relative, err := filepath.Rel(batchPath, finalPath)
	if err != nil {
		return "", "", err
	}
	return finalPath, filepath.ToSlash(relative), nil
}

func frontierOutcomeSHA256(outcome executionoutcome.Classification, result process.Result, worldRecord record.World) (record.SHA256, error) {
	projection := struct {
		Domain             string               `json:"domain"`
		Reason             string               `json:"reason"`
		Termination        string               `json:"termination"`
		ExitCode           *record.Uint64String `json:"exit_code,omitempty"`
		Signal             *string              `json:"signal,omitempty"`
		StdoutSHA256       record.SHA256        `json:"stdout_sha256"`
		StderrSHA256       record.SHA256        `json:"stderr_sha256"`
		IOTranscriptSHA256 record.SHA256        `json:"io_transcript_sha256"`
		WorldTerminal      record.WorldTerminal `json:"world_terminal"`
		WorldFinal         record.SHA256        `json:"world_final"`
	}{
		Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination,
		ExitCode: outcome.ExitCode, Signal: outcome.Signal,
		StdoutSHA256: record.SHA256FromSum(result.Stdout.FullSHA256), StderrSHA256: record.SHA256FromSum(result.Stderr.FullSHA256),
		IOTranscriptSHA256: record.SHA256FromSum(result.IOTranscript.SHA256), WorldTerminal: worldRecord.Terminal, WorldFinal: worldRecord.Final.SemanticDigest,
	}
	encoded, err := record.CanonicalJSON(projection)
	if err != nil {
		return "", err
	}
	return record.DomainHash("gomadv3-choice-frontier-outcome/v1", encoded), nil
}

func cloneSummary(summary Summary) Summary {
	summary.Artifacts = append([]string(nil), summary.Artifacts...)
	summary.SuccessArtifacts = append([]string(nil), summary.SuccessArtifacts...)
	summary.ChoiceTrace = cloneChoiceTraceSummary(summary.ChoiceTrace)
	summary.Frontier = cloneFrontierSummary(summary.Frontier)
	return summary
}

func cloneFailurePaths(values map[record.SHA256]string) map[record.SHA256]string {
	result := make(map[record.SHA256]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneStringSet(values map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for value := range values {
		result[value] = struct{}{}
	}
	return result
}

func syncFrontierDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
