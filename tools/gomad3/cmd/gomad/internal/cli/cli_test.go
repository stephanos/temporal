package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	nativebuild "go.temporal.io/server/tools/gomad3/backends/native/build"
	"go.temporal.io/server/tools/gomad3/backends/native/preparation"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/backends/native/qualification/analysis"
	supportcomparison "go.temporal.io/server/tools/gomad3/backends/native/qualification/comparison"
	qualificationset "go.temporal.io/server/tools/gomad3/backends/native/qualification/set"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/qualification"
	qualificationworkload "go.temporal.io/server/tools/gomad3/qualification/workload"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
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

func TestRunCompareSupportMapsReviewAndIncomparableStatuses(t *testing.T) {
	baseline := publicSetReport()
	candidate := publicSetReport()
	candidate.Toolchain.BoundaryManifestSHA256 = record.HashBytes([]byte("changed"))
	dependencies := compareSupportDependencies{
		open: func(path string) (qualificationset.Report, error) {
			if path == "/baseline.json" {
				return baseline, nil
			}
			return candidate, nil
		},
		compare: supportcomparison.Compare,
	}
	var stdout, stderr bytes.Buffer
	status := runCompareSupportWith([]string{"--baseline", "/baseline.json", "--candidate", "/candidate.json", "--format", "json"}, &stdout, &stderr, dependencies)
	if status != 1 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"review_required":true`) {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	want, err := supportcomparison.Compare(supportcomparison.Input{Baseline: baseline, Candidate: candidate})
	if err != nil {
		t.Fatal(err)
	}
	var decoded supportcomparison.Result
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, want) {
		t.Fatalf("decoded comparison = %#v, want %#v", decoded, want)
	}

	baseline.Dimensions.PortableV3 = false
	dependencies.open = func(path string) (qualificationset.Report, error) {
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

func TestJSONReportDeliveryPreservesSemanticsAndFailures(t *testing.T) {
	analysis := capabilityanalysis.Report{
		Schema: capabilityanalysis.AnalysisSchema, Classification: capabilityanalysis.ClassificationSupported,
		Packs: []target.CompatibilityPackEvidence{}, Requirements: []transcript.Requirement{}, Blockers: []capabilityanalysis.Blocker{},
		GuardedBlockers: []capabilityanalysis.Blocker{}, EliminatedBlockers: []capabilityanalysis.Blocker{},
	}
	var stdout, stderr bytes.Buffer
	if status := writeAnalysis(&stdout, &stderr, "json", analysis); status != 0 || stderr.Len() != 0 {
		t.Fatalf("analysis status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	var decodedAnalysis capabilityanalysis.Report
	if err := json.Unmarshal(stdout.Bytes(), &decodedAnalysis); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decodedAnalysis, analysis) {
		t.Fatalf("decoded analysis = %#v, want %#v", decodedAnalysis, analysis)
	}
	stderr.Reset()
	if status := writeAnalysis(failingWriter{}, &stderr, "json", analysis); status != 3 || !strings.Contains(stderr.String(), "write capability analysis") {
		t.Fatalf("analysis delivery status=%d stderr=%q", status, stderr.String())
	}
	cyclicModule := &target.CapabilityModule{}
	cyclicModule.Replacement = cyclicModule
	analysis.Blockers = []capabilityanalysis.Blocker{{CapabilityFinding: target.CapabilityFinding{Module: cyclicModule}}}
	stderr.Reset()
	if status := writeAnalysis(io.Discard, &stderr, "json", analysis); status != 3 || !strings.Contains(stderr.String(), "encode capability analysis") {
		t.Fatalf("analysis encoding status=%d stderr=%q", status, stderr.String())
	}

	report := publicSetReport()
	stdout.Reset()
	if err := writeQualificationSetResult(&stdout, "json", report); err != nil {
		t.Fatal(err)
	}
	var decodedReport qualificationset.Report
	if err := json.Unmarshal(stdout.Bytes(), &decodedReport); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decodedReport, report) {
		t.Fatalf("decoded qualification set = %#v, want %#v", decodedReport, report)
	}
	if err := writeQualificationSetResult(failingWriter{}, "json", report); err == nil {
		t.Fatal("writeQualificationSetResult() accepted a delivery failure")
	}
	report.Workloads = []qualificationset.WorkloadReport{{Analysis: &analysis}}
	if err := writeQualificationSetResult(io.Discard, "json", report); err == nil || !strings.Contains(err.Error(), "encountered a cycle") {
		t.Fatalf("writeQualificationSetResult() encoding error = %v", err)
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

func publicSetReport() qualificationset.Report {
	return qualificationset.Report{
		Schema: qualificationset.ReportSchema, Name: "test-set", Description: "public fixture",
		ManifestSHA256: record.HashBytes([]byte("manifest")),
		Module:         qualificationset.ModuleIdentity{Path: "example.com/target", GoModSHA256: record.HashBytes([]byte("go.mod"))},
		Platform:       qualificationset.PlatformIdentity{GOOS: "darwin", GOARCH: "arm64"},
		Toolchain: capabilityanalysis.Toolchain{
			GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64",
			BoundaryManifestVersion: "boundary-v1", BoundaryManifestSHA256: record.HashBytes([]byte("boundary")),
		},
		IOProfile:       record.IOContract{Name: "io", ImplementationSHA256: record.SHA256(record.HashBytes([]byte("io"))), InventorySHA256: record.SHA256(record.HashBytes([]byte("inventory")))},
		Dimensions:      qualificationset.EvidenceDimensions{PortableV3: true, Analysis: true, Replay: true, Choice: true},
		ExpectationsMet: true, Workloads: []qualificationset.WorkloadReport{},
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

func TestResolveExploreStrategyRequiresExplicitBoundedSingleSeedExploration(t *testing.T) {
	valid := exploreStrategyOptions{
		Value: "choice-exploration", Seeds: "7", MaxExecutions: 8, MaxChoiceDepth: 4, MaxExplorationBytes: 1 << 20,
		MaxExecutionsSet: true, MaxChoiceDepthSet: true, MaxExplorationBytesSet: true,
	}
	strategy, choices, err := resolveExploreStrategy(valid)
	if err != nil {
		t.Fatal(err)
	}
	if strategy != runner.StrategyChoiceExploration || !choices {
		t.Fatalf("resolveExploreStrategy() = %q, %t", strategy, choices)
	}
	withStart := valid
	withStart.ChoiceStartOrdinalSet = true
	if _, _, err := resolveExploreStrategy(withStart); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name      string
		configure func(*exploreStrategyOptions)
		want      string
	}{
		{name: "count", configure: func(options *exploreStrategyOptions) { options.CountSet = true }, want: "does not accept --count"},
		{name: "multiple seeds", configure: func(options *exploreStrategyOptions) { options.Seeds = "7-8" }, want: "exactly one base seed"},
		{name: "guidance", configure: func(options *exploreStrategyOptions) { options.Guide = true }, want: "does not support --guide"},
		{name: "missing max executions", configure: func(options *exploreStrategyOptions) { options.MaxExecutionsSet = false }, want: "--max-executions"},
		{name: "zero max executions", configure: func(options *exploreStrategyOptions) { options.MaxExecutions = 0 }, want: "--max-executions"},
		{name: "missing max depth", configure: func(options *exploreStrategyOptions) { options.MaxChoiceDepthSet = false }, want: "--max-choice-depth"},
		{name: "missing exploration bytes", configure: func(options *exploreStrategyOptions) { options.MaxExplorationBytesSet = false }, want: "--max-exploration-bytes"},
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

func TestResolveExploreStrategyRequiresExplicitBoundedSimulationExploration(t *testing.T) {
	valid := exploreStrategyOptions{
		Value: "simulation-exploration", Seeds: "7", MaxExecutions: 8, MaxForcedDecisions: 4,
		MaxExplorationBytes: 1 << 20, MaxExplorationResultBytes: 1 << 20,
		SimulationDimensionLimits: runner.SimulationDimensionLimits{Runtime: 4, Scenario: 4, Network: 4, Storage: 4, Fault: 4, Crash: 4},
		MaxExecutionsSet:          true, MaxForcedDecisionsSet: true, MaxExplorationBytesSet: true, MaxExplorationResultBytesSet: true,
		RuntimeLimitSet: true, ScenarioLimitSet: true, NetworkLimitSet: true, StorageLimitSet: true, FaultLimitSet: true, CrashLimitSet: true,
	}
	strategy, choices, err := resolveExploreStrategy(valid)
	if err != nil {
		t.Fatal(err)
	}
	if strategy != runner.StrategySimulationExploration || !choices {
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
		{name: "choice start ordinal", configure: func(options *exploreStrategyOptions) { options.ChoiceStartOrdinalSet = true }, want: "does not accept --choice-start-ordinal"},
		{name: "missing max executions", configure: func(options *exploreStrategyOptions) { options.MaxExecutionsSet = false }, want: "--max-executions"},
		{name: "missing forced decisions", configure: func(options *exploreStrategyOptions) { options.MaxForcedDecisionsSet = false }, want: "--max-forced-decisions"},
		{name: "missing exploration bytes", configure: func(options *exploreStrategyOptions) { options.MaxExplorationBytesSet = false }, want: "--max-exploration-bytes"},
		{name: "missing result bytes", configure: func(options *exploreStrategyOptions) { options.MaxExplorationResultBytesSet = false }, want: "--max-exploration-result-bytes"},
		{name: "missing runtime bound", configure: func(options *exploreStrategyOptions) { options.RuntimeLimitSet = false }, want: "--max-runtime-decisions"},
		{name: "zero crash bound", configure: func(options *exploreStrategyOptions) { options.SimulationDimensionLimits.Crash = 0 }, want: "--max-crash-decisions"},
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

func TestResolveExploreStrategyRejectsExplorationBoundsForSeeds(t *testing.T) {
	for _, options := range []exploreStrategyOptions{
		{Value: "seed", Seeds: "7", MaxExecutions: 1, MaxExecutionsSet: true},
		{Value: "seed", Seeds: "7", ChoiceStartOrdinalSet: true},
	} {
		if _, _, err := resolveExploreStrategy(options); err == nil || !strings.Contains(err.Error(), "require") || !strings.Contains(err.Error(), "--strategy=choice-exploration") {
			t.Fatalf("resolveExploreStrategy(%#v) error = %v", options, err)
		}
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

func TestResolveExploreTypedErrorPresentation(t *testing.T) {
	for _, strategy := range []string{string(runner.StrategyChoiceExploration), string(runner.StrategySimulationExploration)} {
		_, _, err := resolveExploreStrategy(exploreStrategyOptions{Value: strategy, Seeds: "7,8"})
		want := "--strategy=" + strategy + " requires exactly one base seed"
		if err == nil || err.Error() != want {
			t.Fatalf("strategy %q error = %v, want %q", strategy, err, want)
		}
	}
	_, err := resolveExploreCoverage("none", []string{"stdlib.os.openfile"})
	if err == nil || err.Error() != "--require-probe requires --coverage=semantic" {
		t.Fatalf("coverage error = %v", err)
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
		{name: "enabled zero", enabled: true, limitSet: true, wantError: true},
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

// TestRunAnalyzeForwardsTargetAndClassifiesReport pins the target an analyze
// request reviews and the status each review, report, cleanup and output
// result maps to. Each row starts from dependencies that review the target
// and report it unsupported.
func TestRunAnalyzeForwardsTargetAndClassifiesReport(t *testing.T) {
	report := func(classification capabilityanalysis.Classification) func(capabilityanalysis.Input) (capabilityanalysis.Report, error) {
		return func(capabilityanalysis.Input) (capabilityanalysis.Report, error) {
			return capabilityanalysis.Report{Classification: classification}, nil
		}
	}
	cleanupFails := func(_ context.Context, spec target.Spec) (target.Spec, []transcript.Adapter, func() error, error) {
		return spec, []transcript.Adapter{}, func() error { return errors.New("cleanup failed") }, nil
	}
	for _, test := range []struct {
		name      string
		arguments []string
		// configure replaces dependencies; closures it installs report through
		// the row's t.
		configure     func(t *testing.T, dependencies *analyzeDependencies)
		failingOutput bool
		wantStatus    int
		wantStdout    []string
		// wantStderr is a substring of stderr; empty requires empty stderr.
		wantStderr string
		// anyStderr leaves stderr unchecked.
		anyStderr bool
	}{
		{
			name: "emits supported JSON without executing target", arguments: []string{"--format=json", "--build-tag", "gomad_fixture", "go-test", "./pkg", "--", "-test.run=TestScenario"},
			configure: func(t *testing.T, dependencies *analyzeDependencies) {
				dependencies.identity = func(string) (target.ToolchainIdentity, error) {
					return target.ToolchainIdentity{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH}, nil
				}
				dependencies.review = func(_ context.Context, spec target.Spec) (target.CapabilityReview, error) {
					if spec.Kind != target.KindGoTest || spec.Source != "./pkg" || len(spec.Args) != 1 || spec.Args[0] != "-test.run=TestScenario" || len(spec.BuildTags) != 1 || spec.CapabilityMode != target.CapabilityModeClosure {
						t.Fatalf("analysis spec = %#v", spec)
					}
					return target.CapabilityReview{}, nil
				}
				dependencies.build = func(capabilityanalysis.Input) (capabilityanalysis.Report, error) {
					return capabilityanalysis.Report{Schema: capabilityanalysis.AnalysisSchema, Classification: capabilityanalysis.ClassificationSupported, Packs: []target.CompatibilityPackEvidence{}, Requirements: []transcript.Requirement{}, Blockers: []capabilityanalysis.Blocker{}}, nil
				}
			},
			wantStdout: []string{`"schema":"gomad3.capability-analysis/v1"`, `"classification":"supported"`},
		},
		// Unsupported, invalid and infrastructure statuses.
		{name: "unsupported", arguments: []string{"go-run", "./pkg"}, wantStatus: 1, anyStderr: true},
		{name: "opaque executable", arguments: []string{"exec", "--provenance", "p.json", "--", "binary"}, wantStatus: 2, anyStderr: true},
		{name: "invalid package", arguments: []string{"go-run", "./missing"}, configure: func(_ *testing.T, dependencies *analyzeDependencies) {
			dependencies.review = func(context.Context, target.Spec) (target.CapabilityReview, error) {
				return target.CapabilityReview{}, &target.InvalidCapabilityReviewError{Err: errors.New("missing package")}
			}
		}, wantStatus: 2, anyStderr: true},
		{name: "infrastructure", arguments: []string{"go-run", "./pkg"}, configure: func(_ *testing.T, dependencies *analyzeDependencies) {
			dependencies.review = func(context.Context, target.Spec) (target.CapabilityReview, error) {
				return target.CapabilityReview{}, errors.New("decode failed")
			}
		}, wantStatus: 3, anyStderr: true},
		{name: "cleanup failure preserves classification", arguments: []string{"go-run", "./pkg"}, configure: func(_ *testing.T, dependencies *analyzeDependencies) {
			dependencies.prepare = cleanupFails
		}, wantStatus: 1, wantStderr: "cleanup failed"},
		{name: "cleanup failure after supported report", arguments: []string{"go-run", "./pkg"}, configure: func(_ *testing.T, dependencies *analyzeDependencies) {
			dependencies.prepare = func(ctx context.Context, spec target.Spec) (target.Spec, []transcript.Adapter, func() error, error) {
				prepared, _, cleanup, err := cleanupFails(ctx, spec)
				return prepared, nil, cleanup, err
			}
			dependencies.build = report(capabilityanalysis.ClassificationSupported)
		}, wantStatus: 3, wantStderr: "cleanup failed"},
		{name: "builds from prepared review", arguments: []string{"go-run", "./pkg"}, configure: func(t *testing.T, dependencies *analyzeDependencies) {
			inspectCalls, buildCalls := 0, 0
			dependencies.review = nil
			dependencies.inspect = func(_ context.Context, spec target.Spec) (preparation.Inspection, error) {
				inspectCalls++
				return preparation.Inspection{Spec: spec, Review: target.CapabilityReview{Schema: target.CapabilityReviewSchema}}, nil
			}
			dependencies.build = func(input capabilityanalysis.Input) (capabilityanalysis.Report, error) {
				buildCalls++
				if input.Review.Schema != target.CapabilityReviewSchema || input.Spec.Source != "./pkg" {
					t.Fatalf("prepared analysis input = %#v", input)
				}
				return capabilityanalysis.Report{Classification: capabilityanalysis.ClassificationSupported}, nil
			}
			t.Cleanup(func() {
				if inspectCalls != 1 || buildCalls != 1 {
					t.Errorf("inspect=%d build=%d, want one each", inspectCalls, buildCalls)
				}
			})
		}},
		{name: "output failure is infrastructure", arguments: []string{"go-run", "./pkg"}, configure: func(_ *testing.T, dependencies *analyzeDependencies) {
			dependencies.build = report(capabilityanalysis.ClassificationSupported)
		}, failingOutput: true, wantStatus: 3, wantStderr: "write capability analysis"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dependencies := analyzeDependencies{
				toolchain:        func(string) (string, error) { return "/toolchain", nil },
				identity:         func(string) (target.ToolchainIdentity, error) { return target.ToolchainIdentity{}, nil },
				workingDirectory: func() (string, error) { return "/workspace", nil },
				review: func(context.Context, target.Spec) (target.CapabilityReview, error) {
					return target.CapabilityReview{}, nil
				},
				build: func(capabilityanalysis.Input) (capabilityanalysis.Report, error) {
					return capabilityanalysis.Report{Classification: capabilityanalysis.ClassificationUnsupported, Blockers: []capabilityanalysis.Blocker{}}, nil
				},
			}
			if test.configure != nil {
				test.configure(t, &dependencies)
			}
			var stdout, stderr bytes.Buffer
			var output io.Writer = &stdout
			if test.failingOutput {
				output = failingWriter{}
			}
			status := runAnalyzeWith(test.arguments, output, &stderr, dependencies)
			stderrOK := test.anyStderr || strings.Contains(stderr.String(), test.wantStderr) && (test.wantStderr != "" || stderr.Len() == 0)
			if status != test.wantStatus || !stderrOK {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
			for _, want := range test.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
				}
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
		review:           nativebuild.ReviewCapabilities,
		build: func(capabilityanalysis.Input) (capabilityanalysis.Report, error) {
			t.Fatal("analysis report was built after invalid read-only module resolution")
			return capabilityanalysis.Report{}, nil
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

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRunDoctorReportsAvailableContractAsJSON(t *testing.T) {
	executable, artifacts := writeDoctorCommandFixture(t)
	var stdout, stderr bytes.Buffer
	status := application{executable: func() (string, error) { return executable, nil }, environment: os.Getenv}.runDoctor([]string{"--json", "--artifacts", artifacts}, &stdout, &stderr)
	if status != 0 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
	for _, value := range []string{`"schema":"gomad3.doctor/v3"`, `"available":true`, `"boundary_manifest_version":`, `"adapters":[`, `"installation_source":"adjacent"`, `"repair_instruction":`} {
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
	status := application{executable: func() (string, error) { return executable, nil }, environment: os.Getenv}.runDoctor([]string{"--artifacts", filepath.Join(root, "artifacts")}, &stdout, &stderr)
	if status != 1 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "available=false") || !strings.Contains(stdout.String(), "set GOMAD3_TOOLCHAIN_DIR") {
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
		{name: "text", arguments: []string{path}, want: []string{"gomad inspect: kind=campaign", "run-inspect-command", "attempted=1", "seed=7 domain=success", "log: format=gomad3.campaign-log/v1 frame-version=1 state=valid records=5 ", "tail=none tail-bytes=0 repaired-bytes=0", "committed: outcomes=1 "}},
		{name: "json", arguments: []string{"--json", path}, want: []string{`"schema":"gomad3.inspect/v8"`, `"kind":"campaign"`, `"campaign_id":"run-inspect-command"`, `"log":{"format":"gomad3.campaign-log/v1","frame_version":1,"state":"valid"`, `"committed":{"outcomes":1,`}},
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

func TestPrintInspectionReportsInterruptedSimulationExplorationWork(t *testing.T) {
	report := runner.Inspection{
		Kind: "batch", Path: "/batch",
		Lifecycle: &runner.CampaignLifecycleInspection{State: "running", Resumable: true},
		SimulationExploration: &runner.SimulationExplorationInspection{
			Schema: "gomad3.simulation-exploration-inspection/v1",
			Summary: runner.SimulationExplorationSummary{
				MaxExecutions: 8, MaxForcedDecisions: 2, MaxExplorationBytes: 4096, MaxResultBytes: 2048,
				Limits:  runner.SimulationDimensionLimits{Runtime: 1, Scenario: 2, Network: 3, Storage: 4, Fault: 5, Crash: 6},
				Pending: 1, PendingBytes: 512,
			},
			Pending: []runner.SimulationCandidateInspection{{
				SHA256: "sha256:candidate", Overrides: []runner.SimulationOverrideInspection{{
					Dimension: "fault", Ordinal: 0, Selected: 1, Alternatives: 2, Identity: "sha256:override", ControlBytes: 64, ControlSHA256: "sha256:control",
				}},
			}},
			StagedRound: &runner.SimulationStagedRoundInspection{Index: 3, Candidates: 2, Attempted: 1},
		},
	}
	var output bytes.Buffer
	if err := printInspection(&output, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"simulation-exploration:", "pending=1", "runtime=1", "scenario=2", "network=3", "storage=4", "fault=5", "crash=6", "pending-candidate:", "forced-decision:", "staged-round: index=3 candidates=2 attempted=1"} {
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
				Profile: "gomad3-simulation-exploration/v1", ControllerSHA256: "sha256:controller", ExecutionSHA256: "sha256:execution",
				CandidateSHA256: "sha256:candidate", OutcomeSHA256: "sha256:outcome", FailureSHA256: "sha256:failure",
				Plan:   runner.SimulationPayloadInspection{Schema: "gomad3.simulation-exploration-plan/v1", SHA256: "sha256:plan", Bytes: 123},
				Record: runner.SimulationRecordInspection{Schema: "gomad3.cluster-record/v7", SHA256: "sha256:record", Bytes: 456, Limit: 789},
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
		{name: "json", arguments: []string{"--json", "/artifacts/v1/run-interrupted"}, want: []string{`"schema":"gomad3.recovery/v1"`, `"action":"restore-running"`, `"changed":true`, `"state":"recoverable-failure"`, `"resumable":true`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := runRecoverWith(test.arguments, &stdout, &stderr, recoverDependencies{
				recover: func(context.Context, string) (runner.Recovery, error) {
					return runner.Recovery{
						Schema: "gomad3.recovery/v1", Path: "/artifacts/v1/run-interrupted", Action: "restore-running", Changed: true,
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
	stale := filepath.Join(path, ".prepared")
	if err := os.MkdirAll(stale, 0o700); err != nil {
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

// inspectFrame encodes one record as the Runner's log store frames it.
func inspectFrame(kind uint16, payload []byte) []byte {
	table := crc32.MakeTable(crc32.Castagnoli)
	frame := make([]byte, 16+len(payload)+4)
	copy(frame, "GMLG")
	frame[4] = 1
	binary.LittleEndian.PutUint16(frame[6:], kind)
	binary.LittleEndian.PutUint32(frame[8:], uint32(len(payload)))
	binary.LittleEndian.PutUint32(frame[12:], crc32.Checksum(frame[:12], table))
	copy(frame[16:], payload)
	binary.LittleEndian.PutUint32(frame[16+len(payload):], crc32.Update(crc32.Checksum(frame[:16], table), table, payload))
	return frame
}

// writeInspectLog writes log and an empty writer lock into the private store
// directory store.
func writeInspectLog(t *testing.T, store string, log []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(store, "log"), log, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, "lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

// writeInspectBatchFixture writes a published one-execution campaign as the
// campaign log the Runner writes: a private store holding a log of framed
// JSON records and its lock file.
func writeInspectBatchFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "run-inspect-command")
	store := filepath.Join(path, "store")
	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	const execution = `{"selection_ordinal":"0","seed":"7","domain":"success","reason":"success","termination":"exit","failure_signature":null,"artifact":null,"elapsed_nanos":"5","io_transcript_sha256":null,"io_transcript_records":null}`
	records := []struct {
		kind    uint16
		payload string
	}{
		{1, `{"format":"gomad3.campaign-log/v1","campaign_id":"run-inspect-command","strategy":"seed","selection":"7","selection_count":"1"}`},
		{2, `{"state":"prepared"}`},
		{2, `{"state":"running"}`},
		{4, `{"executions":[` + execution + `]}`},
		{5, fmt.Sprintf(`{"schema_version":%d,"schema":"gomad3.campaign/v1","campaign_id":"run-inspect-command","strategy":"seed","selection":"7","selection_count":"1","attempted":"1","succeeded":"1","failures":"0","watchdogs":"0","cancelled":"0","distinct_failures":"0","stop_reason":"seeds_exhausted","journal":{"schema":"gomad3.campaign-log/v1","records":"1","bytes":"%d"},"failure_signatures":[]}`, record.SchemaVersion, len(execution))},
	}
	var log []byte
	for _, entry := range records {
		log = append(log, inspectFrame(entry.kind, []byte(entry.payload))...)
	}
	writeInspectLog(t, store, log)
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
		SemanticCoverage: &record.SemanticCoverage{Schema: record.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	for _, value := range []string{
		`"schema":"gomad3.explore-event/v3"`, `"type":"progress"`, `"phase":"preparing"`,
		`"type":"result"`, `"classification":"target_failure"`, `"novelty":2`,
		`"semantic_coverage":{"schema":"gomad3.semantic-coverage/v1","digest":"sha256:coverage","probes":["stdlib.os.openfile"]}`,
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

func TestExploreReporterReportsSimulationExplorationBoundsAndRemainingWork(t *testing.T) {
	exploration := &runner.SimulationExplorationSummary{
		Parallel: 2, MaxExecutions: 16, MaxForcedDecisions: 4, MaxExplorationBytes: 1 << 20, MaxResultBytes: 64 << 10,
		Limits:            runner.SimulationDimensionLimits{Runtime: 2, Scenario: 3, Network: 4, Storage: 5, Fault: 6, Crash: 7},
		LogicalExecutions: 5, CommittedRounds: 3, Pending: 4, PendingBytes: 2048, SeenCandidates: 9,
		DeduplicatedOutcomes: 3, DistinctFailures: 1, DeepestOverride: 2, OmittedByDimension: 8,
	}
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		reporter := newExploreReporter(jsonOutput, &stdout, &stderr)
		if err := reporter.Result(runner.CampaignResult{
			CampaignPath: "/batch", SelectionCount: 1, Attempted: 5, Failures: 1,
			StopReason: runner.StopDimensionDepthComplete, SimulationExploration: exploration, RecoveryExecutions: 2,
		}); err != nil {
			t.Fatal(err)
		}
		explorationName := "simulation-exploration"
		if jsonOutput {
			explorationName = "simulation_exploration"
		}
		for _, want := range []string{explorationName, "pending", "2048", "runtime", "scenario", "network", "storage", "fault", "crash", "recovery"} {
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
	status := hostApplication().runExplore([]string{"--json", "--parallel", "invalid"}, &stdout, &stderr)
	if status != 2 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
	for _, value := range []string{`"schema":"gomad3.explore-event/v3"`, `"type":"error"`, `"classification":"invalid_input"`} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("explore output = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestExploreErrorReportsClassificationAfterChoiceDiagnosticWriterFailure(t *testing.T) {
	var stdout bytes.Buffer
	stderr := failingWriter{}
	reporter := newExploreReporter(true, &stdout, stderr)
	status := reportExploreFailure(runner.CampaignResult{ChoiceTrace: &runner.ChoiceTraceSummary{}}, errors.New("bad request"), reporter, stderr)
	if status != 2 || !strings.Contains(stdout.String(), `"classification":"invalid_input"`) || !strings.Contains(stdout.String(), "bad request") {
		t.Fatalf("status=%d stdout=%q", status, stdout.String())
	}
}

// qualifyObservation is what one qualify request hands its dependencies: the
// toolchain roots it resolved, every Campaign and replay request, and the
// report it retained, if any. Every request that runs a Campaign retains a
// report.
type qualifyObservation struct {
	requested []string
	runs      []runner.CampaignSpec
	replays   []runner.ReplaySpec
	written   bool
	report    qualification.QualificationReport
}

// TestRunQualifyForwardsFlagsAndClassifiesOutcome pins, for each qualify
// request, the Campaign and replay requests it forwards, the report it
// retains, and the status and result event it reports.
func TestRunQualifyForwardsFlagsAndClassifiesOutcome(t *testing.T) {
	succeeded := func(call int, _ runner.CampaignSpec) (runner.CampaignResult, error) {
		evidence := qualificationEvidence(7)
		return runner.CampaignResult{CampaignPath: fmt.Sprintf("/artifacts/run-%d", call), SelectionCount: 1, Attempted: 1, Succeeded: 1, ExecutionEvidence: &evidence}, nil
	}
	retainedSuccess := func(call int, _ runner.CampaignSpec) (runner.CampaignResult, error) {
		evidence := qualificationEvidence(7)
		return runner.CampaignResult{
			CampaignPath: fmt.Sprintf("/artifacts/run-%d", call), SelectionCount: 1, Attempted: 1, Succeeded: 1,
			RetainedSuccesses: 1, SuccessArtifacts: []string{fmt.Sprintf("/artifacts/success-%d", call)}, ExecutionEvidence: &evidence,
		}, nil
	}
	replayOf := func(kind string) func(*testing.T, int, runner.ReplaySpec) (runner.ReplayResult, error) {
		return func(t *testing.T, call int, config runner.ReplaySpec) (runner.ReplayResult, error) {
			if config.ArtifactPath != fmt.Sprintf("/artifacts/%s-%d", kind, call) {
				t.Fatalf("replay config = %#v", config)
			}
			return runner.ReplayResult{Match: true}, nil
		}
	}
	successReplay := []string{"--json", "--seed", "7", "--replay-successes", "--success-limit", "1", "--success-bytes", "1MiB", "go-test", "./pkg"}
	for _, test := range []struct {
		name      string
		arguments []string
		// run and replay answer the call-th request, counted from 1. A nil
		// function fails the test when the command calls it.
		run        func(call int, config runner.CampaignSpec) (runner.CampaignResult, error)
		replay     func(t *testing.T, call int, config runner.ReplaySpec) (runner.ReplayResult, error)
		wantStatus int
		wantOutput []string
		check      func(t *testing.T, observed qualifyObservation) bool
	}{
		{
			name: "repeat one seed and retain the JSON report",
			arguments: []string{
				"--json", "--seed", "7", "--repeat", "2", "--artifacts", "/artifacts", "--toolchain-root", "/bundle/toolchain", "--require-probe", "stdlib.os.openfile", "--choices", "--choice-bytes", "1MiB",
				"go-test", "./pkg", "--", "-test.run=TestScenario",
			},
			run:        succeeded,
			wantOutput: []string{`"schema":"gomad3.qualify-event/v1"`, `"type":"result"`, `"classification":"qualified"`, `"report_path":"/artifacts/qualifications/v1/report.json"`, `"qualified":true`},
			check: func(t *testing.T, observed qualifyObservation) bool {
				for _, config := range observed.runs {
					if config.Seeds != "7" || config.Parallel != 1 || config.OnFailure != runner.PolicyAll || config.Coverage != runner.CoverageSemanticChoice || !config.CollectExecutionEvidence || config.ChoiceTraceLimit != 1<<20 || config.Target.Source != "./pkg" || config.Target.WorkingDir != "/workspace" || len(config.RequiredSemanticProbes) != 1 {
						t.Fatalf("config = %#v", config)
					}
				}
				return len(observed.runs) == 2 && observed.report.Qualified && reflect.DeepEqual(observed.requested, []string{"/bundle/toolchain"})
			},
		},
		{
			name: "nondeterministic evidence", arguments: []string{"--json", "--seed", "7", "go-test", "./pkg"},
			run: func(call int, _ runner.CampaignSpec) (runner.CampaignResult, error) {
				runRecord := qualificationEvidence(7)
				if call == 2 {
					runRecord.Stdout.FullSHA256 = record.HashBytes([]byte("different"))
				}
				return runner.CampaignResult{CampaignPath: fmt.Sprintf("/artifacts/run-%d", call), SelectionCount: 1, Attempted: 1, Succeeded: 1, ExecutionEvidence: &runRecord}, nil
			},
			wantStatus: 1, wantOutput: []string{`"classification":"nondeterministic"`},
			check: func(_ *testing.T, observed qualifyObservation) bool {
				return !observed.report.Deterministic && observed.report.FirstDivergence == "stdout.full_sha256"
			},
		},
		{
			name: "replays repeated target failure", arguments: []string{"--json", "--seed", "7", "go-test", "./pkg"},
			run: func(call int, _ runner.CampaignSpec) (runner.CampaignResult, error) {
				evidence := qualificationEvidence(7)
				evidence.Outcome = runner.OutcomeEvidence{Domain: "target", Reason: "nonzero_exit", Termination: "exit"}
				return runner.CampaignResult{CampaignPath: fmt.Sprintf("/artifacts/run-%d", call), SelectionCount: 1, Attempted: 1, Failures: 1, Artifacts: []string{fmt.Sprintf("/artifacts/failure-%d", call)}, ExecutionEvidence: &evidence}, nil
			},
			replay:     replayOf("failure"),
			wantStatus: 1, wantOutput: []string{`"classification":"target_failure"`},
			check: func(_ *testing.T, observed qualifyObservation) bool {
				executions := observed.report.Executions
				return len(observed.runs) == 2 && len(observed.replays) == 2 && executions[0].Replay != nil && executions[0].Replay.Match && executions[1].Replay != nil && executions[1].Replay.Match && !observed.report.TargetSuccess
			},
		},
		{
			name: "replays every retained success", arguments: successReplay,
			run: retainedSuccess, replay: replayOf("success"),
			check: func(t *testing.T, observed qualifyObservation) bool {
				for _, config := range observed.runs {
					if config.KeepSuccesses != runner.KeepSuccessesAll || config.SuccessArtifactLimit != 1 || config.SuccessBytesLimit != 1<<20 {
						t.Fatalf("config = %#v", config)
					}
				}
				executions := observed.report.Executions
				return len(observed.runs) == 2 && len(observed.replays) == 2 && observed.report.Qualified && executions[0].Replay != nil && executions[0].Replay.Match && executions[1].Replay != nil && executions[1].Replay.Match
			},
		},
		// Successful replay needs both the request and explicit bounds.
		{name: "successful replay requires bounds", arguments: []string{"--json", "--replay-successes", "go-test", "./pkg"}, wantStatus: 2, wantOutput: []string{`"classification":"invalid_input"`}},
		{name: "success bounds require successful replay", arguments: []string{"--json", "--success-limit", "1", "--success-bytes", "1MiB", "go-test", "./pkg"}, wantStatus: 2, wantOutput: []string{`"classification":"invalid_input"`}},
		{
			name: "retains missing successful replay artifact", arguments: successReplay, run: succeeded, wantStatus: 3,
			check: func(_ *testing.T, observed qualifyObservation) bool {
				failure := observed.report.Failure
				return failure != nil && failure.Classification == "runner_failure" && strings.Contains(failure.Message, "exactly one successful replay artifact")
			},
		},
		{
			name: "retains replay cancellation", arguments: successReplay, run: retainedSuccess,
			replay: func(*testing.T, int, runner.ReplaySpec) (runner.ReplayResult, error) {
				return runner.ReplayResult{}, context.Canceled
			},
			wantStatus: 3,
			check: func(_ *testing.T, observed qualifyObservation) bool {
				report := observed.report
				return report.Failure != nil && report.Failure.Classification == "cancelled" && len(report.Executions) == 2 && report.Executions[0].Replay != nil && report.Executions[0].Replay.Divergence != ""
			},
		},
		{
			name: "retains unsupported boundary", arguments: []string{"--json", "--seed", "7", "go-test", "./pkg"},
			run: func(int, runner.CampaignSpec) (runner.CampaignResult, error) {
				unsupported := &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}
				return runner.CampaignResult{CampaignPath: "/artifacts/run-1"}, &runner.HostError{Reason: "target_preparation", Err: unsupported}
			},
			wantStatus: 2, wantOutput: []string{`"classification":"unsupported_target"`},
			check: func(_ *testing.T, observed qualifyObservation) bool {
				return observed.report.Failure != nil && observed.report.Failure.Capability == "imports os/exec"
			},
		},
		{name: "rejects unbounded repeat", arguments: []string{"--json", "--repeat", "33", "go-test", "./pkg"}, wantStatus: 2, wantOutput: []string{`"classification":"invalid_input"`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			installation := newFakeInstallation()
			var observed qualifyObservation
			dependencies := installation.qualifyDependencies(func(_ context.Context, config runner.CampaignSpec) (runner.CampaignResult, error) {
				if test.run == nil {
					t.Fatal("unexpected run")
				}
				observed.runs = append(observed.runs, config)
				return test.run(len(observed.runs), config)
			}, func(_ context.Context, config runner.ReplaySpec) (runner.ReplayResult, error) {
				if test.replay == nil {
					t.Fatal("unexpected replay")
				}
				observed.replays = append(observed.replays, config)
				return test.replay(t, len(observed.replays), config)
			}, func(_ string, report qualification.QualificationReport) (string, error) {
				if test.run == nil {
					t.Fatal("unexpected report")
				}
				observed.written, observed.report = true, report
				return "/artifacts/qualifications/v1/report.json", nil
			})
			var stdout, stderr bytes.Buffer
			status := runQualifyWith(test.arguments, &stdout, &stderr, dependencies)
			observed.requested = *installation.requested
			if status != test.wantStatus || stderr.Len() != 0 || observed.written != (test.run != nil) || test.check != nil && !test.check(t, observed) {
				t.Fatalf("status=%d observed=%#v stdout=%q stderr=%q", status, observed, stdout.String(), stderr.String())
			}
			for _, want := range test.wantOutput {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("output = %q, missing %q", stdout.String(), want)
				}
			}
		})
	}
}

// TestRunResumeForwardsCampaignAndClassifiesResult pins the resume request
// built from the stored Campaign and the installation, and how its result and
// errors are reported.
func TestRunResumeForwardsCampaignAndClassifiesResult(t *testing.T) {
	for _, test := range []struct {
		name          string
		arguments     []string
		result        runner.CampaignResult
		err           error
		want          runner.ResumeSpec
		wantRequested []string
		wantStatus    int
		wantOutput    []string
	}{
		{
			name: "stored campaign", arguments: []string{"--json", "--toolchain-root", "/bundle/toolchain", "/artifacts/v1/run-partial"},
			result:        runner.CampaignResult{CampaignPath: "/artifacts/v1/run-partial", SelectionCount: 3, Attempted: 3, Succeeded: 3, StopReason: runner.StopSeedsExhausted},
			wantRequested: []string{"/bundle/toolchain"},
			wantOutput:    []string{`"schema":"gomad3.explore-event/v3"`, `"type":"result"`, `"classification":"success"`, `"campaign_path":"/artifacts/v1/run-partial"`},
		},
		{
			name: "invalid journal is an input error", arguments: []string{"--json", "/artifacts/v1/run-partial"},
			err:           &runner.HostError{Reason: "resume_setup", Err: errors.New("batch plan changed")},
			wantRequested: []string{""}, wantStatus: 2, wantOutput: []string{`"classification":"invalid_input"`},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			installation := newFakeInstallation()
			var observed []runner.ResumeSpec
			dependencies := installation.resumeDependencies(func(_ context.Context, config runner.ResumeSpec) (runner.CampaignResult, error) {
				config.Progress = nil
				observed = append(observed, config)
				return test.result, test.err
			})
			var stdout, stderr bytes.Buffer
			status := runResumeWith(test.arguments, &stdout, &stderr, dependencies)
			want := runner.ResumeSpec{
				CampaignPath: "/artifacts/v1/run-partial", RunnerBuild: "sha256:runner", ToolchainRoot: "/toolchain",
				ProgressInterval: 5 * time.Second,
			}
			if status != test.wantStatus || stderr.Len() != 0 || len(observed) != 1 || !reflect.DeepEqual(observed[0], want) || !reflect.DeepEqual(*installation.requested, test.wantRequested) {
				t.Fatalf("status=%d requests=%#v roots=%q stdout=%q stderr=%q", status, observed, *installation.requested, stdout.String(), stderr.String())
			}
			for _, want := range test.wantOutput {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("output = %q, missing %q", stdout.String(), want)
				}
			}
		})
	}
}

func qualificationDependencies(t *testing.T) qualifyDependencies {
	t.Helper()
	return qualifyDependencies{
		install: func(string) (installation, error) {
			return installation{toolchainRoot: "/toolchain", executable: "/bin/gomad", runnerBuild: "sha256:runner"}, nil
		},
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
		Schema: runner.ExecutionEvidenceSchema, Seed: record.Uint64String(seed), RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomad3-target"}, BuildTags: []string{"gomad_fixture"}},
		IOProfile:   record.IOContract{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: fmt.Sprintf("%d", seed)}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: record.SemanticCoverage{Schema: record.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
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
		ChoiceTrace: &runner.ChoiceTraceSummary{Seed: 7, Profile: choice.Profile, Records: 3, BranchingRecords: 2, SHA256: record.HashBytes([]byte("choices")), TerminalState: "complete"},
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
		{err: &record.MissingSemanticProbesError{Probes: []string{"stdlib.os.openfile"}}, want: "semantic_coverage_failure"},
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
		{name: "success", result: runner.ReplayResult{Artifact: artifact.Artifact{Manifest: record.ExecutionRecord{Outcome: record.Outcome{Domain: "success"}}}, Match: true}, want: "reproduced=true diagnostic=false result=success", status: 0},
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

// TestRunMinimizeForwardsFlags pins the minimization request each flag set
// forwards with the resolved installation, and the summary it reports.
func TestRunMinimizeForwardsFlags(t *testing.T) {
	defaultRoot, err := filepath.Abs(".gomad/artifacts")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		arguments  []string
		result     runner.MinimizeResult
		want       runner.MinimizeSpec
		wantOutput string
	}{
		{
			name: "bounded store and installation", arguments: []string{"--artifacts", "/artifacts", "--attempt-budget", "16", "--max-bytes", "8MiB", "/failure"},
			result: runner.MinimizeResult{
				Artifact: artifact.Artifact{Path: "/artifacts/minimized/sha256-result"}, Changed: true,
				Attempts: 7, AttemptBudget: 16, Accepted: []record.MinimizationReduction{{Kind: "fault_entries"}}, StopReason: "minimal",
			},
			want:       runner.MinimizeSpec{ArtifactPath: "/failure", OutputRoot: "/artifacts/minimized", AttemptBudget: 16, MaximumBytes: 8 << 20, ToolchainRoot: "/toolchain"},
			wantOutput: "accepted=1",
		},
		// Minimization resumes only on request.
		{name: "initial run", arguments: []string{"/failure"}, want: runner.MinimizeSpec{ArtifactPath: "/failure", OutputRoot: filepath.Join(defaultRoot, "minimized"), AttemptBudget: 64, ToolchainRoot: "/toolchain"}},
		{name: "resume", arguments: []string{"--resume", "/failure"}, want: runner.MinimizeSpec{ArtifactPath: "/failure", OutputRoot: filepath.Join(defaultRoot, "minimized"), AttemptBudget: 64, ToolchainRoot: "/toolchain", Resume: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var observed []runner.MinimizeSpec
			dependencies := newFakeInstallation().minimizeDependencies(func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
				observed = append(observed, spec)
				return test.result, nil
			})
			var stdout, stderr bytes.Buffer
			status := runMinimizeWith(test.arguments, &stdout, &stderr, dependencies)
			if status != 0 || stderr.Len() != 0 || len(observed) != 1 || !reflect.DeepEqual(observed[0], test.want) || !strings.Contains(stdout.String(), test.wantOutput) {
				t.Fatalf("status=%d requests=%#v want %#v stdout=%q stderr=%q", status, observed, test.want, stdout.String(), stderr.String())
			}
		})
	}
}

// TestRunQualifySetForwardsFlags pins the qualification set request each
// accepted flag set forwards with the current executable, and the shard
// checks that reject a request before the set runs.
func TestRunQualifySetForwardsFlags(t *testing.T) {
	threeSuites := func(string) (qualificationset.Manifest, error) {
		return qualificationset.Manifest{Schema: qualificationset.ManifestSchema, Name: "test-set", Suites: []qualificationset.Workload{{ID: "a"}, {ID: "b"}, {ID: "c"}}}, nil
	}
	defaults := qualificationset.Spec{
		ManifestPath: "/corpus.json", GomadPath: "/bin/gomad", WorkingDir: "/repo", ArtifactRoot: ".gomad/qualification", OutputPath: ".gomad/qualification-set.json",
		MinimumFreeBytes: qualificationset.DefaultMinimumFreeBytes,
	}
	with := func(configure func(*qualificationset.Spec)) *qualificationset.Spec {
		spec := defaults
		configure(&spec)
		return &spec
	}
	for _, test := range []struct {
		name       string
		arguments  []string
		wantStatus int
		// want is the forwarded request, or nil when the set must not run.
		want                 *qualificationset.Spec
		wantOutput           string
		wantError            string
		installationManifest string
	}{
		{
			name: "public paths and executable", arguments: []string{"--artifacts", "/artifacts", "--output", "/report.json", "--prune-qualified-artifacts", "--format", "json"},
			installationManifest: "/selected/installation.json",
			want: with(func(spec *qualificationset.Spec) {
				spec.ArtifactRoot, spec.OutputPath, spec.PruneQualifiedArtifacts = "/artifacts", "/report.json", true
				spec.InstallationManifest = "/selected/installation.json"
			}),
			wantOutput: `"schema":"gomad3.qualification-set-report/v1"`,
		},
		{
			name: "runs one shard", arguments: []string{"--shard", "1/3", "--min-free-bytes", "3GiB"},
			want: with(func(spec *qualificationset.Spec) {
				spec.Shard, spec.MinimumFreeBytes = qualificationset.Shard{Index: 1, Count: 3}, 3<<30
			}),
			wantOutput: "qualification set: name=test-set",
		},
		{name: "checks one shard", arguments: []string{"--shard", "1/2", "--check"}, wantOutput: "qualification manifest: name=test-set workloads=1\n"},
		{name: "rejects index past count", arguments: []string{"--shard", "3/3"}, wantStatus: 2, wantError: "want zero-based INDEX/COUNT"},
		{name: "rejects malformed shard", arguments: []string{"--shard", "1-3"}, wantStatus: 2, wantError: "want zero-based INDEX/COUNT"},
		{name: "rejects count past manifest", arguments: []string{"--shard", "0/4"}, wantStatus: 2, wantError: "exceeds the manifest's 3 workloads"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var observed []qualificationset.Spec
			dependencies := qualifySetDependencies{
				installationManifest: test.installationManifest,
				executable:           func() (string, error) { return "/bin/gomad", nil },
				load:                 threeSuites,
				run: func(_ context.Context, config qualificationset.Spec) (qualificationset.Report, error) {
					observed = append(observed, config)
					return publicSetReport(), nil
				},
			}
			var stdout, stderr bytes.Buffer
			status := runQualifySetWith(append([]string{"--manifest", "/corpus.json", "--working-dir", "/repo"}, test.arguments...), &stdout, &stderr, dependencies)
			if (test.want == nil) != (len(observed) == 0) || test.want != nil && (len(observed) != 1 || !reflect.DeepEqual(observed[0], *test.want)) {
				t.Fatalf("requests = %#v, want %#v", observed, test.want)
			}
			if status != test.wantStatus || !strings.Contains(stdout.String(), test.wantOutput) || !strings.Contains(stderr.String(), test.wantError) || test.wantError == "" && stderr.Len() != 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunMergeSetMapsStatuses(t *testing.T) {
	for _, test := range []struct {
		name       string
		arguments  []string
		mergeErr   error
		wantStatus int
		wantMerge  bool
		wantOutput string
		wantError  string
	}{
		{name: "merged", arguments: []string{"--manifest", "/corpus.json", "--output", "/merged.json", "/a.json", "/b.json"}, wantMerge: true, wantOutput: "qualification set: name=test-set expectations-met=true"},
		{name: "merged as json", arguments: []string{"--manifest", "/corpus.json", "--format", "json", "/a.json"}, wantMerge: true, wantOutput: `"schema":"gomad3.qualification-set-report/v1"`},
		{name: "retained mismatch", arguments: []string{"--manifest", "/corpus.json", "/a.json"}, mergeErr: &qualificationset.ExpectationError{Workloads: []string{"a"}}, wantStatus: 1, wantMerge: true, wantOutput: "qualification set: name=test-set"},
		{name: "invalid shards", arguments: []string{"--manifest", "/corpus.json", "/a.json"}, mergeErr: &qualificationset.InvalidReportError{Err: errors.New("shard reports omit 1 manifest workloads: b")}, wantStatus: 2, wantMerge: true, wantError: "merge qualification set shards: shard reports omit 1 manifest workloads: b"},
		{name: "output failure", arguments: []string{"--manifest", "/corpus.json", "/a.json"}, mergeErr: errors.New("write failed"), wantStatus: 3, wantMerge: true, wantOutput: "qualification set: name=test-set", wantError: "merge qualification set shards: write failed"},
		{name: "requires manifest", arguments: []string{"/a.json"}, wantStatus: 2, wantError: "merge-set requires --manifest"},
		{name: "requires shard reports", arguments: []string{"--manifest", "/corpus.json"}, wantStatus: 2, wantError: "merge-set requires --manifest"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var observed qualificationset.MergeSpec
			merged := false
			dependencies := mergeSetDependencies{merge: func(_ context.Context, spec qualificationset.MergeSpec) (qualificationset.Report, error) {
				merged, observed = true, spec
				if qualificationset.IsInvalidReport(test.mergeErr) {
					return qualificationset.Report{}, test.mergeErr
				}
				return publicSetReport(), test.mergeErr
			}}
			var stdout, stderr bytes.Buffer
			status := runMergeSetWith(test.arguments, &stdout, &stderr, dependencies)
			if status != test.wantStatus || merged != test.wantMerge || !strings.Contains(stdout.String(), test.wantOutput) || !strings.Contains(stderr.String(), test.wantError) || test.wantOutput == "" && stdout.Len() != 0 {
				t.Fatalf("status=%d merged=%t spec=%#v stdout=%q stderr=%q", status, merged, observed, stdout.String(), stderr.String())
			}
			if merged && (observed.ManifestPath != "/corpus.json" || len(observed.ShardReports) == 0 || observed.OutputPath == "") {
				t.Fatalf("merge spec = %#v", observed)
			}
		})
	}
}

func TestFullyAnsweredGuidanceReportsZeroExecutionsAndSuccessStatus(t *testing.T) {
	summary := runner.CampaignResult{CampaignPath: "/campaign", StopReason: runner.StopSeedsExhausted, Guidance: &runner.GuidanceSummary{Requested: 4, Answered: 4}}
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		if err := newExploreReporter(jsonOutput, &stdout, &stderr).Result(summary); err != nil {
			t.Fatal(err)
		}
		if exploreSummaryStatus(summary) != 0 || stderr.Len() != 0 {
			t.Fatalf("status=%d stderr=%s", exploreSummaryStatus(summary), &stderr)
		}
		if jsonOutput {
			var event exploreEvent
			if err := json.Unmarshal(stdout.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if event.Classification != "success" || event.Guidance == nil || *event.Guidance != *summary.Guidance {
				t.Fatalf("event=%#v", event)
			}
			for _, want := range []string{`"requested":4`, `"answered":4`, `"guided":0`, `"new_executions":0`} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("output=%s missing=%s", &stdout, want)
				}
			}
		} else {
			for _, want := range []string{"requested=4 answered=4 guided=0 new-executions=0", "all requested seeds are answered", "--guide-regression", "exit 0"} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("output=%s missing=%s", &stdout, want)
				}
			}
		}
	}
}

func TestGuidanceReportsNoCorpusSelectionAndNewExecutions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	summary := runner.CampaignResult{SelectionCount: 3, Attempted: 3, Succeeded: 3, StopReason: runner.StopSeedsExhausted, Guidance: &runner.GuidanceSummary{Requested: 4, Answered: 1, NewExecutions: 3}}
	if err := newExploreReporter(false, &stdout, &stderr).Result(summary); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "guidance selected no corpus seeds") || !strings.Contains(stdout.String(), "new-executions=3") {
		t.Fatalf("output=%s", &stdout)
	}
}

func TestResumeGuidanceModePreservesExplicitFalseAndTrue(t *testing.T) {
	for _, value := range []string{"false", "true"} {
		var got *bool
		dependencies := resumeDependencies{install: func(string) (installation, error) {
			return installation{toolchainRoot: "/toolchain", executable: "/gomad", runnerBuild: "runner"}, nil
		}, run: func(_ context.Context, spec runner.ResumeSpec) (runner.CampaignResult, error) {
			got = spec.GuideRegression
			return runner.CampaignResult{StopReason: runner.StopSeedsExhausted}, nil
		}}
		var stdout, stderr bytes.Buffer
		status := runResumeWith([]string{"--guide-regression=" + value, "/campaign"}, &stdout, &stderr, dependencies)
		if status != 0 || got == nil || *got != (value == "true") {
			t.Fatalf("status=%d mode=%v", status, got)
		}
	}
}

func TestGuideRegressionRequiresGuidance(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := hostApplication().runExplore([]string{"--json", "--guide-regression", "go-run", "./fixture"}, &stdout, &stderr)
	if status != 2 || !strings.Contains(stdout.String(), "--guide-regression requires --guide") {
		t.Fatalf("status=%d stdout=%s stderr=%s", status, &stdout, &stderr)
	}
}

func TestEmptyShardDoesNotReportPartiallyAnsweredPlanAsFullyAnswered(t *testing.T) {
	var stdout, stderr bytes.Buffer
	summary := runner.CampaignResult{SelectionCount: 0, StopReason: runner.StopSeedsExhausted, Guidance: &runner.GuidanceSummary{Requested: 2, Answered: 1}}
	if err := newExploreReporter(false, &stdout, &stderr).Result(summary); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "all requested seeds are answered") || !strings.Contains(stdout.String(), "requested=2 answered=1") || !strings.Contains(stdout.String(), "guidance selected no corpus seeds") {
		t.Fatalf("empty shard output=%s", &stdout)
	}
}

func TestChoiceExplorationDivergenceClassificationAndStatus(t *testing.T) {
	for _, test := range []struct {
		name           string
		summary        runner.CampaignResult
		classification string
		status         int
	}{
		{"choice confidence failure", runner.CampaignResult{Failures: 1, ReplayDivergences: 1, ChoiceExploration: &runner.ChoiceExplorationSummary{}}, "replay_divergence", 3},
		{"choice mixed failure", runner.CampaignResult{Failures: 2, ReplayDivergences: 1, ChoiceExploration: &runner.ChoiceExplorationSummary{}}, "mixed_failure", 3},
		{"confidence counter without target failure", runner.CampaignResult{ReplayDivergences: 1, ChoiceExploration: &runner.ChoiceExplorationSummary{}}, "replay_divergence", 3},
		{"world replay divergence", runner.CampaignResult{Failures: 1, ReplayDivergences: 1}, "replay_divergence", 1},
		{"target failure", runner.CampaignResult{Failures: 1}, "target_failure", 1},
		{"success", runner.CampaignResult{}, "success", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyExploreSummary(test.summary); got != test.classification {
				t.Fatalf("classification = %s", got)
			}
			if got := exploreSummaryStatus(test.summary); got != test.status {
				t.Fatalf("status = %d", got)
			}
			var stdout, stderr bytes.Buffer
			status := runResumeWith([]string{"--json", "campaign"}, &stdout, &stderr, resumeDependencies{
				install: func(string) (installation, error) {
					return installation{toolchainRoot: "toolchain", executable: "gomad", runnerBuild: "runner"}, nil
				},
				run: func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error) { return test.summary, nil },
			})
			if status != test.status {
				t.Fatalf("resume status=%d output=%s stderr=%s", status, stdout.String(), stderr.String())
			}
			var result exploreEvent
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Classification != test.classification {
				t.Fatalf("reported classification=%s", result.Classification)
			}
		})
	}
}

func TestInspectReportsCandidateReplayDivergence(t *testing.T) {
	candidate := runner.ExecutionInspection{Strategy: string(runner.StrategyChoiceExploration), Domain: "runner", Reason: "replay_divergence", Divergence: &choice.DivergenceEvidence{Ordinal: 7, Reason: choice.DivergenceAlternativeSet}}
	inspected := runner.CampaignInspection{ReplayDivergences: 1, Executions: []runner.ExecutionInspection{candidate}}
	var output bytes.Buffer
	printer := &inspectionPrinter{output: &output}
	printCampaignInspection(printer, &inspected)
	if printer.err != nil {
		t.Fatal(printer.err)
	}
	for _, expected := range []string{"reason=replay_divergence", "divergence-ordinal=7", "divergence-reason=alternative_set"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("inspect missing %s: %s", expected, output.String())
		}
	}
	encoded, err := json.Marshal(inspected)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"\"divergence\"", "\"ordinal\":7", "\"reason\":5", "\"replay_divergences\":1"} {
		if !bytes.Contains(encoded, []byte(expected)) {
			t.Fatalf("inspect JSON missing %s: %s", expected, encoded)
		}
	}
}

func TestExploreDiagnosticsRejectsForcedPrefixesBeforePreparation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := hostApplication().runExplore([]string{"--diagnostics", "--strategy=choice-exploration", "--max-executions=2", "--max-choice-depth=1", "--max-exploration-bytes=1MiB", "go-run", "./fixture"}, &stdout, &stderr)
	if status != 2 || !strings.Contains(stderr.String(), "forced-prefix exploration is unsupported") {
		t.Fatalf("status %d: %s", status, stderr.String())
	}
}

func TestReplayOutputVerifyOnly(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "healthy", true: "EBADF"}[failed], func(t *testing.T) {
			stdout := terminalDiagnostics(t, map[int]bool{1: failed})
			stderr := terminalDiagnostics(t, nil)
			installation := newFakeInstallation()
			calls := 0
			var observed runner.ReplaySpec
			dependencies := installation.replayDependencies(func(ctx context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
				calls++
				observed = spec
				if ctx != context.Background() || len(stdout.attempts) != 0 || len(stderr.attempts) != 0 || !reflect.DeepEqual(*installation.requested, []string{"/bundle"}) {
					t.Fatal("replay callback did not follow installation and precede reporting")
				}
				return runner.ReplayResult{Artifact: artifact.Artifact{Path: "/verified artifact/%s"}, Verified: true}, nil
			})
			status := runReplayWith([]string{"--toolchain-root=/bundle", "--observed=/observed", "--verify-only", "/requested-artifact"}, stdout, stderr, dependencies)
			want := runner.ReplaySpec{ArtifactPath: "/requested-artifact", VerifyOnly: true, ToolchainRoot: "/toolchain", ObservedDir: "/observed"}
			if calls != 1 || !reflect.DeepEqual(observed, want) {
				t.Fatalf("replay calls = %d, request = %#v, want %#v", calls, observed, want)
			}
			checkTerminalDiagnostics(t, stdout, []string{"gomad: verified /verified artifact/%s\n"})
			checkTerminalDiagnostics(t, stderr, nil)
			wantStatus := 0
			if failed {
				wantStatus = 3
			}
			if status != wantStatus {
				t.Fatalf("status = %d, want %d after completed verification", status, wantStatus)
			}
		})
	}
}

func TestReplayOutputEarlierErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		installErr error
		replayErr  error
		status     int
		calls      int
		diagnostic string
	}{
		{"installation", errFakeInstallation, nil, 3, 0, "resolve Gomad installation: fake installation is unavailable\n"},
		{"preflight", nil, &runner.ReplayPreflightError{Err: errors.New("toolchain changed")}, 2, 1, "incompatible replay artifact: toolchain changed\n"},
		{"replay", nil, errors.New("supervisor failed"), 3, 1, "supervisor failed\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout := terminalDiagnostics(t, map[int]bool{1: true})
			stderr := terminalDiagnostics(t, nil)
			installation := newFakeInstallation().failing(test.installErr)
			calls := 0
			dependencies := installation.replayDependencies(func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
				calls++
				want := runner.ReplaySpec{ArtifactPath: "/artifact", VerifyOnly: true, ToolchainRoot: "/toolchain"}
				if !reflect.DeepEqual(spec, want) || len(stdout.attempts) != 0 || len(stderr.attempts) != 0 {
					t.Fatalf("replay request = %#v, want %#v before reporting", spec, want)
				}
				return runner.ReplayResult{}, test.replayErr
			})
			status := runReplayWith([]string{"--toolchain-root=/bundle", "--verify-only", "/artifact"}, stdout, stderr, dependencies)
			if status != test.status || calls != test.calls || !reflect.DeepEqual(*installation.requested, []string{"/bundle"}) {
				t.Fatalf("status = %d, calls = %d, installation requests = %q", status, calls, *installation.requested)
			}
			checkTerminalDiagnostics(t, stdout, nil)
			checkTerminalDiagnostics(t, stderr, []string{test.diagnostic})
		})
	}
}

func TestExternalMinimizeForwardsProviderBoundsResumeAndSharedOutput(t *testing.T) {
	defaultRoot, err := filepath.Abs(".gomad/artifacts")
	if err != nil {
		t.Fatal(err)
	}
	provider := commandBackend{}
	for _, test := range []struct {
		name      string
		arguments []string
		want      runner.MinimizeSpec
		output    string
	}{
		{
			name: "defaults", arguments: []string{"/failure"},
			want:   runner.MinimizeSpec{Backend: provider, ArtifactPath: "/failure", OutputRoot: filepath.Join(defaultRoot, "minimized"), AttemptBudget: 64},
			output: "gomad minimize: changed=true attempts=2/64 accepted=0 stop=minimal artifact=/minimized\n",
		},
		{
			name: "resume bounded root", arguments: []string{"--resume", "--artifacts=/artifacts", "--attempt-budget=4", "--max-bytes=2MiB", "/failure"},
			want:   runner.MinimizeSpec{Backend: provider, ArtifactPath: "/failure", OutputRoot: "/artifacts/minimized", AttemptBudget: 4, MaximumBytes: 2 << 20, Resume: true},
			output: "gomad minimize: changed=true attempts=2/4 accepted=0 stop=minimal artifact=/minimized\n",
		},
		{
			name: "shared JSON", arguments: []string{"--json", "--artifacts=/artifacts", "--attempt-budget=4", "/failure"},
			want:   runner.MinimizeSpec{Backend: provider, ArtifactPath: "/failure", OutputRoot: "/artifacts/minimized", AttemptBudget: 4},
			output: "{\"artifact_path\":\"/minimized\",\"record_hash\":\"\",\"changed\":true,\"attempts\":2,\"attempt_budget\":4,\"accepted\":null,\"stop_reason\":\"minimal\"}\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var observed []runner.MinimizeSpec
			var stdout, stderr bytes.Buffer
			status := runMinimizeWith(test.arguments, &stdout, &stderr, minimizeDependencies{
				install: func(root string) (installation, error) {
					if root != "" {
						t.Fatalf("external minimization selected native toolchain: %s", root)
					}
					return installation{backend: provider, backendName: "wasm"}, nil
				},
				minimize: func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
					observed = append(observed, spec)
					return runner.MinimizeResult{Artifact: artifact.Artifact{Path: "/minimized"}, Changed: true, Attempts: 2, AttemptBudget: test.want.AttemptBudget, StopReason: "minimal"}, nil
				},
			})
			if status != 0 || stderr.Len() != 0 || len(observed) != 1 || !reflect.DeepEqual(observed[0], test.want) || stdout.String() != test.output {
				t.Fatalf("status=%d requests=%#v want=%#v stdout=%q stderr=%q", status, observed, test.want, stdout.String(), stderr.String())
			}
		})
	}
}

func TestExternalMinimizePreservesErrorClassification(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
	}{
		{"preflight", &runner.ReplayPreflightError{Err: errors.New("incompatible retained choice failure")}, 2},
		{"runner", errors.New("minimization infrastructure failure"), 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := runMinimizeWith([]string{"/failure"}, &stdout, &stderr, minimizeDependencies{
				install: func(string) (installation, error) {
					return installation{backend: commandBackend{}, backendName: "wasm"}, nil
				},
				minimize: func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
					if spec.Backend == nil || spec.ToolchainRoot != "" {
						t.Fatalf("external backend lost or native launch configured: %+v", spec)
					}
					return runner.MinimizeResult{}, test.err
				},
			})
			if status != test.status || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.err.Error()) {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

type terminalDiagnosticWriter struct {
	output   bytes.Buffer
	file     *os.File
	failures map[int]bool
	attempts []string
	errors   []error
}

func (writer *terminalDiagnosticWriter) Write(contents []byte) (int, error) {
	writer.attempts = append(writer.attempts, string(contents))
	if writer.failures[len(writer.attempts)] {
		written, err := writer.file.Write(contents)
		writer.errors = append(writer.errors, err)
		return written, err
	}
	writer.errors = append(writer.errors, nil)
	return writer.output.Write(contents)
}

func terminalDiagnostics(t *testing.T, failures map[int]bool) *terminalDiagnosticWriter {
	t.Helper()
	path := filepath.Join(t.TempDir(), "read-only")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	return &terminalDiagnosticWriter{file: file, failures: failures}
}

func checkTerminalDiagnostics(t *testing.T, writer *terminalDiagnosticWriter, want []string) {
	t.Helper()
	if !reflect.DeepEqual(writer.attempts, want) {
		t.Fatalf("attempts = %q, want %q", writer.attempts, want)
	}
	var delivered strings.Builder
	for index, contents := range want {
		if writer.failures[index+1] {
			if !errors.Is(writer.errors[index], syscall.EBADF) {
				t.Fatalf("attempt %d error = %v, want actual EBADF", index+1, writer.errors[index])
			}
		} else {
			if writer.errors[index] != nil {
				t.Fatal(writer.errors[index])
			}
			delivered.WriteString(contents)
		}
	}
	if writer.output.String() != delivered.String() {
		t.Fatalf("delivered = %q, want %q", writer.output.String(), delivered.String())
	}
	contents, err := os.ReadFile(writer.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 0 {
		t.Fatalf("read-only file received %q", contents)
	}
}

func TestTerminalDiagnosticsPublicFailures(t *testing.T) {
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "relative")
	root := t.TempDir()
	plan := filepath.Join(root, "plan.json")
	if err := os.WriteFile(plan, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		arguments  []string
		status     int
		diagnostic string
		reporter   bool
	}{
		{"usage", nil, 2, usage, false},
		{"unknown", []string{"unknown"}, 2, "unknown gomad command \"unknown\"\n" + usage, false},
		{"doctor usage", []string{"doctor", "extra"}, 2, usage, false},
		{"replay usage", []string{"replay"}, 2, usage, false},
		{"shard usage", []string{"execute-shard"}, 2, usage, false},
		{"shard syntax", []string{"execute-shard", "--shard=1", plan}, 2, "invalid shard \"1\": want zero-based INDEX/COUNT\n", false},
		{"merge usage", []string{"merge"}, 2, usage, false},
		{"merge invalid plan", []string{"merge", "--output=" + filepath.Join(root, "merged"), plan, filepath.Join(root, "shard")}, 2, "gomad: invalid_input: merge campaign shards: decode campaign plan: JSON is not canonical\n", false},
		{"explore output", []string{"explore", "--output=plan", "go-run", "./pkg"}, 2, "gomad: invalid_input: --output is only valid with gomad plan\n", true},
		{"explore capability", []string{"explore", "--capability-mode=open"}, 2, "gomad: invalid_input: unknown capability mode \"open\"\n", true},
		{"explore strategy", []string{"explore", "--strategy=random"}, 2, "gomad: invalid_input: unknown exploration strategy \"random\"\n", true},
		{"explore seeds", []string{"explore", "--count=0"}, 2, "gomad: invalid_input: --count must be greater than zero\n", true},
		{"explore regression", []string{"explore", "--guide-regression"}, 2, "gomad: invalid_input: --guide-regression requires --guide\n", true},
		{"explore guidance", []string{"explore", "--guide"}, 2, "gomad: invalid_input: --guide requires --corpus DIR\n", true},
		{"explore coverage", []string{"explore", "--require-probe=stdlib.os.openfile"}, 2, "gomad: invalid_input: --require-probe requires --coverage=semantic\n", true},
		{"explore choices", []string{"explore", "--choice-bytes=1MiB"}, 2, "gomad: invalid_input: --choice-bytes requires --choices\n", true},
		{"explore target", []string{"explore"}, 2, "gomad: invalid_input: target kind is required\n", true},
		{"explore working directory", []string{"explore", "--working-dir=module", "go-run", "./pkg"}, 2, "gomad: invalid_input: working directory \"module\" must be an absolute, clean path\n", true},
		{"qualify input", []string{"qualify", "--repeat=0"}, 2, "gomad: invalid_input: --repeat must be between 2 and 32\n", true},
		{"resume input", []string{"resume"}, 2, "gomad: invalid_input: resume requires one interrupted campaign path\n", true},
		{"replay installation", []string{"replay", plan}, 3, "resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", false},
		{"shard installation", []string{"execute-shard", "--shard=0/1", plan}, 3, "resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", false},
		{"explore installation", []string{"explore", "go-run", "./pkg"}, 3, "gomad: runner_failure: resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", true},
		{"qualify installation", []string{"qualify", "go-run", "./pkg"}, 3, "gomad: runner_failure: resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", true},
		{"resume installation", []string{"resume", plan}, 3, "gomad: runner_failure: resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, failures := range []map[int]bool{nil, {1: true}, {1: true, 2: true}} {
				stdout := terminalDiagnostics(t, nil)
				stderr := terminalDiagnostics(t, failures)
				status := Run(test.arguments, stdout, stderr)
				want, wantStatus := []string{test.diagnostic}, test.status
				if failures[1] && test.reporter {
					want = append(want, "write "+stderr.file.Name()+": bad file descriptor\n")
					wantStatus = 3
				}
				if status != wantStatus || len(stdout.attempts) != 0 {
					t.Fatalf("status = %d, want %d; stdout attempts = %q", status, wantStatus, stdout.attempts)
				}
				checkTerminalDiagnostics(t, stderr, want)
				contents, err := os.ReadFile(plan)
				if err != nil || string(contents) != "{}\n" {
					t.Fatalf("plan = %q, %v", contents, err)
				}
				entries, err := os.ReadDir(root)
				if err != nil || len(entries) != 1 || entries[0].Name() != "plan.json" {
					t.Fatalf("publication entries = %v, %v", entries, err)
				}
			}
		})
	}
}

func TestTerminalDiagnosticsPrivateDispatch(t *testing.T) {
	for _, mode := range []string{coordinatorMode, supervisorMode, targetBootstrapMode} {
		t.Run(mode, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				stdout := terminalDiagnostics(t, nil)
				stderr := terminalDiagnostics(t, map[int]bool{1: failed})
				input, output := strings.NewReader("request"), new(bytes.Buffer)
				calls := 0
				app := application{privateInput: input, privateOutput: output, dispatch: func(got string, in io.Reader, out io.Writer) error {
					if got != mode || in != input || out != output || len(stderr.attempts) != 0 {
						t.Fatal("private dispatch streams or ordering changed")
					}
					calls++
					return errors.New("decode private request: EOF")
				}}
				if status := app.run([]string{mode, "ignored"}, stdout, stderr); status != 3 || calls != 1 || len(stdout.attempts) != 0 || output.Len() != 0 {
					t.Fatalf("status = %d, calls = %d, stdout = %q, private output = %q", status, calls, stdout.attempts, output.String())
				}
				checkTerminalDiagnostics(t, stderr, []string{"decode private request: EOF\n"})
			}
		})
	}
}

func TestTerminalDiagnosticsReplayClassifications(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		text   string
	}{
		{"preflight", &runner.ReplayPreflightError{Err: errors.New("invalid retained artifact")}, 2, "incompatible replay artifact: invalid retained artifact\n"},
		{"host", &runner.HostError{Reason: "replay", Err: errors.New("child failed")}, 3, "gomad3 Runner/host failure: replay: child failed\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				stdout := terminalDiagnostics(t, nil)
				stderr := terminalDiagnostics(t, map[int]bool{1: failed})
				calls := 0
				dependencies := newFakeInstallation().replayDependencies(func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
					if spec.ArtifactPath != "/artifact" || len(stderr.attempts) != 0 {
						t.Fatal("replay operation or ordering changed")
					}
					calls++
					return runner.ReplayResult{}, test.err
				})
				if status := runReplayWith([]string{"/artifact"}, stdout, stderr, dependencies); status != test.status || calls != 1 || len(stdout.attempts) != 0 {
					t.Fatalf("status = %d, calls = %d, stdout = %q", status, calls, stdout.attempts)
				}
				checkTerminalDiagnostics(t, stderr, []string{test.text})
			}
		})
	}
}

func TestTerminalDiagnosticsCompletedOperations(t *testing.T) {
	for _, command := range []string{"explore", "execute-shard", "resume", "replay", "qualify"} {
		t.Run(command, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				root := t.TempDir()
				published := filepath.Join(root, "completed")
				stdout := terminalDiagnostics(t, map[int]bool{1: true})
				stderr := terminalDiagnostics(t, map[int]bool{1: failed})
				completed := 0
				finish := func() {
					if len(stdout.attempts) != 0 || len(stderr.attempts) != 0 {
						t.Fatal("reporting preceded the operation")
					}
					if err := os.WriteFile(published, []byte("committed\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					completed++
				}
				result := runner.CampaignResult{CampaignPath: published, Attempted: 1, Succeeded: 1}
				fake := newFakeInstallation()
				var status int
				var wantOutput string
				switch command {
				case "explore":
					dependencies := fake.exploreDependencies(func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
						finish()
						return result, nil
					}, nil)
					status = runExploreWith([]string{"go-run", "./pkg"}, stdout, stderr, dependencies)
				case "execute-shard":
					dependencies := fake.campaignShardDependencies(func(context.Context, runner.CampaignShardSpec) (runner.CampaignResult, error) {
						finish()
						return result, nil
					})
					status = runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, dependencies)
				case "resume":
					dependencies := fake.resumeDependencies(func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error) {
						finish()
						return result, nil
					})
					status = runResumeWith([]string{"/campaign"}, stdout, stderr, dependencies)
				case "replay":
					dependencies := fake.replayDependencies(func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
						finish()
						return runner.ReplayResult{Match: true}, nil
					})
					status = runReplayWith([]string{"/artifact"}, stdout, stderr, dependencies)
					wantOutput = "gomad: reproduced=true diagnostic=false result=target_failure choice-replay=\n"
				case "qualify":
					dependencies := qualifyDependencies{
						install: fake.install, workingDirectory: func() (string, error) { return "/workspace", nil },
						workload: func(context.Context, qualificationworkload.Spec) (qualificationworkload.Result, error) {
							finish()
							return qualificationworkload.Result{ReportPath: published, Report: qualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true}}, nil
						},
					}
					status = runQualifyWith([]string{"go-run", "./pkg"}, stdout, stderr, dependencies)
					wantOutput = "gomad: qualification qualified=true deterministic=true target-success=true seed=0 repeat=0 report=" + published + "\n"
				}
				if wantOutput == "" {
					wantOutput = "gomad: classification=success attempted=1 succeeded=1 failures=0 watchdogs=0 replay-divergences=0 distinct=0 retained-successes=0 retained-success-bytes=0 stop= artifact=" + published + "\n"
				}
				if status != 3 || completed != 1 {
					t.Fatalf("status = %d, completed = %d", status, completed)
				}
				checkTerminalDiagnostics(t, stdout, []string{wantOutput})
				checkTerminalDiagnostics(t, stderr, []string{"write " + stdout.file.Name() + ": bad file descriptor\n"})
				contents, err := os.ReadFile(published)
				if err != nil || string(contents) != "committed\n" {
					t.Fatalf("completed operation file = %q, %v", contents, err)
				}
			}
		})
	}
}

func TestCampaignSemanticValidationBelongsToRunner(t *testing.T) {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "cli.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []struct {
		function string
		calls    map[string]int
		forbid   map[string]bool
	}{
		{"resolveExploreStrategy", map[string]int{"ParseSingleBaseSeed": 2}, map[string]bool{"Count": true, "ParseSeeds": true}},
		{"resolveExploreCoverage", map[string]int{"ValidateCoverage": 1}, map[string]bool{"MissingRequiredSemanticProbes": true, "CoverageNone": true, "CoverageChoice": true, "CoverageSemantic": true, "CoverageSemanticChoice": true}},
		{"resolveChoiceTrace", map[string]int{"ValidateChoiceTraceLimit": 1}, nil},
		{"parseCampaignRequest", map[string]int{"ValidateChoiceCoverage": 1}, map[string]bool{"CoverageChoice": true, "CoverageSemanticChoice": true}},
		{"resolveExploreGuidance", map[string]int{"NormalizeCoverage": 1}, nil},
	} {
		t.Run(rule.function, func(t *testing.T) {
			var body *ast.BlockStmt
			for _, declaration := range file.Decls {
				if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == rule.function {
					body = function.Body
				}
			}
			if body == nil {
				t.Fatal("campaign validation function is missing")
			}
			calls := make(map[string]int)
			ast.Inspect(body, func(node ast.Node) bool {
				if comparison, ok := node.(*ast.BinaryExpr); ok && rule.function == "resolveChoiceTrace" {
					ast.Inspect(comparison, func(node ast.Node) bool {
						if selector, ok := node.(*ast.SelectorExpr); ok && (selector.Sel.Name == "MinimumChoiceTraceBytes" || selector.Sel.Name == "MaximumChoiceTraceBytes") {
							t.Errorf("%s: CLI duplicates Runner's trace capacity rule", files.Position(selector.Pos()))
						}
						return true
					})
				}
				if selector, ok := node.(*ast.SelectorExpr); ok && rule.forbid[selector.Sel.Name] {
					t.Errorf("%s: CLI duplicates Runner's semantic responsibility through %s", files.Position(selector.Pos()), selector.Sel.Name)
				}
				var expressions []ast.Expr
				switch value := node.(type) {
				case *ast.IfStmt:
					condition, ok := value.Cond.(*ast.BinaryExpr)
					if !ok || condition.Op != token.NEQ {
						break
					}
					left, leftOK := condition.X.(*ast.Ident)
					right, rightOK := condition.Y.(*ast.Ident)
					if !leftOK || !rightOK || left.Name != "err" || right.Name != "nil" {
						break
					}
					if assignment, ok := value.Init.(*ast.AssignStmt); ok {
						expressions = assignment.Rhs
					}
				case *ast.ReturnStmt:
					expressions = value.Results
				}
				for _, expression := range expressions {
					ast.Inspect(expression, func(node ast.Node) bool {
						if call, ok := node.(*ast.CallExpr); ok {
							if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
								if owner, ok := selector.X.(*ast.Ident); ok && owner.Name == "runner" {
									calls[selector.Sel.Name]++
								}
							}
						}
						return true
					})
				}
				return true
			})
			for name, want := range rule.calls {
				if got := calls[name]; got != want {
					t.Errorf("%s: %d checked Runner.%s calls, want %d in actual validation", rule.function, got, name, want)
				}
			}
		})
	}
}
