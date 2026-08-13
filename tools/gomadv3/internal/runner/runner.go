package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	executionoutcome "go.temporal.io/server/tools/gomadv3/internal/outcome"
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/internal/target"
	"go.temporal.io/server/tools/gomadv3/internal/worldrecord"
	"go.temporal.io/server/tools/gomadv3/world"
)

type FailurePolicy string

type CoverageMode string

type KeepSuccesses string

const (
	PolicyFirst  FailurePolicy = "first"
	PolicyBudget FailurePolicy = "budget"
	PolicyAll    FailurePolicy = "all"
)

const (
	KeepSuccessesNone  KeepSuccesses = "none"
	KeepSuccessesNovel KeepSuccesses = "novel"
	KeepSuccessesAll   KeepSuccesses = "all"
)

const (
	CoverageNone     CoverageMode = "none"
	CoverageSemantic CoverageMode = "semantic"
)

type StopReason string

const (
	StopSeedsExhausted StopReason = "seeds_exhausted"
	StopFirstFailure   StopReason = "first_failure"
	StopFailureBudget  StopReason = "failure_budget"
)

type ProgressPhase string

const (
	ProgressPreparing ProgressPhase = "preparing"
	ProgressRunning   ProgressPhase = "running"
	ProgressComplete  ProgressPhase = "complete"
)

type Progress struct {
	Phase                ProgressPhase
	BatchPath            string
	Selected             uint64
	Attempted            uint64
	Running              uint64
	Succeeded            uint64
	Failures             uint64
	Watchdogs            uint64
	ReplayDivergences    uint64
	Cancelled            uint64
	DistinctFailures     uint64
	Artifacts            []string
	RetainedSuccesses    uint64
	RetainedSuccessBytes uint64
	SuccessArtifacts     []string
}

type ProgressFunc func(Progress) error

type Preparer interface {
	Prepare(context.Context, target.Spec) (target.Prepared, error)
}

type Executor interface {
	Run(context.Context, process.Request) (process.Result, error)
}

type Config struct {
	ResumeBatch            string
	Seeds                  string
	Parallel               int
	RunTimeout             time.Duration
	OverallTimeout         time.Duration
	TerminateGrace         time.Duration
	OnFailure              FailurePolicy
	FailureBudget          uint64
	OutputLimit            uint64
	WorldTransitionLimit   uint64
	Artifacts              string
	Environment            []string
	IOROMounts             []string
	IOROMountLimits        romount.Limits
	Target                 target.Spec
	SupervisorCommand      []string
	CoordinatorCommand     []string
	RunnerBuild            string
	Coverage               CoverageMode
	RequiredSemanticProbes []string
	CollectRunEvidence     bool
	KeepSuccesses          KeepSuccesses
	SuccessArtifactLimit   uint64
	SuccessBytesLimit      uint64
	Progress               ProgressFunc
	ProgressInterval       time.Duration
	Preparer               Preparer
	Executor               Executor
}

type Summary struct {
	BatchPath            string
	SelectionCount       uint64
	Attempted            uint64
	Succeeded            uint64
	Failures             uint64
	Watchdogs            uint64
	ReplayDivergences    uint64
	Cancelled            uint64
	DistinctFailures     uint64
	StopReason           StopReason
	Artifacts            []string
	RetainedSuccesses    uint64
	RetainedSuccessBytes uint64
	SuccessArtifacts     []string
	SemanticCoverage     *ioprofile.SemanticCoverage
	RunEvidence          *RunEvidence
}

type HostError struct {
	Reason string
	Err    error
}

func (err *HostError) Error() string {
	if err.Err == nil {
		return "gomadv3 Runner/host failure: " + err.Reason
	}
	return "gomadv3 Runner/host failure: " + err.Reason + ": " + err.Err.Error()
}

func (err *HostError) Unwrap() error {
	return err.Err
}

type targetPreparer struct{}

func (targetPreparer) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return target.Prepare(ctx, spec)
}

type processExecutor struct{}

func (processExecutor) Run(ctx context.Context, request process.Request) (process.Result, error) {
	return process.Run(ctx, request)
}

type runJob struct {
	ordinal uint64
	seed    uint64
}

type runCompletion struct {
	job        runJob
	startedAt  time.Time
	finishedAt time.Time
	result     process.Result
	err        error
	journal    *artifact.RunJournal
}

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func Run(ctx context.Context, config Config) (Summary, error) {
	if config.ResumeBatch != "" {
		var err error
		config, err = resumeRequestDefaults(config)
		if err != nil {
			return Summary{}, err
		}
	}
	if len(config.CoordinatorCommand) != 0 {
		if config.Preparer != nil || config.Executor != nil {
			return Summary{}, fmt.Errorf("isolated Runner does not accept injected preparation or execution")
		}
		return runIsolated(ctx, config)
	}
	return runLocal(ctx, config)
}

func runLocal(ctx context.Context, config Config) (summary Summary, retErr error) {
	resuming := config.ResumeBatch != ""
	selection, baseEnvironment, err := validateConfig(config)
	var readOnlyMounts []romount.Mapping
	var prepared target.Prepared
	var resumePlan artifact.BatchPlan
	if err != nil {
		return Summary{}, err
	}
	if resuming {
		resumePlan, err = artifact.ReadResumePlan(config.ResumeBatch)
		if err == nil {
			config, selection, baseEnvironment, readOnlyMounts, prepared, err = resumeConfiguration(config, resumePlan)
		}
		if err != nil {
			return Summary{}, err
		}
	} else {
		config.Artifacts, err = filepath.Abs(config.Artifacts)
		if err != nil {
			return Summary{}, &HostError{Reason: "artifact_setup", Err: fmt.Errorf("resolve artifact root: %w", err)}
		}
		readOnlyMounts, err = romount.ParseMappings(config.IOROMounts, config.Target.WorkingDir)
		if err != nil {
			return Summary{}, err
		}
		if config.IOROMountLimits == (romount.Limits{}) {
			config.IOROMountLimits = romount.DefaultLimits()
		}
	}
	overallCtx, overallCancel := context.WithTimeout(ctx, config.OverallTimeout)
	defer overallCancel()
	var runID string
	var batchPath string
	var journal *artifact.BatchJournal
	var resumedRuns []artifact.RunRecord
	if resuming {
		batchPath = config.ResumeBatch
		runID = filepath.Base(batchPath)
		var resumeState artifact.ResumeState
		journal, resumeState, err = artifact.ResumeBatchJournal(overallCtx, batchPath)
		if err == nil {
			var equal bool
			equal, err = equalBatchPlans(resumePlan, resumeState.Plan)
			if !equal && err == nil {
				err = fmt.Errorf("batch plan changed while acquiring its resume lock")
			}
			resumedRuns = resumeState.Runs
		}
		if err != nil {
			return Summary{BatchPath: batchPath, SelectionCount: selection.Count()}, &HostError{Reason: "resume_setup", Err: err}
		}
		summary, _, _, _, err = restoreResumeSummary(batchPath, selection, resumedRuns)
		if err != nil {
			return summary, &HostError{Reason: "resume_setup", Err: err}
		}
	} else {
		runID, err = newRunID()
		if err != nil {
			return Summary{}, &HostError{Reason: "run_id", Err: err}
		}
		batchPath = filepath.Join(config.Artifacts, "v1", runID)
		summary = Summary{BatchPath: batchPath, SelectionCount: selection.Count()}
		journal, err = artifact.NewBatchJournal(overallCtx, artifact.BatchConfig{
			Root: config.Artifacts, RunID: runID, Selection: config.Seeds, SelectionCount: selection.Count(),
		})
		if err != nil {
			return summary, &HostError{Reason: "artifact_setup", Err: err}
		}
	}
	defer func() {
		if closeErr := journal.Close(); closeErr != nil {
			retErr = errors.Join(retErr, &HostError{Reason: "runs_close", Err: closeErr})
		}
	}()
	reportProgress := func(phase ProgressPhase, active int) error {
		if config.Progress == nil {
			return nil
		}
		return config.Progress(Progress{
			Phase: phase, BatchPath: summary.BatchPath, Selected: summary.SelectionCount, Attempted: summary.Attempted, Running: uint64(active),
			Succeeded: summary.Succeeded, Failures: summary.Failures, Watchdogs: summary.Watchdogs, ReplayDivergences: summary.ReplayDivergences, Cancelled: summary.Cancelled,
			DistinctFailures: summary.DistinctFailures, Artifacts: append([]string(nil), summary.Artifacts...),
			RetainedSuccesses: summary.RetainedSuccesses, RetainedSuccessBytes: summary.RetainedSuccessBytes, SuccessArtifacts: append([]string(nil), summary.SuccessArtifacts...),
		})
	}
	if err := reportProgress(ProgressPreparing, 0); err != nil {
		return summary, &HostError{Reason: "progress_output", Err: err}
	}
	batchComplete := false
	defer func() {
		if batchComplete || retErr == nil {
			return
		}
		if errors.Is(overallCtx.Err(), context.DeadlineExceeded) {
			return
		}
		reason := "runner_failure"
		var hostError *HostError
		if errors.As(retErr, &hostError) {
			reason = hostError.Reason
		}
		var missing *ioprofile.MissingSemanticProbesError
		if errors.As(retErr, &missing) {
			reason = "semantic_coverage"
		}
		if partialErr := journal.Fail(reason, retErr); partialErr != nil {
			retErr = errors.Join(retErr, partialErr)
		}
	}()
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: "overall_timeout", Err: err}
	}
	selectedProfile := ioprofile.Default()
	if !resuming {
		if err := journal.BeginPreparation(); err != nil {
			return summary, &HostError{Reason: "target_preparation_setup", Err: err}
		}
		config.Target.PreparationRoot = journal.PreparedPath()
		preparer := config.Preparer
		if preparer == nil {
			moduleCache, cacheErr := target.ReadModuleCache(overallCtx, config.Target.ToolchainRoot)
			if cacheErr != nil {
				return summary, cacheErr
			}
			var profileErr error
			config.Target, _, profileErr = selectedProfile.PrepareBuildOverlay(config.Target, moduleCache)
			if profileErr != nil {
				return summary, profileErr
			}
		}
		if preparer == nil {
			preparer = targetPreparer{}
		}
		prepared, err = preparer.Prepare(overallCtx, config.Target)
		if err != nil {
			reason := "target_preparation"
			if errors.Is(overallCtx.Err(), context.DeadlineExceeded) {
				reason = "overall_timeout"
			}
			if partialErr := journal.FailPreparation(reason, err); partialErr != nil {
				err = errors.Join(err, partialErr)
			}
			return summary, &HostError{Reason: reason, Err: err}
		}
		if profileErr := selectedProfile.ValidatePreparedTarget(config.Target, prepared, config.Environment); profileErr != nil {
			return summary, profileErr
		}
		plan, err := batchPlan(config, journal, prepared, baseEnvironment, readOnlyMounts, selection.Count())
		if err != nil {
			return summary, &HostError{Reason: "batch_plan", Err: err}
		}
		if err := journal.RecordPlan(plan); err != nil {
			return summary, &HostError{Reason: "batch_plan", Err: err}
		}
		if err := journal.CompletePreparation(); err != nil {
			return summary, &HostError{Reason: "partial_cleanup", Err: err}
		}
	}
	executor := config.Executor
	if executor == nil {
		executor = processExecutor{}
	}

	if !resuming {
		err = journal.StartRuns()
	}
	if err != nil {
		return summary, &HostError{Reason: "runs_create", Err: err}
	}
	if err := reportProgress(ProgressRunning, 0); err != nil {
		return summary, &HostError{Reason: "progress_output", Err: err}
	}
	activeCtx, activeCancel := context.WithCancel(overallCtx)
	defer activeCancel()
	completions := make(chan runCompletion, config.Parallel)
	completed := make(map[uint64]struct{})
	active := 0
	exhausted := false
	stopped := false
	var hostFailure error
	distinct := make(map[record.SHA256]string)
	semanticProbes := make(map[string]struct{})
	if resuming {
		var restored Summary
		restored, distinct, semanticProbes, completed, err = restoreResumeSummary(batchPath, selection, resumedRuns)
		if err != nil {
			return summary, &HostError{Reason: "resume_setup", Err: err}
		}
		summary = restored
		stopped = resumeAlreadyStopped(config, &summary)
	}
	jobs := pendingJobs{seeds: selection.Iterator(), completed: completed}
	failureStore := artifact.Store{Root: journal.FailuresPath(), Context: overallCtx}
	publishRunnerFailure := func(completion runCompletion, reason string) error {
		if !completion.result.Captured {
			return nil
		}
		worldBundle := noneWorldBundle()
		outcome := executionoutcome.Classification{
			Domain: "runner", Reason: reason, Termination: "none",
			ArtifactKind: record.ArtifactRunnerFailure, ReplayMode: record.ReplayNone,
		}
		mountArtifact, err := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
		if err != nil {
			return fmt.Errorf("construct read-only mount artifact: %w", err)
		}
		manifest, err := manifestForRun(config, prepared, baseEnvironment, completion, outcome, runID, worldBundle.Manifest, mountArtifact)
		if err != nil {
			return fmt.Errorf("construct Runner failure manifest: %w", err)
		}
		published, err := failureStore.Publish(artifact.Input{
			Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
			IOTranscript: completion.result.IOTranscript.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
		})
		if err != nil {
			return fmt.Errorf("publish Runner failure artifact: %w", err)
		}
		signature := published.Manifest.Outcome.FailureSignature
		if _, found := distinct[signature]; !found {
			distinct[signature] = published.Path
			summary.Artifacts = append(summary.Artifacts, published.Path)
		}
		summary.DistinctFailures = uint64(len(distinct))
		artifactRelative, err := filepath.Rel(batchPath, published.Path)
		if err != nil {
			return fmt.Errorf("make Runner failure artifact path relative: %w", err)
		}
		run := artifact.RunRecord{
			SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
			Domain: "runner", Reason: reason, Termination: "none", FailureSignature: &signature, Artifact: &artifactRelative,
			ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
		}
		if err := journal.AppendRun(run); err != nil {
			return fmt.Errorf("append Runner failure result: %w", err)
		}
		return nil
	}
	completePartial := func(run *artifact.RunJournal) {
		if cleanupErr := run.Complete(); cleanupErr != nil && hostFailure == nil {
			hostFailure = &HostError{Reason: "partial_cleanup", Err: cleanupErr}
			stopped = true
			activeCancel()
		}
	}

	launch := func(job runJob) {
		active++
		go runSeed(activeCtx, config, executor, prepared, baseEnvironment, selectedProfile, readOnlyMounts, journal, job, completions)
	}
	var progressTicker *time.Ticker
	var progressTicks <-chan time.Time
	if config.Progress != nil {
		interval := config.ProgressInterval
		if interval <= 0 {
			interval = 5 * time.Second
		}
		progressTicker = time.NewTicker(interval)
		progressTicks = progressTicker.C
		defer progressTicker.Stop()
	}
	runningReported := false
	for active > 0 || !exhausted && !stopped {
		for active < config.Parallel && !exhausted && !stopped && overallCtx.Err() == nil {
			job, ok := jobs.Next()
			if !ok {
				exhausted = true
				break
			}
			launch(job)
		}
		if active > 0 && !runningReported {
			runningReported = true
			if err := reportProgress(ProgressRunning, active); err != nil {
				hostFailure = &HostError{Reason: "progress_output", Err: err}
				stopped = true
				activeCancel()
			}
		}
		if overallCtx.Err() != nil && hostFailure == nil && !stopped {
			hostFailure = &HostError{Reason: "overall_timeout", Err: overallCtx.Err()}
			stopped = true
			activeCancel()
		}
		if active == 0 {
			break
		}
		var completion runCompletion
		select {
		case completion = <-completions:
		case <-progressTicks:
			if err := reportProgress(ProgressRunning, active); err != nil {
				hostFailure = &HostError{Reason: "progress_output", Err: err}
				stopped = true
				activeCancel()
			}
			continue
		}
		active--
		summary.Attempted++
		if overallCtx.Err() != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "overall_timeout", Err: overallCtx.Err()}
			}
			stopped = true
			activeCancel()
			continue
		}
		if completion.err != nil {
			if partialErr := preservePartial(completion.journal); partialErr != nil {
				completion.err = errors.Join(completion.err, partialErr)
			}
			if publishErr := publishRunnerFailure(completion, "target_supervision"); publishErr != nil {
				completion.err = errors.Join(completion.err, publishErr)
			}
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "target_supervision", Err: completion.err}
				stopped = true
				activeCancel()
			}
			continue
		}
		if err := prepared.Verify(); err != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "prepared_target_integrity", Err: err}
				stopped = true
				activeCancel()
			}
			continue
		}
		if completion.result.Cancelled && stopped {
			summary.Cancelled++
			if partialErr := preservePartial(completion.journal); partialErr != nil {
				hostFailure = errors.Join(hostFailure, &HostError{Reason: "partial_write", Err: partialErr})
			}
			if hostFailure != nil {
				reason := "runner_failure"
				var hostError *HostError
				if errors.As(hostFailure, &hostError) {
					reason = hostError.Reason
				}
				if publishErr := publishRunnerFailure(completion, reason); publishErr != nil {
					hostFailure = errors.Join(hostFailure, publishErr)
				}
				continue
			}
			run := artifact.RunRecord{
				SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
				Domain: "runner", Reason: "runner_cancelled", Termination: "none", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
			}
			if err := journal.AppendRun(run); err != nil && hostFailure == nil {
				hostFailure = &HostError{Reason: "runs_append", Err: err}
				stopped = true
				activeCancel()
			}
			continue
		}
		worldBundle := noneWorldBundle()
		if len(completion.result.WorldRecord) != 0 {
			recording, decodeErr := world.DecodeRecording(completion.result.WorldRecord)
			if decodeErr != nil {
				err = decodeErr
			} else {
				worldBundle, err = worldrecord.ComposeRecording(recording, config.WorldTransitionLimit)
			}
			if err == nil {
				initialWorld, _, validateErr := worldrecord.Validate(worldBundle.Manifest, worldBundle.Payloads)
				if validateErr != nil {
					err = validateErr
				} else if worldBundle.Manifest.Initial.Schema != "gomadv3.world.snapshot/v1" || uint64(initialWorld.Config.Seed) != completion.job.seed {
					err = fmt.Errorf("World record seed or schema does not match seed %d", completion.job.seed)
				}
			}
			if err != nil {
				if publishErr := publishRunnerFailure(completion, "world_record"); publishErr != nil {
					err = errors.Join(err, publishErr)
				}
				if hostFailure == nil {
					hostFailure = &HostError{Reason: "world_record", Err: err}
					stopped = true
					activeCancel()
				}
				continue
			}
		}
		runCoverage := ioprofile.SemanticCoverage{}
		if config.Coverage == CoverageSemantic {
			coverage, coverageErr := ioprofile.DecodeSemanticCoverage(completion.result.IOTranscript.Bytes)
			if coverageErr != nil {
				if partialErr := preservePartial(completion.journal); partialErr != nil {
					coverageErr = errors.Join(coverageErr, partialErr)
				}
				if hostFailure == nil {
					hostFailure = &HostError{Reason: "semantic_coverage", Err: coverageErr}
					stopped = true
					activeCancel()
				}
				continue
			}
			runCoverage = coverage
		}
		novelProbes := novelSemanticProbes(runCoverage.Probes, semanticProbes)
		outcome := executionoutcome.Classify(completion.result, false, worldBundle.Manifest.Terminal)
		if config.CollectRunEvidence {
			mountArtifact, evidenceErr := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
			if evidenceErr != nil {
				if hostFailure == nil {
					hostFailure = &HostError{Reason: "run_evidence", Err: evidenceErr}
					stopped = true
					activeCancel()
				}
				continue
			}
			evidence := runEvidence(config, prepared, baseEnvironment, completion, outcome, worldBundle.Manifest, mountArtifact, runCoverage)
			summary.RunEvidence = &evidence
		}
		if err := completion.journal.Transition(artifact.RunClassified); err != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "partial_write", Err: err}
				stopped = true
				activeCancel()
			}
			continue
		}
		if overallErr := overallCtx.Err(); overallErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "overall_timeout", Err: overallErr}
				stopped = true
				activeCancel()
			}
			if partialErr := preservePartial(completion.journal); partialErr != nil {
				hostFailure = errors.Join(hostFailure, &HostError{Reason: "partial_write", Err: partialErr})
			}
			continue
		}
		if outcome.Domain == "success" {
			run := artifact.RunRecord{
				SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
				Domain: "success", Reason: outcome.Reason, Termination: "exit", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
			}
			setRunTranscript(&run, completion.result.IOTranscript)
			run.SemanticProbes = append([]string(nil), runCoverage.Probes...)
			retain := config.KeepSuccesses == KeepSuccessesAll || config.KeepSuccesses == KeepSuccessesNovel && len(novelProbes) != 0
			if retain {
				if !completion.result.IOTranscript.Complete {
					hostFailure = &HostError{Reason: "success_artifact_publication", Err: errors.New("retained success requires a complete I/O transcript for exact replay")}
					stopped = true
					activeCancel()
					continue
				}
				if summary.RetainedSuccesses >= config.SuccessArtifactLimit || summary.RetainedSuccessBytes >= config.SuccessBytesLimit {
					hostFailure = &HostError{Reason: "success_retention_capacity", Err: errors.New("successful-run retention capacity is exhausted")}
					stopped = true
					activeCancel()
					continue
				}
				mountArtifact, publishErr := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
				if publishErr == nil {
					var manifest record.Manifest
					manifest, publishErr = manifestForRun(config, prepared, baseEnvironment, completion, outcome, runID, worldBundle.Manifest, mountArtifact)
					if publishErr == nil {
						var published artifact.Artifact
						published, publishErr = (artifact.Store{Root: journal.SuccessesPath(), Context: overallCtx, MaximumBytes: config.SuccessBytesLimit - summary.RetainedSuccessBytes}).Publish(artifact.Input{
							Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
							IOTranscript: completion.result.IOTranscript.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
						})
						if publishErr == nil {
							relative, relErr := filepath.Rel(batchPath, published.Path)
							if relErr != nil {
								publishErr = relErr
							} else {
								bytes := record.Uint64String(published.StoredBytes)
								run.SuccessArtifact = &relative
								run.SuccessArtifactBytes = &bytes
								if config.KeepSuccesses == KeepSuccessesNovel {
									run.NovelSemanticProbes = append([]string(nil), novelProbes...)
								}
								summary.SuccessArtifacts = append(summary.SuccessArtifacts, published.Path)
								summary.RetainedSuccesses++
								summary.RetainedSuccessBytes += published.StoredBytes
							}
						}
					}
				}
				if publishErr != nil {
					reason := "success_artifact_publication"
					var capacity *artifact.CapacityError
					if errors.As(publishErr, &capacity) {
						reason = "success_retention_capacity"
					}
					hostFailure = &HostError{Reason: reason, Err: publishErr}
					stopped = true
					activeCancel()
					continue
				}
			}
			if err := journal.AppendRun(run); err != nil && hostFailure == nil {
				hostFailure = &HostError{Reason: "runs_append", Err: err}
				stopped = true
				activeCancel()
			}
			if hostFailure == nil {
				summary.Succeeded++
				addSemanticProbes(semanticProbes, runCoverage.Probes)
			}
			completePartial(completion.journal)
			continue
		}

		mountArtifact, manifestErr := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
		if manifestErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "manifest", Err: manifestErr}
				stopped = true
				activeCancel()
			}
			continue
		}
		manifest, manifestErr := manifestForRun(config, prepared, baseEnvironment, completion, outcome, runID, worldBundle.Manifest, mountArtifact)
		if manifestErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "manifest", Err: manifestErr}
				stopped = true
				activeCancel()
			}
			continue
		}
		published, publishErr := failureStore.Publish(artifact.Input{
			Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
			IOTranscript: completion.result.IOTranscript.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
		})
		if publishErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "artifact_publication", Err: publishErr}
				stopped = true
				activeCancel()
			}
			continue
		}
		if overallErr := overallCtx.Err(); overallErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "overall_timeout", Err: overallErr}
				stopped = true
				activeCancel()
			}
			continue
		}
		signature := published.Manifest.Outcome.FailureSignature
		if _, found := distinct[signature]; !found {
			distinct[signature] = published.Path
			summary.Artifacts = append(summary.Artifacts, published.Path)
		}
		summary.Failures++
		if outcome.Domain == "watchdog" {
			summary.Watchdogs++
		}
		if outcome.Reason == "world_replay_divergence" {
			summary.ReplayDivergences++
		}
		artifactRelative, relErr := filepath.Rel(batchPath, published.Path)
		if relErr != nil {
			hostFailure = &HostError{Reason: "artifact_path", Err: relErr}
			stopped = true
			activeCancel()
			continue
		}
		run := artifact.RunRecord{
			SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
			Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, FailureSignature: &signature,
			Artifact: &artifactRelative, ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
		}
		setRunTranscript(&run, completion.result.IOTranscript)
		run.SemanticProbes = append([]string(nil), runCoverage.Probes...)
		if err := journal.AppendRun(run); err != nil && hostFailure == nil {
			hostFailure = &HostError{Reason: "runs_append", Err: err}
			stopped = true
			activeCancel()
		}
		if hostFailure == nil {
			addSemanticProbes(semanticProbes, runCoverage.Probes)
		}
		completePartial(completion.journal)

		summary.DistinctFailures = uint64(len(distinct))
		if !stopped {
			switch config.OnFailure {
			case PolicyFirst:
				summary.StopReason = StopFirstFailure
				stopped = true
				activeCancel()
			case PolicyBudget:
				if summary.DistinctFailures >= config.FailureBudget {
					summary.StopReason = StopFailureBudget
					stopped = true
				}
			}
		}
	}

	if config.Coverage == CoverageSemantic {
		probes := make([]string, 0, len(semanticProbes))
		for probe := range semanticProbes {
			probes = append(probes, probe)
		}
		coverage, coverageErr := ioprofile.SummarizeSemanticProbes(probes)
		if coverageErr != nil && hostFailure == nil {
			hostFailure = &HostError{Reason: "semantic_coverage", Err: coverageErr}
		} else if coverageErr == nil {
			summary.SemanticCoverage = &coverage
		}
	}
	if overallCtx.Err() != nil && hostFailure == nil {
		hostFailure = &HostError{Reason: "overall_timeout", Err: overallCtx.Err()}
	}
	if hostFailure != nil {
		return summary, hostFailure
	}
	if summary.SemanticCoverage != nil {
		missing, err := ioprofile.MissingRequiredSemanticProbes(*summary.SemanticCoverage, config.RequiredSemanticProbes)
		if err != nil {
			return summary, err
		}
		if len(missing) != 0 {
			return summary, &ioprofile.MissingSemanticProbesError{Probes: missing}
		}
	}
	if err := prepared.Verify(); err != nil {
		return summary, &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: "overall_timeout", Err: err}
	}
	if summary.StopReason == "" {
		summary.StopReason = StopSeedsExhausted
	}
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: "overall_timeout", Err: err}
	}
	failureSignatures := make([]record.SHA256, 0, len(distinct))
	for signature := range distinct {
		failureSignatures = append(failureSignatures, signature)
	}
	if err := journal.Publish(artifact.BatchSummary{
		Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures, Watchdogs: summary.Watchdogs,
		Cancelled: summary.Cancelled, DistinctFailures: summary.DistinctFailures, RetainedSuccesses: summary.RetainedSuccesses, RetainedSuccessBytes: summary.RetainedSuccessBytes, StopReason: string(summary.StopReason), FailureSignatures: failureSignatures,
	}); err != nil {
		return summary, &HostError{Reason: "batch_publish", Err: err}
	}
	batchComplete = true
	if err := reportProgress(ProgressComplete, 0); err != nil {
		return summary, &HostError{Reason: "progress_output", Err: err}
	}
	return summary, nil
}

func validateConfig(config Config) (SeedSelection, []record.Environment, error) {
	if config.ResumeBatch != "" {
		if config.Preparer != nil {
			return SeedSelection{}, nil, fmt.Errorf("batch resume does not accept target preparation")
		}
		if config.RunnerBuild == "" {
			return SeedSelection{}, nil, fmt.Errorf("Runner build identity is required for batch resume")
		}
		if config.Executor == nil && len(config.SupervisorCommand) == 0 {
			return SeedSelection{}, nil, fmt.Errorf("supervisor command is required")
		}
		return SeedSelection{}, nil, nil
	}
	selection, err := ParseSeeds(config.Seeds)
	if err != nil {
		return SeedSelection{}, nil, err
	}
	if config.Parallel <= 0 {
		return SeedSelection{}, nil, fmt.Errorf("parallelism must be positive")
	}
	if config.RunTimeout <= 0 || config.OverallTimeout <= 0 {
		return SeedSelection{}, nil, fmt.Errorf("run and overall timeouts must be positive")
	}
	if config.TerminateGrace < 0 || config.TerminateGrace > config.RunTimeout || config.TerminateGrace > config.OverallTimeout {
		return SeedSelection{}, nil, fmt.Errorf("termination grace must fit inside all deadlines")
	}
	if config.OutputLimit == 0 || config.WorldTransitionLimit == 0 {
		return SeedSelection{}, nil, fmt.Errorf("output and World transition limits must be positive")
	}
	if config.Artifacts == "" || config.RunnerBuild == "" {
		return SeedSelection{}, nil, fmt.Errorf("artifact root and Runner build identity are required")
	}
	switch config.Coverage {
	case "", CoverageNone:
		if len(config.RequiredSemanticProbes) != 0 {
			return SeedSelection{}, nil, fmt.Errorf("required semantic probes require semantic coverage")
		}
	case CoverageSemantic:
		if _, err := ioprofile.MissingRequiredSemanticProbes(ioprofile.SemanticCoverage{}, config.RequiredSemanticProbes); err != nil {
			return SeedSelection{}, nil, err
		}
	default:
		return SeedSelection{}, nil, fmt.Errorf("unknown coverage mode %q", config.Coverage)
	}
	switch normalizedKeepSuccesses(config.KeepSuccesses) {
	case KeepSuccessesNone:
		if config.SuccessArtifactLimit != 0 || config.SuccessBytesLimit != 0 {
			return SeedSelection{}, nil, fmt.Errorf("disabled success retention does not accept capacity limits")
		}
	case KeepSuccessesNovel:
		if config.Coverage != CoverageSemantic || config.SuccessArtifactLimit == 0 || config.SuccessBytesLimit == 0 {
			return SeedSelection{}, nil, fmt.Errorf("novel success retention requires semantic coverage and explicit count and byte limits")
		}
	case KeepSuccessesAll:
		if config.SuccessArtifactLimit == 0 || config.SuccessBytesLimit == 0 {
			return SeedSelection{}, nil, fmt.Errorf("success retention requires explicit count and byte limits")
		}
	default:
		return SeedSelection{}, nil, fmt.Errorf("unknown successful-run retention policy %q", config.KeepSuccesses)
	}
	if config.CollectRunEvidence && (selection.Count() != 1 || config.Coverage != CoverageSemantic) {
		return SeedSelection{}, nil, fmt.Errorf("run evidence requires exactly one seed and semantic coverage")
	}
	switch config.OnFailure {
	case PolicyFirst, PolicyAll:
		if config.FailureBudget != 1 {
			return SeedSelection{}, nil, fmt.Errorf("failure budget is only configurable in budget mode")
		}
	case PolicyBudget:
		if config.FailureBudget == 0 {
			return SeedSelection{}, nil, fmt.Errorf("failure budget must be positive")
		}
	default:
		return SeedSelection{}, nil, fmt.Errorf("unknown failure policy %q", config.OnFailure)
	}
	if config.Executor == nil && len(config.SupervisorCommand) == 0 {
		return SeedSelection{}, nil, fmt.Errorf("supervisor command is required")
	}
	if len(config.IOROMounts) != 0 {
		if config.Target.WorkingDir == "" {
			return SeedSelection{}, nil, fmt.Errorf("read-only mounts require a target working directory")
		}
		if _, err := romount.ParseMappings(config.IOROMounts, config.Target.WorkingDir); err != nil {
			return SeedSelection{}, nil, err
		}
		limits := config.IOROMountLimits
		if limits == (romount.Limits{}) {
			limits = romount.DefaultLimits()
		}
		if _, err := romount.Prepare(nil, limits); err != nil {
			return SeedSelection{}, nil, err
		}
	}
	environment, err := parseEnvironment(config.Environment)
	if err != nil {
		return SeedSelection{}, nil, err
	}
	environment = append(environment, record.Environment{Name: "GOMADV3_IO_PROFILE", Value: ioprofile.Deterministic})
	sort.Slice(environment, func(i, j int) bool { return environment[i].Name < environment[j].Name })
	return selection, environment, nil
}

func parseEnvironment(entries []string) ([]record.Environment, error) {
	reserved := map[string]struct{}{
		"GOMADSEED": {}, "GOMADV3_CHILD_SEED": {}, "GOMADV3_IO_PROFILE": {}, "TZ": {}, "CGO_ENABLED": {}, "GODEBUG": {}, "GOMAXPROCS": {}, "GOEXPERIMENT": {},
		"LD_LIBRARY_PATH": {}, "LD_PRELOAD": {}, "DYLD_LIBRARY_PATH": {}, "DYLD_INSERT_LIBRARIES": {}, "LIBPATH": {}, "SHLIB_PATH": {},
	}
	seen := make(map[string]struct{}, len(entries))
	environment := make([]record.Environment, 0, len(entries))
	for _, entry := range entries {
		name, value, found := strings.Cut(entry, "=")
		if !found || !environmentName.MatchString(name) || strings.IndexByte(value, 0) >= 0 {
			return nil, fmt.Errorf("invalid target environment entry %q", entry)
		}
		if _, found := reserved[name]; found || strings.HasPrefix(name, "LD_") || strings.HasPrefix(name, "DYLD_") {
			return nil, fmt.Errorf("target environment name %q is reserved", name)
		}
		if _, found := seen[name]; found {
			return nil, fmt.Errorf("duplicate target environment name %q", name)
		}
		seen[name] = struct{}{}
		environment = append(environment, record.Environment{Name: name, Value: value})
	}
	sort.Slice(environment, func(i, j int) bool { return environment[i].Name < environment[j].Name })
	return environment, nil
}

func runSeed(ctx context.Context, config Config, executor Executor, prepared target.Prepared, baseEnvironment []record.Environment, profile ioprofile.ProfileSpec, readOnlyMounts []romount.Mapping, journal *artifact.BatchJournal, job runJob, completions chan<- runCompletion) {
	startedAt := time.Now().UTC()
	run, err := journal.BeginRun(job.ordinal, job.seed)
	completion := runCompletion{job: job, startedAt: startedAt, journal: run}
	if err != nil {
		completion.err = fmt.Errorf("create per-seed partial directory: %w", err)
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	if err := run.Transition(artifact.RunStarting); err != nil {
		completion.err = err
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	stdoutHead, err := run.CreateOutput("stdout")
	if err != nil {
		completion.err = err
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	stderrHead, err := run.CreateOutput("stderr")
	if err != nil {
		completion.err = errors.Join(err, run.CloseOutput("stdout", stdoutHead))
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	environment := environmentForSeed(baseEnvironment, job.seed)
	arguments := append([]string(nil), prepared.Argv[1:]...)
	var ioConfig []byte
	ioConfig, completion.err = profile.BootstrapFrame(prepared, config.RunnerBuild, job.seed)
	if completion.err == nil {
		completion.result, completion.err = executor.Run(ctx, process.Request{
			SupervisorCommand: append([]string(nil), config.SupervisorCommand...), Command: prepared.Path, Args: arguments, Argv0: prepared.Argv[0],
			BootstrapCommand: []string{config.SupervisorCommand[0], "__target_bootstrap"},
			Dir:              run.WorkPath(), Env: environmentStrings(environment), RunTimeout: config.RunTimeout,
			TerminateGrace: config.TerminateGrace, OutputLimit: config.OutputLimit,
			World: process.WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: config.WorldTransitionLimit, Seed: job.seed},
			IO: &process.IOCapability{
				Config:     append([]byte(nil), ioConfig...),
				Transcript: &process.IOTranscriptCapability{Limit: 64 << 20},
				ReadOnlyMount: &process.ReadOnlyMountCapability{
					Mappings: append([]romount.Mapping(nil), readOnlyMounts...), Limits: config.IOROMountLimits,
				},
			},
			StdoutHead: stdoutHead, StderrHead: stderrHead,
		})
	}
	if partialErr := run.Transition(artifact.RunExited); partialErr != nil {
		completion.err = errors.Join(completion.err, partialErr)
	}
	for _, output := range []struct {
		name string
		file *os.File
	}{{name: "stdout", file: stdoutHead}, {name: "stderr", file: stderrHead}} {
		if closeErr := run.CloseOutput(output.name, output.file); closeErr != nil {
			completion.err = errors.Join(completion.err, closeErr)
		}
	}
	completion.finishedAt = time.Now().UTC()
	if completion.err == nil {
		if err := run.Transition(artifact.RunCaptured); err != nil {
			completion.err = err
		}
	}
	completions <- completion
}

func environmentForSeed(base []record.Environment, seed uint64) []record.Environment {
	environment := append([]record.Environment(nil), base...)
	environment = append(environment, record.Environment{Name: "GOMADSEED", Value: strconv.FormatUint(seed, 10)}, record.Environment{Name: "TZ", Value: "UTC"})
	sort.Slice(environment, func(i, j int) bool { return environment[i].Name < environment[j].Name })
	return environment
}

func environmentStrings(environment []record.Environment) []string {
	result := make([]string, len(environment))
	for index, entry := range environment {
		result[index] = entry.Name + "=" + entry.Value
	}
	return result
}

func manifestForRun(config Config, prepared target.Prepared, baseEnvironment []record.Environment, completion runCompletion, outcome executionoutcome.Classification, runID string, recordedWorld record.World, mountArtifact *romount.ArtifactRecord) (record.Manifest, error) {
	profile := ioprofile.Default()
	recordedProfile := record.IOProfile{Name: profile.Name(), ImplementationSHA256: profile.ImplementationSHA256(), Inventory: string(profile.Inventory()), InventorySHA256: profile.InventorySHA256()}
	if completion.result.IOTranscript.Complete {
		recordedProfile.Transcript = &record.IOTranscript{
			Schema: "gomadv3.io-transcript/v1", File: "io/transcript.bin", SHA256: record.SHA256FromSum(completion.result.IOTranscript.SHA256),
			Bytes: record.Uint64String(len(completion.result.IOTranscript.Bytes)), Records: record.Uint64String(completion.result.IOTranscript.Records),
		}
	}
	if mountArtifact != nil {
		recordedProfile.ReadOnlyMounts = &mountArtifact.Manifest
	}
	return record.Manifest{
		SchemaVersion: record.SchemaVersion, ArtifactKind: outcome.ArtifactKind, CreatedAt: completion.finishedAt.Format(time.RFC3339Nano), BatchID: runID,
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed), ReplayMode: outcome.ReplayMode,
		Runner:    record.Runner{RecordContract: "gomadv3.run-record/v1", RunnerBuild: config.RunnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: record.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Target: record.Target{
			Kind: string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
			Argv: append([]string{}, prepared.Argv...), BuildTags: append([]string{}, prepared.BuildTags...), BuildInfo: prepared.BuildInfo,
		},
		IOProfile:   recordedProfile,
		Environment: environmentForSeed(baseEnvironment, completion.job.seed),
		Limits: record.Limits{
			RunTimeoutNanos: record.Uint64String(config.RunTimeout), OverallTimeoutNanos: record.Uint64String(config.OverallTimeout),
			TerminateGraceNanos: record.Uint64String(config.TerminateGrace), OutputBytes: record.Uint64String(config.OutputLimit),
			WorldTransitionBytes: record.Uint64String(config.WorldTransitionLimit),
			IOTranscriptBytes:    64 << 20,
		},
		World:   recordedWorld,
		Outcome: record.Outcome{Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, ExitCode: outcome.ExitCode, Signal: outcome.Signal, Deadline: outcome.Deadline},
		Streams: record.Streams{Stdout: streamRecord(completion.result.Stdout), Stderr: streamRecord(completion.result.Stderr)},
		Host:    record.Host{StartedAt: completion.startedAt.Format(time.RFC3339Nano), FinishedAt: completion.finishedAt.Format(time.RFC3339Nano), ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt)},
	}, nil
}

func mountArtifactForRun(mappings []romount.Mapping, limits romount.Limits, snapshot romount.Snapshot) (*romount.ArtifactRecord, error) {
	if len(mappings) == 0 {
		return nil, nil
	}
	encoded, err := romount.EncodeArtifact(mappings, limits, snapshot)
	if err != nil {
		return nil, err
	}
	return &encoded, nil
}

func setRunTranscript(run *artifact.RunRecord, transcript process.IOTranscript) {
	if !transcript.Complete {
		return
	}
	digest := record.SHA256FromSum(transcript.SHA256)
	records := record.Uint64String(transcript.Records)
	run.IOTranscriptSHA256 = &digest
	run.IOTranscriptRecords = &records
}

func novelSemanticProbes(observed []string, prior map[string]struct{}) []string {
	novel := make([]string, 0, len(observed))
	for _, probe := range observed {
		if _, found := prior[probe]; !found {
			novel = append(novel, probe)
		}
	}
	return novel
}

func addSemanticProbes(destination map[string]struct{}, probes []string) {
	for _, probe := range probes {
		destination[probe] = struct{}{}
	}
}

func preservePartial(run *artifact.RunJournal) error {
	if run == nil {
		return nil
	}
	return run.Preserve()
}

func streamRecord(output process.Output) record.Stream {
	return record.Stream{
		RetainedSHA256: record.SHA256FromSum(output.RetainedSHA256), FullSHA256: record.SHA256FromSum(output.FullSHA256), TotalBytes: record.Uint64String(output.TotalBytes),
		RetainedBytes: record.Uint64String(output.RetainedBytes), DiscardedBytes: record.Uint64String(output.DiscardedBytes), Truncated: output.Truncated,
	}
}

func noneWorldBundle() worldrecord.Bundle {
	manifest, payloads := record.NoneWorld()
	return worldrecord.Bundle{Manifest: manifest, Payloads: payloads}
}

func elapsedNanos(startedAt, finishedAt time.Time) record.Uint64String {
	elapsed := finishedAt.Sub(startedAt)
	if elapsed < 0 {
		return 0
	}
	return record.Uint64String(elapsed)
}

func newRunID() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "run-" + time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(random), nil
}
