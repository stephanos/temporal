package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/capabilityanalysis"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const capabilityAnalysisTimeout = 30 * time.Second

type analyzeDependencies struct {
	toolchain        func(string) (string, error)
	identity         func(string) (target.ToolchainIdentity, error)
	workingDirectory func() (string, error)
	prepare          func(context.Context, target.Spec) (target.Spec, []record.TargetAdapter, func() error, error)
	review           func(context.Context, target.Spec) (target.CapabilityReview, error)
	build            func(capabilityanalysis.Input) (capabilityanalysis.Report, error)
}

type analyzeArguments struct {
	format        string
	toolchainRoot string
	buildTags     []string
	target        targetInput
}

func runAnalyze(arguments []string, stdout, stderr io.Writer) int {
	return runAnalyzeWith(arguments, stdout, stderr, analyzeDependencies{
		toolchain: func(explicit string) (string, error) {
			root, _, _, err := localIdentity(explicit)
			return root, err
		},
		identity: target.ReadToolchainIdentity, workingDirectory: os.Getwd,
		prepare: prepareAnalysisTarget, review: target.ReviewCapabilities, build: capabilityanalysis.Build,
	})
}

func prepareAnalysisTarget(ctx context.Context, spec target.Spec) (target.Spec, []record.TargetAdapter, func() error, error) {
	root, err := os.MkdirTemp("", "gomadv3-analysis-")
	if err != nil {
		return target.Spec{}, nil, nil, fmt.Errorf("create analysis preparation directory: %w", err)
	}
	cleanup := func() error { return os.RemoveAll(root) }
	if err := os.Chmod(root, 0o700); err != nil {
		return target.Spec{}, nil, nil, errors.Join(fmt.Errorf("make analysis preparation directory private: %w", err), cleanup())
	}
	spec.PreparationRoot = root
	moduleCache, err := target.ReadModuleCache(ctx, spec.ToolchainRoot)
	if err != nil {
		return target.Spec{}, nil, nil, errors.Join(err, cleanup())
	}
	prepared, adapters, err := ioprofile.Default().PrepareBuildAdapters(spec, moduleCache)
	if err != nil {
		return target.Spec{}, nil, nil, errors.Join(err, cleanup())
	}
	return prepared, ioprofile.RecordAdapters(adapters), cleanup, nil
}

func runAnalyzeWith(arguments []string, stdout, stderr io.Writer, dependencies analyzeDependencies) (status int) {
	parsed, status := parseAnalyzeArguments(arguments, stderr)
	if status != 0 {
		return status
	}
	spec, identity, status := resolveAnalyzeTarget(parsed, stderr, dependencies)
	if status != 0 {
		return status
	}
	ctx, cancel := context.WithTimeout(context.Background(), capabilityAnalysisTimeout)
	defer cancel()
	return executeAnalysis(ctx, stdout, stderr, parsed.format, spec, identity, dependencies)
}

func parseAnalyzeArguments(arguments []string, stderr io.Writer) (analyzeArguments, int) {
	flags := flag.NewFlagSet("gomad analyze", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "text", "text or json")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	var buildTags stringList
	flags.Var(&buildTags, "build-tag", "validated Go build tag")
	if err := flags.Parse(arguments); err != nil {
		return analyzeArguments{}, 2
	}
	if *format != "text" && *format != "json" {
		return analyzeArguments{}, writeAnalyzeError(stderr, 2, "invalid analysis format %q\n", *format)
	}
	parsed, err := parseTarget(flags.Args())
	if err != nil {
		return analyzeArguments{}, writeAnalyzeError(stderr, 2, "%v\n", err)
	}
	if parsed.kind == target.KindExec {
		return analyzeArguments{}, writeAnalyzeError(stderr, 2, "gomad analyze requires a go-run or go-test target\n")
	}
	return analyzeArguments{format: *format, toolchainRoot: *toolchainRoot, buildTags: buildTags, target: parsed}, 0
}

func resolveAnalyzeTarget(parsed analyzeArguments, stderr io.Writer, dependencies analyzeDependencies) (target.Spec, target.ToolchainIdentity, int) {
	workingDirectory, err := dependencies.workingDirectory()
	if err != nil {
		return target.Spec{}, target.ToolchainIdentity{}, writeAnalyzeError(stderr, 3, "resolve working directory: %v\n", err)
	}
	resolvedRoot, err := dependencies.toolchain(parsed.toolchainRoot)
	if err != nil {
		return target.Spec{}, target.ToolchainIdentity{}, writeAnalyzeError(stderr, 3, "resolve Gomad toolchain: %v\n", err)
	}
	identity, err := dependencies.identity(resolvedRoot)
	if err != nil {
		return target.Spec{}, target.ToolchainIdentity{}, writeAnalyzeError(stderr, 3, "read Gomad toolchain identity: %v\n", err)
	}
	spec := target.Spec{Kind: parsed.target.kind, Source: parsed.target.source, Args: parsed.target.arguments, BuildTags: parsed.buildTags, WorkingDir: workingDirectory, ToolchainRoot: resolvedRoot}
	return spec, identity, 0
}

func executeAnalysis(ctx context.Context, stdout, stderr io.Writer, format string, spec target.Spec, identity target.ToolchainIdentity, dependencies analyzeDependencies) (status int) {
	adapters := []record.TargetAdapter{}
	if dependencies.prepare != nil {
		var cleanup func() error
		var err error
		spec, adapters, cleanup, err = dependencies.prepare(ctx, spec)
		if err != nil {
			if ioprofile.IsInvalidBuildAdapterConfiguration(err) {
				return writeAnalyzeError(stderr, 2, "prepare capability analysis: %v\n", err)
			}
			return writeAnalyzeError(stderr, 3, "prepare capability analysis: %v\n", err)
		}
		if cleanup != nil {
			defer func() {
				if cleanupErr := cleanup(); cleanupErr != nil {
					if _, writeErr := fmt.Fprintf(stderr, "clean capability analysis preparation: %v\n", cleanupErr); writeErr != nil {
						status = 3
					}
				}
			}()
		}
	}
	review, err := dependencies.review(ctx, spec)
	if err != nil {
		return reportAnalyzeReviewError(stderr, err)
	}
	return writeAnalysis(stdout, stderr, format, dependencies.build, capabilityanalysis.Input{Spec: spec, Review: review, Toolchain: identity, IOProfile: ioprofile.Default(), Adapters: adapters})
}

func reportAnalyzeReviewError(stderr io.Writer, err error) int {
	if target.IsInvalidCapabilityReview(err) {
		return writeAnalyzeError(stderr, 2, "analyze target capabilities: %v\n", err)
	}
	return writeAnalyzeError(stderr, 3, "analyze target capabilities: %v\n", err)
}

func writeAnalysis(stdout, stderr io.Writer, format string, build func(capabilityanalysis.Input) (capabilityanalysis.Report, error), input capabilityanalysis.Input) int {
	report, err := build(input)
	if err != nil {
		return writeAnalyzeError(stderr, 3, "build capability analysis: %v\n", err)
	}
	if format == "json" {
		encoded, err := record.CanonicalJSON(report)
		if err != nil {
			return writeAnalyzeError(stderr, 3, "encode capability analysis: %v\n", err)
		}
		if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
			return writeAnalyzeError(stderr, 3, "write capability analysis: %v\n", err)
		}
	} else {
		if _, err := fmt.Fprint(stdout, capabilityanalysis.FormatText(report)); err != nil {
			return writeAnalyzeError(stderr, 3, "write capability analysis: %v\n", err)
		}
	}
	if report.Classification == capabilityanalysis.ClassificationUnsupported {
		return 1
	}
	return 0
}

func writeAnalyzeError(output io.Writer, status int, format string, arguments ...any) int {
	if _, err := fmt.Fprintf(output, format, arguments...); err != nil {
		return 3
	}
	return status
}
