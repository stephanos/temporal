package campaign

import (
	"errors"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

// The execution journal limits are the semantic capacities a campaign plan
// records for its committed outcomes. The campaign log enforces them on every
// outcome transaction:
//
//   - MaximumExecutions bounds the committed outcomes;
//   - MaximumBytes bounds the summed encoded bytes of those outcomes;
//   - SegmentBytes bounds the encoded bytes of one execution outcome. It is the
//     per-execution payload bound, never a bound on a whole transaction: a
//     transaction of several outcomes derives its frame bound from this
//     envelope and MaximumPartialExecutions;
//   - MaximumPartialExecutions bounds the attempts in flight at once.
//
// SegmentRecords and MaximumSegments described the retired segmented journal.
// They keep their derived values in plans and reports so callers and portable
// plans retain their fields, but the log enforces neither and they limit
// nothing.
const (
	defaultExecutionSegmentBytes      = 1 << 20
	defaultExecutionSegmentRecords    = 1024
	maximumExecutionJournalExecutions = 1_000_000
	maximumExecutionJournalBytes      = 1 << 50
	maximumExecutionSegmentBytes      = 64 << 20
)

type executionJournalLimits struct {
	MaximumExecutions        uint64
	MaximumBytes             uint64
	SegmentBytes             uint64
	SegmentRecords           uint64
	MaximumSegments          uint64
	MaximumPartialExecutions uint64
}

// ExecutionJournalPlan is the recorded form of the execution journal limits.
// SegmentRecords and MaximumSegments are retired compatibility values; see
// executionJournalLimits.
type ExecutionJournalPlan struct {
	MaximumExecutions        record.Uint64String    `json:"maximum_executions"`
	MaximumBytes             record.Uint64String    `json:"maximum_bytes"`
	SegmentBytes             record.Uint64String    `json:"segment_bytes"`
	SegmentRecords           record.Uint64String    `json:"segment_records"`
	MaximumSegments          record.Uint64String    `json:"maximum_segments"`
	MaximumPartialExecutions record.Uint64String    `json:"maximum_partial_executions"`
	CapacityOutcome          journalCapacityOutcome `json:"capacity_outcome"`
}

// executionJournalReference summarizes the committed outcomes a published
// campaign or merged campaign holds: the log format, the outcome count and
// their summed encoded bytes.
type executionJournalReference struct {
	Schema  string              `json:"schema"`
	Records record.Uint64String `json:"records"`
	Bytes   record.Uint64String `json:"bytes"`
}

func normalizeExecutionJournalLimits(config CampaignConfig) (executionJournalLimits, error) {
	limits := config.Journal
	if limits == (executionJournalLimits{}) {
		maximumRuns := config.SelectionCount
		if (config.Strategy == "choice-exploration" || config.Strategy == "simulation-exploration") && config.MaxExecutions != 0 {
			maximumRuns = config.MaxExecutions
		}
		maximumPartialRuns := config.Parallel
		if maximumPartialRuns == 0 {
			maximumPartialRuns = 1
		}
		if maximumPartialRuns > maximumRuns {
			maximumPartialRuns = maximumRuns
		}
		if maximumRuns > ^uint64(0)/defaultExecutionSegmentBytes {
			return executionJournalLimits{}, errors.New("execution journal byte capacity overflows")
		}
		limits = executionJournalLimits{
			MaximumExecutions: maximumRuns, MaximumBytes: max(maximumRuns, 1) * defaultExecutionSegmentBytes,
			SegmentBytes: defaultExecutionSegmentBytes, SegmentRecords: defaultExecutionSegmentRecords,
			MaximumSegments: max(maximumRuns, 1), MaximumPartialExecutions: maximumPartialRuns,
		}
	}
	if limits.MaximumExecutions > maximumExecutionJournalExecutions || limits.MaximumBytes == 0 || limits.MaximumBytes > maximumExecutionJournalBytes || limits.SegmentBytes == 0 || limits.SegmentBytes > maximumExecutionSegmentBytes || limits.SegmentBytes > limits.MaximumBytes ||
		limits.SegmentRecords == 0 || limits.MaximumSegments == 0 || limits.MaximumSegments > max(limits.MaximumExecutions, 1) || limits.MaximumExecutions != 0 && limits.MaximumPartialExecutions == 0 || limits.MaximumPartialExecutions > limits.MaximumExecutions {
		return executionJournalLimits{}, errors.New("execution journal limits are invalid")
	}
	return limits, nil
}

func DeriveExecutionJournalPlan(strategy string, selectionCount, maxExecutions, parallel uint64) (ExecutionJournalPlan, error) {
	limits, err := normalizeExecutionJournalLimits(CampaignConfig{
		Strategy: strategy, SelectionCount: selectionCount, MaxExecutions: maxExecutions, Parallel: parallel,
	})
	if err != nil {
		return ExecutionJournalPlan{}, err
	}
	return recordExecutionJournalLimits(limits), nil
}

func recordExecutionJournalLimits(limits executionJournalLimits) ExecutionJournalPlan {
	return ExecutionJournalPlan{
		MaximumExecutions: record.Uint64String(limits.MaximumExecutions), MaximumBytes: record.Uint64String(limits.MaximumBytes),
		SegmentBytes: record.Uint64String(limits.SegmentBytes), SegmentRecords: record.Uint64String(limits.SegmentRecords),
		MaximumSegments: record.Uint64String(limits.MaximumSegments), MaximumPartialExecutions: record.Uint64String(limits.MaximumPartialExecutions),
		CapacityOutcome: CapacityInfrastructureFailure,
	}
}

func validateExecutionJournalPlan(recorded ExecutionJournalPlan, plan CampaignPlan) error {
	if recorded.CapacityOutcome != CapacityInfrastructureFailure {
		return errors.New("execution journal capacity outcome is invalid")
	}
	limits := executionJournalLimitsFromPlan(recorded)
	if _, err := normalizeExecutionJournalLimits(CampaignConfig{Journal: limits}); err != nil {
		return err
	}
	maximumRuns := uint64(plan.SelectionCount)
	if plan.Strategy == "choice-exploration" || plan.Strategy == "simulation-exploration" {
		maximumRuns = uint64(plan.MaxExecutions)
	}
	maximumPartialRuns := uint64(plan.Parallel)
	if maximumPartialRuns > maximumRuns {
		maximumPartialRuns = maximumRuns
	}
	if limits.MaximumExecutions != maximumRuns || limits.MaximumPartialExecutions != maximumPartialRuns {
		return errors.New("execution journal limits do not match campaign execution limits")
	}
	return nil
}

func executionJournalLimitsFromPlan(recorded ExecutionJournalPlan) executionJournalLimits {
	return executionJournalLimits{
		MaximumExecutions: uint64(recorded.MaximumExecutions), MaximumBytes: uint64(recorded.MaximumBytes),
		SegmentBytes: uint64(recorded.SegmentBytes), SegmentRecords: uint64(recorded.SegmentRecords),
		MaximumSegments: uint64(recorded.MaximumSegments), MaximumPartialExecutions: uint64(recorded.MaximumPartialExecutions),
	}
}
