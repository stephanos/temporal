package campaign

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/exploration"
)

func TestExplorationCommitMutationFaultsLeaveRoundRecoverable(t *testing.T) {
	var observed []mutationPoint
	path, config, err := exerciseExplorationCommitMutations(t, func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ResumeExplorationJournal(context.Background(), path, config, 1<<20); err != nil {
		t.Fatal(err)
	}
	seenPublish := false
	seenCandidateDelete := false
	for _, point := range observed {
		seenPublish = seenPublish || point.Kind == mutationRename && point.Operation == "exploration-publish"
		seenCandidateDelete = seenCandidateDelete || point.Kind == mutationDelete && point.Operation == "exploration-candidate-work"
	}
	if !seenPublish || !seenCandidateDelete {
		t.Fatalf("exploration commit mutation boundaries = %#v", observed)
	}

	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			index := 0
			path, config, err := exerciseExplorationCommitMutations(t, func(point mutationPoint) error {
				if index == faultIndex {
					if point != expected {
						t.Fatalf("mutation %d = %#v, want %#v", index, point, expected)
					}
					index++
					return errors.New("injected exploration failure")
				}
				index++
				return nil
			})
			if err == nil {
				t.Fatal("exploration commit completed despite injected storage failure")
			}
			if _, _, _, err := ResumeExplorationJournal(context.Background(), path, config, 1<<20); err != nil {
				t.Fatalf("recover exploration fault at %#v: %v", expected, err)
			}
		})
	}
}

func TestExplorationCreationMutationFaultsLeaveJournalRetryable(t *testing.T) {
	var observed []mutationPoint
	path := privateDirectory(t)
	state := testExplorationState(t)
	if _, err := NewExplorationJournal(withMutationHook(context.Background(), func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	}), path, state, 1<<20); err != nil {
		t.Fatal(err)
	}
	if len(observed) == 0 {
		t.Fatal("exploration creation observed no mutation boundaries")
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
					return errors.New("injected exploration creation failure")
				}
				index++
				return nil
			})
			if _, err := NewExplorationJournal(ctx, path, state, 1<<20); err == nil {
				t.Fatal("exploration creation completed despite injected storage failure")
			}
			if _, err := NewExplorationJournal(context.Background(), path, state, 1<<20); err != nil {
				t.Fatalf("retry exploration creation after %#v: %v", expected, err)
			}
		})
	}
}

func TestExplorationStageMutationFaultsLeaveRoundRetryable(t *testing.T) {
	var observed []mutationPoint
	path := privateDirectory(t)
	state := testExplorationState(t)
	controller := &mutationController{enabled: false, inject: func(point mutationPoint) error {
		observed = append(observed, point)
		return nil
	}}
	journal, err := NewExplorationJournal(withMutationHook(context.Background(), controller.observe), path, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	controller.enabled = true
	if _, err := journal.StageRound(round); err != nil {
		t.Fatal(err)
	}
	if len(observed) == 0 {
		t.Fatal("exploration staging observed no mutation boundaries")
	}

	for faultIndex, expected := range observed {
		t.Run(fmt.Sprintf("%03d-%s-%s", faultIndex, expected.Kind, expected.Operation), func(t *testing.T) {
			path := privateDirectory(t)
			state := testExplorationState(t)
			index := 0
			controller := &mutationController{enabled: false}
			ctx := withMutationHook(context.Background(), controller.observe)
			journal, err := NewExplorationJournal(ctx, path, state, 1<<20)
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
					return errors.New("injected exploration staging failure")
				}
				index++
				return nil
			}
			controller.enabled = true
			if _, err := journal.StageRound(round); err == nil {
				t.Fatal("exploration staging completed despite injected storage failure")
			}
			controller.enabled = false
			if _, err := journal.StageRound(round); err != nil {
				t.Fatalf("retry exploration staging after %#v: %v", expected, err)
			}
		})
	}
}

type mutationController struct {
	enabled bool
	inject  mutationHook
}

func exerciseExplorationCommitMutations(t *testing.T, inject mutationHook) (string, exploration.ChoiceConfig, error) {
	t.Helper()
	controller := &mutationController{enabled: false, inject: inject}
	ctx := withMutationHook(context.Background(), controller.observe)
	path := privateDirectory(t)
	state := testExplorationState(t)
	journal, err := NewExplorationJournal(ctx, path, state, 1<<20)
	if err != nil {
		return path, state.Config, err
	}
	round, ok := state.NextRound()
	if !ok {
		return path, state.Config, errors.New("exploration root round is unavailable")
	}
	staged, err := journal.StageRound(round)
	if err != nil {
		return path, state.Config, err
	}
	_, segment, err := exploration.CommitChoiceRound(state, round, []exploration.ChoiceResult{{
		CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: record.HashBytes([]byte("success")),
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
