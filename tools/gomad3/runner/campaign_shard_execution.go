package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/captured"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

type CampaignShardSpec struct {
	Backend          backends.Provider `json:"-"`
	PlanPath         string
	Shard            CampaignShard
	Artifacts        string
	ToolchainRoot    string
	RunnerBuild      string
	Progress         CampaignEventFunc
	ProgressInterval time.Duration
	Replayer         ArtifactReplayer
}

func RunCampaignShard(ctx context.Context, spec CampaignShardSpec) (CampaignResult, error) {
	return runCampaignShardWith(ctx, spec, executionDependencies{})
}

func runCampaignShardWith(ctx context.Context, spec CampaignShardSpec, dependencies executionDependencies) (CampaignResult, error) {
	if err := spec.Shard.Validate(); err != nil {
		return CampaignResult{}, err
	}
	opened, err := openCampaignPlan(spec.PlanPath)
	if err != nil {
		return CampaignResult{}, err
	}
	plan := opened.plan
	if plan.RunnerBuild != spec.RunnerBuild {
		return CampaignResult{}, fmt.Errorf("campaign plan Runner build identity %s does not match this Runner %s", plan.RunnerBuild, spec.RunnerBuild)
	}
	if spec.Backend == nil {
		return CampaignResult{}, errors.New("campaign shard requires a backend provider")
	}
	description, err := spec.Backend.Describe(ctx, backends.Configuration{Target: target.Spec{ToolchainRoot: spec.ToolchainRoot}, IOProfile: plan.IOProfile.Name})
	if err != nil {
		return CampaignResult{}, err
	}
	if plan.IOProfile.Name != description.IOProfile.Name || plan.IOProfile.ImplementationSHA256 != description.IOProfile.ImplementationSHA256 || plan.IOProfile.InventorySHA256 != description.IOProfile.InventorySHA256 {
		return CampaignResult{}, errors.New("campaign plan I/O profile identity does not match the backend provider")
	}
	preparedSpec := target.Spec{ToolchainRoot: spec.ToolchainRoot, Kind: opened.prepared.Kind, Source: opened.prepared.Source, Args: append([]string(nil), opened.prepared.Argv[1:]...), BuildTags: append([]string(nil), opened.prepared.BuildTags...), CapabilityMode: opened.prepared.CapabilityMode}
	if opened.prepared.Backend != nil {
		preparedSpec.Backend = opened.prepared.Backend.Name
	}
	if err := spec.Backend.ValidatePrepared(preparedSpec, opened.prepared, environmentStrings(plan.Environment)); err != nil {
		return CampaignResult{}, err
	}
	mountLimits, err := captured.DecodeLimits(deterministicCapturedInputLimits(plan.IOROMountLimits))
	if err != nil {
		return CampaignResult{}, err
	}
	mappings, err := captured.ParseMappings(plan.IOROMounts, opened.path+campaignPlanBundleSuffix)
	if err != nil {
		return CampaignResult{}, err
	}
	mountIdentity, _, err := captureCampaignPlanMounts(mappings, mountLimits)
	if err != nil {
		return CampaignResult{}, fmt.Errorf("verify campaign plan mount identity: %w", err)
	}
	if !equalCampaignPlanMountIdentity(mountIdentity, opened.mounts) {
		return CampaignResult{}, errors.New("campaign plan read-only mount identity changed")
	}
	environment, clockTick, err := campaignPlanEnvironment(plan)
	if err != nil {
		return CampaignResult{}, err
	}
	targetRecord := plan.Prepared.Target
	config := CampaignSpec{
		Backend:    spec.Backend,
		PlanSHA256: opened.identity, Shard: spec.Shard, Strategy: StrategySeed, Seeds: plan.Selection, Parallel: int(plan.Parallel),
		ExecutionTimeout: time.Duration(plan.ExecutionTimeoutNanos), OverallTimeout: time.Duration(plan.OverallTimeoutNanos), TerminateGrace: time.Duration(plan.TerminateGraceNanos),
		OnFailure: PolicyAll, FailureBudget: uint64(plan.FailureBudget), OutputLimit: uint64(plan.OutputBytes), WorldTransitionLimit: uint64(plan.WorldTransitionBytes),
		Diagnostics: campaignPlanDiagnostics(plan), ChoiceTraceLimit: campaignPlanChoiceLimit(plan.ChoiceProfile), ClockTick: clockTick, IOTranscriptLimit: uint64(plan.IOTranscriptBytes), Artifacts: spec.Artifacts, Environment: environment,
		IOROMounts: campaignPlanRuntimeMountValues(mappings), IOROMountLimits: mountLimits,
		Target: target.Spec{
			Kind: target.Kind(targetRecord.Kind), Source: targetRecord.Source, Args: append([]string(nil), targetRecord.Argv[1:]...), BuildTags: append([]string(nil), targetRecord.BuildTags...),
			WorkingDir: filepath.Dir(opened.path), ToolchainRoot: spec.ToolchainRoot, CapabilityMode: target.CapabilityMode(targetRecord.CapabilityMode),
		},
		RunnerBuild: spec.RunnerBuild,
		Coverage:    CoverageMode(plan.Coverage), RequiredSemanticProbes: append([]string(nil), plan.RequiredSemanticProbes...),
		KeepSuccesses: KeepSuccesses(plan.KeepSuccesses), SuccessArtifactLimit: uint64(plan.SuccessArtifactLimit), SuccessBytesLimit: uint64(plan.SuccessBytesLimit),
		Progress: spec.Progress, ProgressInterval: spec.ProgressInterval, Replayer: spec.Replayer,
		Preparer: &campaignPlanPreparer{source: opened.prepared, provider: spec.Backend},
	}
	if plan.Guidance != nil {
		config.Guide = true
		config.GuideRegression = plan.Guidance.Regression
		config.Corpus = plan.Guidance.Corpus
		config.GuideSnapshotSHA256 = plan.Guidance.SnapshotSHA256
		config.guidancePlan = plan.Guidance
	}
	request := campaignRequestFromSpecWith(config, dependencies)
	request.profile = description
	return runLocal(ctx, request)
}

func campaignPlanChoiceLimit(plan *campaign.ChoiceProfilePlan) uint64 {
	if plan == nil {
		return 0
	}
	return uint64(plan.Limit)
}

func campaignPlanEnvironment(plan campaign.CampaignPlan) ([]string, string, error) {
	result := make([]string, 0, len(plan.Environment))
	clockTick := ""
	ioProfile := false
	choiceProfile := plan.ChoiceProfile == nil
	for _, entry := range plan.Environment {
		switch entry.Name {
		case "GOMAD3_IO_PROFILE":
			if ioProfile || entry.Value != plan.IOProfile.Name {
				return nil, "", errors.New("campaign plan I/O profile environment is invalid")
			}
			ioProfile = true
		case "GOMAD3_CHOICE_PROFILE":
			if choiceProfile || plan.ChoiceProfile == nil || entry.Value != plan.ChoiceProfile.Name {
				return nil, "", errors.New("campaign plan choice profile environment is invalid")
			}
			choiceProfile = true
		case choice.DiagnosticProfileEnvironment:
			if entry.Value != choice.DiagnosticProfile || plan.ChoiceProfile == nil {
				return nil, "", errors.New("campaign plan diagnostic profile is invalid")
			}
		case record.ClockTickEnvironment:
			if clockTick != "" || entry.Value != record.ClockTickForward {
				return nil, "", errors.New("campaign plan clock tick environment is invalid")
			}
			clockTick = entry.Value
		default:
			result = append(result, entry.Name+"="+entry.Value)
		}
	}
	if !ioProfile || !choiceProfile {
		return nil, "", errors.New("campaign plan profile environment is incomplete")
	}
	return result, clockTick, nil
}

type campaignPlanPreparer struct {
	source   target.Prepared
	provider backends.Provider
}

func (preparer *campaignPlanPreparer) Prepare(_ context.Context, spec target.Spec) (_ target.Prepared, retErr error) {
	if preparer == nil || preparer.provider == nil {
		return target.Prepared{}, errors.New("campaign plan target is required")
	}
	if err := os.MkdirAll(spec.PreparationRoot, 0o700); err != nil {
		return target.Prepared{}, fmt.Errorf("create shard preparation directory: %w", err)
	}
	if err := os.Chmod(spec.PreparationRoot, 0o700); err != nil {
		return target.Prepared{}, fmt.Errorf("make shard preparation directory private: %w", err)
	}
	input, info, err := hostfs.OpenPath(preparer.source.Path)
	if err != nil {
		return target.Prepared{}, fmt.Errorf("open campaign plan target: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, input.Close()) }()
	if info.Mode().Perm() != 0o500 || uint64(info.Size()) != preparer.source.Size {
		return target.Prepared{}, errors.New("campaign plan target mode or size changed")
	}
	destination := filepath.Join(spec.PreparationRoot, campaignPlanTargetFile)
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o500)
	if err != nil {
		return target.Prepared{}, fmt.Errorf("create shard target: %w", err)
	}
	if _, err := io.Copy(output, io.LimitReader(input, int64(preparer.source.Size)+1)); err != nil {
		return target.Prepared{}, errors.Join(fmt.Errorf("copy shard target: %w", err), output.Close())
	}
	if err := output.Sync(); err != nil {
		return target.Prepared{}, errors.Join(err, output.Close())
	}
	if err := output.Close(); err != nil {
		return target.Prepared{}, err
	}
	prepared := preparer.source
	prepared.Path = destination
	if err := preparer.provider.ValidatePrepared(spec, prepared, nil); err != nil {
		return target.Prepared{}, err
	}
	return prepared, nil
}

func campaignPlanDiagnostics(plan campaign.CampaignPlan) bool {
	for _, entry := range plan.Environment {
		if entry.Name == choice.DiagnosticProfileEnvironment {
			return true
		}
	}
	return false
}
