package runner

import "go.temporal.io/server/tools/gomadv3/runner/internal/campaign"

func orderRunCompletions(selection SeedSelection, completed map[uint64]struct{}, input <-chan runCompletion, output chan<- runCompletion) {
	orderShardRunCompletions(selection, CampaignShard{}, completed, input, output)
}

func orderShardRunCompletions(selection SeedSelection, shard CampaignShard, completed map[uint64]struct{}, input <-chan runCompletion, output chan<- runCompletion) {
	jobs := pendingJobs{seeds: selection.Iterator(), shard: shard, completed: completed}
	next, hasNext := jobs.Next()
	pending := make(map[uint64]runCompletion)
	for completion := range input {
		if _, duplicate := pending[completion.job.ordinal]; duplicate {
			panic("gomadv3: duplicate execution completion")
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
		panic("gomadv3: execution completion order has an unresolved gap")
	}
}

type pendingJobs struct {
	seeds     *SeedIterator
	ordinal   uint64
	shard     CampaignShard
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
		if !normalizedCampaignShard(jobs.shard).Owns(job.ordinal) {
			continue
		}
		if _, found := jobs.completed[job.ordinal]; found {
			continue
		}
		return job, true
	}
}

func newShardedSeedController(selection SeedSelection, shard CampaignShard, completed map[uint64]struct{}, parallel int, policy FailurePolicy, failureBudget uint64, summary CampaignResult) (*campaign.SeedController, error) {
	jobs := &pendingJobs{seeds: selection.Iterator(), shard: shard, completed: completed}
	return campaign.NewSeedController(campaign.SeedControllerConfig{
		Next: func() (campaign.SeedJob, bool) {
			job, ok := jobs.Next()
			return campaign.SeedJob{Ordinal: job.ordinal, Seed: job.seed}, ok
		},
		Parallel: parallel, Policy: campaign.FailurePolicy(policy), FailureBudget: failureBudget,
		Initial: campaign.CampaignStatistics{
			Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures,
			Watchdogs: summary.Watchdogs, ReplayDivergences: summary.ReplayDivergences,
			Cancelled: summary.Cancelled, DistinctFailures: summary.DistinctFailures, StopReason: campaign.ControllerStopReason(summary.StopReason),
		},
	})
}

func synchronizeCampaignStatistics(summary *CampaignResult, statistics campaign.CampaignStatistics) {
	summary.Attempted = statistics.Attempted
	summary.Succeeded = statistics.Succeeded
	summary.Failures = statistics.Failures
	summary.Watchdogs = statistics.Watchdogs
	summary.ReplayDivergences = statistics.ReplayDivergences
	summary.Cancelled = statistics.Cancelled
	summary.DistinctFailures = statistics.DistinctFailures
	summary.StopReason = StopReason(statistics.StopReason)
}
