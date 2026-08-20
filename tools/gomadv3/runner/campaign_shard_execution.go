package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
	"go.temporal.io/server/tools/gomadv3/target"
)

type CampaignShardSpec struct {
	PlanPath          string
	Shard             CampaignShard
	Artifacts         string
	ToolchainRoot     string
	RunnerBuild       string
	SupervisorCommand []string
	Progress          CampaignEventFunc
	ProgressInterval  time.Duration
	Executor          Executor
	Replayer          ArtifactReplayer
}

func RunCampaignShard(ctx context.Context, spec CampaignShardSpec) (CampaignResult, error) {
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
	if !deterministicio.Default().Matches(plan.IOProfile) {
		return CampaignResult{}, errors.New("campaign plan I/O profile identity does not match this Runner")
	}
	if spec.Executor == nil {
		identity, err := target.ReadToolchainIdentity(spec.ToolchainRoot)
		if err != nil {
			return CampaignResult{}, err
		}
		if identity.GoVersion != plan.Toolchain.GoVersion || identity.BuildKey != plan.Toolchain.BuildKey || identity.TargetGOOS != plan.Toolchain.TargetGOOS || identity.TargetGOARCH != plan.Toolchain.TargetGOARCH {
			return CampaignResult{}, errors.New("campaign plan toolchain identity does not match the pinned toolchain")
		}
	}
	mountLimits, err := deterministicio.DecodeLimits(deterministicCapturedInputLimits(plan.IOROMountLimits))
	if err != nil {
		return CampaignResult{}, err
	}
	mappings, err := deterministicio.ParseMappings(plan.IOROMounts, opened.path+campaignPlanBundleSuffix)
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
	environment, err := campaignPlanEnvironment(plan)
	if err != nil {
		return CampaignResult{}, err
	}
	targetRecord := plan.Prepared.Target
	config := CampaignSpec{
		PlanSHA256: opened.identity, Shard: spec.Shard, Strategy: StrategySeed, Seeds: plan.Selection, Parallel: int(plan.Parallel),
		RunTimeout: time.Duration(plan.RunTimeoutNanos), OverallTimeout: time.Duration(plan.OverallTimeoutNanos), TerminateGrace: time.Duration(plan.TerminateGraceNanos),
		OnFailure: PolicyAll, FailureBudget: uint64(plan.FailureBudget), OutputLimit: uint64(plan.OutputBytes), WorldTransitionLimit: uint64(plan.WorldTransitionBytes),
		ChoiceTraceLimit: campaignPlanChoiceLimit(plan.ChoiceProfile), Artifacts: spec.Artifacts, Environment: environment,
		IOROMounts: campaignPlanRuntimeMountValues(mappings), IOROMountLimits: mountLimits,
		Target: target.Spec{
			Kind: target.Kind(targetRecord.Kind), Source: targetRecord.Source, Args: append([]string(nil), targetRecord.Argv[1:]...), BuildTags: append([]string(nil), targetRecord.BuildTags...),
			WorkingDir: filepath.Dir(opened.path), ToolchainRoot: spec.ToolchainRoot, CapabilityMode: target.CapabilityMode(targetRecord.CapabilityMode),
		},
		SupervisorCommand: append([]string(nil), spec.SupervisorCommand...), RunnerBuild: spec.RunnerBuild,
		Coverage: CoverageMode(plan.Coverage), RequiredSemanticProbes: append([]string(nil), plan.RequiredSemanticProbes...),
		KeepSuccesses: KeepSuccesses(plan.KeepSuccesses), SuccessArtifactLimit: uint64(plan.SuccessArtifactLimit), SuccessBytesLimit: uint64(plan.SuccessBytesLimit),
		Progress: spec.Progress, ProgressInterval: spec.ProgressInterval, Executor: spec.Executor, Replayer: spec.Replayer,
		Preparer: &campaignPlanPreparer{source: opened.prepared},
	}
	return runLocal(ctx, config)
}

func campaignPlanChoiceLimit(plan *campaignstore.ChoiceProfilePlan) uint64 {
	if plan == nil {
		return 0
	}
	return uint64(plan.Limit)
}

func campaignPlanEnvironment(plan campaignstore.CampaignPlan) ([]string, error) {
	result := make([]string, 0, len(plan.Environment))
	ioProfile := false
	choiceProfile := plan.ChoiceProfile == nil
	for _, entry := range plan.Environment {
		switch entry.Name {
		case "GOMADV3_IO_PROFILE":
			if ioProfile || entry.Value != deterministicio.Deterministic {
				return nil, errors.New("campaign plan I/O profile environment is invalid")
			}
			ioProfile = true
		case "GOMADV3_CHOICE_PROFILE":
			if choiceProfile || plan.ChoiceProfile == nil || entry.Value != plan.ChoiceProfile.Name {
				return nil, errors.New("campaign plan choice profile environment is invalid")
			}
			choiceProfile = true
		default:
			result = append(result, entry.Name+"="+entry.Value)
		}
	}
	if !ioProfile || !choiceProfile {
		return nil, errors.New("campaign plan profile environment is incomplete")
	}
	return result, nil
}

type campaignPlanPreparer struct {
	source target.Prepared
}

func (preparer *campaignPlanPreparer) Prepare(_ context.Context, spec target.Spec) (_ target.Prepared, retErr error) {
	if preparer == nil {
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
	if err := prepared.Verify(); err != nil {
		return target.Prepared{}, err
	}
	return prepared, nil
}
