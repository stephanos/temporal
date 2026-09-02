import Umpire.Exploration.Selection

/-! Deterministic bounded guidance toward one checked uncovered Model Coordinate. -/

namespace Umpire

/-- Whether a guided selection covered its requested Model Coordinate. -/
inductive GuidedSelectionOutcome where
  | coordinateSelected
  | coordinateUncovered
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable serialized name of one guided-selection outcome. -/
def GuidedSelectionOutcome.name : GuidedSelectionOutcome → String
  | .coordinateSelected => "coordinate-selected"
  | .coordinateUncovered => "coordinate-uncovered"

/-- The canonical candidates selected for one checked uncovered Model Coordinate. -/
structure GuidedSelection where
  private mk ::
  coordinate : ModelCoordinate
  candidates : List ExplorationCandidate
  outcome : GuidedSelectionOutcome
  deriving BEq, DecidableEq, Repr

/-- Project the semantic identities selected by one bounded guided run. -/
def GuidedSelection.identities (selection : GuidedSelection) : List ArtifactChecksum :=
  selection.candidates.map ExplorationCandidate.identity

namespace GuidedSelection.Internal

private def candidateLe
    (coordinate : ModelCoordinate)
    (identity : α → String)
    (coordinates : α → List ModelCoordinate)
    (left right : α) : Bool :=
  match (coordinates left).contains coordinate, (coordinates right).contains coordinate with
  | true, false => true
  | false, true => false
  | _, _ => decide (identity left ≤ identity right)

/-- Apply the fixed coordinate-first, semantic-identity tie ordering to immutable candidates. -/
def prioritize
    (coordinate : ModelCoordinate)
    (identity : α → String)
    (coordinates : α → List ModelCoordinate)
    (candidates : List α) : List α :=
  candidates.mergeSort (candidateLe coordinate identity coordinates)

/-- Report only whether the selected immutable prefix contains the requested coordinate. -/
def outcome
    (coordinate : ModelCoordinate)
    (coordinates : α → List ModelCoordinate)
    (candidates : List α) : GuidedSelectionOutcome :=
  if candidates.any fun candidate => (coordinates candidate).contains coordinate then
    .coordinateSelected
  else
    .coordinateUncovered

/-- Apply coordinate guidance after the engine has established request and universe bindings. -/
def select
    (request : CheckedExplorationRequest LawStatement)
    (candidateUniverse : CandidateUniverse)
    (coordinate : ModelCoordinate) : GuidedSelection :=
  let eligible := ExplorationSelection.Internal.eligibleCandidates request candidateUniverse
  let ordered := prioritize coordinate
    (ArtifactChecksum.render ∘ ExplorationCandidate.identity)
    (fun candidate => candidate.coverage.modelCoordinates) eligible
  let candidates := ordered.take request.limit.value
  {
    coordinate
    candidates
    outcome := outcome coordinate (fun candidate => candidate.coverage.modelCoordinates) candidates
  }

end GuidedSelection.Internal

/--
Select a bounded canonical prefix after ranking exact coordinate matches first. Crossed Space or
policy inputs produce no selection. An absent match stays uncovered and makes no reachability claim.
-/
def selectUncoveredCoordinate
    (request : CheckedExplorationRequest LawStatement)
    (candidateUniverse : CandidateUniverse) : Option GuidedSelection :=
  match request.policy with
  | .exhaustive => none
  | .uncoveredCoordinate coordinate =>
      if candidateUniverse.spaceDefinitionId != request.space.id ||
          candidateUniverse.spaceBehaviorFingerprint != request.space.behaviorFingerprint then
        none
      else
        some (GuidedSelection.Internal.select request candidateUniverse coordinate)

end Umpire
