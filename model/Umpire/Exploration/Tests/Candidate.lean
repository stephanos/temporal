import Umpire.Exploration.Candidate
import Umpire.Space.Tests.Fixtures

/-! Atomic compilation and canonical coverage for one finite Exploration candidate universe. -/

namespace Umpire.ExplorationTests

open Umpire

/-! Only the exact-kernel public builder may construct a CandidateUniverse. -/
/--
error: Unknown constant `Umpire.CandidateUniverse.Internal.fromCompiledSpecs`
-/
#guard_msgs (error, substring := true) in
#check Umpire.CandidateUniverse.Internal.fromCompiledSpecs

/--
error: Unknown constant `Umpire.CandidateUniverse.Internal.fromCompilationResult`
-/
#guard_msgs (error, substring := true) in
#check Umpire.CandidateUniverse.Internal.fromCompilationResult

private theorem except_eq_ok_get
    (result : Except ε α)
    (isSome : result.toOption.isSome = true) :
    result = .ok (result.toOption.get isSome) := by
  cases result with
  | error _ => cases isSome
  | ok _ => rfl

private theorem checkedSpaceResultEq :
    SpaceTests.checkedResult = .ok SpaceTests.checked :=
  except_eq_ok_get SpaceTests.checkedResult (by native_decide)

private theorem checkedSpaceTargetEq :
    SpaceTests.checked.baseQuery.target = Umpire.Examples.Switch.target := by
  exact congrArg (fun query => query.target)
    (checkExperimentSpace_baseQuery checkedSpaceResultEq)

private def kernel : IncrementalPlannerKernel SpaceTests.checked.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedSpaceTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def authoredRequest : ExplorationRequest Umpire.Examples.Switch.LawStatement := {
  space := SpaceTests.checked
  policy := .exhaustive
  limit := { value := 4, unit := .experimentSpecs }
}

private def checkedRequestResult := checkExplorationRequest authoredRequest

private def checkedRequest : CheckedExplorationRequest Umpire.Examples.Switch.LawStatement :=
  checkedRequestResult.toOption.get (by native_decide)

private theorem checkedRequestResultEq : checkedRequestResult = .ok checkedRequest :=
  except_eq_ok_get checkedRequestResult (by native_decide)

private theorem checkedRequestTargetEq :
    checkedRequest.space.baseQuery.target = Umpire.Examples.Switch.target := by
  calc
    checkedRequest.space.baseQuery.target = SpaceTests.checked.baseQuery.target :=
      congrArg (fun space => space.baseQuery.target)
        (checkExplorationRequest_space checkedRequestResultEq)
    _ = Umpire.Examples.Switch.target := checkedSpaceTargetEq

private def candidateKernel : IncrementalPlannerKernel checkedRequest.space.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedRequestTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def compiled := compileBatch SpaceTests.checked kernel

private def universeResult := buildCandidateUniverse checkedRequest candidateKernel

private def universeErrorKindOf
    (result : Except ExplorationError α) : Option ExplorationErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

private def checksumLe (left right : ArtifactChecksum) : Bool :=
  decide (left.render ≤ right.render)

/-!
Every candidate preserves one valid canonical ExperimentSpec, uses its recomputed identity, and
extracts only the coordinates of that Artifact's complete selected Model Trace.
-/
example : universeResult.toOption.map (fun result =>
    result.candidates.length == 4 &&
      result.candidates.all fun candidate =>
        candidate.experimentSpec.isValidTransport &&
          candidate.identity == candidate.experimentSpec.expectedArtifactChecksum &&
          candidate.canonicalBytes == canonicalExperimentSpecBytes candidate.experimentSpec &&
          candidate.coverage.modelCoordinates == [
            .initialState,
            .selectedAction 1,
            .modelOutcome 1,
            .resultingState 1,
            .observation 1 1
          ]) = some true := by
  native_decide

/-! Requested faults stay in a separately named intent partition and make no runtime claim. -/
example : universeResult.toOption.map (fun result =>
    result.candidates.map (fun candidate =>
      candidate.coverage.requestedFaultIntents.map ModelValue.definitionId) |>.mergeSort
        (fun left right =>
          decide ((left.map DefinitionId.value) ≤ (right.map DefinitionId.value)))) = some [
    [],
    [],
    [SpaceTests.delayFaultId],
    [SpaceTests.delayFaultId]
  ] := by
  native_decide

/-! Candidate identity order is canonical and independent of compiled input order. -/
example :
    let forward := compiled.toOption.bind fun specs =>
      (CandidateUniverse.Internal.validateCompiledSpecs checkedRequest specs).toOption
    let reversed := compiled.toOption.bind fun specs =>
      (CandidateUniverse.Internal.validateCompiledSpecs checkedRequest specs.reverse).toOption
    forward = reversed ∧ forward.map (fun result =>
      decide (result.Pairwise fun left right =>
        checksumLe left right = true)) = some true := by
  native_decide

private def invalidArtifact : ExperimentSpec := {
  Umpire.Examples.Switch.compiledArtifact with
  artifactChecksum := experimentSpecChecksumOf "invalid"
}

private def invalidTraceArtifact : ExperimentSpec :=
  let planDraft := {
    Umpire.Examples.Switch.compiledArtifact.plan with
    artifactChecksum := drivePlanChecksumOf ""
    modelOutcomes := []
  }
  let plan := { planDraft with artifactChecksum := planDraft.expectedArtifactChecksum }
  let specDraft := {
    Umpire.Examples.Switch.compiledArtifact with
    artifactChecksum := experimentSpecChecksumOf ""
    plan
  }
  { specDraft with artifactChecksum := specDraft.expectedArtifactChecksum }

/-! Invalid, incomplete, duplicate, or count-mismatched inputs expose no partial universe. -/
example : [
    universeErrorKindOf (CandidateUniverse.Internal.validateCompiledSpecs checkedRequest
      [invalidArtifact]),
    universeErrorKindOf (CandidateUniverse.Internal.validateCompiledSpecs checkedRequest
      [invalidTraceArtifact]),
    universeErrorKindOf (CandidateUniverse.Internal.validateCompiledSpecs checkedRequest [
      Umpire.Examples.Switch.compiledArtifact,
      Umpire.Examples.Switch.compiledArtifact,
      Umpire.Examples.Switch.compiledArtifact,
      Umpire.Examples.Switch.compiledArtifact
    ]),
    universeErrorKindOf (CandidateUniverse.Internal.validateCompiledSpecs checkedRequest
      [Umpire.Examples.Switch.compiledArtifact])
  ] = [
    some .invalidCandidateArtifact,
    some .invalidCandidateArtifact,
    some .duplicateCandidateIdentity,
    some .candidateCountMismatch
  ] := by
  native_decide

/-! The closed v1 universe accepts N = 256 and rejects empty or N + 1 before construction. -/
example : [
    universeErrorKindOf (CandidateUniverse.Internal.checkCandidateCount checkedRequest
      SpaceLimits.v1.maximumPoints),
    universeErrorKindOf (CandidateUniverse.Internal.checkCandidateCount checkedRequest 0),
    universeErrorKindOf (CandidateUniverse.Internal.checkCandidateCount checkedRequest
      (SpaceLimits.v1.maximumPoints + 1))
  ] = [none, some .emptySpace, some .spacePointLimitExceeded] := by
  native_decide

end Umpire.ExplorationTests
