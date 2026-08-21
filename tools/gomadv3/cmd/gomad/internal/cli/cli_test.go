package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/qualification"
	"go.temporal.io/server/tools/gomadv3/runner"
	"go.temporal.io/server/tools/gomadv3/target"
)

func TestByteSizeFlagParsesBinaryUnitsCanonically(t *testing.T) {
	for input, want := range map[string]uint64{"1": 1, "8KiB": 8 << 10, "8MiB": 8 << 20, "2GiB": 2 << 30} {
		var value byteSize
		if err := value.Set(input); err != nil {
			t.Fatalf("Set(%q): %v", input, err)
		}
		if uint64(value) != want {
			t.Fatalf("Set(%q) = %d, want %d", input, value, want)
		}
	}
	for _, input := range []string{"", "0", "1MB", "-1", "01", "18446744073709551615GiB"} {
		var value byteSize
		if err := value.Set(input); err == nil {
			t.Fatalf("Set(%q) succeeded", input)
		}
	}
}

func TestRunQualifySetUsesCurrentExecutableAndPublicPaths(t *testing.T) {
	var observed qualification.SuiteSpec
	dependencies := qualifySetDependencies{
		executable: func() (string, error) { return "/bin/gomad", nil },
		load: func(string) (qualification.SuiteManifest, error) {
			return qualification.SuiteManifest{Schema: qualification.SuiteManifestSchema, Name: "test-set", Suites: []qualification.Workload{{}}}, nil
		},
		run: func(_ context.Context, config qualification.SuiteSpec) (qualification.SuiteReport, error) {
			observed = config
			return publicSetReport(), nil
		},
	}
	var stdout, stderr bytes.Buffer
	status := runQualifySetWith([]string{"--manifest", "/corpus.json", "--working-dir", "/repo", "--artifacts", "/artifacts", "--output", "/report.json", "--format", "json"}, &stdout, &stderr, dependencies)
	if status != 0 || stderr.Len() != 0 || observed.GomadPath != "/bin/gomad" || observed.ManifestPath != "/corpus.json" || observed.WorkingDir != "/repo" || observed.ArtifactRoot != "/artifacts" || observed.OutputPath != "/report.json" || !strings.Contains(stdout.String(), `"schema":"gomadv3.qualification-set-report/v6"`) {
		t.Fatalf("status=%d config=%#v stdout=%q stderr=%q", status, observed, stdout.String(), stderr.String())
	}
}

func TestRunCompareSupportMapsReviewAndIncomparableStatuses(t *testing.T) {
	baseline := publicSetReport()
	candidate := publicSetReport()
	candidate.Toolchain.BoundaryManifestSHA256 = evidence.HashBytes([]byte("changed"))
	dependencies := compareSupportDependencies{
		open: func(path string) (qualification.SuiteReport, error) {
			if path == "/baseline.json" {
				return baseline, nil
			}
			return candidate, nil
		},
		compare: qualification.Compare,
	}
	var stdout, stderr bytes.Buffer
	status := runCompareSupportWith([]string{"--baseline", "/baseline.json", "--candidate", "/candidate.json", "--format", "json"}, &stdout, &stderr, dependencies)
	if status != 1 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"review_required":true`) {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}

	baseline.Dimensions.PortableV3 = false
	dependencies.open = func(path string) (qualification.SuiteReport, error) {
		if path == "/baseline.json" {
			return baseline, nil
		}
		return candidate, nil
	}
	stdout.Reset()
	status = runCompareSupportWith([]string{"--baseline", "/baseline.json", "--candidate", "/candidate.json"}, &stdout, &stderr, dependencies)
	if status != 2 || !strings.Contains(stdout.String(), "incomparable") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestRunCompareSupportDistinguishesInvalidReportsFromIOFailures(t *testing.T) {
	root := t.TempDir()
	invalid := filepath.Join(root, "invalid.json")
	if err := os.WriteFile(invalid, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		baseline string
		status   int
	}{
		{name: "invalid report", baseline: invalid, status: 2},
		{name: "missing report", baseline: filepath.Join(root, "missing.json"), status: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := runCompareSupport([]string{"--baseline", test.baseline, "--candidate", invalid}, &stdout, &stderr)
			if status != test.status || stderr.Len() == 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func publicSetReport() qualification.SuiteReport {
	return qualification.SuiteReport{
		Schema: qualification.SuiteReportSchema, Name: "test-set", Description: "public fixture",
		ManifestSHA256: evidence.HashBytes([]byte("manifest")),
		Module:         qualification.ModuleIdentity{Path: "example.com/target", GoModSHA256: evidence.HashBytes([]byte("go.mod"))},
		Platform:       qualification.PlatformIdentity{GOOS: "darwin", GOARCH: "arm64"},
		Toolchain: qualification.AnalysisToolchain{
			GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64",
			BoundaryManifestVersion: "boundary-v1", BoundaryManifestSHA256: evidence.HashBytes([]byte("boundary")),
		},
		IOProfile:       deterministicio.Contract{Name: "io", ImplementationSHA256: deterministicio.Digest(evidence.HashBytes([]byte("io"))), InventorySHA256: deterministicio.Digest(evidence.HashBytes([]byte("inventory")))},
		Dimensions:      qualification.EvidenceDimensions{PortableV3: true, Analysis: true, Replay: true, Choice: true},
		ExpectationsMet: true, Suites: []qualification.WorkloadReport{},
	}
}

func TestResolveExploreSeedsSupportsCountWithoutAmbiguity(t *testing.T) {
	for _, test := range []struct {
		name               string
		seeds              string
		count              uint64
		seedsSet, countSet bool
		want               string
		wantError          bool
	}{
		{name: "default", seeds: "1", want: "1"},
		{name: "explicit seeds", seeds: "7,9", seedsSet: true, want: "7,9"},
		{name: "one", seeds: "1", count: 1, countSet: true, want: "0"},
		{name: "three", seeds: "1", count: 3, countSet: true, want: "0-2"},
		{name: "zero", seeds: "1", countSet: true, wantError: true},
		{name: "conflict", seeds: "7", count: 3, seedsSet: true, countSet: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveExploreSeeds(test.seeds, test.count, test.seedsSet, test.countSet)
			if (err != nil) != test.wantError || got != test.want {
				t.Fatalf("resolveExploreSeeds() = %q, %v, want %q, error=%t", got, err, test.want, test.wantError)
			}
		})
	}
}

func TestResolveExploreStrategyRequiresExplicitBoundedSingleSeedFrontier(t *testing.T) {
	valid := exploreStrategyOptions{
		Value: "choice-frontier", Seeds: "7", MaxRuns: 8, MaxChoiceDepth: 4, MaxFrontierBytes: 1 << 20,
		MaxRunsSet: true, MaxChoiceDepthSet: true, MaxFrontierBytesSet: true,
	}
	strategy, choices, err := resolveExploreStrategy(valid)
	if err != nil {
		t.Fatal(err)
	}
	if strategy != runner.StrategyChoiceFrontier || !choices {
		t.Fatalf("resolveExploreStrategy() = %q, %t", strategy, choices)
	}

	for _, test := range []struct {
		name      string
		configure func(*exploreStrategyOptions)
		want      string
	}{
		{name: "count", configure: func(options *exploreStrategyOptions) { options.CountSet = true }, want: "does not accept --count"},
		{name: "multiple seeds", configure: func(options *exploreStrategyOptions) { options.Seeds = "7-8" }, want: "exactly one base seed"},
		{name: "guidance", configure: func(options *exploreStrategyOptions) { options.Guide = true }, want: "does not support --guide"},
		{name: "missing max runs", configure: func(options *exploreStrategyOptions) { options.MaxRunsSet = false }, want: "--max-runs"},
		{name: "zero max runs", configure: func(options *exploreStrategyOptions) { options.MaxRuns = 0 }, want: "--max-runs"},
		{name: "missing max depth", configure: func(options *exploreStrategyOptions) { options.MaxChoiceDepthSet = false }, want: "--max-choice-depth"},
		{name: "missing frontier bytes", configure: func(options *exploreStrategyOptions) { options.MaxFrontierBytesSet = false }, want: "--max-frontier-bytes"},
		{name: "unknown", configure: func(options *exploreStrategyOptions) { options.Value = "random" }, want: "unknown exploration strategy"},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := valid
			test.configure(&options)
			if _, _, err := resolveExploreStrategy(options); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("resolveExploreStrategy() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestResolveExploreStrategyRequiresExplicitBoundedCombinedFrontier(t *testing.T) {
	valid := exploreStrategyOptions{
		Value: "combined-frontier", Seeds: "7", MaxRuns: 8, MaxForcedDecisions: 4,
		MaxFrontierBytes: 1 << 20, MaxExplorationResultBytes: 1 << 20,
		CombinedDimensionLimits: runner.CombinedDimensionLimits{Runtime: 4, Scenario: 4, Network: 4, Storage: 4, Fault: 4, Crash: 4},
		MaxRunsSet:              true, MaxForcedDecisionsSet: true, MaxFrontierBytesSet: true, MaxExplorationResultBytesSet: true,
		RuntimeLimitSet: true, ScenarioLimitSet: true, NetworkLimitSet: true, StorageLimitSet: true, FaultLimitSet: true, CrashLimitSet: true,
	}
	strategy, choices, err := resolveExploreStrategy(valid)
	if err != nil {
		t.Fatal(err)
	}
	if strategy != runner.StrategyCombinedFrontier || !choices {
		t.Fatalf("resolveExploreStrategy() = %q, %t", strategy, choices)
	}

	for _, test := range []struct {
		name      string
		configure func(*exploreStrategyOptions)
		want      string
	}{
		{name: "count", configure: func(options *exploreStrategyOptions) { options.CountSet = true }, want: "does not accept --count"},
		{name: "multiple seeds", configure: func(options *exploreStrategyOptions) { options.Seeds = "7-8" }, want: "exactly one base seed"},
		{name: "guidance", configure: func(options *exploreStrategyOptions) { options.Guide = true }, want: "does not support --guide"},
		{name: "choice depth", configure: func(options *exploreStrategyOptions) { options.MaxChoiceDepthSet = true; options.MaxChoiceDepth = 1 }, want: "does not accept --max-choice-depth"},
		{name: "missing max runs", configure: func(options *exploreStrategyOptions) { options.MaxRunsSet = false }, want: "--max-runs"},
		{name: "missing forced decisions", configure: func(options *exploreStrategyOptions) { options.MaxForcedDecisionsSet = false }, want: "--max-forced-decisions"},
		{name: "missing frontier bytes", configure: func(options *exploreStrategyOptions) { options.MaxFrontierBytesSet = false }, want: "--max-frontier-bytes"},
		{name: "missing result bytes", configure: func(options *exploreStrategyOptions) { options.MaxExplorationResultBytesSet = false }, want: "--max-exploration-result-bytes"},
		{name: "missing runtime bound", configure: func(options *exploreStrategyOptions) { options.RuntimeLimitSet = false }, want: "--max-runtime-decisions"},
		{name: "zero crash bound", configure: func(options *exploreStrategyOptions) { options.CombinedDimensionLimits.Crash = 0 }, want: "--max-crash-decisions"},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := valid
			test.configure(&options)
			if _, _, err := resolveExploreStrategy(options); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("resolveExploreStrategy() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestResolveExploreStrategyRejectsFrontierBoundsForSeeds(t *testing.T) {
	_, _, err := resolveExploreStrategy(exploreStrategyOptions{Value: "seed", Seeds: "7", MaxRuns: 1, MaxRunsSet: true})
	if err == nil || !strings.Contains(err.Error(), "require --strategy=choice-frontier") {
		t.Fatalf("resolveExploreStrategy() error = %v", err)
	}
}

func TestResolveExploreCoverageRequiresSemanticModeAndKnownProbes(t *testing.T) {
	for _, test := range []struct {
		mode      string
		required  []string
		want      runner.CoverageMode
		wantError bool
	}{
		{mode: "none", want: runner.CoverageNone},
		{mode: "semantic", required: []string{"stdlib.os.openfile"}, want: runner.CoverageSemantic},
		{mode: "choice", want: runner.CoverageChoice},
		{mode: "semantic+choice", required: []string{"stdlib.os.openfile"}, want: runner.CoverageSemanticChoice},
		{mode: "none", required: []string{"stdlib.os.openfile"}, wantError: true},
		{mode: "choice", required: []string{"stdlib.os.openfile"}, wantError: true},
		{mode: "semantic", required: []string{"unknown.probe"}, wantError: true},
		{mode: "code", wantError: true},
	} {
		got, err := resolveExploreCoverage(test.mode, test.required)
		if (err != nil) != test.wantError || got != test.want {
			t.Fatalf("resolveExploreCoverage(%q, %v) = %q, %v", test.mode, test.required, got, err)
		}
	}
}

func TestResolveExploreGuidanceEnablesSemanticCoverageAndRequiresCorpus(t *testing.T) {
	for _, test := range []struct {
		guide, coverageSet bool
		corpus, coverage   string
		want               string
		wantError          bool
	}{
		{guide: true, corpus: "/corpus", coverage: "none", want: "semantic"},
		{guide: true, corpus: "/corpus", coverage: "semantic", coverageSet: true, want: "semantic"},
		{guide: true, corpus: "/corpus", coverage: "choice", coverageSet: true, want: "choice"},
		{guide: true, corpus: "/corpus", coverage: "semantic+choice", coverageSet: true, want: "semantic+choice"},
		{guide: true, coverage: "none", wantError: true},
		{corpus: "/corpus", coverage: "none", wantError: true},
		{guide: true, corpus: "/corpus", coverage: "none", coverageSet: true, wantError: true},
	} {
		got, err := resolveExploreGuidance(test.guide, test.corpus, test.coverage, test.coverageSet)
		if (err != nil) != test.wantError || got != test.want {
			t.Fatalf("resolveExploreGuidance(%t, %q, %q, %t) = %q, %v", test.guide, test.corpus, test.coverage, test.coverageSet, got, err)
		}
	}
}

func TestResolveChoiceTraceRequiresEnablementAndBoundedCapacity(t *testing.T) {
	for _, test := range []struct {
		name      string
		enabled   bool
		limit     byteSize
		limitSet  bool
		want      uint64
		wantError bool
	}{
		{name: "disabled", limit: 8 << 20},
		{name: "enabled default", enabled: true, limit: 8 << 20, want: 8 << 20},
		{name: "enabled explicit", enabled: true, limit: 1 << 20, limitSet: true, want: 1 << 20},
		{name: "bytes without choices", limit: 1 << 20, limitSet: true, wantError: true},
		{name: "too small", enabled: true, limit: 1, limitSet: true, wantError: true},
		{name: "too large", enabled: true, limit: 65 << 20, limitSet: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			observed, err := resolveChoiceTrace(test.enabled, test.limit, test.limitSet)
			if (err != nil) != test.wantError || observed != test.want {
				t.Fatalf("resolveChoiceTrace() = %d, %v, want %d error=%t", observed, err, test.want, test.wantError)
			}
		})
	}
}

func TestRunRejectsUnknownCommandWithUsageStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := Run([]string{"unknown"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status = %d, stderr = %q", status, stderr.String())
	}
}

func TestParseTargetPreservesArgumentVector(t *testing.T) {
	spec, err := parseTarget([]string{"go-test", "./pkg", "--", "-test.run=Test Name", "literal;$value"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.source != "./pkg" || len(spec.arguments) != 2 || spec.arguments[0] != "-test.run=Test Name" || spec.arguments[1] != "literal;$value" {
		t.Fatalf("target = %#v", spec)
	}
}

func TestRunAnalyzeEmitsSupportedJSONWithoutExecutingTarget(t *testing.T) {
	dependencies := analyzeDependencies{
		toolchain: func(string) (string, error) { return "/toolchain", nil },
		identity: func(string) (target.ToolchainIdentity, error) {
			return target.ToolchainIdentity{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH}, nil
		},
		workingDirectory: func() (string, error) { return "/workspace", nil },
		review: func(_ context.Context, spec target.Spec) (target.CapabilityReview, error) {
			if spec.Kind != target.KindGoTest || spec.Source != "./pkg" || len(spec.Args) != 1 || spec.Args[0] != "-test.run=TestScenario" || len(spec.BuildTags) != 1 || spec.CapabilityMode != target.CapabilityModeClosure {
				t.Fatalf("analysis spec = %#v", spec)
			}
			return target.CapabilityReview{}, nil
		},
		build: func(input qualification.AnalysisInput) (qualification.AnalysisReport, error) {
			return qualification.AnalysisReport{Schema: qualification.AnalysisSchema, Classification: qualification.ClassificationSupported, Packs: []target.CompatibilityPackEvidence{}, Requirements: []deterministicio.Requirement{}, Blockers: []qualification.AnalysisBlocker{}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	status := runAnalyzeWith([]string{"--format=json", "--build-tag", "gomad_fixture", "go-test", "./pkg", "--", "-test.run=TestScenario"}, &stdout, &stderr, dependencies)
	if status != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"schema":"gomadv3.capability-analysis/v4"`) || !strings.Contains(stdout.String(), `"classification":"supported"`) {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestParseCapabilityModeUsesClosedVocabulary(t *testing.T) {
	for _, value := range []string{"closure", "linked", "guarded"} {
		mode, err := parseCapabilityMode(value)
		if err != nil || string(mode) != value {
			t.Fatalf("parseCapabilityMode(%q) = %q, %v", value, mode, err)
		}
	}
	if _, err := parseCapabilityMode("auto"); err == nil {
		t.Fatal("parseCapabilityMode() accepted an unknown mode")
	}
}

func TestAnalyzeClassifiesLinkedCapabilityCapacityAsUnsupported(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := reportAnalyzeError(&stderr, &target.UnsupportedCapabilityCapacityError{Resource: "facts", Required: 100001, Maximum: 100000})
	if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "linked capability capacity") {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
}

func TestCapabilityAnalysisTimeoutAllowsLinkedBuild(t *testing.T) {
	if got := capabilityAnalysisTimeoutForMode(target.CapabilityModeClosure); got != 30*time.Second {
		t.Fatalf("closure timeout = %v", got)
	}
	if got := capabilityAnalysisTimeoutForMode(target.CapabilityModeLinked); got != 2*time.Minute {
		t.Fatalf("linked timeout = %v", got)
	}
	if got := capabilityAnalysisTimeoutForMode(target.CapabilityModeGuarded); got != 2*time.Minute {
		t.Fatalf("guarded timeout = %v", got)
	}
	for _, test := range []struct {
		name      string
		mode      target.CapabilityMode
		requested time.Duration
		want      time.Duration
		wantError bool
	}{
		{name: "closure default", mode: target.CapabilityModeClosure, want: 30 * time.Second},
		{name: "linked default", mode: target.CapabilityModeLinked, want: 2 * time.Minute},
		{name: "guarded default", mode: target.CapabilityModeGuarded, want: 2 * time.Minute},
		{name: "explicit bounded", mode: target.CapabilityModeLinked, requested: 5 * time.Minute, want: 5 * time.Minute},
		{name: "negative", mode: target.CapabilityModeLinked, requested: -time.Second, wantError: true},
		{name: "over maximum", mode: target.CapabilityModeLinked, requested: 30*time.Minute + time.Second, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveCapabilityAnalysisTimeout(test.mode, test.requested)
			if (err != nil) != test.wantError || got != test.want {
				t.Fatalf("resolveCapabilityAnalysisTimeout() = %v, %v, want %v error=%t", got, err, test.want, test.wantError)
			}
		})
	}
}

func TestRunAnalyzeMapsUnsupportedInvalidAndInfrastructureStatuses(t *testing.T) {
	base := analyzeDependencies{
		toolchain:        func(string) (string, error) { return "/toolchain", nil },
		identity:         func(string) (target.ToolchainIdentity, error) { return target.ToolchainIdentity{}, nil },
		workingDirectory: func() (string, error) { return "/workspace", nil },
		review: func(context.Context, target.Spec) (target.CapabilityReview, error) {
			return target.CapabilityReview{}, nil
		},
		build: func(qualification.AnalysisInput) (qualification.AnalysisReport, error) {
			return qualification.AnalysisReport{Classification: qualification.ClassificationUnsupported, Blockers: []qualification.AnalysisBlocker{}}, nil
		},
	}
	for _, test := range []struct {
		name       string
		arguments  []string
		configure  func(*analyzeDependencies)
		wantStatus int
	}{
		{name: "unsupported", arguments: []string{"go-run", "./pkg"}, wantStatus: 1},
		{name: "opaque executable", arguments: []string{"exec", "--provenance", "p.json", "--", "binary"}, wantStatus: 2},
		{name: "invalid package", arguments: []string{"go-run", "./missing"}, configure: func(dependencies *analyzeDependencies) {
			dependencies.review = func(context.Context, target.Spec) (target.CapabilityReview, error) {
				return target.CapabilityReview{}, &target.InvalidCapabilityReviewError{Err: errors.New("missing package")}
			}
		}, wantStatus: 2},
		{name: "infrastructure", arguments: []string{"go-run", "./pkg"}, configure: func(dependencies *analyzeDependencies) {
			dependencies.review = func(context.Context, target.Spec) (target.CapabilityReview, error) {
				return target.CapabilityReview{}, errors.New("decode failed")
			}
		}, wantStatus: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			dependencies := base
			if test.configure != nil {
				test.configure(&dependencies)
			}
			var stdout, stderr bytes.Buffer
			if status := runAnalyzeWith(test.arguments, &stdout, &stderr, dependencies); status != test.wantStatus {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunAnalyzeClassifiesRealReadonlyModuleFailureAsInvalidInput(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/target\n\ngo 1.26.4\n\nrequire github.com/stretchr/testify v1.11.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main\n\nimport _ \"github.com/stretchr/testify/require\"\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	toolchain, err := filepath.Abs("../../../../.toolchain")
	if err != nil {
		t.Fatal(err)
	}
	dependencies := analyzeDependencies{
		toolchain:        func(string) (string, error) { return toolchain, nil },
		identity:         func(string) (target.ToolchainIdentity, error) { return target.ToolchainIdentity{}, nil },
		workingDirectory: func() (string, error) { return directory, nil },
		review:           target.ReviewCapabilities,
		build: func(qualification.AnalysisInput) (qualification.AnalysisReport, error) {
			t.Fatal("analysis report was built after invalid read-only module resolution")
			return qualification.AnalysisReport{}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	if status := runAnalyzeWith([]string{"go-run", "."}, &stdout, &stderr, dependencies); status != 2 || !strings.Contains(stderr.String(), "missing go.sum entry") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	if _, statErr := os.Stat(filepath.Join(directory, "go.sum")); !os.IsNotExist(statErr) {
		t.Fatalf("read-only analysis wrote go.sum: %v", statErr)
	}
}

func TestRunAnalyzePreservesClassificationWhenCleanupFails(t *testing.T) {
	dependencies := analyzeDependencies{
		toolchain:        func(string) (string, error) { return "/toolchain", nil },
		identity:         func(string) (target.ToolchainIdentity, error) { return target.ToolchainIdentity{}, nil },
		workingDirectory: func() (string, error) { return "/workspace", nil },
		prepare: func(_ context.Context, spec target.Spec) (target.Spec, []deterministicio.Adapter, func() error, error) {
			return spec, []deterministicio.Adapter{}, func() error { return errors.New("cleanup failed") }, nil
		},
		review: func(context.Context, target.Spec) (target.CapabilityReview, error) {
			return target.CapabilityReview{}, nil
		},
		build: func(qualification.AnalysisInput) (qualification.AnalysisReport, error) {
			return qualification.AnalysisReport{Classification: qualification.ClassificationUnsupported}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	if status := runAnalyzeWith([]string{"go-run", "./pkg"}, &stdout, &stderr, dependencies); status != 1 || !strings.Contains(stderr.String(), "cleanup failed") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestRunAnalyzeReportsOutputFailuresAsInfrastructure(t *testing.T) {
	dependencies := analyzeDependencies{
		toolchain:        func(string) (string, error) { return "/toolchain", nil },
		identity:         func(string) (target.ToolchainIdentity, error) { return target.ToolchainIdentity{}, nil },
		workingDirectory: func() (string, error) { return "/workspace", nil },
		review: func(context.Context, target.Spec) (target.CapabilityReview, error) {
			return target.CapabilityReview{}, nil
		},
		build: func(qualification.AnalysisInput) (qualification.AnalysisReport, error) {
			return qualification.AnalysisReport{Classification: qualification.ClassificationSupported}, nil
		},
	}
	var stderr bytes.Buffer
	if status := runAnalyzeWith([]string{"go-run", "./pkg"}, failingWriter{}, &stderr, dependencies); status != 3 || !strings.Contains(stderr.String(), "write capability analysis") {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRunDoctorReportsAvailableContractAsJSON(t *testing.T) {
	executable, artifacts := writeDoctorCommandFixture(t)
	var stdout, stderr bytes.Buffer
	status := runDoctor([]string{"--json", "--artifacts", artifacts}, &stdout, &stderr, executable)
	if status != 0 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
	for _, value := range []string{`"schema":"gomadv3.doctor/v3"`, `"available":true`, `"boundary_manifest_version":`, `"adapters":[`, `"installation_source":"adjacent"`, `"repair_instruction":`} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("doctor JSON = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestRunDoctorReportsRepairCommandWhenToolchainIsMissing(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, ".bin", "gomad")
	if err := os.MkdirAll(filepath.Dir(executable), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("runner"), 0o700); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := runDoctor([]string{"--artifacts", filepath.Join(root, "artifacts")}, &stdout, &stderr, executable)
	if status != 1 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "available=false") || !strings.Contains(stdout.String(), "set GOMADV3_TOOLCHAIN_DIR") {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
}

func writeDoctorCommandFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	key := strings.Repeat("b", 64)
	for _, directory := range []string{filepath.Join(root, ".toolchain", "bin"), filepath.Join(root, ".toolchain", "builds", key, "bin"), filepath.Join(root, ".bin")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	goScript := "#!/bin/sh\nprintf 'go1.26.4\\n" + runtime.GOOS + "\\n" + runtime.GOARCH + "\\n0\\n'\n"
	for _, path := range []string{filepath.Join(root, ".toolchain", "bin", "go"), filepath.Join(root, ".toolchain", "builds", key, "bin", "go")} {
		if err := os.WriteFile(path, []byte(goScript), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".toolchain", "build-key"), []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(root, ".bin", "gomad")
	if err := os.WriteFile(executable, []byte("runner"), 0o700); err != nil {
		t.Fatal(err)
	}
	return executable, filepath.Join(root, "artifacts")
}

func TestRunInspectReportsBatchAsTextAndJSON(t *testing.T) {
	path := writeInspectBatchFixture(t)
	for _, test := range []struct {
		name      string
		arguments []string
		want      []string
	}{
		{name: "text", arguments: []string{path}, want: []string{"gomad inspect: kind=batch", "run-inspect-command", "attempted=1", "seed=7 domain=success"}},
		{name: "json", arguments: []string{"--json", path}, want: []string{`"schema":"gomadv3.inspect/v4"`, `"kind":"batch"`, `"run_id":"run-inspect-command"`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if status := runInspect(test.arguments, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
				t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
			}
			for _, want := range test.want {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("inspect output = %q, missing %q", stdout.String(), want)
				}
			}
		})
	}
}

func TestRunInspectRejectsChoicesForBatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := runInspect([]string{"--choices", writeInspectBatchFixture(t)}, &stdout, &stderr); status != 2 || !strings.Contains(stderr.String(), "traced artifact") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestPrintInspectionReportsInterruptedCombinedFrontierWork(t *testing.T) {
	report := runner.Inspection{
		Kind: "batch", Path: "/batch",
		Lifecycle: &runner.CampaignLifecycleInspection{State: "running", Resumable: true},
		CombinedFrontier: &runner.CombinedFrontierInspection{
			Schema: "gomadv3.combined-frontier-inspection/v1",
			Summary: runner.CombinedFrontierSummary{
				MaxRuns: 8, MaxForcedDecisions: 2, MaxFrontierBytes: 4096, MaxResultBytes: 2048,
				Limits:  runner.CombinedDimensionLimits{Runtime: 1, Scenario: 2, Network: 3, Storage: 4, Fault: 5, Crash: 6},
				Pending: 1, PendingBytes: 512,
			},
			Pending: []runner.CombinedCandidateInspection{{
				SHA256: "sha256:candidate", Overrides: []runner.CombinedOverrideInspection{{
					Dimension: "fault", Ordinal: 0, Selected: 1, Alternatives: 2, Identity: "sha256:override", ControlBytes: 64, ControlSHA256: "sha256:control",
				}},
			}},
			StagedRound: &runner.CombinedStagedRoundInspection{Index: 3, Candidates: 2, Attempted: 1},
		},
	}
	var output bytes.Buffer
	if err := printInspection(&output, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"combined-frontier:", "pending=1", "runtime=1", "scenario=2", "network=3", "storage=4", "fault=5", "crash=6", "pending-candidate:", "forced-decision:", "staged-round: index=3 candidates=2 attempted=1"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("inspection output = %q, missing %q", output.String(), want)
		}
	}
}

func TestPrintInspectionReportsSimulationExplorationEvidence(t *testing.T) {
	report := runner.Inspection{
		Kind: "artifact",
		Artifact: &runner.ArtifactInspection{
			Simulation: &runner.SimulationInspection{
				Profile: "gomadv3-simulation-exploration/v1", ControllerSHA256: "sha256:controller", ExecutionSHA256: "sha256:execution",
				CandidateSHA256: "sha256:candidate", OutcomeSHA256: "sha256:outcome", FailureSHA256: "sha256:failure",
				Plan:   runner.SimulationPayloadInspection{Schema: "gomadv3.simulation-exploration-plan/v1", SHA256: "sha256:plan", Bytes: 123},
				Record: runner.SimulationRecordInspection{Schema: "gomadv3.cluster-record/v7", SHA256: "sha256:record", Bytes: 456, Limit: 789},
			},
		},
	}
	var output bytes.Buffer
	if err := printInspection(&output, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"simulation:", "controller=sha256:controller", "execution=sha256:execution", "candidate=sha256:candidate", "outcome=sha256:outcome", "failure=sha256:failure", "plan-bytes=123", "record-bytes=456", "record-limit=789"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("inspection output = %q, missing %q", output.String(), want)
		}
	}
}

func TestRunInspectReportsOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	if status := runInspect([]string{writeInspectBatchFixture(t)}, failingWriter{}, &stderr); status != 3 || !strings.Contains(stderr.String(), "write inspection report") {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
}

func TestRunRecoverReportsStableTextAndJSON(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
		want      []string
	}{
		{name: "text", arguments: []string{"/artifacts/v1/run-interrupted"}, want: []string{"gomad recover:", "action=restore-running", "changed=true", "state=recoverable-failure", "resumable=true"}},
		{name: "json", arguments: []string{"--json", "/artifacts/v1/run-interrupted"}, want: []string{`"schema":"gomadv3.recovery/v1"`, `"action":"restore-running"`, `"changed":true`, `"state":"recoverable-failure"`, `"resumable":true`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := runRecoverWith(test.arguments, &stdout, &stderr, recoverDependencies{
				recover: func(context.Context, string) (runner.Recovery, error) {
					return runner.Recovery{
						Schema: "gomadv3.recovery/v1", Path: "/artifacts/v1/run-interrupted", Action: "restore-running", Changed: true,
						Before: runner.CampaignLifecycleInspection{State: "committing", Repairable: true, Action: "restore-running"},
						After:  runner.CampaignLifecycleInspection{State: "recoverable-failure", LastStableState: "running", Resumable: true},
					}, nil
				},
			})
			if status != 0 || stderr.Len() != 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
			for _, want := range test.want {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("recover output = %q, missing %q", stdout.String(), want)
				}
			}
		})
	}
}

func TestRunRecoverDistinguishesInvalidInputFromInfrastructureFailure(t *testing.T) {
	_, invalidErr := runner.Recover(context.Background(), t.TempDir())
	if invalidErr == nil || !runner.IsInvalidRecoveryError(invalidErr) {
		t.Fatalf("runner.Recover() error = %T %v, want invalid recovery error", invalidErr, invalidErr)
	}
	for _, test := range []struct {
		name       string
		recoverErr error
		wantStatus int
	}{
		{name: "invalid", recoverErr: invalidErr, wantStatus: 2},
		{name: "infrastructure", recoverErr: errors.New("disk unavailable"), wantStatus: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := runRecoverWith([]string{"/artifacts/v1/run-interrupted"}, &stdout, &stderr, recoverDependencies{
				recover: func(context.Context, string) (runner.Recovery, error) {
					return runner.Recovery{}, test.recoverErr
				},
			})
			if status != test.wantStatus || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.recoverErr.Error()) {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunRecoverRepairsPublishedBatchPrivateState(t *testing.T) {
	path := writeInspectBatchFixture(t)
	stale := filepath.Join(path, ".partial", "batch")
	if err := os.MkdirAll(stale, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(path, ".partial"), 0o700); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if status := runRecover([]string{path}, &stdout, &stderr); status != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "action=finalize-publication") || !strings.Contains(stdout.String(), "changed=true") {
		t.Fatalf("status output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if _, err := os.Lstat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale private state remains: %v", err)
	}
}

func writeInspectBatchFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "run-inspect-command")
	err := os.Mkdir(path, 0o700)
	if err != nil {
		t.Fatal(err)
	}
	runs, err := evidence.CanonicalJSON(map[string]any{
		"artifact": nil, "domain": "success", "elapsed_nanos": evidence.Uint64String(5), "failure_signature": nil,
		"io_transcript_records": nil, "io_transcript_sha256": nil, "reason": "success", "seed": evidence.Uint64String(7),
		"selection_ordinal": evidence.Uint64String(0), "termination": "exit",
	})
	if err != nil {
		t.Fatal(err)
	}
	runs = append(runs, '\n')
	if err := os.WriteFile(filepath.Join(path, "runs.jsonl"), runs, 0o600); err != nil {
		t.Fatal(err)
	}
	batch, err := evidence.CanonicalJSON(map[string]any{
		"attempted": evidence.Uint64String(1), "cancelled": evidence.Uint64String(0), "distinct_failures": evidence.Uint64String(0),
		"failure_signatures": []evidence.SHA256{}, "failures": evidence.Uint64String(0), "run_id": "run-inspect-command",
		"runs_sha256": evidence.HashBytes(runs), "schema": "gomadv3.batch/v2", "schema_version": evidence.SchemaVersion,
		"selection": "7", "selection_count": evidence.Uint64String(1), "stop_reason": "seeds_exhausted", "strategy": "seed",
		"succeeded": evidence.Uint64String(1), "watchdogs": evidence.Uint64String(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "batch.json"), batch, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExploreReporterEmitsStableJSONEventsAndEveryArtifact(t *testing.T) {
	var stdout, stderr bytes.Buffer
	reporter := newExploreReporter(true, &stdout, &stderr)
	if err := reporter.Progress(runner.CampaignEvent{
		Phase: runner.ProgressPreparing, CampaignPath: "/batch", Selected: 3,
	}); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Result(runner.CampaignResult{
		CampaignPath: "/batch", SelectionCount: 3, Attempted: 3, Succeeded: 1, Failures: 2, DistinctFailures: 2,
		StopReason: runner.StopSeedsExhausted, Artifacts: []string{"/batch/failures/one", "/batch/failures/two"},
		SemanticCoverage: &deterministicio.SemanticCoverage{Schema: deterministicio.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	for _, value := range []string{
		`"schema":"gomadv3.explore-event/v2"`, `"type":"progress"`, `"phase":"preparing"`,
		`"type":"result"`, `"classification":"target_failure"`, `"novelty":2`,
		`"semantic_coverage":{"schema":"gomadv3.semantic-coverage/v1","digest":"sha256:coverage","probes":["stdlib.os.openfile"]}`,
		`"path":"/batch/failures/one"`, `"path":"/batch/failures/two"`,
	} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("events = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestExploreReporterReportsRetainedSuccessfulRuns(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		reporter := newExploreReporter(jsonOutput, &stdout, &stderr)
		if err := reporter.Result(runner.CampaignResult{
			CampaignPath: "/batch", SelectionCount: 1, Attempted: 1, Succeeded: 1, RetainedSuccesses: 1,
			RetainedSuccessBytes: 4096, SuccessArtifacts: []string{"/batch/successes/sha256-case"}, StopReason: runner.StopSeedsExhausted,
		}); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"retained", "4096", "/batch/successes/sha256-case", "gomad replay"} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("json=%t output = %q, missing %q", jsonOutput, stdout.String(), want)
			}
		}
		if stderr.Len() != 0 {
			t.Fatalf("json=%t stderr = %q", jsonOutput, stderr.String())
		}
	}
}

func TestExploreReporterReportsGuidedCorpusUpdates(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		reporter := newExploreReporter(jsonOutput, &stdout, &stderr)
		if err := reporter.Result(runner.CampaignResult{
			CampaignPath: "/batch", SelectionCount: 4, Attempted: 4, Succeeded: 4, StopReason: runner.StopSeedsExhausted,
			CorpusPath: "/corpus", CorpusEntries: 12, CorpusAdded: 2,
		}); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"/corpus", "12", "2"} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("json=%t output = %q, missing %q", jsonOutput, stdout.String(), want)
			}
		}
	}
}

func TestExploreReporterReportsCombinedFrontierBoundsAndRemainingWork(t *testing.T) {
	frontier := &runner.CombinedFrontierSummary{
		Parallel: 2, MaxRuns: 16, MaxForcedDecisions: 4, MaxFrontierBytes: 1 << 20, MaxResultBytes: 64 << 10,
		Limits:            runner.CombinedDimensionLimits{Runtime: 2, Scenario: 3, Network: 4, Storage: 5, Fault: 6, Crash: 7},
		LogicalExecutions: 5, CommittedRounds: 3, Pending: 4, PendingBytes: 2048, SeenCandidates: 9,
		DeduplicatedOutcomes: 3, DistinctFailures: 1, DeepestOverride: 2, OmittedByDimension: 8,
	}
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		reporter := newExploreReporter(jsonOutput, &stdout, &stderr)
		if err := reporter.Result(runner.CampaignResult{
			CampaignPath: "/batch", SelectionCount: 1, Attempted: 5, Failures: 1,
			StopReason: runner.StopDimensionDepthComplete, CombinedFrontier: frontier, RecoveryExecutions: 2,
		}); err != nil {
			t.Fatal(err)
		}
		frontierName := "combined-frontier"
		if jsonOutput {
			frontierName = "combined_frontier"
		}
		for _, want := range []string{frontierName, "pending", "2048", "runtime", "scenario", "network", "storage", "fault", "crash", "recovery"} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("json=%t output = %q, missing %q", jsonOutput, stdout.String(), want)
			}
		}
		if stderr.Len() != 0 {
			t.Fatalf("json=%t stderr = %q", jsonOutput, stderr.String())
		}
	}
}

func TestRunExploreReportsFlagErrorsAsJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := runExplore([]string{"--json", "--parallel", "invalid"}, &stdout, &stderr)
	if status != 2 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
	for _, value := range []string{`"schema":"gomadv3.explore-event/v2"`, `"type":"error"`, `"classification":"invalid_input"`} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("explore output = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestRunQualifyRepeatsOneSeedAndRetainsJSONReport(t *testing.T) {
	var calls int
	var configs []runner.CampaignSpec
	var retained qualification.QualificationReport
	var resolvedToolchainRoot string
	dependencies := qualifyDependencies{
		identity: func(explicitToolchainRoot string) (string, string, string, error) {
			resolvedToolchainRoot = explicitToolchainRoot
			return "/toolchain", "/bin/gomad", "sha256:runner", nil
		},
		workingDirectory: func() (string, error) { return "/workspace", nil },
		run: func(_ context.Context, config runner.CampaignSpec) (runner.CampaignResult, error) {
			calls++
			configs = append(configs, config)
			evidence := qualificationEvidence(7)
			return runner.CampaignResult{CampaignPath: fmt.Sprintf("/artifacts/run-%d", calls), SelectionCount: 1, Attempted: 1, Succeeded: 1, ExecutionEvidence: &evidence}, nil
		},
		replay: func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
			t.Fatal("unexpected replay")
			return runner.ReplayResult{}, nil
		},
		write: func(_ string, report qualification.QualificationReport) (string, error) {
			retained = report
			return "/artifacts/qualifications/v1/report.json", nil
		},
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{
		"--json", "--seed", "7", "--repeat", "2", "--artifacts", "/artifacts", "--toolchain-root", "/bundle/toolchain", "--require-probe", "stdlib.os.openfile", "--choices", "--choice-bytes", "1MiB",
		"go-test", "./pkg", "--", "-test.run=TestScenario",
	}, &stdout, &stderr, dependencies)
	if status != 0 || stderr.Len() != 0 || calls != 2 || !retained.Qualified || resolvedToolchainRoot != "/bundle/toolchain" {
		t.Fatalf("status=%d calls=%d report=%#v stdout=%q stderr=%q", status, calls, retained, stdout.String(), stderr.String())
	}
	for _, config := range configs {
		if config.Seeds != "7" || config.Parallel != 1 || config.OnFailure != runner.PolicyAll || config.Coverage != runner.CoverageSemanticChoice || !config.CollectRunEvidence || config.ChoiceTraceLimit != 1<<20 || config.Target.Source != "./pkg" || config.Target.WorkingDir != "/workspace" || len(config.RequiredSemanticProbes) != 1 {
			t.Fatalf("config = %#v", config)
		}
	}
	for _, want := range []string{`"schema":"gomadv3.qualify-event/v1"`, `"type":"result"`, `"classification":"qualified"`, `"report_path":"/artifacts/qualifications/v1/report.json"`, `"qualified":true`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("output = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestRunQualifyReportsNondeterministicEvidence(t *testing.T) {
	var calls int
	var retained qualification.QualificationReport
	dependencies := qualificationDependencies(t)
	dependencies.run = func(_ context.Context, _ runner.CampaignSpec) (runner.CampaignResult, error) {
		calls++
		runRecord := qualificationEvidence(7)
		if calls == 2 {
			runRecord.Stdout.FullSHA256 = evidence.HashBytes([]byte("different"))
		}
		return runner.CampaignResult{CampaignPath: fmt.Sprintf("/artifacts/run-%d", calls), SelectionCount: 1, Attempted: 1, Succeeded: 1, ExecutionEvidence: &runRecord}, nil
	}
	dependencies.write = func(_ string, report qualification.QualificationReport) (string, error) {
		retained = report
		return "/report.json", nil
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--seed", "7", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 1 || retained.Deterministic || retained.FirstDivergence != "stdout.full_sha256" || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"classification":"nondeterministic"`) {
		t.Fatalf("status=%d report=%#v stdout=%q stderr=%q", status, retained, stdout.String(), stderr.String())
	}
}

func TestRunQualifyReplaysRepeatedTargetFailure(t *testing.T) {
	var calls, replayCalls int
	var retained qualification.QualificationReport
	dependencies := qualificationDependencies(t)
	dependencies.run = func(_ context.Context, _ runner.CampaignSpec) (runner.CampaignResult, error) {
		calls++
		evidence := qualificationEvidence(7)
		evidence.Outcome = runner.OutcomeEvidence{Domain: "target", Reason: "nonzero_exit", Termination: "exit"}
		return runner.CampaignResult{CampaignPath: fmt.Sprintf("/artifacts/run-%d", calls), SelectionCount: 1, Attempted: 1, Failures: 1, Artifacts: []string{fmt.Sprintf("/artifacts/failure-%d", calls)}, ExecutionEvidence: &evidence}, nil
	}
	dependencies.replay = func(_ context.Context, config runner.ReplaySpec) (runner.ReplayResult, error) {
		replayCalls++
		if config.ArtifactPath != fmt.Sprintf("/artifacts/failure-%d", replayCalls) {
			t.Fatalf("replay config = %#v", config)
		}
		return runner.ReplayResult{Match: true}, nil
	}
	dependencies.write = func(_ string, report qualification.QualificationReport) (string, error) {
		retained = report
		return "/report.json", nil
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--seed", "7", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 1 || calls != 2 || replayCalls != 2 || retained.Runs[0].Replay == nil || !retained.Runs[0].Replay.Match || retained.Runs[1].Replay == nil || !retained.Runs[1].Replay.Match || retained.TargetSuccess || !strings.Contains(stdout.String(), `"classification":"target_failure"`) {
		t.Fatalf("status=%d calls=%d replay=%d report=%#v stdout=%q stderr=%q", status, calls, replayCalls, retained, stdout.String(), stderr.String())
	}
}

func TestRunQualifyReplaysEveryRetainedSuccess(t *testing.T) {
	var calls, replayCalls int
	var retained qualification.QualificationReport
	dependencies := qualificationDependencies(t)
	dependencies.run = func(_ context.Context, config runner.CampaignSpec) (runner.CampaignResult, error) {
		calls++
		if config.KeepSuccesses != runner.KeepSuccessesAll || config.SuccessArtifactLimit != 1 || config.SuccessBytesLimit != 1<<20 {
			t.Fatalf("config = %#v", config)
		}
		evidence := qualificationEvidence(7)
		return runner.CampaignResult{
			CampaignPath: fmt.Sprintf("/artifacts/run-%d", calls), SelectionCount: 1, Attempted: 1, Succeeded: 1,
			RetainedSuccesses: 1, SuccessArtifacts: []string{fmt.Sprintf("/artifacts/success-%d", calls)}, ExecutionEvidence: &evidence,
		}, nil
	}
	dependencies.replay = func(_ context.Context, config runner.ReplaySpec) (runner.ReplayResult, error) {
		replayCalls++
		if config.ArtifactPath != fmt.Sprintf("/artifacts/success-%d", replayCalls) {
			t.Fatalf("replay config = %#v", config)
		}
		return runner.ReplayResult{Match: true}, nil
	}
	dependencies.write = func(_ string, report qualification.QualificationReport) (string, error) {
		retained = report
		return "/report.json", nil
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--seed", "7", "--replay-successes", "--success-limit", "1", "--success-bytes", "1MiB", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 0 || calls != 2 || replayCalls != 2 || !retained.Qualified || retained.Runs[0].Replay == nil || !retained.Runs[0].Replay.Match || retained.Runs[1].Replay == nil || !retained.Runs[1].Replay.Match || stderr.Len() != 0 {
		t.Fatalf("status=%d calls=%d replay=%d report=%#v stdout=%q stderr=%q", status, calls, replayCalls, retained, stdout.String(), stderr.String())
	}
}

func TestRunQualifyRequiresExplicitSuccessfulReplayBounds(t *testing.T) {
	dependencies := qualificationDependencies(t)
	dependencies.run = func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
		t.Fatal("unexpected run")
		return runner.CampaignResult{}, nil
	}
	for _, arguments := range [][]string{
		{"--json", "--replay-successes", "go-test", "./pkg"},
		{"--json", "--success-limit", "1", "--success-bytes", "1MiB", "go-test", "./pkg"},
	} {
		var stdout, stderr bytes.Buffer
		status := runQualifyWith(arguments, &stdout, &stderr, dependencies)
		if status != 2 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"classification":"invalid_input"`) {
			t.Fatalf("arguments=%q status=%d stdout=%q stderr=%q", arguments, status, stdout.String(), stderr.String())
		}
	}
}

func TestRunQualifyRetainsMissingSuccessfulReplayArtifact(t *testing.T) {
	var retained qualification.QualificationReport
	dependencies := qualificationDependencies(t)
	dependencies.run = func(_ context.Context, _ runner.CampaignSpec) (runner.CampaignResult, error) {
		evidence := qualificationEvidence(7)
		return runner.CampaignResult{CampaignPath: "/artifacts/run-1", SelectionCount: 1, Attempted: 1, Succeeded: 1, ExecutionEvidence: &evidence}, nil
	}
	dependencies.write = func(_ string, report qualification.QualificationReport) (string, error) {
		retained = report
		return "/report.json", nil
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--seed", "7", "--replay-successes", "--success-limit", "1", "--success-bytes", "1MiB", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 3 || retained.Failure == nil || retained.Failure.Classification != "runner_failure" || !strings.Contains(retained.Failure.Message, "exactly one successful replay artifact") || stderr.Len() != 0 {
		t.Fatalf("status=%d report=%#v stdout=%q stderr=%q", status, retained, stdout.String(), stderr.String())
	}
}

func TestRunQualifyRetainsReplayCancellation(t *testing.T) {
	var calls int
	var retained qualification.QualificationReport
	dependencies := qualificationDependencies(t)
	dependencies.run = func(_ context.Context, _ runner.CampaignSpec) (runner.CampaignResult, error) {
		calls++
		evidence := qualificationEvidence(7)
		return runner.CampaignResult{
			CampaignPath: fmt.Sprintf("/artifacts/run-%d", calls), SelectionCount: 1, Attempted: 1, Succeeded: 1,
			RetainedSuccesses: 1, SuccessArtifacts: []string{fmt.Sprintf("/artifacts/success-%d", calls)}, ExecutionEvidence: &evidence,
		}, nil
	}
	dependencies.replay = func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
		return runner.ReplayResult{}, context.Canceled
	}
	dependencies.write = func(_ string, report qualification.QualificationReport) (string, error) {
		retained = report
		return "/report.json", nil
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--seed", "7", "--replay-successes", "--success-limit", "1", "--success-bytes", "1MiB", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 3 || retained.Failure == nil || retained.Failure.Classification != "cancelled" || len(retained.Runs) != 2 || retained.Runs[0].Replay == nil || retained.Runs[0].Replay.Divergence == "" || stderr.Len() != 0 {
		t.Fatalf("status=%d report=%#v stdout=%q stderr=%q", status, retained, stdout.String(), stderr.String())
	}
}

func TestRunQualifyRetainsUnsupportedBoundary(t *testing.T) {
	var retained qualification.QualificationReport
	dependencies := qualificationDependencies(t)
	dependencies.run = func(_ context.Context, _ runner.CampaignSpec) (runner.CampaignResult, error) {
		unsupported := &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}
		return runner.CampaignResult{CampaignPath: "/artifacts/run-1"}, &runner.HostError{Reason: "target_preparation", Err: unsupported}
	}
	dependencies.write = func(_ string, report qualification.QualificationReport) (string, error) {
		retained = report
		return "/report.json", nil
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--seed", "7", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 2 || retained.Failure == nil || retained.Failure.Capability != "imports os/exec" || !strings.Contains(stdout.String(), `"classification":"unsupported_target"`) || stderr.Len() != 0 {
		t.Fatalf("status=%d report=%#v stdout=%q stderr=%q", status, retained, stdout.String(), stderr.String())
	}
}

func TestRunQualifyRejectsUnboundedRepeat(t *testing.T) {
	dependencies := qualificationDependencies(t)
	dependencies.run = func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
		t.Fatal("unexpected run")
		return runner.CampaignResult{}, nil
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--repeat", "33", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 2 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"classification":"invalid_input"`) {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestRunResumeUsesStoredBatchAndReportsResult(t *testing.T) {
	var got runner.ResumeSpec
	var resolvedToolchainRoot string
	dependencies := resumeDependencies{
		identity: func(explicitToolchainRoot string) (string, string, string, error) {
			resolvedToolchainRoot = explicitToolchainRoot
			return "/toolchain", "/bin/gomad", "sha256:runner", nil
		},
		run: func(_ context.Context, config runner.ResumeSpec) (runner.CampaignResult, error) {
			got = config
			return runner.CampaignResult{CampaignPath: "/artifacts/v1/run-partial", SelectionCount: 3, Attempted: 3, Succeeded: 3, StopReason: runner.StopSeedsExhausted}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	status := runResumeWith([]string{"--json", "--toolchain-root", "/bundle/toolchain", "/artifacts/v1/run-partial"}, &stdout, &stderr, dependencies)
	if status != 0 || stderr.Len() != 0 || resolvedToolchainRoot != "/bundle/toolchain" || got.CampaignPath != "/artifacts/v1/run-partial" || got.RunnerBuild != "sha256:runner" || got.ToolchainRoot != "/toolchain" || len(got.CoordinatorCommand) != 2 {
		t.Fatalf("status=%d config=%#v stdout=%q stderr=%q", status, got, stdout.String(), stderr.String())
	}
	for _, want := range []string{`"schema":"gomadv3.explore-event/v2"`, `"type":"result"`, `"classification":"success"`, `"batch_path":"/artifacts/v1/run-partial"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("output = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestRunResumeClassifiesInvalidJournalAsInputError(t *testing.T) {
	dependencies := resumeDependencies{
		identity: func(string) (string, string, string, error) { return "/toolchain", "/bin/gomad", "sha256:runner", nil },
		run: func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error) {
			return runner.CampaignResult{}, &runner.HostError{Reason: "resume_setup", Err: errors.New("batch plan changed")}
		},
	}
	var stdout, stderr bytes.Buffer
	status := runResumeWith([]string{"--json", "/artifacts/v1/run-partial"}, &stdout, &stderr, dependencies)
	if status != 2 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"classification":"invalid_input"`) {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func qualificationDependencies(t *testing.T) qualifyDependencies {
	t.Helper()
	return qualifyDependencies{
		identity:         func(string) (string, string, string, error) { return "/toolchain", "/bin/gomad", "sha256:runner", nil },
		workingDirectory: func() (string, error) { return "/workspace", nil },
		run: func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
			t.Fatal("qualification runner is not configured")
			return runner.CampaignResult{}, nil
		},
		replay: func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
			t.Fatal("unexpected replay")
			return runner.ReplayResult{}, nil
		},
		write: func(string, qualification.QualificationReport) (string, error) {
			t.Fatal("qualification writer is not configured")
			return "", nil
		},
	}
}

func qualificationEvidence(seed uint64) runner.ExecutionEvidence {
	return runner.ExecutionEvidence{
		Schema: runner.ExecutionEvidenceSchema, Seed: evidence.Uint64String(seed), RunnerBuild: "sha256:runner",
		Toolchain:   evidence.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      evidence.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomadv3-target"}, BuildTags: []string{"gomad_fixture"}},
		IOProfile:   runner.IOProfileEvidence{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []evidence.Environment{{Name: "GOMADSEED", Value: fmt.Sprintf("%d", seed)}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: evidence.Stream{FullSHA256: "sha256:stdout"}, Stderr: evidence.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: deterministicio.SemanticCoverage{Schema: deterministicio.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}
}

func TestExploreReporterHumanOutputIncludesProgressAndReplayCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	reporter := newExploreReporter(false, &stdout, &stderr)
	if err := reporter.Progress(runner.CampaignEvent{
		Phase: runner.ProgressRunning, CampaignPath: "/batch", Selected: 5, Attempted: 2, Running: 2, Succeeded: 1, Failures: 1, DistinctFailures: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Result(runner.CampaignResult{
		CampaignPath: "/batch", SelectionCount: 5, Attempted: 5, Succeeded: 4, Failures: 1, DistinctFailures: 1,
		StopReason: runner.StopSeedsExhausted, Artifacts: []string{"/batch/failures/one"},
		ChoiceTrace: &runner.ChoiceTraceSummary{Seed: 7, Profile: choice.Profile, Records: 3, BranchingRecords: 2, SHA256: evidence.HashBytes([]byte("choices")), TerminalState: "complete"},
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "attempted=2 running=2") || !strings.Contains(stdout.String(), "retained failure: /batch/failures/one") || !strings.Contains(stdout.String(), "gomad replay /batch/failures/one") || !strings.Contains(stdout.String(), "choices-records=3 choices-decisions=0 choices-branching=2") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestClassifyExploreErrorDistinguishesInputTargetAndRunner(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{err: os.ErrInvalid, want: "invalid_input"},
		{err: &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}, want: "unsupported_target"},
		{err: &runner.HostError{Reason: "target_preparation", Err: &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}}, want: "unsupported_target"},
		{err: &deterministicio.MissingSemanticProbesError{Probes: []string{"stdlib.os.openfile"}}, want: "semantic_coverage_failure"},
		{err: &runner.HostError{Reason: "cancelled", Err: context.Canceled}, want: "cancelled"},
		{err: &runner.HostError{Reason: "overall_timeout", Err: context.DeadlineExceeded}, want: "overall_timeout"},
		{err: &runner.HostError{Reason: "coordinator_exit", Err: os.ErrClosed}, want: "runner_failure"},
	} {
		if got := classifyExploreError(test.err); got != test.want {
			t.Fatalf("classifyExploreError(%T) = %q, want %q", test.err, got, test.want)
		}
	}
}

func TestExploreErrorStatusDistinguishesUserAndOperationalFailures(t *testing.T) {
	for classification, want := range map[string]int{
		"invalid_input": 2, "unsupported_target": 2, "semantic_coverage_failure": 1,
		"cancelled": 3, "overall_timeout": 3, "runner_failure": 3,
	} {
		if got := exploreErrorStatus(classification); got != want {
			t.Fatalf("exploreErrorStatus(%q) = %d, want %d", classification, got, want)
		}
	}
}

func TestClassifyExploreSummaryDistinguishesWatchdogAndReplayDivergence(t *testing.T) {
	for _, test := range []struct {
		summary runner.CampaignResult
		want    string
	}{
		{summary: runner.CampaignResult{Failures: 1, Watchdogs: 1}, want: "watchdog_observation"},
		{summary: runner.CampaignResult{Failures: 1, ReplayDivergences: 1}, want: "replay_divergence"},
		{summary: runner.CampaignResult{Failures: 2, Watchdogs: 1}, want: "mixed_failure"},
		{summary: runner.CampaignResult{Failures: 1}, want: "target_failure"},
	} {
		if got := classifyExploreSummary(test.summary); got != test.want {
			t.Fatalf("classifyExploreSummary(%#v) = %q, want %q", test.summary, got, test.want)
		}
	}
}

func TestReportReplayResultStatesWhetherFailureWasReproduced(t *testing.T) {
	for _, test := range []struct {
		name   string
		result runner.ReplayResult
		want   string
		status int
	}{
		{name: "success", result: runner.ReplayResult{Artifact: evidence.Artifact{Manifest: evidence.ExecutionRecord{Outcome: evidence.Outcome{Domain: "success"}}}, Match: true}, want: "reproduced=true diagnostic=false result=success", status: 0},
		{name: "target failure", result: runner.ReplayResult{Match: true}, want: "reproduced=true diagnostic=false result=target_failure", status: 1},
		{name: "watchdog observation", result: runner.ReplayResult{Match: true, Diagnostic: true}, want: "reproduced=true diagnostic=true result=watchdog_observation", status: 1},
		{name: "divergence", result: runner.ReplayResult{Divergence: "stdout.full_sha256"}, want: "reproduced=false divergence=stdout.full_sha256", status: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			status, err := reportReplayResult(&output, test.result)
			if err != nil || status != test.status || !strings.Contains(output.String(), test.want) {
				t.Fatalf("status = %d, error = %v, output = %q, want %q", status, err, output.String(), test.want)
			}
		})
	}
}

func TestRunMinimizeUsesBoundedArtifactStoreAndCurrentInstallation(t *testing.T) {
	var observed runner.MinimizeSpec
	dependencies := minimizeDependencies{
		identity: func(string) (string, string, string, error) {
			return "/toolchain", "/bin/gomad", "runner", nil
		},
		minimize: func(_ context.Context, config runner.MinimizeSpec) (runner.MinimizeResult, error) {
			observed = config
			return runner.MinimizeResult{
				Artifact: evidence.Artifact{Path: "/artifacts/minimized/sha256-result"}, Changed: true,
				Attempts: 7, AttemptBudget: 16, Accepted: []evidence.MinimizationReduction{{Kind: "fault_entries"}}, StopReason: "minimal",
			}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	status := runMinimizeWith([]string{"--artifacts", "/artifacts", "--attempt-budget", "16", "--max-bytes", "8MiB", "/failure"}, &stdout, &stderr, dependencies)
	if status != 0 || stderr.Len() != 0 || observed.ArtifactPath != "/failure" || observed.OutputRoot != "/artifacts/minimized" || observed.AttemptBudget != 16 || observed.MaximumBytes != 8<<20 || observed.ToolchainRoot != "/toolchain" || len(observed.SupervisorCommand) != 2 || !strings.Contains(stdout.String(), "accepted=1") {
		t.Fatalf("status=%d config=%#v stdout=%q stderr=%q", status, observed, stdout.String(), stderr.String())
	}
}
