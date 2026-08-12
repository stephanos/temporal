package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/internal/target"
	"go.temporal.io/server/tools/gomadv3/internal/worldrecord"
	"go.temporal.io/server/tools/gomadv3/world"
)

type FailurePolicy string

const (
	PolicyFirst  FailurePolicy = "first"
	PolicyBudget FailurePolicy = "budget"
	PolicyAll    FailurePolicy = "all"
)

type StopReason string

const (
	StopSeedsExhausted StopReason = "seeds_exhausted"
	StopFirstFailure   StopReason = "first_failure"
	StopFailureBudget  StopReason = "failure_budget"
)

type Preparer interface {
	Prepare(context.Context, target.Spec) (target.Prepared, error)
}

type Executor interface {
	Run(context.Context, process.Request) (process.Result, error)
}

type Config struct {
	Seeds                string
	Parallel             int
	RunTimeout           time.Duration
	OverallTimeout       time.Duration
	TerminateGrace       time.Duration
	OnFailure            FailurePolicy
	FailureBudget        uint64
	OutputLimit          uint64
	WorldTransitionLimit uint64
	Artifacts            string
	Environment          []string
	IOROMounts           []string
	IOROMountLimits      romount.Limits
	Target               target.Spec
	SupervisorCommand    []string
	CoordinatorCommand   []string
	RunnerBuild          string
	Preparer             Preparer
	Executor             Executor
}

type Summary struct {
	BatchPath        string
	SelectionCount   uint64
	Attempted        uint64
	Succeeded        uint64
	Failures         uint64
	Watchdogs        uint64
	Cancelled        uint64
	DistinctFailures uint64
	StopReason       StopReason
	Artifacts        []string
}

type RunSummary struct {
	SelectionOrdinal    record.Uint64String  `json:"selection_ordinal"`
	Seed                record.Uint64String  `json:"seed"`
	Domain              string               `json:"domain"`
	Reason              string               `json:"reason"`
	Termination         string               `json:"termination"`
	FailureSignature    *record.SHA256       `json:"failure_signature"`
	Artifact            *string              `json:"artifact"`
	ElapsedNanos        record.Uint64String  `json:"elapsed_nanos"`
	IOTranscriptSHA256  *record.SHA256       `json:"io_transcript_sha256"`
	IOTranscriptRecords *record.Uint64String `json:"io_transcript_records"`
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
	partial    string
}

type classifiedOutcome struct {
	Domain       string
	Reason       string
	Termination  string
	ExitCode     *record.Uint64String
	Signal       *string
	Deadline     *string
	ArtifactKind string
	ReplayMode   string
}

type batchRecord struct {
	SchemaVersion     uint32              `json:"schema_version"`
	Schema            string              `json:"schema"`
	RunID             string              `json:"run_id"`
	Selection         string              `json:"selection"`
	SelectionCount    record.Uint64String `json:"selection_count"`
	Attempted         record.Uint64String `json:"attempted"`
	Succeeded         record.Uint64String `json:"succeeded"`
	Failures          record.Uint64String `json:"failures"`
	Watchdogs         record.Uint64String `json:"watchdogs"`
	Cancelled         record.Uint64String `json:"cancelled"`
	DistinctFailures  record.Uint64String `json:"distinct_failures"`
	StopReason        StopReason          `json:"stop_reason"`
	RunsSHA256        record.SHA256       `json:"runs_sha256"`
	FailureSignatures []record.SHA256     `json:"failure_signatures"`
}

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func Run(ctx context.Context, config Config) (Summary, error) {
	if len(config.CoordinatorCommand) != 0 {
		if config.Preparer != nil || config.Executor != nil {
			return Summary{}, fmt.Errorf("isolated Runner does not accept injected preparation or execution")
		}
		return runIsolated(ctx, config)
	}
	return runLocal(ctx, config)
}

func runLocal(ctx context.Context, config Config) (summary Summary, retErr error) {
	selection, baseEnvironment, err := validateConfig(config)
	if err != nil {
		return Summary{}, err
	}
	config.Artifacts, err = filepath.Abs(config.Artifacts)
	if err != nil {
		return Summary{}, &HostError{Reason: "artifact_setup", Err: fmt.Errorf("resolve artifact root: %w", err)}
	}
	readOnlyMounts, err := romount.ParseMappings(config.IOROMounts, config.Target.WorkingDir)
	if err != nil {
		return Summary{}, err
	}
	if config.IOROMountLimits == (romount.Limits{}) {
		config.IOROMountLimits = romount.DefaultLimits()
	}
	overallCtx, overallCancel := context.WithTimeout(ctx, config.OverallTimeout)
	defer overallCancel()
	runID, err := newRunID()
	if err != nil {
		return Summary{}, &HostError{Reason: "run_id", Err: err}
	}
	batchPath := filepath.Join(config.Artifacts, "v1", runID)
	for _, directory := range []string{config.Artifacts, filepath.Join(config.Artifacts, "v1"), batchPath, filepath.Join(batchPath, "failures"), filepath.Join(batchPath, ".partial")} {
		if err := makePrivateDirectories(directory); err != nil {
			return Summary{}, &HostError{Reason: "artifact_setup", Err: err}
		}
	}
	summary = Summary{BatchPath: batchPath, SelectionCount: selection.Count()}
	batchPartial := filepath.Join(batchPath, ".partial", "batch")
	if err := makePrivateDirectories(batchPartial); err != nil {
		return summary, &HostError{Reason: "partial_setup", Err: err}
	}
	if err := writePreparationPartial(batchPartial, "running", nil, nil); err != nil {
		return summary, &HostError{Reason: "partial_write", Err: err}
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
		if partialErr := writePreparationPartial(batchPartial, "failed", &reason, retErr); partialErr != nil {
			retErr = errors.Join(retErr, partialErr)
		}
	}()
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: "overall_timeout", Err: err}
	}
	preparationRoot := filepath.Join(batchPath, ".prepared")
	preparationPartial := filepath.Join(batchPath, ".partial", "preparation")
	if err := makePrivateDirectories(preparationRoot); err != nil {
		return summary, &HostError{Reason: "target_preparation_setup", Err: err}
	}
	if err := makePrivateDirectories(preparationPartial); err != nil {
		return summary, &HostError{Reason: "partial_setup", Err: err}
	}
	if err := writePreparationPartial(preparationPartial, "preparing", nil, nil); err != nil {
		return summary, &HostError{Reason: "partial_write", Err: err}
	}
	config.Target.PreparationRoot = preparationRoot
	preparer := config.Preparer
	selectedProfile := ioprofile.Default()
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
	prepared, err := preparer.Prepare(overallCtx, config.Target)
	if err != nil {
		reason := "target_preparation"
		if errors.Is(overallCtx.Err(), context.DeadlineExceeded) {
			reason = "overall_timeout"
		}
		if partialErr := writePreparationPartial(preparationPartial, "failed", &reason, err); partialErr != nil {
			err = errors.Join(err, partialErr)
		}
		return summary, &HostError{Reason: reason, Err: err}
	}
	if profileErr := selectedProfile.ValidatePreparedTarget(config.Target, prepared, config.Environment); profileErr != nil {
		return summary, profileErr
	}
	if !selectedProfile.Ready {
		return summary, fmt.Errorf("deterministic I/O is not yet executable")
	}
	if err := os.RemoveAll(preparationPartial); err != nil {
		return summary, &HostError{Reason: "partial_cleanup", Err: err}
	}
	executor := config.Executor
	if executor == nil {
		executor = processExecutor{}
	}

	runsPath := filepath.Join(batchPath, "runs.jsonl")
	runsFile, err := os.OpenFile(runsPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return summary, &HostError{Reason: "runs_create", Err: err}
	}
	if err := runsFile.Chmod(0o600); err != nil {
		return summary, &HostError{Reason: "runs_permissions", Err: errors.Join(err, runsFile.Close())}
	}
	runsClosed := false
	defer func() {
		if !runsClosed {
			if closeErr := runsFile.Close(); closeErr != nil {
				retErr = errors.Join(retErr, &HostError{Reason: "runs_close", Err: closeErr})
			}
		}
	}()
	runsHasher := sha256.New()
	runsWriter := io.MultiWriter(runsFile, runsHasher)

	activeCtx, activeCancel := context.WithCancel(overallCtx)
	defer activeCancel()
	completions := make(chan runCompletion, config.Parallel)
	iterator := selection.Iterator()
	active := 0
	exhausted := false
	stopped := false
	var ordinal uint64
	var hostFailure error
	distinct := make(map[record.SHA256]string)
	store := artifact.Store{Root: filepath.Join(batchPath, "failures"), Context: overallCtx}
	publishRunnerFailure := func(completion runCompletion, reason string) error {
		if !completion.result.Captured {
			return nil
		}
		worldBundle := noneWorldBundle()
		outcome := classifiedOutcome{
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
		published, err := store.Publish(artifact.Input{
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
		run := RunSummary{
			SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
			Domain: "runner", Reason: reason, Termination: "none", FailureSignature: &signature, Artifact: &artifactRelative,
			ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
		}
		if err := appendRun(runsWriter, runsFile, run); err != nil {
			return fmt.Errorf("append Runner failure result: %w", err)
		}
		return nil
	}
	completePartial := func(path string) {
		if cleanupErr := removeCompletedPartial(path); cleanupErr != nil && hostFailure == nil {
			hostFailure = &HostError{Reason: "partial_cleanup", Err: cleanupErr}
			stopped = true
			activeCancel()
		}
	}

	launch := func(job runJob) {
		active++
		go runSeed(activeCtx, config, executor, prepared, baseEnvironment, &selectedProfile, readOnlyMounts, batchPath, job, completions)
	}
	for active > 0 || !exhausted && !stopped {
		for active < config.Parallel && !exhausted && !stopped && overallCtx.Err() == nil {
			seed, ok := iterator.Next()
			if !ok {
				exhausted = true
				break
			}
			launch(runJob{ordinal: ordinal, seed: seed})
			ordinal++
		}
		if overallCtx.Err() != nil && hostFailure == nil && !stopped {
			hostFailure = &HostError{Reason: "overall_timeout", Err: overallCtx.Err()}
			stopped = true
			activeCancel()
		}
		if active == 0 {
			break
		}
		completion := <-completions
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
			if partialErr := writePartial(completion.partial, completion.job, "preserve-partial"); partialErr != nil {
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
			if partialErr := writePartial(completion.partial, completion.job, "preserve-partial"); partialErr != nil {
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
			run := RunSummary{
				SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
				Domain: "runner", Reason: "runner_cancelled", Termination: "none", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
			}
			if err := appendRun(runsWriter, runsFile, run); err != nil && hostFailure == nil {
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
		outcome := classify(completion.result, false, worldBundle.Manifest.Terminal)
		if err := writePartial(completion.partial, completion.job, "classified"); err != nil {
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
			if partialErr := writePartial(completion.partial, completion.job, "preserve-partial"); partialErr != nil {
				hostFailure = errors.Join(hostFailure, &HostError{Reason: "partial_write", Err: partialErr})
			}
			continue
		}
		if outcome.Domain == "success" {
			summary.Succeeded++
			run := RunSummary{
				SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
				Domain: "success", Reason: outcome.Reason, Termination: "exit", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
			}
			setRunTranscript(&run, completion.result.IOTranscript)
			if err := appendRun(runsWriter, runsFile, run); err != nil && hostFailure == nil {
				hostFailure = &HostError{Reason: "runs_append", Err: err}
				stopped = true
				activeCancel()
			}
			completePartial(completion.partial)
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
		published, publishErr := store.Publish(artifact.Input{
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
		artifactRelative, relErr := filepath.Rel(batchPath, published.Path)
		if relErr != nil {
			hostFailure = &HostError{Reason: "artifact_path", Err: relErr}
			stopped = true
			activeCancel()
			continue
		}
		run := RunSummary{
			SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
			Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, FailureSignature: &signature,
			Artifact: &artifactRelative, ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
		}
		setRunTranscript(&run, completion.result.IOTranscript)
		if err := appendRun(runsWriter, runsFile, run); err != nil && hostFailure == nil {
			hostFailure = &HostError{Reason: "runs_append", Err: err}
			stopped = true
			activeCancel()
		}
		completePartial(completion.partial)

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

	if overallCtx.Err() != nil && hostFailure == nil {
		hostFailure = &HostError{Reason: "overall_timeout", Err: overallCtx.Err()}
	}
	if hostFailure != nil {
		return summary, hostFailure
	}
	if err := prepared.Verify(); err != nil {
		return summary, &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: "overall_timeout", Err: err}
	}
	if err := os.RemoveAll(preparationRoot); err != nil {
		return summary, &HostError{Reason: "prepared_target_cleanup", Err: err}
	}
	if summary.StopReason == "" {
		summary.StopReason = StopSeedsExhausted
	}
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: "overall_timeout", Err: err}
	}
	if err := runsFile.Sync(); err != nil {
		return summary, &HostError{Reason: "runs_sync", Err: err}
	}
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: "overall_timeout", Err: err}
	}
	if err := runsFile.Close(); err != nil {
		return summary, &HostError{Reason: "runs_close", Err: err}
	}
	runsClosed = true
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: "overall_timeout", Err: err}
	}
	failureSignatures := make([]record.SHA256, 0, len(distinct))
	for signature := range distinct {
		failureSignatures = append(failureSignatures, signature)
	}
	sort.Slice(failureSignatures, func(i, j int) bool { return failureSignatures[i] < failureSignatures[j] })
	batch := batchRecord{
		SchemaVersion: record.SchemaVersion, Schema: "gomadv3.batch/v1", RunID: runID, Selection: config.Seeds,
		SelectionCount: record.Uint64String(summary.SelectionCount), Attempted: record.Uint64String(summary.Attempted), Succeeded: record.Uint64String(summary.Succeeded),
		Failures: record.Uint64String(summary.Failures), Watchdogs: record.Uint64String(summary.Watchdogs), Cancelled: record.Uint64String(summary.Cancelled),
		DistinctFailures: record.Uint64String(summary.DistinctFailures), StopReason: summary.StopReason,
		RunsSHA256:        record.SHA256("sha256:" + hex.EncodeToString(runsHasher.Sum(nil))),
		FailureSignatures: failureSignatures,
	}
	batchBytes, err := record.CanonicalJSON(batch)
	if err != nil {
		return summary, &HostError{Reason: "batch_encode", Err: err}
	}
	if err := atomicWriteContext(overallCtx, filepath.Join(batchPath, "batch.json"), batchBytes); err != nil {
		return summary, &HostError{Reason: "batch_publish", Err: err}
	}
	if err := syncDirectoryContext(overallCtx, batchPath); err != nil {
		return summary, &HostError{Reason: "batch_sync", Err: err}
	}
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: "overall_timeout", Err: err}
	}
	if err := os.RemoveAll(batchPartial); err != nil {
		return summary, &HostError{Reason: "partial_cleanup", Err: err}
	}
	batchComplete = true
	return summary, nil
}

func validateConfig(config Config) (SeedSelection, []record.Environment, error) {
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

func runSeed(ctx context.Context, config Config, executor Executor, prepared target.Prepared, baseEnvironment []record.Environment, profile *ioprofile.Profile, readOnlyMounts []romount.Mapping, batchPath string, job runJob, completions chan<- runCompletion) {
	startedAt := time.Now().UTC()
	partial := filepath.Join(batchPath, ".partial", fmt.Sprintf("%020d-%d", job.ordinal, job.seed))
	completion := runCompletion{job: job, startedAt: startedAt, partial: partial}
	if err := makePrivateDirectories(partial); err != nil {
		completion.err = fmt.Errorf("create per-seed partial directory: %w", err)
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	if err := writePartial(partial, job, "staging"); err != nil {
		completion.err = err
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	if err := makePrivateDirectories(filepath.Join(partial, "work")); err != nil {
		completion.err = fmt.Errorf("create per-seed working directory: %w", err)
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	if err := writePartial(partial, job, "starting"); err != nil {
		completion.err = err
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	stdoutHead, err := openPartialOutput(filepath.Join(partial, "stdout.head"))
	if err != nil {
		completion.err = err
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	stderrHead, err := openPartialOutput(filepath.Join(partial, "stderr.head"))
	if err != nil {
		completion.err = errors.Join(err, closePartialOutput("stdout", stdoutHead))
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
			Dir:              filepath.Join(partial, "work"), Env: environmentStrings(environment), RunTimeout: config.RunTimeout,
			TerminateGrace: config.TerminateGrace, OutputLimit: config.OutputLimit,
			WorldRecordLimit: world.MaximumRecordingBytes, WorldTransitionLimit: config.WorldTransitionLimit,
			WorldSeed:         job.seed,
			IOConfig:          append([]byte(nil), ioConfig...),
			IOROMounts:        append([]romount.Mapping(nil), readOnlyMounts...),
			IOROMountLimits:   config.IOROMountLimits,
			IOTranscriptLimit: 64 << 20,
			StdoutHead:        stdoutHead, StderrHead: stderrHead,
		})
	}
	if partialErr := writePartial(partial, job, "exited"); partialErr != nil {
		completion.err = errors.Join(completion.err, partialErr)
	}
	for _, output := range []struct {
		name string
		file *os.File
	}{{name: "stdout", file: stdoutHead}, {name: "stderr", file: stderrHead}} {
		if syncErr := output.file.Sync(); syncErr != nil {
			completion.err = errors.Join(completion.err, fmt.Errorf("sync partial %s: %w", output.name, syncErr))
		}
		if closeErr := closePartialOutput(output.name, output.file); closeErr != nil {
			completion.err = errors.Join(completion.err, closeErr)
		}
	}
	completion.finishedAt = time.Now().UTC()
	if completion.err == nil {
		if err := writePartial(partial, job, "captured"); err != nil {
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

func classify(result process.Result, cancelled bool, terminal record.WorldTerminal) classifiedOutcome {
	if cancelled || result.Cancelled {
		return classifiedOutcome{Domain: "runner", Reason: "runner_cancelled", Termination: "none", ArtifactKind: record.ArtifactRunnerFailure, ReplayMode: record.ReplayNone}
	}
	if result.WatchdogTimeout {
		deadline := "run_timeout"
		return classifiedOutcome{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline, ArtifactKind: record.ArtifactWatchdogTimeout, ReplayMode: record.ReplayDiagnostic}
	}
	if reason := worldFailureReason(terminal.Kind); reason != "" {
		outcome := classifiedOutcome{Domain: "target", Reason: reason, ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}
		if result.Termination == process.TerminationSignal {
			outcome.Termination = "signal"
			outcome.Signal = &result.Signal
		} else {
			outcome.Termination = "exit"
			exitCode := record.Uint64String(result.ExitCode)
			outcome.ExitCode = &exitCode
		}
		return outcome
	}
	if result.Termination == process.TerminationExit && result.ExitCode == 0 {
		reason := "success"
		if terminal.Kind == string(world.TerminalIdle) {
			reason = "world_idle"
		}
		return classifiedOutcome{Domain: "success", Reason: reason, Termination: "exit"}
	}
	outcome := classifiedOutcome{Domain: "target", Reason: "nonzero_exit", ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}
	if result.Termination == process.TerminationSignal {
		outcome.Termination = "signal"
		outcome.Signal = &result.Signal
		outcome.Reason = "external_signal"
		return outcome
	}
	outcome.Termination = "exit"
	exitCode := record.Uint64String(result.ExitCode)
	outcome.ExitCode = &exitCode
	diagnostic := string(result.Stderr.Bytes)
	switch {
	case strings.HasPrefix(diagnostic, "runtime: GOMADSEED does not support cgo or external linking"):
		outcome.Reason = "unsupported_deterministic_mode"
	case strings.HasPrefix(diagnostic, "fatal error: all goroutines are asleep - deadlock!"):
		outcome.Reason = "deterministic_deadlock"
	case strings.HasPrefix(diagnostic, "panic: test timed out after"):
		outcome.Reason = "logical_test_timeout"
	case strings.HasPrefix(diagnostic, "panic:") || strings.HasPrefix(diagnostic, "fatal error:"):
		outcome.Reason = "panic_or_runtime_fatal"
	}
	return outcome
}

func worldFailureReason(kind string) string {
	switch world.TerminalKind(kind) {
	case world.TerminalDeadlock:
		return "world_deadlock"
	case world.TerminalCapacity:
		return "world_capacity"
	case world.TerminalReplayDivergence:
		return "world_replay_divergence"
	case world.TerminalInvalidInput:
		return "world_invalid_input"
	default:
		return ""
	}
}

func manifestForRun(config Config, prepared target.Prepared, baseEnvironment []record.Environment, completion runCompletion, outcome classifiedOutcome, runID string, recordedWorld record.World, mountArtifact *romount.ArtifactRecord) (record.Manifest, error) {
	profile := ioprofile.Default()
	recordedProfile := record.IOProfile{Name: profile.Name, ImplementationSHA256: profile.ImplementationSHA256, Inventory: string(profile.Inventory), InventorySHA256: profile.InventorySHA256}
	if completion.result.IOTranscript.Complete {
		recordedProfile.Transcript = &record.IOTranscript{
			Schema: "gomadv3.io-transcript/v1", File: "io/transcript.bin", SHA256: sha256Record(completion.result.IOTranscript.SHA256),
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

func setRunTranscript(run *RunSummary, transcript process.IOTranscript) {
	if !transcript.Complete {
		return
	}
	digest := sha256Record(transcript.SHA256)
	records := record.Uint64String(transcript.Records)
	run.IOTranscriptSHA256 = &digest
	run.IOTranscriptRecords = &records
}

func streamRecord(output process.Output) record.Stream {
	return record.Stream{
		RetainedSHA256: sha256Record(output.RetainedSHA256), FullSHA256: sha256Record(output.FullSHA256), TotalBytes: record.Uint64String(output.TotalBytes),
		RetainedBytes: record.Uint64String(output.RetainedBytes), DiscardedBytes: record.Uint64String(output.DiscardedBytes), Truncated: output.Truncated,
	}
}

func sha256Record(value [sha256.Size]byte) record.SHA256 {
	return record.SHA256("sha256:" + hex.EncodeToString(value[:]))
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

func appendRun(writer io.Writer, file *os.File, run RunSummary) error {
	encoded, err := record.CanonicalJSON(run)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return err
	}
	return file.Sync()
}

func writePartial(directory string, job runJob, state string) error {
	payload := struct {
		SchemaVersion    uint32              `json:"schema_version"`
		State            string              `json:"state"`
		SelectionOrdinal record.Uint64String `json:"selection_ordinal"`
		Seed             record.Uint64String `json:"seed"`
	}{SchemaVersion: record.SchemaVersion, State: state, SelectionOrdinal: record.Uint64String(job.ordinal), Seed: record.Uint64String(job.seed)}
	encoded, err := record.CanonicalJSON(payload)
	if err != nil {
		return fmt.Errorf("encode partial state: %w", err)
	}
	if err := atomicWrite(filepath.Join(directory, "partial.json"), encoded); err != nil {
		return fmt.Errorf("write partial state: %w", err)
	}
	return nil
}

func openPartialOutput(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create partial output %s: %w", filepath.Base(path), err)
	}
	if err := file.Chmod(0o600); err != nil {
		return nil, fmt.Errorf("set partial output mode %s: %w", filepath.Base(path), errors.Join(err, file.Close()))
	}
	return file, nil
}

func closePartialOutput(name string, file *os.File) error {
	if err := file.Close(); err != nil {
		return fmt.Errorf("close partial %s: %w", name, err)
	}
	return nil
}

func writePreparationPartial(directory, state string, reason *string, preparationErr error) error {
	var detail *string
	if preparationErr != nil {
		message := preparationErr.Error()
		detail = &message
	}
	payload := struct {
		SchemaVersion uint32  `json:"schema_version"`
		State         string  `json:"state"`
		Reason        *string `json:"reason"`
		Detail        *string `json:"detail"`
	}{SchemaVersion: record.SchemaVersion, State: state, Reason: reason, Detail: detail}
	encoded, err := record.CanonicalJSON(payload)
	if err != nil {
		return fmt.Errorf("encode preparation partial state: %w", err)
	}
	if err := atomicWrite(filepath.Join(directory, "partial.json"), encoded); err != nil {
		return fmt.Errorf("write preparation partial state: %w", err)
	}
	return nil
}

func atomicWrite(path string, data []byte) (retErr error) {
	return atomicWriteContext(context.Background(), path, data)
}

func atomicWriteContext(ctx context.Context, path string, data []byte) (retErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".tmp-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	keep := true
	defer func() {
		if keep {
			if removeErr := os.Remove(temporaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
				retErr = errors.Join(retErr, removeErr)
			}
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(data); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	keep = false
	return syncDirectoryContext(ctx, directory)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func syncDirectoryContext(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := syncDirectory(path); err != nil {
		return err
	}
	return ctx.Err()
}

func removeCompletedPartial(path string) error {
	if path == "" {
		return nil
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove completed partial %s: %w", filepath.Base(path), err)
	}
	return nil
}

func makePrivateDirectories(path string) error {
	path = filepath.Clean(path)
	missing := []string{}
	current := path
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%s is not a directory", current)
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("no existing parent for %s", path)
		}
		current = parent
	}
	for index := len(missing) - 1; index >= 0; index-- {
		if err := os.Mkdir(missing[index], 0o700); err != nil {
			return err
		}
		if err := os.Chmod(missing[index], 0o700); err != nil {
			return err
		}
	}
	return os.Chmod(path, 0o700)
}

func newRunID() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "run-" + time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(random), nil
}
