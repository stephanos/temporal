package umpire3_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeanRejectsIncompleteExecutableModel(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "BadExecutable.lean")
	require.NoError(t, os.WriteFile(fixture, []byte(`import Umpire3.Executable

open Umpire3

inductive State where
  | off
  | on

inductive Action where
  | enable

def model : TransitionSystem where
  State := State
  Action := Action
  Initial := fun state => state = .off
  Step := fun state action next => state = .off ∧ action = .enable ∧ next = .on

def incomplete : ExecutableModel model where
  next := fun _ _ => []
  next_iff := by
    intro state action next
    simp [model]
`), 0o600))

	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", fixture)
	command.Dir = "model"
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "error:")
}

func TestLeanRejectsFalseSafetyProperty(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "BadSafety.lean")
	require.NoError(t, os.WriteFile(fixture, []byte(`import Umpire3.Property

open Umpire3

inductive State where
  | bad

def model : TransitionSystem where
  State := State
  Action := Unit
  Initial := fun _ => True
  Step := fun _ _ _ => False

theorem falseSafety : Safety model (fun _ => False) := by
  intro state reachable
  simp
`), 0o600))

	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", fixture)
	command.Dir = "model"
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "error:")
}

func TestLeanRejectsSuccessAfterCancellationWins(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "BadNexusCancellation.lean")
	require.NoError(t, os.WriteFile(fixture, []byte(`import Temporal.Product.Nexus

open Umpire3
open Umpire3.Temporal.Product.Nexus

theorem successAfterCancellationWins :
    product.Step cancelled .completeSuccess succeeded := by
  simp [product, step, cancelled, succeeded]
`), 0o600))

	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", fixture)
	command.Dir = "model"
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "error:")
}

func TestLeanRejectsUnsafeNexusRefinement(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "BadNexusRefinement.lean")
	require.NoError(t, os.WriteFile(fixture, []byte(`import Umpire3Tests.TemporalSystem

open Umpire3
open Umpire3.Temporal.System.NexusTasks
open Umpire3.Temporal.System.NexusTasks.Tests

theorem unsafeStepProjects :
    ∃ nextProduct,
      Umpire3.Temporal.Product.Nexus.product.StepStar staleReturned.visible nextProduct ∧
      Projects unsafePersisted nextProduct := by
  exact ⟨unsafePersisted.visible, ⟨[.completeSuccess], Runs.cons (by
    simp [Umpire3.Temporal.Product.Nexus.step, staleReturned, retried, ownershipChanged,
      cancellationCommitted, cancellationRequested, dispatched, scheduled, initial, noProgress])
    (Runs.nil unsafePersisted.visible)⟩, rfl⟩
`), 0o600))

	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", fixture)
	command.Dir = "model"
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "error:")
}

func TestLeanRejectsNexusRelationWithoutCancellationEvidence(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "BadNexusRelation.lean")
	require.NoError(t, os.WriteFile(fixture, []byte(`import Umpire3Tests.TemporalSystem

open Umpire3.Temporal.Product.Nexus
open Umpire3.Temporal.System.NexusTasks
open Umpire3.Temporal.System.NexusTasks.Tests

theorem cancellationEvidenceCanBeDropped : Projects cancellationRequested initial := by
  rfl
`), 0o600))

	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", fixture)
	command.Dir = "model"
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "error:")
}
