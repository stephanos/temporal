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

func (controller *SeedController) FinishAttempt() {
	if controller.active == 0 {
		panic("gomadv3: completed an inactive campaign attempt")
	}
	controller.active--
	controller.statistics.Attempted++
}

func (controller *SeedController) RecordSuccess() {
	controller.statistics.Succeeded++
}

func (controller *SeedController) RecordCancelled() {
	controller.statistics.Cancelled++
}

func (controller *SeedController) RecordFailure(domain, reason string, distinct uint64) bool {
	controller.statistics.Failures++
	if domain == "watchdog" {
		controller.statistics.Watchdogs++
	}
	if reason == "world_replay_divergence" {
		controller.statistics.ReplayDivergences++
	}
	controller.statistics.DistinctFailures = distinct
	if controller.stopped {
		return false
	}
	switch controller.policy {
	case FailurePolicyFirst:
		controller.statistics.StopReason = StopFirstFailure
		controller.stopped = true
		return true
	case FailurePolicyBudget:
		if distinct >= controller.failureBudget {
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
