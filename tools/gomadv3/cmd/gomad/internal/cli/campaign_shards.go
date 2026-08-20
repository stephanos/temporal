package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/runner"
)

func runCampaignShard(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad run-shard", flag.ContinueOnError)
	flags.SetOutput(stderr)
	shardValue := flags.String("shard", "", "zero-based INDEX/COUNT shard assignment")
	artifacts := flags.String("artifacts", ".gomad/artifacts", "artifact root")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	jsonOutput := flags.Bool("json", false, "emit stable JSON events")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	shard, err := parseCampaignShard(*shardValue)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	toolchain, executable, runnerBuild, err := localIdentity(*toolchainRoot)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	reporter := newExploreReporter(*jsonOutput, stdout, stderr)
	result, err := runner.RunCampaignShard(context.Background(), runner.CampaignShardSpec{
		PlanPath: flags.Arg(0), Shard: shard, Artifacts: *artifacts, ToolchainRoot: toolchain, RunnerBuild: runnerBuild,
		SupervisorCommand: []string{executable, "__supervisor"}, Progress: reporter.Progress, ProgressInterval: 5 * time.Second,
	})
	if err != nil {
		classification := classifyExploreError(err)
		if writeErr := reporter.Error(classification, err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return exploreErrorStatus(classification)
	}
	if err := reporter.Result(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if result.Failures != 0 {
		return 1
	}
	return 0
}

func parseCampaignShard(value string) (runner.CampaignShard, error) {
	indexValue, countValue, found := strings.Cut(value, "/")
	if !found || indexValue == "" || countValue == "" || strings.Contains(countValue, "/") {
		return runner.CampaignShard{}, fmt.Errorf("invalid shard %q: want zero-based INDEX/COUNT", value)
	}
	index, indexErr := strconv.ParseUint(indexValue, 10, 64)
	count, countErr := strconv.ParseUint(countValue, 10, 64)
	shard := runner.CampaignShard{Index: index, Count: count}
	if indexErr != nil || countErr != nil || strconv.FormatUint(index, 10) != indexValue || strconv.FormatUint(count, 10) != countValue {
		return runner.CampaignShard{}, fmt.Errorf("invalid shard %q: want zero-based INDEX/COUNT", value)
	}
	if err := shard.Validate(); err != nil {
		return runner.CampaignShard{}, err
	}
	return shard, nil
}

func runMergeCampaigns(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad merge", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("output", "", "merged campaign output directory")
	partial := flags.Bool("partial", false, "publish an aggregate with explicit missing ordinals")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if *output == "" || flags.NArg() < 2 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	result, err := runner.MergeCampaignShards(context.Background(), runner.CampaignMergeSpec{
		PlanPath: flags.Arg(0), Shards: append([]string(nil), flags.Args()[1:]...), Output: *output, Partial: *partial,
	})
	if err != nil {
		classification := classifyExploreError(err)
		fmt.Fprintf(stderr, "gomad: %s: merge campaign shards: %v\n", classification, err)
		return exploreErrorStatus(classification)
	}
	if *jsonOutput {
		encoded, err := json.Marshal(result)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
			return 3
		}
		return 0
	}
	if _, err := fmt.Fprintf(stdout, "gomad merge: path=%s plan=%s partial=%t shards=%d selected=%d attempted=%d succeeded=%d failures=%d watchdogs=%d cancelled=%d distinct=%d evidence=%d evidence-bytes=%d journal-segments=%d journal-bytes=%d missing=%s\n", result.Path, result.PlanSHA256, result.Partial, result.Shards, result.SelectionCount, result.Attempted, result.Succeeded, result.Failures, result.Watchdogs, result.Cancelled, result.DistinctFailures, result.RetainedEvidence, result.RetainedBytes, result.JournalSegments, result.JournalBytes, formatMissingOrdinals(result.Missing)); err != nil {
		return 3
	}
	return 0
}

func formatMissingOrdinals(ranges []runner.CampaignOrdinalRange) string {
	if len(ranges) == 0 {
		return "none"
	}
	values := make([]string, len(ranges))
	for index, value := range ranges {
		values[index] = strconv.FormatUint(value.Start, 10)
		if value.End != value.Start {
			values[index] += "-" + strconv.FormatUint(value.End, 10)
		}
	}
	return strings.Join(values, ",")
}
