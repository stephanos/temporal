package inspection

import (
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/exploration"
)

type ChoiceSummary struct {
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

type SimulationSummary struct {
	Parallel                int             `json:"parallel"`
	MaxExecutions           uint64          `json:"max_executions"`
	MaxForcedDecisions      uint64          `json:"max_forced_decisions"`
	MaxExplorationBytes     uint64          `json:"max_exploration_bytes"`
	MaxResultBytes          uint64          `json:"max_result_bytes"`
	FailureBudget           uint64          `json:"failure_budget"`
	Limits                  DimensionLimits `json:"dimension_limits"`
	LogicalExecutions       uint64          `json:"logical_executions"`
	CommittedRounds         uint64          `json:"committed_rounds"`
	Pending                 uint64          `json:"pending"`
	PendingBytes            uint64          `json:"pending_bytes"`
	SeenCandidates          uint64          `json:"seen_candidates"`
	DeduplicatedOutcomes    uint64          `json:"deduplicated_outcomes"`
	DistinctFailures        uint64          `json:"distinct_failures"`
	DeepestOverride         uint64          `json:"deepest_override"`
	OmittedByExecutionBound uint64          `json:"omitted_by_execution_bound"`
	OmittedByDepth          uint64          `json:"omitted_by_depth"`
	OmittedByDimension      uint64          `json:"omitted_by_dimension"`
	OmittedByCapacity       uint64          `json:"omitted_by_capacity"`
	StopReason              string          `json:"stop_reason,omitempty"`
	BoundedComplete         bool            `json:"bounded_complete"`
}

type DimensionLimits struct {
	Runtime  uint64 `json:"runtime"`
	Scenario uint64 `json:"scenario"`
	Network  uint64 `json:"network"`
	Storage  uint64 `json:"storage"`
	Fault    uint64 `json:"fault"`
	Crash    uint64 `json:"crash"`
}

type Lifecycle struct {
	State           string `json:"state"`
	LastStableState string `json:"last_stable_state"`
	Reason          string `json:"reason,omitempty"`
	Detail          string `json:"detail,omitempty"`
	Published       bool   `json:"published"`
	Resumable       bool   `json:"resumable"`
	Repairable      bool   `json:"repairable"`
	Action          string `json:"action,omitempty"`
}

type Shard struct {
	Index uint64 `json:"index"`
	Count uint64 `json:"count"`
}

type OrdinalRange struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

type MergeSummary struct {
	Path             string         `json:"path"`
	PlanSHA256       record.SHA256  `json:"plan_sha256"`
	Partial          bool           `json:"partial"`
	Missing          []OrdinalRange `json:"missing"`
	Shards           uint64         `json:"shards"`
	Attempted        uint64         `json:"attempted"`
	Succeeded        uint64         `json:"succeeded"`
	Failures         uint64         `json:"failures"`
	Watchdogs        uint64         `json:"watchdogs"`
	Cancelled        uint64         `json:"cancelled"`
	DistinctFailures uint64         `json:"distinct_failures"`
	RetainedEvidence uint64         `json:"retained_evidence"`
	RetainedBytes    uint64         `json:"retained_bytes"`
	JournalBytes     uint64         `json:"journal_bytes"`
	// JournalSegments described the retired segmented journal; it is always zero.
	JournalSegments uint64 `json:"journal_segments"`
	SelectionCount  uint64 `json:"selection_count"`
}

func ProjectChoiceSummary(summary exploration.ChoiceSummary) ChoiceSummary {
	return ChoiceSummary{
		Parallel: summary.Parallel, MaxExecutions: summary.MaxExecutions, MaxChoiceDepth: summary.MaxChoiceDepth, StartOrdinal: summary.StartOrdinal,
		MaxExplorationBytes: summary.MaxExplorationBytes, LogicalExecutions: summary.LogicalExecutions,
		CommittedRounds: summary.CommittedRounds, Pending: summary.Pending, PendingBytes: summary.PendingBytes,
		SeenPrefixes: summary.SeenPrefixes, DeduplicatedOutcomes: summary.DeduplicatedOutcomes, DeepestPrefix: summary.DeepestPrefix,
		OmittedByExecutionBound: summary.OmittedByExecutionBound, OmittedByDepth: summary.OmittedByDepth,
		OmittedByCapacity: summary.OmittedByCapacity, OmittedBySelectReadiness: summary.OmittedBySelectReadiness,
		StopReason: string(summary.StopReason), BoundedComplete: summary.BoundedComplete,
	}
}

func ProjectSimulationSummary(summary exploration.SimulationSummary) SimulationSummary {
	return SimulationSummary{
		Parallel: summary.Parallel, MaxExecutions: summary.MaxExecutions, MaxForcedDecisions: summary.MaxForcedDecisions,
		MaxExplorationBytes: summary.MaxExplorationBytes, MaxResultBytes: summary.MaxResultBytes, FailureBudget: summary.FailureBudget,
		Limits: DimensionLimits(summary.Limits), LogicalExecutions: summary.LogicalExecutions,
		CommittedRounds: summary.CommittedRounds, Pending: summary.Pending, PendingBytes: summary.PendingBytes,
		SeenCandidates: summary.SeenCandidates, DeduplicatedOutcomes: summary.DeduplicatedOutcomes, DistinctFailures: summary.DistinctFailures,
		DeepestOverride: summary.DeepestOverride, OmittedByExecutionBound: summary.OmittedByExecutionBound,
		OmittedByDepth: summary.OmittedByDepth, OmittedByDimension: summary.OmittedByDimension,
		OmittedByCapacity: summary.OmittedByCapacity, StopReason: string(summary.StopReason), BoundedComplete: summary.BoundedComplete,
	}
}

func ProjectLifecycle(status campaign.LifecycleStatus) Lifecycle {
	return Lifecycle{
		State: string(status.State), LastStableState: string(status.LastStableState), Reason: status.Reason, Detail: status.Detail,
		Published: status.Published, Resumable: status.Resumable, Repairable: status.Repairable, Action: string(status.Action),
	}
}

func ProjectMerge(merged campaign.MergedCampaign) MergeSummary {
	record := merged.Record
	missing := make([]OrdinalRange, len(record.Missing))
	for index, value := range record.Missing {
		missing[index] = OrdinalRange{Start: uint64(value.Start), End: uint64(value.End)}
	}
	return MergeSummary{
		Path: merged.Path, PlanSHA256: record.PlanSHA256, Partial: record.Partial, Missing: missing, Shards: uint64(len(record.Shards)), Attempted: uint64(record.Attempted),
		Succeeded: uint64(record.Succeeded), Failures: uint64(record.Failures), Watchdogs: uint64(record.Watchdogs), Cancelled: uint64(record.Cancelled), DistinctFailures: uint64(record.DistinctFailures),
		RetainedEvidence: uint64(record.RetainedEvidence), RetainedBytes: uint64(record.EvidenceBytes), JournalBytes: uint64(record.Journal.Bytes), SelectionCount: uint64(record.SelectionCount),
	}
}
