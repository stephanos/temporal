import Umpire.Exploration
import Temporal.Feature.Nexus.Experimental.VariationSpace

/-! Bounded Exploration over the checked experimental Nexus variation Space. -/

namespace Temporal.Feature.Nexus.Experimental.Exploration

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations
open Temporal.Feature.Nexus.Experimental.VariationSpace

private theorem queryResult_target
    (query : CheckedQuery LawStatement)
    (resultEq : queryResult = .ok query) :
    query.target = target := by
  unfold queryResult at resultEq
  generalize behaviorEq :
      behaviorResult.mapError VariationSpacePreparationError.behavior = behaviorResult' at resultEq
  cases behaviorResult' with
  | error error =>
      change Except.error error = Except.ok query at resultEq
      contradiction
  | ok behavior =>
      change (materializeQuery <$> Except.mapError VariationSpacePreparationError.query
        (checkQuery queryContext _)) = Except.ok query at resultEq
      generalize checkedEq :
          (checkQuery queryContext _).mapError VariationSpacePreparationError.query =
            checkedResult at resultEq
      cases checkedResult with
      | error error =>
          change Except.error error = Except.ok query at resultEq
          contradiction
      | ok checked =>
          change Except.ok (materializeQuery checked) = Except.ok query at resultEq
          injection resultEq with resultEq
          subst query
          rfl

private structure PreparedExploration where
  space : CheckedExperimentSpace LawStatement
  kernel : IncrementalPlannerKernel space.baseQuery.target

private def prepare : Except VariationSpacePreparationError PreparedExploration :=
  match queryEq : queryResult with
  | .error error => .error error
  | .ok query =>
      match checkedEq : checkExperimentSpace (.ofQuery query) declaration with
      | .error error => .error (.space error)
      | .ok checked =>
          let queryTargetEq := queryResult_target query queryEq
          let checkedTargetEq : checked.baseQuery.target = target :=
            (congrArg (fun candidate => candidate.target) <|
              checkExperimentSpace_baseQuery checkedEq).trans queryTargetEq
          .ok {
            space := checked
            kernel := Eq.mpr (congrArg IncrementalPlannerKernel checkedTargetEq) incrementalKernel
          }

/-- Typed failure from preparing or selecting the checked Nexus exploration Space. -/
inductive NexusExplorationError where
  | preparation (error : VariationSpacePreparationError)
  | exploration (error : ExplorationError)
  deriving Repr

private def request
    (prepared : PreparedExploration)
    (policy : ExplorationPolicy)
    (limit : Nat)
    (pinned : List ExperimentSpec := []) : ExplorationRequest LawStatement := {
  space := prepared.space
  policy
  limit := { value := limit, unit := .experimentSpecs }
  pinned
}

/-- Select the checked Nexus candidates through one retained Exploration policy. -/
def run
    (policy : ExplorationPolicy)
    (limit : Nat)
    (pinned : List ExperimentSpec := []) : Except NexusExplorationError ExplorationResult := do
  let prepared ← prepare.mapError NexusExplorationError.preparation
  (explore (request prepared policy limit pinned) prepared.kernel).mapError
    NexusExplorationError.exploration

/-- Open a process-local one-candidate session over one fixed Nexus selection. -/
def startSession
    (policy : ExplorationPolicy)
    (limit : Nat)
    (pinned : List ExperimentSpec := []) : Except NexusExplorationError ExplorationSession := do
  let prepared ← prepare.mapError NexusExplorationError.preparation
  (beginSession (request prepared policy limit pinned) prepared.kernel).mapError
    NexusExplorationError.exploration

end Temporal.Feature.Nexus.Experimental.Exploration
