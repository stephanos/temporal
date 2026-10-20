package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/exploration"
)

const combinedExplorationRecordLimit = uint64(128 << 20)

func runSimulationExplorationLocal(run *explorationCampaign) error {
	baseSeed, ok := run.selection.SeedAt(0)
	if !ok || run.selection.Count() != 1 {
		return errors.New("simulation-exploration base seed is unavailable")
	}
	config := run.config
	choiceImplementation, err := choice.ImplementationIdentity(run.prepared.BuildKey)
	if err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	choiceIdentity, err := choiceExecutionIdentity(run.prepared, choiceImplementation)
	if err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	executionSHA256, err := simulationExplorationExecutionIdentity(config, run.prepared, run.profile, choiceImplementation)
	if err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	failureBudget := config.FailureBudget
	if config.OnFailure == PolicyAll {
		failureBudget = config.MaxExecutions
	}
	explorationConfig := exploration.SimulationConfig{
		ExecutionSHA256: executionSHA256, ControllerSHA256: exploration.SimulationImplementationSHA256(), BaseSeed: baseSeed,
		Parallel: config.Parallel, MaxExecutions: config.MaxExecutions, MaxForcedDecisions: config.MaxForcedDecisions,
		MaxExplorationBytes: config.MaxExplorationBytes, MaxResultBytes: config.MaxExplorationResultBytes,
		FailureBudget: failureBudget, Limits: exploration.DimensionLimits(config.SimulationDimensionLimits),
	}
	if _, err := exploration.NewSimulation(explorationConfig); err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	segmentBytes, err := explorationSegmentCapacity(config, config.MaxExplorationResultBytes, "simulation-exploration")
	if err != nil {
		return &HostError{Reason: "simulation_exploration_setup", Err: err}
	}
	return runExploration(run, explorationStrategy[exploration.SimulationConfig, exploration.SimulationState, exploration.SimulationRound, exploration.SimulationRoundSegment, exploration.SimulationResult]{
		reason: "simulation_exploration", subject: "simulation exploration", engine: campaign.SimulationRounds, config: explorationConfig, segmentBytes: segmentBytes,
		jobs: func(state exploration.SimulationState, round exploration.SimulationRound) ([]runJob, error) {
			return simulationExplorationJobs(state, round, choiceIdentity)
		},
		process: func(round *explorationRound[exploration.SimulationState, exploration.SimulationRound], index int, completion runCompletion) (exploration.SimulationResult, campaign.ExecutionRecord, error) {
			return processSimulationExplorationCompletion(round, index, completion, choiceIdentity)
		},
		commit: exploration.CommitSimulationRound,
		project: func(summary *CampaignResult, state exploration.SimulationState) {
			projected := projectSimulationExplorationSummary(state.Summary())
			summary.SimulationExploration = &projected
			summary.StopReason = StopReason(state.Summary().StopReason)
		},
		publish: func(published *campaign.CampaignSummary, state exploration.SimulationState, chain record.SHA256) {
			explored := state.Summary()
			published.SimulationExploration, published.SimulationExplorationImplementationSHA256 = &explored, exploration.SimulationImplementationSHA256()
			published.SimulationExplorationChainSHA256 = chain
		},
	})
}

// simulationExplorationJobs runs a round's candidates from the base seed
// under each candidate's forced simulation and choice decisions.
func simulationExplorationJobs(state exploration.SimulationState, round exploration.SimulationRound, choiceIdentity choice.ExecutionIdentity) ([]runJob, error) {
	jobs := make([]runJob, len(round.Candidates))
	for index, candidate := range round.Candidates {
		candidateExecution, err := exploration.ExecutionForCandidate(state.Config, candidate, choiceIdentity)
		if err != nil {
			return nil, &HostError{Reason: "simulation_exploration_control", Err: err}
		}
		jobs[index] = runJob{
			ordinal: state.LogicalExecutions + uint64(index), seed: state.Config.BaseSeed,
			choiceMode: candidateExecution.ChoiceMode, choiceReplayPlan: candidateExecution.ChoiceReplayPlan,
			choiceSourceTrace: candidateExecution.ChoiceSourceTrace,
			simulationPlan:    string(candidateExecution.SimulationPlan), simulationRecordLimit: combinedExplorationRecordLimit, simulationRecordCount: 1,
		}
	}
	return jobs, nil
}

func processSimulationExplorationCompletion(round *explorationRound[exploration.SimulationState, exploration.SimulationRound], index int, completion runCompletion, choiceIdentity choice.ExecutionIdentity) (exploration.SimulationResult, campaign.ExecutionRecord, error) {
	if err := round.checkCandidate(); err != nil {
		return exploration.SimulationResult{}, campaign.ExecutionRecord{}, err
	}
	state, candidate := round.state, round.round.Candidates[index]
	outcome, err := round.assess(completion, "simulation-exploration")
	if err != nil {
		return exploration.SimulationResult{}, campaign.ExecutionRecord{}, err
	}
	tape, err := choice.ProjectReplayPlan(completion.result.ChoiceTrace.Trace, choiceIdentity)
	if err != nil {
		return exploration.SimulationResult{}, campaign.ExecutionRecord{}, &HostError{Reason: "choice_trace_malformed", Err: err}
	}
	completion.result.ChoiceTrace.TapeSHA256 = tape.SHA256
	completion.result.ChoiceTrace.Decisions = uint64(len(tape.Decisions))
	outcome.completion = completion
	round.tally.summary.ChoiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
	result, simulationProfile, err := simulationExplorationResult(state, candidate, completion, tape)
	if err != nil {
		return exploration.SimulationResult{}, campaign.ExecutionRecord{}, err
	}
	assessed := outcome.assessed
	roundValue, depthValue := record.Uint64String(round.round.Index), record.Uint64String(len(candidate.Overrides))
	run := campaign.ExecutionRecord{
		Strategy: string(StrategySimulationExploration), Round: &roundValue, CandidateSHA256: candidate.SHA256,
		ParentCandidateSHA256: candidate.ParentSHA256, ForcedDepth: &depthValue, OutcomeSHA256: result.OutcomeSHA256,
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
		Domain: assessed.outcome.Domain, Reason: assessed.outcome.Reason, Termination: assessed.outcome.Termination,
		ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
	}
	setRunTranscript(&run, completion.result.IOTranscript)
	setRunChoiceTrace(&run, completion.result.ChoiceTrace)
	run.SemanticProbes = append([]string(nil), assessed.coverage.Probes...)
	run.ChoiceFeatures = append([]string(nil), assessed.choiceFeatures...)
	if round.config.CollectExecutionEvidence {
		mountArtifact, err := mountArtifactForRun(round.readOnlyMounts, round.config.IOROMountLimits, completion.result.IOROMounts)
		if err != nil {
			return exploration.SimulationResult{}, run, &HostError{Reason: "manifest", Err: err}
		}
		runRecord := executionEvidence(round.config, round.prepared, round.baseEnvironment, completion, assessed.outcome, outcome.worldBundle.Manifest, mountArtifact, assessed.coverage, assessed.choiceProjection)
		round.tally.summary.ExecutionEvidence = &runRecord
	}
	if assessed.outcome.Domain == "success" && result.Failed {
		return exploration.SimulationResult{}, run, &HostError{Reason: "simulation_exploration_outcome", Err: errors.New("simulation failure completed with a successful target outcome")}
	}
	simulationPayloads := &artifact.SimulationPayloads{Plan: []byte(completion.job.simulationPlan), Record: completion.result.SimulationRecords[0]}
	if err := round.retain(outcome, &run, completion.result.IOTranscript.Complete, func(input *artifact.ArtifactInput) {
		input.Manifest.SimulationProfile = &simulationProfile
		input.Simulation = simulationPayloads
	}); err != nil {
		return exploration.SimulationResult{}, run, err
	}
	if run.FailureSignature != nil {
		if result.Diverged && assessed.outcome.Reason != "world_replay_divergence" {
			round.tally.summary.ReplayDivergences++
		}
		result.Failed, result.FailureSHA256 = true, *run.FailureSignature
	}
	if err := completeCandidate(completion); err != nil {
		return exploration.SimulationResult{}, run, err
	}
	return result, run, nil
}

// simulationExplorationResult derives a candidate's engine result and the
// simulation profile its artifacts record from its one simulation record.
func simulationExplorationResult(state exploration.SimulationState, candidate exploration.SimulationCandidate, completion runCompletion, tape choice.ReplayPlan) (exploration.SimulationResult, record.SimulationProfile, error) {
	runtimeDecisions, err := exploration.RuntimeDecisions(tape)
	if err != nil {
		return exploration.SimulationResult{}, record.SimulationProfile{}, &HostError{Reason: "simulation_exploration_runtime", Err: err}
	}
	for index := range runtimeDecisions {
		runtimeDecisions[index].SourceTrace = append([]byte(nil), completion.result.ChoiceTrace.Trace.Bytes...)
	}
	if len(completion.result.SimulationRecords) != 1 {
		return exploration.SimulationResult{}, record.SimulationProfile{}, &HostError{Reason: "simulation_exploration_record", Err: fmt.Errorf("simulation exploration records = %d, want 1", len(completion.result.SimulationRecords))}
	}
	simulationRecord := completion.result.SimulationRecords[0]
	result, err := exploration.ResultForRecord(state.Config, candidate, simulationRecord, runtimeDecisions)
	if err != nil {
		return exploration.SimulationResult{}, record.SimulationProfile{}, &HostError{Reason: "simulation_exploration_record", Err: err}
	}
	profile, err := exploration.ProjectArtifact(state.Config, candidate, []byte(completion.job.simulationPlan), simulationRecord, runtimeDecisions, completion.job.simulationRecordLimit)
	if err != nil {
		return exploration.SimulationResult{}, record.SimulationProfile{}, &HostError{Reason: "simulation_exploration_record", Err: err}
	}
	return result, profile, nil
}

type simulationExplorationExecutionIdentityProjection struct {
	TargetSHA256         record.SHA256     `json:"target_sha256"`
	ToolchainBuildKey    string            `json:"toolchain_build_key"`
	GOOS                 string            `json:"goos"`
	GOARCH               string            `json:"goarch"`
	ChoiceImplementation record.SHA256     `json:"choice_implementation_sha256"`
	IOProfile            record.IOContract `json:"io_profile"`
	RunnerBuild          string            `json:"runner_build"`
}

func simulationExplorationExecutionIdentity(config campaignRequest, prepared target.Prepared, profile backend.Description, choiceImplementation [32]byte) (record.SHA256, error) {
	for _, input := range []struct {
		name  string
		value string
	}{
		{name: "target identity", value: prepared.SHA256}, {name: "toolchain build key", value: prepared.BuildKey},
		{name: "GOOS", value: prepared.TargetGOOS}, {name: "GOARCH", value: prepared.TargetGOARCH},
		{name: "I/O profile name", value: profile.IOProfile.Name}, {name: "I/O implementation identity", value: string(profile.IOProfile.ImplementationSHA256)},
		{name: "I/O inventory identity", value: string(profile.IOProfile.InventorySHA256)}, {name: "Runner build", value: config.RunnerBuild},
	} {
		if !utf8.ValidString(input.value) {
			return "", fmt.Errorf("simulation exploration %s is not valid UTF-8", input.name)
		}
	}
	encoded, err := json.Marshal(simulationExplorationExecutionIdentityProjection{
		TargetSHA256: record.SHA256(prepared.SHA256), ToolchainBuildKey: prepared.BuildKey,
		GOOS: prepared.TargetGOOS, GOARCH: prepared.TargetGOARCH, ChoiceImplementation: record.SHA256FromSum(choiceImplementation),
		IOProfile: record.IOContract{Name: profile.IOProfile.Name, ImplementationSHA256: profile.IOProfile.ImplementationSHA256, InventorySHA256: profile.IOProfile.InventorySHA256}, RunnerBuild: config.RunnerBuild,
	})
	if err != nil {
		return "", err
	}
	return record.DomainHash("gomad-simulation-exploration-execution/v1", encoded), nil
}
