package runner

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func resumeRequestDefaults(config Config) (Config, error) {
	path, err := filepath.Abs(config.ResumeBatch)
	if err != nil {
		return Config{}, fmt.Errorf("resolve resumable batch path: %w", err)
	}
	config.ResumeBatch = path
	plan, err := artifact.ReadResumePlan(path)
	if err != nil {
		return Config{}, err
	}
	config.OverallTimeout = time.Duration(plan.OverallTimeoutNanos)
	return config, nil
}

func resumeConfiguration(request Config, plan artifact.BatchPlan) (Config, SeedSelection, []record.Environment, []romount.Mapping, target.Prepared, error) {
	if plan.RunnerBuild != request.RunnerBuild {
		return Config{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("recorded Runner build identity %s does not match this Runner %s", plan.RunnerBuild, request.RunnerBuild)
	}
	profile := ioprofile.Default()
	if !profile.Matches(plan.IOProfile) {
		return Config{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("recorded I/O profile identity does not match this Runner")
	}
	selection, err := ParseSeeds(plan.Selection)
	if err != nil || selection.Count() != uint64(plan.SelectionCount) {
		return Config{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("recorded seed selection is invalid: %w", err)
	}
	mountLimits, err := romount.DecodeLimits(plan.IOROMountLimits)
	if err != nil {
		return Config{}, SeedSelection{}, nil, nil, target.Prepared{}, err
	}
	mounts, err := romount.ParseMappings(plan.IOROMounts, "")
	if err != nil {
		return Config{}, SeedSelection{}, nil, nil, target.Prepared{}, err
	}
	prepared := target.Prepared{
		Path: filepath.Join(request.ResumeBatch, filepath.FromSlash(plan.Prepared.Path)), Kind: target.Kind(plan.Prepared.Target.Kind), Source: plan.Prepared.Target.Source,
		SHA256: string(plan.Prepared.Target.SHA256), Size: uint64(plan.Prepared.Target.Size), Argv: append([]string(nil), plan.Prepared.Target.Argv...),
		BuildTags: append([]string(nil), plan.Prepared.Target.BuildTags...), Adapters: cloneAdapters(plan.Prepared.Target.Adapters), Compatibility: cloneCompatibility(plan.Prepared.Target.Compatibility), BuildInfo: cloneBuildInfo(plan.Prepared.Target.BuildInfo),
		GoVersion: plan.Toolchain.GoVersion, BuildKey: plan.Toolchain.BuildKey, TargetGOOS: plan.Toolchain.TargetGOOS, TargetGOARCH: plan.Toolchain.TargetGOARCH,
	}
	if err := prepared.Verify(); err != nil {
		return Config{}, SeedSelection{}, nil, nil, target.Prepared{}, err
	}
	if err := ioprofile.Default().VerifyAdapters(prepared.Adapters); err != nil {
		return Config{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("verify recorded adapters: %w", err)
	}
	if request.Executor == nil {
		identity, err := target.ReadToolchainIdentity(request.Target.ToolchainRoot)
		if err != nil {
			return Config{}, SeedSelection{}, nil, nil, target.Prepared{}, err
		}
		if identity.GoVersion != plan.Toolchain.GoVersion || identity.BuildKey != plan.Toolchain.BuildKey || identity.TargetGOOS != plan.Toolchain.TargetGOOS || identity.TargetGOARCH != plan.Toolchain.TargetGOARCH {
			return Config{}, SeedSelection{}, nil, nil, target.Prepared{}, fmt.Errorf("recorded toolchain identity does not match the pinned toolchain")
		}
	}
	config := Config{
		ResumeBatch: request.ResumeBatch, Strategy: Strategy(plan.Strategy), Seeds: plan.Selection, Parallel: int(plan.Parallel), RunTimeout: time.Duration(plan.RunTimeoutNanos), OverallTimeout: time.Duration(plan.OverallTimeoutNanos), TerminateGrace: time.Duration(plan.TerminateGraceNanos),
		OnFailure: FailurePolicy(plan.OnFailure), FailureBudget: uint64(plan.FailureBudget), OutputLimit: uint64(plan.OutputBytes), WorldTransitionLimit: uint64(plan.WorldTransitionBytes),
		MaxRuns: uint64(plan.MaxRuns), MaxChoiceDepth: uint64(plan.MaxChoiceDepth), MaxFrontierBytes: uint64(plan.MaxFrontierBytes),
		Artifacts: filepath.Dir(filepath.Dir(request.ResumeBatch)), IOROMounts: append([]string(nil), plan.IOROMounts...), IOROMountLimits: mountLimits,
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
	return config, selection, append([]record.Environment(nil), plan.Environment...), mounts, prepared, nil
}

func equalBatchPlans(left, right artifact.BatchPlan) (bool, error) {
	leftBytes, err := record.CanonicalJSON(left)
	if err != nil {
		return false, err
	}
	rightBytes, err := record.CanonicalJSON(right)
	return bytes.Equal(leftBytes, rightBytes), err
}

type resumeSummaryState struct {
	summary        Summary
	distinct       map[record.SHA256]string
	probes         map[string]struct{}
	choiceFeatures map[string]struct{}
	completed      map[uint64]struct{}
}

func restoreResumeSummary(batchPath string, selection SeedSelection, runs []artifact.RunRecord) (resumeSummaryState, error) {
	state := resumeSummaryState{
		summary:  Summary{BatchPath: batchPath, SelectionCount: selection.Count(), Attempted: uint64(len(runs))},
		distinct: make(map[record.SHA256]string), probes: make(map[string]struct{}), choiceFeatures: make(map[string]struct{}), completed: make(map[uint64]struct{}, len(runs)),
	}
	for index, run := range runs {
		ordinal := uint64(run.SelectionOrdinal)
		seed, ok := selection.SeedAt(ordinal)
		if run.Strategy == string(StrategyChoiceFrontier) {
			seed, ok = selection.SeedAt(0)
		}
		if !ok || seed != uint64(run.Seed) {
			return resumeSummaryState{}, fmt.Errorf("resumable run %d seed does not match selection ordinal", index+1)
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
				state.distinct[*run.FailureSignature] = path
				state.summary.Artifacts = append(state.summary.Artifacts, path)
			}
		default:
			return resumeSummaryState{}, fmt.Errorf("resumable run %d domain %q cannot be reused", index+1, run.Domain)
		}
	}
	state.summary.DistinctFailures = uint64(len(state.distinct))
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
