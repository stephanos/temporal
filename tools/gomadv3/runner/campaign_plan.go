package runner

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaign"
	choiceengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/choice"
	simulationengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulation"
	"go.temporal.io/server/tools/gomadv3/target"
)

func campaignPlan(config CampaignSpec, journal *campaign.CampaignJournal, prepared target.Prepared, environment []record.Environment, mounts []readonlymount.Mapping, selectionCount uint64) (campaign.CampaignPlan, error) {
	preparedPath, err := filepath.Rel(journal.Path(), prepared.Path)
	if err != nil {
		return campaign.CampaignPlan{}, fmt.Errorf("make prepared target path relative: %w", err)
	}
	preparedPath = filepath.ToSlash(preparedPath)
	if strings.HasPrefix(preparedPath, "../") || preparedPath == ".." {
		return campaign.CampaignPlan{}, fmt.Errorf("prepared target is outside its campaign")
	}
	return campaignPlanRecord(config, journal.ExecutionJournalPlan(), preparedPath, prepared, environment, mounts, selectionCount)
}

func campaignPlanRecord(config CampaignSpec, journalPlan campaign.ExecutionJournalPlan, preparedPath string, prepared target.Prepared, environment []record.Environment, mounts []readonlymount.Mapping, selectionCount uint64) (campaign.CampaignPlan, error) {
	profile := deterministicio.Default()
	requiredProbes := append([]string(nil), config.RequiredSemanticProbes...)
	sort.Strings(requiredProbes)
	mountValues := make([]string, len(mounts))
	for index, mount := range mounts {
		mountValues[index] = mount.Source + "=" + strings.TrimPrefix(mount.Target, "/")
	}
	var shard *campaign.CampaignShard
	if config.Shard.Count != 0 {
		shard = &campaign.CampaignShard{Index: record.Uint64String(config.Shard.Index), Count: record.Uint64String(config.Shard.Count)}
	}
	plan := campaign.CampaignPlan{
		Schema: campaign.CampaignPlanSchema, PlanSHA256: config.PlanSHA256, Shard: shard,
		Strategy: string(normalizedStrategy(config.Strategy)), Selection: config.Seeds, SelectionCount: record.Uint64String(selectionCount), Parallel: record.Uint64String(config.Parallel),
		Journal:       &journalPlan,
		MaxExecutions: record.Uint64String(config.MaxExecutions), MaxChoiceDepth: record.Uint64String(config.MaxChoiceDepth), MaxForcedDecisions: record.Uint64String(config.MaxForcedDecisions),
		MaxExplorationBytes: record.Uint64String(config.MaxExplorationBytes), MaxExplorationResultBytes: record.Uint64String(config.MaxExplorationResultBytes), SimulationDimensionLimits: simulationengine.DimensionLimits(config.SimulationDimensionLimits),
		ExecutionTimeoutNanos: record.Uint64String(config.ExecutionTimeout), OverallTimeoutNanos: record.Uint64String(config.OverallTimeout), TerminateGraceNanos: record.Uint64String(config.TerminateGrace),
		OnFailure: string(config.OnFailure), FailureBudget: record.Uint64String(config.FailureBudget), OutputBytes: record.Uint64String(config.OutputLimit), WorldTransitionBytes: record.Uint64String(config.WorldTransitionLimit),
		RunnerBuild: config.RunnerBuild,
		Toolchain:   prepared.RecordToolchain(),
		Prepared: campaign.PreparedTargetPlan{
			Path: preparedPath, Target: prepared.RecordTarget(),
		},
		IOProfile:   profile.Identity(),
		Environment: append([]record.Environment(nil), environment...), IOROMounts: mountValues, IOROMountLimits: recordedCapturedInputLimits(readonlymount.CapturedInputLimitsOf(config.IOROMountLimits)),
		Coverage: string(normalizedCoverage(config.Coverage)), RequiredSemanticProbes: requiredProbes,
		KeepSuccesses: string(normalizedKeepSuccesses(config.KeepSuccesses)), SuccessArtifactLimit: record.Uint64String(config.SuccessArtifactLimit), SuccessBytesLimit: record.Uint64String(config.SuccessBytesLimit),
	}
	if normalizedStrategy(config.Strategy) == StrategyChoiceExploration {
		plan.ChoiceExplorationImplementationSHA256 = choiceengine.ImplementationSHA256()
	}
	if normalizedStrategy(config.Strategy) == StrategySimulationExploration {
		plan.SimulationExplorationImplementationSHA256 = simulationengine.ImplementationSHA256()
	}
	if config.ChoiceTraceLimit != 0 {
		implementation, err := choice.ImplementationIdentity(prepared.BuildKey)
		if err != nil {
			return campaign.CampaignPlan{}, fmt.Errorf("derive choice profile implementation identity: %w", err)
		}
		plan.ChoiceProfile = &campaign.ChoiceProfilePlan{
			Name: choice.Profile, ImplementationSHA256: record.SHA256FromSum(implementation), Limit: record.Uint64String(config.ChoiceTraceLimit),
		}
	}
	if config.Guide {
		plan.Guidance = &campaign.GuidancePlan{Corpus: config.Corpus, SnapshotSHA256: config.GuideSnapshotSHA256}
	}
	artifacts, err := campaign.DeriveArtifactCapacityPlan(plan)
	if err != nil {
		return campaign.CampaignPlan{}, fmt.Errorf("derive artifact capacity: %w", err)
	}
	plan.Artifacts = &artifacts
	return plan, nil
}

func normalizedStrategy(strategy Strategy) Strategy {
	if strategy == "" {
		return StrategySeed
	}
	return strategy
}

func normalizedCoverage(mode CoverageMode) CoverageMode {
	if mode == "" {
		return CoverageNone
	}
	return mode
}

func normalizedKeepSuccesses(policy KeepSuccesses) KeepSuccesses {
	if policy == "" {
		return KeepSuccessesNone
	}
	return policy
}
