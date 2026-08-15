package runner

func orderRunCompletions(selection SeedSelection, completed map[uint64]struct{}, input <-chan runCompletion, output chan<- runCompletion) {
	jobs := pendingJobs{seeds: selection.Iterator(), completed: completed}
	next, hasNext := jobs.Next()
	pending := make(map[uint64]runCompletion)
	for completion := range input {
		if _, duplicate := pending[completion.job.ordinal]; duplicate {
			panic("gomadv3: duplicate run completion")
		}
		pending[completion.job.ordinal] = completion
		for hasNext {
			ready, found := pending[next.ordinal]
			if !found {
				break
			}
			delete(pending, next.ordinal)
			output <- ready
			next, hasNext = jobs.Next()
		}
	}
	if len(pending) != 0 {
		panic("gomadv3: run completion order has an unresolved gap")
	}
}

type pendingJobs struct {
	seeds     *SeedIterator
	ordinal   uint64
	completed map[uint64]struct{}
}

func (jobs *pendingJobs) Next() (runJob, bool) {
	for {
		seed, ok := jobs.seeds.Next()
		if !ok {
			return runJob{}, false
		}
		job := runJob{ordinal: jobs.ordinal, seed: seed}
		jobs.ordinal++
		if _, found := jobs.completed[job.ordinal]; found {
			continue
		}
		return job, true
	}
}

type seedCampaign struct {
	jobs          pendingJobs
	parallel      int
	policy        FailurePolicy
	failureBudget uint64
	summary       *CampaignResult
	active        int
	exhausted     bool
	stopped       bool
}

func newSeedCampaign(selection SeedSelection, completed map[uint64]struct{}, parallel int, policy FailurePolicy, failureBudget uint64, summary *CampaignResult) *seedCampaign {
	campaign := &seedCampaign{
		jobs: pendingJobs{seeds: selection.Iterator(), completed: completed}, parallel: parallel,
		policy: policy, failureBudget: failureBudget, summary: summary,
	}
	switch policy {
	case PolicyFirst:
		if summary.Failures != 0 {
			summary.StopReason = StopFirstFailure
			campaign.stopped = true
		}
	case PolicyBudget:
		if summary.DistinctFailures >= failureBudget {
			summary.StopReason = StopFailureBudget
			campaign.stopped = true
		}
	}
	return campaign
}

func (campaign *seedCampaign) Next() (runJob, bool) {
	if campaign.stopped || campaign.exhausted || campaign.active >= campaign.parallel {
		return runJob{}, false
	}
	job, ok := campaign.jobs.Next()
	if !ok {
		campaign.exhausted = true
		return runJob{}, false
	}
	campaign.active++
	return job, true
}

func (campaign *seedCampaign) FinishAttempt() {
	if campaign.active == 0 {
		panic("gomadv3: completed an inactive campaign attempt")
	}
	campaign.active--
	campaign.summary.Attempted++
}

func (campaign *seedCampaign) RecordSuccess() {
	campaign.summary.Succeeded++
}

func (campaign *seedCampaign) RecordCancelled() {
	campaign.summary.Cancelled++
}

func (campaign *seedCampaign) RecordFailure(domain, reason string, distinct uint64) bool {
	campaign.summary.Failures++
	if domain == "watchdog" {
		campaign.summary.Watchdogs++
	}
	if reason == "world_replay_divergence" {
		campaign.summary.ReplayDivergences++
	}
	campaign.summary.DistinctFailures = distinct
	if campaign.stopped {
		return false
	}
	switch campaign.policy {
	case PolicyFirst:
		campaign.summary.StopReason = StopFirstFailure
		campaign.stopped = true
		return true
	case PolicyBudget:
		if distinct >= campaign.failureBudget {
			campaign.summary.StopReason = StopFailureBudget
			campaign.stopped = true
		}
	}
	return false
}

func (campaign *seedCampaign) Stop() {
	campaign.stopped = true
}

func (campaign *seedCampaign) Stopped() bool {
	return campaign.stopped
}

func (campaign *seedCampaign) Active() int {
	return campaign.active
}

func (campaign *seedCampaign) Done() bool {
	return campaign.active == 0 && (campaign.exhausted || campaign.stopped)
}

func (campaign *seedCampaign) Finalize() {
	if campaign.summary.StopReason == "" {
		campaign.summary.StopReason = StopSeedsExhausted
	}
}
