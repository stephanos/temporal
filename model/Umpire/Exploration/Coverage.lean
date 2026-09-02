import Umpire.Artifact.Set

/-! Canonical Model Trace coordinates retained by one compiled Exploration candidate. -/

namespace Umpire

/--
Pure model coverage present in one selected trace. Requested faults remain explicitly labeled as
intent and do not claim that Execution realized them.
-/
structure CandidateCoverage where
  modelCoordinates : List ModelCoordinate
  requestedFaultIntents : List ModelValue
  deriving BEq, DecidableEq, Repr

private def traceStepsOfPlan :
    List ModelValue → List ModelValue → List ModelValue → List ObservationCheckpoint → Nat →
      Option (List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
  | [], [], [], [], _ => some []
  | action :: actions, outcome :: outcomes, state :: states, checkpoint :: checkpoints,
      transition => do
      if checkpoint.transition != transition then
        none
      let rest ← traceStepsOfPlan actions outcomes states checkpoints (transition + 1)
      pure ({
        selectedAction := action
        modelOutcome := outcome
        resultingState := state
        observations := checkpoint.observations
      } :: rest)
  | _, _, _, _, _ => none

private def modelTraceOfExperimentSpec?
    (spec : ExperimentSpec) : Option (ModelTrace ModelValue ModelValue ModelValue ModelValue) := do
  let steps ← traceStepsOfPlan spec.plan.requestedActions spec.plan.modelOutcomes
    spec.plan.resultingStates spec.plan.checkpoints 1
  pure { initialState := spec.plan.initialState, steps }

/-- Extract coverage only when the canonical Artifact encodes one complete selected Model Trace. -/
def CandidateCoverage.ofExperimentSpec? (spec : ExperimentSpec) : Option CandidateCoverage := do
  if !spec.isValidTransport then
    none
  let trace ← modelTraceOfExperimentSpec? spec
  pure {
    modelCoordinates := trace.coordinates
    requestedFaultIntents := spec.plan.requestedFaults
  }

end Umpire
