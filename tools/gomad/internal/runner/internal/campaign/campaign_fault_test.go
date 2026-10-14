package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
)

// faultPlan injects one failure at the faultIndex-th observed point; with
// partial set, a "log-write" fault writes that many frame bytes first.
type faultPlan struct {
	faultIndex int
	partial    int
	index      int
	fired      bool
	expected   mutationPoint
	synced     int
}

func (plan *faultPlan) observe(t *testing.T) mutationHook {
	return func(point mutationPoint) error {
		defer func() { plan.index++ }()
		if plan.index != plan.faultIndex {
			if point.Kind == "log-synced" && !plan.fired {
				plan.synced++
			}
			return nil
		}
		if point != plan.expected {
			t.Fatalf("point %d = %#v, want %#v", plan.index, point, plan.expected)
		}
		plan.fired = true
		if plan.partial != 0 {
			return &logstore.PartialWrite{Bytes: plan.partial}
		}
		return errors.New("injected storage failure")
	}
}

// exerciseCampaignLifecycle drives a campaign from prepared through publication
// with the hook enabled after creation. It returns the campaign path, the
// journal (closed) and the first error.
func exerciseCampaignLifecycle(t *testing.T, hook mutationHook) (string, error) {
	t.Helper()
	controller := &mutationController{inject: hook}
	ctx := withMutationHook(context.Background(), controller.observe)
	journal, err := NewCampaignJournal(ctx, CampaignConfig{Root: t.TempDir(), CampaignID: "run-log-fault", Selection: "7-9", SelectionCount: 3})
	if err != nil {
		t.Fatal(err)
	}
	path := journal.Path()
	finish := func(operationErr error) (string, error) {
		controller.enabled = false
		if operationErr != nil {
			if failErr := journal.Fail("storage_failure", operationErr); failErr == nil && isLogPointFailure(operationErr) {
				t.Fatalf("writer accepted a record after an uncertain append: %v", operationErr)
			}
		}
		return path, errors.Join(operationErr, journal.Close())
	}
	controller.enabled = true
	if err := journal.BeginPreparation(); err != nil {
		return finish(err)
	}
	preparedPath := filepath.Join(journal.PreparedPath(), "build", "target")
	controller.enabled = false
	if err := os.MkdirAll(filepath.Dir(preparedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	target := []byte("prepared target")
	if err := os.WriteFile(preparedPath, target, 0o500); err != nil {
		t.Fatal(err)
	}
	controller.enabled = true
	if err := journal.RecordPlan(testBatchPlan(journal, record.HashBytes(target), uint64(len(target)))); err != nil {
		return finish(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		return finish(err)
	}
	if err := journal.StartExecutions(); err != nil {
		return finish(err)
	}
	run, err := journal.BeginExecution(0, 7)
	if err != nil {
		return finish(err)
	}
	if err := run.Transition(ExecutionStarting); err != nil {
		return finish(err)
	}
	stdout, err := run.CreateOutput("stdout")
	if err != nil {
		return finish(err)
	}
	if err := run.CloseOutput("stdout", stdout); err != nil {
		return finish(err)
	}
	for _, state := range []executionState{ExecutionExited, ExecutionCaptured, ExecutionClassified} {
		if err := run.Transition(state); err != nil {
			return finish(err)
		}
	}
	for ordinal, seed := range []uint64{7, 8, 9} {
		if err := journal.AppendExecution(successExecution(uint64(ordinal), seed)); err != nil {
			return finish(err)
		}
	}
	if err := run.Complete(); err != nil {
		return finish(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 3, Succeeded: 3, StopReason: "seeds_exhausted"}); err != nil {
		return finish(err)
	}
	return finish(nil)
}

func isLogPointFailure(err error) bool {
	return errors.Is(err, logstore.ErrPoisoned)
}

func observeCampaignPoints(t *testing.T, exercise func(*testing.T, mutationHook) (string, error)) ([]mutationPoint, string) {
	t.Helper()
	var observed []mutationPoint
	path, err := exercise(t, func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return observed, path
}

func TestCampaignLogFaultsAtEveryAppendAndSyncPointLeaveReplayableState(t *testing.T) {
	observed, referencePath := observeCampaignPoints(t, exerciseCampaignLifecycle)
	reference, _ := campaignLogRecords(t, referencePath)
	assertLifecycle(t, referencePath, lifecyclePublished, lifecyclePublished, false)
	kinds := map[mutationKind]bool{}
	for _, point := range observed {
		kinds[point.Kind] = true
	}
	for _, kind := range []mutationKind{"log-write", "log-sync", "log-synced", mutationCreate, mutationFileSync, mutationDelete} {
		if !kinds[kind] {
			t.Fatalf("campaign lifecycle observed no %s point: %#v", kind, observed)
		}
	}
	type variant struct {
		index   int
		partial int
	}
	var variants []variant
	for index, point := range observed {
		variants = append(variants, variant{index: index})
		if point.Kind == "log-write" {
			variants = append(variants, variant{index: index, partial: 1}, variant{index: index, partial: 17})
		}
	}
	for _, test := range variants {
		expected := observed[test.index]
		t.Run(fmt.Sprintf("%03d-%s-%s-partial%d", test.index, expected.Kind, expected.Operation, test.partial), func(t *testing.T) {
			plan := &faultPlan{faultIndex: test.index, partial: test.partial, expected: expected}
			path, err := exerciseCampaignLifecycle(t, plan.observe(t))
			if err == nil || !plan.fired {
				t.Fatalf("campaign completed despite the fault at %#v", expected)
			}
			// One envelope precedes the hook; every acknowledged append
			// survives, and a frame whose write completed survives once.
			durable := 1 + plan.synced
			if expected.Kind == "log-sync" || expected.Kind == "log-synced" {
				durable++
			}
			replayed, report := campaignLogRecords(t, path)
			prefix := replayed
			if expected.Kind[:min(4, len(expected.Kind))] != "log-" && len(prefix) == durable+1 && prefix[durable] == kindLifecycle {
				prefix = prefix[:durable]
			}
			if !slices.Equal(prefix, reference[:durable]) {
				t.Fatalf("fault at %#v replayed %v, want the reference prefix %v", expected, replayed, reference[:durable])
			}
			if (test.partial != 0) != (report.Tail != logstore.TailNone) {
				t.Fatalf("fault at %#v left tail %v", expected, report.Tail)
			}
			status, err := InspectCampaignLifecycle(path)
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case status.Published:
				if _, err := RecoverCampaign(context.Background(), path); err != nil {
					t.Fatalf("recover published campaign after %#v: %v", expected, err)
				}
				if again, err := RecoverCampaign(context.Background(), path); err != nil || again.Changed {
					t.Fatalf("repeated recovery after %#v = %#v, %v", expected, again, err)
				}
			case status.Resumable:
				resumed, state, err := ResumeCampaignJournal(context.Background(), path)
				if err != nil {
					t.Fatalf("resume after %#v: %v", expected, err)
				}
				if err := resumed.Close(); err != nil {
					t.Fatal(err)
				}
				committed := 0
				for _, kind := range replayed {
					if kind == kindOutcome {
						committed++
					}
				}
				if len(state.Executions) != committed {
					t.Fatalf("resume after %#v restored %d outcomes, want %d", expected, len(state.Executions), committed)
				}
				if after, _ := campaignLogRecords(t, path); after[len(after)-1] != kindLifecycle {
					t.Fatalf("resume after %#v did not commit its transition: %v", expected, after)
				}
			default:
				if status.State != lifecycleRecoverableFailure && status.State != lifecyclePlanned || status.LastStableState != lifecyclePlanned {
					t.Fatalf("fault at %#v left lifecycle %#v", expected, status)
				}
				if _, err := RecoverCampaign(context.Background(), path); err == nil || !IsIntegrityError(err) {
					t.Fatalf("recover unprepared campaign after %#v: %v", expected, err)
				}
			}
		})
	}
}

func TestCampaignCreationFaultsLeaveNoInvalidCampaign(t *testing.T) {
	var observed []mutationPoint
	journal, err := NewCampaignJournal(withMutationHook(context.Background(), func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	}), CampaignConfig{Root: t.TempDir(), CampaignID: "run-creation-fault", Selection: "7", SelectionCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			root := t.TempDir()
			plan := &faultPlan{faultIndex: faultIndex, expected: expected}
			journal, err := NewCampaignJournal(withMutationHook(context.Background(), plan.observe(t)), CampaignConfig{Root: root, CampaignID: "run-creation-fault", Selection: "7", SelectionCount: 1})
			if err == nil {
				if closeErr := journal.Close(); closeErr != nil {
					t.Fatal(closeErr)
				}
				t.Fatal("campaign creation completed despite injected storage failure")
			}
			path := filepath.Join(root, "v1", "run-creation-fault")
			if _, statErr := os.Lstat(path); errors.Is(statErr, os.ErrNotExist) {
				return
			} else if statErr != nil {
				t.Fatal(statErr)
			}
			if _, inspectErr := InspectCampaignLifecycle(path); inspectErr != nil {
				t.Fatalf("fault at %#v left an invalid visible campaign: %v", expected, inspectErr)
			}
		})
	}
}

// interruptedCampaignWithStaleWork is a running campaign with one committed
// outcome, an uncommitted attempt's work and an unreferenced success artifact.
func interruptedCampaignWithStaleWork(t *testing.T) string {
	t.Helper()
	fixture := newFailureCampaign(t)
	journal := fixture.journal
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(fixture.publishFailure(t, 0, 7, "")); err != nil {
		t.Fatal(err)
	}
	fixture.publishFailure(t, 1, 8, "orphan_reason")
	incomplete, err := journal.BeginExecution(1, 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := incomplete.Transition(ExecutionStarting); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	return journal.Path()
}

func TestResumeFaultsLeaveCampaignRetryable(t *testing.T) {
	observed, _ := observeCampaignPoints(t, func(t *testing.T, hook mutationHook) (string, error) {
		path := interruptedCampaignWithStaleWork(t)
		resumed, _, err := ResumeCampaignJournal(withMutationHook(context.Background(), hook), path)
		if err != nil {
			return path, err
		}
		return path, resumed.Close()
	})
	if !slices.ContainsFunc(observed, func(point mutationPoint) bool { return point.Operation == "owned-payload-sweep" }) || !slices.ContainsFunc(observed, func(point mutationPoint) bool { return point.Kind == "log-synced" }) {
		t.Fatalf("resume points = %#v", observed)
	}
	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			path := interruptedCampaignWithStaleWork(t)
			plan := &faultPlan{faultIndex: faultIndex, expected: expected}
			if resumed, _, err := ResumeCampaignJournal(withMutationHook(context.Background(), plan.observe(t)), path); err == nil {
				if closeErr := resumed.Close(); closeErr != nil {
					t.Fatal(closeErr)
				}
				t.Fatal("resume completed despite injected storage failure")
			}
			status, err := InspectCampaignLifecycle(path)
			if err != nil {
				t.Fatal(err)
			}
			if !status.Resumable {
				t.Fatalf("fault at %#v left lifecycle %#v", expected, status)
			}
			resumed, state, err := ResumeCampaignJournal(context.Background(), path)
			if err != nil {
				t.Fatalf("retry resume after %#v: %v", expected, err)
			}
			if err := resumed.Close(); err != nil {
				t.Fatal(err)
			}
			if len(state.Executions) != 1 || len(directoryNames(t, filepath.Join(path, "failures"))) != 1 {
				t.Fatalf("retried resume after %#v = %#v", expected, state)
			}
		})
	}
}

// publishedCampaignWithStalePreparedTarget publishes a campaign whose
// post-publication cleanup was interrupted.
func publishedCampaignWithStalePreparedTarget(t *testing.T) string {
	t.Helper()
	journal := newPreparedLifecycleJournal(t, "run-recovery-fault")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	for ordinal, seed := range []uint64{7, 8, 9} {
		if err := journal.AppendExecution(successExecution(uint64(ordinal), seed)); err != nil {
			t.Fatal(err)
		}
	}
	controller := &mutationController{enabled: true, inject: func(point mutationPoint) error {
		if point.Operation == "prepared-target" {
			return errors.New("injected cleanup failure")
		}
		return nil
	}}
	journal.ctx = withMutationHook(context.Background(), controller.observe)
	if err := journal.Publish(CampaignSummary{Attempted: 3, Succeeded: 3, StopReason: "seeds_exhausted"}); err == nil {
		t.Fatal("Publish() ignored its interrupted cleanup")
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	return journal.Path()
}

func TestPublishedRecoveryFaultsRemainIdempotent(t *testing.T) {
	observed, _ := observeCampaignPoints(t, func(t *testing.T, hook mutationHook) (string, error) {
		path := publishedCampaignWithStalePreparedTarget(t)
		_, err := RecoverCampaign(withMutationHook(context.Background(), hook), path)
		return path, err
	})
	if !slices.ContainsFunc(observed, func(point mutationPoint) bool {
		return point.Kind == mutationDelete && point.Operation == "prepared-target"
	}) {
		t.Fatalf("recovery points = %#v", observed)
	}
	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			path := publishedCampaignWithStalePreparedTarget(t)
			plan := &faultPlan{faultIndex: faultIndex, expected: expected}
			if _, err := RecoverCampaign(withMutationHook(context.Background(), plan.observe(t)), path); err == nil {
				t.Fatal("recovery completed despite injected storage failure")
			} else if IsIntegrityError(err) {
				t.Fatalf("storage fault at %#v was classified as integrity: %v", expected, err)
			}
			status, err := InspectCampaignLifecycle(path)
			if err != nil {
				t.Fatal(err)
			}
			if !status.Published {
				t.Fatalf("fault at %#v lost published authority: %#v", expected, status)
			}
			if _, err := RecoverCampaign(context.Background(), path); err != nil {
				t.Fatalf("retry recovery after %#v: %v", expected, err)
			}
			if again, err := RecoverCampaign(context.Background(), path); err != nil || again.Changed {
				t.Fatalf("repeated recovery after %#v = %#v, %v", expected, again, err)
			}
		})
	}
}
