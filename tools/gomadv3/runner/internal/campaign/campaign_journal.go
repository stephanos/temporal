package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
	"go.temporal.io/server/tools/gomadv3/record"
	choiceengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/choice"
	simulationengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulation"
)

type CampaignConfig struct {
	Root           string
	CampaignID     string
	PlanSHA256     record.SHA256
	Shard          *CampaignShard
	Strategy       string
	Selection      string
	SelectionCount uint64
	MaxExecutions  uint64
	Parallel       uint64
	Journal        ExecutionJournalLimits
}

type CampaignSummary struct {
	Attempted                                 uint64
	Succeeded                                 uint64
	Failures                                  uint64
	Watchdogs                                 uint64
	Cancelled                                 uint64
	DistinctFailures                          uint64
	RetainedSuccesses                         uint64
	RetainedSuccessBytes                      uint64
	StopReason                                string
	FailureSignatures                         []record.SHA256
	ChoiceExploration                         *choiceengine.Summary
	ChoiceExplorationImplementationSHA256     record.SHA256
	ChoiceExplorationChainSHA256              record.SHA256
	SimulationExploration                     *simulationengine.Summary
	SimulationExplorationImplementationSHA256 record.SHA256
	SimulationExplorationChainSHA256          record.SHA256
	RecoveryExecutions                        uint64
}

type ExecutionRecord struct {
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

type CampaignJournal struct {
	ctx                 context.Context
	config              CampaignConfig
	path                string
	segmentedRuns       *segmentedExecutionJournal
	published           bool
	resumeLock          *hostfs.Lock
	lifecycle           LifecycleState
	lastStableLifecycle LifecycleState
	artifactPlan        *ArtifactCapacityPlan
	partialMu           sync.Mutex
	partialRuns         uint64
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
	ctx            context.Context
	path           string
	ordinal        uint64
	seed           uint64
	state          ExecutionState
	releasePartial func()
}

type CampaignRecord struct {
	SchemaVersion                             uint32                     `json:"schema_version"`
	Schema                                    string                     `json:"schema"`
	CampaignID                                string                     `json:"campaign_id"`
	PlanSHA256                                record.SHA256              `json:"plan_sha256,omitempty"`
	Shard                                     *CampaignShard             `json:"shard,omitempty"`
	Strategy                                  string                     `json:"strategy"`
	Selection                                 string                     `json:"selection"`
	SelectionCount                            record.Uint64String        `json:"selection_count"`
	Attempted                                 record.Uint64String        `json:"attempted"`
	Succeeded                                 record.Uint64String        `json:"succeeded"`
	Failures                                  record.Uint64String        `json:"failures"`
	Watchdogs                                 record.Uint64String        `json:"watchdogs"`
	Cancelled                                 record.Uint64String        `json:"cancelled"`
	DistinctFailures                          record.Uint64String        `json:"distinct_failures"`
	RetainedSuccesses                         record.Uint64String        `json:"retained_successes,omitempty"`
	RetainedSuccessBytes                      record.Uint64String        `json:"retained_success_bytes,omitempty"`
	StopReason                                string                     `json:"stop_reason"`
	Journal                                   *ExecutionJournalReference `json:"journal,omitempty"`
	Artifacts                                 *ArtifactCapacityPlan      `json:"artifacts,omitempty"`
	FailureSignatures                         []record.SHA256            `json:"failure_signatures"`
	ChoiceExploration                         *choiceengine.Summary      `json:"choice_exploration,omitempty"`
	ChoiceExplorationImplementationSHA256     record.SHA256              `json:"choice_exploration_implementation_sha256,omitempty"`
	ChoiceExplorationChainSHA256              record.SHA256              `json:"choice_exploration_chain_sha256,omitempty"`
	SimulationExploration                     *simulationengine.Summary  `json:"simulation_exploration,omitempty"`
	SimulationExplorationImplementationSHA256 record.SHA256              `json:"simulation_exploration_implementation_sha256,omitempty"`
	SimulationExplorationChainSHA256          record.SHA256              `json:"simulation_exploration_chain_sha256,omitempty"`
	RecoveryExecutions                        record.Uint64String        `json:"recovery_executions,omitempty"`
}

func NewCampaignJournal(ctx context.Context, config CampaignConfig) (_ *CampaignJournal, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if config.Strategy == "" {
		config.Strategy = "seed"
	}
	if config.Strategy != "seed" && config.Strategy != "choice-exploration" && config.Strategy != "simulation-exploration" {
		return nil, errors.New("campaign journal strategy is invalid")
	}
	if config.Root == "" || config.CampaignID == "" || config.Selection == "" || config.SelectionCount == 0 {
		return nil, errors.New("campaign journal root, campaign ID, selection, and selection count are required")
	}
	if (config.PlanSHA256 == "") != (config.Shard == nil) {
		return nil, errors.New("campaign journal portable plan identity is incomplete")
	}
	if config.Shard != nil && (!validRecordSHA256(config.PlanSHA256) || config.Shard.Count == 0 || config.Shard.Index >= config.Shard.Count || config.Strategy != "seed") {
		return nil, errors.New("campaign journal shard identity is invalid")
	}
	limits, err := normalizeExecutionJournalLimits(config)
	if err != nil {
		return nil, err
	}
	config.Journal = limits
	if config.Shard != nil {
		shard := *config.Shard
		config.Shard = &shard
	}
	path := filepath.Join(config.Root, "v1", config.CampaignID)
	for _, directory := range []string{config.Root, filepath.Join(config.Root, "v1")} {
		if err := makePrivateDirectoriesContext(ctx, directory); err != nil {
			return nil, err
		}
	}
	if err := observeMutation(ctx, mutationCreate, "campaign-directory"); err != nil {
		return nil, err
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return nil, fmt.Errorf("create campaign journal directory: %w", err)
	}
	defer func() {
		if retErr == nil {
			return
		}
		retErr = errors.Join(retErr, removeCompletedPartialContext(ctx, path, "campaign-directory"))
		retErr = errors.Join(retErr, syncDirectoryContext(ctx, filepath.Dir(path)))
	}()
	if err := os.Chmod(path, 0o700); err != nil {
		return nil, fmt.Errorf("set campaign journal directory mode: %w", err)
	}
	for _, directory := range []string{filepath.Join(path, "failures"), filepath.Join(path, "successes"), filepath.Join(path, ".partial"), filepath.Join(path, ".partial", "campaign")} {
		if err := makePrivateDirectoriesContext(ctx, directory); err != nil {
			return nil, err
		}
	}
	journal := &CampaignJournal{ctx: ctx, config: config, path: path}
	if err := journal.transitionLifecycle(LifecyclePlanned, "", nil); err != nil {
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

func (journal *CampaignJournal) ExecutionJournalPlan() ExecutionJournalPlan {
	return recordExecutionJournalLimits(journal.config.Journal)
}

func (journal *CampaignJournal) SetSelection(selection string, count uint64) error {
	if selection == "" || count == 0 {
		return errors.New("campaign selection and count are required")
	}
	if journal.segmentedRuns != nil || journal.published {
		return errors.New("campaign selection cannot change after executions start")
	}
	journal.config.Selection = selection
	journal.config.SelectionCount = count
	return nil
}

func (journal *CampaignJournal) BeginPreparation() error {
	if err := makePrivateDirectoriesContext(journal.ctx, journal.PreparedPath()); err != nil {
		return err
	}
	partial := filepath.Join(journal.path, ".partial", "preparation")
	if err := makePrivateDirectoriesContext(journal.ctx, partial); err != nil {
		return err
	}
	return journal.writeLifecycle(partial, "preparing", "", nil)
}

func (journal *CampaignJournal) CompletePreparation() error {
	if err := removeCompletedPartialContext(journal.ctx, filepath.Join(journal.path, ".partial", "preparation"), "preparation"); err != nil {
		return err
	}
	return journal.transitionLifecycle(LifecyclePrepared, "", nil)
}

func (journal *CampaignJournal) FailPreparation(reason string, cause error) error {
	return journal.writeLifecycleContext(context.WithoutCancel(journal.ctx), filepath.Join(journal.path, ".partial", "preparation"), "failed", reason, cause)
}

func (journal *CampaignJournal) Fail(reason string, cause error) error {
	return journal.transitionLifecycleContext(context.WithoutCancel(journal.ctx), LifecycleRecoverableFailure, reason, cause)
}

func (journal *CampaignJournal) StartExecutions() error {
	if journal.segmentedRuns != nil {
		return errors.New("campaign execution journal is already open")
	}
	if journal.lifecycle == LifecyclePlanned {
		if err := journal.transitionLifecycle(LifecyclePrepared, "", nil); err != nil {
			return err
		}
	}
	segmented, err := newSegmentedExecutionJournal(journal.ctx, journal.path, journal.config.Journal)
	if err != nil {
		return err
	}
	if err := journal.transitionLifecycle(LifecycleRunning, "", nil); err != nil {
		return errors.Join(err, segmented.close())
	}
	journal.segmentedRuns = segmented
	return nil
}

func (journal *CampaignJournal) AppendExecution(run ExecutionRecord) error {
	if journal.segmentedRuns == nil {
		return errors.New("campaign execution journal is not open")
	}
	encoded, err := canonicaljson.CanonicalJSON(run)
	if err != nil {
		return err
	}
	return journal.segmentedRuns.append(append(encoded, '\n'))
}

func (journal *CampaignJournal) BeginExecution(ordinal, seed uint64) (_ *ExecutionJournal, retErr error) {
	journal.partialMu.Lock()
	if journal.partialRuns == journal.config.Journal.MaximumPartialExecutions {
		journal.partialMu.Unlock()
		return nil, &JournalCapacityError{
			Limit: JournalLimitPartialExecutions, Required: journal.partialRuns + 1,
			Maximum: journal.config.Journal.MaximumPartialExecutions, Outcome: CapacityInfrastructureFailure,
		}
	}
	journal.partialRuns++
	journal.partialMu.Unlock()
	release := func() {
		journal.partialMu.Lock()
		journal.partialRuns--
		journal.partialMu.Unlock()
	}
	defer func() {
		if retErr != nil {
			release()
		}
	}()
	path := filepath.Join(journal.path, ".partial", fmt.Sprintf("%020d-%d", ordinal, seed))
	if err := makePrivateDirectoriesContext(journal.ctx, path); err != nil {
		return nil, err
	}
	run := &ExecutionJournal{ctx: journal.ctx, path: path, ordinal: ordinal, seed: seed, state: ExecutionStaging, releasePartial: release}
	if err := run.writeState(ExecutionStaging); err != nil {
		return run, err
	}
	if err := makePrivateDirectoriesContext(journal.ctx, run.WorkPath()); err != nil {
		return run, err
	}
	return run, nil
}

func (journal *CampaignJournal) Publish(summary CampaignSummary) error {
	if journal.published {
		return errors.New("campaign journal is already published")
	}
	if journal.segmentedRuns == nil {
		return errors.New("campaign execution journal is not open")
	}
	if err := journal.ctx.Err(); err != nil {
		return err
	}
	if err := journal.transitionLifecycle(LifecycleCommitting, "", nil); err != nil {
		return err
	}
	reference, err := journal.segmentedRuns.reference()
	if err != nil {
		return err
	}
	journalReference := &reference
	if err := journal.ctx.Err(); err != nil {
		return err
	}
	failureSignatures := append([]record.SHA256(nil), summary.FailureSignatures...)
	if failureSignatures == nil {
		failureSignatures = []record.SHA256{}
	}
	sort.Slice(failureSignatures, func(i, j int) bool { return failureSignatures[i] < failureSignatures[j] })
	batch := CampaignRecord{
		SchemaVersion: record.SchemaVersion, Schema: "gomadv3.campaign/v1", CampaignID: journal.config.CampaignID, Strategy: journal.config.Strategy, Selection: journal.config.Selection,
		PlanSHA256: journal.config.PlanSHA256, Shard: cloneCampaignShard(journal.config.Shard),
		SelectionCount: record.Uint64String(journal.config.SelectionCount), Attempted: record.Uint64String(summary.Attempted), Succeeded: record.Uint64String(summary.Succeeded),
		Failures: record.Uint64String(summary.Failures), Watchdogs: record.Uint64String(summary.Watchdogs), Cancelled: record.Uint64String(summary.Cancelled),
		DistinctFailures: record.Uint64String(summary.DistinctFailures), StopReason: summary.StopReason,
		RetainedSuccesses: record.Uint64String(summary.RetainedSuccesses), RetainedSuccessBytes: record.Uint64String(summary.RetainedSuccessBytes),
		Journal: journalReference, Artifacts: journal.artifactPlan, FailureSignatures: failureSignatures,
		ChoiceExploration: summary.ChoiceExploration, ChoiceExplorationImplementationSHA256: summary.ChoiceExplorationImplementationSHA256, ChoiceExplorationChainSHA256: summary.ChoiceExplorationChainSHA256, RecoveryExecutions: record.Uint64String(summary.RecoveryExecutions),
		SimulationExploration: summary.SimulationExploration, SimulationExplorationImplementationSHA256: summary.SimulationExplorationImplementationSHA256, SimulationExplorationChainSHA256: summary.SimulationExplorationChainSHA256,
	}
	encoded, err := canonicaljson.CanonicalJSON(batch)
	if err != nil {
		return err
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(journal.path, "campaign.json"), encoded); err != nil {
		return err
	}
	if err := syncDirectoryContext(journal.ctx, journal.path); err != nil {
		return err
	}
	if err := journal.ctx.Err(); err != nil {
		return err
	}
	if err := removeCompletedPartialContext(journal.ctx, journal.PreparedPath(), "prepared-target"); err != nil {
		return err
	}
	if journal.segmentedRuns != nil {
		if err := removeCompletedPartialContext(journal.ctx, journal.segmentedRuns.partialPath(), "execution-journal"); err != nil {
			return err
		}
	}
	if err := removeCompletedPartialContext(journal.ctx, filepath.Join(journal.path, ".partial", "campaign"), "campaign-lifecycle"); err != nil {
		return err
	}
	journal.published = true
	return nil
}

func cloneCampaignShard(shard *CampaignShard) *CampaignShard {
	if shard == nil {
		return nil
	}
	copy := *shard
	return &copy
}

func (journal *CampaignJournal) Close() error {
	var result error
	if journal.segmentedRuns != nil {
		result = journal.segmentedRuns.close()
	}
	if journal.resumeLock != nil {
		result = errors.Join(result, releaseResumeLock(journal.resumeLock))
		journal.resumeLock = nil
	}
	return result
}

func (journal *CampaignJournal) writeLifecycle(directory, state, reason string, cause error) error {
	return journal.writeLifecycleContext(journal.ctx, directory, state, reason, cause)
}

func (journal *CampaignJournal) writeLifecycleContext(ctx context.Context, directory, state, reason string, cause error) error {
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
	encoded, err := canonicaljson.CanonicalJSON(payload)
	if err != nil {
		return err
	}
	return atomicWriteContext(ctx, filepath.Join(directory, "partial.json"), encoded)
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
	if err := observeMutation(run.ctx, mutationCreate, "execution-output"); err != nil {
		return nil, err
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
	if err := syncFileContext(run.ctx, file, "execution-output"); err != nil {
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
		return fmt.Errorf("invalid execution journal transition %q -> %q", run.state, next)
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
		return fmt.Errorf("cannot complete execution journal in %q state", run.state)
	}
	if err := removeCompletedPartialContext(run.ctx, run.path, "execution-partial"); err != nil {
		return err
	}
	if run.releasePartial != nil {
		run.releasePartial()
		run.releasePartial = nil
	}
	return nil
}

func (run *ExecutionJournal) writeState(state ExecutionState) error {
	payload := struct {
		SchemaVersion    uint32              `json:"schema_version"`
		State            ExecutionState      `json:"state"`
		SelectionOrdinal record.Uint64String `json:"selection_ordinal"`
		Seed             record.Uint64String `json:"seed"`
	}{SchemaVersion: record.SchemaVersion, State: state, SelectionOrdinal: record.Uint64String(run.ordinal), Seed: record.Uint64String(run.seed)}
	encoded, err := canonicaljson.CanonicalJSON(payload)
	if err != nil {
		return err
	}
	return atomicWriteContext(run.ctx, filepath.Join(run.path, "partial.json"), encoded)
}

func atomicWrite(path string, data []byte) error {
	return atomicWriteContext(context.Background(), path, data)
}

func atomicWriteContext(ctx context.Context, path string, data []byte) (retErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err := observeMutation(ctx, mutationCreate, "atomic-temporary"); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".tmp-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	keep := true
	defer func() {
		if keep {
			if removeErr := observeMutation(ctx, mutationDelete, "atomic-temporary"); removeErr != nil {
				retErr = errors.Join(retErr, removeErr)
			} else if removeErr := os.Remove(temporaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
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
	if err := syncFileContext(ctx, temporary, "atomic-temporary"); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := renameContext(ctx, temporaryPath, path, "atomic-publish"); err != nil {
		return err
	}
	keep = false
	return syncDirectoryContext(ctx, directory)
}

func removeCompletedPartial(path string) error {
	return removeCompletedPartialContext(context.Background(), path, "completed-partial")
}

func removeCompletedPartialContext(ctx context.Context, path, operation string) error {
	if path == "" {
		return nil
	}
	if err := observeMutation(ctx, mutationDelete, operation); err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove completed partial %s: %w", filepath.Base(path), err)
	}
	return nil
}

func makePrivateDirectories(path string) error {
	return makePrivateDirectoriesContext(context.Background(), path)
}

func makePrivateDirectoriesContext(ctx context.Context, path string) error {
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
		if err := observeMutation(ctx, mutationCreate, "private-directory"); err != nil {
			return err
		}
		if err := os.Mkdir(missing[index], 0o700); err != nil {
			return err
		}
		if err := os.Chmod(missing[index], 0o700); err != nil {
			return err
		}
	}
	return os.Chmod(path, 0o700)
}
