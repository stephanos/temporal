package runner

import (
	"fmt"
	"path/filepath"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

type campaignOptions struct {
	campaignIdentityOptions
	campaignTargetOptions
	campaignSearchOptions
	campaignResourceOptions
	campaignObservationOptions
	campaignRetentionOptions
	campaignGuidanceOptions
	effectiveStrategy Strategy
	effectiveShard    CampaignShard
}

type campaignIdentityOptions struct {
	ResumeCampaign string
	PlanSHA256     record.SHA256
	Shard          CampaignShard
	Artifacts      string
}

type campaignTargetOptions struct {
	Target          target.Spec
	Environment     []string
	IOROMounts      []string
	IOROMountLimits readonlymount.Limits
}

type campaignSearchOptions struct {
	Strategy                  Strategy
	Seeds                     string
	MaxExecutions             uint64
	MaxChoiceDepth            uint64
	ChoiceStartOrdinal        uint64
	MaxForcedDecisions        uint64
	MaxExplorationBytes       uint64
	MaxExplorationResultBytes uint64
	SimulationDimensionLimits SimulationDimensionLimits
	OnFailure                 FailurePolicy
	FailureBudget             uint64
}

type campaignResourceOptions struct {
	Parallel             int
	ExecutionTimeout     time.Duration
	OverallTimeout       time.Duration
	TerminateGrace       time.Duration
	OutputLimit          uint64
	WorldTransitionLimit uint64
}

type campaignObservationOptions struct {
	Diagnostics              bool `json:",omitempty"`
	ChoiceTraceLimit         uint64
	IOTranscriptLimit        uint64
	ClockTick                string
	Coverage                 CoverageMode
	RequiredSemanticProbes   []string
	CollectExecutionEvidence bool
	ProgressInterval         time.Duration
}

type campaignRetentionOptions struct {
	KeepSuccesses        KeepSuccesses
	SuccessArtifactLimit uint64
	SuccessBytesLimit    uint64
}

type campaignGuidanceOptions struct {
	Guide                   bool
	GuideRegression         bool
	GuideRegressionOverride *bool
	Corpus                  string
	GuideSnapshotSHA256     record.SHA256
}

type campaignRuntime struct {
	SupervisorCommand    []string
	CoordinatorCommand   []string
	RunnerBuild          string
	Progress             CampaignEventFunc
	Preparer             Preparer
	Executor             Executor
	Replayer             ArtifactReplayer
	guidancePlan         *campaign.GuidancePlan
	resumePreflight      *campaign.ResumePreflight
	failureArtifactLimit uint64
	failureBytesLimit    uint64
}

type campaignRequest struct {
	campaignOptions
	campaignRuntime
}

func campaignRequestForExplore(spec CampaignSpec) (campaignRequest, error) {
	if spec.ResumeCampaign != "" {
		path, err := filepath.Abs(spec.ResumeCampaign)
		if err != nil {
			return campaignRequest{}, fmt.Errorf("resolve resumable campaign path: %w", err)
		}
		preflight, err := campaign.PreflightResume(path)
		if err != nil {
			return campaignRequest{}, err
		}
		spec.ResumeCampaign = path
		spec.OverallTimeout = time.Duration(preflight.Plan.OverallTimeoutNanos)
		spec.resumePreflight = &preflight
	}
	return campaignRequestFromSpec(spec), nil
}

func campaignRequestFromSpec(spec CampaignSpec) campaignRequest {
	request := campaignRequest{
		campaignOptions: campaignOptions{
			campaignIdentityOptions: campaignIdentityOptions{
				ResumeCampaign: spec.ResumeCampaign, PlanSHA256: spec.PlanSHA256, Shard: spec.Shard, Artifacts: spec.Artifacts,
			},
			campaignTargetOptions: campaignTargetOptions{
				Target: spec.Target, Environment: spec.Environment, IOROMounts: spec.IOROMounts, IOROMountLimits: spec.IOROMountLimits,
			},
			campaignSearchOptions: campaignSearchOptions{
				Strategy: spec.Strategy, Seeds: spec.Seeds, MaxExecutions: spec.MaxExecutions, MaxChoiceDepth: spec.MaxChoiceDepth,
				ChoiceStartOrdinal: spec.ChoiceStartOrdinal, MaxForcedDecisions: spec.MaxForcedDecisions,
				MaxExplorationBytes: spec.MaxExplorationBytes, MaxExplorationResultBytes: spec.MaxExplorationResultBytes,
				SimulationDimensionLimits: spec.SimulationDimensionLimits, OnFailure: spec.OnFailure, FailureBudget: spec.FailureBudget,
			},
			campaignResourceOptions: campaignResourceOptions{
				Parallel: spec.Parallel, ExecutionTimeout: spec.ExecutionTimeout, OverallTimeout: spec.OverallTimeout,
				TerminateGrace: spec.TerminateGrace, OutputLimit: spec.OutputLimit, WorldTransitionLimit: spec.WorldTransitionLimit,
			},
			campaignObservationOptions: campaignObservationOptions{
				Diagnostics: spec.Diagnostics, ChoiceTraceLimit: spec.ChoiceTraceLimit, IOTranscriptLimit: spec.IOTranscriptLimit,
				ClockTick: spec.ClockTick, Coverage: spec.Coverage, RequiredSemanticProbes: spec.RequiredSemanticProbes,
				CollectExecutionEvidence: spec.CollectExecutionEvidence, ProgressInterval: spec.ProgressInterval,
			},
			campaignRetentionOptions: campaignRetentionOptions{
				KeepSuccesses: spec.KeepSuccesses, SuccessArtifactLimit: spec.SuccessArtifactLimit, SuccessBytesLimit: spec.SuccessBytesLimit,
			},
			campaignGuidanceOptions: campaignGuidanceOptions{
				Guide: spec.Guide, GuideRegression: spec.GuideRegression, GuideRegressionOverride: spec.GuideRegressionOverride,
				Corpus: spec.Corpus, GuideSnapshotSHA256: spec.GuideSnapshotSHA256,
			},
		},
		campaignRuntime: campaignRuntime{
			SupervisorCommand: spec.SupervisorCommand, CoordinatorCommand: spec.CoordinatorCommand, RunnerBuild: spec.RunnerBuild,
			Progress: spec.Progress, Preparer: spec.Preparer, Executor: spec.Executor, Replayer: spec.Replayer,
			guidancePlan: spec.guidancePlan, resumePreflight: spec.resumePreflight,
			failureArtifactLimit: spec.failureArtifactLimit, failureBytesLimit: spec.failureBytesLimit,
		},
	}
	request.normalize()
	return request
}

func (options *campaignOptions) normalize() {
	options.effectiveStrategy = normalizedStrategy(options.Strategy)
	options.effectiveShard = normalizedCampaignShard(options.Shard)
}

func (options campaignOptions) strategy() Strategy {
	return options.effectiveStrategy
}

func (options campaignOptions) shard() CampaignShard {
	return options.effectiveShard
}
