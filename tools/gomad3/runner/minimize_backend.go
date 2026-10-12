package runner

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
)

func (session *minimizationSession) evaluateBackendChoice(ctx context.Context, candidate simulationengine.Candidate) (minimizationTrial, error) {
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
	observed, err := runBackend(ctx, session.config.Backend, backend.Request{
		Target: session.prepared.CloneBackend(), Seed: uint64(manifest.Seed), Environment: replayEnvironment(manifest.Environment),
		Timeout: session.campaign.ExecutionTimeout, OutputBytes: session.campaign.OutputLimit,
		TranscriptBytes: uint64(manifest.Limits.IOTranscriptBytes), Choice: backendChoiceRequest(capability),
	}, nil, nil)
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
	retained, err := manifestForRun(campaignRequestFromSpec(session.campaign), session.prepared, session.baseEnvironment, completion, outcome, manifest.CampaignID, worldBundle.Manifest, nil)
	if err != nil {
		return minimizationTrial{}, err
	}
	return session.evaluateInput(ctx, executionArtifactInput(retained, session.prepared, observed, nil, worldBundle))
}
