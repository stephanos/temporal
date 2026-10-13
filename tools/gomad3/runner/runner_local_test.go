package runner

import (
	"context"
	"errors"
	"fmt"
	"testing"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestLocalCampaignPreparationShortCircuits(t *testing.T) {
	primary := errors.New("preparing output failed")
	for _, test := range []struct {
		name   string
		cancel bool
		reason string
	}{
		{name: "progress failure", reason: "progress_output"},
		{name: "parent cancellation", cancel: true, reason: "cancelled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			preparer := newFakePreparer(t)
			executor := &fakeExecutor{}
			config, dependencies := testConfig(t, preparer, executor, "1-3", PolicyAll, 2)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var events []CampaignEvent
			config.Progress = func(event CampaignEvent) error {
				events = append(events, event)
				if test.cancel {
					cancel()
					return nil
				}
				return primary
			}
			summary, err := exploreWith(ctx, config, dependencies)
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.reason {
				t.Fatalf("error = %v, want HostError reason %q", err, test.reason)
			}
			if test.cancel {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want cancellation", err)
				}
			} else {
				if !errors.Is(err, primary) {
					t.Fatalf("error = %v, want progress error", err)
				}
			}
			if preparer.calls != 0 || len(executor.directories()) != 0 {
				t.Fatalf("preparation/execution ran: calls=%d dirs=%v", preparer.calls, executor.directories())
			}
			if len(events) != 1 || events[0].Phase != ProgressPreparing || events[0].Selected != 3 {
				t.Fatalf("events = %#v", events)
			}
			if summary.Attempted != 0 || summary.CampaignPath == "" {
				t.Fatalf("summary = %#v", summary)
			}
			committed, readErr := campaign.ReadCommittedState(summary.CampaignPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if committed.Lifecycle.Published {
				t.Fatalf("campaign was published: %#v", committed.Lifecycle)
			}
		})
	}
}

func TestLocalCompletionErrorPrecedence(t *testing.T) {
	primary := errors.New("supervisor failed")
	for _, test := range []struct {
		name     string
		cancel   bool
		prepared bool
		runErr   error
		want     string
	}{
		{name: "cancellation before supervision and evidence", cancel: true, runErr: primary, want: "cancelled"},
		{name: "supervision before integrity and evidence", runErr: primary, want: "target_supervision"},
		{name: "integrity before World evidence", want: "prepared_target_integrity"},
		{name: "World evidence before watchdog outcome", prepared: true, want: "world_record"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			local := localCampaign{
				config:     campaignRequest{campaignRuntime: campaignRuntime{Backend: scriptProvider()}},
				overallCtx: ctx, activeCtx: ctx, activeCancel: cancel,
				prepared: target.Prepared{Path: "missing prepared target"},
				attempts: map[uint64]*seedAttempt{0: {job: runJob{seed: 1}}},
			}
			if test.prepared {
				local.prepared = newFakePreparer(t).prepared
			}
			if test.cancel {
				cancel()
			}
			local.arrive(runCompletion{job: runJob{seed: 1}, err: test.runErr, result: backend.ExecutionResult{
				WorldRecord: []byte("malformed World evidence"), WatchdogTimeout: true,
			}})
			local.advance()
			var hostError *HostError
			if !errors.As(local.hostFailure, &hostError) || hostError.Reason != test.want {
				t.Fatalf("error = %v, want HostError reason %q", local.hostFailure, test.want)
			}
			if test.runErr != nil && !test.cancel && !errors.Is(local.hostFailure, primary) {
				t.Fatalf("error lost supervisor cause: %v", local.hostFailure)
			}
			if local.summary.Attempted != 1 || local.summary.Watchdogs != 0 || local.active() != 0 || !local.stopped || ctx.Err() == nil {
				t.Fatalf("summary = %#v, active=%d stopped=%v", local.summary, local.active(), local.stopped)
			}
		})
	}
}

func TestLocalFinalizationPreservesHostFailureBeforePublication(t *testing.T) {
	for _, priorFailure := range []bool{false, true} {
		t.Run(fmt.Sprintf("prior_failure=%t", priorFailure), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cleanup := errors.New("cannot remove attempt work")
			local := localCampaign{
				overallCtx: context.Background(), activeCtx: ctx, activeCancel: cancel,
				attempts: map[uint64]*seedAttempt{0: {job: runJob{seed: 1}, arrived: true, cancelled: true, cleanupErr: cleanup}},
			}
			if priorFailure {
				local.hostFailure = errors.New("earlier failure")
			}
			primary := local.hostFailure
			local.advance()
			if priorFailure {
				if local.hostFailure != primary || ctx.Err() != nil || local.stopped {
					t.Fatalf("earlier error changed: error=%v context=%v stopped=%v", local.hostFailure, ctx.Err(), local.stopped)
				}
			} else {
				var hostError *HostError
				if !errors.As(local.hostFailure, &hostError) || hostError.Reason != "partial_cleanup" || !errors.Is(local.hostFailure, cleanup) {
					t.Fatalf("error = %v, want attempt cleanup failure", local.hostFailure)
				}
				if !errors.Is(ctx.Err(), context.Canceled) || !local.stopped {
					t.Fatal("cleanup failure did not stop active work")
				}
			}
			if err := local.finishSeedCampaign(); err != local.hostFailure {
				t.Fatalf("final error = %v, want primary %v", err, local.hostFailure)
			}
			if local.batchComplete {
				t.Fatal("failed campaign was completed")
			}
		})
	}
}

func TestLocalFinalizationClassifiesPublicationFailure(t *testing.T) {
	selection, err := ParseSeeds("1")
	if err != nil {
		t.Fatal(err)
	}
	statistics := campaign.CampaignStatistics{}
	statistics.Record(campaign.CompletedSuccess())
	local := localCampaign{
		config:     campaignRequest{campaignRuntime: campaignRuntime{Backend: scriptProvider()}},
		overallCtx: context.Background(), selection: selection, statistics: statistics,
		prepared: newFakePreparer(t).prepared, journal: &campaign.CampaignJournal{},
	}
	err = local.finishSeedCampaign()
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "campaign_publish" || hostError.Err.Error() != "campaign execution journal is not open" {
		t.Fatalf("error = %v, want campaign publication failure", err)
	}
	if local.batchComplete || local.summary.Attempted != 1 || local.summary.Succeeded != 1 || local.summary.StopReason != StopSeedsExhausted {
		t.Fatalf("completed=%v summary=%#v", local.batchComplete, local.summary)
	}
}

func TestLocalStoppedCancellationPreservesPriorFailure(t *testing.T) {
	primary := &HostError{Reason: "progress_output", Err: errors.New("progress failed")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancel()
	local := localCampaign{
		config:     campaignRequest{campaignRuntime: campaignRuntime{Backend: scriptProvider()}},
		overallCtx: context.Background(), activeCtx: ctx, activeCancel: cancel, stopped: true,
		hostFailure: primary, prepared: newFakePreparer(t).prepared,
		attempts: map[uint64]*seedAttempt{0: {job: runJob{seed: 1}}},
	}
	local.arrive(runCompletion{job: runJob{seed: 1}, result: backend.ExecutionResult{Cancelled: true}})
	local.advance()
	if local.hostFailure != primary || local.summary.Attempted != 1 || local.summary.Cancelled != 1 || local.active() != 0 {
		t.Fatalf("error = %v, summary = %#v, active=%d", local.hostFailure, local.summary, local.active())
	}
	if local.batchComplete {
		t.Fatal("cancelled campaign was completed")
	}
}

// TestCampaignRequestValidationStepPrecedence pins the error precedence across
// each adjacent pair of campaign request validation steps. Every case combines a
// fault owned by one step with a fault owned by the following step and expects
// the earlier step's exact error.
func TestCampaignRequestValidationStepPrecedence(t *testing.T) {
	bases := make(map[string]CampaignSpec)
	for _, test := range campaignOptionsCases() {
		bases[test.name] = test.config
	}
	for _, test := range []struct {
		name   string
		base   string
		change func(*CampaignSpec)
		want   string
	}{
		{name: "resume skips request checks", base: "resume", change: func(c *CampaignSpec) { c.Parallel = 0; c.Seeds = "" }},
		{name: "selection before shard", base: "sharded", change: func(c *CampaignSpec) { c.Seeds = ""; c.PlanSHA256 = "" }, want: "invalid seed selection: empty input"},
		{name: "shard before strategy bounds", base: "sharded", change: func(c *CampaignSpec) { c.PlanSHA256 = ""; c.MaxExecutions = 1 }, want: "sharded campaign requires a canonical plan identity"},
		{name: "strategy bounds before execution limits", base: "seed-default", change: func(c *CampaignSpec) { c.MaxExecutions = 1; c.Parallel = 0 }, want: "exploration bounds require the choice-exploration strategy"},
		{name: "execution limits before success retention", base: "seed-default", change: func(c *CampaignSpec) { c.Parallel = 0; c.SuccessArtifactLimit = 1 }, want: "parallelism must be positive"},
		{name: "success retention before evidence", base: "seed-default", change: func(c *CampaignSpec) { c.SuccessArtifactLimit = 1; c.CollectExecutionEvidence = true }, want: "disabled success retention does not accept capacity limits"},
		{name: "evidence before failure policy", base: "seed-default", change: func(c *CampaignSpec) { c.CollectExecutionEvidence = true; c.FailureBudget = 2 }, want: "execution evidence requires exactly one seed and semantic coverage"},
		{name: "failure policy before provider", base: "seed-default", change: func(c *CampaignSpec) { c.FailureBudget = 2; c.Backend = nil }, want: "failure budget is only configurable in budget mode"},
		{name: "provider before read-only mounts", base: "seed-default", change: func(c *CampaignSpec) { c.Backend = nil; c.IOROMounts = []string{"/tmp=data"} }, want: "backend provider is required"},
		{name: "read-only mounts before environment", base: "seed-default", change: func(c *CampaignSpec) { c.IOROMounts = []string{"/tmp=data"}; c.Environment = []string{"bad"} }, want: "read-only mounts require a target working directory"},
		{name: "environment before clock tick", base: "seed-default", change: func(c *CampaignSpec) { c.Environment = []string{"bad"}; c.ClockTick = "unknown" }, want: `invalid target environment entry "bad"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, found := bases[test.base]
			if !found {
				t.Fatalf("missing campaign options base %q", test.base)
			}
			test.change(&config)
			_, _, err := validateConfig(config)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != test.want {
				t.Fatalf("validateConfig() error = %q, want %q", got, test.want)
			}
		})
	}
}
