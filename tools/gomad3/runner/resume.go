package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

type ResumeSpec struct {
	CampaignPath       string
	ToolchainRoot      string
	RunnerBuild        string
	SupervisorCommand  []string
	CoordinatorCommand []string
	Progress           CampaignEventFunc
	ProgressInterval   time.Duration
	Executor           Executor
	Replayer           ArtifactReplayer
}

func Resume(ctx context.Context, spec ResumeSpec) (CampaignResult, error) {
	return Explore(ctx, CampaignSpec{
		ResumeCampaign: spec.CampaignPath, RunnerBuild: spec.RunnerBuild,
		Target:            target.Spec{ToolchainRoot: spec.ToolchainRoot},
		SupervisorCommand: append([]string(nil), spec.SupervisorCommand...), CoordinatorCommand: append([]string(nil), spec.CoordinatorCommand...),
		Progress: spec.Progress, ProgressInterval: spec.ProgressInterval, Executor: spec.Executor, Replayer: spec.Replayer,
	})
}

func resumeRequestDefaults(config CampaignSpec) (CampaignSpec, error) {
	path, err := filepath.Abs(config.ResumeCampaign)
	if err != nil {
		return CampaignSpec{}, fmt.Errorf("resolve resumable campaign path: %w", err)
	}
	config.ResumeCampaign = path
	preflight, err := campaign.PreflightResume(path)
	if err != nil {
		return CampaignSpec{}, err
	}
	config.OverallTimeout = time.Duration(preflight.Plan.OverallTimeoutNanos)
	config.resumePreflight = &preflight
	return config, nil
}

func resumeConfiguration(request CampaignSpec, plan campaign.CampaignPlan) (CampaignSpec, SeedSelection, []record.Environment, []readonlymount.Mapping, target.Prepared, error) {
	if plan.RunnerBuild != request.RunnerBuild {
		return CampaignSpec{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("recorded Runner build identity %s does not match this Runner %s", plan.RunnerBuild, request.RunnerBuild)
	}
	profile := deterministicio.Default()
	if !profile.Matches(plan.IOProfile) {
		return CampaignSpec{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("recorded I/O profile identity does not match this Runner")
	}
	selection, err := ParseSeeds(plan.Selection)
	if err != nil || selection.Count() != uint64(plan.SelectionCount) {
		return CampaignSpec{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("recorded seed selection is invalid: %w", err)
	}
	mountLimits, err := readonlymount.DecodeLimits(deterministicCapturedInputLimits(plan.IOROMountLimits))
	if err != nil {
		return CampaignSpec{}, SeedSelection{}, nil, nil, target.Prepared{}, err
	}
	mounts, err := readonlymount.ParseMappings(plan.IOROMounts, "")
	if err != nil {
		return CampaignSpec{}, SeedSelection{}, nil, nil, target.Prepared{}, err
	}
	prepared := target.Prepared{
		Path: filepath.Join(request.ResumeCampaign, filepath.FromSlash(plan.Prepared.Path)), Kind: target.Kind(plan.Prepared.Target.Kind), Source: plan.Prepared.Target.Source,
		SHA256: string(plan.Prepared.Target.SHA256), Size: uint64(plan.Prepared.Target.Size), Argv: append([]string(nil), plan.Prepared.Target.Argv...),
		BuildTags: append([]string(nil), plan.Prepared.Target.BuildTags...), Adapters: cloneAdapters(plan.Prepared.Target.Adapters), Compatibility: cloneCompatibility(plan.Prepared.Target.Compatibility), BuildInfo: cloneBuildInfo(plan.Prepared.Target.BuildInfo),
		GoVersion: plan.Toolchain.GoVersion, BuildKey: plan.Toolchain.BuildKey, TargetGOOS: plan.Toolchain.TargetGOOS, TargetGOARCH: plan.Toolchain.TargetGOARCH,
		CapabilityMode: target.CapabilityMode(plan.Prepared.Target.CapabilityMode), CapabilityManifest: target.CapabilityManifestFromRecord(plan.Prepared.Target.CapabilityManifest),
	}
	if err := prepared.Verify(); err != nil {
		return CampaignSpec{}, SeedSelection{}, nil, nil, target.Prepared{}, err
	}
	if err := deterministicio.Default().VerifyAdapters(deterministicAdapters(prepared.Adapters)); err != nil {
		return CampaignSpec{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("verify recorded adapters: %w", err)
	}
	if request.Executor == nil {
		identity, err := target.ReadToolchainIdentity(request.Target.ToolchainRoot)
		if err != nil {
			return CampaignSpec{}, SeedSelection{}, nil, nil, target.Prepared{}, err
		}
		if identity.GoVersion != plan.Toolchain.GoVersion || identity.BuildKey != plan.Toolchain.BuildKey || identity.TargetGOOS != plan.Toolchain.TargetGOOS || identity.TargetGOARCH != plan.Toolchain.TargetGOARCH {
			return CampaignSpec{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("recorded toolchain identity does not match the pinned toolchain")
		}
	}
	config := CampaignSpec{
		ResumeCampaign: request.ResumeCampaign, PlanSHA256: plan.PlanSHA256, Shard: runnerCampaignShard(plan.Shard),
		Strategy: Strategy(plan.Strategy), Seeds: plan.Selection, Parallel: int(plan.Parallel), ExecutionTimeout: time.Duration(plan.ExecutionTimeoutNanos), OverallTimeout: time.Duration(plan.OverallTimeoutNanos), TerminateGrace: time.Duration(plan.TerminateGraceNanos),
		OnFailure: FailurePolicy(plan.OnFailure), FailureBudget: uint64(plan.FailureBudget), OutputLimit: uint64(plan.OutputBytes), WorldTransitionLimit: uint64(plan.WorldTransitionBytes),
		MaxExecutions: uint64(plan.MaxExecutions), MaxChoiceDepth: uint64(plan.MaxChoiceDepth), MaxForcedDecisions: uint64(plan.MaxForcedDecisions),
		MaxExplorationBytes: uint64(plan.MaxExplorationBytes), MaxExplorationResultBytes: uint64(plan.MaxExplorationResultBytes), SimulationDimensionLimits: SimulationDimensionLimits(plan.SimulationDimensionLimits),
		Artifacts: filepath.Dir(filepath.Dir(request.ResumeCampaign)), IOROMounts: append([]string(nil), plan.IOROMounts...), IOROMountLimits: mountLimits,
		Target: target.Spec{ToolchainRoot: request.Target.ToolchainRoot}, SupervisorCommand: append([]string(nil), request.SupervisorCommand...), RunnerBuild: request.RunnerBuild,
		Coverage: CoverageMode(plan.Coverage), RequiredSemanticProbes: append([]string(nil), plan.RequiredSemanticProbes...),
		KeepSuccesses: KeepSuccesses(plan.KeepSuccesses), SuccessArtifactLimit: uint64(plan.SuccessArtifactLimit), SuccessBytesLimit: uint64(plan.SuccessBytesLimit),
		Progress: request.Progress, ProgressInterval: request.ProgressInterval, Executor: request.Executor, Replayer: request.Replayer,
	}
	if config.Strategy == "" {
		config.Strategy = StrategySeed
	}
	if plan.ChoiceProfile != nil {
		config.ChoiceTraceLimit = uint64(plan.ChoiceProfile.Limit)
	}
	if plan.Guidance != nil {
		config.Guide = true
		config.Corpus = plan.Guidance.Corpus
		config.GuideSnapshotSHA256 = plan.Guidance.SnapshotSHA256
	}
	if plan.Artifacts != nil {
		config.failureArtifactLimit = uint64(plan.Artifacts.FailureArtifacts)
		config.failureBytesLimit = uint64(plan.Artifacts.FailureBytes)
	}
	return config, selection, append([]record.Environment(nil), plan.Environment...), mounts, prepared, nil
}

func runnerCampaignShard(shard *campaign.CampaignShard) CampaignShard {
	if shard == nil {
		return CampaignShard{}
	}
	return CampaignShard{Index: uint64(shard.Index), Count: uint64(shard.Count)}
}

func equalBatchPlans(left, right campaign.CampaignPlan) (bool, error) {
	leftBytes, err := canonicaljson.CanonicalJSON(left)
	if err != nil {
		return false, err
	}
	rightBytes, err := canonicaljson.CanonicalJSON(right)
	return bytes.Equal(leftBytes, rightBytes), err
}

type resumeSummaryState struct {
	summary              CampaignResult
	distinct             map[record.SHA256]string
	probes               map[string]struct{}
	choiceFeatures       map[string]struct{}
	completed            map[uint64]struct{}
	failureArtifactBytes uint64
}

func restoreResumeSummary(batchPath string, selection SeedSelection, runs []campaign.ExecutionRecord) (resumeSummaryState, error) {
	state := resumeSummaryState{
		summary:  CampaignResult{CampaignPath: batchPath, SelectionCount: selection.Count(), Attempted: uint64(len(runs))},
		distinct: make(map[record.SHA256]string), probes: make(map[string]struct{}), choiceFeatures: make(map[string]struct{}), completed: make(map[uint64]struct{}, len(runs)),
	}
	for index, run := range runs {
		ordinal := uint64(run.SelectionOrdinal)
		seed, ok := selection.SeedAt(ordinal)
		if run.Strategy == string(StrategyChoiceExploration) || run.Strategy == string(StrategySimulationExploration) {
			seed, ok = selection.SeedAt(0)
		}
		if !ok || seed != uint64(run.Seed) {
			return resumeSummaryState{}, fmt.Errorf("resumable execution %d seed does not match selection ordinal", index+1)
		}
		state.completed[ordinal] = struct{}{}
		for _, probe := range run.SemanticProbes {
			state.probes[probe] = struct{}{}
		}
		for _, feature := range run.ChoiceFeatures {
			state.choiceFeatures[feature] = struct{}{}
		}
		switch run.Domain {
		case "success":
			state.summary.Succeeded++
			if run.SuccessArtifact != nil {
				path := filepath.Join(batchPath, filepath.FromSlash(*run.SuccessArtifact))
				state.summary.RetainedSuccesses++
				state.summary.RetainedSuccessBytes += uint64(*run.SuccessArtifactBytes)
				state.summary.SuccessArtifacts = append(state.summary.SuccessArtifacts, path)
			}
		case "target", "watchdog":
			state.summary.Failures++
			if run.Domain == "watchdog" {
				state.summary.Watchdogs++
			}
			if run.Reason == "world_replay_divergence" {
				state.summary.ReplayDivergences++
			}
			if _, found := state.distinct[*run.FailureSignature]; !found {
				path := filepath.Join(batchPath, filepath.FromSlash(*run.Artifact))
				opened, err := artifact.OpenArtifact(path)
				if err != nil {
					return resumeSummaryState{}, fmt.Errorf("open resumable failure artifact %d: %w", index+1, err)
				}
				if opened.StoredBytes > ^uint64(0)-state.failureArtifactBytes {
					return resumeSummaryState{}, errors.Join(errors.New("resumable failure artifact bytes overflow"), opened.Close())
				}
				state.failureArtifactBytes += opened.StoredBytes
				if err := opened.Close(); err != nil {
					return resumeSummaryState{}, fmt.Errorf("close resumable failure artifact %d: %w", index+1, err)
				}
				state.distinct[*run.FailureSignature] = path
				state.summary.Artifacts = append(state.summary.Artifacts, path)
			}
		default:
			return resumeSummaryState{}, fmt.Errorf("resumable execution %d domain %q cannot be reused", index+1, run.Domain)
		}
	}
	state.summary.DistinctFailures = uint64(len(state.distinct))
	state.summary.failureArtifactBytes = state.failureArtifactBytes
	return state, nil
}

func sortedProbeList(probes map[string]struct{}) []string {
	result := make([]string, 0, len(probes))
	for probe := range probes {
		result = append(result, probe)
	}
	sort.Strings(result)
	return result
}
