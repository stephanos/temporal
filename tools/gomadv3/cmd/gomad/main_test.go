package main

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

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/qualify"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/replay"
	"go.temporal.io/server/tools/gomadv3/internal/runner"
	"go.temporal.io/server/tools/gomadv3/internal/target"
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

func TestResolveExploreCoverageRequiresSemanticModeAndKnownProbes(t *testing.T) {
	for _, test := range []struct {
		mode      string
		required  []string
		want      runner.CoverageMode
		wantError bool
	}{
		{mode: "none", want: runner.CoverageNone},
		{mode: "semantic", required: []string{"stdlib.os.openfile"}, want: runner.CoverageSemantic},
		{mode: "none", required: []string{"stdlib.os.openfile"}, wantError: true},
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

func TestRunRejectsUnknownCommandWithUsageStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"unknown"}, &stdout, &stderr); status != 2 {
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
		{name: "json", arguments: []string{"--json", path}, want: []string{`"schema":"gomadv3.inspect/v1"`, `"kind":"batch"`, `"run_id":"run-inspect-command"`}},
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

func writeInspectBatchFixture(t *testing.T) string {
	t.Helper()
	journal, err := artifact.NewBatchJournal(context.Background(), artifact.BatchConfig{
		Root: t.TempDir(), RunID: "run-inspect-command", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = journal.Close() })
	if err := journal.StartRuns(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendRun(artifact.RunRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(artifact.BatchSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	return journal.Path()
}

func TestExploreReporterEmitsStableJSONEventsAndEveryArtifact(t *testing.T) {
	var stdout, stderr bytes.Buffer
	reporter := newExploreReporter(true, &stdout, &stderr)
	if err := reporter.Progress(runner.Progress{
		Phase: runner.ProgressPreparing, BatchPath: "/batch", Selected: 3,
	}); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Result(runner.Summary{
		BatchPath: "/batch", SelectionCount: 3, Attempted: 3, Succeeded: 1, Failures: 2, DistinctFailures: 2,
		StopReason: runner.StopSeedsExhausted, Artifacts: []string{"/batch/failures/one", "/batch/failures/two"},
		SemanticCoverage: &ioprofile.SemanticCoverage{Schema: ioprofile.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	for _, value := range []string{
		`"schema":"gomadv3.explore-event/v1"`, `"type":"progress"`, `"phase":"preparing"`,
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
		if err := reporter.Result(runner.Summary{
			BatchPath: "/batch", SelectionCount: 1, Attempted: 1, Succeeded: 1, RetainedSuccesses: 1,
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
		if err := reporter.Result(runner.Summary{
			BatchPath: "/batch", SelectionCount: 4, Attempted: 4, Succeeded: 4, StopReason: runner.StopSeedsExhausted,
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

func TestRunExploreReportsFlagErrorsAsJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := runExplore([]string{"--json", "--parallel", "invalid"}, &stdout, &stderr)
	if status != 2 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
	for _, value := range []string{`"schema":"gomadv3.explore-event/v1"`, `"type":"error"`, `"classification":"invalid_input"`} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("explore output = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestRunQualifyRepeatsOneSeedAndRetainsJSONReport(t *testing.T) {
	var calls int
	var configs []runner.Config
	var retained qualify.Report
	var resolvedToolchainRoot string
	dependencies := qualifyDependencies{
		identity: func(explicitToolchainRoot string) (string, string, string, error) {
			resolvedToolchainRoot = explicitToolchainRoot
			return "/toolchain", "/bin/gomad", "sha256:runner", nil
		},
		workingDirectory: func() (string, error) { return "/workspace", nil },
		run: func(_ context.Context, config runner.Config) (runner.Summary, error) {
			calls++
			configs = append(configs, config)
			evidence := qualificationEvidence(7)
			return runner.Summary{BatchPath: fmt.Sprintf("/artifacts/run-%d", calls), SelectionCount: 1, Attempted: 1, Succeeded: 1, RunEvidence: &evidence}, nil
		},
		replay: func(context.Context, replay.Config) (replay.Result, error) {
			t.Fatal("unexpected replay")
			return replay.Result{}, nil
		},
		write: func(_ string, report qualify.Report) (string, error) {
			retained = report
			return "/artifacts/qualifications/v1/report.json", nil
		},
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{
		"--json", "--seed", "7", "--repeat", "2", "--artifacts", "/artifacts", "--toolchain-root", "/bundle/toolchain", "--require-probe", "stdlib.os.openfile",
		"go-test", "./pkg", "--", "-test.run=TestScenario",
	}, &stdout, &stderr, dependencies)
	if status != 0 || stderr.Len() != 0 || calls != 2 || !retained.Qualified || resolvedToolchainRoot != "/bundle/toolchain" {
		t.Fatalf("status=%d calls=%d report=%#v stdout=%q stderr=%q", status, calls, retained, stdout.String(), stderr.String())
	}
	for _, config := range configs {
		if config.Seeds != "7" || config.Parallel != 1 || config.OnFailure != runner.PolicyAll || config.Coverage != runner.CoverageSemantic || !config.CollectRunEvidence || config.Target.Source != "./pkg" || config.Target.WorkingDir != "/workspace" || len(config.RequiredSemanticProbes) != 1 {
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
	var retained qualify.Report
	dependencies := qualificationDependencies(t)
	dependencies.run = func(_ context.Context, _ runner.Config) (runner.Summary, error) {
		calls++
		evidence := qualificationEvidence(7)
		if calls == 2 {
			evidence.Stdout.FullSHA256 = record.HashBytes([]byte("different"))
		}
		return runner.Summary{BatchPath: fmt.Sprintf("/artifacts/run-%d", calls), SelectionCount: 1, Attempted: 1, Succeeded: 1, RunEvidence: &evidence}, nil
	}
	dependencies.write = func(_ string, report qualify.Report) (string, error) { retained = report; return "/report.json", nil }
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--seed", "7", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 1 || retained.Deterministic || retained.FirstDivergence != "stdout.full_sha256" || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"classification":"nondeterministic"`) {
		t.Fatalf("status=%d report=%#v stdout=%q stderr=%q", status, retained, stdout.String(), stderr.String())
	}
}

func TestRunQualifyReplaysRepeatedTargetFailure(t *testing.T) {
	var calls, replayCalls int
	var retained qualify.Report
	dependencies := qualificationDependencies(t)
	dependencies.run = func(_ context.Context, _ runner.Config) (runner.Summary, error) {
		calls++
		evidence := qualificationEvidence(7)
		evidence.Outcome = runner.OutcomeEvidence{Domain: "target", Reason: "nonzero_exit", Termination: "exit"}
		return runner.Summary{BatchPath: fmt.Sprintf("/artifacts/run-%d", calls), SelectionCount: 1, Attempted: 1, Failures: 1, Artifacts: []string{fmt.Sprintf("/artifacts/failure-%d", calls)}, RunEvidence: &evidence}, nil
	}
	dependencies.replay = func(_ context.Context, config replay.Config) (replay.Result, error) {
		replayCalls++
		if config.ArtifactPath != "/artifacts/failure-1" {
			t.Fatalf("replay config = %#v", config)
		}
		return replay.Result{Match: true}, nil
	}
	dependencies.write = func(_ string, report qualify.Report) (string, error) { retained = report; return "/report.json", nil }
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--seed", "7", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 1 || calls != 2 || replayCalls != 1 || retained.Replay == nil || !retained.Replay.Match || retained.TargetSuccess || !strings.Contains(stdout.String(), `"classification":"target_failure"`) {
		t.Fatalf("status=%d calls=%d replay=%d report=%#v stdout=%q stderr=%q", status, calls, replayCalls, retained, stdout.String(), stderr.String())
	}
}

func TestRunQualifyRetainsUnsupportedBoundary(t *testing.T) {
	var retained qualify.Report
	dependencies := qualificationDependencies(t)
	dependencies.run = func(_ context.Context, _ runner.Config) (runner.Summary, error) {
		unsupported := &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}
		return runner.Summary{BatchPath: "/artifacts/run-1"}, &runner.HostError{Reason: "target_preparation", Err: unsupported}
	}
	dependencies.write = func(_ string, report qualify.Report) (string, error) { retained = report; return "/report.json", nil }
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--seed", "7", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 2 || retained.Failure == nil || retained.Failure.Capability != "imports os/exec" || !strings.Contains(stdout.String(), `"classification":"unsupported_target"`) || stderr.Len() != 0 {
		t.Fatalf("status=%d report=%#v stdout=%q stderr=%q", status, retained, stdout.String(), stderr.String())
	}
}

func TestRunQualifyRejectsUnboundedRepeat(t *testing.T) {
	dependencies := qualificationDependencies(t)
	dependencies.run = func(context.Context, runner.Config) (runner.Summary, error) {
		t.Fatal("unexpected run")
		return runner.Summary{}, nil
	}
	var stdout, stderr bytes.Buffer
	status := runQualifyWith([]string{"--json", "--repeat", "33", "go-test", "./pkg"}, &stdout, &stderr, dependencies)
	if status != 2 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"classification":"invalid_input"`) {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestRunResumeUsesStoredBatchAndReportsResult(t *testing.T) {
	var got runner.Config
	var resolvedToolchainRoot string
	dependencies := resumeDependencies{
		identity: func(explicitToolchainRoot string) (string, string, string, error) {
			resolvedToolchainRoot = explicitToolchainRoot
			return "/toolchain", "/bin/gomad", "sha256:runner", nil
		},
		run: func(_ context.Context, config runner.Config) (runner.Summary, error) {
			got = config
			return runner.Summary{BatchPath: "/artifacts/v1/run-partial", SelectionCount: 3, Attempted: 3, Succeeded: 3, StopReason: runner.StopSeedsExhausted}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	status := runResumeWith([]string{"--json", "--toolchain-root", "/bundle/toolchain", "/artifacts/v1/run-partial"}, &stdout, &stderr, dependencies)
	if status != 0 || stderr.Len() != 0 || resolvedToolchainRoot != "/bundle/toolchain" || got.ResumeBatch != "/artifacts/v1/run-partial" || got.RunnerBuild != "sha256:runner" || got.Target.ToolchainRoot != "/toolchain" || len(got.CoordinatorCommand) != 2 {
		t.Fatalf("status=%d config=%#v stdout=%q stderr=%q", status, got, stdout.String(), stderr.String())
	}
	for _, want := range []string{`"schema":"gomadv3.explore-event/v1"`, `"type":"result"`, `"classification":"success"`, `"batch_path":"/artifacts/v1/run-partial"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("output = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestRunResumeClassifiesInvalidJournalAsInputError(t *testing.T) {
	dependencies := resumeDependencies{
		identity: func(string) (string, string, string, error) { return "/toolchain", "/bin/gomad", "sha256:runner", nil },
		run: func(context.Context, runner.Config) (runner.Summary, error) {
			return runner.Summary{}, &runner.HostError{Reason: "resume_setup", Err: errors.New("batch plan changed")}
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
		run: func(context.Context, runner.Config) (runner.Summary, error) {
			t.Fatal("qualification runner is not configured")
			return runner.Summary{}, nil
		},
		replay: func(context.Context, replay.Config) (replay.Result, error) {
			t.Fatal("unexpected replay")
			return replay.Result{}, nil
		},
		write: func(string, qualify.Report) (string, error) {
			t.Fatal("qualification writer is not configured")
			return "", nil
		},
	}
}

func qualificationEvidence(seed uint64) runner.RunEvidence {
	return runner.RunEvidence{
		Schema: runner.RunEvidenceSchema, Seed: record.Uint64String(seed), RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomadv3-target"}, BuildTags: []string{"gomad_fixture"}},
		IOProfile:   runner.IOProfileEvidence{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: fmt.Sprintf("%d", seed)}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: ioprofile.SemanticCoverage{Schema: ioprofile.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}
}

func TestExploreReporterHumanOutputIncludesProgressAndReplayCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	reporter := newExploreReporter(false, &stdout, &stderr)
	if err := reporter.Progress(runner.Progress{
		Phase: runner.ProgressRunning, BatchPath: "/batch", Selected: 5, Attempted: 2, Running: 2, Succeeded: 1, Failures: 1, DistinctFailures: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Result(runner.Summary{
		BatchPath: "/batch", SelectionCount: 5, Attempted: 5, Succeeded: 4, Failures: 1, DistinctFailures: 1,
		StopReason: runner.StopSeedsExhausted, Artifacts: []string{"/batch/failures/one"},
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "attempted=2 running=2") || !strings.Contains(stdout.String(), "retained failure: /batch/failures/one") || !strings.Contains(stdout.String(), "gomad replay /batch/failures/one") {
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
		{err: &ioprofile.MissingSemanticProbesError{Probes: []string{"stdlib.os.openfile"}}, want: "semantic_coverage_failure"},
		{err: &runner.HostError{Reason: "coordinator_exit", Err: os.ErrClosed}, want: "runner_failure"},
	} {
		if got := classifyExploreError(test.err); got != test.want {
			t.Fatalf("classifyExploreError(%T) = %q, want %q", test.err, got, test.want)
		}
	}
}

func TestClassifyExploreSummaryDistinguishesWatchdogAndReplayDivergence(t *testing.T) {
	for _, test := range []struct {
		summary runner.Summary
		want    string
	}{
		{summary: runner.Summary{Failures: 1, Watchdogs: 1}, want: "watchdog_observation"},
		{summary: runner.Summary{Failures: 1, ReplayDivergences: 1}, want: "replay_divergence"},
		{summary: runner.Summary{Failures: 2, Watchdogs: 1}, want: "mixed_failure"},
		{summary: runner.Summary{Failures: 1}, want: "target_failure"},
	} {
		if got := classifyExploreSummary(test.summary); got != test.want {
			t.Fatalf("classifyExploreSummary(%#v) = %q, want %q", test.summary, got, test.want)
		}
	}
}

func TestReportReplayResultStatesWhetherFailureWasReproduced(t *testing.T) {
	for _, test := range []struct {
		name   string
		result replay.Result
		want   string
		status int
	}{
		{name: "success", result: replay.Result{Artifact: artifact.Artifact{Manifest: record.Manifest{Outcome: record.Outcome{Domain: "success"}}}, Match: true}, want: "reproduced=true diagnostic=false result=success", status: 0},
		{name: "target failure", result: replay.Result{Match: true}, want: "reproduced=true diagnostic=false result=target_failure", status: 1},
		{name: "watchdog observation", result: replay.Result{Match: true, Diagnostic: true}, want: "reproduced=true diagnostic=true result=watchdog_observation", status: 1},
		{name: "divergence", result: replay.Result{Divergence: "stdout.full_sha256"}, want: "reproduced=false divergence=stdout.full_sha256", status: 1},
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
