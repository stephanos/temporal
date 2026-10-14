package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
)

type localCampaign struct {
	config          campaignRequest
	resuming        bool
	selection       SeedSelection
	baseEnvironment []record.Environment
	readOnlyMounts  []captured.Mapping
	prepared        target.Prepared
	resumePlan      campaign.CampaignPlan
	guidance        *guidanceCampaign
	overallCtx      context.Context
	runID           string
	batchPath       string
	journal         *campaign.CampaignJournal
	resumedRuns     []campaign.ExecutionRecord
	summary         CampaignResult
	batchComplete   bool
	selectedProfile backend.Description
	executor        executionRunner
	activeCtx       context.Context
	activeCancel    context.CancelFunc
	rawCompletions  chan runCompletion
	launched        int
	attempts        map[uint64]*seedAttempt
	jobs            *pendingJobs
	group           *stopGroup
	stopped         bool
	statistics      campaign.CampaignStatistics
	hostFailure     error
	// distinct holds the committed failure signatures; published holds every
	// failure artifact the campaign published, committed or still pending.
	distinct             map[record.SHA256]string
	published            map[record.SHA256]string
	failureArtifactBytes uint64
	semanticProbes       map[string]struct{}
	choiceFeatures       map[string]struct{}
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
	local.selectedProfile = local.config.profile
	if err := local.prepareTarget(); err != nil {
		return local.summary, err
	}
	local.executor = local.config.executor
	if local.executor == nil {
		local.executor = processExecutor{provider: local.config.Backend}
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
		err := runChoiceExplorationLocal(local.explorationCampaign())
		if err == nil {
			local.batchComplete = true
		}
		return local.summary, err
	}
	if local.config.strategy() == StrategySimulationExploration {
		err := runSimulationExplorationLocal(local.explorationCampaign())
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
			local.config = campaignRequestFromSpecWith(resumed, local.config.executionDependencies)
			if err == nil {
				local.config.profile, err = local.config.Backend.Describe(context.Background(), backend.Configuration{Target: local.config.Target, ClockTick: local.config.ClockTick, Environment: local.config.Environment, RequiredSemanticProbes: local.config.RequiredSemanticProbes})
			}
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
		local.readOnlyMounts, err = captured.ParseMappings(local.config.IOROMounts, local.config.Target.WorkingDir)
		if err != nil {
			return err
		}
		if local.config.IOROMountLimits == (captured.Limits{}) {
			local.config.IOROMountLimits = captured.DefaultLimits()
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
	var missing *record.MissingSemanticProbesError
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
		local.prepared, err = local.config.prepareTarget(local.overallCtx, preparationRequest{
			Target: local.config.Target, Environment: recordedEnvironmentStrings(local.baseEnvironment), Preparer: local.config.Preparer, Provider: local.config.Backend,
		})
		if err != nil {
			var failure *backend.PreparationError
			if errors.As(err, &failure) && failure.Stage == "target" {
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

// publishRunnerFailure publishes a captured attempt that failed the host as a
// Runner failure artifact and returns the evidence record that references
// it; an attempt that captured nothing, or only a choice trace supervision
// rejected, has none.
func (local *localCampaign) publishRunnerFailure(completion runCompletion, reason string) (*campaign.ExecutionRecord, error) {
	if !completion.result.Captured {
		return nil, nil
	}
	if reason == "choice_trace_malformed" || reason == "choice_trace_unterminated" {
		return nil, nil
	}
	worldBundle := noneWorldBundle()
	outcome := backend.Classification{
		Domain: "runner", Reason: reason, Termination: "none",
		ArtifactKind: record.ArtifactRunnerFailure, ReplayMode: record.ReplayNone,
	}
	mountArtifact, err := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
	if err != nil {
		return nil, fmt.Errorf("construct read-only mount artifact: %w", err)
	}
	manifest, err := manifestForRun(local.config, local.prepared, local.baseEnvironment, completion, outcome, local.runID, worldBundle.Manifest, mountArtifact)
	if err != nil {
		return nil, fmt.Errorf("construct Runner failure manifest: %w", err)
	}
	published, err := publishBoundedFailureArtifact(local.overallCtx, local.config, local.journal.FailuresPath(), manifest.Outcome.FailureSignature, local.published, &local.failureArtifactBytes, executionArtifactInput(manifest, local.prepared, completion.result, mountArtifact, worldBundle))
	if err != nil {
		return nil, fmt.Errorf("publish Runner failure artifact: %w", err)
	}
	signature := published.Manifest.Outcome.FailureSignature
	if _, found := local.published[signature]; !found {
		local.published[signature] = published.Path
	}
	if _, found := local.distinct[signature]; !found {
		local.distinct[signature] = published.Path
		local.summary.Artifacts = append(local.summary.Artifacts, published.Path)
	}
	local.summary.DistinctFailures = uint64(len(local.distinct))
	artifactRelative, err := filepath.Rel(local.batchPath, published.Path)
	if err != nil {
		return nil, fmt.Errorf("make Runner failure artifact path relative: %w", err)
	}
	run := campaign.ExecutionRecord{
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: "runner", Reason: reason, Termination: "none", FailureSignature: &signature, Artifact: &artifactRelative,
		ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	return &run, nil
}

// arrive observes one attempt's completion as soon as it arrives: it
// classifies the attempt, publishes its evidence and records its terminal
// receipt, or keeps the host failure it observed for the attempt's turn.
func (local *localCampaign) arrive(completion runCompletion) {
	attempt := local.attempts[completion.job.ordinal]
	attempt.arrived = true
	attempt.completion = completion
	attempt.failure = local.observe(attempt, completion)
}

// observe classifies an arrived completion in the order the Runner has always
// checked it. It returns a host failure, or leaves the attempt's observation
// or campaign-requested cancellation on the attempt.
func (local *localCampaign) observe(attempt *seedAttempt, completion runCompletion) *seedHostFailure {
	if err := local.overallCtx.Err(); err != nil {
		return &seedHostFailure{reason: contextFailureReason(err), cause: err}
	}
	if completion.err != nil {
		if completion.result.ChoiceTrace.Profile != "" && completion.result.ChoiceTrace.Trace.Summary.Terminal == choice.TerminalOverflow {
			attempt.choiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
		}
		return &seedHostFailure{reason: supervisionFailureReason(completion.err), cause: completion.err, evidence: true}
	}
	if err := local.config.Backend.ValidatePrepared(local.config.Target, local.prepared, recordedEnvironmentStrings(local.baseEnvironment)); err != nil {
		return &seedHostFailure{reason: "prepared_target_integrity", cause: err}
	}
	if local.config.ChoiceTraceLimit != 0 {
		attempt.choiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
	}
	if completion.result.Cancelled && local.cancellationRequested() {
		return local.observeCancellation(attempt, completion)
	}
	worldBundle, err := assessWorld(completion.result, completion.job.seed, local.config.WorldTransitionLimit)
	if err != nil {
		return &seedHostFailure{reason: "world_record", cause: err, evidence: true}
	}
	assessed, assessErr := assessCompletion(local.overallCtx, local.config.Backend, completion.result, worldBundle.Manifest.Terminal, local.config.Coverage, local.prepared)
	if assessErr != nil {
		return &seedHostFailure{reason: assessErr.Reason, cause: assessErr.Err}
	}
	observation := &seedObservation{}
	if local.config.Diagnostics && len(completion.result.DiagnosticTrace.Bytes) != 0 {
		trace, file, err := retainDiagnosticTrace(local.overallCtx, local.journal.Path(), completion.job.ordinal, completion.result.DiagnosticTrace)
		if err != nil {
			return &seedHostFailure{reason: "diagnostic_write", cause: err}
		}
		observation.diagnostic, observation.diagnostics = trace, []artifact.FileReference{file}
	}
	if local.config.CollectExecutionEvidence {
		mountArtifact, err := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
		if err != nil {
			return &seedHostFailure{reason: "execution_evidence", cause: err}
		}
		evidence := executionEvidence(local.config, local.prepared, local.baseEnvironment, completion, assessed.outcome, worldBundle.Manifest, mountArtifact, assessed.coverage, assessed.choiceProjection)
		observation.evidence = &evidence
		observation.evidenceElapsed = uint64(elapsedNanos(completion.startedAt, completion.finishedAt))
	}
	if err := completion.journal.Transition(campaign.ExecutionClassified); err != nil {
		return &seedHostFailure{reason: "partial_write", cause: err}
	}
	if err := local.overallCtx.Err(); err != nil {
		return &seedHostFailure{reason: contextFailureReason(err), cause: err}
	}
	var failure *seedHostFailure
	if assessed.outcome.Domain == "success" {
		failure = local.observeSuccess(observation, completion, worldBundle, assessed)
	} else {
		failure = local.observeFailure(observation, completion, worldBundle, assessed)
	}
	if failure != nil {
		return failure
	}
	attempt.observation = observation
	if assertions := assessed.outcome.SoftAssertions; assertions != nil && assertions.ReportOnly {
		observation.run.SoftAssertions = campaign.CloneSoftAssertionReport(assertions)
	}
	if observation.successErr == nil {
		if err := local.recordReceipt(observation); err != nil {
			return &seedHostFailure{reason: "runs_append", cause: err}
		}
		attempt.received = true
	}
	attempt.cleanupErr = completion.journal.Complete()
	return nil
}

// cancellationRequested reports whether the campaign has cancelled its active
// attempts, for a pending first-failure stop or a host failure.
func (local *localCampaign) cancellationRequested() bool {
	return local.activeCtx.Err() != nil
}

// observeCancellation records the actual cancellation a pending first-failure
// stop requested as the attempt's receipt. A cancellation a host failure
// requested is that failure's evidence instead.
func (local *localCampaign) observeCancellation(attempt *seedAttempt, completion runCompletion) *seedHostFailure {
	if local.hostFailure != nil || local.journal.Frontier().PendingStop != campaign.StopFirstFailure {
		attempt.cancelled = true
		return nil
	}
	observation := &seedObservation{run: campaign.ExecutionRecord{
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: "runner", Reason: "runner_cancelled", Termination: "none", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}}
	if err := local.recordReceipt(observation); err != nil {
		return &seedHostFailure{reason: "runs_append", cause: err}
	}
	attempt.observation, attempt.received = observation, true
	return nil
}

// observeSuccess publishes a success the ordered outcome may retain before
// its receipt. It is published when the committed state would retain it;
// the ordered outcome judges against at least that state, so it retains at
// most what was published. A retention error is kept for that judgement.
func (local *localCampaign) observeSuccess(observation *seedObservation, completion runCompletion, worldBundle backend.Bundle, assessed completedExecution) *seedHostFailure {
	outcome := assessed.outcome
	run := campaign.ExecutionRecord{
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: "success", Reason: outcome.Reason, Termination: "exit", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	run.SemanticProbes = append([]string(nil), assessed.coverage.Probes...)
	run.ChoiceFeatures = append([]string(nil), assessed.choiceFeatures...)
	observation.run = run
	observation.transcriptComplete = completion.result.IOTranscript.Complete || len(completion.result.BackendEvidence) != 0
	retention, retentionErr := decideSuccessRetention(local.config, assessed, observation.transcriptComplete, local.semanticProbes, local.choiceFeatures, local.summary.RetainedSuccesses, local.summary.RetainedSuccessBytes)
	if retentionErr != nil {
		observation.successErr = retentionErr
	} else if retention.retain {
		published, err := local.publishSuccess(completion, worldBundle, outcome, retention.maximumBytes)
		if err != nil {
			observation.successErr = successPublicationFailure(err)
		}
		observation.success = published
	}
	if local.guidance != nil {
		mountArtifact, err := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
		if err == nil {
			observation.corpusAdded, err = local.guidance.MergeRun(local.overallCtx, completion, outcome, worldBundle, mountArtifact, assessed.coverage)
		}
		if err != nil {
			return &seedHostFailure{reason: "guided_corpus", cause: err}
		}
	}
	return nil
}

func (local *localCampaign) publishSuccess(completion runCompletion, worldBundle backend.Bundle, outcome backend.Classification, maximumBytes uint64) (*publishedSuccess, error) {
	mountArtifact, err := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
	if err != nil {
		return nil, err
	}
	manifest, err := manifestForRun(local.config, local.prepared, local.baseEnvironment, completion, outcome, local.runID, worldBundle.Manifest, mountArtifact)
	if err != nil {
		return nil, err
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: local.journal.SuccessesPath(), Context: local.overallCtx, MaximumBytes: maximumBytes, Key: artifact.StoreKeyExecution, TargetPool: artifact.TargetPool(local.config.Artifacts)}, executionArtifactInput(manifest, local.prepared, completion.result, mountArtifact, worldBundle))
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(local.batchPath, published.Path)
	if err != nil {
		return nil, err
	}
	return &publishedSuccess{relative: filepath.ToSlash(relative), path: published.Path, storedBytes: published.StoredBytes}, nil
}

// observeFailure publishes a classified failure's artifact before its
// receipt. A cancellation the campaign did not request fails the host.
func (local *localCampaign) observeFailure(observation *seedObservation, completion runCompletion, worldBundle backend.Bundle, assessed completedExecution) *seedHostFailure {
	outcome := assessed.outcome
	if outcome.Domain == "runner" {
		return &seedHostFailure{reason: "runner_cancelled", cause: errors.New("seed execution was cancelled")}
	}
	mountArtifact, err := mountArtifactForRun(local.readOnlyMounts, local.config.IOROMountLimits, completion.result.IOROMounts)
	if err != nil {
		return &seedHostFailure{reason: "manifest", cause: err}
	}
	manifest, err := manifestForRun(local.config, local.prepared, local.baseEnvironment, completion, outcome, local.runID, worldBundle.Manifest, mountArtifact)
	if err != nil {
		return &seedHostFailure{reason: "manifest", cause: err}
	}
	published, err := publishBoundedFailureArtifact(local.overallCtx, local.config, local.journal.FailuresPath(), manifest.Outcome.FailureSignature, local.published, &local.failureArtifactBytes, executionArtifactInput(manifest, local.prepared, completion.result, mountArtifact, worldBundle))
	if err != nil {
		return &seedHostFailure{reason: "artifact_publication", cause: err}
	}
	signature := published.Manifest.Outcome.FailureSignature
	if _, found := local.published[signature]; !found {
		local.published[signature] = published.Path
	}
	if err := local.overallCtx.Err(); err != nil {
		return &seedHostFailure{reason: contextFailureReason(err), cause: err}
	}
	if local.guidance != nil {
		observation.corpusAdded, err = local.guidance.MergeRun(local.overallCtx, completion, outcome, worldBundle, mountArtifact, assessed.coverage)
		if err != nil {
			return &seedHostFailure{reason: "guided_corpus", cause: err}
		}
	}
	artifactRelative, err := filepath.Rel(local.batchPath, published.Path)
	if err != nil {
		return &seedHostFailure{reason: "artifact_path", cause: err}
	}
	artifactRelative = filepath.ToSlash(artifactRelative)
	run := campaign.ExecutionRecord{
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, FailureSignature: &signature,
		Artifact: &artifactRelative, ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	run.SemanticProbes = append([]string(nil), assessed.coverage.Probes...)
	run.ChoiceFeatures = append([]string(nil), assessed.choiceFeatures...)
	observation.run, observation.failurePath = run, published.Path
	return nil
}

func (local *localCampaign) finishSeedCampaign() error {
	if coverageHasSemantic(local.config.Coverage) {
		probes := make([]string, 0, len(local.semanticProbes))
		for probe := range local.semanticProbes {
			probes = append(probes, probe)
		}
		evidence, coverageErr := local.config.Backend.Evidence(context.WithoutCancel(local.overallCtx), backend.EvidenceRequest{Probes: probes})
		coverage := evidence.Coverage
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
		evidence, err := local.config.Backend.Evidence(context.WithoutCancel(local.overallCtx), backend.EvidenceRequest{Probes: local.summary.SemanticCoverage.Probes, RequiredSemanticProbes: local.config.RequiredSemanticProbes})
		missing := evidence.MissingRequiredProbes
		if err != nil {
			return err
		}
		if len(missing) != 0 {
			return &record.MissingSemanticProbesError{Probes: missing}
		}
	}
	if err := local.config.Backend.ValidatePrepared(local.config.Target, local.prepared, recordedEnvironmentStrings(local.baseEnvironment)); err != nil {
		return &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	if err := local.overallCtx.Err(); err != nil {
		return &HostError{Reason: contextFailureReason(err), Err: err}
	}
	local.statistics.Finalize()
	synchronizeCampaignStatistics(&local.summary, local.statistics)
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
