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

private theorem except_eq_ok_get
    (result : Except ε α)
    (isSome : result.toOption.isSome = true) :
    result = .ok (result.toOption.get isSome) := by
  cases result with
  | error error => cases isSome
  | ok value => rfl

private theorem checkedResultEq : checkedResult = .ok checked :=
  except_eq_ok_get checkedResult (by native_decide)

private theorem checkedTargetEq :
    checked.baseQuery.target = Umpire.Examples.Switch.target := by
  exact congrArg (fun query => query.target)
    (checkExperimentSpace_baseQuery checkedResultEq)

private def baseKernel : IncrementalPlannerKernel checked.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def transportedKernel : IncrementalPlannerKernel lowered.query.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel lowered.targetEq)
    baseKernel

private def transportedRun :=
  planWithArtifactIntent lowered.query transportedKernel lowered.intent

private def batchResult :=
  compileBatch checked baseKernel

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

private def verifiedPlannerRun : PlannerRun :=
  Umpire.PlanningTests.run 2 (.verify Umpire.PlanningTests.property) .exhaustive

private def verifiedPointRejection :=
  SpaceCompiler.Internal.appendPlannerRun checked lowered.id
    [Umpire.Examples.Switch.compiledArtifact] verifiedPlannerRun

/-!
A verified point with no Artifact rejects the canonical point and never returns its existing prefix.
-/
example :
    (verifiedPointRejection.toOption,
      (compileErrorOf verifiedPointRejection).map fun error => (error.kind, error.pointId)) =
    (none, some (.verifiedWithoutArtifact, lowered.id)) := by
  native_decide

private def exhaustedPlannerRun : PlannerRun :=
  Umpire.PlanningTests.run 64 (.counterexample Umpire.PlanningTests.property)
    .shortest 1 17 false

private def absentPlannerRun : PlannerRun :=
  Umpire.PlanningTests.run 0 (.counterexample Umpire.PlanningTests.property) .exhaustive

private def staticallyUnsatisfiableBehavior : CheckedBehavior := {
  Umpire.PlanningTests.behavior with
  spaceStatus := .unsatisfiable
  behaviorFingerprint := behaviorFingerprintOf "space-compiler-test/unsatisfiable"
}

private def unsatisfiablePlannerRun : PlannerRun :=
  Umpire.PlanningTests.run 0 (.verify Umpire.PlanningTests.property) .exhaustive
    10 17 true staticallyUnsatisfiableBehavior

private def rejectedPlannerKind (run : PlannerRun) :
    Option (SpaceCompilationErrorKind × DefinitionId) :=
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

private def foundPlannerRun : PlannerRun :=
  Umpire.PlanningTests.run 2 (.witness Umpire.PlanningTests.property) .shortest

private def foundPlannerSpec : ExperimentSpec :=
  foundPlannerRun.artifact.get (by native_decide)

private def duplicateSpecRejection :=
  SpaceCompiler.Internal.appendPlannerRun checked lowered.id [foundPlannerSpec] foundPlannerRun

/-! Duplicate final ExperimentSpec identity rejects the point without returning the prior spec. -/
example :
    (duplicateSpecRejection.toOption,
      compileErrorKindOf duplicateSpecRejection) =
    (none, some .duplicateExperimentSpecIdentity) := by
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
