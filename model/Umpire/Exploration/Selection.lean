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

private def isPinned
    (request : CheckedExplorationRequest LawStatement)
    (candidate : ExplorationCandidate) : Bool :=
  request.pinned.any fun pinned =>
    pinned.experimentSpec.artifactChecksum == candidate.identity

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
    let candidates := candidateUniverse.candidates.filter fun candidate =>
      !isPinned request candidate
    some {
      candidates := candidates.take request.limit.value
      outcome := if candidates.length ≤ request.limit.value then
        .exhausted
      else
        .limitReached
    }

end Umpire
