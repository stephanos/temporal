package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	capabilityanalysis "go.temporal.io/server/tools/gomadv3/qualification/analysis"
	"go.temporal.io/server/tools/gomadv3/target"
)

const capabilityAnalysisTimeout = 30 * time.Second
const linkedCapabilityAnalysisTimeout = 2 * time.Minute
const maximumCapabilityAnalysisTimeout = 30 * time.Minute

type analyzeDependencies struct {
	toolchain        func(string) (string, error)
	identity         func(string) (target.ToolchainIdentity, error)
	workingDirectory func() (string, error)
	prepare          func(context.Context, target.Spec) (target.Spec, []deterministicio.Adapter, func() error, error)
	analyze          func(context.Context, capabilityanalysis.Spec) (capabilityanalysis.Report, error)
	review           func(context.Context, target.Spec) (target.CapabilityReview, error)
	build            func(capabilityanalysis.Input) (capabilityanalysis.Report, error)
}

type analyzeArguments struct {
	format         string
	toolchainRoot  string
	buildTags      []string
	capabilityMode target.CapabilityMode
	timeout        time.Duration
	target         targetInput
}

func runAnalyze(arguments []string, stdout, stderr io.Writer) int {
	return runAnalyzeWith(arguments, stdout, stderr, analyzeDependencies{
		toolchain: func(explicit string) (string, error) {
			root, _, _, err := localIdentity(explicit)
			return root, err
		},
		identity: target.ReadToolchainIdentity, workingDirectory: os.Getwd,
		prepare: prepareAnalysisTarget, analyze: capabilityanalysis.Analyze,
	})
}

func prepareAnalysisTarget(ctx context.Context, spec target.Spec) (target.Spec, []deterministicio.Adapter, func() error, error) {
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
	prepared, adapters, err := deterministicio.Default().PrepareBuildAdapters(spec, moduleCache)
	if err != nil {
		return target.Spec{}, nil, nil, errors.Join(err, cleanup())
	}
	return prepared, deterministicio.SelectedAdapters(adapters), cleanup, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), parsed.timeout)
	defer cancel()
	return executeAnalysis(ctx, stdout, stderr, parsed.format, spec, identity, dependencies)
}

func capabilityAnalysisTimeoutForMode(mode target.CapabilityMode) time.Duration {
	if mode == target.CapabilityModeLinked || mode == target.CapabilityModeGuarded {
		return linkedCapabilityAnalysisTimeout
	}
	return capabilityAnalysisTimeout
}

func resolveCapabilityAnalysisTimeout(mode target.CapabilityMode, requested time.Duration) (time.Duration, error) {
	if requested < 0 || requested > maximumCapabilityAnalysisTimeout {
		return 0, fmt.Errorf("analysis timeout must be non-negative and no greater than %s", maximumCapabilityAnalysisTimeout)
	}
	if requested == 0 {
		return capabilityAnalysisTimeoutForMode(mode), nil
	}
	return requested, nil
}

func parseAnalyzeArguments(arguments []string, stderr io.Writer) (analyzeArguments, int) {
	flags := flag.NewFlagSet("gomad analyze", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "text", "text or json")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	capabilityMode := flags.String("capability-mode", string(target.CapabilityModeClosure), "closure, linked, or guarded capability assessment")
	timeout := flags.Duration("timeout", 0, "analysis wall-time bound")
	var buildTags stringList
	flags.Var(&buildTags, "build-tag", "validated Go build tag")
	if err := flags.Parse(arguments); err != nil {
		return analyzeArguments{}, 2
	}
	if *format != "text" && *format != "json" {
		return analyzeArguments{}, writeCommandError(stderr, 2, "invalid analysis format %q\n", *format)
	}
	mode, err := parseCapabilityMode(*capabilityMode)
	if err != nil {
		return analyzeArguments{}, writeCommandError(stderr, 2, "%v\n", err)
	}
	resolvedTimeout, err := resolveCapabilityAnalysisTimeout(mode, *timeout)
	if err != nil {
		return analyzeArguments{}, writeCommandError(stderr, 2, "%v\n", err)
	}
	parsed, err := parseTarget(flags.Args())
	if err != nil {
		return analyzeArguments{}, writeCommandError(stderr, 2, "%v\n", err)
	}
	if parsed.kind == target.KindExec {
		return analyzeArguments{}, writeCommandError(stderr, 2, "gomad analyze requires a go-run or go-test target\n")
	}
	return analyzeArguments{format: *format, toolchainRoot: *toolchainRoot, buildTags: buildTags, capabilityMode: mode, timeout: resolvedTimeout, target: parsed}, 0
}

func resolveAnalyzeTarget(parsed analyzeArguments, stderr io.Writer, dependencies analyzeDependencies) (target.Spec, target.ToolchainIdentity, int) {
	workingDirectory, err := dependencies.workingDirectory()
	if err != nil {
		return target.Spec{}, target.ToolchainIdentity{}, writeCommandError(stderr, 3, "resolve working directory: %v\n", err)
	}
	resolvedRoot, err := dependencies.toolchain(parsed.toolchainRoot)
	if err != nil {
		return target.Spec{}, target.ToolchainIdentity{}, writeCommandError(stderr, 3, "resolve Gomad toolchain: %v\n", err)
	}
	identity, err := dependencies.identity(resolvedRoot)
	if err != nil {
		return target.Spec{}, target.ToolchainIdentity{}, writeCommandError(stderr, 3, "read Gomad toolchain identity: %v\n", err)
	}
	spec := target.Spec{Kind: parsed.target.kind, Source: parsed.target.source, Args: parsed.target.arguments, BuildTags: parsed.buildTags, WorkingDir: workingDirectory, ToolchainRoot: resolvedRoot, CapabilityMode: parsed.capabilityMode}
	return spec, identity, 0
}

func executeAnalysis(ctx context.Context, stdout, stderr io.Writer, format string, spec target.Spec, identity target.ToolchainIdentity, dependencies analyzeDependencies) (status int) {
	adapters := []deterministicio.Adapter{}
	var err error
	if dependencies.prepare != nil {
		var cleanup func() error
		spec, adapters, cleanup, err = dependencies.prepare(ctx, spec)
		if err != nil {
			if deterministicio.IsInvalidBuildAdapterConfiguration(err) {
				return writeCommandError(stderr, 2, "prepare capability analysis: %v\n", err)
			}
			return writeCommandError(stderr, 3, "prepare capability analysis: %v\n", err)
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
	var report capabilityanalysis.Report
	if dependencies.analyze != nil {
		report, err = dependencies.analyze(ctx, capabilityanalysis.Spec{
			Target: spec, Toolchain: identity, IOProfile: deterministicio.Default(), Adapters: adapters,
		})
	} else {
		var review target.CapabilityReview
		review, err = dependencies.review(ctx, spec)
		if err == nil {
			report, err = dependencies.build(capabilityanalysis.Input{Spec: spec, Review: review, Toolchain: identity, IOProfile: deterministicio.Default(), Adapters: adapters})
		}
	}
	if err != nil {
		return reportAnalyzeError(stderr, err)
	}
	return writeAnalysis(stdout, stderr, format, report)
}

func reportAnalyzeError(stderr io.Writer, err error) int {
	var capacity *target.UnsupportedCapabilityCapacityError
	if errors.As(err, &capacity) {
		return writeCommandError(stderr, 1, "analyze target capabilities: %v\n", err)
	}
	if target.IsInvalidCapabilityReview(err) {
		return writeCommandError(stderr, 2, "analyze target capabilities: %v\n", err)
	}
	return writeCommandError(stderr, 3, "analyze target capabilities: %v\n", err)
}

func writeAnalysis(stdout, stderr io.Writer, format string, report capabilityanalysis.Report) int {
	if format == "json" {
		encoded, err := canonicaljson.CanonicalJSON(report)
		if err != nil {
			return writeCommandError(stderr, 3, "encode capability analysis: %v\n", err)
		}
		if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
			return writeCommandError(stderr, 3, "write capability analysis: %v\n", err)
		}
	} else {
		if _, err := fmt.Fprint(stdout, capabilityanalysis.FormatText(report)); err != nil {
			return writeCommandError(stderr, 3, "write capability analysis: %v\n", err)
		}
	}
	if report.Classification == capabilityanalysis.ClassificationUnsupported {
		return 1
	}
	return 0
}

func writeCommandError(output io.Writer, status int, format string, arguments ...any) int {
	if _, err := fmt.Fprintf(output, format, arguments...); err != nil {
		return 3
	}
	return status
}
