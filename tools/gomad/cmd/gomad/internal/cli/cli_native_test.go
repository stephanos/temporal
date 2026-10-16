//go:build gomad_native

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	nativebuild "go.temporal.io/server/tools/gomad/internal/backends/native/build"
	"go.temporal.io/server/tools/gomad/internal/backends/native/preparation"
	capabilityanalysis "go.temporal.io/server/tools/gomad/internal/backends/native/qualification/analysis"
	supportcomparison "go.temporal.io/server/tools/gomad/internal/backends/native/qualification/comparison"
	qualificationset "go.temporal.io/server/tools/gomad/internal/backends/native/qualification/set"
	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/qualification"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

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
			wantStdout: []string{`"schema":"gomad.capability-analysis/v1"`, `"classification":"supported"`},
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

func TestRunDoctorReportsAvailableContractAsJSON(t *testing.T) {
	executable, artifacts := writeDoctorCommandFixture(t)
	var stdout, stderr bytes.Buffer
	status := application{executable: func() (string, error) { return executable, nil }, environment: os.Getenv}.runDoctor([]string{"--json", "--artifacts", artifacts}, &stdout, &stderr)
	if status != 0 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
	for _, value := range []string{`"schema":"gomad.doctor/v3"`, `"available":true`, `"boundary_manifest_version":`, `"adapters":[`, `"installation_source":"adjacent"`, `"repair_instruction":`} {
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
	if status != 1 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "available=false") || !strings.Contains(stdout.String(), "set GOMAD_TOOLCHAIN_DIR") {
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
			wantOutput: `"schema":"gomad.qualification-set-report/v1"`,
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
		{name: "merged as json", arguments: []string{"--manifest", "/corpus.json", "--format", "json", "/a.json"}, wantMerge: true, wantOutput: `"schema":"gomad.qualification-set-report/v1"`},
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

func TestResolveExploreCoverageRequiresSemanticModeAndKnownProbes(t *testing.T) {
	for _, test := range []struct {
		mode      string
		required  []string
		want      campaignoptions.CoverageMode
		wantError bool
	}{
		{mode: "none", want: campaignoptions.CoverageNone},
		{mode: "semantic", required: []string{"stdlib.os.openfile"}, want: campaignoptions.CoverageSemantic},
		{mode: "choice", want: campaignoptions.CoverageChoice},
		{mode: "semantic+choice", required: []string{"stdlib.os.openfile"}, want: campaignoptions.CoverageSemanticChoice},
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

func TestTerminalDiagnosticsPublicFailures(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "relative")
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
		{"merge invalid plan", []string{"merge", "--output=" + filepath.Join(root, "merged"), plan, filepath.Join(root, "shard")}, 2, "gomad: invalid_input: merge campaign shards: campaign plan protocol identity is invalid\n", false},
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
		{"replay installation", []string{"replay", plan}, 3, "resolve Gomad installation: GOMAD_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", false},
		{"shard installation", []string{"execute-shard", "--shard=0/1", plan}, 3, "resolve Gomad installation: GOMAD_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", false},
		{"explore installation", []string{"explore", "go-run", "./pkg"}, 3, "gomad: runner_failure: resolve Gomad installation: GOMAD_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", true},
		{"qualify installation", []string{"qualify", "go-run", "./pkg"}, 3, "gomad: runner_failure: resolve Gomad installation: GOMAD_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", true},
		{"resume installation", []string{"resume", plan}, 3, "gomad: runner_failure: resolve Gomad installation: GOMAD_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", true},
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
			wantOutput: []string{`"schema":"gomad.qualify-event/v1"`, `"type":"result"`, `"classification":"qualified"`, `"report_path":"/artifacts/qualifications/v1/report.json"`, `"qualified":true`},
			check: func(t *testing.T, observed qualifyObservation) bool {
				for _, config := range observed.runs {
					if config.Seeds != "7" || config.Parallel != 1 || config.OnFailure != runner.PolicyAll || config.Coverage != campaignoptions.CoverageSemanticChoice || !config.CollectExecutionEvidence || config.ChoiceTraceLimit != 1<<20 || config.Target.Source != "./pkg" || config.Target.WorkingDir != "/workspace" || len(config.RequiredSemanticProbes) != 1 {
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
				evidence.Outcome = record.OutcomeEvidence{Domain: "target", Reason: "nonzero_exit", Termination: "exit"}
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
