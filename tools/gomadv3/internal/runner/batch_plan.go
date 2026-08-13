package runner

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
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
	return artifact.BatchPlan{
		Schema: artifact.BatchPlanSchema, Selection: config.Seeds, SelectionCount: record.Uint64String(selectionCount), Parallel: record.Uint64String(config.Parallel),
		RunTimeoutNanos: record.Uint64String(config.RunTimeout), OverallTimeoutNanos: record.Uint64String(config.OverallTimeout), TerminateGraceNanos: record.Uint64String(config.TerminateGrace),
		OnFailure: string(config.OnFailure), FailureBudget: record.Uint64String(config.FailureBudget), OutputBytes: record.Uint64String(config.OutputLimit), WorldTransitionBytes: record.Uint64String(config.WorldTransitionLimit),
		RunnerBuild: config.RunnerBuild,
		Toolchain:   record.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Prepared: artifact.PreparedTargetPlan{
			Path: preparedPath,
			Target: record.Target{
				Kind: string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
				Argv: append([]string{}, prepared.Argv...), BuildTags: append([]string{}, prepared.BuildTags...), BuildInfo: cloneBuildInfo(prepared.BuildInfo),
			},
		},
		IOProfile:   artifact.IOProfilePlan{Name: profile.Name(), ImplementationSHA256: profile.ImplementationSHA256(), InventorySHA256: profile.InventorySHA256()},
		Environment: append([]record.Environment(nil), environment...), IOROMounts: mountValues, IOROMountLimits: romount.RecordLimits(config.IOROMountLimits),
		Coverage: string(normalizedCoverage(config.Coverage)), RequiredSemanticProbes: requiredProbes,
		KeepSuccesses: string(normalizedKeepSuccesses(config.KeepSuccesses)), SuccessArtifactLimit: record.Uint64String(config.SuccessArtifactLimit), SuccessBytesLimit: record.Uint64String(config.SuccessBytesLimit),
	}, nil
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
