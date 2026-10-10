package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
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
			_, statErr := os.Stat(summary.CampaignPath + "/campaign.json")
			if !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("campaign publication error = %v, want absent", statErr)
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
			selection, err := ParseSeeds("1")
			if err != nil {
				t.Fatal(err)
			}
			controller, err := newShardedSeedController(selection, CampaignShard{}, nil, 1, PolicyAll, 1, CampaignResult{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := controller.Next(); !ok {
				t.Fatal("seed was not admitted")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			local := localCampaign{
				overallCtx: ctx, activeCancel: cancel, controller: controller,
				prepared: target.Prepared{Path: "missing prepared target"},
			}
			if test.prepared {
				local.prepared = newFakePreparer(t).prepared
			}
			if test.cancel {
				cancel()
			}
			local.handleCompletion(runCompletion{job: runJob{seed: 1}, err: test.runErr, result: execution.Result{
				WorldRecord: []byte("malformed World evidence"), WatchdogTimeout: true,
			}})
			var hostError *HostError
			if !errors.As(local.hostFailure, &hostError) || hostError.Reason != test.want {
				t.Fatalf("error = %v, want HostError reason %q", local.hostFailure, test.want)
			}
			if test.runErr != nil && !test.cancel && !errors.Is(local.hostFailure, primary) {
				t.Fatalf("error lost supervisor cause: %v", local.hostFailure)
			}
			if local.summary.Attempted != 1 || local.summary.Watchdogs != 0 || controller.Active() != 0 || !controller.Stopped() {
				t.Fatalf("summary = %#v, active=%d stopped=%v", local.summary, controller.Active(), controller.Stopped())
			}
		})
	}
}

func TestLocalFinalizationPreservesHostFailureBeforePublication(t *testing.T) {
	for _, priorFailure := range []bool{false, true} {
		t.Run(fmt.Sprintf("prior_failure=%t", priorFailure), func(t *testing.T) {
			selection, err := ParseSeeds("1")
			if err != nil {
				t.Fatal(err)
			}
			controller, err := newShardedSeedController(selection, CampaignShard{}, nil, 1, PolicyAll, 1, CampaignResult{})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			local := localCampaign{overallCtx: context.Background(), activeCancel: cancel, controller: controller}
			if priorFailure {
				local.hostFailure = errors.New("earlier failure")
			}
			primary := local.hostFailure
			local.completePartial(&campaign.ExecutionJournal{})
			if priorFailure {
				if local.hostFailure != primary || ctx.Err() != nil || controller.Stopped() {
					t.Fatalf("earlier error changed: error=%v context=%v stopped=%v", local.hostFailure, ctx.Err(), controller.Stopped())
				}
			} else {
				var hostError *HostError
				if !errors.As(local.hostFailure, &hostError) || hostError.Reason != "partial_cleanup" || hostError.Err.Error() != "cannot complete execution journal in \"\" state" {
					t.Fatalf("error = %v, want journal cleanup failure", local.hostFailure)
				}
				if !errors.Is(ctx.Err(), context.Canceled) || !controller.Stopped() {
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
	controller, err := newShardedSeedController(selection, CampaignShard{}, nil, 1, PolicyAll, 1, CampaignResult{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := controller.Next(); !ok {
		t.Fatal("seed was not admitted")
	}
	controller.Complete(campaign.CompletedSuccess())
	if _, ok := controller.Next(); ok {
		t.Fatal("selection has an unexpected second seed")
	}
	local := localCampaign{
		overallCtx: context.Background(), selection: selection, controller: controller,
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
	selection, err := ParseSeeds("1")
	if err != nil {
		t.Fatal(err)
	}
	controller, err := newShardedSeedController(selection, CampaignShard{}, nil, 1, PolicyAll, 1, CampaignResult{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := controller.Next(); !ok {
		t.Fatal("seed was not admitted")
	}
	controller.Stop()
	primary := &HostError{Reason: "progress_output", Err: errors.New("progress failed")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	local := localCampaign{
		overallCtx: ctx, activeCancel: cancel, controller: controller,
		hostFailure: primary, prepared: newFakePreparer(t).prepared,
	}
	local.handleCompletion(runCompletion{result: execution.Result{Cancelled: true}})
	if local.hostFailure != primary || local.summary.Attempted != 1 || local.summary.Cancelled != 1 || controller.Active() != 0 {
		t.Fatalf("error = %v, summary = %#v, active=%d", local.hostFailure, local.summary, controller.Active())
	}
	if local.batchComplete {
		t.Fatal("cancelled campaign was completed")
	}
}
