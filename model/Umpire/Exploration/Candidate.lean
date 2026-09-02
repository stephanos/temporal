import Umpire.Exploration.Coverage
import Umpire.Exploration.Language
import Umpire.Space.Compiler

/-! Atomic compilation of one checked finite Space into canonical Exploration candidates. -/

namespace Umpire

/-- One canonical ExperimentSpec and the pure model coverage already present in its selected trace. -/
structure ExplorationCandidate where
  private mk ::
  identity : ArtifactChecksum
  experimentSpec : ExperimentSpec
  canonicalBytes : String
  coverage : CandidateCoverage
  deriving BEq, DecidableEq, Repr

/-- One identity-ordered finite candidate set compiled from exactly one checked Experiment Space. -/
structure CandidateUniverse where
  private mk ::
  spaceDefinitionId : DefinitionId
  candidates : List ExplorationCandidate
  deriving BEq, DecidableEq, Repr

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

private def candidateError
    (request : CheckedExplorationRequest LawStatement)
    (kind : ExplorationErrorKind)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : ExplorationError := {
  kind
  definitionId := request.space.id
  sourcePath := if request.space.source.path == "" then "<unknown>" else request.space.source.path
  offendingValue
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

private def compilationError (error : SpaceCompilationError) : ExplorationError := {
  kind := .candidateCompilationFailed
  definitionId := error.pointId
  sourcePath := error.sourcePath
  offendingValue := canonicalSpaceCompilationErrorJson error
  relatedDefinitionIds := canonicalIds error.relatedDefinitionIds
}

private def candidateLe (left right : ExplorationCandidate) : Bool :=
  decide (left.identity.render ≤ right.identity.render)

private def candidateOfExperimentSpec
    (request : CheckedExplorationRequest LawStatement)
    (spec : ExperimentSpec) : Except ExplorationError ExplorationCandidate := do
  let coverage ← match CandidateCoverage.ofExperimentSpec? spec with
    | some coverage => pure coverage
    | none => throw (candidateError request .invalidCandidateArtifact
        spec.artifactChecksum.render [spec.plan.queryDefinitionId])
  pure {
    identity := spec.expectedArtifactChecksum
    experimentSpec := spec
    canonicalBytes := canonicalExperimentSpecBytes spec
    coverage
  }

private def firstDuplicateCandidate : List ExplorationCandidate → Option ExplorationCandidate
  | first :: second :: rest =>
      if first.identity == second.identity then
        some second
      else
        firstDuplicateCandidate (second :: rest)
  | _ => none

namespace CandidateUniverse.Internal

/-- Check the closed v1 cardinality bound before constructing any candidate universe value. -/
def checkCandidateCount
    (request : CheckedExplorationRequest LawStatement)
    (count : Nat) : Except ExplorationError Unit := do
  if count == 0 then
    throw (candidateError request .emptySpace "0")
  if count > SpaceLimits.v1.maximumPoints then
    throw (candidateError request .spacePointLimitExceeded (toString count))

/--
Validate, recompute, and identity-order one complete compiler result without exposing a partial
candidate prefix.
-/
def fromCompiledSpecs
    (request : CheckedExplorationRequest LawStatement)
    (specs : List ExperimentSpec) : Except ExplorationError CandidateUniverse := do
  checkCandidateCount request specs.length
  let candidates ← specs.mapM (candidateOfExperimentSpec request)
  let orderedCandidates := candidates.mergeSort candidateLe
  match firstDuplicateCandidate orderedCandidates with
  | some duplicate =>
      throw (candidateError request .duplicateCandidateIdentity duplicate.identity.render
        [duplicate.experimentSpec.plan.queryDefinitionId])
  | none => pure ()
  if orderedCandidates.length != request.space.pointCount then
    throw (candidateError request .candidateCountMismatch
      (toString request.space.pointCount ++ ":" ++ toString orderedCandidates.length))
  pure { spaceDefinitionId := request.space.id, candidates := orderedCandidates }

/-- Convert one all-or-nothing Space compiler result into one all-or-nothing candidate universe. -/
def fromCompilationResult
    (request : CheckedExplorationRequest LawStatement)
    (result : Except SpaceCompilationError (List ExperimentSpec)) :
    Except ExplorationError CandidateUniverse :=
  match result with
  | .error error => .error (compilationError error)
  | .ok specs => fromCompiledSpecs request specs

end CandidateUniverse.Internal

/-- Compile one checked Space through the caller's exact kernel into its canonical finite universe. -/
def buildCandidateUniverse
    (request : CheckedExplorationRequest LawStatement)
    (kernel : IncrementalPlannerKernel request.space.baseQuery.target) :
    Except ExplorationError CandidateUniverse :=
  CandidateUniverse.Internal.fromCompilationResult request
    (compileBatch request.space kernel)

end Umpire
