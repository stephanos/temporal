package runner

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/campaign"
)

// A seed campaign schedules through its log's admission frontier. Ordinals
// are admitted durably before their attempts launch. Each attempt's actual
// completion is observed as it arrives, in any order: its evidence is
// published and a terminal receipt recorded before it waits for its turn.
// Attempts are then handled in ordinal order. The next ordered receipt
// commits as one outcome that also admits the permitted refill, unless it
// triggers the failure policy: then admission freezes, first-failure cancels
// the active attempts, and once every admitted attempt has a receipt the
// trigger and every drained outcome commit as one stop group. Counters,
// retention and novelty advance only after their outcome is durable.
//
// A host failure observed on arrival is reported when its attempt's turn
// comes, so the first failure in ordinal order is the reported one. It never
// becomes a receipt: an attempt that failed the host runs again on resume,
// while an attempt with a durable receipt never does.

// seedAttempt is one admitted ordinal from admission until it is resolved.
type seedAttempt struct {
	job     runJob
	arrived bool
	handled bool
	// failure is a host failure observed on arrival.
	failure *seedHostFailure
	// cancelled is a cancellation the campaign requested for a host failure;
	// it is reported as that failure's evidence, not as a receipt.
	cancelled  bool
	cleanupErr error
	completion runCompletion
	// observation is the attempt's actual classified completion; received
	// reports whether its terminal receipt is durable.
	observation *seedObservation
	received    bool
	choiceTrace *record.ChoiceTraceSummary
}

// seedHostFailure is a host failure observed when an attempt arrived.
// evidence names the Runner failure reason its captured result is published
// as when the attempt's turn comes, or is empty when it has none.
type seedHostFailure struct {
	reason   string
	cause    error
	evidence bool
}

// seedObservation is an attempt's actual classified completion: the receipt
// observation and the payloads it published, plus what the ordered outcome
// reports but the receipt does not hold.
type seedObservation struct {
	run                campaign.ExecutionRecord
	success            *publishedSuccess
	successErr         *HostError
	transcriptComplete bool
	failurePath        string
	diagnostics        []artifact.FileReference
	diagnostic         *record.DiagnosticTraceReference
	evidence           *record.ExecutionEvidence
	evidenceElapsed    uint64
	corpusAdded        bool
}

// publishedSuccess is a success artifact an attempt published before its
// receipt, for its ordered outcome to retain or retire.
type publishedSuccess struct {
	relative    string
	path        string
	storedBytes uint64
}

// seedEffect is what one committed outcome applies to the campaign's
// counters, retention and novelty.
type seedEffect struct {
	completion       campaign.Completion
	probes, choices  []string
	novel            bool
	retained         *publishedSuccess
	failureSignature record.SHA256
	failurePath      string
}

// stopGroup collects the ordered outcomes of a pending policy stop until
// every admitted attempt has its receipt.
type stopGroup struct {
	reason  campaign.ControllerStopReason
	runs    []campaign.ExecutionRecord
	effects []seedEffect
	members map[uint64]struct{}
}

func (local *localCampaign) runSeeds() error {
	local.activeCtx, local.activeCancel = context.WithCancel(local.overallCtx)
	defer local.activeCancel()
	local.rawCompletions = make(chan runCompletion, local.config.Parallel)
	local.attempts = make(map[uint64]*seedAttempt)
	local.distinct = make(map[record.SHA256]string)
	local.published = make(map[record.SHA256]string)
	local.semanticProbes = make(map[string]struct{})
	local.choiceFeatures = make(map[string]struct{})
	committed := make(map[uint64]struct{})
	if local.resuming {
		restored, err := restoreResumeSummary(local.batchPath, local.selection, local.resumedRuns)
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
		maps.Copy(local.published, restored.distinct)
		local.failureArtifactBytes = restored.failureArtifactBytes
		local.semanticProbes = restored.probes
		local.choiceFeatures = restored.choiceFeatures
		committed = restored.completed
		if local.guidance != nil {
			snapshot := local.guidance.Snapshot()
			local.summary.CorpusPath = local.guidance.corpus.Path()
			local.summary.CorpusEntries = uint64(len(snapshot.Entries))
		}
	}
	frontier := local.journal.Frontier()
	local.statistics = campaign.CampaignStatistics{
		Attempted: local.summary.Attempted, Succeeded: local.summary.Succeeded, Failures: local.summary.Failures,
		Watchdogs: local.summary.Watchdogs, ReplayDivergences: local.summary.ReplayDivergences,
		Cancelled: local.summary.Cancelled, DistinctFailures: local.summary.DistinctFailures, StopReason: frontier.Stopped,
	}
	local.stopped = frontier.Stopped != ""
	synchronizeCampaignStatistics(&local.summary, local.statistics)
	skipped := maps.Clone(committed)
	for _, job := range frontier.Outstanding {
		skipped[job.Ordinal] = struct{}{}
		local.attempts[job.Ordinal] = &seedAttempt{job: runJob{ordinal: job.Ordinal, seed: job.Seed}}
	}
	for ordinal, receipt := range frontier.Receipts {
		observation, err := local.resumedObservation(receipt)
		if err != nil {
			return &HostError{Reason: "resume_setup", Err: err}
		}
		attempt := local.attempts[ordinal]
		attempt.arrived, attempt.observation, attempt.received = true, observation, true
	}
	local.jobs = &pendingJobs{seeds: local.selection.Iterator(), shard: local.config.Shard, completed: skipped}
	var progressTicks <-chan time.Time
	if local.config.Progress != nil {
		interval := local.config.ProgressInterval
		if interval <= 0 {
			interval = 5 * time.Second
		}
		progressTicker := time.NewTicker(interval)
		progressTicks = progressTicker.C
		defer progressTicker.Stop()
	}
	local.scheduleSeeds(frontier, progressTicks)
	return local.finishSeedCampaign()
}

// resumedObservation rebuilds a durable receipt's observation. Its payloads
// were validated when the campaign resumed.
func (local *localCampaign) resumedObservation(receipt campaign.Receipt) (*seedObservation, error) {
	observation := &seedObservation{run: receipt.Observation, diagnostics: slices.Clone(receipt.Diagnostics)}
	for _, reference := range receipt.Artifacts {
		path := filepath.Join(local.batchPath, filepath.FromSlash(reference))
		opened, err := artifact.OpenArtifact(path)
		if err != nil {
			return nil, fmt.Errorf("open received artifact: %w", err)
		}
		storedBytes := opened.StoredBytes()
		if err := opened.Close(); err != nil {
			return nil, fmt.Errorf("close received artifact: %w", err)
		}
		if strings.HasPrefix(reference, "successes/") {
			observation.success = &publishedSuccess{relative: reference, path: path, storedBytes: storedBytes}
			observation.transcriptComplete = true
			continue
		}
		if run := receipt.Observation; run.FailureSignature != nil && run.Artifact != nil && *run.Artifact == reference {
			observation.failurePath = path
			if _, found := local.published[*run.FailureSignature]; !found {
				local.published[*run.FailureSignature] = path
				local.failureArtifactBytes += storedBytes
			}
		}
	}
	return observation, nil
}

func (local *localCampaign) scheduleSeeds(frontier campaign.Frontier, progressTicks <-chan time.Time) {
	if frontier.PendingStop == campaign.StopFirstFailure {
		local.activeCancel()
	}
	for _, job := range frontier.Retry() {
		local.launchSeed(runJob{ordinal: job.Ordinal, seed: job.Seed})
	}
	local.advance()
	local.admit()
	runningReported := false
	for {
		if active := local.active(); active > 0 && !runningReported {
			runningReported = true
			if err := local.reportProgress(ProgressRunning, active); err != nil {
				local.failHost(&HostError{Reason: "progress_output", Err: err})
			}
		}
		if err := local.overallCtx.Err(); err != nil && local.hostFailure == nil && !local.stopped && local.group == nil {
			local.failHost(&HostError{Reason: contextFailureReason(err), Err: err})
		}
		if local.launched == 0 {
			return
		}
		select {
		case completion := <-local.rawCompletions:
			local.launched--
			local.arrive(completion)
			local.advance()
		case <-progressTicks:
			if err := local.reportProgress(ProgressRunning, local.active()); err != nil {
				local.failHost(&HostError{Reason: "progress_output", Err: err})
			}
		}
	}
}

// active counts the admitted attempts whose outcome is not yet handled.
func (local *localCampaign) active() int {
	active := 0
	for _, attempt := range local.attempts {
		if !attempt.handled {
			active++
		}
	}
	return active
}

func (local *localCampaign) launchSeed(job runJob) {
	if local.summary.Guidance != nil && !answeredSeed(local.config.guidancePlan, job.seed) {
		local.summary.Guidance.NewExecutions++
	}
	if _, found := local.attempts[job.ordinal]; !found {
		local.attempts[job.ordinal] = &seedAttempt{job: job}
	}
	local.launched++
	readiness := newRunReadiness()
	go runSeed(local.activeCtx, local.config, local.executor, local.prepared, local.baseEnvironment, local.selectedProfile, local.readOnlyMounts, local.journal, job, readiness, local.rawCompletions)
	readiness.wait()
}

// admitting reports whether the campaign may admit more ordinals now.
func (local *localCampaign) admitting() bool {
	return local.hostFailure == nil && !local.stopped && local.group == nil && local.overallCtx.Err() == nil
}

// nextJobs takes up to limit unadmitted ordinals from the selection.
func (local *localCampaign) nextJobs(limit uint64) []runJob {
	var jobs []runJob
	for uint64(len(jobs)) < limit {
		job, ok := local.jobs.Next()
		if !ok {
			break
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func seedJobs(jobs []runJob) []campaign.SeedJob {
	admitted := make([]campaign.SeedJob, len(jobs))
	for index, job := range jobs {
		admitted[index] = campaign.SeedJob{Ordinal: job.ordinal, Seed: job.seed}
	}
	return admitted
}

// admit durably reserves the free slots before their attempts launch.
func (local *localCampaign) admit() {
	if !local.admitting() {
		return
	}
	frontier := local.journal.Frontier()
	if frontier.PendingStop != "" || frontier.Stopped != "" {
		return
	}
	jobs := local.nextJobs(frontier.Available)
	if len(jobs) == 0 {
		return
	}
	if err := local.journal.Admit(seedJobs(jobs)); err != nil {
		local.failHost(&HostError{Reason: "runs_append", Err: err})
		return
	}
	for _, job := range jobs {
		local.launchSeed(job)
	}
}

// failHost records the first host failure and stops the campaign: nothing
// more is admitted and active attempts are cancelled.
func (local *localCampaign) failHost(err error) {
	if local.hostFailure != nil {
		return
	}
	local.hostFailure = err
	local.stopped = true
	local.activeCancel()
}

// advance handles every arrived attempt whose turn has come, in ordinal
// order, then commits a pending stop group once it is complete.
func (local *localCampaign) advance() {
	for {
		var next *seedAttempt
		for _, attempt := range local.attempts {
			if !attempt.handled && (next == nil || attempt.job.ordinal < next.job.ordinal) {
				next = attempt
			}
		}
		if next == nil || !next.arrived {
			break
		}
		next.handled = true
		local.handle(next)
		if next.cleanupErr != nil {
			local.failHost(&HostError{Reason: "partial_cleanup", Err: next.cleanupErr})
		}
	}
	local.commitStopGroup()
}

func (local *localCampaign) handle(attempt *seedAttempt) {
	if attempt.choiceTrace != nil {
		local.summary.ChoiceTrace = attempt.choiceTrace
	}
	switch {
	case attempt.failure != nil:
		local.statistics.Record(campaign.CompletedUnclassified())
		synchronizeCampaignStatistics(&local.summary, local.statistics)
		failure := attempt.failure
		cause := failure.cause
		if failure.evidence {
			if err := local.commitRunnerFailure(attempt, failure.reason); err != nil {
				cause = errors.Join(cause, err)
			}
		}
		local.failHost(&HostError{Reason: failure.reason, Err: cause})
	case attempt.cancelled:
		local.statistics.Record(campaign.CompletedCancelled())
		synchronizeCampaignStatistics(&local.summary, local.statistics)
		reason := "runner_failure"
		var hostError *HostError
		if errors.As(local.hostFailure, &hostError) {
			reason = hostError.Reason
		}
		if err := local.commitRunnerFailure(attempt, reason); err != nil {
			local.hostFailure = errors.Join(local.hostFailure, err)
		}
	default:
		local.handleObservation(attempt)
	}
}

// commitRunnerFailure publishes a captured attempt that failed the host as a
// Runner failure artifact and commits it as host failure evidence, which
// needs no receipt or turn and which a resumed campaign runs again.
func (local *localCampaign) commitRunnerFailure(attempt *seedAttempt, reason string) error {
	run, err := local.publishRunnerFailure(attempt.completion, reason)
	if err != nil || run == nil {
		return err
	}
	if err := local.journal.CommitOutcome(*run, nil); err != nil {
		return fmt.Errorf("append Runner failure result: %w", err)
	}
	delete(local.attempts, attempt.job.ordinal)
	return nil
}

// handleObservation decides an observed attempt's retention in its turn and
// commits its outcome, alone with the permitted refill or within a stop
// group.
func (local *localCampaign) handleObservation(attempt *seedAttempt) {
	observation := attempt.observation
	if observation.diagnostic != nil {
		local.summary.Diagnostics = observation.diagnostic
	}
	if observation.evidence != nil {
		local.summary.ExecutionEvidence = observation.evidence
		local.summary.ExecutionElapsedNanos = observation.evidenceElapsed
	}
	if observation.corpusAdded {
		local.summary.CorpusAdded++
		local.summary.CorpusEntries = uint64(len(local.guidance.Snapshot().Entries))
	}
	run, effect, hostErr := local.decideOutcome(observation)
	if hostErr != nil {
		local.statistics.Record(campaign.CompletedUnclassified())
		synchronizeCampaignStatistics(&local.summary, local.statistics)
		local.failHost(hostErr)
		return
	}
	if !attempt.received {
		if err := local.recordReceipt(observation); err != nil {
			local.statistics.Record(campaign.CompletedUnclassified())
			synchronizeCampaignStatistics(&local.summary, local.statistics)
			local.failHost(&HostError{Reason: "runs_append", Err: err})
			return
		}
		attempt.received = true
	}
	frontier := local.journal.Frontier()
	if local.group == nil && frontier.PendingStop != "" {
		local.group = &stopGroup{reason: frontier.PendingStop, members: map[uint64]struct{}{}}
		if frontier.PendingStop == campaign.StopFirstFailure {
			local.activeCancel()
		}
	}
	if local.group != nil {
		local.group.runs = append(local.group.runs, run)
		local.group.effects = append(local.group.effects, effect)
		local.group.members[attempt.job.ordinal] = struct{}{}
		return
	}
	if len(frontier.Outstanding) == 0 || frontier.Outstanding[0].Ordinal != attempt.job.ordinal {
		// An earlier attempt failed the host without a receipt; this one
		// keeps its durable receipt for a resumed campaign to commit.
		return
	}
	// The outcome always reserves the refill the uninterrupted campaign would,
	// so a later stop group admits the same ordinals however the campaign is
	// interrupted. A campaign that is failing launches none of it; a resumed
	// campaign runs those receipt-less admissions.
	refill := local.nextJobs(frontier.Available + 1)
	if err := local.journal.CommitOutcome(run, seedJobs(refill)); err != nil {
		local.statistics.Record(campaign.CompletedUnclassified())
		synchronizeCampaignStatistics(&local.summary, local.statistics)
		local.failHost(&HostError{Reason: "runs_append", Err: err})
		return
	}
	delete(local.attempts, attempt.job.ordinal)
	local.applyEffect(effect)
	if err := reportSoftAssertions(local.config.Progress, local.summary.CampaignPath, run); err != nil {
		local.failHost(&HostError{Reason: "progress_output", Err: err})
		return
	}
	launch := local.admitting()
	for _, job := range refill {
		if !launch {
			local.attempts[job.ordinal] = &seedAttempt{job: job, handled: true}
			continue
		}
		local.launchSeed(job)
	}
}

func (local *localCampaign) recordReceipt(observation *seedObservation) error {
	var pending []string
	if observation.run.Artifact != nil {
		pending = append(pending, *observation.run.Artifact)
	}
	if observation.success != nil {
		pending = append(pending, observation.success.relative)
	}
	return local.journal.RecordReceipt(campaign.Receipt{Observation: observation.run, Artifacts: pending, Diagnostics: observation.diagnostics})
}

// decideOutcome makes an observation's ordered decisions: whether a success
// is retained and what made it novel, judged against the committed state and
// the stop group's earlier members, and the completion the policy counts.
func (local *localCampaign) decideOutcome(observation *seedObservation) (campaign.ExecutionRecord, seedEffect, *HostError) {
	run := observation.run
	effect := seedEffect{probes: run.SemanticProbes, choices: run.ChoiceFeatures, novel: local.hostFailure == nil}
	probes, choices, retained, retainedBytes, distinct := local.decisionView()
	switch run.Domain {
	case "success":
		assessed := completedExecution{coverage: record.SemanticCoverage{Probes: run.SemanticProbes}, choiceFeatures: run.ChoiceFeatures}
		decision, err := decideSuccessRetention(local.config, assessed, observation.transcriptComplete, probes, choices, retained, retainedBytes)
		if err != nil {
			return run, effect, err
		}
		if decision.retain {
			published := observation.success
			if published == nil {
				if observation.successErr != nil {
					return run, effect, observation.successErr
				}
				return run, effect, &HostError{Reason: "success_artifact_publication", Err: errors.New("retained success has no published artifact")}
			}
			if published.storedBytes > decision.maximumBytes {
				return run, effect, &HostError{Reason: "success_retention_capacity", Err: &artifact.CapacityError{Required: published.storedBytes, Maximum: decision.maximumBytes}}
			}
			decision.annotate(&run, published.relative, published.storedBytes)
			effect.retained = published
		}
		effect.completion = campaign.CompletedSuccess()
		if local.hostFailure != nil {
			effect.completion = campaign.CompletedUnclassified()
		}
	case "target", "watchdog":
		signature := *run.FailureSignature
		if _, found := distinct[signature]; !found {
			distinct[signature] = struct{}{}
		}
		effect.failureSignature, effect.failurePath = signature, observation.failurePath
		effect.completion = campaign.CompletedFailure(run.Domain, run.Reason, uint64(len(distinct)))
	default:
		effect.completion = campaign.CompletedCancelled()
	}
	return run, effect, nil
}

// decisionView is the state ordered decisions are judged against: the
// committed state plus the effects of the pending stop group's members.
func (local *localCampaign) decisionView() (probes, choices map[string]struct{}, retained, retainedBytes uint64, distinct map[record.SHA256]struct{}) {
	probes, choices = maps.Clone(local.semanticProbes), maps.Clone(local.choiceFeatures)
	retained, retainedBytes = local.summary.RetainedSuccesses, local.summary.RetainedSuccessBytes
	distinct = make(map[record.SHA256]struct{}, len(local.distinct))
	for signature := range local.distinct {
		distinct[signature] = struct{}{}
	}
	if local.group != nil {
		for _, effect := range local.group.effects {
			if effect.novel {
				addStrings(probes, effect.probes)
				addStrings(choices, effect.choices)
			}
			if effect.retained != nil {
				retained++
				retainedBytes += effect.retained.storedBytes
			}
			if effect.failureSignature != "" {
				distinct[effect.failureSignature] = struct{}{}
			}
		}
	}
	return probes, choices, retained, retainedBytes, distinct
}

// applyEffect advances the counters, retention and novelty by one committed
// outcome.
func (local *localCampaign) applyEffect(effect seedEffect) {
	if effect.failureSignature != "" {
		if _, found := local.distinct[effect.failureSignature]; !found {
			local.distinct[effect.failureSignature] = effect.failurePath
			local.summary.Artifacts = append(local.summary.Artifacts, effect.failurePath)
		}
	}
	if effect.retained != nil {
		local.summary.SuccessArtifacts = append(local.summary.SuccessArtifacts, effect.retained.path)
		local.summary.RetainedSuccesses++
		local.summary.RetainedSuccessBytes += effect.retained.storedBytes
	}
	if effect.novel {
		addStrings(local.semanticProbes, effect.probes)
		addStrings(local.choiceFeatures, effect.choices)
	}
	local.statistics.Record(effect.completion)
	synchronizeCampaignStatistics(&local.summary, local.statistics)
}

// commitStopGroup commits a pending stop group once every admitted attempt
// has its receipt and has been handled: the trigger and every drained
// outcome, in ordinal order, as one transaction.
func (local *localCampaign) commitStopGroup() {
	group := local.group
	if group == nil || len(group.runs) == 0 {
		return
	}
	frontier := local.journal.Frontier()
	if frontier.PendingStop != group.reason {
		return
	}
	for _, job := range frontier.Outstanding {
		if _, member := group.members[job.Ordinal]; !member {
			return
		}
	}
	if err := local.journal.CommitStopGroup(group.runs); err != nil {
		local.failHost(&HostError{Reason: "runs_append", Err: err})
		return
	}
	for index, run := range group.runs {
		delete(local.attempts, uint64(run.SelectionOrdinal))
		local.applyEffect(group.effects[index])
	}
	local.group = nil
	local.stopped = true
	for _, run := range group.runs {
		if err := reportSoftAssertions(local.config.Progress, local.summary.CampaignPath, run); err != nil {
			local.failHost(&HostError{Reason: "progress_output", Err: err})
			return
		}
	}
	if local.hostFailure == nil {
		// A campaign that failed the host reports that failure, not the
		// policy stop it drained into.
		local.statistics.StopReason = group.reason
		synchronizeCampaignStatistics(&local.summary, local.statistics)
	}
}

func reportSoftAssertions(progress CampaignEventFunc, path string, run campaign.ExecutionRecord) error {
	if progress == nil || run.SoftAssertions == nil || !run.SoftAssertions.ReportOnly || len(run.SoftAssertions.Messages) == 0 {
		return nil
	}
	seed := uint64(run.Seed)
	return progress(CampaignEvent{Phase: ProgressRunning, CampaignPath: path, ExecutionSeed: &seed, SoftAssertions: campaign.CloneSoftAssertionReport(run.SoftAssertions)})
}

// pendingJobs yields the selection's ordinals that this campaign shard owns
// and has neither committed nor admitted, in ordinal order.
type pendingJobs struct {
	seeds     *SeedIterator
	ordinal   uint64
	shard     CampaignShard
	completed map[uint64]struct{}
}

func (jobs *pendingJobs) Next() (runJob, bool) {
	for {
		seed, ok := jobs.seeds.Next()
		if !ok {
			return runJob{}, false
		}
		job := runJob{ordinal: jobs.ordinal, seed: seed}
		jobs.ordinal++
		if !normalizedCampaignShard(jobs.shard).Owns(job.ordinal) {
			continue
		}
		if _, found := jobs.completed[job.ordinal]; found {
			continue
		}
		return job, true
	}
}

func synchronizeCampaignStatistics(summary *CampaignResult, statistics campaign.CampaignStatistics) {
	summary.Attempted = statistics.Attempted
	summary.Succeeded = statistics.Succeeded
	summary.Failures = statistics.Failures
	summary.Watchdogs = statistics.Watchdogs
	summary.ReplayDivergences = statistics.ReplayDivergences
	summary.Cancelled = statistics.Cancelled
	summary.DistinctFailures = statistics.DistinctFailures
	summary.StopReason = StopReason(statistics.StopReason)
}
