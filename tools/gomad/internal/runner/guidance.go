package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	guide "go.temporal.io/server/tools/gomad/internal/runner/internal/corpus"
)

type guidanceCampaign struct {
	corpus   *guide.Corpus
	config   campaignRequest
	prepared target.Prepared
	baseEnv  []record.Environment
	runID    string
	replayer ArtifactReplayer
}

func openGuidance(ctx context.Context, config campaignRequest, prepared target.Prepared, baseEnvironment []record.Environment, runID string) (*guidanceCampaign, error) {
	if config.Replayer == nil && len(config.Backend.Capabilities().ReplayModes) == 0 {
		return nil, errors.New("guided exploration requires a replay-capable provider")
	}
	targetRecord := prepared.RecordTarget()
	description, err := config.Backend.Describe(ctx, backends.Configuration{Target: config.Target, ClockTick: config.ClockTick, Environment: environmentStrings(baseEnvironment), RequiredSemanticProbes: config.RequiredSemanticProbes})
	if err != nil {
		return nil, err
	}
	identity, err := guide.IdentityFor(targetRecord, prepared.RecordToolchain(), description, environmentForSeed(baseEnvironment, 0))
	if coverageHasChoice(config.Coverage) {
		implementation, identityErr := choice.ImplementationIdentity(prepared.BuildKey)
		if identityErr != nil {
			return nil, identityErr
		}
		identity, err = guide.IdentityForChoice(targetRecord, prepared.RecordToolchain(), description, environmentForSeed(baseEnvironment, 0), guide.ChoiceProfileIdentity{
			Profile: choice.Profile, ImplementationSHA256: record.SHA256FromSum(implementation), Limit: record.Uint64String(config.ChoiceTraceLimit),
		})
	}
	if err != nil {
		return nil, err
	}
	corpus, err := guide.Open(ctx, config.Corpus, identity, description, config.Backend)
	if err != nil {
		return nil, err
	}
	replayer := config.Replayer
	if replayer == nil {
		replayer = artifactReplayer{dependencies: executionDependencies{executor: config.executor}}
	}
	return &guidanceCampaign{
		corpus: corpus, config: config, prepared: prepared, baseEnv: append([]record.Environment(nil), baseEnvironment...), runID: runID, replayer: replayer,
	}, nil
}

func (campaign *guidanceCampaign) Close() error {
	if campaign == nil {
		return nil
	}
	return campaign.corpus.Close()
}

func (campaign *guidanceCampaign) Snapshot() guide.Snapshot {
	return campaign.corpus.Snapshot()
}

func (campaign *guidanceCampaign) MergeRun(
	ctx context.Context,
	completion runCompletion,
	outcome backends.Classification,
	worldBundle backends.Bundle,
	mountArtifact *captured.CapturedInputs,
	coverage record.SemanticCoverage,
) (bool, error) {
	if !completion.result.IOTranscript.Complete || outcome.ReplayMode == record.ReplayNone {
		return false, nil
	}
	manifest, err := manifestForRun(campaign.config, campaign.prepared, campaign.baseEnv, completion, outcome, campaign.runID, worldBundle.Manifest, mountArtifact)
	if err != nil {
		return false, err
	}
	var choiceFeatures *choice.FeatureProjection
	if coverageHasChoice(campaign.config.Coverage) {
		projected, _, projectErr := projectChoiceFeatures(completion.result.ChoiceTrace, campaign.prepared)
		if projectErr != nil {
			return false, projectErr
		}
		choiceFeatures = &projected
	}
	candidate := guide.Candidate{
		Artifact: executionArtifactInput(manifest, campaign.prepared, completion.result, mountArtifact, worldBundle),
		Coverage: coverage, Choices: choiceFeatures,
	}
	return campaign.corpus.Admit(ctx, candidate, func(ctx context.Context, path string) (guide.ReplayResult, error) {
		replayConfig := ReplaySpec{
			ArtifactPath: path, ToolchainRoot: campaign.config.Target.ToolchainRoot,
			Backend: campaign.config.Backend,
		}
		replayed, err := campaign.replayer.Replay(ctx, replayConfig)
		return guide.ReplayResult{Verified: replayed.Verified, Match: replayed.Match, Diagnostic: replayed.Diagnostic, Divergence: replayed.Divergence}, err
	})
}

func guidedCorpusPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve guided corpus path: %w", err)
	}
	return absolute, nil
}

type GuidanceSummary struct {
	Regression    bool   `json:"regression"`
	Requested     uint64 `json:"requested"`
	Answered      uint64 `json:"answered"`
	Guided        uint64 `json:"guided"`
	NewExecutions uint64 `json:"new_executions"`
}

func selectGuidedSeeds(base SeedSelection, snapshot guide.Snapshot, corpus string, regression bool) (SeedSelection, *campaign.GuidancePlan, error) {
	answeredSet := make(map[uint64]struct{})
	for _, entry := range snapshot.Entries {
		if entry.Replay.Verified && entry.Replay.Match {
			answeredSet[uint64(entry.Seed)] = struct{}{}
		}
	}
	answered := make([]uint64, 0, len(answeredSet))
	for seed := range answeredSet {
		answered = append(answered, seed)
	}
	slices.Sort(answered)
	frozen := &campaign.GuidancePlan{Corpus: corpus, SnapshotSHA256: snapshot.SnapshotSHA256, Regression: regression, RequestedSelection: base.String(), RequestedCount: record.Uint64String(base.Count()), AnsweredSeeds: make([]record.Uint64String, len(answered))}
	for i, seed := range answered {
		frozen.AnsweredSeeds[i] = record.Uint64String(seed)
	}
	selection := base
	if !regression {
		selection = excludeAnsweredSeeds(base, answered)
		frozen.AnsweredCount = record.Uint64String(base.Count() - selection.Count())
	}
	prioritized := snapshot.PrioritizedSeeds()
	if !regression {
		filtered := prioritized[:0]
		for _, seed := range prioritized {
			if _, found := answeredSet[seed]; !found {
				filtered = append(filtered, seed)
			}
		}
		prioritized = filtered
	}
	unguided := selection.Count() / 4
	if selection.Count()%4 != 0 {
		unguided++
	}
	frozen.GuidedCount = record.Uint64String(min(uint64(len(prioritized)), selection.Count()-unguided))
	mixed, err := mixGuidedSelection(selection, prioritized)
	return mixed, frozen, err
}

func guidanceSummary(plan *campaign.GuidancePlan, newExecutions uint64) *GuidanceSummary {
	if plan == nil {
		return nil
	}
	return &GuidanceSummary{Regression: plan.Regression, Requested: uint64(plan.RequestedCount), Answered: uint64(plan.AnsweredCount), Guided: uint64(plan.GuidedCount), NewExecutions: newExecutions}
}

func answeredSeed(plan *campaign.GuidancePlan, seed uint64) bool {
	if plan == nil {
		return false
	}
	_, found := slices.BinarySearch(plan.AnsweredSeeds, record.Uint64String(seed))
	return found
}

func newGuidedExecutions(plan *campaign.GuidancePlan, runs []campaign.ExecutionRecord) uint64 {
	var count uint64
	for _, run := range runs {
		if !answeredSeed(plan, uint64(run.Seed)) {
			count++
		}
	}
	return count
}

func cloneGuidanceSummary(summary *GuidanceSummary) *GuidanceSummary {
	if summary == nil {
		return nil
	}
	result := *summary
	return &result
}
