package runner

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
	"go.temporal.io/server/tools/gomadv3/runner/internal/frontier"
	"go.temporal.io/server/tools/gomadv3/target"
)

func campaignPlan(config CampaignSpec, journal *campaignstore.CampaignJournal, prepared target.Prepared, environment []evidence.Environment, mounts []deterministicio.Mapping, selectionCount uint64) (campaignstore.CampaignPlan, error) {
	preparedPath, err := filepath.Rel(journal.Path(), prepared.Path)
	if err != nil {
		return campaignstore.CampaignPlan{}, fmt.Errorf("make prepared target path relative: %w", err)
	}
	preparedPath = filepath.ToSlash(preparedPath)
	if strings.HasPrefix(preparedPath, "../") || preparedPath == ".." {
		return campaignstore.CampaignPlan{}, fmt.Errorf("prepared target is outside its batch")
	}
	profile := deterministicio.Default()
	requiredProbes := append([]string(nil), config.RequiredSemanticProbes...)
	sort.Strings(requiredProbes)
	mountValues := make([]string, len(mounts))
	for index, mount := range mounts {
		mountValues[index] = mount.Source + "=" + strings.TrimPrefix(mount.Target, "/")
	}
	plan := campaignstore.CampaignPlan{
		Schema: campaignstore.CampaignPlanSchema, Strategy: string(normalizedStrategy(config.Strategy)), Selection: config.Seeds, SelectionCount: evidence.Uint64String(selectionCount), Parallel: evidence.Uint64String(config.Parallel),
		MaxRuns: evidence.Uint64String(config.MaxRuns), MaxChoiceDepth: evidence.Uint64String(config.MaxChoiceDepth), MaxFrontierBytes: evidence.Uint64String(config.MaxFrontierBytes),
		RunTimeoutNanos: evidence.Uint64String(config.RunTimeout), OverallTimeoutNanos: evidence.Uint64String(config.OverallTimeout), TerminateGraceNanos: evidence.Uint64String(config.TerminateGrace),
		OnFailure: string(config.OnFailure), FailureBudget: evidence.Uint64String(config.FailureBudget), OutputBytes: evidence.Uint64String(config.OutputLimit), WorldTransitionBytes: evidence.Uint64String(config.WorldTransitionLimit),
		RunnerBuild: config.RunnerBuild,
		Toolchain:   prepared.RecordToolchain(),
		Prepared: campaignstore.PreparedTargetPlan{
			Path: preparedPath, Target: prepared.RecordTarget(),
		},
		IOProfile:   profile.Identity(),
		Environment: append([]evidence.Environment(nil), environment...), IOROMounts: mountValues, IOROMountLimits: recordedCapturedInputLimits(deterministicio.CapturedInputLimitsOf(config.IOROMountLimits)),
		Coverage: string(normalizedCoverage(config.Coverage)), RequiredSemanticProbes: requiredProbes,
		KeepSuccesses: string(normalizedKeepSuccesses(config.KeepSuccesses)), SuccessArtifactLimit: evidence.Uint64String(config.SuccessArtifactLimit), SuccessBytesLimit: evidence.Uint64String(config.SuccessBytesLimit),
	}
	if normalizedStrategy(config.Strategy) == StrategyChoiceFrontier {
		plan.FrontierImplementationSHA256 = frontier.ImplementationSHA256()
	}
	if config.ChoiceTraceLimit != 0 {
		implementation, err := choice.ImplementationIdentity(prepared.BuildKey)
		if err != nil {
			return campaignstore.CampaignPlan{}, fmt.Errorf("derive choice profile implementation identity: %w", err)
		}
		plan.ChoiceProfile = &campaignstore.ChoiceProfilePlan{
			Name: choice.Profile, ImplementationSHA256: evidence.SHA256FromSum(implementation), Limit: evidence.Uint64String(config.ChoiceTraceLimit),
		}
	}
	if config.Guide {
		plan.Guidance = &campaignstore.GuidancePlan{Corpus: config.Corpus, SnapshotSHA256: config.GuideSnapshotSHA256}
	}
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
