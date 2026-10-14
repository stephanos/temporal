package runner

import (
	"context"
	"fmt"
	"time"

	execution "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/exploration"
)

func (session *minimizationSession) evaluateBackendChoice(ctx context.Context, candidate exploration.SimulationCandidate) (minimizationTrial, error) {
	manifest := session.opened.Manifest()
	capability := &execution.ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile,
		ImplementationSHA256: session.choiceIdentity.ImplementationSHA256, ExecutionIdentity: session.choiceIdentity, Limit: uint64(manifest.Limits.ChoiceTraceBytes)}
	if len(candidate.Overrides) != 0 {
		prefix, err := session.exactChoiceTape.Prefix(uint64(len(candidate.Overrides)))
		if err != nil {
			return minimizationTrial{}, err
		}
		capability.Mode, capability.ReplayPlan = choice.ModePrefix, &prefix
	}
	startedAt := time.Now().UTC()
	request := execution.ExecutionRequest{
		Target: session.prepared.CloneBackend(), Seed: uint64(manifest.Seed), RunnerBuild: manifest.Runner.RunnerBuild,
		Command: session.prepared.Path, Args: append([]string(nil), session.prepared.Argv[1:]...), Argv0: session.prepared.Argv[0],
		Dir: session.workDirectory, Env: replayEnvironment(manifest.Environment),
		ExecutionTimeout: session.campaign.ExecutionTimeout, TerminateGrace: session.campaign.TerminateGrace, OutputLimit: session.campaign.OutputLimit,
		IO:     &execution.IOCapability{Profile: manifest.IOProfile, Transcript: &execution.IOTranscriptCapability{Limit: uint64(manifest.Limits.IOTranscriptBytes)}},
		Choice: capability,
	}
	var observed execution.ExecutionResult
	var err error
	if session.executor != nil {
		observed, err = session.executor.Run(ctx, request)
	} else {
		observed, err = session.config.Backend.Execute(ctx, request)
	}
	if backendChoiceReplayDivergence(err) {
		return minimizationTrial{accepted: false}, nil
	}
	if err != nil {
		return minimizationTrial{}, fmt.Errorf("execute minimization backend candidate: %w", err)
	}
	if err := validateObservedChoiceTrace(session.campaign.ChoiceTraceLimit, capability, &observed.ChoiceTrace); err != nil {
		return minimizationTrial{}, err
	}
	worldBundle := noneWorldBundle()
	outcome := execution.Classify(observed, false, worldBundle.Manifest.Terminal)
	if outcome.ArtifactKind != record.ArtifactTargetFailure {
		return minimizationTrial{accepted: false}, nil
	}
	outcome.ReplayMode = session.prepared.Backend.ReplayMode
	completion := runCompletion{job: runJob{seed: uint64(manifest.Seed)}, startedAt: startedAt, finishedAt: time.Now().UTC(), result: observed}
	requestConfig := campaignRequestFromSpec(session.campaign)
	requestConfig.profile = session.profile
	retained, err := manifestForRun(requestConfig, session.prepared, session.baseEnvironment, completion, outcome, manifest.CampaignID, worldBundle.Manifest, nil)
	if err != nil {
		return minimizationTrial{}, err
	}
	return session.evaluateInput(ctx, executionArtifactInput(retained, session.prepared, observed, nil, worldBundle))
}
