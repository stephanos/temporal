import Umpire.Exploration.Engine

/-! Pure process-local sequencing for one checked bounded Exploration result. -/

namespace Umpire

/-- One selected Plan in the process-local session's fixed execution order. -/
structure CandidateCursorCandidate where
  private mk ::
  plan : Plan
  deriving BEq, DecidableEq, Repr

/-- The semantic identity of one process-local session candidate. -/
def CandidateCursorCandidate.identity
    (candidate : CandidateCursorCandidate) : ArtifactChecksum :=
  candidate.plan.artifactChecksum

/-- A fixed selected order with at most one candidate awaiting exact admission. -/
structure CandidateCursor where
  private mk ::
  remaining : List CandidateCursorCandidate
  outstanding : Option CandidateCursorCandidate
  deriving BEq, DecidableEq, Repr

private def sessionCandidateOfPinned
    (pinned : PinnedPlan) : CandidateCursorCandidate := {
  plan := pinned.plan
}

private def sessionCandidateOfExploratory
    (candidate : ExplorationCandidate) : CandidateCursorCandidate := {
  plan := candidate.plan
}

/-- Check and select one Exploration request before opening its process-local candidate session. -/
def beginSession
    (request : ExplorationRequest LawStatement)
    (kernel : SearchView request.space.baseQuery.target) :
    Except ExplorationError CandidateCursor := do
  let result ← explore request kernel
  pure {
    remaining := result.pinned.map sessionCandidateOfPinned ++
      result.exploratory.map sessionCandidateOfExploratory
    outstanding := none
  }

/-- Return the next fixed candidate and a session that must observe it before advancing. -/
def CandidateCursor.next
    (session : CandidateCursor) : Option (CandidateCursorCandidate × CandidateCursor) :=
  match session.outstanding, session.remaining with
  | none, candidate :: remaining =>
      some (candidate, { remaining, outstanding := some candidate })
  | _, _ => none

/-- Admit exactly the immutable binding for the outstanding candidate. -/
def CandidateCursor.observe
    (session : CandidateCursor)
    (bindings : List ArtifactBinding) : Option CandidateCursor :=
  match session.outstanding, bindings with
  | some candidate, [binding] =>
      if binding == candidate.plan.artifactBinding then
        some { session with outstanding := none }
      else
        none
  | _, _ => none

end Umpire
