import Umpire.Exploration.Guided

/-!
Pure orchestration for one bounded Exploration request. `explore` checks every input, compiles the
complete finite candidate universe, and only then constructs the pinned and exploratory partitions.
-/

namespace Umpire

/-- The closed reason an eligible candidate was omitted from the exploratory partition. -/
inductive ExplorationOmissionReason where
  | pinnedPrecedence
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable serialized name of one Exploration omission reason. -/
def ExplorationOmissionReason.name : ExplorationOmissionReason → String
  | .pinnedPrecedence => "pinned-precedence"

/-- One canonical exploratory identity omitted because its pinned Regression takes precedence. -/
structure ExplorationOmission where
  identity : ArtifactChecksum
  reason : ExplorationOmissionReason
  deriving BEq, DecidableEq, Repr

/-- The exact pinned-first partitions and truthful outcomes from one bounded Exploration. -/
structure ExplorationResult where
  private mk ::
  pinned : List PinnedExperimentSpec
  exploratory : List ExplorationCandidate
  omissions : List ExplorationOmission
  coordinateOutcome : Option GuidedSelectionOutcome
  completion : ExhaustiveSelectionOutcome
  deriving BEq, DecidableEq, Repr

/-- Project selected semantic identities in their pinned-then-exploratory execution order. -/
def ExplorationResult.selectedIdentities (result : ExplorationResult) : List ArtifactChecksum :=
  result.pinned.map (fun pinned => pinned.experimentSpec.artifactChecksum) ++
    result.exploratory.map ExplorationCandidate.identity

private def pinnedPrecedenceOmissions
    (request : CheckedExplorationRequest LawStatement)
    (candidateUniverse : CandidateUniverse) : List ExplorationOmission :=
  candidateUniverse.candidates.filterMap fun candidate =>
    if ExplorationSelection.Internal.isPinned request candidate then
      some { identity := candidate.identity, reason := .pinnedPrecedence }
    else
      none

private def completionOf
    (request : CheckedExplorationRequest LawStatement)
    (candidateUniverse : CandidateUniverse) : ExhaustiveSelectionOutcome :=
  if (ExplorationSelection.Internal.eligibleCandidates request candidateUniverse).length ≤
      request.limit.value then
    .exhausted
  else
    .limitReached

private def pinnedSelectsCoordinate
    (request : CheckedExplorationRequest LawStatement)
    (coordinate : ModelCoordinate) : Bool :=
  request.pinned.any fun pinned =>
    (CandidateCoverage.ofExperimentSpec? pinned.experimentSpec).any fun coverage =>
      coverage.modelCoordinates.contains coordinate

private def exploreChecked
    (request : CheckedExplorationRequest LawStatement)
    (candidateUniverse : CandidateUniverse) :
    ExplorationResult :=
  let omissions := pinnedPrecedenceOmissions request candidateUniverse
  match request.policy with
  | .exhaustive =>
      let selection := ExhaustiveSelection.Internal.select request candidateUniverse
      {
        pinned := request.pinned
        exploratory := selection.candidates
        omissions
        coordinateOutcome := none
        completion := selection.outcome
      }
  | .uncoveredCoordinate coordinate =>
      let selection := GuidedSelection.Internal.select request candidateUniverse coordinate
      {
        pinned := request.pinned
        exploratory := selection.candidates
        omissions
        coordinateOutcome := some <| if pinnedSelectsCoordinate request coordinate then
          .coordinateSelected
        else
          selection.outcome
        completion := completionOf request candidateUniverse
      }

/--
Check and compile one request atomically, then apply its retained policy without runtime, session,
Nexus, or promotion behavior.
-/
def explore
    (request : ExplorationRequest LawStatement)
    (kernel : IncrementalPlannerKernel request.space.baseQuery.target) :
    Except ExplorationError ExplorationResult :=
  match checkedEq : checkExplorationRequest request with
  | .error error => .error error
  | .ok checked =>
      let checkedKernel : IncrementalPlannerKernel checked.space.baseQuery.target :=
        Eq.mpr (congrArg (fun space => IncrementalPlannerKernel space.baseQuery.target)
          (checkExplorationRequest_space checkedEq)) kernel
      match buildCandidateUniverse checked checkedKernel with
      | .error error => .error error
      | .ok candidateUniverse =>
          .ok (exploreChecked checked candidateUniverse)

end Umpire
