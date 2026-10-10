package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/internal/preparation"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

type localCampaign struct {
	config               campaignRequest
	resuming             bool
	selection            SeedSelection
	baseEnvironment      []record.Environment
	readOnlyMounts       []readonlymount.Mapping
	prepared             target.Prepared
	resumePlan           campaign.CampaignPlan
	guidance             *guidanceCampaign
	overallCtx           context.Context
	runID                string
	batchPath            string
	journal              *campaign.CampaignJournal
	resumedRuns          []campaign.ExecutionRecord
	summary              CampaignResult
	batchComplete        bool
	selectedProfile      deterministicio.Spec
	executor             executionRunner
	activeCtx            context.Context
	activeCancel         context.CancelFunc
	rawCompletions       chan runCompletion
	completed            map[uint64]struct{}
	hostFailure          error
	distinct             map[record.SHA256]string
	failureArtifactBytes uint64
	semanticProbes       map[string]struct{}
	choiceFeatures       map[string]struct{}
	controller           *campaign.SeedController
}

func runLocal(ctx context.Context, config campaignRequest) (summary CampaignResult, retErr error) {
	local := localCampaign{config: config, resuming: config.ResumeCampaign != ""}
	err := local.validateRequest()
	defer func() {
		if local.guidance != nil {
			retErr = errors.Join(retErr, local.guidance.Close())
		}
	}()
	if err != nil {
		return CampaignResult{}, err
	}
	overallCtx, overallCancel := context.WithTimeout(ctx, local.config.OverallTimeout)
	defer overallCancel()
	local.overallCtx = overallCtx
	if err := local.openCampaign(); err != nil {
		return local.summary, err
	}
	defer func() {
		if closeErr := local.journal.Close(); closeErr != nil {
			retErr = errors.Join(retErr, &HostError{Reason: "runs_close", Err: closeErr})
		}
	}()
	if err := local.reportProgress(ProgressPreparing, 0); err != nil {
		return local.summary, &HostError{Reason: "progress_output", Err: err}
	}
	defer local.failCampaign(&retErr)
	if err := overallCtx.Err(); err != nil {
		return local.summary, &HostError{Reason: contextFailureReason(err), Err: err}
	}
	local.selectedProfile = deterministicio.Default()
	if err := local.prepareTarget(); err != nil {
		return local.summary, err
	}
	local.executor = local.config.executor
	if local.executor == nil {
		local.executor = processExecutor{}
	}
	if !local.resuming {
		err = local.journal.StartExecutions()
	}
	if err != nil {
		return local.summary, &HostError{Reason: "runs_create", Err: err}
	}
	if err := local.reportProgress(ProgressRunning, 0); err != nil {
		return local.summary, &HostError{Reason: "progress_output", Err: err}
	}
	if local.config.strategy() == StrategyChoiceExploration {
		err := runChoiceExplorationLocal(overallCtx, local.config, local.selection, local.baseEnvironment, local.readOnlyMounts, local.prepared, local.selectedProfile, local.journal, local.runID, local.resuming, local.resumedRuns, &local.summary, local.reportProgress)
		if err == nil {
			local.batchComplete = true
		}
		return local.summary, err
	}
	if local.config.strategy() == StrategySimulationExploration {
		err := runSimulationExplorationLocal(overallCtx, local.config, local.selection, local.baseEnvironment, local.readOnlyMounts, local.prepared, local.selectedProfile, local.journal, local.runID, local.resuming, local.resumedRuns, &local.summary, local.reportProgress)
		if err == nil {
			local.batchComplete = true
		}
		return local.summary, err
	}
	err = local.runSeeds()
	return local.summary, err
}

func (local *localCampaign) validateRequest() error {
	var err error
	local.selection, local.baseEnvironment, err = validateCampaignRequest(local.config)
	if err != nil {
		return err
	}
	if local.resuming {
		preflight := local.config.resumePreflight
		if preflight == nil {
			var opened campaign.ResumePreflight
			opened, err = campaign.PreflightResume(local.config.ResumeCampaign)
			preflight = &opened
		}
		if err == nil {
			local.resumePlan = preflight.Plan
			var resumed CampaignSpec
			resumed, local.selection, local.baseEnvironment, local.readOnlyMounts, local.prepared, err = resumeConfiguration(local.config, local.resumePlan)
			local.config = campaignRequestFromSpecWith(resumed, executionDependencies{executor: local.config.executor})
			local.config.resumePreflight = preflight
		}
		if err != nil {
			return err
		}
	} else {
		local.config.Artifacts, err = filepath.Abs(local.config.Artifacts)
		if err != nil {
			return &HostError{Reason: "artifact_setup", Err: fmt.Errorf("resolve artifact root: %w", err)}
		}
		local.readOnlyMounts, err = readonlymount.ParseMappings(local.config.IOROMounts, local.config.Target.WorkingDir)
		if err != nil {
			return err
		}
		if local.config.IOROMountLimits == (readonlymount.Limits{}) {
			local.config.IOROMountLimits = readonlymount.DefaultLimits()
		}
		if local.config.Guide {
			local.config.Corpus, err = guidedCorpusPath(local.config.Corpus)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (local *localCampaign) openCampaign() error {
	var err error
	if local.resuming {
		local.batchPath = local.config.ResumeCampaign
		local.runID = filepath.Base(local.batchPath)
		var resumeState campaign.ResumeState
		local.journal, resumeState, err = campaign.ResumeCampaignJournal(local.overallCtx, local.batchPath)
		if err == nil {
			var equal bool
			equal, err = equalBatchPlans(local.resumePlan, resumeState.Plan)
			if !equal && err == nil {
				err = errors.New("campaign plan changed while acquiring its resume lock")
			}
			local.resumedRuns = resumeState.Executions
		}
		if err != nil {
			local.summary = CampaignResult{CampaignPath: local.batchPath, SelectionCount: local.selection.Count()}
			return &HostError{Reason: "resume_setup", Err: err}
		}
		var restored resumeSummaryState
		restored, err = restoreResumeSummary(local.batchPath, local.selection, local.resumedRuns)
		local.summary = restored.summary
		local.summary.Guidance = guidanceSummary(local.config.guidancePlan, newGuidedExecutions(local.config.guidancePlan, local.resumedRuns))
		if local.config.guidancePlan != nil {
			local.summary.CorpusPath = local.config.guidancePlan.Corpus
		}
		local.summary.SelectionCount = local.config.shard().SelectionCount(local.selection.Count())
		if err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
	} else {
		local.runID, err = newRunID()
		if err != nil {
			return &HostError{Reason: "campaign_id", Err: err}
		}
		local.batchPath = filepath.Join(local.config.Artifacts, "v1", local.runID)
		local.summary = CampaignResult{CampaignPath: local.batchPath, SelectionCount: local.config.shard().SelectionCount(local.selection.Count())}
		local.journal, err = campaign.NewCampaignJournal(local.overallCtx, campaign.CampaignConfig{
			Root: local.config.Artifacts, CampaignID: local.runID, PlanSHA256: local.config.PlanSHA256, Shard: campaignStoreShard(local.config.Shard),
			Strategy: string(local.config.strategy()), Guidance: local.config.guidancePlan, Selection: local.config.Seeds, SelectionCount: local.selection.Count(), MaxExecutions: local.config.MaxExecutions, Parallel: uint64(local.config.Parallel),
		})
		if err != nil {
			return &HostError{Reason: "artifact_setup", Err: err}
		}
	}
	return nil
}

func (local *localCampaign) reportProgress(phase CampaignPhase, active int) error {
	if local.config.Progress == nil {
		return nil
	}
	return local.config.Progress(CampaignEvent{
		Phase: phase, CampaignPath: local.summary.CampaignPath, Selected: local.summary.SelectionCount, Attempted: local.summary.Attempted, Running: uint64(active),
		Succeeded: local.summary.Succeeded, Failures: local.summary.Failures, Watchdogs: local.summary.Watchdogs, ReplayDivergences: local.summary.ReplayDivergences, Cancelled: local.summary.Cancelled,
		DistinctFailures: local.summary.DistinctFailures, Artifacts: append([]string(nil), local.summary.Artifacts...),
		RetainedSuccesses: local.summary.RetainedSuccesses, RetainedSuccessBytes: local.summary.RetainedSuccessBytes, SuccessArtifacts: append([]string(nil), local.summary.SuccessArtifacts...),
		Guidance: cloneGuidanceSummary(local.summary.Guidance), CorpusPath: local.summary.CorpusPath, CorpusEntries: local.summary.CorpusEntries, CorpusAdded: local.summary.CorpusAdded,
		ChoiceTrace: cloneChoiceTraceSummary(local.summary.ChoiceTrace), ChoiceExploration: cloneChoiceExplorationSummary(local.summary.ChoiceExploration), SimulationExploration: cloneSimulationExplorationSummary(local.summary.SimulationExploration), RecoveryExecutions: local.summary.RecoveryExecutions,
	})
}

func (local *localCampaign) failCampaign(retErr *error) {
	if local.batchComplete || *retErr == nil {
		return
	}
	if errors.Is(local.overallCtx.Err(), context.DeadlineExceeded) {
		return
	}
	reason := "runner_failure"
	var hostError *HostError
	if errors.As(*retErr, &hostError) {
		reason = hostError.Reason
	}
	var missing *deterministicio.MissingSemanticProbesError
	if errors.As(*retErr, &missing) {
		reason = "semantic_coverage"
	}
	if partialErr := local.journal.Fail(reason, *retErr); partialErr != nil {
		*retErr = errors.Join(*retErr, partialErr)
	}
}

func (local *localCampaign) prepareTarget() error {
	var err error
	if !local.resuming {
		if err := local.journal.BeginPreparation(); err != nil {
			return &HostError{Reason: "target_preparation_setup", Err: err}
		}
		local.config.Target.PreparationRoot = local.journal.PreparedPath()
		local.prepared, err = preparation.Prepare(local.overallCtx, preparation.Request{
			Target: local.config.Target, Environment: local.config.Environment, Preparer: local.config.Preparer,
		})
		if err != nil {
			if preparation.StageOf(err) == preparation.StageTarget {
				reason := "target_preparation"
				if contextErr := local.overallCtx.Err(); contextErr != nil {
					reason = contextFailureReason(contextErr)
				}
				if partialErr := local.journal.FailPreparation(reason, err); partialErr != nil {
					err = errors.Join(err, partialErr)
				}
				return &HostError{Reason: reason, Err: err}
			}
			return err
		}
		if local.config.Guide && local.config.PlanSHA256 == "" {
			local.guidance, err = openGuidance(local.overallCtx, local.config, local.prepared, local.baseEnvironment, local.runID)
			if err != nil {
				return &HostError{Reason: "guided_corpus", Err: err}
			}
			snapshot := local.guidance.Snapshot()
			if local.config.guidancePlan == nil {
				local.selection, local.config.guidancePlan, err = selectGuidedSeeds(local.selection, snapshot, local.config.Corpus, local.config.GuideRegression)
			}
			if err != nil {
				return &HostError{Reason: "guided_selection", Err: err}
			}
			local.config.Seeds = local.selection.String()
			local.config.GuideSnapshotSHA256 = snapshot.SnapshotSHA256
			local.guidance.config = local.config
			local.summary.SelectionCount = local.config.shard().SelectionCount(local.selection.Count())
			local.summary.Guidance = guidanceSummary(local.config.guidancePlan, 0)
			local.summary.CorpusPath = local.guidance.corpus.Path()
			local.summary.CorpusEntries = uint64(len(snapshot.Entries))
			if err := local.journal.SetSelection(local.config.Seeds, local.selection.Count(), local.config.guidancePlan); err != nil {
				return &HostError{Reason: "guided_selection", Err: err}
			}
		}
		if local.config.guidancePlan != nil {
			local.summary.Guidance = guidanceSummary(local.config.guidancePlan, 0)
			local.summary.CorpusPath = local.config.guidancePlan.Corpus
		}
		plan, err := campaignPlan(local.config, local.journal, local.prepared, local.baseEnvironment, local.readOnlyMounts, local.selection.Count())
		if err != nil {
			return &HostError{Reason: "campaign_plan", Err: err}
		}
		if err := local.journal.RecordPlan(plan); err != nil {
			return &HostError{Reason: "campaign_plan", Err: err}
		}
		local.config.failureArtifactLimit = uint64(plan.Artifacts.FailureArtifacts)
		local.config.failureBytesLimit = uint64(plan.Artifacts.FailureBytes)
		if err := local.journal.CompletePreparation(); err != nil {
			return &HostError{Reason: "partial_cleanup", Err: err}
		}
	}
	if local.resuming && local.config.Guide && local.config.PlanSHA256 == "" {
		local.guidance, err = openGuidance(local.overallCtx, local.config, local.prepared, local.baseEnvironment, local.runID)
		if err != nil {
			return &HostError{Reason: "guided_corpus", Err: err}
		}
		snapshot := local.guidance.Snapshot()
		local.summary.CorpusPath = local.guidance.corpus.Path()
		local.summary.CorpusEntries = uint64(len(snapshot.Entries))
	}
	return nil
}

func (local *localCampaign) runSeeds() error {
	var err error
	local.activeCtx, local.activeCancel = context.WithCancel(local.overallCtx)
	defer local.activeCancel()
	local.rawCompletions = make(chan runCompletion, local.config.Parallel)
	completions := make(chan runCompletion, local.config.Parallel)
	local.completed = make(map[uint64]struct{})
	local.distinct = make(map[record.SHA256]string)
	local.failureArtifactBytes = uint64(0)
	local.semanticProbes = make(map[string]struct{})
	local.choiceFeatures = make(map[string]struct{})
	if local.resuming {
		var restored resumeSummaryState
		restored, err = restoreResumeSummary(local.batchPath, local.selection, local.resumedRuns)
		if err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
		local.summary = restored.summary
		local.summary.Guidance = guidanceSummary(local.config.guidancePlan, newGuidedExecutions(local.config.guidancePlan, local.resumedRuns))
		if local.config.guidancePlan != nil {
			local.summary.CorpusPath = local.config.guidancePlan.Corpus
		}
		local.summary.SelectionCount = local.config.shard().SelectionCount(local.selection.Count())
		local.distinct = restored.distinct
		local.failureArtifactBytes = restored.failureArtifactBytes
		local.semanticProbes = restored.probes
		local.choiceFeatures = restored.choiceFeatures
		local.completed = restored.completed
		if local.guidance != nil {
			snapshot := local.guidance.Snapshot()
			local.summary.CorpusPath = local.guidance.corpus.Path()
			local.summary.CorpusEntries = uint64(len(snapshot.Entries))
		}
	}
	completionOrderDone := make(chan struct{})
	go func() {
		defer close(completionOrderDone)
		orderShardRunCompletions(local.selection, local.config.Shard, local.completed, local.rawCompletions, completions)
	}()
	defer func() {
		close(local.rawCompletions)
		<-completionOrderDone
	}()
	local.controller, err = newShardedSeedController(local.selection, local.config.Shard, local.completed, local.config.Parallel, local.config.OnFailure, local.config.FailureBudget, local.summary)
	if err != nil {
		return &HostError{Reason: "campaign_setup", Err: err}
	}
	synchronizeCampaignStatistics(&local.summary, local.controller.Statistics())
	var progressTicker *time.Ticker
	var progressTicks <-chan time.Time
	if local.config.Progress != nil {
		interval := local.config.ProgressInterval
		if interval <= 0 {
			interval = 5 * time.Second
		}
		progressTicker = time.NewTicker(interval)
		progressTicks = progressTicker.C
		defer progressTicker.Stop()
	}
	local.scheduleSeeds(completions, progressTicks)
	return local.finishSeedCampaign()
}

func (local *localCampaign) recordCompletion(outcome campaign.Completion) bool {
	cancelActive := local.controller.Complete(outcome)
	synchronizeCampaignStatistics(&local.summary, local.controller.Statistics())
	return cancelActive
}

func (local *localCampaign) publishRunnerFailure(completion runCompletion, reason string) error {
	if !completion.result.Captured {
		return nil
	}
	if reason == "choice_trace_malformed" || reason == "choice_trace_unterminated" {
		return nil
	}
	worldBundle := noneWorldBundle()
	outcome := execution.Classification{
		Domain: "runner", Reason: reason, Termination: "none",
		ArtifactKind: record.ArtifactRunnerFailure, ReplayMode: record.ReplayNone,
	}
	mountArtifact, err := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
	if err != nil {
		return fmt.Errorf("construct read-only mount artifact: %w", err)
	}
	manifest, err := manifestForRun(local.config, local.prepared, local.baseEnvironment, completion, outcome, local.runID, worldBundle.Manifest, mountArtifact)
	if err != nil {
		return fmt.Errorf("construct Runner failure manifest: %w", err)
	}
	published, err := publishBoundedFailureArtifact(local.overallCtx, local.config, local.journal.FailuresPath(), manifest.Outcome.FailureSignature, local.distinct, &local.failureArtifactBytes, executionArtifactInput(manifest, local.prepared, completion.result, mountArtifact, worldBundle))
	if err != nil {
		return fmt.Errorf("publish Runner failure artifact: %w", err)
	}
	signature := published.Manifest.Outcome.FailureSignature
	if _, found := local.distinct[signature]; !found {
		local.distinct[signature] = published.Path
		local.summary.Artifacts = append(local.summary.Artifacts, published.Path)
	}
	local.summary.DistinctFailures = uint64(len(local.distinct))
	artifactRelative, err := filepath.Rel(local.batchPath, published.Path)
	if err != nil {
		return fmt.Errorf("make Runner failure artifact path relative: %w", err)
	}
	run := campaign.ExecutionRecord{
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: "runner", Reason: reason, Termination: "none", FailureSignature: &signature, Artifact: &artifactRelative,
		ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	if err := local.journal.AppendExecution(run); err != nil {
		return fmt.Errorf("append Runner failure result: %w", err)
	}
	return nil
}

func (local *localCampaign) completePartial(run *campaign.ExecutionJournal) {
	if cleanupErr := run.Complete(); cleanupErr != nil && local.hostFailure == nil {
		local.hostFailure = &HostError{Reason: "partial_cleanup", Err: cleanupErr}
		local.controller.Stop()
		local.activeCancel()
	}
}

func (local *localCampaign) launchSeed(job runJob) {
	readiness := newRunReadiness()
	go runSeed(local.activeCtx, local.config, local.executor, local.prepared, local.baseEnvironment, local.selectedProfile, local.readOnlyMounts, local.journal, job, readiness, local.rawCompletions)
	readiness.wait()
}

func (local *localCampaign) scheduleSeeds(completions <-chan runCompletion, progressTicks <-chan time.Time) {
	runningReported := false
	for !local.controller.Done() {
		admittedThisTurn := false
		for local.overallCtx.Err() == nil {
			scheduled, ok := local.controller.Next()
			if !ok {
				break
			}
			job := runJob{ordinal: scheduled.Ordinal, seed: scheduled.Seed}
			if local.summary.Guidance != nil && !answeredSeed(local.config.guidancePlan, job.seed) {
				local.summary.Guidance.NewExecutions++
			}
			local.launchSeed(job)
			admittedThisTurn = true
		}
		if local.controller.Active() > 0 && !runningReported {
			runningReported = true
			if err := local.reportProgress(ProgressRunning, local.controller.Active()); err != nil {
				local.hostFailure = &HostError{Reason: "progress_output", Err: err}
				local.controller.Stop()
				local.activeCancel()
			}
		}
		if local.overallCtx.Err() != nil && local.hostFailure == nil && !local.controller.Stopped() {
			local.hostFailure = &HostError{Reason: contextFailureReason(local.overallCtx.Err()), Err: local.overallCtx.Err()}
			local.controller.Stop()
			local.activeCancel()
		}
		if local.controller.Active() == 0 {
			break
		}
		var completion runCompletion
		select {
		case completion = <-completions:
		case <-progressTicks:
			if admittedThisTurn {
				continue
			}
			if err := local.reportProgress(ProgressRunning, local.controller.Active()); err != nil {
				local.hostFailure = &HostError{Reason: "progress_output", Err: err}
				local.controller.Stop()
				local.activeCancel()
			}
			continue
		}
		local.handleCompletion(completion)
	}
}

func (local *localCampaign) handleCompletion(completion runCompletion) {
	if local.overallCtx.Err() != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: contextFailureReason(local.overallCtx.Err()), Err: local.overallCtx.Err()}
		}
		local.controller.Stop()
		local.activeCancel()
		return
	}
	if completion.err != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		reason := supervisionFailureReason(completion.err)
		if completion.result.ChoiceTrace.Profile != "" && completion.result.ChoiceTrace.Trace.Summary.Terminal == choice.TerminalOverflow {
			local.summary.ChoiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
		}
		if partialErr := preservePartial(completion.journal); partialErr != nil {
			completion.err = errors.Join(completion.err, partialErr)
		}
		if publishErr := local.publishRunnerFailure(completion, reason); publishErr != nil {
			completion.err = errors.Join(completion.err, publishErr)
		}
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: reason, Err: completion.err}
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	if err := local.prepared.Verify(); err != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: "prepared_target_integrity", Err: err}
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	if local.config.ChoiceTraceLimit != 0 {
		local.summary.ChoiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
	}
	if completion.result.Cancelled && local.controller.Stopped() {
		local.recordCompletion(campaign.CompletedCancelled())
		if partialErr := preservePartial(completion.journal); partialErr != nil {
			local.hostFailure = errors.Join(local.hostFailure, &HostError{Reason: "partial_write", Err: partialErr})
		}
		if local.hostFailure != nil {
			reason := "runner_failure"
			var hostError *HostError
			if errors.As(local.hostFailure, &hostError) {
				reason = hostError.Reason
			}
			if publishErr := local.publishRunnerFailure(completion, reason); publishErr != nil {
				local.hostFailure = errors.Join(local.hostFailure, publishErr)
			}
			return
		}
		run := campaign.ExecutionRecord{
			SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
			Domain: "runner", Reason: "runner_cancelled", Termination: "none", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
		}
		if err := local.journal.AppendExecution(run); err != nil && local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: "runs_append", Err: err}
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	worldBundle, err := assessWorld(completion.result, completion.job.seed, local.config.WorldTransitionLimit)
	if err != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if publishErr := local.publishRunnerFailure(completion, "world_record"); publishErr != nil {
			err = errors.Join(err, publishErr)
		}
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: "world_record", Err: err}
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	assessed, assessErr := assessCompletion(completion.result, worldBundle.Manifest.Terminal, local.config.Coverage, local.prepared)
	if assessErr != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if partialErr := preservePartial(completion.journal); partialErr != nil {
			assessErr.Err = errors.Join(assessErr.Err, partialErr)
		}
		if local.hostFailure == nil {
			local.hostFailure = assessErr
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	local.recordAssessedCompletion(completion, worldBundle, assessed)
}

func (local *localCampaign) recordAssessedCompletion(completion runCompletion, worldBundle execution.Bundle, assessed completedExecution) {
	runCoverage, runChoiceProjection, outcome := assessed.coverage, assessed.choiceProjection, assessed.outcome
	if local.config.Diagnostics && len(completion.result.DiagnosticTrace.Bytes) != 0 {
		trace, err := retainDiagnosticTrace(local.journal.Path(), completion.job.ordinal, completion.result.DiagnosticTrace)
		if err != nil {
			local.recordCompletion(campaign.CompletedUnclassified())
			local.hostFailure = &HostError{Reason: "diagnostic_write", Err: err}
			local.controller.Stop()
			local.activeCancel()
			return
		}
		local.summary.Diagnostics = trace
	}
	if local.config.CollectExecutionEvidence {
		mountArtifact, evidenceErr := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
		if evidenceErr != nil {
			local.recordCompletion(campaign.CompletedUnclassified())
			if local.hostFailure == nil {
				local.hostFailure = &HostError{Reason: "execution_evidence", Err: evidenceErr}
				local.controller.Stop()
				local.activeCancel()
			}
			return
		}
		runRecord := executionEvidence(local.config, local.prepared, local.baseEnvironment, completion, outcome, worldBundle.Manifest, mountArtifact, runCoverage, runChoiceProjection)
		local.summary.ExecutionEvidence = &runRecord
		local.summary.ExecutionElapsedNanos = uint64(elapsedNanos(completion.startedAt, completion.finishedAt))
	}
	if err := completion.journal.Transition(campaign.ExecutionClassified); err != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: "partial_write", Err: err}
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	if overallErr := local.overallCtx.Err(); overallErr != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: contextFailureReason(overallErr), Err: overallErr}
			local.controller.Stop()
			local.activeCancel()
		}
		if partialErr := preservePartial(completion.journal); partialErr != nil {
			local.hostFailure = errors.Join(local.hostFailure, &HostError{Reason: "partial_write", Err: partialErr})
		}
		return
	}
	if outcome.Domain == "success" {
		local.recordSuccessfulExecution(completion, worldBundle, assessed)
		return
	}
	local.recordFailedExecution(completion, worldBundle, assessed)
}

func (local *localCampaign) recordSuccessfulExecution(completion runCompletion, worldBundle execution.Bundle, assessed completedExecution) {
	runCoverage, runChoiceFeatures, outcome := assessed.coverage, assessed.choiceFeatures, assessed.outcome
	run := campaign.ExecutionRecord{
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: "success", Reason: outcome.Reason, Termination: "exit", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	run.SemanticProbes = append([]string(nil), runCoverage.Probes...)
	run.ChoiceFeatures = append([]string(nil), runChoiceFeatures...)
	retention, retentionErr := decideSuccessRetention(local.config, assessed, completion.result.IOTranscript.Complete, local.semanticProbes, local.choiceFeatures, local.summary.RetainedSuccesses, local.summary.RetainedSuccessBytes)
	if retentionErr != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		local.hostFailure = retentionErr
		local.controller.Stop()
		local.activeCancel()
		return
	}
	if retention.retain {
		mountArtifact, publishErr := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
		if publishErr == nil {
			var manifest record.ExecutionRecord
			manifest, publishErr = manifestForRun(local.config, local.prepared, local.baseEnvironment, completion, outcome, local.runID, worldBundle.Manifest, mountArtifact)
			if publishErr == nil {
				var published artifact.Artifact
				published, publishErr = artifact.PublishArtifact(artifact.Store{Root: local.journal.SuccessesPath(), Context: local.overallCtx, MaximumBytes: retention.maximumBytes, Key: artifact.StoreKeyExecution, TargetPool: artifact.TargetPool(local.config.Artifacts)}, executionArtifactInput(manifest, local.prepared, completion.result, mountArtifact, worldBundle))
				if publishErr == nil {
					relative, relErr := filepath.Rel(local.batchPath, published.Path)
					if relErr != nil {
						publishErr = relErr
					} else {
						retention.annotate(&run, relative, published.StoredBytes)
						local.summary.SuccessArtifacts = append(local.summary.SuccessArtifacts, published.Path)
						local.summary.RetainedSuccesses++
						local.summary.RetainedSuccessBytes += published.StoredBytes
					}
				}
			}
		}
		if publishErr != nil {
			local.recordCompletion(campaign.CompletedUnclassified())
			local.hostFailure = successPublicationFailure(publishErr)
			local.controller.Stop()
			local.activeCancel()
			return
		}
	}
	if local.guidance != nil {
		mountArtifact, guideErr := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
		var added bool
		if guideErr == nil {
			added, guideErr = local.guidance.MergeRun(local.overallCtx, completion, outcome, worldBundle, mountArtifact, runCoverage)
		}
		if guideErr != nil {
			local.recordCompletion(campaign.CompletedUnclassified())
			local.hostFailure = &HostError{Reason: "guided_corpus", Err: guideErr}
			local.controller.Stop()
			local.activeCancel()
			return
		}
		if added {
			local.summary.CorpusAdded++
			local.summary.CorpusEntries = uint64(len(local.guidance.Snapshot().Entries))
		}
	}
	if err := local.journal.AppendExecution(run); err != nil && local.hostFailure == nil {
		local.hostFailure = &HostError{Reason: "runs_append", Err: err}
		local.controller.Stop()
		local.activeCancel()
	}
	if local.hostFailure == nil {
		local.recordCompletion(campaign.CompletedSuccess())
		addStrings(local.semanticProbes, runCoverage.Probes)
		addStrings(local.choiceFeatures, runChoiceFeatures)
	} else {
		local.recordCompletion(campaign.CompletedUnclassified())
	}
	local.completePartial(completion.journal)
}

func (local *localCampaign) recordFailedExecution(completion runCompletion, worldBundle execution.Bundle, assessed completedExecution) {
	runCoverage, runChoiceFeatures, outcome := assessed.coverage, assessed.choiceFeatures, assessed.outcome
	mountArtifact, manifestErr := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
	if manifestErr != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: "manifest", Err: manifestErr}
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	manifest, manifestErr := manifestForRun(local.config, local.prepared, local.baseEnvironment, completion, outcome, local.runID, worldBundle.Manifest, mountArtifact)
	if manifestErr != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: "manifest", Err: manifestErr}
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	published, publishErr := publishBoundedFailureArtifact(local.overallCtx, local.config, local.journal.FailuresPath(), manifest.Outcome.FailureSignature, local.distinct, &local.failureArtifactBytes, executionArtifactInput(manifest, local.prepared, completion.result, mountArtifact, worldBundle))
	if publishErr != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: "artifact_publication", Err: publishErr}
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	if overallErr := local.overallCtx.Err(); overallErr != nil {
		local.recordCompletion(campaign.CompletedUnclassified())
		if local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: contextFailureReason(overallErr), Err: overallErr}
			local.controller.Stop()
			local.activeCancel()
		}
		return
	}
	if local.guidance != nil {
		added, guideErr := local.guidance.MergeRun(local.overallCtx, completion, outcome, worldBundle, mountArtifact, runCoverage)
		if guideErr != nil {
			local.recordCompletion(campaign.CompletedUnclassified())
			local.hostFailure = &HostError{Reason: "guided_corpus", Err: guideErr}
			local.controller.Stop()
			local.activeCancel()
			return
		}
		if added {
			local.summary.CorpusAdded++
			local.summary.CorpusEntries = uint64(len(local.guidance.Snapshot().Entries))
		}
	}
	signature := published.Manifest.Outcome.FailureSignature
	if _, found := local.distinct[signature]; !found {
		local.distinct[signature] = published.Path
		local.summary.Artifacts = append(local.summary.Artifacts, published.Path)
	}
	cancelActive := local.recordCompletion(campaign.CompletedFailure(outcome.Domain, outcome.Reason, uint64(len(local.distinct))))
	artifactRelative, relErr := filepath.Rel(local.batchPath, published.Path)
	if relErr != nil {
		local.hostFailure = &HostError{Reason: "artifact_path", Err: relErr}
		local.controller.Stop()
		local.activeCancel()
		return
	}
	run := campaign.ExecutionRecord{
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, FailureSignature: &signature,
		Artifact: &artifactRelative, ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	run.SemanticProbes = append([]string(nil), runCoverage.Probes...)
	run.ChoiceFeatures = append([]string(nil), runChoiceFeatures...)
	if err := local.journal.AppendExecution(run); err != nil && local.hostFailure == nil {
		local.hostFailure = &HostError{Reason: "runs_append", Err: err}
		local.controller.Stop()
		local.activeCancel()
	}
	if local.hostFailure == nil {
		addStrings(local.semanticProbes, runCoverage.Probes)
		addStrings(local.choiceFeatures, runChoiceFeatures)
	}
	local.completePartial(completion.journal)

	if cancelActive {
		local.activeCancel()
	}
}

func (local *localCampaign) finishSeedCampaign() error {
	if coverageHasSemantic(local.config.Coverage) {
		probes := make([]string, 0, len(local.semanticProbes))
		for probe := range local.semanticProbes {
			probes = append(probes, probe)
		}
		coverage, coverageErr := deterministicio.SummarizeSemanticProbes(probes)
		if coverageErr != nil && local.hostFailure == nil {
			local.hostFailure = &HostError{Reason: "semantic_coverage", Err: coverageErr}
		} else if coverageErr == nil {
			local.summary.SemanticCoverage = &coverage
		}
	}
	if local.overallCtx.Err() != nil && local.hostFailure == nil {
		local.hostFailure = &HostError{Reason: contextFailureReason(local.overallCtx.Err()), Err: local.overallCtx.Err()}
	}
	if local.hostFailure != nil {
		if local.summary.ChoiceTrace != nil {
			if progressErr := local.reportProgress(ProgressRunning, 0); progressErr != nil {
				local.hostFailure = errors.Join(local.hostFailure, &HostError{Reason: "progress_output", Err: progressErr})
			}
		}
		return local.hostFailure
	}
	if local.summary.SemanticCoverage != nil && local.selection.Count() != 0 {
		missing, err := deterministicio.MissingRequiredSemanticProbes(*local.summary.SemanticCoverage, local.config.RequiredSemanticProbes)
		if err != nil {
			return err
		}
		if len(missing) != 0 {
			return &deterministicio.MissingSemanticProbesError{Probes: missing}
		}
	}
	if err := local.prepared.Verify(); err != nil {
		return &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	if err := local.overallCtx.Err(); err != nil {
		return &HostError{Reason: contextFailureReason(err), Err: err}
	}
	local.controller.Finalize()
	synchronizeCampaignStatistics(&local.summary, local.controller.Statistics())
	if err := local.overallCtx.Err(); err != nil {
		return &HostError{Reason: contextFailureReason(err), Err: err}
	}
	failureSignatures := make([]record.SHA256, 0, len(local.distinct))
	for signature := range local.distinct {
		failureSignatures = append(failureSignatures, signature)
	}
	if err := local.journal.Publish(campaign.CampaignSummary{
		Attempted: local.summary.Attempted, Succeeded: local.summary.Succeeded, Failures: local.summary.Failures, Watchdogs: local.summary.Watchdogs,
		Cancelled: local.summary.Cancelled, DistinctFailures: local.summary.DistinctFailures, RetainedSuccesses: local.summary.RetainedSuccesses, RetainedSuccessBytes: local.summary.RetainedSuccessBytes, StopReason: string(local.summary.StopReason), FailureSignatures: failureSignatures,
	}); err != nil {
		return &HostError{Reason: "campaign_publish", Err: err}
	}
	local.batchComplete = true
	if err := local.reportProgress(ProgressComplete, 0); err != nil {
		return &HostError{Reason: "progress_output", Err: err}
	}
	return nil
}
