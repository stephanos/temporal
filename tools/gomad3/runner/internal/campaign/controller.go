package campaign

import "errors"

type FailurePolicy string

const (
	FailurePolicyFirst  FailurePolicy = "first"
	FailurePolicyBudget FailurePolicy = "budget"
	FailurePolicyAll    FailurePolicy = "all"
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

// Completion is how one scheduled attempt ended. Its zero value is not a
// completion; use one of the Completed constructors.
type Completion struct {
	kind             completionKind
	domain           string
	reason           string
	distinctFailures uint64
}

type completionKind uint8

const (
	completionInvalid completionKind = iota
	completionSuccess
	completionCancelled
	completionFailure
	completionUnclassified
)

// CompletedSuccess is an attempt classified as a success.
func CompletedSuccess() Completion { return Completion{kind: completionSuccess} }

// CompletedCancelled is an attempt cancelled by a stopped campaign.
func CompletedCancelled() Completion { return Completion{kind: completionCancelled} }

// CompletedFailure is an attempt classified as a failure with its outcome
// domain and reason. distinctFailures is the campaign's distinct failure
// signature count including this failure; the failure budget compares it.
func CompletedFailure(domain, reason string, distinctFailures uint64) Completion {
	return Completion{kind: completionFailure, domain: domain, reason: reason, distinctFailures: distinctFailures}
}

// CompletedUnclassified is an attempt that ended before classification, such
// as one whose evidence failed the host. It counts as attempted only.
func CompletedUnclassified() Completion { return Completion{kind: completionUnclassified} }

type SeedControllerConfig struct {
	Next          func() (SeedJob, bool)
	Parallel      int
	Policy        FailurePolicy
	FailureBudget uint64
	Initial       CampaignStatistics
}

type SeedController struct {
	next          func() (SeedJob, bool)
	parallel      int
	policy        FailurePolicy
	failureBudget uint64
	statistics    CampaignStatistics
	active        int
	exhausted     bool
	stopped       bool
}

func NewSeedController(config SeedControllerConfig) (*SeedController, error) {
	if config.Next == nil || config.Parallel <= 0 {
		return nil, errors.New("seed campaign controller requires a source and positive parallelism")
	}
	if config.Policy == FailurePolicyBudget && config.FailureBudget == 0 {
		return nil, errors.New("seed campaign failure budget must be positive")
	}
	if config.Policy != FailurePolicyFirst && config.Policy != FailurePolicyBudget && config.Policy != FailurePolicyAll {
		return nil, errors.New("seed campaign failure policy is invalid")
	}
	controller := &SeedController{
		next: config.Next, parallel: config.Parallel, policy: config.Policy,
		failureBudget: config.FailureBudget, statistics: config.Initial,
	}
	switch config.Policy {
	case FailurePolicyFirst:
		if config.Initial.Failures != 0 {
			controller.statistics.StopReason = StopFirstFailure
			controller.stopped = true
		}
	case FailurePolicyBudget:
		if config.Initial.DistinctFailures >= config.FailureBudget {
			controller.statistics.StopReason = StopFailureBudget
			controller.stopped = true
		}
	}
	return controller, nil
}

func (controller *SeedController) Next() (SeedJob, bool) {
	if controller.stopped || controller.exhausted || controller.active >= controller.parallel {
		return SeedJob{}, false
	}
	job, ok := controller.next()
	if !ok {
		controller.exhausted = true
		return SeedJob{}, false
	}
	controller.active++
	return job, true
}

// Complete counts one scheduled attempt as completed: it releases the
// attempt's slot, counts it as attempted and classified, and applies the
// failure policy, all in one transition. It reports whether active work must
// be cancelled. Completing without active work, or with the zero Completion,
// is an invariant violation that leaves the controller unchanged.
func (controller *SeedController) Complete(completion Completion) bool {
	if controller.active == 0 {
		panic("gomad3: completed an inactive campaign attempt")
	}
	if completion.kind == completionInvalid {
		panic("gomad3: completed a campaign attempt without a classification")
	}
	controller.active--
	controller.statistics.Attempted++
	switch completion.kind {
	case completionSuccess:
		controller.statistics.Succeeded++
	case completionCancelled:
		controller.statistics.Cancelled++
	case completionFailure:
		return controller.recordFailure(completion)
	}
	return false
}

func (controller *SeedController) recordFailure(completion Completion) bool {
	controller.statistics.Failures++
	if completion.domain == "watchdog" {
		controller.statistics.Watchdogs++
	}
	if completion.reason == "world_replay_divergence" {
		controller.statistics.ReplayDivergences++
	}
	controller.statistics.DistinctFailures = completion.distinctFailures
	if controller.stopped {
		return false
	}
	switch controller.policy {
	case FailurePolicyFirst:
		controller.statistics.StopReason = StopFirstFailure
		controller.stopped = true
		return true
	case FailurePolicyBudget:
		if completion.distinctFailures >= controller.failureBudget {
			controller.statistics.StopReason = StopFailureBudget
			controller.stopped = true
		}
	}
	return false
}

func (controller *SeedController) Stop() {
	controller.stopped = true
}

func (controller *SeedController) Stopped() bool {
	return controller.stopped
}

func (controller *SeedController) Active() int {
	return controller.active
}

func (controller *SeedController) Done() bool {
	return controller.active == 0 && (controller.exhausted || controller.stopped)
}

func (controller *SeedController) Finalize() {
	if controller.statistics.StopReason == "" {
		controller.statistics.StopReason = StopSeedsExhausted
	}
}

func (controller *SeedController) Statistics() CampaignStatistics {
	return controller.statistics
}
