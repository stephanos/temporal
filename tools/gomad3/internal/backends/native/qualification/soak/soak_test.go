package soak

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/evidence/qualification"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

func evidence(seed uint64, buildKey string) record.ExecutionEvidence {
	return record.ExecutionEvidence{
		Schema: record.ExecutionEvidenceSchema, Seed: record.Uint64String(seed), RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.27.1", BuildKey: buildKey, TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomad3-target"}, BuildTags: []string{}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}},
		IOProfile:   record.IOContract{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: strconv.FormatUint(seed, 10)}, {Name: "TZ", Value: "UTC"}},
		Outcome:     record.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: record.SemanticCoverage{Schema: record.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{}},
	}
}

// diagnosticTrace writes a one-record diagnostic trace whose seeded-stream
// draw counter is draw.
func diagnosticTrace(t *testing.T, directory string, draw byte) *record.DiagnosticTraceReference {
	t.Helper()
	data := make([]byte, 160)
	copy(data, []byte{'G', 'O', 'M', 'A', 'D', 'D', 'G', 1})
	binary.BigEndian.PutUint32(data[8:12], 1)
	data[12] = 1
	binary.BigEndian.PutUint64(data[16:24], 160)
	binary.BigEndian.PutUint64(data[24:32], 160)
	binary.BigEndian.PutUint64(data[32:40], 1)
	data[143] = draw
	trace, err := choice.DecodeDiagnosticTrace(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "trace-"+strconv.Itoa(int(draw))+".bin")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return &record.DiagnosticTraceReference{Path: path, DiagnosticEvidence: record.DiagnosticEvidence{Profile: choice.DiagnosticProfile, SHA256: record.SHA256FromSum(trace.SHA256), Records: 1}}
}

func key(t *testing.T, buildKey string) CohortKey {
	t.Helper()
	identity, err := executionIdentity(evidence(11, buildKey))
	if err != nil {
		t.Fatal(err)
	}
	return CohortKey{Workload: "w", Seed: 11, Platform: "darwin/arm64", ExecutionIdentity: identity}
}

func TestCohortBatchSequenceAAThenBBIsADivergence(t *testing.T) {
	ledger := newLedger()
	cohortKey := key(t, "one")
	var outcomes, comparisons []string
	for batch, digest := range []record.SHA256{"sha256:a", "sha256:a", "sha256:b", "sha256:b"} {
		outcome, comparison, _ := ledger.observe(observation{run: "1", batch: uint64(batch + 1), key: cohortKey, outcome: OutcomeClean, digest: digest, repetitions: 32})
		outcomes, comparisons = append(outcomes, outcome), append(comparisons, comparison)
	}
	if want := []string{OutcomeClean, OutcomeClean, OutcomeDivergence, OutcomeDivergence}; !slices.Equal(outcomes, want) {
		t.Fatalf("outcomes = %v, want %v", outcomes, want)
	}
	if want := []string{ComparisonEstablished, ComparisonMatches, ComparisonDiffers, ComparisonDiffers}; !slices.Equal(comparisons, want) {
		t.Fatalf("comparisons = %v, want %v", comparisons, want)
	}
	if len(ledger.Cohorts) != 1 || ledger.Cohorts[0].Counts != (Counts{Batches: 4, Repetitions: 128, CleanRepetitions: 64, Divergences: 2}) {
		t.Fatalf("cohorts = %+v", ledger.Cohorts)
	}
}

func TestNewToolchainIdentityStartsANewCohort(t *testing.T) {
	ledger := newLedger()
	first, second := key(t, "one"), key(t, "two")
	if first == second {
		t.Fatal("a changed build key kept the execution identity")
	}
	for _, step := range []struct {
		key    CohortKey
		digest record.SHA256
	}{{first, "sha256:a"}, {second, "sha256:b"}, {second, "sha256:b"}} {
		if outcome, _, _ := ledger.observe(observation{run: "1", key: step.key, outcome: OutcomeClean, digest: step.digest, repetitions: 32}); outcome != OutcomeClean {
			t.Fatalf("outcome = %s", outcome)
		}
	}
	if len(ledger.Cohorts) != 2 || ledger.Cohorts[0].Counts.Repetitions != 32 || ledger.Cohorts[1].Counts.Repetitions != 64 {
		t.Fatalf("cohorts = %+v", ledger.Cohorts)
	}
	if ledger.Cohorts[1].Baseline.EvidenceDigest != "sha256:b" {
		t.Fatalf("new cohort baseline = %+v", ledger.Cohorts[1].Baseline)
	}
}

func TestExecutionIdentityBindsCompleteInputsAndRejectsInvalidStrings(t *testing.T) {
	baseline := evidence(11, "one")
	want, err := executionIdentity(baseline)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*record.ExecutionEvidence){
		func(value *record.ExecutionEvidence) { value.Environment[0].Value = "12" },
		func(value *record.ExecutionEvidence) { value.Target.Argv[0] = "changed" },
		func(value *record.ExecutionEvidence) { value.Limits.OutputBytes = 1 },
		func(value *record.ExecutionEvidence) {
			digest := record.HashBytes([]byte("mounts"))
			value.ReadOnlyMountsSHA256 = &digest
		},
		func(value *record.ExecutionEvidence) {
			value.Choices = &record.ChoiceEvidence{Profile: "choice", ImplementationSHA256: record.HashBytes([]byte("choice")), Limit: 1}
		},
		func(value *record.ExecutionEvidence) {
			value.Diagnostics = &record.DiagnosticEvidence{Profile: "diagnostic"}
		},
	} {
		changed := baseline
		changed.Environment = append([]record.Environment(nil), baseline.Environment...)
		changed.Target.Argv = append([]string(nil), baseline.Target.Argv...)
		mutate(&changed)
		got, err := executionIdentity(changed)
		if err != nil {
			t.Fatal(err)
		}
		if got == want {
			t.Fatalf("identity did not bind mutation: %#v", changed)
		}
	}
	invalid := baseline
	invalid.RunnerBuild = "runner\xff"
	if _, err := executionIdentity(invalid); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("executionIdentity() error = %v", err)
	}
}

func TestCumulativeCountsAccumulateAcrossRetainedRuns(t *testing.T) {
	directory := t.TempDir()
	ledger := newLedger()
	cohortKey := key(t, "one")
	ledger.observe(observation{run: "run-1", key: cohortKey, outcome: OutcomeClean, digest: "sha256:a", repetitions: 64})
	ledger.Cohorts[0].Baseline.Evidence = "baselines/x/evidence.json"
	if err := ledger.save(directory); err != nil {
		t.Fatal(err)
	}
	restored, err := LoadLedger(directory)
	if err != nil {
		t.Fatal(err)
	}
	restored.observe(observation{run: "run-2", key: cohortKey, outcome: OutcomeClean, digest: "sha256:a", repetitions: 64})
	restored.observe(observation{run: "run-2", key: cohortKey, outcome: OutcomeOverflow, repetitions: 3})
	cohort := restored.Cohorts[0]
	if cohort.Counts != (Counts{Batches: 3, Repetitions: 131, CleanRepetitions: 128, Overflows: 1}) || !slices.Equal(cohort.Runs, []string{"run-1", "run-2"}) {
		t.Fatalf("cohort = %+v", cohort)
	}
}

func TestLoadLedgerRejectsTamperedCohort(t *testing.T) {
	directory := t.TempDir()
	ledger := newLedger()
	ledger.observe(observation{run: "1", key: key(t, "one"), outcome: OutcomeTargetFailure})
	ledger.Cohorts[0].Key.Workload = "other"
	if err := ledger.save(directory); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLedger(directory); err == nil {
		t.Fatal("a cohort whose key no longer matches its id loaded")
	}
}

func TestLedgerPublicationAndStrictParsing(t *testing.T) {
	directory := t.TempDir()
	ledger := newLedger()
	ledger.observe(observation{run: "1", key: key(t, "one"), outcome: OutcomeTargetFailure})
	if err := ledger.save(directory); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, ledgerFile)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o644 || len(contents) == 0 || contents[len(contents)-1] != '\n' || !bytes.Contains(contents, []byte("\n  \"runs\"")) {
		t.Fatalf("ledger publication = mode %v bytes %q error %v", info.Mode(), contents, err)
	}
	for _, invalid := range [][]byte{
		[]byte("{"),
		bytes.Replace(contents, []byte(`"schema":`), []byte(`"schema":"gomad3.determinism-soak-ledger/v1","schema":`), 1),
		bytes.Replace(contents, []byte(`{`), []byte(`{"unknown":true,`), 1),
		append(append([]byte(nil), contents...), []byte(`{}`)...),
		{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'},
	} {
		if err := os.WriteFile(path, invalid, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadLedger(directory); err == nil {
			t.Fatalf("LoadLedger() accepted %q", invalid)
		}
	}
}

func TestClassifyBatchKeepsOverflowAndInfrastructureApartFromDivergence(t *testing.T) {
	overflowed := evidence(11, "one")
	overflowed.Outcome = record.OutcomeEvidence{Domain: "runner", Reason: "choice_trace_overflow", Termination: "exit"}
	for _, test := range []struct {
		name   string
		report qualification.QualificationReport
		want   string
	}{
		{"qualified", qualification.QualificationReport{Deterministic: true, TargetSuccess: true}, OutcomeClean},
		{"fresh runs disagree", qualification.QualificationReport{TargetSuccess: true, FirstDivergence: "stdout.full_sha256"}, OutcomeDivergence},
		{"overflow before divergence", qualification.QualificationReport{Executions: []qualification.QualificationExecutionReport{{Evidence: &overflowed}}}, OutcomeOverflow},
		{"runner failure", qualification.QualificationReport{Failure: &qualification.QualificationFailure{Classification: "runner_failure", Message: "supervisor exited"}}, OutcomeInfrastructure},
		{"overall timeout", qualification.QualificationReport{Failure: &qualification.QualificationFailure{Classification: "overall_timeout", Message: "deadline"}}, OutcomeInfrastructure},
		{"deterministic target failure", qualification.QualificationReport{Deterministic: true, Evidence: &overflowed}, OutcomeTargetFailure},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got, _ := classifyBatch(test.report); got != test.want {
				t.Fatalf("outcome = %s, want %s", got, test.want)
			}
		})
	}
}

const testSetManifest = `{
  "schema": "gomad3.qualification-set/v3",
  "name": "soak-test",
  "description": "soak test selection",
  "module": "example.test",
  "seeds": [11],
  "repeat": 2,
  "run_timeout": "1m",
  "overall_timeout": "5m",
  "terminate_grace": "2s",
  "output_bytes": 1048576,
  "world_transition_bytes": 1048576,
  "suites": [
    {
      "id": "core-workload",
      "name": "Core workload",
      "tier": 1,
      "capability_mode": "closure",
      "invariant": "the workload is repeatable",
      "package": "./pkg",
      "test": "TestWorkload",
      "choice_bytes": 8388608,
      "replay_successes": true,
      "success_artifact_limit": 1,
      "success_bytes_limit": 134217728,
      "expectation": {"classification": "qualified"}
    }
  ]
}`

func writeSoakManifest(t *testing.T, minimum, batches uint64, informational bool) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "set.json"), []byte(testSetManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		Schema: ManifestSchema, Name: "soak-test", Description: "test soak", Seeds: []uint64{11}, BatchRepeat: 3, MinimumBatches: minimum, Batches: batches,
		BatchTimeout: "10m", Budget: "1h", LoadWorkers: 1, Sizing: "test sizing",
		Selections: []Selection{{Manifest: "set.json", WorkingDir: ".", Suites: []string{"core-workload"}}},
	}
	if informational {
		manifest.InformationalPlatforms = map[string]string{"darwin/arm64": "test finding"}
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "soak.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func argument(args []string, name string) string {
	for _, value := range args {
		if after, ok := strings.CutPrefix(value, name+"="); ok {
			return after
		}
	}
	return ""
}

// fakeQualify writes a real qualification report whose repetitions carry the
// given diagnostic draw counters, the way gomad qualify --json does.
func fakeQualify(t *testing.T, draws func(batch int) []byte) func(context.Context, Command) CommandResult {
	batch := 0
	return func(_ context.Context, command Command) CommandResult {
		batch++
		if !slices.Contains(command.Args, "--diagnostics") || slices.Contains(command.Args, "--replay-successes") || argument(command.Args, "--repeat") != "3" {
			return CommandResult{Err: errors.New("soak batch arguments are wrong: " + strings.Join(command.Args, " "))}
		}
		artifacts := argument(command.Args, "--artifacts")
		seed, err := strconv.ParseUint(argument(command.Args, "--seed"), 10, 64)
		if err != nil {
			return CommandResult{Err: err}
		}
		var executions []qualification.QualificationExecution
		for index, draw := range draws(batch) {
			trace := diagnosticTrace(t, filepath.Join(artifacts, "campaign-"+strconv.Itoa(index), "diagnostics"), draw)
			executionEvidence := evidence(seed, "one")
			executionEvidence.Diagnostics = &trace.DiagnosticEvidence
			executions = append(executions, qualification.QualificationExecution{
				CampaignPath: filepath.Dir(filepath.Dir(trace.Path)), WallElapsedNanos: 1000, Evidence: executionEvidence, Diagnostics: trace,
			})
		}
		report, err := qualification.BuildQualificationReport(qualification.QualificationInput{Command: append([]string{"gomad"}, command.Args...), Executions: executions})
		if err != nil {
			return CommandResult{Err: err}
		}
		path, err := qualification.WriteQualificationReport(artifacts, report)
		if err != nil {
			return CommandResult{Err: err}
		}
		var stdout bytes.Buffer
		if err := qualification.WriteResultEvent(&stdout, report, path); err != nil {
			return CommandResult{Err: err}
		}
		return CommandResult{ExitCode: qualification.ExitStatus(qualification.ClassifyQualification(report)), Stdout: stdout.Bytes()}
	}
}

func runSpec(t *testing.T, manifest string, ledger string, execute func(context.Context, Command) CommandResult) Spec {
	t.Helper()
	root := t.TempDir()
	return Spec{
		ManifestPath: manifest, GomadPath: "gomad", WorkRoot: filepath.Join(root, "work"), LedgerDir: ledger, OutputDir: filepath.Join(root, "output"),
		RunID: "run-" + filepath.Base(root), Platform: "darwin/arm64", Execute: execute,
	}
}

func TestRunRetainsBothTracesAndDifferOutputForACrossBatchDivergence(t *testing.T) {
	manifest := writeSoakManifest(t, 2, 2, false)
	spec := runSpec(t, manifest, t.TempDir(), fakeQualify(t, func(batch int) []byte {
		return bytes.Repeat([]byte{byte(batch - 1)}, 3)
	}))
	report, err := Run(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || report.Verdict != "fail" || ExitStatus(report) != 1 || report.Totals != (Counts{Batches: 2, Repetitions: 6, CleanRepetitions: 3, Divergences: 1}) {
		t.Fatalf("report = %+v", report)
	}
	diverged := report.BatchReports[1]
	if diverged.Outcome != OutcomeDivergence || diverged.Comparison != ComparisonDiffers || diverged.Divergence == nil || diverged.Divergence.Ordinal != 0 {
		t.Fatalf("diverged batch = %+v", diverged)
	}
	for _, name := range []string{"expected.bin", "actual.bin", "expected-evidence.json", "actual-evidence.json", "differ.json", "differ.txt"} {
		if _, err := os.Stat(filepath.Join(spec.OutputDir, diverged.Retained, name)); err != nil {
			t.Fatalf("retained %s: %v", name, err)
		}
	}
	differ, err := os.ReadFile(filepath.Join(spec.OutputDir, diverged.Retained, "differ.txt"))
	if err != nil || !strings.HasPrefix(string(differ), "first-divergent-ordinal=0 fields=") {
		t.Fatalf("differ output = %q, %v", differ, err)
	}
	if _, err := os.Stat(filepath.Join(spec.WorkRoot, "core-workload-seed-11-batch-1")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("batch Campaigns were not pruned: %v", err)
	}
	if len(report.Cohorts) != 1 || report.Cohorts[0].Bound != 0 || report.Cohorts[0].Cumulative.Divergences != 1 {
		t.Fatalf("cohorts = %+v", report.Cohorts)
	}
	if summary, err := os.ReadFile(filepath.Join(spec.OutputDir, "soak-summary.md")); err != nil || !strings.Contains(string(summary), "N = 6 fresh repetitions") {
		t.Fatalf("summary = %q, %v", summary, err)
	}
}

func TestRunGivesBatchesAbsolutePaths(t *testing.T) {
	manifest := writeSoakManifest(t, 1, 1, false)
	t.Chdir(t.TempDir())
	var artifacts, gomad string
	execute := fakeQualify(t, func(int) []byte { return []byte{0, 0, 0} })
	report, err := Run(context.Background(), Spec{
		ManifestPath: manifest, GomadPath: "bin/gomad", WorkRoot: "work", LedgerDir: "ledger", OutputDir: "output", Platform: "darwin/arm64",
		Execute: func(ctx context.Context, command Command) CommandResult {
			artifacts, gomad = argument(command.Args, "--artifacts"), command.Executable
			return execute(ctx, command)
		},
	})
	if err != nil || !report.Passed || !filepath.IsAbs(artifacts) || !filepath.IsAbs(gomad) {
		t.Fatalf("report = %+v, artifacts %q, gomad %q, %v", report, artifacts, gomad, err)
	}
}

func TestRunComparesWithTheBaselineOfAnEarlierRetainedRun(t *testing.T) {
	manifest := writeSoakManifest(t, 1, 1, false)
	ledger := t.TempDir()
	first, err := Run(context.Background(), runSpec(t, manifest, ledger, fakeQualify(t, func(int) []byte { return []byte{0, 0, 0} })))
	if err != nil || !first.Passed || first.Cohorts[0].Bound != 3 {
		t.Fatalf("first run = %+v, %v", first, err)
	}
	second, err := Run(context.Background(), runSpec(t, manifest, ledger, fakeQualify(t, func(int) []byte { return []byte{0, 0, 0} })))
	if err != nil || !second.Passed || second.LedgerRuns != 2 || second.Cohorts[0].Cumulative.Repetitions != 6 || second.Cohorts[0].Runs != 2 || second.Cohorts[0].Bound != 6 {
		t.Fatalf("second run = %+v, %v", second, err)
	}
	third, err := Run(context.Background(), runSpec(t, manifest, ledger, fakeQualify(t, func(int) []byte { return []byte{1, 1, 1} })))
	if err != nil || third.Passed || third.BatchReports[0].Comparison != ComparisonDiffers || third.Cohorts[0].Bound != 0 {
		t.Fatalf("third run = %+v, %v", third, err)
	}
}

func TestRunRetainsTheLocalisedPairOfAWithinBatchDivergence(t *testing.T) {
	spec := runSpec(t, writeSoakManifest(t, 1, 1, false), t.TempDir(), fakeQualify(t, func(int) []byte { return []byte{0, 0, 2} }))
	report, err := Run(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	batch := report.BatchReports[0]
	if batch.Outcome != OutcomeDivergence || batch.Comparison != "" || batch.Divergence == nil || report.Cohorts[0].Bound != 0 {
		t.Fatalf("batch = %+v", batch)
	}
	differ, err := os.ReadFile(filepath.Join(spec.OutputDir, batch.Retained, "differ.json"))
	if err != nil || !bytes.Contains(differ, []byte(`"equal": false`)) {
		t.Fatalf("differ = %s, %v", differ, err)
	}
}

func TestInformationalPlatformReportsADivergenceWithoutFailing(t *testing.T) {
	spec := runSpec(t, writeSoakManifest(t, 1, 1, true), t.TempDir(), fakeQualify(t, func(int) []byte { return []byte{0, 1, 1} }))
	report, err := Run(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if report.Gate != GateInformational || report.InformationalReason != "test finding" || report.Verdict != "informational_divergence" || ExitStatus(report) != 0 {
		t.Fatalf("report = %+v", report)
	}
	if summary, _ := os.ReadFile(filepath.Join(spec.OutputDir, "soak-summary.md")); !strings.Contains(string(summary), "Informational gate") {
		t.Fatalf("summary = %s", summary)
	}
}

func TestInfrastructureFailureIsNotAPassOrADivergence(t *testing.T) {
	failing := func(context.Context, Command) CommandResult {
		var stdout bytes.Buffer
		_ = qualification.WriteErrorEvent(&stdout, "runner_failure", errors.New("toolchain missing"))
		return CommandResult{ExitCode: 3, Stdout: stdout.Bytes()}
	}
	report, err := Run(context.Background(), runSpec(t, writeSoakManifest(t, 1, 1, true), t.TempDir(), failing))
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || report.Verdict != "fail" || ExitStatus(report) != 3 || report.Totals != (Counts{Batches: 1, InfrastructureFailures: 1}) || len(report.Cohorts) != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunStopsAddingRoundsPastTheMinimumWhenTheBudgetIsShort(t *testing.T) {
	spec := runSpec(t, writeSoakManifest(t, 1, 3, false), t.TempDir(), fakeQualify(t, func(int) []byte { return []byte{0, 0, 0} }))
	clock := time.Unix(0, 0)
	spec.Now = func() time.Time {
		clock = clock.Add(10 * time.Minute)
		return clock
	}
	report, err := Run(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.Batches != 1 || report.StopReason != "budget" || report.RepetitionsPerSeed != 3 || len(report.BatchReports) != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestExhaustedBudgetIsReportedAsInfrastructure(t *testing.T) {
	spec := runSpec(t, writeSoakManifest(t, 2, 2, false), t.TempDir(), fakeQualify(t, func(int) []byte { return []byte{0, 0, 0} }))
	clock := time.Unix(0, 0)
	spec.Now = func() time.Time {
		clock = clock.Add(40 * time.Minute)
		return clock
	}
	report, err := Run(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || report.BatchReports[1].Outcome != OutcomeInfrastructure || !strings.Contains(report.BatchReports[1].Message, "budget") {
		t.Fatalf("report = %+v", report)
	}
}

func TestSelectedManifestIsValid(t *testing.T) {
	for _, path := range []string{"../soak.json"} {
		manifest, err := LoadManifest(path)
		if err != nil {
			t.Fatal(err)
		}
		plans, err := resolve(path, manifest, nil)
		if err != nil {
			t.Fatal(err)
		}
		guarded := slices.ContainsFunc(plans, func(plan workloadPlan) bool { return plan.workload.CapabilityMode == "guarded" })
		if !guarded || !slices.Equal(manifest.Seeds, []uint64{11, 17}) {
			t.Fatalf("soak selection must include a guarded-mode workload and seeds 11 and 17: %+v", manifest)
		}
		if _, ok := manifest.InformationalPlatforms["linux/amd64"]; !ok {
			t.Fatal("linux/amd64 must stay informational while fn-105 D12 is open")
		}
	}
}

func TestLoadManifestAcceptsEquivalentJSONAndRejectsInvalidStrictInput(t *testing.T) {
	contents, err := os.ReadFile("../soak.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "soak.json")
	var compact bytes.Buffer
	if err := json.Compact(&compact, contents); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, compact.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(path); err != nil {
		t.Fatalf("LoadManifest() rejected equivalent JSON: %v", err)
	}
	for name, invalid := range map[string][]byte{
		"duplicate": bytes.Replace(contents, []byte(`"schema":`), []byte(`"schema":"gomad3.determinism-soak/v1","schema":`), 1),
		"unknown":   bytes.Replace(contents, []byte(`{`), []byte(`{"unknown":true,`), 1),
		"trailing":  append(append([]byte(nil), contents...), []byte(`{}`)...),
		"utf8":      {'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'},
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, invalid, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadManifest(path); err == nil {
				t.Fatalf("LoadManifest() accepted %q", invalid)
			}
		})
	}
}

func TestSoakManifestIdentityUsesTypedDeterministicEncoding(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"smoke.json", "temporal.json"} {
		contents, err := os.ReadFile(filepath.Join("..", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifestBytes, err := os.ReadFile("../soak.json")
	if err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, manifestBytes); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "soak.json")
	if err := os.WriteFile(manifestPath, compact.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	newSpec := func(suffix string) Spec {
		return Spec{
			ManifestPath: manifestPath, GomadPath: "gomad", WorkRoot: filepath.Join(root, "work-"+suffix),
			LedgerDir: filepath.Join(root, "ledger-"+suffix), OutputDir: filepath.Join(root, "output-"+suffix),
			RunID: "identity-" + suffix, Platform: "darwin/arm64", Execute: func(context.Context, Command) CommandResult { return CommandResult{} },
			Now: func() time.Time { return time.Unix(1, 0) },
		}
	}
	compactRun, err := newRun(newSpec("compact"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	indentedRun, err := newRun(newSpec("indented"))
	if err != nil {
		t.Fatal(err)
	}
	if compactRun.report.ManifestSHA256 != indentedRun.report.ManifestSHA256 {
		t.Fatalf("equivalent soak manifest identities differ: compact %s, indented %s", compactRun.report.ManifestSHA256, indentedRun.report.ManifestSHA256)
	}
}

func TestOverflowRepetitionsDoNotEnterTheBound(t *testing.T) {
	ledger := newLedger()
	cohortKey := key(t, "one")
	ledger.observe(observation{run: "1", key: cohortKey, outcome: OutcomeClean, digest: "sha256:a", repetitions: 32})
	ledger.observe(observation{run: "1", key: cohortKey, outcome: OutcomeOverflow, repetitions: 32})
	if counts := ledger.Cohorts[0].Counts; counts.CleanRepetitions != 32 || counts.Repetitions != 64 {
		t.Fatalf("counts = %+v", counts)
	}
}

func TestFailedBaselineRetentionIsInfrastructureAndLeavesNoBaseline(t *testing.T) {
	ledger := newLedger()
	cohortKey := key(t, "one")
	outcome, comparison, cohort := ledger.observe(observation{run: "1", key: cohortKey, outcome: OutcomeClean, digest: "sha256:a", repetitions: 32,
		retainBaseline: func(*Cohort) error { return errors.New("disk full") }})
	if outcome != OutcomeInfrastructure || comparison != "" || cohort.Baseline != nil || cohort.Counts.CleanRepetitions != 0 || cohort.Counts.InfrastructureFailures != 1 {
		t.Fatalf("outcome %s, cohort %+v", outcome, cohort)
	}
	if outcome, comparison, _ := ledger.observe(observation{run: "1", key: cohortKey, outcome: OutcomeClean, digest: "sha256:b", repetitions: 32}); outcome != OutcomeClean || comparison != ComparisonEstablished {
		t.Fatalf("next batch outcome %s, comparison %s", outcome, comparison)
	}
}
