package campaign

type FailurePolicy string

const (
	failurePolicyFirst  FailurePolicy = "first"
	failurePolicyBudget FailurePolicy = "budget"
	failurePolicyAll    FailurePolicy = "all"
)

type ControllerStopReason string

const (
	StopSeedsExhausted ControllerStopReason = "seeds_exhausted"
	StopFirstFailure   ControllerStopReason = "first_failure"
	StopFailureBudget  ControllerStopReason = "failure_budget"
)

type SeedJob struct {
	Ordinal uint64
	Seed    uint64
}

type completionKind uint8

const (
	completionInvalid completionKind = iota
	completionUnclassified
	CompletionSuccess
	CompletionCancelled
	completionFailure
)

type Completion struct {
	Kind             completionKind
	Domain           string
	Reason           string
	DistinctFailures uint64
}

type CampaignStatistics struct {
	Attempted         uint64
	Succeeded         uint64
	Failures          uint64
	Watchdogs         uint64
	ReplayDivergences uint64
	Cancelled         uint64
	DistinctFailures  uint64
	StopReason        ControllerStopReason
}

// CompletedSuccess is an attempt classified as a success.
func CompletedSuccess() Completion { return Completion{Kind: CompletionSuccess} }

// CompletedCancelled is an attempt cancelled by a stopped campaign.
func CompletedCancelled() Completion { return Completion{Kind: CompletionCancelled} }

// CompletedFailure is an attempt classified as a failure with its outcome
// domain and reason. distinctFailures includes this failure.
func CompletedFailure(domain, reason string, distinctFailures uint64) Completion {
	return Completion{Kind: completionFailure, Domain: domain, Reason: reason, DistinctFailures: distinctFailures}
}

// CompletedUnclassified is an attempt that ended before classification, such
// as one whose evidence failed the host. It counts as attempted only.
func CompletedUnclassified() Completion { return Completion{Kind: completionUnclassified} }

// Record counts one ordered completion into the statistics. A failure
// replaces DistinctFailures with the completion's count, which includes it.
// The stop reason is not decided here: it is the committed stop group's, or
// seeds_exhausted once a campaign finishes without one (Finalize).
func (statistics *CampaignStatistics) Record(completion Completion) {
	if completion.Kind == completionInvalid || completion.Kind > completionFailure {
		panic("gomad: completed a campaign attempt without a classification")
	}
	statistics.Attempted++
	switch completion.Kind {
	case CompletionSuccess:
		statistics.Succeeded++
	case CompletionCancelled:
		statistics.Cancelled++
	case completionFailure:
		statistics.Failures++
		if completion.Domain == "watchdog" {
			statistics.Watchdogs++
		}
		if completion.Reason == "world_replay_divergence" {
			statistics.ReplayDivergences++
		}
		statistics.DistinctFailures = completion.DistinctFailures
	case completionInvalid, completionUnclassified:
	}
}

// Finalize names a campaign that finished without a policy stop exhausted.
func (statistics *CampaignStatistics) Finalize() {
	if statistics.StopReason == "" {
		statistics.StopReason = StopSeedsExhausted
	}
}

// PolicyStop is the failure policy's pure decision for one ordered
// completion: the stop it triggers, or "". First-failure stops on any
// failure and requests cancellation of active attempts; a failure budget
// stops once the distinct failures reach it and lets active attempts finish.
// The campaign log derives a pending stop from the next ordered receipt with
// this decision, and the Runner requests cancellation or lets active attempts
// drain by it.
func PolicyStop(policy FailurePolicy, failureBudget uint64, completion Completion) ControllerStopReason {
	if completion.Kind != completionFailure {
		return ""
	}
	switch {
	case policy == failurePolicyFirst:
		return StopFirstFailure
	case policy == failurePolicyBudget && completion.DistinctFailures >= failureBudget:
		return StopFailureBudget
	default:
		return ""
	}
}
