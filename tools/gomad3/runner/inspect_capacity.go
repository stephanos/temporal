package runner

import (
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
)

// ExecutionJournalLimitsInspection is the detached published journal capacity evidence.
type ExecutionJournalLimitsInspection struct {
	MaximumExecutions        record.Uint64String `json:"maximum_executions"`
	MaximumBytes             record.Uint64String `json:"maximum_bytes"`
	SegmentBytes             record.Uint64String `json:"segment_bytes"`
	SegmentRecords           record.Uint64String `json:"segment_records"`
	MaximumSegments          record.Uint64String `json:"maximum_segments"`
	MaximumPartialExecutions record.Uint64String `json:"maximum_partial_executions"`
	CapacityOutcome          string              `json:"capacity_outcome"`
}

// ArtifactCapacityInspection is the detached published artifact capacity evidence.
type ArtifactCapacityInspection struct {
	FailureArtifacts record.Uint64String `json:"failure_artifacts"`
	FailureBytes     record.Uint64String `json:"failure_bytes"`
	SuccessArtifacts record.Uint64String `json:"success_artifacts"`
	SuccessBytes     record.Uint64String `json:"success_bytes"`
	TotalBytes       record.Uint64String `json:"total_bytes"`
	TranscriptBytes  record.Uint64String `json:"transcript_bytes"`
	FailureOutcome   string              `json:"failure_outcome"`
	SuccessOutcome   string              `json:"success_outcome"`
}

func projectExecutionJournalLimits(value campaign.ExecutionJournalPlan) ExecutionJournalLimitsInspection {
	return ExecutionJournalLimitsInspection{
		MaximumExecutions:        value.MaximumExecutions,
		MaximumBytes:             value.MaximumBytes,
		SegmentBytes:             value.SegmentBytes,
		SegmentRecords:           value.SegmentRecords,
		MaximumSegments:          value.MaximumSegments,
		MaximumPartialExecutions: value.MaximumPartialExecutions,
		CapacityOutcome:          string(value.CapacityOutcome),
	}
}

func projectArtifactCapacity(value campaign.ArtifactCapacityPlan) ArtifactCapacityInspection {
	return ArtifactCapacityInspection{
		FailureArtifacts: value.FailureArtifacts,
		FailureBytes:     value.FailureBytes,
		SuccessArtifacts: value.SuccessArtifacts,
		SuccessBytes:     value.SuccessBytes,
		TotalBytes:       value.TotalBytes,
		TranscriptBytes:  value.TranscriptBytes,
		FailureOutcome:   string(value.FailureOutcome),
		SuccessOutcome:   string(value.SuccessOutcome),
	}
}
