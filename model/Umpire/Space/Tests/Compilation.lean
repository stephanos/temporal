import Umpire.Space.Compiler
import Umpire.Space.Tests.Fixtures
import Umpire.Planning.Tests.Fixtures

/-! Exact assignment lowering and atomic checked-Space compilation. -/

namespace Umpire.SpaceTests

open Umpire

private def validAssignment : List ModelValue := [
  { definitionId := stateAxisId, value := stateOffId.value },
  { definitionId := faultAxisId, value := faultDelayId.value }
]

private def canonicalValidAssignment : List ModelValue := [
  { definitionId := faultAxisId, value := faultDelayId.value },
  { definitionId := stateAxisId, value := stateOffId.value }
]

private def loweredResult := lowerSpacePoint checked validAssignment

private theorem loweredResult_isSome : loweredResult.toOption.isSome = true := by
  native_decide

private def lowered := loweredResult.toOption.get loweredResult_isSome

private def authoredGap : KnownGap := {
  kind := .claim
  code := id "space.test.known-gap.authored"
  subject := some checked.baseQuery.id
  detail := some "The derived Space Query retains this authored limitation."
}

private def authoredGaps : KnownGapSet :=
  (KnownGapSet.checkCanonical [authoredGap]).toOption.get (by native_decide)

private def baseQueryWithAuthoredGaps := {
  checked.baseQuery with authoredKnownGaps := authoredGaps
}

private def checkedWithAuthoredGapsResult :=
  checkExperimentSpace (.ofQuery baseQueryWithAuthoredGaps) declaration

private def checkedWithAuthoredGaps :=
  checkedWithAuthoredGapsResult.toOption.get (by native_decide)

private def conflictingGap : KnownGap := {
  plannerExecutionEvidenceKnownGap with detail := some "conflicting authored detail"
}

private def conflictingGaps : KnownGapSet :=
  (KnownGapSet.checkCanonical [conflictingGap]).toOption.get (by native_decide)

private def checkedWithConflictingGapsResult :=
  checkExperimentSpace (.ofQuery {
    checked.baseQuery with authoredKnownGaps := conflictingGaps
  }) declaration

private def checkedWithConflictingGaps :=
  checkedWithConflictingGapsResult.toOption.get (by native_decide)

private def loweredWithAuthoredGapsResult :=
  lowerSpacePoint checkedWithAuthoredGaps validAssignment

private theorem except_eq_ok_get
    (result : Except ε α)
    (isSome : result.toOption.isSome = true) :
    result = .ok (result.toOption.get isSome) := by
  cases result with
  | error error => cases isSome
  | ok value => rfl

private theorem checkedResultEq : checkedResult = .ok checked :=
  except_eq_ok_get checkedResult (by native_decide)

private theorem checkedWithConflictingGapsResultEq :
    checkedWithConflictingGapsResult = .ok checkedWithConflictingGaps :=
  except_eq_ok_get checkedWithConflictingGapsResult (by native_decide)

private theorem checkedTargetEq :
    checked.baseQuery.target = Umpire.Examples.Switch.target := by
  exact congrArg (fun query => query.target)
    (checkExperimentSpace_baseQuery checkedResultEq)

private def baseKernel : IncrementalPlannerKernel checked.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def conflictingBaseKernel :
    IncrementalPlannerKernel checkedWithConflictingGaps.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel (congrArg (fun query => query.target)
    (checkExperimentSpace_baseQuery checkedWithConflictingGapsResultEq))) baseKernel

private def transportedKernel : IncrementalPlannerKernel lowered.query.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel lowered.targetEq)
    baseKernel

private def transportedRun :=
  planWithArtifactIntent lowered.query transportedKernel lowered.intent

private def batchResult :=
  compileBatch checked baseKernel

private def conflictingBatchResult :=
  compileBatch checkedWithConflictingGaps conflictingBaseKernel

private def compileErrorOf
    (result : Except SpaceCompilationError α) : Option SpaceCompilationError :=
  match result with
  | .ok _ => none
  | .error error => some error

private def compileErrorKindOf
    (result : Except SpaceCompilationError α) : Option SpaceCompilationErrorKind :=
  (compileErrorOf result).map SpaceCompilationError.kind

/-! Lowering derives fresh checked identities while retaining the exact base target. -/
example : loweredResult.toOption.map (fun point =>
    point.query.id != checked.baseQuery.id &&
      point.query.behavior.id != checked.baseQuery.behavior.id &&
      point.query.target.id == checked.baseQuery.target.id &&
      point.intent.selectedChoices == canonicalValidAssignment) = some true := by
  native_decide

/-! Space-derived Queries retain the complete nonempty authored set from their base Query. -/
example : loweredWithAuthoredGapsResult.toOption.map (fun point =>
    point.query.authoredKnownGaps.toList) = some authoredGaps.toList := by
  native_decide

/-! Space compilation retains the complete typed Known Gap failure from ordinary planning. -/
example : (compileErrorOf conflictingBatchResult).map (fun error =>
    (error.kind, error.knownGapError)) = some (.knownGapCheckFailed, some {
      kind := .conflictingDetail
      code := plannerExecutionEvidenceKnownGap.code
      subject := plannerExecutionEvidenceKnownGap.subject
    }) := by
  native_decide

/-! Derived Behavior and Query identities reject every collision with visible definitions. -/
example : [lowered.query.id, lowered.query.behavior.id].map (fun collision =>
    let result := SpaceCompiler.Internal.rejectDerivedIdentityCollisions checked lowered.id
      [collision]
    (result.toOption,
      (compileErrorOf result).map fun error => (error.kind, error.pointId))) = [
    (none, some (.derivedIdentityCollision, lowered.id)),
    (none, some (.derivedIdentityCollision, lowered.id))
  ] := by
  native_decide

/-! The target-equality proof transports the one caller-owned kernel into ordinary planning. -/
example : (transportedRun.toOption.bind PlannerRun.artifact).isSome = true := by
  native_decide

/-! The complete two-by-two product compiles atomically in canonical assignment order. -/
example : batchResult.toOption.map (fun specs =>
    (specs.length, specs.map fun spec => (
      spec.plan.selectedChoices,
      spec.plan.selectedVariants,
      spec.plan.requestedFaults.map ModelValue.definitionId))) = some (4, [
    ([
      { definitionId := faultAxisId, value := faultBaselineId.value },
      { definitionId := stateAxisId, value := stateBaselineId.value }
    ], [], []),
    ([
      { definitionId := faultAxisId, value := faultBaselineId.value },
      { definitionId := stateAxisId, value := stateOffId.value }
    ], [Umpire.Examples.Switch.offState], []),
    ([
      { definitionId := faultAxisId, value := faultDelayId.value },
      { definitionId := stateAxisId, value := stateBaselineId.value }
    ], [], [delayFaultId]),
    ([
      { definitionId := faultAxisId, value := faultDelayId.value },
      { definitionId := stateAxisId, value := stateOffId.value }
    ], [Umpire.Examples.Switch.offState], [delayFaultId])
  ]) := by
  native_decide

/-! Space intent never authors or patches target-owned trace semantics. -/
example : batchResult.toOption.map (fun specs => specs.all fun spec =>
    spec.plan.requestedActions == Umpire.Examples.Switch.compiledArtifact.plan.requestedActions &&
      spec.plan.modelOutcomes == Umpire.Examples.Switch.compiledArtifact.plan.modelOutcomes &&
      spec.plan.resultingStates == Umpire.Examples.Switch.compiledArtifact.plan.resultingStates &&
      spec.plan.checkpoints == Umpire.Examples.Switch.compiledArtifact.plan.checkpoints &&
      spec.plan.selectionReason == Umpire.Examples.Switch.compiledArtifact.plan.selectionReason) =
    some true := by
  native_decide

/-! Exact assignments reject the first canonical missing, extra, duplicate, or unknown choice. -/
example : [
    [{ definitionId := stateAxisId, value := stateOffId.value }],
    validAssignment ++ [{ definitionId := id "space.test.axis.extra", value := stateOffId.value }],
    validAssignment ++ [{ definitionId := stateAxisId, value := stateBaselineId.value }],
    [
      { definitionId := stateAxisId, value := "space.test.choice.unknown" },
      { definitionId := faultAxisId, value := faultDelayId.value }
    ]
  ].map (fun assignment => compileErrorKindOf (lowerSpacePoint checked assignment)) = [
    some .missingChoice,
    some .extraChoice,
    some .duplicateChoice,
    some .unknownChoice
  ] := by
  native_decide

/-! Malformed authoring order cannot change the point identity carried by an assignment error. -/
example :
    let extra := { definitionId := id "space.test.axis.extra", value := stateOffId.value }
    let first := compileErrorOf (lowerSpacePoint checked (extra :: validAssignment))
    let second := compileErrorOf (lowerSpacePoint checked (validAssignment ++ [extra]))
    first.map SpaceCompilationError.pointId = second.map SpaceCompilationError.pointId := by
  native_decide

private def verifiedPlannerRun : Except KnownGapError PlannerRun :=
  Umpire.PlanningTests.run 2 (.verify Umpire.PlanningTests.property) .exhaustive

private def verifiedPointRejection :=
  verifiedPlannerRun.toOption.map fun run =>
    SpaceCompiler.Internal.appendPlannerRun checked lowered.id
      [Umpire.Examples.Switch.compiledArtifact] run

/-!
A verified point with no Artifact rejects the canonical point and never returns its existing prefix.
-/
example :
    verifiedPointRejection.map (fun rejection =>
      (rejection.toOption,
        (compileErrorOf rejection).map fun error => (error.kind, error.pointId))) =
    some (none, some (.verifiedWithoutArtifact, lowered.id)) := by
  native_decide

private def exhaustedPlannerRun : Except KnownGapError PlannerRun :=
  Umpire.PlanningTests.run 64 (.counterexample Umpire.PlanningTests.property)
    .shortest 1 17 false

private def absentPlannerRun : Except KnownGapError PlannerRun :=
  Umpire.PlanningTests.run 0 (.counterexample Umpire.PlanningTests.property) .exhaustive

private def staticallyUnsatisfiableBehavior : CheckedScenario := {
  Umpire.PlanningTests.behavior with
  spaceStatus := .unsatisfiable
  behaviorFingerprint := behaviorFingerprintOf "space-compiler-test/unsatisfiable"
}

private def unsatisfiablePlannerRun : Except KnownGapError PlannerRun :=
  Umpire.PlanningTests.run 0 (.verify Umpire.PlanningTests.property) .exhaustive
    10 17 true staticallyUnsatisfiableBehavior

private def rejectedPlannerKind (result : Except KnownGapError PlannerRun) :
    Option (SpaceCompilationErrorKind × DefinitionId) :=
  match result with
  | .error _ => none
  | .ok run =>
      let rejected := SpaceCompiler.Internal.appendPlannerRun checked lowered.id
        [Umpire.Examples.Switch.compiledArtifact] run
      if rejected.toOption.isSome then
        none
      else
        (compileErrorOf rejected).map fun error => (error.kind, error.pointId)

/-! Every non-artifact planner termination rejects the canonical point with no partial list. -/
example : [
    rejectedPlannerKind exhaustedPlannerRun,
    rejectedPlannerKind absentPlannerRun,
    rejectedPlannerKind unsatisfiablePlannerRun
  ] = [
    some (.budgetExhausted, lowered.id),
    some (.noArtifact, lowered.id),
    some (.unsatisfiable, lowered.id)
  ] := by
  native_decide

private def foundPlannerRun : Except KnownGapError PlannerRun :=
  Umpire.PlanningTests.run 2 (.witness Umpire.PlanningTests.property) .shortest

private def duplicateSpecRejection :=
  foundPlannerRun.toOption.bind fun run =>
    run.artifact.map fun spec =>
      SpaceCompiler.Internal.appendPlannerRun checked lowered.id [spec] run

/-! Duplicate final ExperimentSpec identity rejects the point without returning the prior spec. -/
example :
    duplicateSpecRejection.map (fun rejection =>
      (rejection.toOption, compileErrorKindOf rejection)) =
    some (none, some .duplicateExperimentSpecIdentity) := by
  native_decide

private def duplicatePointRejection :=
  SpaceCompiler.Internal.appendPointIdentity checked [lowered.id] lowered.id

/-! Duplicate derived point identity rejects the point without returning the prior identity. -/
example :
    (duplicatePointRejection.toOption,
      compileErrorKindOf duplicatePointRejection) =
    (none, some .duplicatePointIdentity) := by
  native_decide

end Umpire.SpaceTests
