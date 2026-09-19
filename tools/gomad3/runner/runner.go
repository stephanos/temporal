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

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	choiceengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/choice"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/world"
)

type FailurePolicy string

type CoverageMode string

type KeepSuccesses string

type Strategy string

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
	CoverageNone           CoverageMode = "none"
	CoverageSemantic       CoverageMode = "semantic"
	CoverageChoice         CoverageMode = "choice"
	CoverageSemanticChoice CoverageMode = "semantic+choice"
)

const (
	StrategySeed                  Strategy = "seed"
	StrategyChoiceExploration     Strategy = "choice-exploration"
	StrategySimulationExploration Strategy = "simulation-exploration"
)

type StopReason string

const (
	StopSeedsExhausted          StopReason = "seeds_exhausted"
	StopFirstFailure            StopReason = "first_failure"
	StopFailureBudget           StopReason = "failure_budget"
	StopExplorationExhausted    StopReason = "exploration_exhausted"
	StopChoiceDepthComplete     StopReason = "choice_depth_complete"
	StopSimulationDepthComplete StopReason = "simulation_depth_complete"
	StopDimensionDepthComplete  StopReason = "dimension_depth_complete"
	StopMaxExecutions           StopReason = "max_executions"
	StopExplorationCapacity     StopReason = "exploration_capacity"
)

type CampaignPhase string

const (
	ProgressPreparing CampaignPhase = "preparing"
	ProgressRunning   CampaignPhase = "running"
	ProgressComplete  CampaignPhase = "complete"
)

type CampaignEvent struct {
	Phase                 CampaignPhase
	CampaignPath          string `json:"campaign_path"`
	Selected              uint64
	Attempted             uint64
	Running               uint64
	Succeeded             uint64
	Failures              uint64
	Watchdogs             uint64
	ReplayDivergences     uint64
	Cancelled             uint64
	DistinctFailures      uint64
	Artifacts             []string
	RetainedSuccesses     uint64
	RetainedSuccessBytes  uint64
	SuccessArtifacts      []string
	CorpusPath            string
	CorpusEntries         uint64
	CorpusAdded           uint64
	ChoiceTrace           *ChoiceTraceSummary
	ChoiceExploration     *ChoiceExplorationSummary
	SimulationExploration *SimulationExplorationSummary
	RecoveryExecutions    uint64
}

type CampaignEventFunc func(CampaignEvent) error

type Preparer interface {
	Prepare(context.Context, target.Spec) (target.Prepared, error)
}

type Executor interface {
	Run(context.Context, execution.Spec) (execution.Result, error)
}

type ArtifactReplayer interface {
	Replay(context.Context, ReplaySpec) (ReplayResult, error)
}

type CampaignSpec struct {
	ResumeCampaign            string
	PlanSHA256                record.SHA256
	Shard                     CampaignShard
	Strategy                  Strategy
	Seeds                     string
	Parallel                  int
	ExecutionTimeout          time.Duration
	OverallTimeout            time.Duration
	TerminateGrace            time.Duration
	OnFailure                 FailurePolicy
	FailureBudget             uint64
	OutputLimit               uint64
	WorldTransitionLimit      uint64
	ChoiceTraceLimit          uint64
	MaxExecutions             uint64
	MaxChoiceDepth            uint64
	MaxForcedDecisions        uint64
	MaxExplorationBytes       uint64
	MaxExplorationResultBytes uint64
	SimulationDimensionLimits SimulationDimensionLimits
	Artifacts                 string
	Environment               []string
	IOROMounts                []string
	IOROMountLimits           readonlymount.Limits
	Target                    target.Spec
	SupervisorCommand         []string
	CoordinatorCommand        []string
	RunnerBuild               string
	Coverage                  CoverageMode
	RequiredSemanticProbes    []string
	CollectExecutionEvidence  bool
	KeepSuccesses             KeepSuccesses
	SuccessArtifactLimit      uint64
	SuccessBytesLimit         uint64
	Guide                     bool
	Corpus                    string
	GuideSnapshotSHA256       record.SHA256
	Progress                  CampaignEventFunc
	ProgressInterval          time.Duration
	Preparer                  Preparer
	Executor                  Executor
	Replayer                  ArtifactReplayer
	resumePreflight           *campaign.ResumePreflight
	failureArtifactLimit      uint64
	failureBytesLimit         uint64
}

type CampaignResult struct {
	CampaignPath          string `json:"campaign_path"`
	SelectionCount        uint64
	Attempted             uint64
	Succeeded             uint64
	Failures              uint64
	Watchdogs             uint64
	ReplayDivergences     uint64
	Cancelled             uint64
	DistinctFailures      uint64
	StopReason            StopReason
	Artifacts             []string
	RetainedSuccesses     uint64
	RetainedSuccessBytes  uint64
	SuccessArtifacts      []string
	SemanticCoverage      *deterministicio.SemanticCoverage
	ExecutionEvidence     *ExecutionEvidence `json:"execution_evidence"`
	CorpusPath            string
	CorpusEntries         uint64
	CorpusAdded           uint64
	ChoiceTrace           *ChoiceTraceSummary
	ChoiceExploration     *ChoiceExplorationSummary
	SimulationExploration *SimulationExplorationSummary
	RecoveryExecutions    uint64
	failureArtifactBytes  uint64
}

type ChoiceTraceSummary struct {
	Seed             uint64
	Profile          string
	Limit            uint64
	SHA256           record.SHA256
	Records          uint64
	BranchingRecords uint64
	Runnable         uint64
	SelectPoll       uint64
	SelectResult     uint64
	TerminalState    string
	TapeSHA256       record.SHA256
	Decisions        uint64
}

type ChoiceExplorationSummary struct {
	Parallel                int    `json:"parallel"`
	MaxExecutions           uint64 `json:"max_executions"`
	MaxChoiceDepth          uint64 `json:"max_choice_depth"`
	MaxExplorationBytes     uint64 `json:"max_exploration_bytes"`
	LogicalExecutions       uint64 `json:"logical_executions"`
	CommittedRounds         uint64 `json:"committed_rounds"`
	Pending                 uint64 `json:"pending"`
	PendingBytes            uint64 `json:"pending_bytes"`
	SeenPrefixes            uint64 `json:"seen_prefixes"`
	DeduplicatedOutcomes    uint64 `json:"deduplicated_outcomes"`
	DeepestPrefix           uint64 `json:"deepest_prefix"`
	OmittedByExecutionBound uint64 `json:"omitted_by_execution_bound"`
	OmittedByDepth          uint64 `json:"omitted_by_depth"`
	OmittedByCapacity       uint64 `json:"omitted_by_capacity"`
	StopReason              string `json:"stop_reason,omitempty"`
	BoundedComplete         bool   `json:"bounded_complete"`
}

type SimulationExplorationSummary struct {
	Parallel                int                       `json:"parallel"`
	MaxExecutions           uint64                    `json:"max_executions"`
	MaxForcedDecisions      uint64                    `json:"max_forced_decisions"`
	MaxExplorationBytes     uint64                    `json:"max_exploration_bytes"`
	MaxResultBytes          uint64                    `json:"max_result_bytes"`
	FailureBudget           uint64                    `json:"failure_budget"`
	Limits                  SimulationDimensionLimits `json:"dimension_limits"`
	LogicalExecutions       uint64                    `json:"logical_executions"`
	CommittedRounds         uint64                    `json:"committed_rounds"`
	Pending                 uint64                    `json:"pending"`
	PendingBytes            uint64                    `json:"pending_bytes"`
	SeenCandidates          uint64                    `json:"seen_candidates"`
	DeduplicatedOutcomes    uint64                    `json:"deduplicated_outcomes"`
	DistinctFailures        uint64                    `json:"distinct_failures"`
	DeepestOverride         uint64                    `json:"deepest_override"`
	OmittedByExecutionBound uint64                    `json:"omitted_by_execution_bound"`
	OmittedByDepth          uint64                    `json:"omitted_by_depth"`
	OmittedByDimension      uint64                    `json:"omitted_by_dimension"`
	OmittedByCapacity       uint64                    `json:"omitted_by_capacity"`
	StopReason              string                    `json:"stop_reason,omitempty"`
	BoundedComplete         bool                      `json:"bounded_complete"`
}

type SimulationDimensionLimits struct {
	Runtime  uint64 `json:"runtime"`
	Scenario uint64 `json:"scenario"`
	Network  uint64 `json:"network"`
	Storage  uint64 `json:"storage"`
	Fault    uint64 `json:"fault"`
	Crash    uint64 `json:"crash"`
}

type HostError struct {
	Reason string
	Err    error
}

func (err *HostError) Error() string {
	if err.Err == nil {
		return "gomad3 Runner/host failure: " + err.Reason
	}
	return "gomad3 Runner/host failure: " + err.Reason + ": " + err.Err.Error()
}

func (err *HostError) Unwrap() error {
	return err.Err
}

func contextFailureReason(err error) string {
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	return "overall_timeout"
}

type targetPreparer struct{}

func (targetPreparer) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return target.Prepare(ctx, spec)
}

type processExecutor struct{}

func (processExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	return execution.Run(ctx, request)
}

type artifactReplayer struct{}

func (artifactReplayer) Replay(ctx context.Context, config ReplaySpec) (ReplayResult, error) {
	return Replay(ctx, config)
}

type runJob struct {
	ordinal               uint64
	seed                  uint64
	choiceMode            choice.Mode
	choiceReplayPlan      *choice.ReplayPlan
	simulationPlan        string
	simulationRecordLimit uint64
	simulationRecordCount uint64
}

type runCompletion struct {
	job        runJob
	startedAt  time.Time
	finishedAt time.Time
	result     execution.Result
	err        error
	journal    *campaign.ExecutionJournal
}

type runReadiness struct {
	ready    chan struct{}
	signaled bool
}

func newRunReadiness() *runReadiness {
	return &runReadiness{ready: make(chan struct{})}
}

func (readiness *runReadiness) signal() {
	if readiness.signaled {
		return
	}
	close(readiness.ready)
	readiness.signaled = true
}

func (readiness *runReadiness) wait() {
	<-readiness.ready
}

type runJournalFactory interface {
	BeginExecution(uint64, uint64) (*campaign.ExecutionJournal, error)
}

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func Explore(ctx context.Context, config CampaignSpec) (CampaignResult, error) {
	if config.ResumeCampaign != "" {
		var err error
		config, err = resumeRequestDefaults(config)
		if err != nil {
			return CampaignResult{}, err
		}
	}
	if len(config.CoordinatorCommand) != 0 {
		if config.Preparer != nil || config.Executor != nil || config.Replayer != nil {
			return CampaignResult{}, fmt.Errorf("isolated Runner does not accept injected preparation or execution")
		}
		return runIsolated(ctx, config)
	}
	return runLocal(ctx, config)
}

func runLocal(ctx context.Context, config CampaignSpec) (summary CampaignResult, retErr error) {
	resuming := config.ResumeCampaign != ""
	selection, baseEnvironment, err := validateConfig(config)
	var readOnlyMounts []readonlymount.Mapping
	var prepared target.Prepared
	var resumePlan campaign.CampaignPlan
	var guidance *guidanceCampaign
	defer func() {
		if guidance != nil {
			retErr = errors.Join(retErr, guidance.Close())
		}
	}()
	if err != nil {
		return CampaignResult{}, err
	}
	if resuming {
		preflight := config.resumePreflight
		if preflight == nil {
			var opened campaign.ResumePreflight
			opened, err = campaign.PreflightResume(config.ResumeCampaign)
			preflight = &opened
		}
		if err == nil {
			resumePlan = preflight.Plan
			config, selection, baseEnvironment, readOnlyMounts, prepared, err = resumeConfiguration(config, resumePlan)
			config.resumePreflight = preflight
		}
		if err != nil {
			return CampaignResult{}, err
		}
	} else {
		config.Artifacts, err = filepath.Abs(config.Artifacts)
		if err != nil {
			return CampaignResult{}, &HostError{Reason: "artifact_setup", Err: fmt.Errorf("resolve artifact root: %w", err)}
		}
		readOnlyMounts, err = readonlymount.ParseMappings(config.IOROMounts, config.Target.WorkingDir)
		if err != nil {
			return CampaignResult{}, err
		}
		if config.IOROMountLimits == (readonlymount.Limits{}) {
			config.IOROMountLimits = readonlymount.DefaultLimits()
		}
		if config.Guide {
			config.Corpus, err = guidedCorpusPath(config.Corpus)
			if err != nil {
				return CampaignResult{}, err
			}
		}
	}
	overallCtx, overallCancel := context.WithTimeout(ctx, config.OverallTimeout)
	defer overallCancel()
	var runID string
	var batchPath string
	var journal *campaign.CampaignJournal
	var resumedRuns []campaign.ExecutionRecord
	if resuming {
		batchPath = config.ResumeCampaign
		runID = filepath.Base(batchPath)
		var resumeState campaign.ResumeState
		journal, resumeState, err = campaign.ResumeCampaignJournal(overallCtx, batchPath)
		if err == nil {
			var equal bool
			equal, err = equalBatchPlans(resumePlan, resumeState.Plan)
			if !equal && err == nil {
				err = errors.New("campaign plan changed while acquiring its resume lock")
			}
			resumedRuns = resumeState.Executions
		}
		if err != nil {
			return CampaignResult{CampaignPath: batchPath, SelectionCount: selection.Count()}, &HostError{Reason: "resume_setup", Err: err}
		}
		var restored resumeSummaryState
		restored, err = restoreResumeSummary(batchPath, selection, resumedRuns)
		summary = restored.summary
		summary.SelectionCount = normalizedCampaignShard(config.Shard).SelectionCount(selection.Count())
		if err != nil {
			return summary, &HostError{Reason: "resume_setup", Err: err}
		}
	} else {
		runID, err = newRunID()
		if err != nil {
			return CampaignResult{}, &HostError{Reason: "campaign_id", Err: err}
		}
		batchPath = filepath.Join(config.Artifacts, "v1", runID)
		summary = CampaignResult{CampaignPath: batchPath, SelectionCount: normalizedCampaignShard(config.Shard).SelectionCount(selection.Count())}
		journal, err = campaign.NewCampaignJournal(overallCtx, campaign.CampaignConfig{
			Root: config.Artifacts, CampaignID: runID, PlanSHA256: config.PlanSHA256, Shard: campaignStoreShard(config.Shard),
			Strategy: string(normalizedStrategy(config.Strategy)), Selection: config.Seeds, SelectionCount: selection.Count(), MaxExecutions: config.MaxExecutions, Parallel: uint64(config.Parallel),
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
	reportProgress := func(phase CampaignPhase, active int) error {
		if config.Progress == nil {
			return nil
		}
		return config.Progress(CampaignEvent{
			Phase: phase, CampaignPath: summary.CampaignPath, Selected: summary.SelectionCount, Attempted: summary.Attempted, Running: uint64(active),
			Succeeded: summary.Succeeded, Failures: summary.Failures, Watchdogs: summary.Watchdogs, ReplayDivergences: summary.ReplayDivergences, Cancelled: summary.Cancelled,
			DistinctFailures: summary.DistinctFailures, Artifacts: append([]string(nil), summary.Artifacts...),
			RetainedSuccesses: summary.RetainedSuccesses, RetainedSuccessBytes: summary.RetainedSuccessBytes, SuccessArtifacts: append([]string(nil), summary.SuccessArtifacts...),
			CorpusPath: summary.CorpusPath, CorpusEntries: summary.CorpusEntries, CorpusAdded: summary.CorpusAdded,
			ChoiceTrace: cloneChoiceTraceSummary(summary.ChoiceTrace), ChoiceExploration: cloneChoiceExplorationSummary(summary.ChoiceExploration), SimulationExploration: cloneSimulationExplorationSummary(summary.SimulationExploration), RecoveryExecutions: summary.RecoveryExecutions,
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
		var missing *deterministicio.MissingSemanticProbesError
		if errors.As(retErr, &missing) {
			reason = "semantic_coverage"
		}
		if partialErr := journal.Fail(reason, retErr); partialErr != nil {
			retErr = errors.Join(retErr, partialErr)
		}
	}()
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: contextFailureReason(err), Err: err}
	}
	selectedProfile := deterministicio.Default()
	if !resuming {
		if err := journal.BeginPreparation(); err != nil {
			return summary, &HostError{Reason: "target_preparation_setup", Err: err}
		}
		config.Target.PreparationRoot = journal.PreparedPath()
		preparer := config.Preparer
		selectedAdapters := []deterministicio.BuildAdapter{}
		if preparer == nil {
			moduleCache, cacheErr := target.ReadModuleCache(overallCtx, config.Target.ToolchainRoot)
			if cacheErr != nil {
				return summary, cacheErr
			}
			var profileErr error
			config.Target, selectedAdapters, profileErr = selectedProfile.PrepareBuildAdapters(config.Target, moduleCache)
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
			if contextErr := overallCtx.Err(); contextErr != nil {
				reason = contextFailureReason(contextErr)
			}
			if partialErr := journal.FailPreparation(reason, err); partialErr != nil {
				err = errors.Join(err, partialErr)
			}
			return summary, &HostError{Reason: reason, Err: err}
		}
		prepared.Adapters = executionAdapters(selectedAdapters)
		if profileErr := selectedProfile.ValidatePreparedTarget(config.Target, prepared, config.Environment); profileErr != nil {
			return summary, profileErr
		}
		if config.Guide {
			guidance, err = openGuidance(overallCtx, config, prepared, baseEnvironment, runID)
			if err != nil {
				return summary, &HostError{Reason: "guided_corpus", Err: err}
			}
			snapshot := guidance.Snapshot()
			selection, err = mixGuidedSelection(selection, snapshot.PrioritizedSeeds())
			if err != nil {
				return summary, &HostError{Reason: "guided_selection", Err: err}
			}
			config.Seeds = selection.String()
			config.GuideSnapshotSHA256 = snapshot.SnapshotSHA256
			guidance.config = config
			summary.SelectionCount = selection.Count()
			summary.CorpusPath = guidance.corpus.Path()
			summary.CorpusEntries = uint64(len(snapshot.Entries))
			if err := journal.SetSelection(config.Seeds, selection.Count()); err != nil {
				return summary, &HostError{Reason: "guided_selection", Err: err}
			}
		}
		plan, err := campaignPlan(config, journal, prepared, baseEnvironment, readOnlyMounts, selection.Count())
		if err != nil {
			return summary, &HostError{Reason: "campaign_plan", Err: err}
		}
		if err := journal.RecordPlan(plan); err != nil {
			return summary, &HostError{Reason: "campaign_plan", Err: err}
		}
		config.failureArtifactLimit = uint64(plan.Artifacts.FailureArtifacts)
		config.failureBytesLimit = uint64(plan.Artifacts.FailureBytes)
		if err := journal.CompletePreparation(); err != nil {
			return summary, &HostError{Reason: "partial_cleanup", Err: err}
		}
	}
	if resuming && config.Guide {
		guidance, err = openGuidance(overallCtx, config, prepared, baseEnvironment, runID)
		if err != nil {
			return summary, &HostError{Reason: "guided_corpus", Err: err}
		}
		snapshot := guidance.Snapshot()
		summary.CorpusPath = guidance.corpus.Path()
		summary.CorpusEntries = uint64(len(snapshot.Entries))
	}
	executor := config.Executor
	if executor == nil {
		executor = processExecutor{}
	}

	if !resuming {
		err = journal.StartExecutions()
	}
	if err != nil {
		return summary, &HostError{Reason: "runs_create", Err: err}
	}
	if err := reportProgress(ProgressRunning, 0); err != nil {
		return summary, &HostError{Reason: "progress_output", Err: err}
	}
	if normalizedStrategy(config.Strategy) == StrategyChoiceExploration {
		err := runChoiceExplorationLocal(overallCtx, config, selection, baseEnvironment, readOnlyMounts, prepared, selectedProfile, journal, runID, resuming, resumedRuns, &summary, reportProgress)
		if err == nil {
			batchComplete = true
		}
		return summary, err
	}
	if normalizedStrategy(config.Strategy) == StrategySimulationExploration {
		err := runSimulationExplorationLocal(overallCtx, config, selection, baseEnvironment, readOnlyMounts, prepared, selectedProfile, journal, runID, resuming, resumedRuns, &summary, reportProgress)
		if err == nil {
			batchComplete = true
		}
		return summary, err
	}
	activeCtx, activeCancel := context.WithCancel(overallCtx)
	defer activeCancel()
	rawCompletions := make(chan runCompletion, config.Parallel)
	completions := make(chan runCompletion, config.Parallel)
	completed := make(map[uint64]struct{})
	var hostFailure error
	distinct := make(map[record.SHA256]string)
	failureArtifactBytes := uint64(0)
	semanticProbes := make(map[string]struct{})
	choiceFeatures := make(map[string]struct{})
	if resuming {
		var restored resumeSummaryState
		restored, err = restoreResumeSummary(batchPath, selection, resumedRuns)
		if err != nil {
			return summary, &HostError{Reason: "resume_setup", Err: err}
		}
		summary = restored.summary
		summary.SelectionCount = normalizedCampaignShard(config.Shard).SelectionCount(selection.Count())
		distinct = restored.distinct
		failureArtifactBytes = restored.failureArtifactBytes
		semanticProbes = restored.probes
		choiceFeatures = restored.choiceFeatures
		completed = restored.completed
		if guidance != nil {
			snapshot := guidance.Snapshot()
			summary.CorpusPath = guidance.corpus.Path()
			summary.CorpusEntries = uint64(len(snapshot.Entries))
		}
	}
	completionOrderDone := make(chan struct{})
	go func() {
		defer close(completionOrderDone)
		orderShardRunCompletions(selection, config.Shard, completed, rawCompletions, completions)
	}()
	defer func() {
		close(rawCompletions)
		<-completionOrderDone
	}()
	controller, err := newShardedSeedController(selection, config.Shard, completed, config.Parallel, config.OnFailure, config.FailureBudget, summary)
	if err != nil {
		return summary, &HostError{Reason: "campaign_setup", Err: err}
	}
	synchronizeCampaignStatistics(&summary, controller.Statistics())
	publishRunnerFailure := func(completion runCompletion, reason string) error {
		if !completion.result.Captured {
			return nil
		}
		if reason == "choice_trace_malformed" || reason == "choice_trace_unterminated" {
			return nil
		}
		worldBundle := noneWorldBundle()
		outcome := execution.Classification{
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
		published, err := publishBoundedFailureArtifact(overallCtx, config, journal.FailuresPath(), manifest.Outcome.FailureSignature, distinct, &failureArtifactBytes, artifact.ArtifactInput{
			Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
			IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
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
		run := campaign.ExecutionRecord{
			SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
			Domain: "runner", Reason: reason, Termination: "none", FailureSignature: &signature, Artifact: &artifactRelative,
			ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
		}
		setRunTranscript(&run, completion.result.IOTranscript)
		setRunChoiceTrace(&run, completion.result.ChoiceTrace)
		if err := journal.AppendExecution(run); err != nil {
			return fmt.Errorf("append Runner failure result: %w", err)
		}
		return nil
	}
	completePartial := func(run *campaign.ExecutionJournal) {
		if cleanupErr := run.Complete(); cleanupErr != nil && hostFailure == nil {
			hostFailure = &HostError{Reason: "partial_cleanup", Err: cleanupErr}
			controller.Stop()
			activeCancel()
		}
	}

	launch := func(job runJob) {
		readiness := newRunReadiness()
		go runSeed(activeCtx, config, executor, prepared, baseEnvironment, selectedProfile, readOnlyMounts, journal, job, readiness, rawCompletions)
		readiness.wait()
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
	for !controller.Done() {
		admittedThisTurn := false
		for overallCtx.Err() == nil {
			scheduled, ok := controller.Next()
			if !ok {
				break
			}
			job := runJob{ordinal: scheduled.Ordinal, seed: scheduled.Seed}
			launch(job)
			admittedThisTurn = true
		}
		if controller.Active() > 0 && !runningReported {
			runningReported = true
			if err := reportProgress(ProgressRunning, controller.Active()); err != nil {
				hostFailure = &HostError{Reason: "progress_output", Err: err}
				controller.Stop()
				activeCancel()
			}
		}
		if overallCtx.Err() != nil && hostFailure == nil && !controller.Stopped() {
			hostFailure = &HostError{Reason: contextFailureReason(overallCtx.Err()), Err: overallCtx.Err()}
			controller.Stop()
			activeCancel()
		}
		if controller.Active() == 0 {
			break
		}
		var completion runCompletion
		select {
		case completion = <-completions:
		case <-progressTicks:
			if admittedThisTurn {
				continue
			}
			if err := reportProgress(ProgressRunning, controller.Active()); err != nil {
				hostFailure = &HostError{Reason: "progress_output", Err: err}
				controller.Stop()
				activeCancel()
			}
			continue
		}
		controller.FinishAttempt()
		synchronizeCampaignStatistics(&summary, controller.Statistics())
		if overallCtx.Err() != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: contextFailureReason(overallCtx.Err()), Err: overallCtx.Err()}
			}
			controller.Stop()
			activeCancel()
			continue
		}
		if completion.err != nil {
			reason := supervisionFailureReason(completion.err)
			if completion.result.ChoiceTrace.Profile != "" && completion.result.ChoiceTrace.Trace.Summary.Terminal == choice.TerminalOverflow {
				summary.ChoiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
			}
			if partialErr := preservePartial(completion.journal); partialErr != nil {
				completion.err = errors.Join(completion.err, partialErr)
			}
			if publishErr := publishRunnerFailure(completion, reason); publishErr != nil {
				completion.err = errors.Join(completion.err, publishErr)
			}
			if hostFailure == nil {
				hostFailure = &HostError{Reason: reason, Err: completion.err}
				controller.Stop()
				activeCancel()
			}
			continue
		}
		if err := prepared.Verify(); err != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "prepared_target_integrity", Err: err}
				controller.Stop()
				activeCancel()
			}
			continue
		}
		if config.ChoiceTraceLimit != 0 {
			summary.ChoiceTrace = choiceTraceSummary(completion.job.seed, completion.result.ChoiceTrace)
		}
		if completion.result.Cancelled && controller.Stopped() {
			controller.RecordCancelled()
			synchronizeCampaignStatistics(&summary, controller.Statistics())
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
			run := campaign.ExecutionRecord{
				SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
				Domain: "runner", Reason: "runner_cancelled", Termination: "none", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
			}
			if err := journal.AppendExecution(run); err != nil && hostFailure == nil {
				hostFailure = &HostError{Reason: "runs_append", Err: err}
				controller.Stop()
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
				worldBundle, err = execution.ComposeRecording(recording, config.WorldTransitionLimit)
			}
			if err == nil {
				initialWorld, _, validateErr := execution.Validate(worldBundle.Manifest, worldBundle.Payloads)
				if validateErr != nil {
					err = validateErr
				} else if worldBundle.Manifest.Initial.Schema != "gomad3.world.snapshot/v1" || uint64(initialWorld.Config.Seed) != completion.job.seed {
					err = fmt.Errorf("World record seed or schema does not match seed %d", completion.job.seed)
				}
			}
			if err != nil {
				if publishErr := publishRunnerFailure(completion, "world_record"); publishErr != nil {
					err = errors.Join(err, publishErr)
				}
				if hostFailure == nil {
					hostFailure = &HostError{Reason: "world_record", Err: err}
					controller.Stop()
					activeCancel()
				}
				continue
			}
		}
		runCoverage, coverageErr := deterministicio.SummarizeSemanticProbes(nil)
		if coverageErr != nil {
			return summary, &HostError{Reason: "semantic_coverage", Err: coverageErr}
		}
		if coverageHasSemantic(config.Coverage) {
			coverage, coverageErr := deterministicio.DecodeSemanticCoverage(completion.result.IOTranscript.Bytes)
			if coverageErr != nil {
				if partialErr := preservePartial(completion.journal); partialErr != nil {
					coverageErr = errors.Join(coverageErr, partialErr)
				}
				if hostFailure == nil {
					hostFailure = &HostError{Reason: "semantic_coverage", Err: coverageErr}
					controller.Stop()
					activeCancel()
				}
				continue
			}
			runCoverage = coverage
		}
		runChoiceFeatures := []string{}
		var runChoiceProjection *choice.FeatureProjection
		if coverageHasChoice(config.Coverage) {
			projected, features, choiceErr := projectChoiceFeatures(completion.result.ChoiceTrace, prepared)
			if choiceErr != nil {
				if partialErr := preservePartial(completion.journal); partialErr != nil {
					choiceErr = errors.Join(choiceErr, partialErr)
				}
				if hostFailure == nil {
					hostFailure = &HostError{Reason: "choice_coverage", Err: choiceErr}
					controller.Stop()
					activeCancel()
				}
				continue
			}
			runChoiceProjection = &projected
			runChoiceFeatures = features
		}
		novelProbes := novelSemanticProbes(runCoverage.Probes, semanticProbes)
		novelChoices := novelStrings(runChoiceFeatures, choiceFeatures)
		outcome := execution.Classify(completion.result, false, worldBundle.Manifest.Terminal)
		if config.CollectExecutionEvidence {
			mountArtifact, evidenceErr := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
			if evidenceErr != nil {
				if hostFailure == nil {
					hostFailure = &HostError{Reason: "execution_evidence", Err: evidenceErr}
					controller.Stop()
					activeCancel()
				}
				continue
			}
			runRecord := executionEvidence(config, prepared, baseEnvironment, completion, outcome, worldBundle.Manifest, mountArtifact, runCoverage, runChoiceProjection)
			summary.ExecutionEvidence = &runRecord
		}
		if err := completion.journal.Transition(campaign.ExecutionClassified); err != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "partial_write", Err: err}
				controller.Stop()
				activeCancel()
			}
			continue
		}
		if overallErr := overallCtx.Err(); overallErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: contextFailureReason(overallErr), Err: overallErr}
				controller.Stop()
				activeCancel()
			}
			if partialErr := preservePartial(completion.journal); partialErr != nil {
				hostFailure = errors.Join(hostFailure, &HostError{Reason: "partial_write", Err: partialErr})
			}
			continue
		}
		if outcome.Domain == "success" {
			run := campaign.ExecutionRecord{
				SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
				Domain: "success", Reason: outcome.Reason, Termination: "exit", ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
			}
			setRunTranscript(&run, completion.result.IOTranscript)
			setRunChoiceTrace(&run, completion.result.ChoiceTrace)
			run.SemanticProbes = append([]string(nil), runCoverage.Probes...)
			run.ChoiceFeatures = append([]string(nil), runChoiceFeatures...)
			retain := config.KeepSuccesses == KeepSuccessesAll || config.KeepSuccesses == KeepSuccessesNovel && (len(novelProbes) != 0 || len(novelChoices) != 0)
			if retain {
				if !completion.result.IOTranscript.Complete {
					hostFailure = &HostError{Reason: "success_artifact_publication", Err: errors.New("retained success requires a complete I/O transcript for exact replay")}
					controller.Stop()
					activeCancel()
					continue
				}
				if summary.RetainedSuccesses >= config.SuccessArtifactLimit || summary.RetainedSuccessBytes >= config.SuccessBytesLimit {
					hostFailure = &HostError{Reason: "success_retention_capacity", Err: errors.New("successful-execution retention capacity is exhausted")}
					controller.Stop()
					activeCancel()
					continue
				}
				mountArtifact, publishErr := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
				if publishErr == nil {
					var manifest record.ExecutionRecord
					manifest, publishErr = manifestForRun(config, prepared, baseEnvironment, completion, outcome, runID, worldBundle.Manifest, mountArtifact)
					if publishErr == nil {
						var published artifact.Artifact
						published, publishErr = artifact.PublishArtifact(artifact.Store{Root: journal.SuccessesPath(), Context: overallCtx, MaximumBytes: config.SuccessBytesLimit - summary.RetainedSuccessBytes}, artifact.ArtifactInput{
							Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
							IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
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
									run.NovelChoiceFeatures = append([]string(nil), novelChoices...)
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
					controller.Stop()
					activeCancel()
					continue
				}
			}
			if guidance != nil {
				mountArtifact, guideErr := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
				var added bool
				if guideErr == nil {
					added, guideErr = guidance.MergeRun(overallCtx, completion, outcome, worldBundle, mountArtifact, runCoverage)
				}
				if guideErr != nil {
					hostFailure = &HostError{Reason: "guided_corpus", Err: guideErr}
					controller.Stop()
					activeCancel()
					continue
				}
				if added {
					summary.CorpusAdded++
					summary.CorpusEntries = uint64(len(guidance.Snapshot().Entries))
				}
			}
			if err := journal.AppendExecution(run); err != nil && hostFailure == nil {
				hostFailure = &HostError{Reason: "runs_append", Err: err}
				controller.Stop()
				activeCancel()
			}
			if hostFailure == nil {
				controller.RecordSuccess()
				synchronizeCampaignStatistics(&summary, controller.Statistics())
				addSemanticProbes(semanticProbes, runCoverage.Probes)
				addStrings(choiceFeatures, runChoiceFeatures)
			}
			completePartial(completion.journal)
			continue
		}

		mountArtifact, manifestErr := mountArtifactForRun(readOnlyMounts, config.IOROMountLimits, completion.result.IOROMounts)
		if manifestErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "manifest", Err: manifestErr}
				controller.Stop()
				activeCancel()
			}
			continue
		}
		manifest, manifestErr := manifestForRun(config, prepared, baseEnvironment, completion, outcome, runID, worldBundle.Manifest, mountArtifact)
		if manifestErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "manifest", Err: manifestErr}
				controller.Stop()
				activeCancel()
			}
			continue
		}
		published, publishErr := publishBoundedFailureArtifact(overallCtx, config, journal.FailuresPath(), manifest.Outcome.FailureSignature, distinct, &failureArtifactBytes, artifact.ArtifactInput{
			Manifest: manifest, TargetPath: prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
			IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
		})
		if publishErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: "artifact_publication", Err: publishErr}
				controller.Stop()
				activeCancel()
			}
			continue
		}
		if overallErr := overallCtx.Err(); overallErr != nil {
			if hostFailure == nil {
				hostFailure = &HostError{Reason: contextFailureReason(overallErr), Err: overallErr}
				controller.Stop()
				activeCancel()
			}
			continue
		}
		if guidance != nil {
			added, guideErr := guidance.MergeRun(overallCtx, completion, outcome, worldBundle, mountArtifact, runCoverage)
			if guideErr != nil {
				hostFailure = &HostError{Reason: "guided_corpus", Err: guideErr}
				controller.Stop()
				activeCancel()
				continue
			}
			if added {
				summary.CorpusAdded++
				summary.CorpusEntries = uint64(len(guidance.Snapshot().Entries))
			}
		}
		signature := published.Manifest.Outcome.FailureSignature
		if _, found := distinct[signature]; !found {
			distinct[signature] = published.Path
			summary.Artifacts = append(summary.Artifacts, published.Path)
		}
		cancelActive := controller.RecordFailure(outcome.Domain, outcome.Reason, uint64(len(distinct)))
		synchronizeCampaignStatistics(&summary, controller.Statistics())
		artifactRelative, relErr := filepath.Rel(batchPath, published.Path)
		if relErr != nil {
			hostFailure = &HostError{Reason: "artifact_path", Err: relErr}
			controller.Stop()
			activeCancel()
			continue
		}
		run := campaign.ExecutionRecord{
			SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed),
			Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, FailureSignature: &signature,
			Artifact: &artifactRelative, ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt),
		}
		setRunTranscript(&run, completion.result.IOTranscript)
		setRunChoiceTrace(&run, completion.result.ChoiceTrace)
		run.SemanticProbes = append([]string(nil), runCoverage.Probes...)
		run.ChoiceFeatures = append([]string(nil), runChoiceFeatures...)
		if err := journal.AppendExecution(run); err != nil && hostFailure == nil {
			hostFailure = &HostError{Reason: "runs_append", Err: err}
			controller.Stop()
			activeCancel()
		}
		if hostFailure == nil {
			addSemanticProbes(semanticProbes, runCoverage.Probes)
			addStrings(choiceFeatures, runChoiceFeatures)
		}
		completePartial(completion.journal)

		if cancelActive {
			activeCancel()
		}
	}

	if coverageHasSemantic(config.Coverage) {
		probes := make([]string, 0, len(semanticProbes))
		for probe := range semanticProbes {
			probes = append(probes, probe)
		}
		coverage, coverageErr := deterministicio.SummarizeSemanticProbes(probes)
		if coverageErr != nil && hostFailure == nil {
			hostFailure = &HostError{Reason: "semantic_coverage", Err: coverageErr}
		} else if coverageErr == nil {
			summary.SemanticCoverage = &coverage
		}
	}
	if overallCtx.Err() != nil && hostFailure == nil {
		hostFailure = &HostError{Reason: contextFailureReason(overallCtx.Err()), Err: overallCtx.Err()}
	}
	if hostFailure != nil {
		if summary.ChoiceTrace != nil {
			if progressErr := reportProgress(ProgressRunning, 0); progressErr != nil {
				hostFailure = errors.Join(hostFailure, &HostError{Reason: "progress_output", Err: progressErr})
			}
		}
		return summary, hostFailure
	}
	if summary.SemanticCoverage != nil {
		missing, err := deterministicio.MissingRequiredSemanticProbes(*summary.SemanticCoverage, config.RequiredSemanticProbes)
		if err != nil {
			return summary, err
		}
		if len(missing) != 0 {
			return summary, &deterministicio.MissingSemanticProbesError{Probes: missing}
		}
	}
	if err := prepared.Verify(); err != nil {
		return summary, &HostError{Reason: "prepared_target_integrity", Err: err}
	}
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: contextFailureReason(err), Err: err}
	}
	controller.Finalize()
	synchronizeCampaignStatistics(&summary, controller.Statistics())
	if err := overallCtx.Err(); err != nil {
		return summary, &HostError{Reason: contextFailureReason(err), Err: err}
	}
	failureSignatures := make([]record.SHA256, 0, len(distinct))
	for signature := range distinct {
		failureSignatures = append(failureSignatures, signature)
	}
	if err := journal.Publish(campaign.CampaignSummary{
		Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures, Watchdogs: summary.Watchdogs,
		Cancelled: summary.Cancelled, DistinctFailures: summary.DistinctFailures, RetainedSuccesses: summary.RetainedSuccesses, RetainedSuccessBytes: summary.RetainedSuccessBytes, StopReason: string(summary.StopReason), FailureSignatures: failureSignatures,
	}); err != nil {
		return summary, &HostError{Reason: "campaign_publish", Err: err}
	}
	batchComplete = true
	if err := reportProgress(ProgressComplete, 0); err != nil {
		return summary, &HostError{Reason: "progress_output", Err: err}
	}
	return summary, nil
}

func validateConfig(config CampaignSpec) (SeedSelection, []record.Environment, error) {
	if config.ResumeCampaign != "" {
		if config.Preparer != nil {
			return SeedSelection{}, nil, fmt.Errorf("campaign resume does not accept target preparation")
		}
		if config.RunnerBuild == "" {
			return SeedSelection{}, nil, fmt.Errorf("Runner build identity is required for campaign resume")
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
	strategy := config.Strategy
	if strategy == "" {
		strategy = StrategySeed
	}
	shard := normalizedCampaignShard(config.Shard)
	if err := shard.Validate(); err != nil {
		return SeedSelection{}, nil, err
	}
	if config.Shard.Count != 0 {
		if config.PlanSHA256 == "" {
			return SeedSelection{}, nil, errors.New("sharded campaign requires a canonical plan identity")
		}
		if _, err := record.ParseSHA256(string(config.PlanSHA256)); err != nil {
			return SeedSelection{}, nil, fmt.Errorf("canonical plan identity: %w", err)
		}
		if strategy != StrategySeed || config.Guide {
			return SeedSelection{}, nil, errors.New("static sharding requires an unguided seed campaign")
		}
		if config.OnFailure != PolicyAll {
			return SeedSelection{}, nil, errors.New("sharded campaign requires on-failure=all")
		}
	}
	switch strategy {
	case StrategySeed:
		if config.MaxExecutions != 0 || config.MaxChoiceDepth != 0 || config.MaxForcedDecisions != 0 || config.MaxExplorationBytes != 0 || config.MaxExplorationResultBytes != 0 || config.SimulationDimensionLimits != (SimulationDimensionLimits{}) {
			return SeedSelection{}, nil, errors.New("exploration bounds require the choice-exploration strategy")
		}
	case StrategyChoiceExploration:
		if config.MaxForcedDecisions != 0 || config.MaxExplorationResultBytes != 0 || config.SimulationDimensionLimits != (SimulationDimensionLimits{}) {
			return SeedSelection{}, nil, errors.New("simulation exploration bounds require the simulation-exploration strategy")
		}
		if selection.Count() != 1 {
			return SeedSelection{}, nil, errors.New("choice-exploration exploration requires exactly one base seed")
		}
		if config.Guide {
			return SeedSelection{}, nil, errors.New("choice-exploration strategy does not support guided exploration")
		}
		if config.ChoiceTraceLimit == 0 {
			return SeedSelection{}, nil, errors.New("choice-exploration strategy requires an enabled choice trace")
		}
		if config.MaxExecutions == 0 {
			return SeedSelection{}, nil, errors.New("choice-exploration max executions must be positive")
		}
		if config.MaxChoiceDepth == 0 {
			return SeedSelection{}, nil, errors.New("choice-exploration choice depth must be positive")
		}
		if config.MaxExplorationBytes == 0 {
			return SeedSelection{}, nil, errors.New("choice-exploration exploration bytes must be positive")
		}
	case StrategySimulationExploration:
		if selection.Count() != 1 {
			return SeedSelection{}, nil, errors.New("simulation-exploration exploration requires exactly one base seed")
		}
		if config.Guide {
			return SeedSelection{}, nil, errors.New("simulation-exploration strategy does not support guided exploration")
		}
		if config.ChoiceTraceLimit == 0 {
			return SeedSelection{}, nil, errors.New("simulation-exploration strategy requires an enabled choice trace")
		}
		if config.MaxExecutions == 0 {
			return SeedSelection{}, nil, errors.New("simulation-exploration max executions must be positive")
		}
		if config.MaxChoiceDepth != 0 {
			return SeedSelection{}, nil, errors.New("choice depth requires the choice-exploration strategy")
		}
		if config.MaxForcedDecisions == 0 {
			return SeedSelection{}, nil, errors.New("simulation-exploration forced decisions must be positive")
		}
		if config.MaxExplorationBytes == 0 {
			return SeedSelection{}, nil, errors.New("simulation-exploration exploration bytes must be positive")
		}
		if config.MaxExplorationResultBytes == 0 {
			return SeedSelection{}, nil, errors.New("simulation-exploration result bytes must be positive")
		}
		if err := validateSimulationDimensionLimits(config.SimulationDimensionLimits); err != nil {
			return SeedSelection{}, nil, err
		}
	default:
		return SeedSelection{}, nil, fmt.Errorf("unknown exploration strategy %q", config.Strategy)
	}
	if config.Parallel <= 0 {
		return SeedSelection{}, nil, fmt.Errorf("parallelism must be positive")
	}
	if config.ExecutionTimeout <= 0 || config.OverallTimeout <= 0 {
		return SeedSelection{}, nil, errors.New("execution and overall timeouts must be positive")
	}
	if config.TerminateGrace < 0 || config.TerminateGrace > config.ExecutionTimeout || config.TerminateGrace > config.OverallTimeout {
		return SeedSelection{}, nil, errors.New("termination grace must fit inside all deadlines")
	}
	if config.OutputLimit == 0 || config.WorldTransitionLimit == 0 {
		return SeedSelection{}, nil, errors.New("output and World transition limits must be positive")
	}
	if config.ChoiceTraceLimit != 0 && (config.ChoiceTraceLimit < execution.MinimumChoiceTraceBytes || config.ChoiceTraceLimit > execution.MaximumChoiceTraceBytes) {
		return SeedSelection{}, nil, fmt.Errorf("choice trace capacity must be between %d bytes and 64 MiB", execution.MinimumChoiceTraceBytes)
	}
	if config.Artifacts == "" || config.RunnerBuild == "" {
		return SeedSelection{}, nil, errors.New("artifact root and Runner build identity are required")
	}
	switch config.Coverage {
	case "", CoverageNone:
		if len(config.RequiredSemanticProbes) != 0 {
			return SeedSelection{}, nil, errors.New("required semantic probes require semantic coverage")
		}
	case CoverageSemantic, CoverageSemanticChoice:
		if _, err := deterministicio.MissingRequiredSemanticProbes(deterministicio.SemanticCoverage{}, config.RequiredSemanticProbes); err != nil {
			return SeedSelection{}, nil, err
		}
	case CoverageChoice:
		if len(config.RequiredSemanticProbes) != 0 {
			return SeedSelection{}, nil, errors.New("required semantic probes require semantic coverage")
		}
	default:
		return SeedSelection{}, nil, fmt.Errorf("unknown coverage mode %q", config.Coverage)
	}
	if coverageHasChoice(config.Coverage) && config.ChoiceTraceLimit == 0 {
		return SeedSelection{}, nil, errors.New("choice coverage requires an enabled choice trace")
	}
	switch normalizedKeepSuccesses(config.KeepSuccesses) {
	case KeepSuccessesNone:
		if config.SuccessArtifactLimit != 0 || config.SuccessBytesLimit != 0 {
			return SeedSelection{}, nil, errors.New("disabled success retention does not accept capacity limits")
		}
	case KeepSuccessesNovel:
		if normalizedCoverage(config.Coverage) == CoverageNone || config.SuccessArtifactLimit == 0 || config.SuccessBytesLimit == 0 {
			return SeedSelection{}, nil, errors.New("novel success retention requires coverage and explicit count and byte limits")
		}
	case KeepSuccessesAll:
		if config.SuccessArtifactLimit == 0 || config.SuccessBytesLimit == 0 {
			return SeedSelection{}, nil, errors.New("success retention requires explicit count and byte limits")
		}
	default:
		return SeedSelection{}, nil, fmt.Errorf("unknown successful-execution retention policy %q", config.KeepSuccesses)
	}
	if config.CollectExecutionEvidence && (selection.Count() != 1 || !coverageHasSemantic(config.Coverage)) {
		return SeedSelection{}, nil, errors.New("execution evidence requires exactly one seed and semantic coverage")
	}
	if config.Guide {
		if config.Corpus == "" || normalizedCoverage(config.Coverage) == CoverageNone {
			return SeedSelection{}, nil, errors.New("guided exploration requires a corpus and coverage")
		}
	} else if config.Corpus != "" || config.GuideSnapshotSHA256 != "" {
		return SeedSelection{}, nil, errors.New("a guided corpus requires guided exploration")
	}
	switch config.OnFailure {
	case PolicyFirst, PolicyAll:
		if config.FailureBudget != 1 {
			return SeedSelection{}, nil, errors.New("failure budget is only configurable in budget mode")
		}
	case PolicyBudget:
		if config.FailureBudget == 0 {
			return SeedSelection{}, nil, errors.New("failure budget must be positive")
		}
	default:
		return SeedSelection{}, nil, fmt.Errorf("unknown failure policy %q", config.OnFailure)
	}
	if config.Executor == nil && len(config.SupervisorCommand) == 0 {
		return SeedSelection{}, nil, errors.New("supervisor command is required")
	}
	if len(config.IOROMounts) != 0 {
		if config.Target.WorkingDir == "" {
			return SeedSelection{}, nil, errors.New("read-only mounts require a target working directory")
		}
		if _, err := readonlymount.ParseMappings(config.IOROMounts, config.Target.WorkingDir); err != nil {
			return SeedSelection{}, nil, err
		}
		limits := config.IOROMountLimits
		if limits == (readonlymount.Limits{}) {
			limits = readonlymount.DefaultLimits()
		}
		if _, err := readonlymount.Prepare(nil, limits); err != nil {
			return SeedSelection{}, nil, err
		}
	}
	environment, err := parseEnvironment(config.Environment)
	if err != nil {
		return SeedSelection{}, nil, err
	}
	environment = append(environment, record.Environment{Name: "GOMAD3_IO_PROFILE", Value: deterministicio.Deterministic})
	if config.ChoiceTraceLimit != 0 {
		environment = append(environment, record.Environment{Name: "GOMAD3_CHOICE_PROFILE", Value: choice.Profile})
	}
	sort.Slice(environment, func(i, j int) bool { return environment[i].Name < environment[j].Name })
	return selection, environment, nil
}

func validateSimulationDimensionLimits(limits SimulationDimensionLimits) error {
	for _, dimension := range []struct {
		name  string
		limit uint64
	}{
		{name: "runtime", limit: limits.Runtime},
		{name: "scenario", limit: limits.Scenario},
		{name: "network", limit: limits.Network},
		{name: "storage", limit: limits.Storage},
		{name: "fault", limit: limits.Fault},
		{name: "crash", limit: limits.Crash},
	} {
		if dimension.limit == 0 {
			return fmt.Errorf("simulation-exploration %s dimension bound must be positive", dimension.name)
		}
	}
	return nil
}

func parseEnvironment(entries []string) ([]record.Environment, error) {
	reserved := map[string]struct{}{
		"GOMADSEED": {}, "GOMAD3_CHILD_SEED": {}, "GOMAD3_IO_PROFILE": {}, "GOMAD3_CHOICE_PROFILE": {}, "GOMAD3_CHOICE_MODE": {}, "GOMAD3_CHOICE_TRACE_FD": {}, "GOMAD3_CHOICE_TERMINAL_FD": {}, "GOMAD3_CHOICE_TRACE_BYTES": {}, "GOMAD3_CHOICE_TAPE_FD": {}, "GOMAD3_CHOICE_TAPE_BYTES": {}, "GOMAD3_SIMULATION_ROLE": {}, "GOMAD3_SIMULATION_REQUEST_FD": {}, "GOMAD3_SIMULATION_RESPONSE_FD": {}, "GOMAD3_SIMULATION_BOOTSTRAP_FD": {}, "GOMAD3_SIMULATION_CONTROL_FD": {}, "TZ": {}, "CGO_ENABLED": {}, "GODEBUG": {}, "GOMAXPROCS": {}, "GOEXPERIMENT": {},
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

func runSeed(ctx context.Context, config CampaignSpec, executor Executor, prepared target.Prepared, baseEnvironment []record.Environment, profile deterministicio.Spec, readOnlyMounts []readonlymount.Mapping, journal runJournalFactory, job runJob, readiness *runReadiness, completions chan<- runCompletion) {
	defer readiness.signal()
	startedAt := time.Now().UTC()
	run, err := journal.BeginExecution(job.ordinal, job.seed)
	completion := runCompletion{job: job, startedAt: startedAt, journal: run}
	if err != nil {
		completion.err = fmt.Errorf("create per-seed partial directory: %w", err)
		completion.finishedAt = time.Now().UTC()
		completions <- completion
		return
	}
	if err := run.Transition(campaign.ExecutionStarting); err != nil {
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
	var choiceCapability *execution.ChoiceCapability
	if completion.err == nil {
		choiceCapability, completion.err = choiceCapabilityForJob(config, prepared, job)
	}
	var simulationCapability *execution.SimulationCapability
	if completion.err == nil {
		simulationCapability, completion.err = simulationCapabilityForJob(executor, job)
	}
	if completion.err == nil {
		readiness.signal()
		request := execution.Spec{
			SupervisorCommand: append([]string(nil), config.SupervisorCommand...), Command: prepared.Path, Args: arguments, Argv0: prepared.Argv[0],
			Dir: run.WorkPath(), Env: environmentStrings(environment), ExecutionTimeout: config.ExecutionTimeout,
			TerminateGrace: config.TerminateGrace, OutputLimit: config.OutputLimit,
			World: execution.WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: config.WorldTransitionLimit, Seed: job.seed},
			IO: &execution.IOCapability{
				Config:     append([]byte(nil), ioConfig...),
				Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20},
				ReadOnlyMount: &execution.ReadOnlyMountCapability{
					Mappings: append([]readonlymount.Mapping(nil), readOnlyMounts...), Limits: config.IOROMountLimits,
				},
			},
			StdoutHead: stdoutHead, StderrHead: stderrHead,
		}
		request.Simulation = simulationCapability
		request.Choice = choiceCapability
		if len(config.SupervisorCommand) != 0 {
			request.BootstrapCommand = []string{config.SupervisorCommand[0], "__target_bootstrap"}
		}
		completion.result, completion.err = executor.Run(ctx, request)
		if completion.err == nil {
			completion.err = validateObservedChoiceTrace(config.ChoiceTraceLimit, choiceCapability, &completion.result.ChoiceTrace)
		}
	}
	if partialErr := run.Transition(campaign.ExecutionExited); partialErr != nil {
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
		if err := run.Transition(campaign.ExecutionCaptured); err != nil {
			completion.err = err
		}
	}
	completions <- completion
}

func simulationCapabilityForJob(executor Executor, job runJob) (*execution.SimulationCapability, error) {
	if job.simulationPlan == "" {
		if job.simulationRecordLimit != 0 || job.simulationRecordCount != 0 {
			return nil, errors.New("simulation exploration record bounds require a plan")
		}
		if _, ok := executor.(processExecutor); ok {
			return &execution.SimulationCapability{Role: execution.SimulationRoleCoordinator}, nil
		}
		return nil, nil
	}
	if job.simulationRecordLimit == 0 || job.simulationRecordCount == 0 {
		return nil, errors.New("simulation exploration plan requires record bounds")
	}
	return &execution.SimulationCapability{
		Role: execution.SimulationRoleCoordinator, ExplorationPlan: []byte(job.simulationPlan),
		ExplorationRecordLimit: job.simulationRecordLimit, ExplorationRecordCount: job.simulationRecordCount,
	}, nil
}

func choiceCapabilityForJob(config CampaignSpec, prepared target.Prepared, job runJob) (*execution.ChoiceCapability, error) {
	if config.ChoiceTraceLimit == 0 {
		return nil, nil
	}
	implementation, err := choice.ImplementationIdentity(prepared.BuildKey)
	if err != nil {
		return nil, fmt.Errorf("derive choice profile implementation identity: %w", err)
	}
	identity, err := choiceExecutionIdentity(prepared, implementation)
	if err != nil {
		return nil, err
	}
	mode := job.choiceMode
	if mode == 0 {
		mode = choice.ModeRecord
	}
	capability := &execution.ChoiceCapability{
		Mode: mode, Profile: choice.Profile, ImplementationSHA256: implementation,
		ExecutionIdentity: identity, Limit: config.ChoiceTraceLimit,
	}
	if job.choiceReplayPlan != nil {
		replayPlan := *job.choiceReplayPlan
		capability.ReplayPlan = &replayPlan
	}
	return capability, nil
}

func validateObservedChoiceTrace(limit uint64, capability *execution.ChoiceCapability, observed *execution.ChoiceTrace) error {
	if limit == 0 {
		return nil
	}
	if observed == nil || observed.Profile == "" || observed.Trace.Summary.Terminal == 0 {
		return execution.ErrChoiceTraceUnterminated
	}
	if capability == nil || observed.Profile != choice.Profile || observed.ImplementationSHA256 != capability.ImplementationSHA256 || observed.Limit != limit || observed.Trace.Summary.Terminal != choice.TerminalComplete {
		return execution.ErrChoiceTraceMalformed
	}
	tape, err := choice.ProjectReplayPlan(observed.Trace, capability.ExecutionIdentity)
	if err != nil {
		return errors.Join(execution.ErrChoiceTraceMalformed, err)
	}
	observed.TapeSHA256 = tape.SHA256
	observed.Decisions = uint64(len(tape.Decisions))
	return nil
}

func choiceExecutionIdentity(prepared target.Prepared, implementation [32]byte) (choice.ExecutionIdentity, error) {
	targetIdentity, err := record.ParseSHA256(prepared.SHA256)
	if err != nil {
		return choice.ExecutionIdentity{}, fmt.Errorf("decode choice target identity: %w", err)
	}
	targetSHA256, err := targetIdentity.Bytes()
	if err != nil {
		return choice.ExecutionIdentity{}, fmt.Errorf("decode choice target identity: %w", err)
	}
	return choice.ExecutionIdentity{
		TargetSHA256: targetSHA256, ToolchainBuildKey: prepared.BuildKey,
		GOOS: prepared.TargetGOOS, GOARCH: prepared.TargetGOARCH, ImplementationSHA256: implementation,
	}, nil
}

func environmentForSeed(base []record.Environment, seed uint64) []record.Environment {
	environment := append([]record.Environment(nil), base...)
	environment = append(environment, record.Environment{Name: "GOMADSEED", Value: strconv.FormatUint(seed, 10)}, record.Environment{Name: "TZ", Value: "UTC"})
	sort.Slice(environment, func(i, j int) bool { return environment[i].Name < environment[j].Name })
	return environment
}

func environmentStrings(environment []record.Environment) []string {
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		if entry.Name == "GOMAD3_CHOICE_PROFILE" {
			continue
		}
		result = append(result, entry.Name+"="+entry.Value)
	}
	return result
}

func manifestForRun(config CampaignSpec, prepared target.Prepared, baseEnvironment []record.Environment, completion runCompletion, outcome execution.Classification, runID string, recordedWorld record.World, mountArtifact *readonlymount.CapturedInputs) (record.ExecutionRecord, error) {
	profile := deterministicio.Default()
	recordedProfile := recordedIOProfile(profile)
	if completion.result.IOTranscript.Complete {
		recordedProfile.Transcript = &record.IOTranscript{
			Schema: "gomad3.io-transcript/v1", File: "io/transcript.bin", SHA256: record.SHA256FromSum(completion.result.IOTranscript.SHA256),
			Bytes: record.Uint64String(len(completion.result.IOTranscript.Bytes)), Records: record.Uint64String(completion.result.IOTranscript.Records),
		}
	}
	if mountArtifact != nil {
		mounts := recordedCapturedInputs(mountArtifact.Manifest)
		recordedProfile.ReadOnlyMounts = &mounts
	}
	var recordedChoices *record.ChoiceProfile
	if config.ChoiceTraceLimit != 0 {
		implementation, err := choice.ImplementationIdentity(prepared.BuildKey)
		if err != nil {
			return record.ExecutionRecord{}, fmt.Errorf("derive choice profile implementation identity: %w", err)
		}
		observed := completion.result.ChoiceTrace
		expectedTerminal := choice.TerminalComplete
		terminalState := "complete"
		if outcome.ArtifactKind == record.ArtifactRunnerFailure && outcome.Reason == "choice_trace_overflow" {
			expectedTerminal = choice.TerminalOverflow
			terminalState = "overflow"
		}
		if observed.Profile != choice.Profile || observed.ImplementationSHA256 != implementation || observed.Limit != config.ChoiceTraceLimit || observed.Trace.Summary.Terminal != expectedTerminal {
			return record.ExecutionRecord{}, errors.New("enabled choice profile did not produce the required terminal trace")
		}
		recordedChoices = &record.ChoiceProfile{
			Name: choice.Profile, ImplementationSHA256: record.SHA256FromSum(implementation),
			Trace: record.ChoiceTrace{
				Schema: "gomad3.choice-trace/v2", File: "choices.bin", SHA256: record.SHA256FromSum(observed.Trace.SHA256),
				Bytes: record.Uint64String(len(observed.Trace.Bytes)), Records: record.Uint64String(observed.Trace.Summary.Records),
				BranchingRecords: record.Uint64String(observed.Trace.Summary.Branching), TerminalState: terminalState, Limit: record.Uint64String(observed.Limit),
			},
		}
		if expectedTerminal == choice.TerminalComplete {
			identity, identityErr := choiceExecutionIdentity(prepared, implementation)
			if identityErr != nil {
				return record.ExecutionRecord{}, identityErr
			}
			tape, tapeErr := choice.ProjectReplayPlan(observed.Trace, identity)
			if tapeErr != nil {
				return record.ExecutionRecord{}, fmt.Errorf("derive choice tape identity: %w", tapeErr)
			}
			recordedChoices.Trace.TapeSHA256 = record.SHA256FromSum(tape.SHA256)
			recordedChoices.Trace.Decisions = record.Uint64String(len(tape.Decisions))
		}
	}
	return record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: outcome.ArtifactKind, CreatedAt: completion.finishedAt.Format(time.RFC3339Nano), CampaignID: runID,
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed), ReplayMode: outcome.ReplayMode,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: config.RunnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: record.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Target: record.Target{
			Kind: string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
			Argv: append([]string{}, prepared.Argv...), BuildTags: append([]string{}, prepared.BuildTags...), Adapters: cloneAdapters(prepared.Adapters), Compatibility: cloneCompatibility(prepared.Compatibility), BuildInfo: prepared.BuildInfo,
		},
		IOProfile:     recordedProfile,
		ChoiceProfile: recordedChoices,
		Environment:   environmentForSeed(baseEnvironment, completion.job.seed),
		Limits: record.Limits{
			ExecutionTimeoutNanos: record.Uint64String(config.ExecutionTimeout), OverallTimeoutNanos: record.Uint64String(config.OverallTimeout),
			TerminateGraceNanos: record.Uint64String(config.TerminateGrace), OutputBytes: record.Uint64String(config.OutputLimit),
			WorldTransitionBytes: record.Uint64String(config.WorldTransitionLimit),
			IOTranscriptBytes:    64 << 20,
			ChoiceTraceBytes:     record.Uint64String(config.ChoiceTraceLimit),
		},
		World:   recordedWorld,
		Outcome: record.Outcome{Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, ExitCode: outcome.ExitCode, Signal: outcome.Signal, Deadline: outcome.Deadline},
		Streams: record.Streams{Stdout: streamRecord(completion.result.Stdout), Stderr: streamRecord(completion.result.Stderr)},
		Host:    record.Host{StartedAt: completion.startedAt.Format(time.RFC3339Nano), FinishedAt: completion.finishedAt.Format(time.RFC3339Nano), ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt)},
	}, nil
}

func mountArtifactForRun(mappings []readonlymount.Mapping, limits readonlymount.Limits, snapshot readonlymount.Snapshot) (*readonlymount.CapturedInputs, error) {
	if len(mappings) == 0 {
		return nil, nil
	}
	encoded, err := readonlymount.EncodeCapturedInputs(mappings, limits, snapshot)
	if err != nil {
		return nil, err
	}
	return &encoded, nil
}

func setRunTranscript(run *campaign.ExecutionRecord, transcript deterministicio.Transcript) {
	if !transcript.Complete {
		return
	}
	digest := record.SHA256FromSum(transcript.SHA256)
	records := record.Uint64String(transcript.Records)
	run.IOTranscriptSHA256 = &digest
	run.IOTranscriptRecords = &records
}

func setRunChoiceTrace(run *campaign.ExecutionRecord, trace execution.ChoiceTrace) {
	if trace.Profile == "" || trace.Trace.Summary.Terminal != choice.TerminalComplete && trace.Trace.Summary.Terminal != choice.TerminalOverflow {
		return
	}
	digest := record.SHA256FromSum(trace.Trace.SHA256)
	records := record.Uint64String(trace.Trace.Summary.Records)
	branching := record.Uint64String(trace.Trace.Summary.Branching)
	terminal := choiceTerminalState(trace.Trace.Summary.Terminal)
	run.ChoiceTraceSHA256 = &digest
	run.ChoiceTraceRecords = &records
	run.ChoiceTraceBranchingRecords = &branching
	run.ChoiceTraceTerminalState = &terminal
	if trace.TapeSHA256 != ([32]byte{}) {
		tapeSHA256 := record.SHA256FromSum(trace.TapeSHA256)
		decisions := record.Uint64String(trace.Decisions)
		run.ChoiceTapeSHA256 = &tapeSHA256
		run.ChoiceDecisions = &decisions
	}
}

func supervisionFailureReason(err error) string {
	switch {
	case errors.Is(err, execution.ErrChoiceTraceOverflow):
		return "choice_trace_overflow"
	case errors.Is(err, execution.ErrChoiceTraceMalformed):
		return "choice_trace_malformed"
	case errors.Is(err, execution.ErrChoiceTraceUnterminated):
		return "choice_trace_unterminated"
	default:
		return "target_supervision"
	}
}

func choiceTraceSummary(seed uint64, trace execution.ChoiceTrace) *ChoiceTraceSummary {
	summary := &ChoiceTraceSummary{
		Seed: seed, Profile: trace.Profile, Limit: trace.Limit, SHA256: record.SHA256FromSum(trace.Trace.SHA256),
		Records: trace.Trace.Summary.Records, BranchingRecords: trace.Trace.Summary.Branching,
		Runnable: trace.Trace.Summary.Runnable, SelectPoll: trace.Trace.Summary.SelectPoll, SelectResult: trace.Trace.Summary.SelectResult,
		TerminalState: choiceTerminalState(trace.Trace.Summary.Terminal), Decisions: trace.Decisions,
	}
	if trace.TapeSHA256 != ([32]byte{}) {
		summary.TapeSHA256 = record.SHA256FromSum(trace.TapeSHA256)
	}
	return summary
}

func choiceTerminalState(state choice.TerminalState) string {
	switch state {
	case choice.TerminalComplete:
		return "complete"
	case choice.TerminalOverflow:
		return "overflow"
	default:
		return "unknown"
	}
}

func cloneChoiceTraceSummary(summary *ChoiceTraceSummary) *ChoiceTraceSummary {
	if summary == nil {
		return nil
	}
	cloned := *summary
	return &cloned
}

func cloneChoiceExplorationSummary(summary *ChoiceExplorationSummary) *ChoiceExplorationSummary {
	if summary == nil {
		return nil
	}
	cloned := *summary
	return &cloned
}

func cloneSimulationExplorationSummary(summary *SimulationExplorationSummary) *SimulationExplorationSummary {
	if summary == nil {
		return nil
	}
	cloned := *summary
	return &cloned
}

func projectChoiceExplorationSummary(summary choiceengine.Summary) ChoiceExplorationSummary {
	return ChoiceExplorationSummary{
		Parallel: summary.Parallel, MaxExecutions: summary.MaxExecutions, MaxChoiceDepth: summary.MaxChoiceDepth,
		MaxExplorationBytes: summary.MaxExplorationBytes, LogicalExecutions: summary.LogicalExecutions,
		CommittedRounds: summary.CommittedRounds, Pending: summary.Pending, PendingBytes: summary.PendingBytes,
		SeenPrefixes: summary.SeenPrefixes, DeduplicatedOutcomes: summary.DeduplicatedOutcomes, DeepestPrefix: summary.DeepestPrefix,
		OmittedByExecutionBound: summary.OmittedByExecutionBound, OmittedByDepth: summary.OmittedByDepth,
		OmittedByCapacity: summary.OmittedByCapacity, StopReason: string(summary.StopReason), BoundedComplete: summary.BoundedComplete,
	}
}

func projectChoiceExplorationSummaryPointer(summary *choiceengine.Summary) *ChoiceExplorationSummary {
	if summary == nil {
		return nil
	}
	projected := projectChoiceExplorationSummary(*summary)
	return &projected
}

func projectSimulationExplorationSummary(summary simulationengine.Summary) SimulationExplorationSummary {
	return SimulationExplorationSummary{
		Parallel: summary.Parallel, MaxExecutions: summary.MaxExecutions, MaxForcedDecisions: summary.MaxForcedDecisions,
		MaxExplorationBytes: summary.MaxExplorationBytes, MaxResultBytes: summary.MaxResultBytes, FailureBudget: summary.FailureBudget,
		Limits: SimulationDimensionLimits(summary.Limits), LogicalExecutions: summary.LogicalExecutions,
		CommittedRounds: summary.CommittedRounds, Pending: summary.Pending, PendingBytes: summary.PendingBytes,
		SeenCandidates: summary.SeenCandidates, DeduplicatedOutcomes: summary.DeduplicatedOutcomes, DistinctFailures: summary.DistinctFailures,
		DeepestOverride: summary.DeepestOverride, OmittedByExecutionBound: summary.OmittedByExecutionBound,
		OmittedByDepth: summary.OmittedByDepth, OmittedByDimension: summary.OmittedByDimension,
		OmittedByCapacity: summary.OmittedByCapacity, StopReason: string(summary.StopReason), BoundedComplete: summary.BoundedComplete,
	}
}

func projectSimulationExplorationSummaryPointer(summary *simulationengine.Summary) *SimulationExplorationSummary {
	if summary == nil {
		return nil
	}
	projected := projectSimulationExplorationSummary(*summary)
	return &projected
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

func preservePartial(run *campaign.ExecutionJournal) error {
	if run == nil {
		return nil
	}
	return run.Preserve()
}

func publishBoundedFailureArtifact(
	ctx context.Context,
	config CampaignSpec,
	root string,
	signature record.SHA256,
	distinct map[record.SHA256]string,
	storedBytes *uint64,
	input artifact.ArtifactInput,
) (artifact.Artifact, error) {
	store := artifact.Store{Root: root, Context: ctx}
	_, existing := distinct[signature]
	if !existing && config.failureArtifactLimit != 0 && uint64(len(distinct)) == config.failureArtifactLimit {
		return artifact.Artifact{}, &campaign.ArtifactCapacityError{
			Limit: campaign.ArtifactLimitFailureCount, Required: uint64(len(distinct)) + 1,
			Maximum: config.failureArtifactLimit, Outcome: campaign.CapacityInfrastructureFailure,
		}
	}
	if !existing && config.failureBytesLimit != 0 {
		if *storedBytes >= config.failureBytesLimit {
			return artifact.Artifact{}, &campaign.ArtifactCapacityError{
				Limit: campaign.ArtifactLimitFailureBytes, Required: *storedBytes + 1,
				Maximum: config.failureBytesLimit, Outcome: campaign.CapacityInfrastructureFailure,
			}
		}
		store.MaximumBytes = config.failureBytesLimit - *storedBytes
	}
	published, err := artifact.PublishArtifact(store, input)
	if err != nil {
		var capacity *artifact.CapacityError
		if !existing && errors.As(err, &capacity) {
			required := *storedBytes + capacity.Required
			if required < *storedBytes {
				required = ^uint64(0)
			}
			return artifact.Artifact{}, &campaign.ArtifactCapacityError{
				Limit: campaign.ArtifactLimitFailureBytes, Required: required,
				Maximum: config.failureBytesLimit, Outcome: campaign.CapacityInfrastructureFailure,
			}
		}
		return artifact.Artifact{}, err
	}
	if !existing {
		*storedBytes += published.StoredBytes
	}
	return published, nil
}

func streamRecord(output hostexec.Output) record.Stream {
	return record.Stream{
		RetainedSHA256: record.SHA256FromSum(output.RetainedSHA256), FullSHA256: record.SHA256FromSum(output.FullSHA256), TotalBytes: record.Uint64String(output.TotalBytes),
		RetainedBytes: record.Uint64String(output.RetainedBytes), DiscardedBytes: record.Uint64String(output.DiscardedBytes), Truncated: output.Truncated,
	}
}

func noneWorldBundle() execution.Bundle {
	manifest, payloads := record.NoneWorld()
	return execution.Bundle{Manifest: manifest, Payloads: payloads}
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
	return "campaign-" + time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(random), nil
}
