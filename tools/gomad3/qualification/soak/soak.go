package soak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/qualification"
	"go.temporal.io/server/tools/gomad3/qualification/set"
	"go.temporal.io/server/tools/gomad3/record"
)

const ReportSchema = "gomad3.determinism-soak-report/v1"

// Gates. A strict gate fails on any divergence; an informational gate reports
// divergences without failing because an open finding names the channel.
const (
	GateStrict        = "strict"
	GateInformational = "informational"
)

const maximumCommandOutputBytes = 64 << 20

// Command is one gomad qualify batch invocation.
type Command struct {
	Executable string
	Args       []string
	Dir        string
	Timeout    time.Duration
	Grace      time.Duration
}

type CommandResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Err      error
}

type Spec struct {
	ManifestPath string
	GomadPath    string
	// WorkRoot holds each batch's Campaigns; a batch's directory is removed
	// once its evidence is retained unless KeepBatches is set.
	WorkRoot string
	// LedgerDir holds the cumulative ledger and cohort baselines carried from
	// one scheduled run to the next.
	LedgerDir string
	// OutputDir receives the report, the summary, every batch's qualification
	// report, and the traces and differ output of each divergence.
	OutputDir string
	RunID     string
	// Workloads and Seeds restrict the run to part of the manifest selection.
	Workloads []string
	Seeds     []uint64
	// Batches and Budget override the manifest's maximum rounds and budget
	// when nonzero; Batches also lowers the minimum to at most itself.
	Batches     uint64
	Budget      time.Duration
	KeepBatches bool
	// Platform overrides the host GOOS/GOARCH; tests set it.
	Platform string
	Execute  func(context.Context, Command) CommandResult
	Now      func() time.Time
}

type Load struct {
	Workers  uint64 `json:"workers"`
	HostCPUs uint64 `json:"host_cpus"`
}

type BatchReport struct {
	Workload    string              `json:"workload"`
	Seed        record.Uint64String `json:"seed"`
	Batch       uint64              `json:"batch"`
	Outcome     string              `json:"outcome"`
	Comparison  string              `json:"comparison,omitempty"`
	Qualify     string              `json:"qualify_classification,omitempty"`
	Repetitions uint64              `json:"repetitions"`
	// EvidenceDigest is the batch's evidence baseline, the digest of its
	// first execution.
	EvidenceDigest record.SHA256 `json:"evidence_digest,omitempty"`
	Cohort         string        `json:"cohort,omitempty"`
	Message        string        `json:"message,omitempty"`
	ElapsedNanos   uint64        `json:"elapsed_nanos"`
	// ExecutionWallNanos sums the wall time of the batch's executions, the
	// per-run cost that sizes N.
	ExecutionWallNanos uint64                       `json:"execution_wall_nanos"`
	Report             string                       `json:"report,omitempty"`
	Retained           string                       `json:"retained,omitempty"`
	Divergence         *choice.DiagnosticDivergence `json:"diagnostic_divergence,omitempty"`
}

type CohortReport struct {
	ID        string           `json:"id"`
	Key       CohortKey        `json:"key"`
	Toolchain record.Toolchain `json:"toolchain"`
	// Run counts this run's batches; Cumulative counts every retained run.
	Run        Counts `json:"run"`
	Cumulative Counts `json:"cumulative"`
	Runs       uint64 `json:"runs"`
	// Bound is the quotable result: fresh repetitions with zero divergences,
	// zero when the cohort has diverged.
	Bound uint64 `json:"bound"`
}

type Report struct {
	Schema         string        `json:"schema"`
	Name           string        `json:"name"`
	ManifestSHA256 record.SHA256 `json:"manifest_sha256"`
	RunID          string        `json:"run_id"`
	Platform       string        `json:"platform"`
	Gate           string        `json:"gate"`
	// InformationalReason names the open finding of an informational gate.
	InformationalReason string                `json:"informational_reason,omitempty"`
	Toolchains          []record.Toolchain    `json:"toolchains"`
	Seeds               []record.Uint64String `json:"seeds"`
	Workloads           []string              `json:"workloads"`
	BatchRepeat         uint64                `json:"repeat"`
	MinimumBatches      uint64                `json:"minimum_batches"`
	MaximumBatches      uint64                `json:"maximum_batches"`
	// Batches is the number of rounds this run executed and StopReason why it
	// stopped: "maximum_batches", or "budget" when the previous round's cost
	// no longer fit.
	Batches    uint64 `json:"batches"`
	StopReason string `json:"stop_reason"`
	// RepetitionsPerSeed is N, the fresh repetitions per workload and seed
	// this run scheduled.
	RepetitionsPerSeed uint64         `json:"repetitions_per_seed"`
	Diagnostics        bool           `json:"diagnostics"`
	Choices            bool           `json:"choices"`
	Load               Load           `json:"load"`
	Sizing             string         `json:"sizing"`
	Started            string         `json:"started"`
	ElapsedNanos       uint64         `json:"elapsed_nanos"`
	Totals             Counts         `json:"totals"`
	LedgerRuns         uint64         `json:"ledger_runs"`
	Passed             bool           `json:"passed"`
	Verdict            string         `json:"verdict"`
	BatchReports       []BatchReport  `json:"results"`
	Cohorts            []CohortReport `json:"cohorts"`
}

// ExitStatus maps a report to gomadtool soak's status: 0 for a pass or an
// informational divergence, 1 for a divergence or target failure, and 3 when
// an overflow or infrastructure failure left the soak unable to conclude.
func ExitStatus(report Report) int {
	switch {
	case report.Passed:
		return 0
	case report.Totals.Divergences != 0 && report.Gate == GateStrict, report.Totals.TargetFailures != 0:
		return 1
	case report.Totals.Overflows != 0 || report.Totals.InfrastructureFailures != 0:
		return 3
	default:
		return 0
	}
}

// soakRun is one run's resolved configuration and accumulating state.
type soakRun struct {
	spec           Spec
	manifest       Manifest
	plans          []workloadPlan
	seeds          []uint64
	minimumBatches uint64
	maximumBatches uint64
	batchTimeout   time.Duration
	deadline       time.Time
	started        time.Time
	ledger         *Ledger
	report         Report
	runCounts      map[string]*Counts
}

func Run(ctx context.Context, spec Spec) (Report, error) {
	run, err := newRun(spec)
	if err != nil {
		return Report{}, err
	}
	stopLoad, err := startLoad(int(run.manifest.LoadWorkers))
	if err != nil {
		return Report{}, err
	}
	run.executeRounds(ctx)
	stopLoad()
	return run.publish()
}

// normalizeSpec fills defaults and makes every path absolute: each qualify
// batch executes in its workload's working directory.
func normalizeSpec(spec Spec) (Spec, error) {
	if spec.ManifestPath == "" || spec.GomadPath == "" || spec.WorkRoot == "" || spec.LedgerDir == "" || spec.OutputDir == "" {
		return Spec{}, errors.New("soak needs a manifest, a gomad executable, and work, ledger, and output directories")
	}
	paths := []*string{&spec.ManifestPath, &spec.WorkRoot, &spec.LedgerDir, &spec.OutputDir}
	if strings.ContainsRune(spec.GomadPath, filepath.Separator) {
		paths = append(paths, &spec.GomadPath)
	}
	for _, path := range paths {
		absolute, err := filepath.Abs(*path)
		if err != nil {
			return Spec{}, err
		}
		*path = absolute
	}
	if spec.Execute == nil {
		spec.Execute = executeCommand
	}
	if spec.Now == nil {
		spec.Now = time.Now
	}
	if spec.Platform == "" {
		spec.Platform = runtime.GOOS + "/" + runtime.GOARCH
	}
	return spec, nil
}

func newRun(spec Spec) (*soakRun, error) {
	spec, err := normalizeSpec(spec)
	if err != nil {
		return nil, err
	}
	manifest, err := LoadManifest(spec.ManifestPath)
	if err != nil {
		return nil, err
	}
	manifestBytes, err := os.ReadFile(spec.ManifestPath)
	if err != nil {
		return nil, err
	}
	plans, err := resolve(spec.ManifestPath, manifest, spec.Workloads)
	if err != nil {
		return nil, err
	}
	run := &soakRun{spec: spec, manifest: manifest, plans: plans, seeds: manifest.Seeds, runCounts: map[string]*Counts{}}
	for _, seed := range spec.Seeds {
		if !slices.Contains(manifest.Seeds, seed) {
			return nil, fmt.Errorf("soak manifest does not select seed %d", seed)
		}
	}
	if len(spec.Seeds) != 0 {
		run.seeds = spec.Seeds
	}
	run.maximumBatches, run.minimumBatches = manifest.Batches, manifest.MinimumBatches
	if spec.Batches != 0 {
		run.maximumBatches, run.minimumBatches = spec.Batches, min(manifest.MinimumBatches, spec.Batches)
	}
	budget, _ := time.ParseDuration(manifest.Budget)
	if spec.Budget != 0 {
		budget = spec.Budget
	}
	run.batchTimeout, _ = time.ParseDuration(manifest.BatchTimeout)
	if run.ledger, err = LoadLedger(spec.LedgerDir); err != nil {
		return nil, err
	}
	for _, directory := range []string{spec.WorkRoot, spec.LedgerDir, spec.OutputDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return nil, err
		}
	}
	run.started = spec.Now()
	run.deadline = run.started.Add(budget)
	if run.spec.RunID == "" {
		run.spec.RunID = "local-" + run.started.UTC().Format("20060102T150405Z")
	}
	run.report = Report{
		Schema: ReportSchema, Name: manifest.Name, ManifestSHA256: record.HashBytes(manifestBytes), RunID: run.spec.RunID, Platform: spec.Platform,
		Gate: GateStrict, Toolchains: []record.Toolchain{}, BatchRepeat: manifest.BatchRepeat,
		MinimumBatches: run.minimumBatches, MaximumBatches: run.maximumBatches, StopReason: "maximum_batches", Diagnostics: true, Choices: true,
		Load: Load{Workers: manifest.LoadWorkers, HostCPUs: uint64(runtime.NumCPU())}, Sizing: manifest.Sizing,
		Started: run.started.UTC().Format(time.RFC3339), BatchReports: []BatchReport{}, Cohorts: []CohortReport{},
	}
	if reason, ok := manifest.InformationalPlatforms[spec.Platform]; ok {
		run.report.Gate, run.report.InformationalReason = GateInformational, reason
	}
	for _, seed := range run.seeds {
		run.report.Seeds = append(run.report.Seeds, record.Uint64String(seed))
	}
	for _, plan := range plans {
		run.report.Workloads = append(run.report.Workloads, plan.workload.ID)
	}
	run.ledger.Runs = append(run.ledger.Runs, LedgerRun{ID: run.spec.RunID, Platform: spec.Platform, ManifestSHA256: run.report.ManifestSHA256, Started: run.report.Started})
	return run, nil
}

// executeRounds runs rounds of one batch per workload and seed. A round past
// the minimum starts only when the previous round's measured cost, with a
// fifth for variance, fits the rest of the budget.
func (run *soakRun) executeRounds(ctx context.Context) {
	var previousRound time.Duration
	for batch := uint64(1); batch <= run.maximumBatches; batch++ {
		if batch > run.minimumBatches && previousRound+previousRound/5 > run.deadline.Sub(run.spec.Now()) {
			run.report.StopReason = "budget"
			break
		}
		roundStarted := run.spec.Now()
		run.report.Batches = batch
		for _, plan := range run.plans {
			for _, seed := range run.seeds {
				run.record(runBatch(ctx, run.spec, run.manifest, plan, seed, batch, run.batchTimeout, run.deadline, run.ledger, &run.report))
			}
		}
		previousRound = run.spec.Now().Sub(roundStarted)
	}
	run.report.RepetitionsPerSeed = run.report.Batches * run.manifest.BatchRepeat
}

func (run *soakRun) record(batch BatchReport) {
	run.report.BatchReports = append(run.report.BatchReports, batch)
	run.report.Totals.add(batch.Outcome, batch.Repetitions)
	if batch.Cohort == "" {
		return
	}
	if run.runCounts[batch.Cohort] == nil {
		run.runCounts[batch.Cohort] = &Counts{}
	}
	run.runCounts[batch.Cohort].add(batch.Outcome, batch.Repetitions)
}

// publish reports every cohort this run touched with its cumulative counts and
// bound, saves the ledger, and writes the report and summary.
func (run *soakRun) publish() (Report, error) {
	report := &run.report
	for _, cohort := range run.ledger.Cohorts {
		counts, ok := run.runCounts[cohort.ID]
		if !ok {
			continue
		}
		bound := cohort.Counts.Repetitions
		if cohort.Counts.Divergences != 0 {
			bound = 0
		}
		report.Cohorts = append(report.Cohorts, CohortReport{
			ID: cohort.ID, Key: cohort.Key, Toolchain: cohort.Toolchain, Run: *counts, Cumulative: cohort.Counts, Runs: uint64(len(cohort.Runs)), Bound: bound,
		})
	}
	report.LedgerRuns = uint64(len(run.ledger.Runs))
	report.ElapsedNanos = uint64(run.spec.Now().Sub(run.started))
	inconclusive := report.Totals.Overflows != 0 || report.Totals.TargetFailures != 0 || report.Totals.InfrastructureFailures != 0
	report.Passed = report.Totals.Divergences == 0 && !inconclusive
	switch {
	case report.Passed:
		report.Verdict = "pass"
	case !inconclusive && report.Gate == GateInformational:
		report.Verdict = "informational_divergence"
	default:
		report.Verdict = "fail"
	}
	if err := run.ledger.save(run.spec.LedgerDir); err != nil {
		return *report, fmt.Errorf("save soak ledger: %w", err)
	}
	if err := writeJSON(filepath.Join(run.spec.OutputDir, "soak-report.json"), *report); err != nil {
		return *report, err
	}
	return *report, os.WriteFile(filepath.Join(run.spec.OutputDir, "soak-summary.md"), []byte(Summary(*report)), 0o644)
}

func runBatch(ctx context.Context, spec Spec, manifest Manifest, plan workloadPlan, seed, batch uint64, batchTimeout time.Duration, deadline time.Time, ledger *Ledger, report *Report) BatchReport {
	name := fmt.Sprintf("%s-seed-%d-batch-%d", plan.workload.ID, seed, batch)
	result := BatchReport{Workload: plan.workload.ID, Seed: record.Uint64String(seed), Batch: batch}
	grace, _ := time.ParseDuration(plan.setManifest.TerminateGrace)
	remaining := deadline.Sub(spec.Now())
	if remaining <= grace+time.Minute {
		result.Outcome, result.Message = OutcomeInfrastructure, "soak budget exhausted before the batch could start"
		return result
	}
	overall := min(batchTimeout, remaining-grace-30*time.Second)
	artifacts := filepath.Join(spec.WorkRoot, name)
	if err := os.RemoveAll(artifacts); err != nil {
		result.Outcome, result.Message = OutcomeInfrastructure, err.Error()
		return result
	}
	if !spec.KeepBatches {
		defer func() { _ = os.RemoveAll(artifacts) }()
	}
	started := spec.Now()
	executed := spec.Execute(ctx, Command{
		Executable: spec.GomadPath, Args: set.SoakQualifyArguments(plan.setManifest, plan.workload, seed, manifest.BatchRepeat, overall, artifacts),
		Dir: plan.workingDir, Timeout: overall + grace + 30*time.Second, Grace: grace,
	})
	result.ElapsedNanos = uint64(spec.Now().Sub(started))
	qualified, err := openBatchReport(artifacts, executed)
	if err != nil {
		result.Outcome, result.Message = OutcomeInfrastructure, err.Error()
		return result
	}
	reportPath := filepath.Join("batches", name+".json")
	if err := writeJSON(filepath.Join(spec.OutputDir, reportPath), qualified); err != nil {
		result.Outcome, result.Message = OutcomeInfrastructure, err.Error()
		return result
	}
	result.Report = reportPath
	result.Qualify = qualification.ClassifyQualification(qualified)
	result.EvidenceDigest = qualified.EvidenceDigest
	result.Divergence = diagnosticDivergence(qualified)
	for _, run := range qualified.Executions {
		result.ExecutionWallNanos += uint64(run.WallElapsedNanos)
	}
	result.Repetitions = uint64(len(qualified.Executions))
	result.Outcome, result.Message = classifyBatch(qualified)
	if qualified.Evidence == nil {
		return result
	}
	platform := qualified.Evidence.Toolchain.TargetGOOS + "/" + qualified.Evidence.Toolchain.TargetGOARCH
	identity, err := executionIdentity(*qualified.Evidence)
	if err != nil || platform != spec.Platform {
		result.Outcome, result.Message = OutcomeInfrastructure, errors.Join(fmt.Errorf("batch platform %s, soak platform %s", platform, spec.Platform), err).Error()
		return result
	}
	if !slices.Contains(report.Toolchains, qualified.Evidence.Toolchain) {
		report.Toolchains = append(report.Toolchains, qualified.Evidence.Toolchain)
	}
	outcome, comparison, cohort := ledger.observe(observation{
		run: spec.RunID, batch: batch, toolchain: qualified.Evidence.Toolchain, outcome: result.Outcome, digest: qualified.EvidenceDigest, repetitions: result.Repetitions,
		key: CohortKey{Workload: plan.workload.ID, Seed: record.Uint64String(seed), Platform: platform, ExecutionIdentity: identity},
	})
	result.Outcome, result.Comparison, result.Cohort = outcome, comparison, cohort.ID
	switch {
	case comparison == ComparisonEstablished:
		// The digest alone keeps later comparisons working; a missing file
		// only leaves a later divergence without the baseline's copy.
		if err := retainBaseline(spec.LedgerDir, cohort, qualified); err != nil {
			result.Message = fmt.Sprintf("cohort baseline files were not retained: %v", err)
		}
	case comparison == ComparisonDiffers:
		result.Message = fmt.Sprintf("evidence %s differs from cohort baseline %s (run %s batch %d)", qualified.EvidenceDigest, cohort.Baseline.EvidenceDigest, cohort.Baseline.Run, cohort.Baseline.Batch)
		retained, divergence, err := retainBaselineDivergence(spec, name, cohort, qualified)
		result.Retained, result.Divergence = retained, divergence
		if err != nil {
			result.Message += "; " + err.Error()
		}
	case result.Outcome == OutcomeDivergence:
		retained, err := retainBatchDivergence(spec, name, qualified)
		result.Retained = retained
		if err != nil {
			result.Message += "; " + err.Error()
		}
	default:
		// A baseline match and a non-divergent failure retain nothing more.
	}
	return result
}

// openBatchReport reads the qualification report a batch retained. A batch
// without a retained report is an infrastructure failure.
func openBatchReport(artifacts string, executed CommandResult) (qualification.QualificationReport, error) {
	if executed.Err != nil {
		return qualification.QualificationReport{}, fmt.Errorf("gomad qualify: %w%s", executed.Err, stderrSuffix(executed.Stderr))
	}
	event, err := qualification.DecodeResultEvent(executed.Stdout)
	if err != nil {
		return qualification.QualificationReport{}, fmt.Errorf("gomad qualify exited %d: %w%s", executed.ExitCode, err, stderrSuffix(executed.Stderr))
	}
	path, err := filepath.Abs(event.ReportPath)
	if err != nil {
		return qualification.QualificationReport{}, err
	}
	if relative, err := filepath.Rel(artifacts, path); err != nil || !filepath.IsLocal(relative) {
		return qualification.QualificationReport{}, errors.New("qualification report is outside the batch artifact root")
	}
	report, err := qualification.OpenQualificationReport(path)
	if err != nil {
		return qualification.QualificationReport{}, err
	}
	classification := qualification.ClassifyQualification(report)
	if event.Classification != classification || executed.ExitCode != qualification.ExitStatus(classification) {
		return qualification.QualificationReport{}, fmt.Errorf("qualification result classification or status is inconsistent: %s/%d", event.Classification, executed.ExitCode)
	}
	return report, nil
}

func stderrSuffix(stderr []byte) string {
	text := strings.TrimSpace(string(stderr))
	if text == "" {
		return ""
	}
	if len(text) > 2048 {
		text = text[len(text)-2048:]
	}
	return ": " + text
}

// classifyBatch places a batch's own result before any cross-batch comparison.
// An overflow comes first because an overflowed trace cannot support a
// localisation or a determinism claim; a runner failure is infrastructure.
func classifyBatch(report qualification.QualificationReport) (outcome, message string) {
	for index, run := range report.Executions {
		if evidence := run.Evidence; evidence != nil && (strings.HasSuffix(evidence.Outcome.Reason, "_overflow") || evidence.Choices != nil && evidence.Choices.TerminalState == "overflow") {
			return OutcomeOverflow, fmt.Sprintf("execution %d trace overflow: %s", index+1, evidence.Outcome.Reason)
		}
	}
	if report.Failure != nil {
		if strings.Contains(report.Failure.Message, "overflow") {
			return OutcomeOverflow, report.Failure.Classification + ": " + report.Failure.Message
		}
		return OutcomeInfrastructure, report.Failure.Classification + ": " + report.Failure.Message
	}
	switch classification := qualification.ClassifyQualification(report); classification {
	case "qualified":
		return OutcomeClean, ""
	case "nondeterministic":
		return OutcomeDivergence, "fresh repetitions disagree at " + report.FirstDivergence
	case "replay_divergence":
		return OutcomeDivergence, "a retained execution diverged on replay"
	case "target_failure":
		if report.Evidence == nil {
			return OutcomeTargetFailure, "the target failed"
		}
		return OutcomeTargetFailure, "the target failed: " + report.Evidence.Outcome.Reason
	default:
		return OutcomeInfrastructure, classification
	}
}

func diagnosticDivergence(report qualification.QualificationReport) *choice.DiagnosticDivergence {
	if report.DiagnosticDivergence == nil {
		return nil
	}
	divergence := report.DiagnosticDivergence.DiagnosticDivergence
	return &divergence
}

func retainBaseline(ledgerDir string, cohort *Cohort, report qualification.QualificationReport) error {
	directory := filepath.Join("baselines", cohort.ID)
	cohort.Baseline.Evidence = filepath.Join(directory, "evidence.json")
	if err := os.MkdirAll(filepath.Join(ledgerDir, directory), 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(ledgerDir, cohort.Baseline.Evidence), report.Evidence); err != nil {
		return err
	}
	if trace := report.Executions[0].Diagnostics; trace != nil {
		cohort.Baseline.Trace = filepath.Join(directory, "trace.bin")
		return copyFile(trace.Path, filepath.Join(ledgerDir, cohort.Baseline.Trace))
	}
	return nil
}

// retainBaselineDivergence keeps the cohort baseline's evidence and trace, the
// batch's first evidence and trace, and the differ output between the traces.
func retainBaselineDivergence(spec Spec, name string, cohort *Cohort, report qualification.QualificationReport) (string, *choice.DiagnosticDivergence, error) {
	retained := filepath.Join("divergences", name)
	directory := filepath.Join(spec.OutputDir, retained)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", nil, err
	}
	expectedTrace, actualTrace := "", ""
	errs := []error{copyFile(filepath.Join(spec.LedgerDir, cohort.Baseline.Evidence), filepath.Join(directory, "expected-evidence.json"))}
	errs = append(errs, writeJSON(filepath.Join(directory, "actual-evidence.json"), report.Evidence))
	if cohort.Baseline.Trace != "" {
		expectedTrace = filepath.Join(directory, "expected.bin")
		errs = append(errs, copyFile(filepath.Join(spec.LedgerDir, cohort.Baseline.Trace), expectedTrace))
	}
	if trace := report.Executions[0].Diagnostics; trace != nil {
		actualTrace = filepath.Join(directory, "actual.bin")
		errs = append(errs, copyFile(trace.Path, actualTrace))
	}
	divergence, err := writeDiffer(directory, expectedTrace, actualTrace)
	return retained, divergence, errors.Join(append(errs, err)...)
}

// retainBatchDivergence keeps the two traces of the pair the qualification
// report localised, or of the first repetition whose evidence differs from the
// first execution, and the differ output between them.
func retainBatchDivergence(spec Spec, name string, report qualification.QualificationReport) (string, error) {
	baseline, other := 0, -1
	if report.DiagnosticDivergence != nil {
		baseline, other = int(report.DiagnosticDivergence.BaselineIteration)-1, int(report.DiagnosticDivergence.Iteration)-1
	} else {
		for index, run := range report.Executions {
			if run.EvidenceDigest != report.Executions[0].EvidenceDigest {
				other = index
				break
			}
		}
	}
	retained := filepath.Join("divergences", name)
	directory := filepath.Join(spec.OutputDir, retained)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	if other < 0 || other >= len(report.Executions) || baseline < 0 || baseline >= len(report.Executions) {
		return retained, os.WriteFile(filepath.Join(directory, "differ.txt"), []byte("no fresh execution pair to compare\n"), 0o644)
	}
	var errs []error
	paths := [2]string{}
	for slot, index := range [2]int{baseline, other} {
		label := [2]string{"expected", "actual"}[slot]
		errs = append(errs, writeJSON(filepath.Join(directory, label+"-evidence.json"), report.Executions[index].Evidence))
		if trace := report.Executions[index].Diagnostics; trace != nil {
			paths[slot] = filepath.Join(directory, label+".bin")
			errs = append(errs, copyFile(trace.Path, paths[slot]))
		}
	}
	_, err := writeDiffer(directory, paths[0], paths[1])
	return retained, errors.Join(append(errs, err)...)
}

// writeDiffer writes gomadtool diagnostic-diff's JSON and text output for two
// retained traces.
func writeDiffer(directory, expected, actual string) (*choice.DiagnosticDivergence, error) {
	if expected == "" || actual == "" {
		return nil, os.WriteFile(filepath.Join(directory, "differ.txt"), []byte("a diagnostic trace is unavailable for this pair\n"), 0o644)
	}
	difference, err := diffTraces(expected, actual)
	if err != nil {
		return nil, errors.Join(err, os.WriteFile(filepath.Join(directory, "differ.txt"), []byte(err.Error()+"\n"), 0o644))
	}
	text := "diagnostic traces match\n"
	if difference != nil {
		text = fmt.Sprintf("first-divergent-ordinal=%d fields=%s\n", difference.Ordinal, strings.Join(difference.Fields, ","))
	}
	output := struct {
		Equal      bool                         `json:"equal"`
		Divergence *choice.DiagnosticDivergence `json:"divergence,omitempty"`
	}{Equal: difference == nil, Divergence: difference}
	return difference, errors.Join(writeJSON(filepath.Join(directory, "differ.json"), output), os.WriteFile(filepath.Join(directory, "differ.txt"), []byte(text), 0o644))
}

func diffTraces(expectedPath, actualPath string) (*choice.DiagnosticDivergence, error) {
	expected, err := choice.ReadDiagnosticTrace(expectedPath)
	if err != nil {
		return nil, err
	}
	actual, err := choice.ReadDiagnosticTrace(actualPath)
	if err != nil {
		return nil, err
	}
	return choice.DiffDiagnostics(expected.Bytes, actual.Bytes)
}

func executeCommand(ctx context.Context, command Command) CommandResult {
	executed, err := hostexec.Run(ctx, hostexec.Request{
		Command: append([]string{command.Executable}, command.Args...), Dir: command.Dir, Env: os.Environ(),
		Timeout: command.Timeout, TerminateGrace: command.Grace, OutputLimit: maximumCommandOutputBytes,
	})
	result := CommandResult{ExitCode: executed.ExitCode, Stdout: executed.Stdout.Bytes, Stderr: executed.Stderr.Bytes, Err: err}
	if executed.WatchdogTimeout {
		result.Err = errors.Join(result.Err, context.DeadlineExceeded)
	}
	if executed.Cancelled {
		result.Err = errors.Join(result.Err, context.Canceled)
	}
	if executed.Stdout.Truncated || executed.Stderr.Truncated {
		result.Err = errors.Join(result.Err, errors.New("qualification command output exceeded its bound"))
	}
	return result
}

// startLoad runs count busy host threads, unrelated to the workload, until the
// returned function stops them.
func startLoad(count int) (func(), error) {
	var stopped atomic.Bool
	started := make(chan struct{}, count)
	var workers sync.WaitGroup
	workers.Add(count)
	for range count {
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			defer workers.Done()
			started <- struct{}{}
			for !stopped.Load() {
			}
		}()
	}
	var once sync.Once
	stopAll := func() {
		once.Do(func() {
			stopped.Store(true)
			workers.Wait()
		})
	}
	for range count {
		select {
		case <-started:
		case <-time.After(time.Second):
			stopAll()
			return nil, errors.New("soak load worker failed to start")
		}
	}
	return stopAll, nil
}

func copyFile(source, destination string) error {
	contents, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, contents, 0o644)
}

func writeJSON(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(encoded, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

// Summary renders the report as the Markdown a workflow step summary shows.
func Summary(report Report) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "## Determinism soak %s on %s: %s\n\n", report.Name, report.Platform, report.Verdict)
	if report.Gate == GateInformational {
		fmt.Fprintf(&builder, "Informational gate: divergences do not fail this platform while %s.\n\n", report.InformationalReason)
	}
	seeds := make([]string, len(report.Seeds))
	for index, seed := range report.Seeds {
		seeds[index] = string(strconv.AppendUint(nil, uint64(seed), 10))
	}
	fmt.Fprintf(&builder, "- N = %d fresh repetitions per workload and seed (%d batches of %d; minimum %d, maximum %d, stopped at %s), seeds %s, choice tracing and diagnostics on\n",
		report.RepetitionsPerSeed, report.Batches, report.BatchRepeat, report.MinimumBatches, report.MaximumBatches, report.StopReason, strings.Join(seeds, ", "))
	fmt.Fprintf(&builder, "- Load: %d busy host threads on %d CPUs\n", report.Load.Workers, report.Load.HostCPUs)
	for _, toolchain := range report.Toolchains {
		fmt.Fprintf(&builder, "- Toolchain: %s, build key %s\n", toolchain.GoVersion, toolchain.BuildKey)
	}
	totals := report.Totals
	fmt.Fprintf(&builder, "- This run: %d batches, %d repetitions, %d divergences, %d overflows, %d target failures, %d infrastructure failures\n",
		totals.Batches, totals.Repetitions, totals.Divergences, totals.Overflows, totals.TargetFailures, totals.InfrastructureFailures)
	fmt.Fprintf(&builder, "- Ledger: %d retained runs\n\n", report.LedgerRuns)
	builder.WriteString("| Cohort | Workload | Seed | Run repetitions | Cumulative repetitions | Cumulative divergences | Runs | Bound |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, cohort := range report.Cohorts {
		fmt.Fprintf(&builder, "| %s | %s | %d | %d | %d | %d | %d | %d |\n", cohort.ID, cohort.Key.Workload, uint64(cohort.Key.Seed),
			cohort.Run.Repetitions, cohort.Cumulative.Repetitions, cohort.Cumulative.Divergences, cohort.Runs, cohort.Bound)
	}
	for _, batch := range report.BatchReports {
		if batch.Outcome != OutcomeClean {
			fmt.Fprintf(&builder, "\n- %s seed %d batch %d: %s.", batch.Workload, uint64(batch.Seed), batch.Batch, batch.Outcome)
			if batch.Message != "" {
				fmt.Fprintf(&builder, " %s.", strings.TrimSuffix(batch.Message, "."))
			}
			if batch.Retained != "" {
				fmt.Fprintf(&builder, " Retained: %s.", batch.Retained)
			}
		}
	}
	builder.WriteString("\n")
	return builder.String()
}
