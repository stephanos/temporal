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
	require.NoError(t, os.WriteFile(fixture, []byte(`import Temporal.Families.NexusCancellation.Feature

open Umpire3
open Umpire3.Temporal.Feature.NexusCancellationFencing

theorem successAfterCancellationWins :
    behavior.Step .smoke .cancelled .completeSuccess .succeeded := by
  simp [behavior, successors]
`), 0o600))

	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", fixture)
	command.Dir = "model"
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "error:")
}

func TestLeanRejectsUnsafeNexusRefinement(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "BadNexusRefinement.lean")
	require.NoError(t, os.WriteFile(fixture, []byte(`import Temporal.Families.NexusCancellation.Refinement

open Umpire3
open Umpire3.Temporal.Refinement.NexusCancellationFencing

theorem unsafeStepProjects :
    StepSimulation System.mutatedBehavior Feature.behavior Projects actionMap := by
  exact stepSimulates
`), 0o600))

	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", fixture)
	command.Dir = "model"
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "error:")
}

func TestLeanRejectsNexusRelationWithoutCancellationEvidence(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "BadNexusRelation.lean")
	require.NoError(t, os.WriteFile(fixture, []byte(`import Temporal.Families.NexusCancellation.Refinement

open Umpire3.Temporal.Refinement.NexusCancellationFencing

theorem cancellationEvidenceCanBeDropped : Projects System.afterCancellationAccepted Feature.initial := by
  rfl
`), 0o600))

	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", fixture)
	command.Dir = "model"
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "error:")
}
