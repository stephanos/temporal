package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"

	"go.temporal.io/server/tools/gomadv3/internal/choicefrontier"
	"go.temporal.io/server/tools/gomadv3/internal/filelock"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

type BatchConfig struct {
	Root           string
	RunID          string
	Strategy       string
	Selection      string
	SelectionCount uint64
}

type BatchSummary struct {
	Attempted                    uint64
	Succeeded                    uint64
	Failures                     uint64
	Watchdogs                    uint64
	Cancelled                    uint64
	DistinctFailures             uint64
	RetainedSuccesses            uint64
	RetainedSuccessBytes         uint64
	StopReason                   string
	FailureSignatures            []record.SHA256
	Frontier                     *choicefrontier.Summary
	FrontierImplementationSHA256 record.SHA256
	FrontierChainSHA256          record.SHA256
	RecoveryExecutions           uint64
}

type RunRecord struct {
	Strategy                    string               `json:"strategy,omitempty"`
	Round                       *record.Uint64String `json:"round,omitempty"`
	CandidateSHA256             record.SHA256        `json:"candidate_sha256,omitempty"`
	ParentCandidateSHA256       record.SHA256        `json:"parent_candidate_sha256,omitempty"`
	PrefixSHA256                record.SHA256        `json:"prefix_sha256,omitempty"`
	ForcedDepth                 *record.Uint64String `json:"forced_depth,omitempty"`
	OutcomeSHA256               record.SHA256        `json:"outcome_sha256,omitempty"`
	SelectionOrdinal            record.Uint64String  `json:"selection_ordinal"`
	Seed                        record.Uint64String  `json:"seed"`
	Domain                      string               `json:"domain"`
	Reason                      string               `json:"reason"`
	Termination                 string               `json:"termination"`
	FailureSignature            *record.SHA256       `json:"failure_signature"`
	Artifact                    *string              `json:"artifact"`
	ElapsedNanos                record.Uint64String  `json:"elapsed_nanos"`
	IOTranscriptSHA256          *record.SHA256       `json:"io_transcript_sha256"`
	IOTranscriptRecords         *record.Uint64String `json:"io_transcript_records"`
	ChoiceTraceSHA256           *record.SHA256       `json:"choice_trace_sha256,omitempty"`
	ChoiceTraceRecords          *record.Uint64String `json:"choice_trace_records,omitempty"`
	ChoiceTraceBranchingRecords *record.Uint64String `json:"choice_trace_branching_records,omitempty"`
	ChoiceTraceTerminalState    *string              `json:"choice_trace_terminal_state,omitempty"`
	ChoiceTapeSHA256            *record.SHA256       `json:"choice_tape_sha256,omitempty"`
	ChoiceDecisions             *record.Uint64String `json:"choice_decisions,omitempty"`
	SemanticProbes              []string             `json:"semantic_probes,omitempty"`
	ChoiceFeatures              []string             `json:"choice_features,omitempty"`
	SuccessArtifact             *string              `json:"success_artifact,omitempty"`
	SuccessArtifactBytes        *record.Uint64String `json:"success_artifact_bytes,omitempty"`
	NovelSemanticProbes         []string             `json:"novel_semantic_probes,omitempty"`
	NovelChoiceFeatures         []string             `json:"novel_choice_features,omitempty"`
}

type BatchJournal struct {
	ctx        context.Context
	config     BatchConfig
	path       string
	runsFile   *os.File
	runsHasher hash.Hash
	runsWriter io.Writer
	runsBytes  uint64
	published  bool
	resumeLock *filelock.Lock
}

type RunState string

const (
	RunStaging    RunState = "staging"
	RunStarting   RunState = "starting"
	RunExited     RunState = "exited"
	RunCaptured   RunState = "captured"
	RunClassified RunState = "classified"
)

type RunJournal struct {
	path    string
	ordinal uint64
	seed    uint64
	state   RunState
}

type BatchRecord struct {
	SchemaVersion                uint32                  `json:"schema_version"`
	Schema                       string                  `json:"schema"`
	RunID                        string                  `json:"run_id"`
	Strategy                     string                  `json:"strategy,omitempty"`
	Selection                    string                  `json:"selection"`
	SelectionCount               record.Uint64String     `json:"selection_count"`
	Attempted                    record.Uint64String     `json:"attempted"`
	Succeeded                    record.Uint64String     `json:"succeeded"`
	Failures                     record.Uint64String     `json:"failures"`
	Watchdogs                    record.Uint64String     `json:"watchdogs"`
	Cancelled                    record.Uint64String     `json:"cancelled"`
	DistinctFailures             record.Uint64String     `json:"distinct_failures"`
	RetainedSuccesses            record.Uint64String     `json:"retained_successes,omitempty"`
	RetainedSuccessBytes         record.Uint64String     `json:"retained_success_bytes,omitempty"`
	StopReason                   string                  `json:"stop_reason"`
	RunsSHA256                   record.SHA256           `json:"runs_sha256"`
	FailureSignatures            []record.SHA256         `json:"failure_signatures"`
	Frontier                     *choicefrontier.Summary `json:"frontier,omitempty"`
	FrontierImplementationSHA256 record.SHA256           `json:"frontier_implementation_sha256,omitempty"`
	FrontierChainSHA256          record.SHA256           `json:"frontier_chain_sha256,omitempty"`
	RecoveryExecutions           record.Uint64String     `json:"recovery_executions,omitempty"`
}

func NewBatchJournal(ctx context.Context, config BatchConfig) (*BatchJournal, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if config.Strategy == "" {
		config.Strategy = "seed"
	}
	if config.Strategy != "seed" && config.Strategy != "choice-frontier" {
		return nil, errors.New("batch journal strategy is invalid")
	}
	if config.Root == "" || config.RunID == "" || config.Selection == "" || config.SelectionCount == 0 {
		return nil, errors.New("batch journal root, run ID, selection, and selection count are required")
	}
	path := filepath.Join(config.Root, "v1", config.RunID)
	for _, directory := range []string{config.Root, filepath.Join(config.Root, "v1"), path, filepath.Join(path, "failures"), filepath.Join(path, "successes"), filepath.Join(path, ".partial"), filepath.Join(path, ".partial", "batch")} {
		if err := makePrivateDirectories(directory); err != nil {
			return nil, err
		}
	}
	journal := &BatchJournal{ctx: ctx, config: config, path: path}
	if err := journal.writeLifecycle(filepath.Join(path, ".partial", "batch"), "running", "", nil); err != nil {
		return nil, err
	}
	return journal, nil
}

func (journal *BatchJournal) Path() string {
	return journal.path
}

func (journal *BatchJournal) PreparedPath() string {
	return filepath.Join(journal.path, ".prepared")
}

func (journal *BatchJournal) FailuresPath() string {
	return filepath.Join(journal.path, "failures")
}

func (journal *BatchJournal) SuccessesPath() string {
	return filepath.Join(journal.path, "successes")
}

func (journal *BatchJournal) SetSelection(selection string, count uint64) error {
	if selection == "" || count == 0 {
		return errors.New("batch selection and count are required")
	}
	if journal.runsFile != nil || journal.published {
		return errors.New("batch selection cannot change after runs start")
	}
	journal.config.Selection = selection
	journal.config.SelectionCount = count
	return nil
}

func (journal *BatchJournal) BeginPreparation() error {
	if err := makePrivateDirectories(journal.PreparedPath()); err != nil {
		return err
	}
	partial := filepath.Join(journal.path, ".partial", "preparation")
	if err := makePrivateDirectories(partial); err != nil {
		return err
	}
	return journal.writeLifecycle(partial, "preparing", "", nil)
}

func (journal *BatchJournal) CompletePreparation() error {
	return removeCompletedPartial(filepath.Join(journal.path, ".partial", "preparation"))
}

func (journal *BatchJournal) FailPreparation(reason string, cause error) error {
	return journal.writeLifecycle(filepath.Join(journal.path, ".partial", "preparation"), "failed", reason, cause)
}

func (journal *BatchJournal) Fail(reason string, cause error) error {
	return journal.writeLifecycle(filepath.Join(journal.path, ".partial", "batch"), "failed", reason, cause)
}

func (journal *BatchJournal) StartRuns() error {
	if journal.runsFile != nil {
		return errors.New("batch runs journal is already open")
	}
	file, err := os.OpenFile(filepath.Join(journal.path, "runs.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		return errors.Join(err, file.Close())
	}
	journal.runsFile = file
	journal.runsHasher = sha256.New()
	journal.runsWriter = io.MultiWriter(file, journal.runsHasher)
	journal.runsBytes = 0
	return nil
}

func (journal *BatchJournal) AppendRun(run RunRecord) error {
	if journal.runsFile == nil {
		return errors.New("batch runs journal is not open")
	}
	encoded, err := record.CanonicalJSON(run)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if uint64(len(encoded)) > maximumRunsBytes-journal.runsBytes {
		return fmt.Errorf("batch runs journal capacity of %d bytes would be exceeded", maximumRunsBytes)
	}
	written, err := journal.runsWriter.Write(encoded)
	journal.runsBytes += uint64(written)
	if err != nil {
		return err
	}
	if written != len(encoded) {
		return io.ErrShortWrite
	}
	return journal.runsFile.Sync()
}

func (journal *BatchJournal) BeginRun(ordinal, seed uint64) (*RunJournal, error) {
	path := filepath.Join(journal.path, ".partial", fmt.Sprintf("%020d-%d", ordinal, seed))
	if err := makePrivateDirectories(path); err != nil {
		return nil, err
	}
	run := &RunJournal{path: path, ordinal: ordinal, seed: seed, state: RunStaging}
	if err := run.writeState(RunStaging); err != nil {
		return run, err
	}
	if err := makePrivateDirectories(run.WorkPath()); err != nil {
		return run, err
	}
	return run, nil
}

func (journal *BatchJournal) Publish(summary BatchSummary) error {
	if journal.published {
		return errors.New("batch journal is already published")
	}
	if journal.runsFile == nil {
		return errors.New("batch runs journal is not open")
	}
	if err := journal.ctx.Err(); err != nil {
		return err
	}
	if err := journal.runsFile.Sync(); err != nil {
		return err
	}
	if err := journal.runsFile.Close(); err != nil {
		return err
	}
	journal.runsFile = nil
	if err := journal.ctx.Err(); err != nil {
		return err
	}
	failureSignatures := append([]record.SHA256(nil), summary.FailureSignatures...)
	if failureSignatures == nil {
		failureSignatures = []record.SHA256{}
	}
	sort.Slice(failureSignatures, func(i, j int) bool { return failureSignatures[i] < failureSignatures[j] })
	batch := BatchRecord{
		SchemaVersion: record.SchemaVersion, Schema: "gomadv3.batch/v2", RunID: journal.config.RunID, Strategy: journal.config.Strategy, Selection: journal.config.Selection,
		SelectionCount: record.Uint64String(journal.config.SelectionCount), Attempted: record.Uint64String(summary.Attempted), Succeeded: record.Uint64String(summary.Succeeded),
		Failures: record.Uint64String(summary.Failures), Watchdogs: record.Uint64String(summary.Watchdogs), Cancelled: record.Uint64String(summary.Cancelled),
		DistinctFailures: record.Uint64String(summary.DistinctFailures), StopReason: summary.StopReason,
		RetainedSuccesses: record.Uint64String(summary.RetainedSuccesses), RetainedSuccessBytes: record.Uint64String(summary.RetainedSuccessBytes),
		RunsSHA256: record.SHA256("sha256:" + hex.EncodeToString(journal.runsHasher.Sum(nil))), FailureSignatures: failureSignatures,
		Frontier: summary.Frontier, FrontierImplementationSHA256: summary.FrontierImplementationSHA256, FrontierChainSHA256: summary.FrontierChainSHA256, RecoveryExecutions: record.Uint64String(summary.RecoveryExecutions),
	}
	encoded, err := record.CanonicalJSON(batch)
	if err != nil {
		return err
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(journal.path, "batch.json"), encoded); err != nil {
		return err
	}
	if err := syncDirectoryContext(journal.ctx, journal.path); err != nil {
		return err
	}
	if err := journal.ctx.Err(); err != nil {
		return err
	}
	if err := os.RemoveAll(journal.PreparedPath()); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(journal.path, ".partial", "batch")); err != nil {
		return err
	}
	journal.published = true
	return nil
}

func (journal *BatchJournal) Close() error {
	var result error
	if journal.runsFile != nil {
		result = journal.runsFile.Close()
		journal.runsFile = nil
	}
	if journal.resumeLock != nil {
		result = errors.Join(result, releaseResumeLock(journal.resumeLock))
		journal.resumeLock = nil
	}
	return result
}

func (journal *BatchJournal) writeLifecycle(directory, state, reason string, cause error) error {
	var reasonValue *string
	if reason != "" {
		reasonValue = &reason
	}
	var detail *string
	if cause != nil {
		message := cause.Error()
		detail = &message
	}
	payload := struct {
		SchemaVersion uint32  `json:"schema_version"`
		State         string  `json:"state"`
		Reason        *string `json:"reason"`
		Detail        *string `json:"detail"`
	}{SchemaVersion: record.SchemaVersion, State: state, Reason: reasonValue, Detail: detail}
	encoded, err := record.CanonicalJSON(payload)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(directory, "partial.json"), encoded)
}

func (run *RunJournal) Path() string {
	return run.path
}

func (run *RunJournal) WorkPath() string {
	return filepath.Join(run.path, "work")
}

func (run *RunJournal) CreateOutput(name string) (*os.File, error) {
	if run.state != RunStarting {
		return nil, fmt.Errorf("partial output requires %q state", RunStarting)
	}
	if name != "stdout" && name != "stderr" {
		return nil, fmt.Errorf("invalid partial output %q", name)
	}
	file, err := os.OpenFile(filepath.Join(run.path, name+".head"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file, nil
}

func (run *RunJournal) CloseOutput(name string, file *os.File) error {
	if file == nil {
		return fmt.Errorf("partial %s output is required", name)
	}
	var result error
	if err := file.Sync(); err != nil {
		result = fmt.Errorf("sync partial %s: %w", name, err)
	}
	if err := file.Close(); err != nil {
		result = errors.Join(result, fmt.Errorf("close partial %s: %w", name, err))
	}
	return result
}

func (run *RunJournal) Transition(next RunState) error {
	allowed := run.state == RunStaging && next == RunStarting ||
		run.state == RunStarting && next == RunExited ||
		run.state == RunExited && next == RunCaptured ||
		run.state == RunCaptured && next == RunClassified
	if !allowed {
		return fmt.Errorf("invalid run journal transition %q -> %q", run.state, next)
	}
	if err := run.writeState(next); err != nil {
		return err
	}
	run.state = next
	return nil
}

func (run *RunJournal) Preserve() error {
	return run.writeState("preserve-partial")
}

func (run *RunJournal) Complete() error {
	if run.state != RunClassified {
		return fmt.Errorf("cannot complete run journal in %q state", run.state)
	}
	return removeCompletedPartial(run.path)
}

func (run *RunJournal) writeState(state RunState) error {
	payload := struct {
		SchemaVersion    uint32              `json:"schema_version"`
		State            RunState            `json:"state"`
		SelectionOrdinal record.Uint64String `json:"selection_ordinal"`
		Seed             record.Uint64String `json:"seed"`
	}{SchemaVersion: record.SchemaVersion, State: state, SelectionOrdinal: record.Uint64String(run.ordinal), Seed: record.Uint64String(run.seed)}
	encoded, err := record.CanonicalJSON(payload)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(run.path, "partial.json"), encoded)
}

func atomicWrite(path string, data []byte) error {
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
