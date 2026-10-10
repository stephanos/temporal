package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
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
	"go.temporal.io/server/tools/gomad3/runner/backend"
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
	StopChoiceStartUnreached    StopReason = "choice_start_unreached"
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
	Guidance              *GuidanceSummary `json:"guidance,omitempty"`
	ChoiceTrace           *ChoiceTraceSummary
	ChoiceExploration     *ChoiceExplorationSummary
	SimulationExploration *SimulationExplorationSummary
	RecoveryExecutions    uint64
}

type CampaignEventFunc func(CampaignEvent) error

type Preparer interface {
	Prepare(context.Context, target.Spec) (target.Prepared, error)
}

type executionRunner interface {
	Run(context.Context, execution.Spec) (execution.Result, error)
}

type ArtifactReplayer interface {
	Replay(context.Context, ReplaySpec) (ReplayResult, error)
}

type CampaignSpec struct {
	Backend              backend.Provider `json:"-"`
	ResumeCampaign       string
	PlanSHA256           record.SHA256
	Shard                CampaignShard
	Strategy             Strategy
	Seeds                string
	Parallel             int
	ExecutionTimeout     time.Duration
	OverallTimeout       time.Duration
	TerminateGrace       time.Duration
	OnFailure            FailurePolicy
	FailureBudget        uint64
	OutputLimit          uint64
	WorldTransitionLimit uint64
	Diagnostics          bool
	ChoiceTraceLimit     uint64
	// IOTranscriptLimit bounds the I/O transcript; zero means
	// deterministicio.DefaultTranscriptBytes.
	IOTranscriptLimit uint64
	// ClockTick is the virtual-clock tick policy; empty means record.ClockTickStrict.
	ClockTick                 string
	MaxExecutions             uint64
	MaxChoiceDepth            uint64
	ChoiceStartOrdinal        uint64
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
	GuideRegression           bool
	GuideRegressionOverride   *bool
	guidancePlan              *campaign.GuidancePlan
	Corpus                    string
	GuideSnapshotSHA256       record.SHA256
	Progress                  CampaignEventFunc
	ProgressInterval          time.Duration
	Preparer                  Preparer
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
	ExecutionElapsedNanos uint64
	Diagnostics           *DiagnosticTraceReference `json:"diagnostics,omitempty"`
	CorpusPath            string
	CorpusEntries         uint64
	CorpusAdded           uint64
	Guidance              *GuidanceSummary `json:"guidance,omitempty"`
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
	PeakGoroutines   uint32
}

type ChoiceExplorationSummary struct {
	Parallel                 int    `json:"parallel"`
	MaxExecutions            uint64 `json:"max_executions"`
	MaxChoiceDepth           uint64 `json:"max_choice_depth"`
	StartOrdinal             uint64 `json:"start_ordinal,omitempty"`
	MaxExplorationBytes      uint64 `json:"max_exploration_bytes"`
	LogicalExecutions        uint64 `json:"logical_executions"`
	CommittedRounds          uint64 `json:"committed_rounds"`
	Pending                  uint64 `json:"pending"`
	PendingBytes             uint64 `json:"pending_bytes"`
	SeenPrefixes             uint64 `json:"seen_prefixes"`
	DeduplicatedOutcomes     uint64 `json:"deduplicated_outcomes"`
	DeepestPrefix            uint64 `json:"deepest_prefix"`
	OmittedByExecutionBound  uint64 `json:"omitted_by_execution_bound"`
	OmittedByDepth           uint64 `json:"omitted_by_depth"`
	OmittedByCapacity        uint64 `json:"omitted_by_capacity"`
	OmittedBySelectReadiness uint64 `json:"omitted_by_select_readiness"`
	StopReason               string `json:"stop_reason,omitempty"`
	BoundedComplete          bool   `json:"bounded_complete"`
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

type artifactReplayer struct {
	dependencies executionDependencies
}

func (replayer artifactReplayer) Replay(ctx context.Context, config ReplaySpec) (ReplayResult, error) {
	return replayWith(ctx, config, replayer.dependencies)
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
	return exploreWith(ctx, config, executionDependencies{})
}

func exploreWith(ctx context.Context, config CampaignSpec, dependencies executionDependencies) (CampaignResult, error) {
	request, err := campaignRequestForExplore(config, dependencies)
	if err != nil {
		return CampaignResult{}, err
	}
	if len(request.CoordinatorCommand) != 0 {
		if request.Preparer != nil || request.injected() || request.Replayer != nil || request.Backend != nil {
			return CampaignResult{}, fmt.Errorf("isolated Runner does not accept injected preparation or execution")
		}
		return runIsolated(ctx, request)
	}
	return runLocal(ctx, request)
}

func validateConfig(config CampaignSpec) (SeedSelection, []record.Environment, error) {
	return validateCampaignRequest(campaignRequestFromSpec(config))
}

func validateCampaignRequest(config campaignRequest) (SeedSelection, []record.Environment, error) {
	if config.ResumeCampaign != "" {
		if config.Preparer != nil {
			return SeedSelection{}, nil, fmt.Errorf("campaign resume does not accept target preparation")
		}
		if config.RunnerBuild == "" {
			return SeedSelection{}, nil, fmt.Errorf("Runner build identity is required for campaign resume")
		}
		if config.executor == nil && config.Backend == nil && len(config.SupervisorCommand) == 0 {
			return SeedSelection{}, nil, fmt.Errorf("supervisor command is required")
		}
		return SeedSelection{}, nil, nil
	}
	selection, err := ParseSeeds(config.Seeds)
	if config.guidancePlan != nil {
		selection, err = parseCampaignSelection(config.Seeds, uint64(config.guidancePlan.RequestedCount-config.guidancePlan.AnsweredCount))
	}
	if err != nil {
		return SeedSelection{}, nil, err
	}
	strategy := config.strategy()
	shard := config.shard()
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
		if strategy != StrategySeed || config.Guide && config.guidancePlan == nil {
			return SeedSelection{}, nil, errors.New("static sharding requires a seed campaign with frozen guidance")
		}
		if config.OnFailure != PolicyAll {
			return SeedSelection{}, nil, errors.New("sharded campaign requires on-failure=all")
		}
	}
	switch strategy {
	case StrategySeed:
		if config.MaxExecutions != 0 || config.MaxChoiceDepth != 0 || config.ChoiceStartOrdinal != 0 || config.MaxForcedDecisions != 0 || config.MaxExplorationBytes != 0 || config.MaxExplorationResultBytes != 0 || config.SimulationDimensionLimits != (SimulationDimensionLimits{}) {
			return SeedSelection{}, nil, errors.New("exploration bounds require the choice-exploration strategy")
		}
	case StrategyChoiceExploration:
		if config.MaxForcedDecisions != 0 || config.MaxExplorationResultBytes != 0 || config.SimulationDimensionLimits != (SimulationDimensionLimits{}) {
			return SeedSelection{}, nil, errors.New("simulation exploration bounds require the simulation-exploration strategy")
		}
		if err := requireSingleBaseSeed(selection); err != nil {
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
		if err := requireSingleBaseSeed(selection); err != nil {
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
		if config.ChoiceStartOrdinal != 0 {
			return SeedSelection{}, nil, errors.New("choice start ordinal requires the choice-exploration strategy")
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
	if config.Diagnostics && (config.ChoiceTraceLimit == 0 || config.strategy() != StrategySeed) {
		return SeedSelection{}, nil, errors.New("diagnostics require choice recording with the seed strategy")
	}
	if err := ValidateChoiceTraceLimit(config.ChoiceTraceLimit); err != nil {
		return SeedSelection{}, nil, err
	}
	if config.Artifacts == "" || config.RunnerBuild == "" {
		return SeedSelection{}, nil, errors.New("artifact root and Runner build identity are required")
	}
	if err := ValidateCoverage(config.Coverage, config.RequiredSemanticProbes); err != nil {
		return SeedSelection{}, nil, err
	}
	if err := ValidateChoiceCoverage(config.Coverage, config.ChoiceTraceLimit); err != nil {
		return SeedSelection{}, nil, err
	}
	switch normalizedKeepSuccesses(config.KeepSuccesses) {
	case KeepSuccessesNone:
		if config.SuccessArtifactLimit != 0 || config.SuccessBytesLimit != 0 {
			return SeedSelection{}, nil, errors.New("disabled success retention does not accept capacity limits")
		}
	case KeepSuccessesNovel:
		if NormalizeCoverage(config.Coverage, false) == CoverageNone || config.SuccessArtifactLimit == 0 || config.SuccessBytesLimit == 0 {
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
		if config.Corpus == "" || NormalizeCoverage(config.Coverage, false) == CoverageNone {
			return SeedSelection{}, nil, errors.New("guided exploration requires a corpus and coverage")
		}
	} else if config.GuideRegression || config.Corpus != "" || config.GuideSnapshotSHA256 != "" {
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
	if config.executor == nil && config.Backend == nil && len(config.SupervisorCommand) == 0 {
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
	ioProfile := deterministicio.Deterministic
	if config.Target.Backend != "" {
		if config.Backend == nil || config.Preparer != nil || config.strategy() != StrategySeed || NormalizeCoverage(config.Coverage, false) != CoverageNone || config.Guide || len(config.IOROMounts) != 0 || config.ClockTick != "" || config.CollectExecutionEvidence {
			return SeedSelection{}, nil, errors.New("external backend request requires its provider and a supported observed seed profile")
		}
		ioProfile = config.Target.Backend
	} else if config.Backend != nil {
		return SeedSelection{}, nil, errors.New("backend provider requires an explicit backend selector")
	}
	environment = append(environment, record.Environment{Name: "GOMAD3_IO_PROFILE", Value: ioProfile})
	if err := deterministicio.ValidateTranscriptLimit(ioTranscriptLimit(config)); err != nil {
		return SeedSelection{}, nil, err
	}
	switch config.ClockTick {
	case "", record.ClockTickStrict:
	case record.ClockTickForward:
		environment = append(environment, record.Environment{Name: record.ClockTickEnvironment, Value: record.ClockTickForward})
	default:
		return SeedSelection{}, nil, fmt.Errorf("clock tick policy %q must be strict or forward", config.ClockTick)
	}
	if config.ChoiceTraceLimit != 0 {
		environment = append(environment, record.Environment{Name: "GOMAD3_CHOICE_PROFILE", Value: choice.Profile})
	}
	if config.Diagnostics {
		environment = append(environment, record.Environment{Name: choice.DiagnosticProfileEnvironment, Value: choice.DiagnosticProfile})
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
		choice.DiagnosticProfileEnvironment: {}, "GOMAD3_DIAGNOSTIC_TRACE_FD": {}, "GOMAD3_DIAGNOSTIC_TRACE_BYTES": {}, "GOMAD3_DIAGNOSTIC_PERTURB_DRAW": {}, "GOMADSEED": {}, "GOMAD3_CHILD_SEED": {}, "GOMAD3_IO_PROFILE": {}, record.ClockTickEnvironment: {}, "GOMAD3_CHOICE_PROFILE": {}, "GOMAD3_CHOICE_MODE": {}, "GOMAD3_CHOICE_TRACE_FD": {}, "GOMAD3_CHOICE_TERMINAL_FD": {}, "GOMAD3_CHOICE_TRACE_BYTES": {}, "GOMAD3_CHOICE_TAPE_FD": {}, "GOMAD3_CHOICE_TAPE_BYTES": {}, "GOMAD3_SIMULATION_ROLE": {}, "GOMAD3_SIMULATION_REQUEST_FD": {}, "GOMAD3_SIMULATION_RESPONSE_FD": {}, "GOMAD3_SIMULATION_BOOTSTRAP_FD": {}, "GOMAD3_SIMULATION_CONTROL_FD": {}, "TZ": {}, "CGO_ENABLED": {}, "GODEBUG": {}, "GOMAXPROCS": {}, "GOEXPERIMENT": {},
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

func runSeed(ctx context.Context, config campaignRequest, executor executionRunner, prepared target.Prepared, baseEnvironment []record.Environment, profile deterministicio.Spec, readOnlyMounts []readonlymount.Mapping, journal runJournalFactory, job runJob, readiness *runReadiness, completions chan<- runCompletion) {
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
	if prepared.Backend == nil {
		ioConfig, completion.err = config.bootstrapFrame(profile, prepared, config.RunnerBuild, job.seed)
	}
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
				Transcript: &execution.IOTranscriptCapability{Limit: ioTranscriptLimit(config)},
				ReadOnlyMount: &execution.ReadOnlyMountCapability{
					Mappings: append([]readonlymount.Mapping(nil), readOnlyMounts...), Limits: config.IOROMountLimits,
				},
			},
			StdoutHead: stdoutHead, StderrHead: stderrHead,
		}
		request.Simulation = simulationCapability
		request.Choice = choiceCapability
		request.Diagnostics = config.Diagnostics
		if len(config.SupervisorCommand) != 0 {
			request.BootstrapCommand = []string{config.SupervisorCommand[0], "__target_bootstrap"}
		}
		if prepared.Backend != nil {
			completion.result, completion.err = runBackend(ctx, config.Backend, backend.Request{Target: prepared.CloneBackend(), Seed: job.seed, Environment: environmentStrings(environment), Timeout: config.ExecutionTimeout, OutputBytes: config.OutputLimit, TranscriptBytes: ioTranscriptLimit(config), Choice: backendChoiceRequest(choiceCapability), Diagnostics: config.Diagnostics}, stdoutHead, stderrHead)
		} else {
			completion.result, completion.err = executor.Run(ctx, request)
		}
		// A target the watchdog or a cancellation killed never wrote its
		// terminal choice frame; that termination is the outcome, not the
		// missing frame, so the run is classified and retained as such.
		if completion.err == nil && !completion.result.WatchdogTimeout && !completion.result.Cancelled {
			completion.err = validateObservedChoiceTrace(config.ChoiceTraceLimit, choiceCapability, &completion.result.ChoiceTrace)
			if completion.err == nil && config.Diagnostics {
				trace, err := choice.DecodeDiagnosticTrace(completion.result.DiagnosticTrace.Bytes)
				if err == nil && uint64(len(trace.Records)) != completion.result.ChoiceTrace.Trace.Summary.Records {
					err = errors.New("diagnostic and choice record counts disagree")
				}
				completion.result.DiagnosticTrace, completion.err = trace, err
			}
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

func simulationCapabilityForJob(executor executionRunner, job runJob) (*execution.SimulationCapability, error) {
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

func choiceCapabilityForJob(config campaignRequest, prepared target.Prepared, job runJob) (*execution.ChoiceCapability, error) {
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
		if entry.Name == "GOMAD3_CHOICE_PROFILE" || entry.Name == choice.DiagnosticProfileEnvironment {
			continue
		}
		result = append(result, entry.Name+"="+entry.Value)
	}
	return result
}

func manifestForRun(config campaignRequest, prepared target.Prepared, baseEnvironment []record.Environment, completion runCompletion, outcome execution.Classification, runID string, recordedWorld record.World, mountArtifact *readonlymount.CapturedInputs) (record.ExecutionRecord, error) {
	profile := deterministicio.Default()
	recordedProfile := recordedIOProfile(profile)
	if prepared.Backend != nil {
		recordedProfile = record.BackendIOProfile(*prepared.Backend)
	}
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
	// A watchdog or cancellation kills the target before it writes its
	// terminal choice frame, so such a run records no choice profile.
	if config.ChoiceTraceLimit != 0 && outcome.ArtifactKind != record.ArtifactWatchdogTimeout && outcome.Reason != "runner_cancelled" {
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
				Schema: "gomad3.choice-trace/v3", File: "choices.bin", SHA256: record.SHA256FromSum(observed.Trace.SHA256),
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
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: outcome.ArtifactKind, CreatedAt: completion.finishedAt.Format(time.RFC3339Nano), CampaignID: runID,
		SelectionOrdinal: record.Uint64String(completion.job.ordinal), Seed: record.Uint64String(completion.job.seed), ReplayMode: outcome.ReplayMode,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: config.RunnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: record.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Target: record.Target{
			Backend: recordedBackend(prepared, completion.result),
			Kind:    string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
			Argv: append([]string{}, prepared.Argv...), BuildTags: append([]string{}, prepared.BuildTags...), Adapters: cloneAdapters(prepared.Adapters), Compatibility: cloneCompatibility(prepared.Compatibility), BuildInfo: prepared.BuildInfo,
		},
		IOProfile:     recordedProfile,
		ChoiceProfile: recordedChoices,
		Environment:   environmentForSeed(baseEnvironment, completion.job.seed),
		Limits: record.Limits{
			ExecutionTimeoutNanos: record.Uint64String(config.ExecutionTimeout), OverallTimeoutNanos: record.Uint64String(config.OverallTimeout),
			TerminateGraceNanos: record.Uint64String(config.TerminateGrace), OutputBytes: record.Uint64String(config.OutputLimit),
			WorldTransitionBytes: record.Uint64String(config.WorldTransitionLimit),
			IOTranscriptBytes:    record.Uint64String(ioTranscriptLimit(config)),
			ChoiceTraceBytes:     record.Uint64String(config.ChoiceTraceLimit),
		},
		World:   recordedWorld,
		Outcome: record.Outcome{Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, ExitCode: outcome.ExitCode, Signal: outcome.Signal, Deadline: outcome.Deadline},
		Streams: record.Streams{Stdout: streamRecord(completion.result.Stdout), Stderr: streamRecord(completion.result.Stderr)},
		Host:    record.Host{StartedAt: completion.startedAt.Format(time.RFC3339Nano), FinishedAt: completion.finishedAt.Format(time.RFC3339Nano), ElapsedNanos: elapsedNanos(completion.startedAt, completion.finishedAt)},
	}
	if prepared.Backend != nil {
		manifest.Target = prepared.RecordTarget()
		manifest.Target.Backend = recordedBackend(prepared, completion.result)
	}
	return manifest, nil
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
		TerminalState: choiceTerminalState(trace.Trace.Summary.Terminal), Decisions: trace.Decisions, PeakGoroutines: trace.Trace.Summary.PeakGoroutines,
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
		Parallel: summary.Parallel, MaxExecutions: summary.MaxExecutions, MaxChoiceDepth: summary.MaxChoiceDepth, StartOrdinal: summary.StartOrdinal,
		MaxExplorationBytes: summary.MaxExplorationBytes, LogicalExecutions: summary.LogicalExecutions,
		CommittedRounds: summary.CommittedRounds, Pending: summary.Pending, PendingBytes: summary.PendingBytes,
		SeenPrefixes: summary.SeenPrefixes, DeduplicatedOutcomes: summary.DeduplicatedOutcomes, DeepestPrefix: summary.DeepestPrefix,
		OmittedByExecutionBound: summary.OmittedByExecutionBound, OmittedByDepth: summary.OmittedByDepth,
		OmittedByCapacity: summary.OmittedByCapacity, OmittedBySelectReadiness: summary.OmittedBySelectReadiness,
		StopReason: string(summary.StopReason), BoundedComplete: summary.BoundedComplete,
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

func preservePartial(run *campaign.ExecutionJournal) error {
	if run == nil {
		return nil
	}
	return run.Preserve()
}

func publishBoundedFailureArtifact(
	ctx context.Context,
	config campaignRequest,
	root string,
	signature record.SHA256,
	distinct map[record.SHA256]string,
	storedBytes *uint64,
	input artifact.ArtifactInput,
) (artifact.Artifact, error) {
	store := artifact.Store{Root: root, Context: ctx, TargetPool: artifact.TargetPool(config.Artifacts)}
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

// ioTranscriptLimit is the campaign's I/O transcript bound, defaulting to
// deterministicio.DefaultTranscriptBytes.
func ioTranscriptLimit(config campaignRequest) uint64 {
	if config.IOTranscriptLimit == 0 {
		return deterministicio.DefaultTranscriptBytes
	}
	return config.IOTranscriptLimit
}
