package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/exploration"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

type CampaignConfig struct {
	Root           string
	CampaignID     string
	PlanSHA256     record.SHA256
	Shard          *CampaignShard
	Strategy       string
	Selection      string
	SelectionCount uint64
	Guidance       *GuidancePlan
	MaxExecutions  uint64
	Parallel       uint64
	Journal        executionJournalLimits
}

type CampaignSummary struct {
	Attempted                                 uint64
	Succeeded                                 uint64
	Failures                                  uint64
	ReplayDivergences                         uint64
	Watchdogs                                 uint64
	Cancelled                                 uint64
	DistinctFailures                          uint64
	RetainedSuccesses                         uint64
	RetainedSuccessBytes                      uint64
	StopReason                                string
	FailureSignatures                         []record.SHA256
	ChoiceExploration                         *exploration.ChoiceSummary
	ChoiceExplorationImplementationSHA256     record.SHA256
	ChoiceExplorationChainSHA256              record.SHA256
	SimulationExploration                     *exploration.SimulationSummary
	SimulationExplorationImplementationSHA256 record.SHA256
	SimulationExplorationChainSHA256          record.SHA256
	RecoveryExecutions                        uint64
}

type ExecutionRecord struct {
	Strategy                    string                     `json:"strategy,omitempty"`
	Round                       *record.Uint64String       `json:"round,omitempty"`
	CandidateSHA256             record.SHA256              `json:"candidate_sha256,omitempty"`
	ParentCandidateSHA256       record.SHA256              `json:"parent_candidate_sha256,omitempty"`
	PrefixSHA256                record.SHA256              `json:"prefix_sha256,omitempty"`
	ForcedDepth                 *record.Uint64String       `json:"forced_depth,omitempty"`
	OutcomeSHA256               record.SHA256              `json:"outcome_sha256,omitempty"`
	Divergence                  *choice.DivergenceEvidence `json:"divergence,omitempty"`
	SelectionOrdinal            record.Uint64String        `json:"selection_ordinal"`
	Seed                        record.Uint64String        `json:"seed"`
	Domain                      string                     `json:"domain"`
	Reason                      string                     `json:"reason"`
	Termination                 string                     `json:"termination"`
	FailureSignature            *record.SHA256             `json:"failure_signature"`
	Artifact                    *string                    `json:"artifact"`
	ElapsedNanos                record.Uint64String        `json:"elapsed_nanos"`
	IOTranscriptSHA256          *record.SHA256             `json:"io_transcript_sha256"`
	IOTranscriptRecords         *record.Uint64String       `json:"io_transcript_records"`
	ChoiceTraceSHA256           *record.SHA256             `json:"choice_trace_sha256,omitempty"`
	ChoiceTraceRecords          *record.Uint64String       `json:"choice_trace_records,omitempty"`
	ChoiceTraceBranchingRecords *record.Uint64String       `json:"choice_trace_branching_records,omitempty"`
	ChoiceTraceTerminalState    *string                    `json:"choice_trace_terminal_state,omitempty"`
	ChoiceTapeSHA256            *record.SHA256             `json:"choice_tape_sha256,omitempty"`
	ChoiceDecisions             *record.Uint64String       `json:"choice_decisions,omitempty"`
	SemanticProbes              []string                   `json:"semantic_probes,omitempty"`
	ChoiceFeatures              []string                   `json:"choice_features,omitempty"`
	SuccessArtifact             *string                    `json:"success_artifact,omitempty"`
	SuccessArtifactBytes        *record.Uint64String       `json:"success_artifact_bytes,omitempty"`
	NovelSemanticProbes         []string                   `json:"novel_semantic_probes,omitempty"`
	NovelChoiceFeatures         []string                   `json:"novel_choice_features,omitempty"`
}

type CampaignRecord struct {
	Guidance                                  *GuidancePlan                  `json:"guidance,omitempty"`
	SchemaVersion                             uint32                         `json:"schema_version"`
	Schema                                    string                         `json:"schema"`
	CampaignID                                string                         `json:"campaign_id"`
	PlanSHA256                                record.SHA256                  `json:"plan_sha256,omitempty"`
	Shard                                     *CampaignShard                 `json:"shard,omitempty"`
	Strategy                                  string                         `json:"strategy"`
	Selection                                 string                         `json:"selection"`
	SelectionCount                            record.Uint64String            `json:"selection_count"`
	Attempted                                 record.Uint64String            `json:"attempted"`
	Succeeded                                 record.Uint64String            `json:"succeeded"`
	Failures                                  record.Uint64String            `json:"failures"`
	ReplayDivergences                         record.Uint64String            `json:"replay_divergences,omitempty"`
	Watchdogs                                 record.Uint64String            `json:"watchdogs"`
	Cancelled                                 record.Uint64String            `json:"cancelled"`
	DistinctFailures                          record.Uint64String            `json:"distinct_failures"`
	RetainedSuccesses                         record.Uint64String            `json:"retained_successes,omitempty"`
	RetainedSuccessBytes                      record.Uint64String            `json:"retained_success_bytes,omitempty"`
	StopReason                                string                         `json:"stop_reason"`
	Journal                                   *executionJournalReference     `json:"journal,omitempty"`
	Artifacts                                 *ArtifactCapacityPlan          `json:"artifacts,omitempty"`
	FailureSignatures                         []record.SHA256                `json:"failure_signatures"`
	ChoiceExploration                         *exploration.ChoiceSummary     `json:"choice_exploration,omitempty"`
	ChoiceExplorationImplementationSHA256     record.SHA256                  `json:"choice_exploration_implementation_sha256,omitempty"`
	ChoiceExplorationChainSHA256              record.SHA256                  `json:"choice_exploration_chain_sha256,omitempty"`
	SimulationExploration                     *exploration.SimulationSummary `json:"simulation_exploration,omitempty"`
	SimulationExplorationImplementationSHA256 record.SHA256                  `json:"simulation_exploration_implementation_sha256,omitempty"`
	SimulationExplorationChainSHA256          record.SHA256                  `json:"simulation_exploration_chain_sha256,omitempty"`
	RecoveryExecutions                        record.Uint64String            `json:"recovery_executions,omitempty"`
}

// CampaignJournal is the exclusive writer of one campaign directory. It holds
// the campaign log's lock from creation or resume until Close, and advances
// its replayed state only after each record is durable.
type CampaignJournal struct {
	ctx         context.Context
	config      CampaignConfig
	path        string
	mu          sync.Mutex
	writer      *logstore.Writer
	state       *campaignState
	started     bool
	partialMu   sync.Mutex
	partialRuns uint64
}

type executionState string

const (
	executionStaging    executionState = "staging"
	ExecutionStarting   executionState = "starting"
	ExecutionExited     executionState = "exited"
	ExecutionCaptured   executionState = "captured"
	ExecutionClassified executionState = "classified"
)

// ExecutionJournal is one attempt's ephemeral work directory. Its state lives
// only in memory: the campaign log, not the directory, records outcomes, so
// an interrupted attempt's directory is reclaimable work.
type ExecutionJournal struct {
	ctx            context.Context
	path           string
	ordinal        uint64
	seed           uint64
	state          executionState
	stateFile      bool
	releasePartial func()
}

func NewCampaignJournal(ctx context.Context, config CampaignConfig) (_ *CampaignJournal, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if config.Strategy == "" {
		config.Strategy = "seed"
	}
	if config.Root == "" {
		return nil, errors.New("campaign journal root, campaign ID, selection, and selection count are required")
	}
	if err := validateCampaignConfig(config); err != nil {
		return nil, err
	}
	limits, err := normalizeExecutionJournalLimits(config)
	if err != nil {
		return nil, err
	}
	config.Journal = limits
	config.Shard = cloneCampaignShard(config.Shard)
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
	journal := &CampaignJournal{ctx: ctx, config: config, path: path, state: newCampaignState()}
	defer func() {
		if retErr == nil {
			return
		}
		if journal.writer != nil {
			retErr = errors.Join(retErr, journal.writer.Close())
		}
		retErr = errors.Join(retErr, removeCompletedPartialContext(ctx, path, "campaign-directory"))
		retErr = errors.Join(retErr, syncDirectoryContext(ctx, filepath.Dir(path)))
	}()
	if err := os.Chmod(path, 0o700); err != nil {
		return nil, fmt.Errorf("set campaign journal directory mode: %w", err)
	}
	// The store directory is created here with its private mode, so the log
	// store pins a 0700 directory even under a restrictive umask.
	for _, directory := range []string{journal.FailuresPath(), journal.SuccessesPath(), filepath.Join(path, ".partial"), filepath.Join(path, storeDirectory)} {
		if err := makePrivateDirectoriesContext(ctx, directory); err != nil {
			return nil, err
		}
	}
	writer, _, err := logstore.OpenWriter(filepath.Join(path, storeDirectory), campaignLogOptions(logStoreHook(ctx), true), journal.state.consume)
	if err != nil {
		return nil, fmt.Errorf("create campaign log: %w", err)
	}
	journal.writer = writer
	if err := syncDirectoryContext(ctx, path); err != nil {
		return nil, err
	}
	open := campaignOpenRecord{
		Format: CampaignLogFormat, CampaignID: config.CampaignID, Strategy: config.Strategy, PlanSHA256: config.PlanSHA256, Shard: cloneCampaignShard(config.Shard),
		Selection: config.Selection, SelectionCount: record.Uint64String(config.SelectionCount), Guidance: config.Guidance,
		MaxExecutions: record.Uint64String(config.MaxExecutions), Parallel: record.Uint64String(config.Parallel),
	}
	if err := journal.append(kindCampaignOpen, open); err != nil {
		return nil, err
	}
	return journal, nil
}

// append commits one record: it validates the record against the replayed
// state, appends and syncs it, and applies it only once it is durable. A
// failed append poisons the writer; the state stays at the last
// acknowledged record.
func (journal *CampaignJournal) append(kind logstore.Kind, value any) error {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	return journal.appendLocked(kind, value)
}

func (journal *CampaignJournal) appendLocked(kind logstore.Kind, value any) error {
	if journal.writer == nil {
		return errors.New("campaign log is closed")
	}
	payload, err := logstore.EncodeJSON(value)
	if err != nil {
		return err
	}
	apply, err := journal.state.prepare(kind, payload)
	if err != nil {
		return err
	}
	if err := journal.writer.Append(kind, payload); err != nil {
		return fmt.Errorf("append campaign log record: %w", err)
	}
	apply()
	return nil
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

// SetSelection replaces the selection before the plan freezes it. Guided
// campaigns choose their seeds after preparing the target; the plan record
// carries the final selection.
func (journal *CampaignJournal) SetSelection(selection string, count uint64, guidance *GuidancePlan) error {
	if (selection == "") != (count == 0) || count == 0 && !isFullyAnsweredGuidance(guidance) {
		return errors.New("campaign selection and count are required")
	}
	journal.mu.Lock()
	defer journal.mu.Unlock()
	if journal.started || journal.state.plan != nil || journal.state.published != nil {
		return errors.New("campaign selection cannot change after executions start")
	}
	limits, err := normalizeExecutionJournalLimits(CampaignConfig{Strategy: journal.config.Strategy, SelectionCount: count, MaxExecutions: journal.config.MaxExecutions, Parallel: journal.config.Parallel})
	if err != nil {
		return err
	}
	journal.config.Journal = limits
	journal.config.Guidance = guidance
	journal.config.Selection = selection
	journal.config.SelectionCount = count
	return nil
}

// BeginPreparation creates the prepared-target payload directory. Until the
// plan is recorded and preparation completes, the campaign stays planned and
// cannot be resumed.
func (journal *CampaignJournal) BeginPreparation() error {
	return makePrivateDirectoriesContext(journal.ctx, journal.PreparedPath())
}

func (journal *CampaignJournal) CompletePreparation() error {
	return journal.transition(journal.ctx, lifecyclePrepared, "", nil)
}

// FailPreparation records why preparation failed. It is recorded even when
// the campaign's context has ended.
func (journal *CampaignJournal) FailPreparation(reason string, cause error) error {
	return journal.transition(context.WithoutCancel(journal.ctx), lifecycleRecoverableFailure, reason, cause)
}

// Fail records a recoverable failure with its reason. It is recorded even
// when the campaign's context has ended.
func (journal *CampaignJournal) Fail(reason string, cause error) error {
	return journal.transition(context.WithoutCancel(journal.ctx), lifecycleRecoverableFailure, reason, cause)
}

func (journal *CampaignJournal) transition(ctx context.Context, next lifecycleState, reason string, cause error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	transition := lifecycleRecord{State: next, Reason: reason}
	if next == lifecycleRecoverableFailure && cause != nil {
		detail := cause.Error()
		if len(detail) > maximumLifecycleText {
			detail = detail[:maximumLifecycleText]
			for !utf8.ValidString(detail) {
				detail = detail[:len(detail)-1]
			}
		}
		transition.Detail = detail
	}
	if err := journal.append(kindLifecycle, transition); err != nil {
		return fmt.Errorf("publish campaign lifecycle: %w", err)
	}
	return nil
}

func (journal *CampaignJournal) StartExecutions() error {
	if journal.started {
		return errors.New("campaign execution journal is already open")
	}
	if journal.state.lifecycle == lifecyclePlanned {
		if err := journal.transition(journal.ctx, lifecyclePrepared, "", nil); err != nil {
			return err
		}
	}
	if err := journal.transition(journal.ctx, lifecycleRunning, "", nil); err != nil {
		return err
	}
	journal.started = true
	return nil
}

// AppendExecution commits one execution outcome.
func (journal *CampaignJournal) AppendExecution(run ExecutionRecord) error {
	if !journal.started {
		return errors.New("campaign execution journal is not open")
	}
	return journal.append(kindOutcome, outcomeRecord{Executions: []ExecutionRecord{run}})
}

func (journal *CampaignJournal) BeginExecution(ordinal, seed uint64) (_ *ExecutionJournal, retErr error) {
	journal.partialMu.Lock()
	if journal.partialRuns == journal.config.Journal.MaximumPartialExecutions {
		journal.partialMu.Unlock()
		return nil, &JournalCapacityError{
			Limit: journalLimitPartialExecutions, Required: journal.partialRuns + 1,
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
	path := filepath.Join(journal.path, ".partial", executionWorkName(ordinal, seed))
	if err := makePrivateDirectoriesContext(journal.ctx, path); err != nil {
		return nil, err
	}
	run := &ExecutionJournal{ctx: journal.ctx, path: path, ordinal: ordinal, seed: seed, state: executionStaging, releasePartial: release}
	if err := makePrivateDirectoriesContext(journal.ctx, run.WorkPath()); err != nil {
		return run, err
	}
	return run, nil
}

func executionWorkName(ordinal, seed uint64) string {
	return fmt.Sprintf("%020d-%d", ordinal, seed)
}

// Publish commits the campaign's final summary. The publication record is
// the campaign's last; removing the prepared target afterwards is cleanup
// that recovery finishes if it is interrupted.
func (journal *CampaignJournal) Publish(summary CampaignSummary) error {
	if !journal.started {
		return errors.New("campaign execution journal is not open")
	}
	if journal.state.published != nil {
		return errors.New("campaign journal is already published")
	}
	if err := journal.ctx.Err(); err != nil {
		return err
	}
	failureSignatures := append([]record.SHA256(nil), summary.FailureSignatures...)
	if failureSignatures == nil {
		failureSignatures = []record.SHA256{}
	}
	sort.Slice(failureSignatures, func(i, j int) bool { return failureSignatures[i] < failureSignatures[j] })
	journal.mu.Lock()
	batch := journal.state.publicationIdentity()
	batch.SchemaVersion, batch.Schema = record.SchemaVersion, "gomad3.campaign/v1"
	batch.Attempted, batch.Succeeded = record.Uint64String(summary.Attempted), record.Uint64String(summary.Succeeded)
	batch.Failures, batch.ReplayDivergences = record.Uint64String(summary.Failures), record.Uint64String(summary.ReplayDivergences)
	batch.Watchdogs, batch.Cancelled = record.Uint64String(summary.Watchdogs), record.Uint64String(summary.Cancelled)
	batch.DistinctFailures, batch.StopReason = record.Uint64String(summary.DistinctFailures), summary.StopReason
	batch.RetainedSuccesses, batch.RetainedSuccessBytes = record.Uint64String(summary.RetainedSuccesses), record.Uint64String(summary.RetainedSuccessBytes)
	batch.FailureSignatures = failureSignatures
	batch.ChoiceExploration, batch.ChoiceExplorationImplementationSHA256, batch.ChoiceExplorationChainSHA256 = summary.ChoiceExploration, summary.ChoiceExplorationImplementationSHA256, summary.ChoiceExplorationChainSHA256
	batch.SimulationExploration, batch.SimulationExplorationImplementationSHA256, batch.SimulationExplorationChainSHA256 = summary.SimulationExploration, summary.SimulationExplorationImplementationSHA256, summary.SimulationExplorationChainSHA256
	batch.RecoveryExecutions = record.Uint64String(summary.RecoveryExecutions)
	err := journal.appendLocked(kindPublish, batch)
	journal.mu.Unlock()
	if err != nil {
		return err
	}
	if err := removeCompletedPartialContext(journal.ctx, journal.PreparedPath(), "prepared-target"); err != nil {
		return err
	}
	return syncDirectoryContext(journal.ctx, journal.path)
}

func cloneCampaignShard(shard *CampaignShard) *CampaignShard {
	if shard == nil {
		return nil
	}
	copy := *shard
	return &copy
}

// Close releases the campaign log and its lock. It is idempotent.
func (journal *CampaignJournal) Close() error {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	if journal.writer == nil {
		return nil
	}
	err := journal.writer.Close()
	journal.writer = nil
	return err
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

func (run *ExecutionJournal) Transition(next executionState) error {
	allowed := run.state == executionStaging && next == ExecutionStarting ||
		run.state == ExecutionStarting && next == ExecutionExited ||
		run.state == ExecutionExited && next == ExecutionCaptured ||
		run.state == ExecutionCaptured && next == ExecutionClassified
	if !allowed {
		return fmt.Errorf("invalid execution journal transition %q -> %q", run.state, next)
	}
	if run.stateFile {
		if err := run.writeState(next); err != nil {
			return err
		}
	}
	run.state = next
	return nil
}

// Preserve keeps an interrupted attempt's work directory for inspection
// until the campaign resumes, which reclaims it.
func (run *ExecutionJournal) Preserve() error {
	if run.stateFile {
		return run.writeState("preserve-partial")
	}
	return nil
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
