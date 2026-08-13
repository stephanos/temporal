package runner

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func batchPlan(config Config, journal *artifact.BatchJournal, prepared target.Prepared, environment []record.Environment, mounts []romount.Mapping, selectionCount uint64) (artifact.BatchPlan, error) {
	preparedPath, err := filepath.Rel(journal.Path(), prepared.Path)
	if err != nil {
		return artifact.BatchPlan{}, fmt.Errorf("make prepared target path relative: %w", err)
	}
	preparedPath = filepath.ToSlash(preparedPath)
	if strings.HasPrefix(preparedPath, "../") || preparedPath == ".." {
		return artifact.BatchPlan{}, fmt.Errorf("prepared target is outside its batch")
	}
	profile := ioprofile.Default()
	requiredProbes := append([]string(nil), config.RequiredSemanticProbes...)
	sort.Strings(requiredProbes)
	mountValues := make([]string, len(mounts))
	for index, mount := range mounts {
		mountValues[index] = mount.Source + "=" + strings.TrimPrefix(mount.Target, "/")
	}
	plan := artifact.BatchPlan{
		Schema: artifact.BatchPlanSchema, Selection: config.Seeds, SelectionCount: record.Uint64String(selectionCount), Parallel: record.Uint64String(config.Parallel),
		RunTimeoutNanos: record.Uint64String(config.RunTimeout), OverallTimeoutNanos: record.Uint64String(config.OverallTimeout), TerminateGraceNanos: record.Uint64String(config.TerminateGrace),
		OnFailure: string(config.OnFailure), FailureBudget: record.Uint64String(config.FailureBudget), OutputBytes: record.Uint64String(config.OutputLimit), WorldTransitionBytes: record.Uint64String(config.WorldTransitionLimit),
		RunnerBuild: config.RunnerBuild,
		Toolchain:   prepared.RecordToolchain(),
		Prepared: artifact.PreparedTargetPlan{
			Path: preparedPath, Target: prepared.RecordTarget(),
		},
		IOProfile:   profile.Identity(),
		Environment: append([]record.Environment(nil), environment...), IOROMounts: mountValues, IOROMountLimits: romount.RecordLimits(config.IOROMountLimits),
		Coverage: string(normalizedCoverage(config.Coverage)), RequiredSemanticProbes: requiredProbes,
		KeepSuccesses: string(normalizedKeepSuccesses(config.KeepSuccesses)), SuccessArtifactLimit: record.Uint64String(config.SuccessArtifactLimit), SuccessBytesLimit: record.Uint64String(config.SuccessBytesLimit),
	}
	if config.ChoiceTraceLimit != 0 {
		implementation, err := choicewire.ImplementationIdentity(prepared.BuildKey)
		if err != nil {
			return artifact.BatchPlan{}, fmt.Errorf("derive choice profile implementation identity: %w", err)
		}
		plan.ChoiceProfile = &artifact.ChoiceProfilePlan{
			Name: choicewire.Profile, ImplementationSHA256: record.SHA256FromSum(implementation), Limit: record.Uint64String(config.ChoiceTraceLimit),
		}
	}
	if config.Guide {
		plan.Guidance = &artifact.GuidancePlan{Corpus: config.Corpus, SnapshotSHA256: config.GuideSnapshotSHA256}
	}
	return plan, nil
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
