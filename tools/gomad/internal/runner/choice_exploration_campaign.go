package runner

import (
	"encoding/json"
	"errors"
	"unicode/utf8"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/exploration"
)

func runChoiceExplorationLocal(run *explorationCampaign) error {
	baseSeed, ok := run.selection.SeedAt(0)
	if !ok || run.selection.Count() != 1 {
		return errors.New("choice-exploration base seed is unavailable")
	}
	config := run.config
	implementation, err := choice.ImplementationIdentity(run.prepared.BuildKey)
	if err != nil {
		return &HostError{Reason: "choice_exploration_setup", Err: err}
	}
	executionIdentity, err := choiceExecutionIdentity(run.prepared, implementation)
	if err != nil {
		return &HostError{Reason: "choice_exploration_setup", Err: err}
	}
	explorationConfig := exploration.ChoiceConfig{
		Execution: executionIdentity, ControllerSHA256: exploration.ChoiceImplementationSHA256(), BaseSeed: baseSeed, Parallel: config.Parallel,
		MaxExecutions: config.MaxExecutions, MaxChoiceDepth: config.MaxChoiceDepth, StartOrdinal: config.ChoiceStartOrdinal, MaxExplorationBytes: config.MaxExplorationBytes,
		FailurePolicy: exploration.FailurePolicy(config.OnFailure), FailureBudget: config.FailureBudget,
	}
	if _, err := exploration.NewChoice(explorationConfig); err != nil {
		return &HostError{Reason: "choice_exploration_setup", Err: err}
	}
	segmentBytes, err := explorationSegmentCapacity(config, config.ChoiceTraceLimit, "choice-exploration")
	if err != nil {
		return &HostError{Reason: "choice_exploration_setup", Err: err}
	}
	return runExploration(run, explorationStrategy[exploration.ChoiceConfig, exploration.ChoiceState, exploration.ChoiceRound, exploration.ChoiceRoundSegment, exploration.ChoiceResult]{
		reason: "choice_exploration", subject: "choice-exploration", engine: campaign.ChoiceRounds, config: explorationConfig, segmentBytes: segmentBytes,
		jobs: choiceExplorationJobs,
		classifies: func(state exploration.ChoiceState, round exploration.ChoiceRound, index int, completion runCompletion) bool {
			return isCandidateReplayDivergence(completion, round.Candidates[index], state.Config.Execution)
		},
		process: processExplorationCompletion,
		commit:  exploration.CommitChoiceRound,
		project: func(summary *CampaignResult, state exploration.ChoiceState) {
			projected := projectChoiceExplorationSummary(state.Summary())
			summary.ChoiceExploration = &projected
			summary.StopReason = StopReason(state.Summary().StopReason)
		},
		publish: func(published *campaign.CampaignSummary, state exploration.ChoiceState, chain record.SHA256) {
			explored := state.Summary()
			published.ReplayDivergences = run.summary.ReplayDivergences
			published.ChoiceExploration, published.ChoiceExplorationImplementationSHA256 = &explored, exploration.ChoiceImplementationSHA256()
			published.ChoiceExplorationChainSHA256 = chain
		},
	})
}

// choiceExplorationJobs runs a round's candidates from the base seed, forcing
// each candidate's recorded prefix.
func choiceExplorationJobs(state exploration.ChoiceState, round exploration.ChoiceRound) ([]runJob, error) {
	jobs := make([]runJob, len(round.Candidates))
	for index, candidate := range round.Candidates {
		var tape *choice.ReplayPlan
		mode := choice.ModeRecord
		if candidate.ForcedDepth != 0 {
			prefix, err := candidate.PrefixReplayPlan(state.Config.Execution)
			if err != nil {
				return nil, &HostError{Reason: "choice_exploration_prefix", Err: err}
			}
			tape = &prefix
			mode = choice.ModePrefix
		}
		jobs[index] = runJob{ordinal: state.LogicalExecutions + uint64(index), seed: state.Config.BaseSeed, choiceMode: mode, choiceReplayPlan: tape, choiceSourceTrace: candidate.SourceTrace}
	}
	return jobs, nil
}

func processExplorationCompletion(round *explorationRound[exploration.ChoiceState, exploration.ChoiceRound], index int, completion runCompletion) (exploration.ChoiceResult, campaign.ExecutionRecord, error) {
	if err := round.checkCandidate(); err != nil {
		return exploration.ChoiceResult{}, campaign.ExecutionRecord{}, err
	}
	state, candidate := round.state, round.round.Candidates[index]
	roundValue, depthValue := record.Uint64String(round.round.Index), record.Uint64String(candidate.ForcedDepth)
	run := campaign.ExecutionRecord{
		Strategy: string(StrategyChoiceExploration), Round: &roundValue, CandidateSHA256: candidate.SHA256,
		ParentCandidateSHA256: candidate.ParentSHA256, PrefixSHA256: candidate.PrefixSHA256, ForcedDepth: &depthValue,
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	if isCandidateReplayDivergence(completion, candidate, state.Config.Execution) {
		return recordChoiceDivergence(round, candidate, completion, run)
	}
	outcome, err := round.assess(completion, "choice-exploration")
	if err != nil {
		return exploration.ChoiceResult{}, run, err
	}
	tape, err := choice.ProjectReplayPlan(completion.result.ChoiceTrace.Trace, state.Config.Execution)
	if err != nil {
		return exploration.ChoiceResult{}, run, &HostError{Reason: "choice_trace_malformed", Err: err}
	}
	completion.result.ChoiceTrace.TapeSHA256 = tape.SHA256
	completion.result.ChoiceTrace.Decisions = uint64(len(tape.Decisions))
	outcome.completion = completion
	round.tally.summary.ChoiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
	assessed := outcome.assessed
	outcomeSHA256, err := explorationOutcomeSHA256(assessed.outcome, completion.result, outcome.worldBundle.Manifest)
	if err != nil {
		return exploration.ChoiceResult{}, run, &HostError{Reason: "choice_exploration_outcome", Err: err}
	}
	run.OutcomeSHA256, run.Domain, run.Reason, run.Termination = outcomeSHA256, assessed.outcome.Domain, assessed.outcome.Reason, assessed.outcome.Termination
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	run.SemanticProbes = append([]string(nil), assessed.coverage.Probes...)
	run.ChoiceFeatures = append([]string(nil), assessed.choiceFeatures...)
	if round.config.CollectExecutionEvidence {
		mountArtifact, err := mountArtifactForRun(round.readOnlyMounts, round.config.IOROMountLimits, completion.result.IOROMounts)
		if err != nil {
			return exploration.ChoiceResult{}, run, &HostError{Reason: "manifest", Err: err}
		}
		runRecord := executionEvidence(round.config, round.prepared, round.baseEnvironment, completion, assessed.outcome, outcome.worldBundle.Manifest, mountArtifact, assessed.coverage, assessed.choiceProjection)
		runRecord.ChoiceExploration = &record.ChoiceExplorationEvidence{
			ImplementationSHA256: exploration.ChoiceImplementationSHA256(), Round: roundValue, CandidateSHA256: candidate.SHA256,
			ParentSHA256: candidate.ParentSHA256, PrefixSHA256: candidate.PrefixSHA256, ForcedDepth: depthValue, OutcomeSHA256: outcomeSHA256,
		}
		round.tally.summary.ExecutionEvidence = &runRecord
	}
	if err := round.retain(outcome, &run, completion.result.IOTranscript.Complete || len(completion.result.BackendEvidence) != 0, nil); err != nil {
		return exploration.ChoiceResult{}, run, err
	}
	if err := completeCandidate(completion); err != nil {
		return exploration.ChoiceResult{}, run, err
	}
	result := exploration.ChoiceResult{CandidateSHA256: candidate.SHA256, OutcomeSHA256: outcomeSHA256, Failed: assessed.outcome.Domain == "target" || assessed.outcome.Domain == "watchdog", Trace: &tape, SourceTrace: completion.result.ChoiceTrace.Trace.Bytes}
	if run.FailureSignature != nil {
		result.FailureSHA256 = *run.FailureSignature
	}
	return result, run, nil
}

// recordChoiceDivergence records a forced prefix the target diverged from as
// a replay divergence: a failure of the candidate, not of the host.
func recordChoiceDivergence(round *explorationRound[exploration.ChoiceState, exploration.ChoiceRound], candidate exploration.ChoiceCandidate, completion runCompletion, run campaign.ExecutionRecord) (exploration.ChoiceResult, campaign.ExecutionRecord, error) {
	var divergence *backend.ChoiceReplayDivergenceError
	errors.As(completion.err, &divergence)
	evidence := choice.ProjectDivergenceEvidence(divergence.Divergence)
	run.Domain, run.Reason, run.Termination, run.Divergence = "runner", "replay_divergence", "none", &evidence
	if err := completion.journal.Transition(campaign.ExecutionCaptured); err != nil {
		return exploration.ChoiceResult{}, run, &HostError{Reason: "partial_write", Err: err}
	}
	if err := completeCandidate(completion); err != nil {
		return exploration.ChoiceResult{}, run, err
	}
	summary := &round.tally.summary
	summary.Attempted++
	summary.Failures++
	summary.ReplayDivergences++
	return exploration.ChoiceResult{CandidateSHA256: candidate.SHA256, Divergence: &divergence.Divergence}, run, nil
}

func isCandidateReplayDivergence(completion runCompletion, candidate exploration.ChoiceCandidate, identity choice.ExecutionIdentity) bool {
	if completion.result.Cancelled || completion.result.WatchdogTimeout || completion.job.choiceMode != choice.ModePrefix {
		return false
	}
	for err := completion.err; err != nil; {
		if joined, ok := err.(interface{ Unwrap() []error }); ok {
			if len(joined.Unwrap()) != 1 {
				return false
			}
			err = joined.Unwrap()[0]
		} else if wrapped, ok := err.(interface{ Unwrap() error }); ok {
			err = wrapped.Unwrap()
		} else {
			break
		}
	}
	var divergence *backend.ChoiceReplayDivergenceError
	return errors.As(completion.err, &divergence) && divergence != nil && exploration.ValidateCandidateDivergence(candidate, identity, divergence.Divergence) == nil
}

type choiceOutcomeIdentityProjection struct {
	Domain             string               `json:"domain"`
	Reason             string               `json:"reason"`
	Termination        string               `json:"termination"`
	ExitCode           *record.Uint64String `json:"exit_code,omitempty"`
	Signal             *string              `json:"signal,omitempty"`
	StdoutSHA256       record.SHA256        `json:"stdout_sha256"`
	StderrSHA256       record.SHA256        `json:"stderr_sha256"`
	IOTranscriptSHA256 record.SHA256        `json:"io_transcript_sha256"`
	BackendSHA256      record.SHA256        `json:"backend_evidence_sha256,omitempty"`
	WorldTerminal      record.WorldTerminal `json:"world_terminal"`
	WorldFinal         record.SHA256        `json:"world_final"`
}

func explorationOutcomeSHA256(outcome backend.Classification, result backend.ExecutionResult, worldRecord record.World) (record.SHA256, error) {
	projection := choiceOutcomeIdentityProjection{
		Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination,
		ExitCode: outcome.ExitCode, Signal: outcome.Signal,
		StdoutSHA256: record.SHA256FromSum(result.Stdout.FullSHA256), StderrSHA256: record.SHA256FromSum(result.Stderr.FullSHA256),
		IOTranscriptSHA256: record.SHA256FromSum(result.IOTranscript.SHA256), WorldTerminal: worldRecord.Terminal, WorldFinal: worldRecord.Final.SemanticDigest,
	}
	if len(result.BackendEvidence) != 0 {
		projection.BackendSHA256 = record.HashBytes(result.BackendEvidence)
	}
	strings := []string{projection.Domain, projection.Reason, projection.Termination, projection.WorldTerminal.Kind, projection.WorldTerminal.Detail, string(projection.WorldFinal)}
	if projection.Signal != nil {
		strings = append(strings, *projection.Signal)
	}
	for _, value := range strings {
		if !utf8.ValidString(value) {
			return "", errors.New("choice exploration outcome identity is not valid UTF-8")
		}
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		return "", err
	}
	return record.DomainHash("gomad-choice-exploration-outcome/v1", encoded), nil
}
