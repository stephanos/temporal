package campaignstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/frontier"
)

func TestPreparedCampaignMutationFaultsLeavePublishedOrResumableState(t *testing.T) {
	var observed []mutationPoint
	path, err := exercisePreparedCampaignMutations(t, func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	status, err := InspectCampaignLifecycle(path)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Published {
		t.Fatalf("unfaulted campaign lifecycle = %#v", status)
	}
	seenKinds := make(map[mutationKind]bool)
	for _, point := range observed {
		seenKinds[point.Kind] = true
	}
	for _, kind := range []mutationKind{mutationCreate, mutationFileSync, mutationDirectorySync, mutationRename, mutationDelete} {
		if !seenKinds[kind] {
			t.Fatalf("successful campaign observed no %s boundary: %#v", kind, observed)
		}
	}

	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			index := 0
			path, err := exercisePreparedCampaignMutations(t, func(point mutationPoint) error {
				if index == faultIndex {
					if point != expected {
						t.Fatalf("mutation %d = %#v, want %#v", index, point, expected)
					}
					index++
					return errors.New("injected storage failure")
				}
				index++
				return nil
			})
			if err == nil {
				t.Fatal("campaign completed despite injected storage failure")
			}
			status, inspectErr := InspectCampaignLifecycle(path)
			if inspectErr != nil {
				t.Fatal(inspectErr)
			}
			if status.Published {
				return
			}
			if status.State != LifecycleRecoverableFailure || !status.Resumable {
				t.Fatalf("fault at %#v left lifecycle %#v", expected, status)
			}
			if _, recoverErr := RecoverCampaign(context.Background(), path); recoverErr != nil {
				t.Fatalf("recover fault at %#v: %v", expected, recoverErr)
			}
		})
	}
}

func TestCampaignCreationMutationFaultsLeaveNoInvalidBatch(t *testing.T) {
	var observed []mutationPoint
	root := t.TempDir()
	journal, err := NewCampaignJournal(withMutationHook(context.Background(), func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	}), CampaignConfig{Root: root, CampaignID: "run-creation-fault", Selection: "7", SelectionCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	if len(observed) == 0 {
		t.Fatal("campaign creation observed no mutation boundaries")
	}

	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			root := t.TempDir()
			index := 0
			ctx := withMutationHook(context.Background(), func(point mutationPoint) error {
				if index == faultIndex {
					if point != expected {
						t.Fatalf("mutation %d = %#v, want %#v", index, point, expected)
					}
					index++
					return errors.New("injected creation failure")
				}
				index++
				return nil
			})
			journal, err := NewCampaignJournal(ctx, CampaignConfig{Root: root, CampaignID: "run-creation-fault", Selection: "7", SelectionCount: 1})
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
				t.Fatalf("fault at %#v left invalid visible batch: %v", expected, inspectErr)
			}
		})
	}
}

func TestCampaignPreparationMutationFaultsRetainLifecycle(t *testing.T) {
	var observed []mutationPoint
	if _, err := exercisePreparationMutations(t, func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(observed) == 0 {
		t.Fatal("campaign preparation observed no mutation boundaries")
	}
	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			index := 0
			path, err := exercisePreparationMutations(t, func(point mutationPoint) error {
				if index == faultIndex {
					if point != expected {
						t.Fatalf("mutation %d = %#v, want %#v", index, point, expected)
					}
					index++
					return errors.New("injected preparation failure")
				}
				index++
				return nil
			})
			if err == nil {
				t.Fatal("campaign preparation completed despite injected storage failure")
			}
			status, inspectErr := InspectCampaignLifecycle(path)
			if inspectErr != nil {
				t.Fatal(inspectErr)
			}
			if status.State != LifecycleRecoverableFailure {
				t.Fatalf("fault at %#v left lifecycle %#v", expected, status)
			}
		})
	}
}

func TestPublishedRecoveryMutationFaultsRemainIdempotent(t *testing.T) {
	var observed []mutationPoint
	path := publishedCampaignWithStalePrivateState(t)
	ctx := withMutationHook(context.Background(), func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	})
	if _, err := RecoverCampaign(ctx, path); err != nil {
		t.Fatal(err)
	}
	seenDelete := false
	seenDirectorySync := false
	for _, point := range observed {
		seenDelete = seenDelete || point.Kind == mutationDelete
		seenDirectorySync = seenDirectorySync || point.Kind == mutationDirectorySync
	}
	if !seenDelete || !seenDirectorySync {
		t.Fatalf("recovery mutation boundaries = %#v", observed)
	}

	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			path := publishedCampaignWithStalePrivateState(t)
			index := 0
			ctx := withMutationHook(context.Background(), func(point mutationPoint) error {
				if index == faultIndex {
					if point != expected {
						t.Fatalf("mutation %d = %#v, want %#v", index, point, expected)
					}
					index++
					return errors.New("injected recovery failure")
				}
				index++
				return nil
			})
			if _, err := RecoverCampaign(ctx, path); err == nil {
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
		})
	}
}

func TestResumeMutationFaultsLeaveJournalRetryable(t *testing.T) {
	var observed []mutationPoint
	path := interruptedCampaignForResumeFault(t)
	ctx := withMutationHook(context.Background(), func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	})
	resumed, _, err := ResumeCampaignJournal(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := resumed.Close(); err != nil {
		t.Fatal(err)
	}
	seenArchiveRename := false
	for _, point := range observed {
		seenArchiveRename = seenArchiveRename || point.Kind == mutationRename && point.Operation == "resume-archive"
	}
	if !seenArchiveRename {
		t.Fatalf("resume mutation boundaries = %#v", observed)
	}

	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			path := interruptedCampaignForResumeFault(t)
			index := 0
			ctx := withMutationHook(context.Background(), func(point mutationPoint) error {
				if index == faultIndex {
					if point != expected {
						t.Fatalf("mutation %d = %#v, want %#v", index, point, expected)
					}
					index++
					return errors.New("injected resume failure")
				}
				index++
				return nil
			})
			if resumed, _, err := ResumeCampaignJournal(ctx, path); err == nil {
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
			resumed, _, err := ResumeCampaignJournal(context.Background(), path)
			if err != nil {
				t.Fatalf("retry resume after %#v: %v", expected, err)
			}
			if err := resumed.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFrontierCommitMutationFaultsLeaveRoundRecoverable(t *testing.T) {
	var observed []mutationPoint
	path, config, err := exerciseFrontierCommitMutations(t, func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ResumeFrontierJournal(context.Background(), path, config, 1<<20); err != nil {
		t.Fatal(err)
	}
	seenPublish := false
	seenCandidateDelete := false
	for _, point := range observed {
		seenPublish = seenPublish || point.Kind == mutationRename && point.Operation == "frontier-publish"
		seenCandidateDelete = seenCandidateDelete || point.Kind == mutationDelete && point.Operation == "frontier-candidate-work"
	}
	if !seenPublish || !seenCandidateDelete {
		t.Fatalf("frontier commit mutation boundaries = %#v", observed)
	}

	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			index := 0
			path, config, err := exerciseFrontierCommitMutations(t, func(point mutationPoint) error {
				if index == faultIndex {
					if point != expected {
						t.Fatalf("mutation %d = %#v, want %#v", index, point, expected)
					}
					index++
					return errors.New("injected frontier failure")
				}
				index++
				return nil
			})
			if err == nil {
				t.Fatal("frontier commit completed despite injected storage failure")
			}
			if _, _, _, err := ResumeFrontierJournal(context.Background(), path, config, 1<<20); err != nil {
				t.Fatalf("recover frontier fault at %#v: %v", expected, err)
			}
		})
	}
}

func TestFrontierCreationMutationFaultsLeaveJournalRetryable(t *testing.T) {
	var observed []mutationPoint
	path := privateDirectory(t)
	state := testFrontierState(t)
	if _, err := NewFrontierJournal(withMutationHook(context.Background(), func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	}), path, state, 1<<20); err != nil {
		t.Fatal(err)
	}
	if len(observed) == 0 {
		t.Fatal("frontier creation observed no mutation boundaries")
	}

	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			path := privateDirectory(t)
			index := 0
			ctx := withMutationHook(context.Background(), func(point mutationPoint) error {
				if index == faultIndex {
					if point != expected {
						t.Fatalf("mutation %d = %#v, want %#v", index, point, expected)
					}
					index++
					return errors.New("injected frontier creation failure")
				}
				index++
				return nil
			})
			if _, err := NewFrontierJournal(ctx, path, state, 1<<20); err == nil {
				t.Fatal("frontier creation completed despite injected storage failure")
			}
			if _, err := NewFrontierJournal(context.Background(), path, state, 1<<20); err != nil {
				t.Fatalf("retry frontier creation after %#v: %v", expected, err)
			}
		})
	}
}

func TestFrontierStageMutationFaultsLeaveRoundRetryable(t *testing.T) {
	var observed []mutationPoint
	path := privateDirectory(t)
	state := testFrontierState(t)
	controller := &mutationController{enabled: false, inject: func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	}}
	journal, err := NewFrontierJournal(withMutationHook(context.Background(), controller.observe), path, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	controller.enabled = true
	if _, err := journal.StageRound(round); err != nil {
		t.Fatal(err)
	}
	if len(observed) == 0 {
		t.Fatal("frontier staging observed no mutation boundaries")
	}

	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			path := privateDirectory(t)
			state := testFrontierState(t)
			index := 0
			controller := &mutationController{enabled: false}
			ctx := withMutationHook(context.Background(), controller.observe)
			journal, err := NewFrontierJournal(ctx, path, state, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			round, _ := state.NextRound()
			controller.inject = func(point mutationPoint) error {
				if index == faultIndex {
					if point != expected {
						t.Fatalf("mutation %d = %#v, want %#v", index, point, expected)
					}
					index++
					return errors.New("injected frontier staging failure")
				}
				index++
				return nil
			}
			controller.enabled = true
			if _, err := journal.StageRound(round); err == nil {
				t.Fatal("frontier staging completed despite injected storage failure")
			}
			controller.enabled = false
			if _, err := journal.StageRound(round); err != nil {
				t.Fatalf("retry frontier staging after %#v: %v", expected, err)
			}
		})
	}
}

func exercisePreparedCampaignMutations(t *testing.T, inject mutationHook) (string, error) {
	t.Helper()
	controller := &mutationController{enabled: false, inject: inject}
	ctx := withMutationHook(context.Background(), controller.observe)
	journal, err := NewCampaignJournal(ctx, CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-mutation-fault", Selection: "7-9", SelectionCount: 3,
	})
	if err != nil {
		return "", err
	}
	path := journal.Path()
	finish := func(operationErr error) (string, error) {
		controller.enabled = false
		status, inspectErr := InspectCampaignLifecycle(path)
		if inspectErr == nil && !status.Published && operationErr != nil {
			operationErr = errors.Join(operationErr, journal.Fail("storage_failure", operationErr))
		}
		return path, errors.Join(operationErr, journal.Close())
	}
	if err := journal.BeginPreparation(); err != nil {
		return finish(err)
	}
	preparedPath := filepath.Join(journal.PreparedPath(), "build", "target")
	if err := os.MkdirAll(filepath.Dir(preparedPath), 0o700); err != nil {
		return finish(err)
	}
	target := []byte("prepared target")
	if err := os.WriteFile(preparedPath, target, 0o500); err != nil {
		return finish(err)
	}
	if err := journal.RecordPlan(testBatchPlan(journal, evidence.HashBytes(target), uint64(len(target)))); err != nil {
		return finish(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		return finish(err)
	}
	controller.enabled = true
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
	if _, err := stdout.Write([]byte("output")); err != nil {
		return finish(errors.Join(err, stdout.Close()))
	}
	if err := run.CloseOutput("stdout", stdout); err != nil {
		return finish(err)
	}
	for _, state := range []ExecutionState{ExecutionExited, ExecutionCaptured, ExecutionClassified} {
		if err := run.Transition(state); err != nil {
			return finish(err)
		}
	}
	if err := journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit",
	}); err != nil {
		return finish(err)
	}
	if err := run.Complete(); err != nil {
		return finish(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		return finish(err)
	}
	return finish(nil)
}

func exercisePreparationMutations(t *testing.T, inject mutationHook) (string, error) {
	t.Helper()
	controller := &mutationController{enabled: false, inject: inject}
	ctx := withMutationHook(context.Background(), controller.observe)
	journal, err := NewCampaignJournal(ctx, CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-preparation-fault", Selection: "7-9", SelectionCount: 3,
	})
	if err != nil {
		return "", err
	}
	path := journal.Path()
	finish := func(operationErr error) (string, error) {
		controller.enabled = false
		if operationErr != nil {
			operationErr = errors.Join(operationErr, journal.Fail("storage_failure", operationErr))
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
		return finish(err)
	}
	target := []byte("prepared target")
	if err := os.WriteFile(preparedPath, target, 0o500); err != nil {
		return finish(err)
	}
	controller.enabled = true
	if err := journal.RecordPlan(testBatchPlan(journal, evidence.HashBytes(target), uint64(len(target)))); err != nil {
		return finish(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		return finish(err)
	}
	return finish(nil)
}

type mutationController struct {
	enabled bool
	inject  mutationHook
}

func publishedCampaignWithStalePrivateState(t *testing.T) string {
	t.Helper()
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-recovery-fault", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(ExecutionRecord{SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{journal.PreparedPath(), filepath.Join(journal.Path(), ".partial", "preparation"), filepath.Join(journal.Path(), ".partial", "batch")} {
		if err := makePrivateDirectories(path); err != nil {
			t.Fatal(err)
		}
	}
	return journal.Path()
}

func interruptedCampaignForResumeFault(t *testing.T) string {
	t.Helper()
	journal := newPreparedLifecycleJournal(t, "run-resume-fault")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(ExecutionRecord{SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
		t.Fatal(err)
	}
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

func exerciseFrontierCommitMutations(t *testing.T, inject mutationHook) (string, frontier.Config, error) {
	t.Helper()
	controller := &mutationController{enabled: false, inject: inject}
	ctx := withMutationHook(context.Background(), controller.observe)
	path := privateDirectory(t)
	state := testFrontierState(t)
	journal, err := NewFrontierJournal(ctx, path, state, 1<<20)
	if err != nil {
		return path, state.Config, err
	}
	round, ok := state.NextRound()
	if !ok {
		return path, state.Config, errors.New("frontier root round is unavailable")
	}
	staged, err := journal.StageRound(round)
	if err != nil {
		return path, state.Config, err
	}
	_, segment, err := frontier.CommitRound(state, round, []frontier.Result{{
		CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: evidence.HashBytes([]byte("success")),
	}})
	if err != nil {
		return path, state.Config, err
	}
	controller.enabled = true
	return path, state.Config, journal.CommitRound(staged, segment)
}

func (controller *mutationController) observe(point mutationPoint) error {
	if !controller.enabled {
		return nil
	}
	return controller.inject(point)
}
