import Umpire.Exploration.Guided

/-!
Pure orchestration for one bounded Exploration request. `explore` checks every input, compiles the
complete finite candidate universe, and only then constructs the pinned and exploratory partitions.
-/

namespace Umpire

/-- The closed reason an eligible candidate was omitted from the exploratory partition. -/
inductive DroppedCandidateReason where
  | pinnedPrecedence
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable serialized name of one Exploration omission reason. -/
def DroppedCandidateReason.name : DroppedCandidateReason → String
  | .pinnedPrecedence => "pinned-precedence"

/-- One canonical exploratory identity omitted because its pinned Regression takes precedence. -/
structure DroppedCandidate where
  identity : ArtifactChecksum
  reason : DroppedCandidateReason
  deriving BEq, DecidableEq, Repr

/-- The exact pinned-first partitions and truthful outcomes from one bounded Exploration. -/
structure ExplorationResult where
  private mk ::
  pinned : List PinnedPlan
  exploratory : List ExplorationCandidate
  omissions : List DroppedCandidate
  coordinateOutcome : Option GuidedSelectionOutcome
  completion : ExhaustiveSelectionOutcome
  deriving BEq, DecidableEq, Repr

/-- Project selected semantic identities in their pinned-then-exploratory execution order. -/
def ExplorationResult.selectedIdentities (result : ExplorationResult) : List ArtifactChecksum :=
  result.pinned.map (fun pinned => pinned.plan.artifactChecksum) ++
    result.exploratory.map ExplorationCandidate.identity

private def pinnedPrecedenceOmissions
    (request : CheckedExplorationRequest LawStatement)
    (candidateSet : CandidateSet) : List DroppedCandidate :=
  candidateSet.candidates.filterMap fun candidate =>
    if ExplorationSelection.Internal.isPinned request candidate then
      some { identity := candidate.identity, reason := .pinnedPrecedence }
    else
      none

private def completionOf
    (request : CheckedExplorationRequest LawStatement)
    (candidateSet : CandidateSet) : ExhaustiveSelectionOutcome :=
  if (ExplorationSelection.Internal.eligibleCandidates request candidateSet).length ≤
      request.limit.value then
    .exhausted
  else
    .limitReached

private def pinnedSelectsCoordinate
    (request : CheckedExplorationRequest LawStatement)
    (coordinate : ModelCoordinate) : Bool :=
  request.pinned.any fun pinned =>
    (CandidateCoverage.ofExperimentSpec? pinned.plan).any fun coverage =>
      coverage.modelCoordinates.contains coordinate

private def exploreChecked
    (request : CheckedExplorationRequest LawStatement)
    (candidateSet : CandidateSet) :
    ExplorationResult :=
  let omissions := pinnedPrecedenceOmissions request candidateSet
  match request.policy with
  | .exhaustive =>
      let selection := ExhaustiveSelection.Internal.select request candidateSet
      {
        pinned := request.pinned
        exploratory := selection.candidates
        omissions
        coordinateOutcome := none
        completion := selection.outcome
      }
  | .uncoveredCoordinate coordinate =>
      let selection := GuidedSelection.Internal.select request candidateSet coordinate
      {
        pinned := request.pinned
        exploratory := selection.candidates
        omissions
        coordinateOutcome := some <| if pinnedSelectsCoordinate request coordinate then
          .coordinateSelected
        else
          selection.outcome
        completion := completionOf request candidateSet
      }

/--
Check and compile one request atomically, then apply its retained policy without runtime, session,
Nexus, or promotion behavior.
-/
def explore
    (request : ExplorationRequest LawStatement)
    (kernel : SearchView request.space.baseQuery.target) :
    Except ExplorationError ExplorationResult :=
  match checkedEq : checkExplorationRequest request with
  | .error error => .error error
  | .ok checked =>
      let checkedKernel : SearchView checked.space.baseQuery.target :=
        Eq.mpr (congrArg (fun space => SearchView space.baseQuery.target)
          (checkExplorationRequest_space checkedEq)) kernel
      match buildCandidateSet checked checkedKernel with
      | .error error => .error error
      | .ok candidateSet =>
          .ok (exploreChecked checked candidateSet)

end Umpire
