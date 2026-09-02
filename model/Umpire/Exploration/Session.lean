import Umpire.Exploration.Engine

/-! Pure process-local sequencing for one checked bounded Exploration result. -/

namespace Umpire

/-- One selected ExperimentSpec in the process-local session's fixed execution order. -/
structure ExplorationSessionCandidate where
  private mk ::
  experimentSpec : ExperimentSpec
  deriving BEq, DecidableEq, Repr

/-- The semantic identity of one process-local session candidate. -/
def ExplorationSessionCandidate.identity
    (candidate : ExplorationSessionCandidate) : ArtifactChecksum :=
  candidate.experimentSpec.artifactChecksum

/-- A fixed selected order with at most one candidate awaiting exact admission. -/
structure ExplorationSession where
  private mk ::
  remaining : List ExplorationSessionCandidate
  outstanding : Option ExplorationSessionCandidate
  deriving BEq, DecidableEq, Repr

private def sessionCandidateOfPinned
    (pinned : PinnedExperimentSpec) : ExplorationSessionCandidate := {
  experimentSpec := pinned.experimentSpec
}

private def sessionCandidateOfExploratory
    (candidate : ExplorationCandidate) : ExplorationSessionCandidate := {
  experimentSpec := candidate.experimentSpec
}

/-- Check and select one Exploration request before opening its process-local candidate session. -/
def beginSession
    (request : ExplorationRequest LawStatement)
    (kernel : IncrementalPlannerKernel request.space.baseQuery.target) :
    Except ExplorationError ExplorationSession := do
  let result ← explore request kernel
  pure {
    remaining := result.pinned.map sessionCandidateOfPinned ++
      result.exploratory.map sessionCandidateOfExploratory
    outstanding := none
  }

/-- Return the next fixed candidate and a session that must observe it before advancing. -/
def ExplorationSession.next
    (session : ExplorationSession) : Option (ExplorationSessionCandidate × ExplorationSession) :=
  match session.outstanding, session.remaining with
  | none, candidate :: remaining =>
      some (candidate, { remaining, outstanding := some candidate })
  | _, _ => none

/-- Admit exactly the immutable binding for the outstanding candidate. -/
def ExplorationSession.observe
    (session : ExplorationSession)
    (bindings : List ArtifactBinding) : Option ExplorationSession :=
  match session.outstanding, bindings with
  | some candidate, [binding] =>
      if binding == candidate.experimentSpec.artifactBinding then
        some { session with outstanding := none }
      else
        none
  | _, _ => none

end Umpire
