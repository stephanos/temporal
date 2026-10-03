package runner

import (
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

// campaignOptions is the serializable intent of one campaign: everything a
// local run and an isolated coordinator must agree on. It is the coordinator
// request's options member, so an option added here crosses the process
// boundary without a second field list. Callbacks, injected dependencies,
// resolved child commands, Runner identity and resume state stay in
// campaignRun. Campaign plan and resume records are separate versioned
// contracts that map to and from these options.
//
// The groups are embedded so each option keeps its CampaignSpec name, and
// tagged so the request nests them by invariant.
type campaignOptions struct {
	campaignTargetIntent   `json:"Target"`
	campaignSearch         `json:"Search"`
	campaignResourceLimits `json:"Limits"`
	campaignObservation    `json:"Observation"`
	campaignRetention      `json:"Retention"`
}

// campaignTargetIntent identifies the campaign and the target program, its
// environment and its read-only inputs.
type campaignTargetIntent struct {
	ResumeCampaign string
	PlanSHA256     record.SHA256
	Shard          CampaignShard
	Target         target.Spec
	Environment    []string
	// ClockTick is the virtual-clock tick policy; empty means record.ClockTickStrict.
	ClockTick       string
	IOROMounts      []string
	IOROMountLimits readonlymount.Limits
}

// campaignSearch selects executions: the strategy, its seeds and bounds,
// failure stopping and guidance.
type campaignSearch struct {
	Strategy                  Strategy
	Seeds                     string
	Parallel                  int
	OnFailure                 FailurePolicy
	FailureBudget             uint64
	MaxExecutions             uint64
	MaxChoiceDepth            uint64
	ChoiceStartOrdinal        uint64
	MaxForcedDecisions        uint64
	MaxExplorationBytes       uint64
	MaxExplorationResultBytes uint64
	SimulationDimensionLimits SimulationDimensionLimits
	Guide                     bool
	GuideRegression           bool
	GuideRegressionOverride   *bool
	Corpus                    string
	GuideSnapshotSHA256       record.SHA256
}

// campaignResourceLimits bounds each execution and the whole campaign.
type campaignResourceLimits struct {
	ExecutionTimeout     time.Duration
	OverallTimeout       time.Duration
	TerminateGrace       time.Duration
	OutputLimit          uint64
	WorldTransitionLimit uint64
	// IOTranscriptLimit bounds the I/O transcript; zero means
	// deterministicio.DefaultTranscriptBytes.
	IOTranscriptLimit uint64
}

// campaignObservation selects what each execution records and reports.
type campaignObservation struct {
	Diagnostics              bool `json:",omitempty"`
	ChoiceTraceLimit         uint64
	Coverage                 CoverageMode
	RequiredSemanticProbes   []string
	CollectExecutionEvidence bool
	ProgressInterval         time.Duration
}

// campaignRetention selects where evidence is published and which
// successful executions are kept.
type campaignRetention struct {
	Artifacts            string
	KeepSuccesses        KeepSuccesses
	SuccessArtifactLimit uint64
	SuccessBytesLimit    uint64
}

// campaignRun is one campaign as the Runner executes it: the normalized
// options plus the process wiring that never crosses the coordinator
// request as intent.
type campaignRun struct {
	campaignOptions
	SupervisorCommand    []string
	CoordinatorCommand   []string
	RunnerBuild          string
	Progress             CampaignEventFunc
	Preparer             Preparer
	Replayer             ArtifactReplayer
	guidancePlan         *campaign.GuidancePlan
	resumePreflight      *campaign.ResumePreflight
	failureArtifactLimit uint64
	failureBytesLimit    uint64
	dependencies
}

// newCampaignRun is the one conversion from the public request to the
// campaign the Runner executes. It copies the request's lists, so the run
// owns them and an empty list is absent, and normalizes the options.
func newCampaignRun(spec CampaignSpec) campaignRun {
	run := campaignRun{
		SupervisorCommand: append([]string(nil), spec.SupervisorCommand...), CoordinatorCommand: append([]string(nil), spec.CoordinatorCommand...), RunnerBuild: spec.RunnerBuild,
		Progress: spec.Progress, Preparer: spec.Preparer, Replayer: spec.Replayer,
	}
	run.campaignTargetIntent = campaignTargetIntent{
		ResumeCampaign: spec.ResumeCampaign, PlanSHA256: spec.PlanSHA256, Shard: spec.Shard, Target: spec.Target, Environment: append([]string(nil), spec.Environment...),
		ClockTick: spec.ClockTick, IOROMounts: append([]string(nil), spec.IOROMounts...), IOROMountLimits: spec.IOROMountLimits,
	}
	run.campaignSearch = campaignSearch{
		Strategy: spec.Strategy, Seeds: spec.Seeds, Parallel: spec.Parallel, OnFailure: spec.OnFailure, FailureBudget: spec.FailureBudget,
		MaxExecutions: spec.MaxExecutions, MaxChoiceDepth: spec.MaxChoiceDepth, ChoiceStartOrdinal: spec.ChoiceStartOrdinal, MaxForcedDecisions: spec.MaxForcedDecisions,
		MaxExplorationBytes: spec.MaxExplorationBytes, MaxExplorationResultBytes: spec.MaxExplorationResultBytes, SimulationDimensionLimits: spec.SimulationDimensionLimits,
		Guide: spec.Guide, GuideRegression: spec.GuideRegression, GuideRegressionOverride: spec.GuideRegressionOverride, Corpus: spec.Corpus, GuideSnapshotSHA256: spec.GuideSnapshotSHA256,
	}
	run.campaignResourceLimits = campaignResourceLimits{
		ExecutionTimeout: spec.ExecutionTimeout, OverallTimeout: spec.OverallTimeout, TerminateGrace: spec.TerminateGrace,
		OutputLimit: spec.OutputLimit, WorldTransitionLimit: spec.WorldTransitionLimit, IOTranscriptLimit: spec.IOTranscriptLimit,
	}
	run.campaignObservation = campaignObservation{
		Diagnostics: spec.Diagnostics, ChoiceTraceLimit: spec.ChoiceTraceLimit, Coverage: spec.Coverage, RequiredSemanticProbes: append([]string(nil), spec.RequiredSemanticProbes...),
		CollectExecutionEvidence: spec.CollectExecutionEvidence, ProgressInterval: spec.ProgressInterval,
	}
	run.campaignRetention = campaignRetention{
		Artifacts: spec.Artifacts, KeepSuccesses: spec.KeepSuccesses, SuccessArtifactLimit: spec.SuccessArtifactLimit, SuccessBytesLimit: spec.SuccessBytesLimit,
	}
	run.campaignOptions = run.normalized()
	return run
}

// normalized applies the option defaults that do not depend on the host. An
// empty strategy is the seed strategy. A zero shard stays zero because it is
// the unsharded campaign, which the plan records by omission;
// normalizedCampaignShard projects it where a count is needed. A request the
// coordinator decodes is normalized again, so a request that omits a
// defaulted option means what it means locally.
func (options campaignOptions) normalized() campaignOptions {
	options.Strategy = normalizedStrategy(options.Strategy)
	return options
}

// normalizeCampaign produces the campaign Explore runs locally or sends to
// the isolated coordinator. A resumed campaign also takes its resolved path,
// opened preflight and recorded overall deadline here, before either path
// starts.
func normalizeCampaign(spec CampaignSpec) (campaignRun, error) {
	run := newCampaignRun(spec)
	if run.ResumeCampaign == "" {
		return run, nil
	}
	return resumeRequestDefaults(run)
}
