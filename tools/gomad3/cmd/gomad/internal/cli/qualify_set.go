package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	qualificationset "go.temporal.io/server/tools/gomad3/qualification/set"
)

type qualifySetDependencies struct {
	executable func() (string, error)
	load       func(string) (qualificationset.Manifest, error)
	run        func(context.Context, qualificationset.Spec) (qualificationset.Report, error)
}

func runQualifySet(arguments []string, stdout, stderr io.Writer) int {
	return runQualifySetWith(arguments, stdout, stderr, qualifySetDependencies{
		executable: os.Executable, load: qualificationset.LoadManifest, run: qualificationset.Run,
	})
}

func runQualifySetWith(arguments []string, stdout, stderr io.Writer, dependencies qualifySetDependencies) int {
	flags := flag.NewFlagSet("gomad qualify-set", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifestPath := flags.String("manifest", "", "qualification manifest")
	workingDirectory := flags.String("working-dir", "", "target module directory")
	artifacts := flags.String("artifacts", ".gomad/qualification", "qualification artifact root")
	output := flags.String("output", ".gomad/qualification-set.json", "qualification set report")
	format := flags.String("format", "text", "text or json")
	check := flags.Bool("check", false, "validate the manifest without executing targets")
	pruneQualified := flags.Bool("prune-qualified-artifacts", false, "delete each qualified seed's retained Campaigns once its evidence is final")
	shardValue := flags.String("shard", "", "zero-based INDEX/COUNT subset of the manifest's workloads, merged later with merge-set")
	minimumFree := byteSize(qualificationset.DefaultMinimumFreeBytes)
	flags.Var(&minimumFree, "min-free-bytes", "stop before a seed starts on an artifact volume with less free space")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}
	if *manifestPath == "" || *workingDirectory == "" || (*format != "text" && *format != "json") {
		return writeCommandError(stderr, 2, "qualify-set requires --manifest, --working-dir, and --format=text|json\n")
	}
	var shard qualificationset.Shard
	if *shardValue != "" {
		index, count, err := parseShardAssignment(*shardValue)
		if err != nil {
			return writeCommandError(stderr, 2, "%v\n", err)
		}
		shard = qualificationset.Shard{Index: index, Count: count}
		if err := shard.Validate(); err != nil {
			return writeCommandError(stderr, 2, "%v\n", err)
		}
	}
	manifest, err := dependencies.load(*manifestPath)
	if err != nil {
		return writeCommandError(stderr, 2, "load qualification manifest: %v\n", err)
	}
	selected, err := shard.Select(manifest)
	if err != nil {
		return writeCommandError(stderr, 2, "%v\n", err)
	}
	if *check {
		if *format == "json" {
			encoded, encodeErr := canonicaljson.CanonicalJSON(struct {
				Schema    string `json:"schema"`
				Name      string `json:"name"`
				Workloads uint64 `json:"workloads"`
			}{"gomad3.qualification-set-check/v1", manifest.Name, uint64(len(selected))})
			if encodeErr != nil {
				return writeCommandError(stderr, 3, "encode qualification manifest result: %v\n", encodeErr)
			}
			if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
				return 3
			}
		} else if _, err := fmt.Fprintf(stdout, "qualification manifest: name=%s workloads=%d\n", manifest.Name, len(selected)); err != nil {
			return 3
		}
		return 0
	}
	executable, err := dependencies.executable()
	if err != nil {
		return writeCommandError(stderr, 3, "resolve gomad executable: %v\n", err)
	}
	report, runErr := dependencies.run(context.Background(), qualificationset.Spec{
		ManifestPath: *manifestPath, GomadPath: executable, WorkingDir: *workingDirectory,
		ArtifactRoot: *artifacts, OutputPath: *output, Shard: shard, PruneQualifiedArtifacts: *pruneQualified, MinimumFreeBytes: uint64(minimumFree),
	})
	if err := writeQualificationSetResult(stdout, *format, report); err != nil {
		return writeCommandError(stderr, 3, "write qualification set result: %v\n", err)
	}
	if runErr == nil {
		return 0
	}
	status, message := classifyQualificationSetError(report, runErr)
	if message == "" {
		return status
	}
	return writeCommandError(stderr, status, "%s: %v\n", message, runErr)
}

type mergeSetDependencies struct {
	merge func(context.Context, qualificationset.MergeSpec) (qualificationset.Report, error)
}

func runMergeSet(arguments []string, stdout, stderr io.Writer) int {
	return runMergeSetWith(arguments, stdout, stderr, mergeSetDependencies{merge: qualificationset.Merge})
}

// runMergeSetWith keeps qualify-set's status contract: 0 when every merged
// expectation matched, 1 when the merged report retains a mismatch, 2 when the
// shards or manifest are invalid, and 3 when the report cannot be written.
func runMergeSetWith(arguments []string, stdout, stderr io.Writer, dependencies mergeSetDependencies) int {
	flags := flag.NewFlagSet("gomad merge-set", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifestPath := flags.String("manifest", "", "qualification manifest the shards ran")
	output := flags.String("output", ".gomad/qualification-set.json", "merged qualification set report")
	format := flags.String("format", "text", "text or json")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if *manifestPath == "" || flags.NArg() == 0 || (*format != "text" && *format != "json") {
		return writeCommandError(stderr, 2, "merge-set requires --manifest, --format=text|json, and at least one shard report\n")
	}
	report, mergeErr := dependencies.merge(context.Background(), qualificationset.MergeSpec{
		ManifestPath: *manifestPath, ShardReports: append([]string(nil), flags.Args()...), OutputPath: *output,
	})
	if mergeErr != nil && qualificationset.IsInvalidReport(mergeErr) {
		return writeCommandError(stderr, 2, "merge qualification set shards: %v\n", mergeErr)
	}
	if err := writeQualificationSetResult(stdout, *format, report); err != nil {
		return writeCommandError(stderr, 3, "write qualification set result: %v\n", err)
	}
	if mergeErr == nil {
		return 0
	}
	var mismatch *qualificationset.ExpectationError
	if errors.As(mergeErr, &mismatch) {
		return 1
	}
	return writeCommandError(stderr, 3, "merge qualification set shards: %v\n", mergeErr)
}

func classifyQualificationSetError(report qualificationset.Report, runErr error) (int, string) {
	for _, workload := range report.Workloads {
		if workload.Classification == "invalid_input" || workload.AnalysisError == "invalid_input" {
			return 2, "invalid qualification set workload"
		}
	}
	if errors.Is(runErr, qualificationset.ErrLowFreeSpace) {
		return 3, "qualification set stopped"
	}
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) || report.InfrastructureErrors != 0 {
		return 3, "qualification set infrastructure failure"
	}
	var mismatch *qualificationset.ExpectationError
	if errors.As(runErr, &mismatch) {
		return 1, ""
	}
	if report.Schema == "" {
		return 2, "invalid qualification set input"
	}
	return 3, "qualification set failure"
}

func writeQualificationSetResult(output io.Writer, format string, report qualificationset.Report) error {
	if format == "json" {
		encoded, err := canonicaljson.CanonicalJSON(report)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "%s\n", encoded)
		return err
	}
	_, err := fmt.Fprintf(output, "qualification set: name=%s expectations-met=%t supported=%d unsupported=%d failed=%d infrastructure-errors=%d completed=%d/%d\n", report.Name, report.ExpectationsMet, report.Supported, report.Unsupported, report.Failed, report.InfrastructureErrors, report.Completed, report.Selected)
	return err
}
