package campaignstore

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

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
	"go.temporal.io/server/tools/gomadv3/runner/internal/frontier"
)

type CampaignConfig struct {
	Root           string
	CampaignID     string
	Strategy       string
	Selection      string
	SelectionCount uint64
}

type CampaignSummary struct {
	Attempted                    uint64
	Succeeded                    uint64
	Failures                     uint64
	Watchdogs                    uint64
	Cancelled                    uint64
	DistinctFailures             uint64
	RetainedSuccesses            uint64
	RetainedSuccessBytes         uint64
	StopReason                   string
	FailureSignatures            []evidence.SHA256
	Frontier                     *frontier.Summary
	FrontierImplementationSHA256 evidence.SHA256
	FrontierChainSHA256          evidence.SHA256
	RecoveryExecutions           uint64
}

type ExecutionRecord struct {
	Strategy                    string                 `json:"strategy,omitempty"`
	Round                       *evidence.Uint64String `json:"round,omitempty"`
	CandidateSHA256             evidence.SHA256        `json:"candidate_sha256,omitempty"`
	ParentCandidateSHA256       evidence.SHA256        `json:"parent_candidate_sha256,omitempty"`
	PrefixSHA256                evidence.SHA256        `json:"prefix_sha256,omitempty"`
	ForcedDepth                 *evidence.Uint64String `json:"forced_depth,omitempty"`
	OutcomeSHA256               evidence.SHA256        `json:"outcome_sha256,omitempty"`
	SelectionOrdinal            evidence.Uint64String  `json:"selection_ordinal"`
	Seed                        evidence.Uint64String  `json:"seed"`
	Domain                      string                 `json:"domain"`
	Reason                      string                 `json:"reason"`
	Termination                 string                 `json:"termination"`
	FailureSignature            *evidence.SHA256       `json:"failure_signature"`
	Artifact                    *string                `json:"artifact"`
	ElapsedNanos                evidence.Uint64String  `json:"elapsed_nanos"`
	IOTranscriptSHA256          *evidence.SHA256       `json:"io_transcript_sha256"`
	IOTranscriptRecords         *evidence.Uint64String `json:"io_transcript_records"`
	ChoiceTraceSHA256           *evidence.SHA256       `json:"choice_trace_sha256,omitempty"`
	ChoiceTraceRecords          *evidence.Uint64String `json:"choice_trace_records,omitempty"`
	ChoiceTraceBranchingRecords *evidence.Uint64String `json:"choice_trace_branching_records,omitempty"`
	ChoiceTraceTerminalState    *string                `json:"choice_trace_terminal_state,omitempty"`
	ChoiceTapeSHA256            *evidence.SHA256       `json:"choice_tape_sha256,omitempty"`
	ChoiceDecisions             *evidence.Uint64String `json:"choice_decisions,omitempty"`
	SemanticProbes              []string               `json:"semantic_probes,omitempty"`
	ChoiceFeatures              []string               `json:"choice_features,omitempty"`
	SuccessArtifact             *string                `json:"success_artifact,omitempty"`
	SuccessArtifactBytes        *evidence.Uint64String `json:"success_artifact_bytes,omitempty"`
	NovelSemanticProbes         []string               `json:"novel_semantic_probes,omitempty"`
	NovelChoiceFeatures         []string               `json:"novel_choice_features,omitempty"`
}

type CampaignJournal struct {
	ctx        context.Context
	config     CampaignConfig
	path       string
	runsFile   *os.File
	runsHasher hash.Hash
	runsWriter io.Writer
	runsBytes  uint64
	published  bool
	resumeLock *hostfs.Lock
}

type ExecutionState string

const (
	ExecutionStaging    ExecutionState = "staging"
	ExecutionStarting   ExecutionState = "starting"
	ExecutionExited     ExecutionState = "exited"
	ExecutionCaptured   ExecutionState = "captured"
	ExecutionClassified ExecutionState = "classified"
)

type ExecutionJournal struct {
	path    string
	ordinal uint64
	seed    uint64
	state   ExecutionState
}

type CampaignRecord struct {
	SchemaVersion                uint32                `json:"schema_version"`
	Schema                       string                `json:"schema"`
	CampaignID                   string                `json:"run_id"`
	Strategy                     string                `json:"strategy,omitempty"`
	Selection                    string                `json:"selection"`
	SelectionCount               evidence.Uint64String `json:"selection_count"`
	Attempted                    evidence.Uint64String `json:"attempted"`
	Succeeded                    evidence.Uint64String `json:"succeeded"`
	Failures                     evidence.Uint64String `json:"failures"`
	Watchdogs                    evidence.Uint64String `json:"watchdogs"`
	Cancelled                    evidence.Uint64String `json:"cancelled"`
	DistinctFailures             evidence.Uint64String `json:"distinct_failures"`
	RetainedSuccesses            evidence.Uint64String `json:"retained_successes,omitempty"`
	RetainedSuccessBytes         evidence.Uint64String `json:"retained_success_bytes,omitempty"`
	StopReason                   string                `json:"stop_reason"`
	RunsSHA256                   evidence.SHA256       `json:"runs_sha256"`
	FailureSignatures            []evidence.SHA256     `json:"failure_signatures"`
	Frontier                     *frontier.Summary     `json:"frontier,omitempty"`
	FrontierImplementationSHA256 evidence.SHA256       `json:"frontier_implementation_sha256,omitempty"`
	FrontierChainSHA256          evidence.SHA256       `json:"frontier_chain_sha256,omitempty"`
	RecoveryExecutions           evidence.Uint64String `json:"recovery_executions,omitempty"`
}

func NewCampaignJournal(ctx context.Context, config CampaignConfig) (*CampaignJournal, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if config.Strategy == "" {
		config.Strategy = "seed"
	}
	if config.Strategy != "seed" && config.Strategy != "choice-frontier" {
		return nil, errors.New("batch journal strategy is invalid")
	}
	if config.Root == "" || config.CampaignID == "" || config.Selection == "" || config.SelectionCount == 0 {
		return nil, errors.New("batch journal root, run ID, selection, and selection count are required")
	}
	path := filepath.Join(config.Root, "v1", config.CampaignID)
	for _, directory := range []string{config.Root, filepath.Join(config.Root, "v1"), path, filepath.Join(path, "failures"), filepath.Join(path, "successes"), filepath.Join(path, ".partial"), filepath.Join(path, ".partial", "batch")} {
		if err := makePrivateDirectories(directory); err != nil {
			return nil, err
		}
	}
	journal := &CampaignJournal{ctx: ctx, config: config, path: path}
	if err := journal.writeLifecycle(filepath.Join(path, ".partial", "batch"), "running", "", nil); err != nil {
		return nil, err
	}
	return journal, nil
}

func (journal *CampaignJournal) Path() string {
	return journal.path
}

func (journal *CampaignJournal) PreparedPath() string {
	return filepath.Join(journal.path, ".prepared")
}

func (journal *CampaignJournal) FailuresPath() string {
	return filepath.Join(journal.path, "failures")
}

func (journal *CampaignJournal) SuccessesPath() string {
	return filepath.Join(journal.path, "successes")
}

func (journal *CampaignJournal) SetSelection(selection string, count uint64) error {
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

func (journal *CampaignJournal) BeginPreparation() error {
	if err := makePrivateDirectories(journal.PreparedPath()); err != nil {
		return err
	}
	partial := filepath.Join(journal.path, ".partial", "preparation")
	if err := makePrivateDirectories(partial); err != nil {
		return err
	}
	return journal.writeLifecycle(partial, "preparing", "", nil)
}

func (journal *CampaignJournal) CompletePreparation() error {
	return removeCompletedPartial(filepath.Join(journal.path, ".partial", "preparation"))
}

func (journal *CampaignJournal) FailPreparation(reason string, cause error) error {
	return journal.writeLifecycle(filepath.Join(journal.path, ".partial", "preparation"), "failed", reason, cause)
}

func (journal *CampaignJournal) Fail(reason string, cause error) error {
	return journal.writeLifecycle(filepath.Join(journal.path, ".partial", "batch"), "failed", reason, cause)
}

func (journal *CampaignJournal) StartExecutions() error {
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

func (journal *CampaignJournal) AppendExecution(run ExecutionRecord) error {
	if journal.runsFile == nil {
		return errors.New("batch runs journal is not open")
	}
	encoded, err := evidence.CanonicalJSON(run)
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

func (journal *CampaignJournal) BeginExecution(ordinal, seed uint64) (*ExecutionJournal, error) {
	path := filepath.Join(journal.path, ".partial", fmt.Sprintf("%020d-%d", ordinal, seed))
	if err := makePrivateDirectories(path); err != nil {
		return nil, err
	}
	run := &ExecutionJournal{path: path, ordinal: ordinal, seed: seed, state: ExecutionStaging}
	if err := run.writeState(ExecutionStaging); err != nil {
		return run, err
	}
	if err := makePrivateDirectories(run.WorkPath()); err != nil {
		return run, err
	}
	return run, nil
}

func (journal *CampaignJournal) Publish(summary CampaignSummary) error {
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
	failureSignatures := append([]evidence.SHA256(nil), summary.FailureSignatures...)
	if failureSignatures == nil {
		failureSignatures = []evidence.SHA256{}
	}
	sort.Slice(failureSignatures, func(i, j int) bool { return failureSignatures[i] < failureSignatures[j] })
	batch := CampaignRecord{
		SchemaVersion: evidence.SchemaVersion, Schema: "gomadv3.batch/v2", CampaignID: journal.config.CampaignID, Strategy: journal.config.Strategy, Selection: journal.config.Selection,
		SelectionCount: evidence.Uint64String(journal.config.SelectionCount), Attempted: evidence.Uint64String(summary.Attempted), Succeeded: evidence.Uint64String(summary.Succeeded),
		Failures: evidence.Uint64String(summary.Failures), Watchdogs: evidence.Uint64String(summary.Watchdogs), Cancelled: evidence.Uint64String(summary.Cancelled),
		DistinctFailures: evidence.Uint64String(summary.DistinctFailures), StopReason: summary.StopReason,
		RetainedSuccesses: evidence.Uint64String(summary.RetainedSuccesses), RetainedSuccessBytes: evidence.Uint64String(summary.RetainedSuccessBytes),
		RunsSHA256: evidence.SHA256("sha256:" + hex.EncodeToString(journal.runsHasher.Sum(nil))), FailureSignatures: failureSignatures,
		Frontier: summary.Frontier, FrontierImplementationSHA256: summary.FrontierImplementationSHA256, FrontierChainSHA256: summary.FrontierChainSHA256, RecoveryExecutions: evidence.Uint64String(summary.RecoveryExecutions),
	}
	encoded, err := evidence.CanonicalJSON(batch)
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

func (journal *CampaignJournal) Close() error {
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

func (journal *CampaignJournal) writeLifecycle(directory, state, reason string, cause error) error {
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
	}{SchemaVersion: evidence.SchemaVersion, State: state, Reason: reasonValue, Detail: detail}
	encoded, err := evidence.CanonicalJSON(payload)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(directory, "partial.json"), encoded)
}

func (run *ExecutionJournal) Path() string {
	return run.path
}

func (run *ExecutionJournal) WorkPath() string {
	return filepath.Join(run.path, "work")
}

func (run *ExecutionJournal) CreateOutput(name string) (*os.File, error) {
	if run.state != ExecutionStarting {
		return nil, fmt.Errorf("partial output requires %q state", ExecutionStarting)
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

func (run *ExecutionJournal) CloseOutput(name string, file *os.File) error {
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

func (run *ExecutionJournal) Transition(next ExecutionState) error {
	allowed := run.state == ExecutionStaging && next == ExecutionStarting ||
		run.state == ExecutionStarting && next == ExecutionExited ||
		run.state == ExecutionExited && next == ExecutionCaptured ||
		run.state == ExecutionCaptured && next == ExecutionClassified
	if !allowed {
		return fmt.Errorf("invalid run journal transition %q -> %q", run.state, next)
	}
	if err := run.writeState(next); err != nil {
		return err
	}
	run.state = next
	return nil
}

func (run *ExecutionJournal) Preserve() error {
	return run.writeState("preserve-partial")
}

func (run *ExecutionJournal) Complete() error {
	if run.state != ExecutionClassified {
		return fmt.Errorf("cannot complete run journal in %q state", run.state)
	}
	return removeCompletedPartial(run.path)
}

func (run *ExecutionJournal) writeState(state ExecutionState) error {
	payload := struct {
		SchemaVersion    uint32                `json:"schema_version"`
		State            ExecutionState        `json:"state"`
		SelectionOrdinal evidence.Uint64String `json:"selection_ordinal"`
		Seed             evidence.Uint64String `json:"seed"`
	}{SchemaVersion: evidence.SchemaVersion, State: state, SelectionOrdinal: evidence.Uint64String(run.ordinal), Seed: evidence.Uint64String(run.seed)}
	encoded, err := evidence.CanonicalJSON(payload)
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
