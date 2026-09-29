package set

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
)

type MergeSpec struct {
	ManifestPath string
	ShardReports []string
	OutputPath   string
}

// Merge combines the shard reports one manifest's Shard runs published into
// the report a whole run would have published. Every shard must come from the
// same manifest, module, platform, toolchain, I/O profile, seeds, and pruning
// choice, and together the shards must cover each manifest workload exactly
// once; anything else is invalid input, never a partial aggregate. The merged
// report is written before an expectation mismatch is reported, as Run does.
func Merge(ctx context.Context, spec MergeSpec) (Report, error) {
	if spec.OutputPath == "" || len(spec.ShardReports) == 0 {
		return Report{}, invalidReport(errors.New("qualification set merge requires a manifest, shard reports, and an output path"))
	}
	manifest, err := LoadManifest(spec.ManifestPath)
	if err != nil {
		return Report{}, invalidReport(err)
	}
	digest, err := manifestDigest(manifest)
	if err != nil {
		return Report{}, err
	}
	var merged Report
	workloads := make(map[string]WorkloadReport, len(manifest.Suites))
	for index, path := range spec.ShardReports {
		shard, err := OpenReport(path)
		if err != nil {
			return Report{}, invalidReport(fmt.Errorf("shard report %s: %w", path, err))
		}
		if shard.ManifestSHA256 != digest {
			return Report{}, invalidReport(fmt.Errorf("shard report %s was produced from another manifest", path))
		}
		if index == 0 {
			merged = shard
		} else if !sameSetRun(merged, shard) {
			return Report{}, invalidReport(fmt.Errorf("shard report %s was produced by another run configuration than %s", path, spec.ShardReports[0]))
		}
		for _, workload := range shard.Workloads {
			if _, duplicate := workloads[workload.ID]; duplicate {
				return Report{}, invalidReport(fmt.Errorf("shard report %s repeats workload %s", path, workload.ID))
			}
			workloads[workload.ID] = workload
		}
	}
	merged.Selected = uint64(len(manifest.Suites))
	merged.Workloads = make([]WorkloadReport, 0, len(manifest.Suites))
	var missing []string
	for _, suite := range manifest.Suites {
		workload, found := workloads[suite.ID]
		if !found {
			missing = append(missing, suite.ID)
			continue
		}
		merged.Workloads = append(merged.Workloads, workload)
		delete(workloads, suite.ID)
	}
	if len(missing) != 0 {
		return Report{}, invalidReport(fmt.Errorf("shard reports omit %d manifest workloads: %s", len(missing), strings.Join(missing, ", ")))
	}
	if len(workloads) != 0 {
		return Report{}, invalidReport(fmt.Errorf("shard reports include workloads the manifest does not declare: %s", strings.Join(slices.Sorted(maps.Keys(workloads)), ", ")))
	}
	finalizeSetReportCounters(&merged)
	failed := make([]string, 0, len(merged.Workloads))
	for _, workload := range merged.Workloads {
		completed := workload.Analysis != nil && (workload.Analysis.Classification == capabilityanalysis.ClassificationUnsupported || len(workload.Seeds) == len(merged.Seeds))
		if !completed || !workload.ExpectationMet {
			failed = append(failed, workload.ID)
		}
	}
	merged.ExpectationsMet = merged.Completed == merged.Selected && len(failed) == 0
	if err := writeReport(ctx, spec.OutputPath, merged); err != nil {
		return merged, err
	}
	if !merged.ExpectationsMet {
		return merged, &ExpectationError{Workloads: failed}
	}
	return merged, nil
}

// sameSetRun holds when two shard reports describe the same manifest run:
// everything a report records about the run except the workloads it owns and
// the counters derived from them.
func sameSetRun(left, right Report) bool {
	left.Workloads, right.Workloads = nil, nil
	left.Selected, right.Selected = 0, 0
	left.ExpectationsMet, right.ExpectationsMet = false, false
	finalizeSetReportCounters(&left)
	finalizeSetReportCounters(&right)
	return reflect.DeepEqual(left, right)
}
