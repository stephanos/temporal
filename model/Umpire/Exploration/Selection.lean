import Umpire.Exploration.Candidate

/-! Deterministic bounded exhaustive selection over one canonical candidate universe. -/

namespace Umpire

/-- Whether bounded selection considered the complete finite universe or stopped at its Limit. -/
inductive ExhaustiveSelectionOutcome where
  | exhausted
  | limitReached
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable serialized name of one bounded exhaustive-selection outcome. -/
def ExhaustiveSelectionOutcome.name : ExhaustiveSelectionOutcome → String
  | .exhausted => "exhausted"
  | .limitReached => "limit-reached"

/-- The canonical candidate prefix selected within one checked Exploration Limit. -/
structure ExhaustiveSelection where
  private mk ::
  candidates : List ExplorationCandidate
  outcome : ExhaustiveSelectionOutcome
  deriving BEq, DecidableEq, Repr

/-- Project the semantic identities selected by one bounded exhaustive run. -/
def ExhaustiveSelection.identities (selection : ExhaustiveSelection) : List ArtifactChecksum :=
  selection.candidates.map ExplorationCandidate.identity

namespace ExplorationSelection.Internal

/-- Whether one canonical candidate is already retained by the checked pinned partition. -/
def isPinned
    (request : CheckedExplorationRequest LawStatement)
    (candidate : ExplorationCandidate) : Bool :=
  request.pinned.any fun pinned =>
    pinned.experimentSpec.artifactChecksum == candidate.identity

/-- The canonical candidate partition still eligible for the Exploration Limit. -/
def eligibleCandidates
    (request : CheckedExplorationRequest LawStatement)
    (candidateUniverse : CandidateUniverse) : List ExplorationCandidate :=
  candidateUniverse.candidates.filter fun candidate => !isPinned request candidate

end ExplorationSelection.Internal

namespace ExhaustiveSelection.Internal

/-- Apply the exhaustive policy after the engine has established request and universe bindings. -/
def select
    (request : CheckedExplorationRequest LawStatement)
    (candidateUniverse : CandidateUniverse) : ExhaustiveSelection :=
  let candidates := ExplorationSelection.Internal.eligibleCandidates request candidateUniverse
  {
    candidates := candidates.take request.limit.value
    outcome := if candidates.length ≤ request.limit.value then
      .exhausted
    else
      .limitReached
  }

end ExhaustiveSelection.Internal

/--
Select the identity-ordered prefix for one checked exhaustive request. Crossed Space or policy
inputs produce no selection. Only reaching the finite universe end reports exhaustion.
-/
def selectExhaustive
    (request : CheckedExplorationRequest LawStatement)
    (candidateUniverse : CandidateUniverse) : Option ExhaustiveSelection :=
  if request.policy != .exhaustive ||
      candidateUniverse.spaceDefinitionId != request.space.id ||
      candidateUniverse.spaceBehaviorFingerprint != request.space.behaviorFingerprint then
    none
  else
    some (ExhaustiveSelection.Internal.select request candidateUniverse)

end Umpire
