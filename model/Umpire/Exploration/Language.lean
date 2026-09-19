import Umpire.Exploration.Core

/-! Atomic checking and canonical diagnostics for bounded Exploration requests. -/

namespace Umpire

/-- One pinned canonical Plan checked independently of the exploration budget. -/
structure PinnedRegression where
  private mk ::
  plan : Plan
  canonicalBytes : String
  deriving BEq, DecidableEq, Repr

/--
Validated exploration inputs. Construction checks the Space bound, policy coordinate, typed Limit,
and pinned Artifact partition together.
-/
structure CheckedExplorationRequest (LawStatement : Law → Prop) where
  private mk ::
  space : CheckedVariationSpace LawStatement
  policy : ExplorationPolicy
  limit : Limit
  pinned : List PinnedRegression

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

private def requestError
    (request : ExplorationRequest LawStatement)
    (kind : ExplorationErrorKind)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : ExplorationError := {
  kind
  definitionId := request.space.id
  sourcePath := if request.space.source.path == "" then "<unknown>" else request.space.source.path
  offendingValue
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

/-- Canonical machine-readable projection of one typed Exploration failure. -/
def canonicalExplorationErrorJson (error : ExplorationError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"definitionId\":" ++ quote error.definitionId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      array (canonicalIds error.relatedDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++ "}"

private def maximumTraceSteps (space : CheckedVariationSpace LawStatement) : Nat :=
  Nat.min space.baseQuery.limits.steps.value space.baseQuery.limits.actions.value

private def maximumObservationPositions (space : CheckedVariationSpace LawStatement) : Nat :=
  space.baseQuery.target.behaviorTable.transitions.foldl
    (fun maximum transition => Nat.max maximum transition.facts.length) 0

private def coordinateKnown
    (space : CheckedVariationSpace LawStatement) : ModelCoordinate → Bool
  | .initialState => true
  | .selectedAction step | .outcome step | .state step =>
      step > 0 && step ≤ maximumTraceSteps space
  | .fact step position =>
      step > 0 && step ≤ maximumTraceSteps space &&
        position > 0 && position ≤ maximumObservationPositions space

private def pinnedLe (left right : Plan) : Bool :=
  decide (left.artifactChecksum.render ≤ right.artifactChecksum.render)

private def firstDuplicatePinned : List Plan → Option Plan
  | first :: second :: rest =>
      if first.artifactChecksum == second.artifactChecksum then
        some second
      else
        firstDuplicatePinned (second :: rest)
  | _ => none

private def pinnedMatchesContract
    (space : CheckedVariationSpace LawStatement)
    (spec : Plan) : Bool :=
  spec.plan.targetDefinitionId == space.baseQuery.target.id &&
    spec.plan.targetBehaviorFingerprint == space.baseQuery.target.behaviorFingerprint &&
    spec.plan.kernelDefinitionId == space.baseQuery.target.machine.metadata.id &&
    spec.plan.kernelBehaviorFingerprint == space.baseQuery.target.behaviorFingerprint

private def checkPinned
    (request : ExplorationRequest LawStatement) :
    Except ExplorationError (List PinnedRegression) := do
  let pinned := request.pinned.mergeSort pinnedLe
  match firstDuplicatePinned pinned with
  | some duplicate =>
      throw (requestError request .duplicatePinnedIdentity duplicate.artifactChecksum.render
        [duplicate.plan.queryDefinitionId])
  | none => pure ()
  let mut checked := []
  for spec in pinned do
    if !spec.isValidTransport then
      throw (requestError request .invalidPinnedArtifact spec.artifactChecksum.render
        [spec.plan.queryDefinitionId])
    if !pinnedMatchesContract request.space spec then
      throw (requestError request .incompatiblePinnedContract spec.artifactChecksum.render [
        spec.plan.targetDefinitionId,
        spec.plan.kernelDefinitionId,
        request.space.baseQuery.target.id,
        request.space.baseQuery.target.machine.metadata.id
      ])
    checked := checked ++ [{
      plan := spec
      canonicalBytes := canonicalPlanBytes spec
    }]
  pure checked

private def checkExplorationRequestInputs
    (request : ExplorationRequest LawStatement) :
    Except ExplorationError (List PinnedRegression) := do
  if request.space.pointCount == 0 then
    throw (requestError request .emptySpace "0")
  if request.space.pointCount > SpaceLimits.v1.maximumPoints then
    throw (requestError request .spacePointLimitExceeded (toString request.space.pointCount))
  if request.limit.value == 0 || request.limit.value > SpaceLimits.v1.maximumPoints then
    throw (requestError request .invalidLimitValue (toString request.limit.value))
  if request.limit.unit != .plans then
    throw (requestError request .invalidLimitUnit request.limit.unit.name)
  match request.policy with
  | .exhaustive => pure ()
  | .uncoveredCoordinate coordinate =>
      if !coordinateKnown request.space coordinate then
        throw (requestError request .unknownCoordinate coordinate.name)
  checkPinned request

/-- Check all bounded Exploration inputs before candidate compilation or selection. -/
def checkExplorationRequest
    (request : ExplorationRequest LawStatement) :
    Except ExplorationError (CheckedExplorationRequest LawStatement) := do
  let pinned ← checkExplorationRequestInputs request
  pure {
    space := request.space
    policy := request.policy
    limit := request.limit
    pinned
  }

/-- A successfully checked Exploration request retains the exact Space supplied by its caller. -/
theorem checkExplorationRequest_space
    (resultEq : checkExplorationRequest request = .ok checked) :
    checked.space = request.space := by
  unfold checkExplorationRequest at resultEq
  cases inputsEq : checkExplorationRequestInputs request with
  | error error =>
      rw [inputsEq] at resultEq
      contradiction
  | ok pinned =>
      rw [inputsEq] at resultEq
      cases resultEq
      rfl

end Umpire
