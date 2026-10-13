package runner

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"

	backend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	captured "go.temporal.io/server/tools/gomad3/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/campaign"
)

// Choice and simulation exploration run through one round driver. Each round
// is the engine's next round: its candidates run together, their outcomes are
// classified and their evidence published, and the round, the engine's segment
// and every candidate's execution commit as one campaign log transaction
// through the campaign's round owner. Counters, retention and novelty advance
// only once the round is durable; a round that fails runs again in full on
// resume.

// explorationCampaign is the context every exploration round runs in.
type explorationCampaign struct {
	ctx             context.Context
	config          campaignRequest
	selection       SeedSelection
	baseEnvironment []record.Environment
	readOnlyMounts  []captured.Mapping
	prepared        target.Prepared
	profile         backend.Description
	journal         *campaign.CampaignJournal
	runID           string
	resuming        bool
	resumedRuns     []campaign.ExecutionRecord
	summary         *CampaignResult
	reportProgress  func(CampaignPhase, int) error
	executor        executionRunner
}

func (local *localCampaign) explorationCampaign() *explorationCampaign {
	return &explorationCampaign{
		ctx: local.overallCtx, config: local.config, selection: local.selection, baseEnvironment: local.baseEnvironment,
		readOnlyMounts: local.readOnlyMounts, prepared: local.prepared, profile: local.selectedProfile, journal: local.journal,
		runID: local.runID, resuming: local.resuming, resumedRuns: local.resumedRuns, summary: &local.summary,
		reportProgress: local.reportProgress, executor: local.executor,
	}
}

// explorationTally is what a round's outcomes advance. A round works on a
// copy that replaces the campaign's only once the round is committed.
type explorationTally struct {
	summary  CampaignResult
	distinct map[record.SHA256]string
	probes   map[string]struct{}
	choices  map[string]struct{}
}

func (tally explorationTally) clone() explorationTally {
	return explorationTally{summary: cloneSummary(tally.summary), distinct: maps.Clone(tally.distinct), probes: maps.Clone(tally.probes), choices: maps.Clone(tally.choices)}
}

// explorationStrategy binds one exploration strategy to the round driver: its
// engine and frozen configuration, how its candidates run, how a completion
// becomes its engine result and execution record, and how its summary is
// reported and published.
type explorationStrategy[C, S, R, G, X any] struct {
	// reason prefixes the strategy's failure reasons; subject names it in
	// diagnostics.
	reason       string
	subject      string
	engine       *campaign.RoundEngine[C, S, R, G]
	config       C
	segmentBytes uint64
	jobs         func(S, R) ([]runJob, error)
	// classifies reports a completion error the strategy classifies itself
	// rather than failing the round.
	classifies func(S, R, int, runCompletion) bool
	process    func(*explorationRound[S, R], int, runCompletion) (X, campaign.ExecutionRecord, error)
	commit     func(S, R, []X) (S, G, error)
	project    func(*CampaignResult, S)
	publish    func(*campaign.CampaignSummary, S, record.SHA256)
}

// explorationRound is one round's candidates being classified.
type explorationRound[S, R any] struct {
	*explorationCampaign
	state S
	round R
	tally *explorationTally
}

func runExploration[C, S, R, G, X any](run *explorationCampaign, strategy explorationStrategy[C, S, R, G, X]) error {
	rounds, err := campaign.OpenRounds(run.journal, strategy.engine, strategy.config, strategy.segmentBytes)
	if err != nil {
		reason := strategy.reason + "_setup"
		if run.resuming {
			reason = "resume_setup"
		}
		return &HostError{Reason: reason, Err: err}
	}
	tally := explorationTally{distinct: map[record.SHA256]string{}, probes: map[string]struct{}{}, choices: map[string]struct{}{}}
	if run.resuming {
		restored, err := restoreResumeSummary(run.journal.Path(), run.selection, run.resumedRuns)
		if err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
		*run.summary = restored.summary
		tally.distinct, tally.probes, tally.choices = restored.distinct, restored.probes, restored.choiceFeatures
	}
	run.summary.RecoveryExecutions = rounds.RecoveryExecutions()
	state := rounds.State()
	strategy.project(run.summary, state)
	for {
		round, available := strategy.engine.NextRound(state)
		if !available {
			break
		}
		jobs, err := strategy.jobs(state, round)
		if err != nil {
			return err
		}
		if err := rounds.Start(round); err != nil {
			return &HostError{Reason: strategy.reason + "_stage", Err: err}
		}
		completions, err := run.executeRound(jobs, strategy.reason, strategy.subject, func(index int, completion runCompletion) bool {
			return strategy.classifies != nil && strategy.classifies(state, round, index, completion)
		})
		if err != nil {
			return err
		}
		tally.summary = cloneSummary(*run.summary)
		roundTally := tally.clone()
		explored := &explorationRound[S, R]{explorationCampaign: run, state: state, round: round, tally: &roundTally}
		results := make([]X, len(completions))
		runs := make([]campaign.ExecutionRecord, len(completions))
		for index, completion := range completions {
			if results[index], runs[index], err = strategy.process(explored, index, completion); err != nil {
				return err
			}
		}
		next, segment, err := strategy.commit(state, round, results)
		if err == nil {
			err = rounds.Commit(round, segment, runs)
		}
		if err != nil {
			return &HostError{Reason: strategy.reason + "_commit", Err: err}
		}
		state, tally = next, roundTally
		*run.summary = roundTally.summary
		strategy.project(run.summary, state)
		if err := run.reportProgress(ProgressRunning, 0); err != nil {
			return &HostError{Reason: "progress_output", Err: err}
		}
	}
	return run.finish(tally, func(summary *campaign.CampaignSummary) {
		strategy.publish(summary, state, rounds.ChainSHA256())
		summary.RecoveryExecutions = rounds.RecoveryExecutions()
	})
}

// executeRound runs a round's candidates together and returns their
// completions in candidate order. A supervision error the strategy does not
// classify, or a cancellation, fails the round.
func (run *explorationCampaign) executeRound(jobs []runJob, reason, subject string, classifies func(int, runCompletion) bool) ([]runCompletion, error) {
	roundCtx, cancel := context.WithCancel(run.ctx)
	defer cancel()
	completionChannel := make(chan runCompletion, len(jobs))
	for _, job := range jobs {
		readiness := newRunReadiness()
		go runSeed(roundCtx, run.config, run.executor, run.prepared, run.baseEnvironment, run.profile, run.readOnlyMounts, run.journal, job, readiness, completionChannel)
		readiness.wait()
	}
	progressErr := run.reportProgress(ProgressRunning, len(jobs))
	if progressErr != nil {
		cancel()
	}
	completions, err := collectRoundCompletions(run.ctx, cancel, completionChannel, jobs[0].ordinal, len(jobs), reason+"_order", subject)
	if err != nil {
		return nil, err
	}
	if progressErr != nil {
		return nil, &HostError{Reason: "progress_output", Err: progressErr}
	}
	for index, completion := range completions {
		if completion.err != nil && !classifies(index, completion) {
			return nil, &HostError{Reason: supervisionFailureReason(completion.err), Err: completion.err}
		}
		if completion.result.Cancelled {
			return nil, &HostError{Reason: "runner_cancelled", Err: errors.New(subject + " candidate was cancelled")}
		}
	}
	return completions, nil
}

// finish checks the campaign's coverage and prepared target and publishes it.
func (run *explorationCampaign) finish(tally explorationTally, publish func(*campaign.CampaignSummary)) error {
	ctx, config := run.ctx, run.config
	if coverageHasSemantic(config.Coverage) {
		evidence, err := config.Backend.Evidence(context.WithoutCancel(ctx), backend.EvidenceRequest{Probes: sortedProbeList(tally.probes)})
		if err != nil {
			return &HostError{Reason: "semantic_coverage", Err: err}
		}
		coverage := evidence.Coverage
		run.summary.SemanticCoverage = &coverage
		evidence, err = config.Backend.Evidence(context.WithoutCancel(ctx), backend.EvidenceRequest{Probes: coverage.Probes, RequiredSemanticProbes: config.RequiredSemanticProbes})
		if err != nil {
			return err
		}
		if len(evidence.MissingRequiredProbes) != 0 {
			return &record.MissingSemanticProbesError{Probes: evidence.MissingRequiredProbes}
		}
	}
	if err := config.Backend.ValidatePrepared(config.Target, run.prepared, recordedEnvironmentStrings(run.baseEnvironment)); err != nil {
		return &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	if err := ctx.Err(); err != nil {
		return &HostError{Reason: contextFailureReason(err), Err: err}
	}
	failureSignatures := make([]record.SHA256, 0, len(tally.distinct))
	for signature := range tally.distinct {
		failureSignatures = append(failureSignatures, signature)
	}
	summary := run.summary
	published := campaign.CampaignSummary{
		Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures, Watchdogs: summary.Watchdogs,
		Cancelled: summary.Cancelled, DistinctFailures: summary.DistinctFailures, RetainedSuccesses: summary.RetainedSuccesses,
		RetainedSuccessBytes: summary.RetainedSuccessBytes, StopReason: string(summary.StopReason), FailureSignatures: failureSignatures,
	}
	publish(&published)
	if err := run.journal.Publish(published); err != nil {
		return &HostError{Reason: "campaign_publish", Err: err}
	}
	if err := run.reportProgress(ProgressComplete, 0); err != nil {
		return &HostError{Reason: "progress_output", Err: err}
	}
	return nil
}

// candidateOutcome is one candidate's assessed completion.
type candidateOutcome struct {
	completion  runCompletion
	worldBundle backend.Bundle
	assessed    completedExecution
}

// assess validates a candidate's World record and classifies its outcome. A
// candidate the Runner itself classified cannot be expanded and fails the
// round.
func (round *explorationRound[S, R]) assess(completion runCompletion, subject string) (candidateOutcome, error) {
	worldBundle, err := assessWorld(completion.result, completion.job.seed, round.config.WorldTransitionLimit)
	if err != nil {
		return candidateOutcome{}, &HostError{Reason: "world_record", Err: err}
	}
	assessed, assessErr := assessCompletion(round.ctx, round.config.Backend, completion.result, worldBundle.Manifest.Terminal, round.config.Coverage, round.prepared)
	if assessErr != nil {
		return candidateOutcome{}, assessErr
	}
	if assessed.outcome.Domain == "runner" {
		return candidateOutcome{}, &HostError{Reason: assessed.outcome.Reason, Err: errors.New(subject + " controller result is not expandable")}
	}
	return candidateOutcome{completion: completion, worldBundle: worldBundle, assessed: assessed}, nil
}

// checkCandidate fails a round whose campaign ended or whose prepared target
// changed before a candidate is classified.
func (round *explorationRound[S, R]) checkCandidate() error {
	if err := round.ctx.Err(); err != nil {
		return &HostError{Reason: contextFailureReason(err), Err: err}
	}
	if err := round.config.Backend.ValidatePrepared(round.config.Target, round.prepared, recordedEnvironmentStrings(round.baseEnvironment)); err != nil {
		return &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	return nil
}

// retain publishes a classified candidate's evidence in the campaign's owned
// success or failure root and counts it in the round's tally: a success the
// retention policy keeps, or a failure deduplicated by its signature. decorate
// adds a strategy's own manifest and payloads to the artifact.
func (round *explorationRound[S, R]) retain(outcome candidateOutcome, run *campaign.ExecutionRecord, transcriptComplete bool, decorate func(*artifact.ArtifactInput)) error {
	tally, completion, assessed := round.tally, outcome.completion, outcome.assessed
	mountArtifact, err := mountArtifactForRun(round.readOnlyMounts, round.config.IOROMountLimits, completion.result.IOROMounts)
	if err != nil {
		return &HostError{Reason: "manifest", Err: err}
	}
	input := func() (artifact.ArtifactInput, error) {
		manifest, err := manifestForRun(round.config, round.prepared, round.baseEnvironment, completion, assessed.outcome, round.runID, outcome.worldBundle.Manifest, mountArtifact)
		if err != nil {
			return artifact.ArtifactInput{}, err
		}
		input := executionArtifactInput(manifest, round.prepared, completion.result, mountArtifact, outcome.worldBundle)
		if decorate != nil {
			decorate(&input)
		}
		return input, nil
	}
	if assessed.outcome.Domain == "success" {
		if err := round.retainSuccess(assessed, run, transcriptComplete, input); err != nil {
			return err
		}
		tally.summary.Succeeded++
	} else if err := round.retainFailure(assessed, run, input); err != nil {
		return err
	}
	tally.summary.Attempted++
	tally.summary.DistinctFailures = uint64(len(tally.distinct))
	addStrings(tally.probes, assessed.coverage.Probes)
	addStrings(tally.choices, assessed.choiceFeatures)
	return nil
}

func (round *explorationRound[S, R]) retainSuccess(assessed completedExecution, run *campaign.ExecutionRecord, transcriptComplete bool, input func() (artifact.ArtifactInput, error)) error {
	tally := round.tally
	retention, retentionErr := decideSuccessRetention(round.config, assessed, transcriptComplete, tally.probes, tally.choices, tally.summary.RetainedSuccesses, tally.summary.RetainedSuccessBytes)
	if retentionErr != nil {
		return retentionErr
	}
	if !retention.retain {
		return nil
	}
	retained, err := input()
	if err != nil {
		return &HostError{Reason: "success_artifact_publication", Err: err}
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: round.journal.SuccessesPath(), Context: round.ctx, MaximumBytes: retention.maximumBytes, Key: artifact.StoreKeyExecution, TargetPool: artifact.TargetPool(round.config.Artifacts)}, retained)
	if err != nil {
		return successPublicationFailure(err)
	}
	relative, err := filepath.Rel(round.journal.Path(), published.Path)
	if err != nil {
		return &HostError{Reason: "artifact_path", Err: err}
	}
	retention.annotate(run, filepath.ToSlash(relative), published.StoredBytes)
	tally.summary.SuccessArtifacts = append(tally.summary.SuccessArtifacts, published.Path)
	tally.summary.RetainedSuccesses++
	tally.summary.RetainedSuccessBytes += published.StoredBytes
	return nil
}

func (round *explorationRound[S, R]) retainFailure(assessed completedExecution, run *campaign.ExecutionRecord, input func() (artifact.ArtifactInput, error)) error {
	tally := round.tally
	failure, err := input()
	if err != nil {
		return &HostError{Reason: "manifest", Err: err}
	}
	published, err := publishBoundedFailureArtifact(round.ctx, round.config, round.journal.FailuresPath(), failure.Manifest.Outcome.FailureSignature, tally.distinct, &tally.summary.failureArtifactBytes, failure)
	if err != nil {
		return &HostError{Reason: "artifact_publication", Err: err}
	}
	signature := published.Manifest.Outcome.FailureSignature
	if _, found := tally.distinct[signature]; !found {
		tally.distinct[signature] = published.Path
		tally.summary.Artifacts = append(tally.summary.Artifacts, published.Path)
	}
	relative, err := filepath.Rel(round.journal.Path(), tally.distinct[signature])
	if err != nil {
		return &HostError{Reason: "artifact_path", Err: err}
	}
	relative = filepath.ToSlash(relative)
	run.FailureSignature, run.Artifact = &signature, &relative
	tally.summary.Failures++
	if assessed.outcome.Domain == "watchdog" {
		tally.summary.Watchdogs++
	}
	if assessed.outcome.Reason == "world_replay_divergence" {
		tally.summary.ReplayDivergences++
	}
	return nil
}

// completeCandidate settles a classified candidate's attempt work.
func completeCandidate(completion runCompletion) error {
	if err := completion.journal.Transition(campaign.ExecutionClassified); err != nil {
		return &HostError{Reason: "partial_write", Err: err}
	}
	if err := completion.journal.Complete(); err != nil {
		return &HostError{Reason: "partial_cleanup", Err: err}
	}
	return nil
}

// collectRoundCompletions waits for every candidate of an exploration round
// and returns the completions in candidate order. cancel stops the candidates
// still running; subject names the strategy in an ordering failure.
//
// A parent context that ended fails the round with that end's classification.
// It is read after the last completion, not from the receive that happened to
// win: a candidate the end interrupted reports an error of its own, and a
// round that received it first would pass it on as a supervision failure.
func collectRoundCompletions(
	ctx context.Context,
	cancel context.CancelFunc,
	completionChannel <-chan runCompletion,
	startOrdinal uint64,
	candidates int,
	orderReason string,
	subject string,
) ([]runCompletion, error) {
	completions := make([]runCompletion, candidates)
	seen := make([]bool, candidates)
	received := 0
	parentDone := ctx.Done()
	for received < candidates {
		select {
		case completion := <-completionChannel:
			if completion.job.ordinal < startOrdinal || completion.job.ordinal >= startOrdinal+uint64(len(completions)) {
				cancel()
				return nil, &HostError{Reason: orderReason, Err: errors.New(subject + " completion ordinal is outside its round")}
			}
			index := int(completion.job.ordinal - startOrdinal)
			if seen[index] {
				cancel()
				return nil, &HostError{Reason: orderReason, Err: errors.New(subject + " completion ordinal is duplicated")}
			}
			completions[index] = completion
			seen[index] = true
			received++
		case <-parentDone:
			cancel()
			parentDone = nil
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, &HostError{Reason: contextFailureReason(err), Err: err}
	}
	return completions, nil
}

func cloneSummary(summary CampaignResult) CampaignResult {
	summary.Artifacts = append([]string(nil), summary.Artifacts...)
	summary.SuccessArtifacts = append([]string(nil), summary.SuccessArtifacts...)
	summary.ChoiceTrace = cloneChoiceTraceSummary(summary.ChoiceTrace)
	summary.ChoiceExploration = cloneChoiceExplorationSummary(summary.ChoiceExploration)
	summary.SimulationExploration = cloneSimulationExplorationSummary(summary.SimulationExploration)
	return summary
}

// explorationSegmentCapacity bounds one round's encoded segment by its
// components: the exploration byte bound plus one result payload bound per
// candidate of a round, within the 1 GiB journal bound.
func explorationSegmentCapacity(config campaignRequest, resultBytes uint64, subject string) (uint64, error) {
	const maximum = uint64(1 << 30)
	const overhead = uint64(4 << 20)
	exceeded := fmt.Errorf("%s round evidence exceeds the 1 GiB journal bound", subject)
	if config.MaxExplorationBytes > maximum-overhead {
		return 0, exceeded
	}
	runsPerRound := min(uint64(config.Parallel), config.MaxExecutions)
	if runsPerRound == 0 || resultBytes > (maximum-config.MaxExplorationBytes)/runsPerRound {
		return 0, exceeded
	}
	required := config.MaxExplorationBytes + runsPerRound*resultBytes
	if required > maximum-overhead {
		return 0, exceeded
	}
	return required + overhead, nil
}
