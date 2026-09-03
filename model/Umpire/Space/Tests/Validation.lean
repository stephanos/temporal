import Umpire.Space.Tests.Fixtures

/-! Exact v1 bounds, reference closure, effect conflicts, and canonical error checks. -/

namespace Umpire.SpaceTests

open Umpire

example : ChoiceDeclaration.baselineChoice stateBaselineId source = ({
    id := stateBaselineId
    source
    baseline := true
  } : ChoiceDeclaration) := by
  rfl

example : ChoiceDeclaration.boundValue stateOffId source Umpire.Examples.Switch.offState = ({
    id := stateOffId
    source
    binding := some Umpire.Examples.Switch.offState
  } : ChoiceDeclaration) := by
  rfl

example : ChoiceDeclaration.selectedFault faultDelayId source delayFaultId = ({
    id := faultDelayId
    source
    faults := [delayFaultId]
  } : ChoiceDeclaration) := by
  rfl

example : VariationAxisDeclaration.faultAxis faultAxisId source [faultDelay, faultBaseline] = ({
    id := faultAxisId
    source
    choices := [faultDelay, faultBaseline]
  } : VariationAxisDeclaration) := by
  rfl

example : FaultIntentDeclaration.atOccurrence delayFaultId source
    (id "switch.occurrence.flip") Umpire.Examples.Switch.flipActionId
    Umpire.Examples.Switch.switchCapabilityId = ({
      id := delayFaultId
      source
      occurrence := id "switch.occurrence.flip"
      action := Umpire.Examples.Switch.flipActionId
      capability := Umpire.Examples.Switch.switchCapabilityId
    } : FaultIntentDeclaration) := by
  rfl

example : CoverageGoalDeclaration.seek stateGoalId source
    (.axisChoice stateAxisId stateOffId) 2 = ({
      id := stateGoalId
      source
      subject := .axisChoice stateAxisId stateOffId
      minimum := 2
    } : CoverageGoalDeclaration) := by
  rfl

example : [
    DefinitionKind.experimentSpace.name,
    DefinitionKind.variationAxis.name,
    DefinitionKind.choice.name,
    DefinitionKind.fault.name,
    DefinitionKind.coverageGoal.name
  ] = ["experiment-space", "variation-axis", "choice", "fault", "coverage-goal"] := by
  native_decide

def spaceValuedRoleContext : BehaviorCheckContext := {
  definitions := [{
    id := id "space.test.value.metadata"
    kind := .experimentSpace
    source
    canonicalBehavior := "space-value/v1"
  }]
}

def spaceValuedRole : BehaviorDeclaration := {
  id := id "space.test.behavior.invalid-role"
  source
  roles := [{ id := id "space.test.role.invalid", valueKind := .experimentSpace }]
}

example : (checkBehavior spaceValuedRoleContext spaceValuedRole).toOption.isNone = true := by
  native_decide

example : checked.pointCount = 4 ∧
    checked.axes.map CheckedVariationAxis.id = [faultAxisId, stateAxisId] ∧
    checked.faults.map CheckedFaultIntent.id = [delayFaultId, failureFaultId] ∧
    checked.coverageGoals.map CheckedCoverageGoal.id =
      [delayGoalId, semanticGoalId, propertyGoalId, stateGoalId] := by
  native_decide

def reordered : ExperimentSpaceDeclaration := {
  declaration with
  axes := declaration.axes.reverse
  faults := declaration.faults.reverse
  coverageGoals := declaration.coverageGoals.reverse
}

def deeplyReordered : ExperimentSpaceDeclaration := {
  declaration with
  axes := (declaration.axes.map fun axis => { axis with choices := axis.choices.reverse }).reverse
  faults := (declaration.faults.map fun fault => {
    fault with incompatibleWith := fault.incompatibleWith.reverse
  }).reverse
  coverageGoals := declaration.coverageGoals.reverse
}

example : (checkExperimentSpace context reordered).toOption.map canonicalExperimentSpaceJson =
      checkedResult.toOption.map canonicalExperimentSpaceJson ∧
    (checkExperimentSpace context reordered).toOption.map CheckedExperimentSpace.behaviorFingerprint =
      checkedResult.toOption.map CheckedExperimentSpace.behaviorFingerprint := by
  native_decide

example : (checkExperimentSpace context deeplyReordered).toOption.map canonicalExperimentSpaceJson =
      checkedResult.toOption.map canonicalExperimentSpaceJson ∧
    (checkExperimentSpace context deeplyReordered).toOption.map
      CheckedExperimentSpace.behaviorFingerprint =
      checkedResult.toOption.map CheckedExperimentSpace.behaviorFingerprint ∧
    (checkExperimentSpace context deeplyReordered).toOption.any fun candidate =>
      candidate == checked := by
  native_decide

def checkedError (candidate : ExperimentSpaceDeclaration) : Option SpaceErrorKind :=
  errorKindOf (checkExperimentSpace context candidate)

def generatedChoice (axisIndex choiceIndex : Nat) : ChoiceDeclaration := {
  id := id ("space.test.choice.generated-" ++ toString axisIndex ++ "-" ++ toString choiceIndex)
  source
  baseline := choiceIndex == 0
  faults := if choiceIndex == 0 then [] else
    [id ("space.test.fault.generated-" ++ toString axisIndex ++ "-" ++ toString choiceIndex)]
}

def generatedAxis (axisIndex choiceCount : Nat) : VariationAxisDeclaration := {
  id := id ("space.test.axis.generated-" ++ toString axisIndex)
  source
  choices := (List.range choiceCount).map (generatedChoice axisIndex)
}

def pointOverflowDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := (List.range 5).map fun index => generatedAxis index 4
}

def duplicateInvalidAxisId : DefinitionId := id "space.test.axis.duplicate-invalid"

def duplicateInvalidShortAxis : VariationAxisDeclaration := {
  generatedAxis 90 1 with id := duplicateInvalidAxisId
}

def duplicateInvalidLongAxis : VariationAxisDeclaration := {
  generatedAxis 91 17 with id := duplicateInvalidAxisId
}

def duplicateInvalidAxes : ExperimentSpaceDeclaration := {
  declaration with axes := [duplicateInvalidShortAxis, duplicateInvalidLongAxis]
}

def reversedDuplicateInvalidAxes : ExperimentSpaceDeclaration := {
  duplicateInvalidAxes with axes := duplicateInvalidAxes.axes.reverse
}

def singletonAxis : VariationAxisDeclaration := {
  stateAxis with choices := [stateBaseline]
}

example : [
    checkedError { declaration with axes := [] },
    checkedError { declaration with axes := List.replicate 9 stateAxis },
    checkedError { declaration with axes := [singletonAxis] },
    checkedError { declaration with axes := [generatedAxis 100 17] },
    checkedError pointOverflowDeclaration,
    checkedError { declaration with faults := List.replicate 13 delayFault },
    checkedError { declaration with coverageGoals := [] },
    checkedError { declaration with coverageGoals := List.replicate 65 stateGoal }
  ] = [
    some .axisCountOutOfRange,
    some .axisCountOutOfRange,
    some .choiceCountOutOfRange,
    some .choiceCountOutOfRange,
    some .pointCountExceeded,
    some .faultCountOutOfRange,
    some .coverageGoalCountOutOfRange,
    some .coverageGoalCountOutOfRange
  ] := by
  native_decide

example : checkedError duplicateInvalidAxes = some .duplicateDefinitionId ∧
    canonicalErrorOf (checkExperimentSpace context duplicateInvalidAxes) =
      canonicalErrorOf (checkExperimentSpace context reversedDuplicateInvalidAxes) := by
  native_decide

def caseCollidingAxis : VariationAxisDeclaration := {
  stateAxis with
  choices := [stateOff, { stateOff with id := id "SPACE.TEST.CHOICE.STATE-OFF" }]
}

def secondStateAxis : VariationAxisDeclaration := {
  id := id "space.test.axis.second-state"
  source
  role := some Umpire.Examples.Switch.switchRoleId
  choices := [
    { stateBaseline with id := id "space.test.choice.second-state-baseline" },
    { stateOff with id := id "space.test.choice.second-state-off" }
  ]
}

def baselineWithEffect : ChoiceDeclaration := { stateOff with baseline := true }
def secondBaseline : ChoiceDeclaration := {
  stateBaseline with id := id "space.test.choice.second-baseline"
}
def emptyNonBaseline : ChoiceDeclaration := {
  stateOff with binding := none
}
def duplicateStateEffect : ChoiceDeclaration := {
  stateOff with id := id "space.test.choice.duplicate-state-effect"
}

def stateAxisWith (choices : List ChoiceDeclaration) : VariationAxisDeclaration := {
  stateAxis with choices
}

def withStateAxis (axis : VariationAxisDeclaration) : ExperimentSpaceDeclaration := {
  declaration with axes := [axis, faultAxis]
}

example : [
    checkedError { declaration with baseQuery := id "space.test.query.stale" },
    checkedError (withStateAxis caseCollidingAxis),
    checkedError { declaration with axes := [stateAxis, secondStateAxis, faultAxis] },
    checkedError (withStateAxis (stateAxisWith [baselineWithEffect, stateBaseline])),
    checkedError (withStateAxis (stateAxisWith [stateBaseline, secondBaseline])),
    checkedError (withStateAxis (stateAxisWith [stateBaseline, emptyNonBaseline])),
    checkedError (withStateAxis (stateAxisWith [stateOff, duplicateStateEffect]))
  ] = [
    some .baseQueryMismatch,
    some .duplicateDefinitionId,
    some .duplicateControlledRole,
    some .baselineHasEffect,
    some .multipleBaseline,
    some .emptyChoiceEffect,
    some .duplicateChoiceEffect
  ] := by
  native_decide

def missingRoleAxis : VariationAxisDeclaration := {
  stateAxis with role := some (id "space.test.role.missing")
}

def outcomeRoleId : DefinitionId := id "space.test.role.outcome"
def outcomeRole : ResourceRole := {
  id := outcomeRoleId
  valueKind := .outcome
}

def outcomeQuery : CheckedQuery Umpire.Examples.Switch.LawStatement := {
  Umpire.Examples.Switch.exactActionQuery with
  behavior := {
    Umpire.Examples.Switch.exactActionQuery.behavior with roles := [outcomeRole]
  }
}

def outcomeContext : SpaceCheckContext Umpire.Examples.Switch.LawStatement := .ofQuery outcomeQuery

def outcomeAxis : VariationAxisDeclaration := {
  stateAxis with
  role := some outcomeRoleId
  choices := [
    stateBaseline,
    { stateOff with binding := some Umpire.Examples.Switch.appliedOutcome }
  ]
}

def outcomeRoleError : Option SpaceErrorKind :=
  errorKindOf (checkExperimentSpace outcomeContext (withStateAxis outcomeAxis))

def withStateBinding (value : ModelValue) : ExperimentSpaceDeclaration :=
  withStateAxis (stateAxisWith [stateBaseline, { stateOff with binding := some value }])

def conflictingQuery : CheckedQuery Umpire.Examples.Switch.LawStatement := {
  Umpire.Examples.Switch.exactActionQuery with
  behavior := {
    Umpire.Examples.Switch.exactActionQuery.behavior with
    setup := [{
      id := id "space.test.setup.conflict"
      relation := .different
      left := .role Umpire.Examples.Switch.switchRoleId
      right := .value Umpire.Examples.Switch.offState
    }]
  }
}

def conflictingContext : SpaceCheckContext Umpire.Examples.Switch.LawStatement :=
  .ofQuery conflictingQuery

def conflictingBindingError : Option SpaceErrorKind :=
  errorKindOf (checkExperimentSpace conflictingContext declaration)

example : [
    checkedError (withStateAxis missingRoleAxis),
    outcomeRoleError,
    checkedError (withStateBinding {
      definitionId := id "space.test.value.missing"
      value := "missing"
    }),
    checkedError (withStateBinding Umpire.Examples.Switch.flipAction),
    checkedError (withStateBinding Umpire.Examples.Switch.onState),
    checkedError (withStateBinding {
      definitionId := Umpire.Examples.Switch.flipPropertyId
      value := "property"
    }),
    conflictingBindingError
  ] = [
    some .unknownRole,
    some .unwritableRole,
    some .unknownValue,
    some .wrongValueKind,
    some .unavailableValue,
    some .unknownValue,
    some .conflictingBinding
  ] := by
  native_decide

def withFaultChoice (choice : ChoiceDeclaration) : ExperimentSpaceDeclaration := {
  declaration with axes := [stateAxis, { faultAxis with choices := [faultBaseline, choice] }]
}

def ghostFaultChoice : ChoiceDeclaration := {
  faultDelay with faults := [id "space.test.fault.missing"]
}

def duplicateFaultChoice : ChoiceDeclaration := {
  faultDelay with faults := [delayFaultId, delayFaultId]
}

def duplicateSelectionDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := [
    { stateAxis with choices := [stateBaseline, { stateOff with faults := [delayFaultId] }] },
    faultAxis
  ]
}

def incompatibleSelectionDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := [
    { stateAxis with choices := [stateBaseline, { stateOff with faults := [failureFaultId] }] },
    faultAxis
  ]
}

def unknownOccurrenceFault : FaultIntentDeclaration := {
  delayFault with occurrence := id "space.test.occurrence.missing"
}

def mismatchedActionFault : FaultIntentDeclaration := {
  delayFault with action := id "space.test.action.stale"
}

def unknownCapabilityFault : FaultIntentDeclaration := {
  delayFault with capability := id "space.test.capability.missing"
}

def wrongKindCapabilityFault : FaultIntentDeclaration := {
  delayFault with capability := Umpire.Examples.Switch.flipActionId
}

def asymmetricFailureFault : FaultIntentDeclaration := {
  failureFault with incompatibleWith := []
}

def withDelayFault (fault : FaultIntentDeclaration) : ExperimentSpaceDeclaration := {
  declaration with faults := [fault, failureFault]
}

def ghostCapabilityId : DefinitionId := id "space.test.capability.unprovided"

def ghostCapabilityMetadata : DefinitionMetadata := {
  id := ghostCapabilityId
  kind := .capability
  source
  canonicalBehavior := "space-unprovided-capability/v1"
}

def targetWithGhostAuthoring : AuthoredTarget Umpire.Examples.Switch.LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make {
    Umpire.Examples.Switch.targetDefinition with
    definitions := Umpire.Examples.Switch.definitions ++ [ghostCapabilityMetadata]
  } Umpire.Examples.Switch.targetComposition
    (.available Umpire.Examples.Switch.transitionKernel rfl Umpire.Examples.Switch.finitePlanning)

def targetWithGhost : QueryTarget Umpire.Examples.Switch.LawStatement :=
  checkedTarget targetWithGhostAuthoring

def queryWithGhost : CheckedQuery Umpire.Examples.Switch.LawStatement := {
  Umpire.Examples.Switch.exactActionQuery with
  target := targetWithGhost
  completeness := (CheckedQueryTarget.ofTarget targetWithGhost).completeness
}

def ghostContext : SpaceCheckContext Umpire.Examples.Switch.LawStatement := .ofQuery queryWithGhost

def missingCapabilityFault : FaultIntentDeclaration := {
  delayFault with capability := ghostCapabilityId
}

def missingCapabilityError : Option SpaceErrorKind :=
  errorKindOf (checkExperimentSpace ghostContext (withDelayFault missingCapabilityFault))

example : [
    checkedError (withFaultChoice ghostFaultChoice),
    checkedError (withFaultChoice duplicateFaultChoice),
    checkedError duplicateSelectionDeclaration,
    checkedError incompatibleSelectionDeclaration,
    checkedError (withDelayFault unknownOccurrenceFault),
    checkedError (withDelayFault mismatchedActionFault),
    checkedError (withDelayFault unknownCapabilityFault),
    checkedError (withDelayFault wrongKindCapabilityFault),
    missingCapabilityError,
    checkedError { declaration with faults := [delayFault, asymmetricFailureFault] }
  ] = [
    some .unknownFault,
    some .duplicateFaultSelection,
    some .duplicateFaultSelection,
    some .incompatibleFaultSelection,
    some .unknownOccurrence,
    some .occurrenceActionMismatch,
    some .unknownCapability,
    some .wrongReferenceKind,
    some .missingCapability,
    some .asymmetricFaultIncompatibility
  ] := by
  native_decide

def goalWith
    (goalId : String)
    (subject : CoverageSubject)
    (minimum : Nat) : CoverageGoalDeclaration := {
  id := id goalId
  source
  subject
  minimum
}

def withGoal (goal : CoverageGoalDeclaration) : ExperimentSpaceDeclaration := {
  declaration with coverageGoals := [goal]
}

example : [
    checkedError (withGoal (goalWith "space.test.coverage.zero" (.state
      Umpire.Examples.Switch.powerStateId) 0)),
    checkedError (withGoal (goalWith "space.test.coverage.out-of-range" (.state
      Umpire.Examples.Switch.powerStateId) 5)),
    checkedError (withGoal (goalWith "space.test.coverage.unknown-axis"
      (.axisChoice (id "space.test.axis.missing") stateOffId) 1)),
    checkedError (withGoal (goalWith "space.test.coverage.unknown-choice"
      (.axisChoice stateAxisId (id "space.test.choice.missing")) 1)),
    checkedError (withGoal (goalWith "space.test.coverage.unknown-fault"
      (.fault (id "space.test.fault.missing")) 1)),
    checkedError (withGoal (goalWith "space.test.coverage.unknown-state"
      (.state (id "space.test.state.missing")) 1)),
    checkedError (withGoal (goalWith "space.test.coverage.wrong-kind"
      (.state Umpire.Examples.Switch.flipActionId) 1)),
    checkedError (withGoal (goalWith "space.test.coverage.unknown-property"
      (.property (id "space.test.property.missing")) 1)),
    checkedError (withGoal (goalWith "space.test.coverage.impossible-choice"
      (.axisChoice stateAxisId stateOffId) 3)),
    checkedError (withGoal (goalWith "space.test.coverage.impossible-fault"
      (.fault delayFaultId) 3)),
    checkedError { declaration with coverageGoals := [
      stateGoal,
      { stateGoal with id := id "SPACE.TEST.COVERAGE.STATE-OFF" }
    ] }
  ] = [
    some .invalidCoverageMinimum,
    some .invalidCoverageMinimum,
    some .unknownCoverageSubject,
    some .unknownCoverageSubject,
    some .unknownCoverageSubject,
    some .unknownCoverageSubject,
    some .wrongCoverageSubjectKind,
    some .unknownCoverageSubject,
    some .impossibleCoverageGoal,
    some .impossibleCoverageGoal,
    some .duplicateDefinitionId
  ] := by
  native_decide

def firstUnknownFaultChoice : ChoiceDeclaration := {
  id := id "space.test.choice.first-unknown-fault"
  source
  faults := [id "space.test.fault.z-missing"]
}

def secondUnknownFaultChoice : ChoiceDeclaration := {
  id := id "space.test.choice.second-unknown-fault"
  source
  faults := [id "space.test.fault.a-missing"]
}

def reorderedInvalidAxis : VariationAxisDeclaration := {
  faultAxis with choices := [firstUnknownFaultChoice, secondUnknownFaultChoice]
}

def reorderedInvalidDeclaration : ExperimentSpaceDeclaration := {
  declaration with axes := [stateAxis, reorderedInvalidAxis]
}

example : canonicalErrorOf (checkExperimentSpace context reorderedInvalidDeclaration) =
    canonicalErrorOf (checkExperimentSpace context {
      reorderedInvalidDeclaration with
      axes := reorderedInvalidDeclaration.axes.reverse
      coverageGoals := reorderedInvalidDeclaration.coverageGoals.reverse
      faults := reorderedInvalidDeclaration.faults.reverse
    }) := by
  native_decide

end Umpire.SpaceTests
