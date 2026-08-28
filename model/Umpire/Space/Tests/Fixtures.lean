import Umpire.Space.Language
import Umpire.Examples.Switch

/-! Shared checked Query closure and authored Space declarations for validation tests. -/

namespace Umpire.SpaceTests

open Umpire

def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Umpire/Space/Tests.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def spaceId : DefinitionId := id "space.test.switch"
def stateAxisId : DefinitionId := id "space.test.axis.state"
def faultAxisId : DefinitionId := id "space.test.axis.fault"
def stateBaselineId : DefinitionId := id "space.test.choice.state-baseline"
def stateOffId : DefinitionId := id "space.test.choice.state-off"
def faultBaselineId : DefinitionId := id "space.test.choice.fault-baseline"
def faultDelayId : DefinitionId := id "space.test.choice.fault-delay"
def delayFaultId : DefinitionId := id "space.test.fault.delay"
def failureFaultId : DefinitionId := id "space.test.fault.failure"
def stateGoalId : DefinitionId := id "space.test.coverage.state-off"
def delayGoalId : DefinitionId := id "space.test.coverage.delay"
def semanticGoalId : DefinitionId := id "space.test.coverage.power-state"
def propertyGoalId : DefinitionId := id "space.test.coverage.property"

def stateBaseline : ChoiceDeclaration := {
  id := stateBaselineId
  source
  baseline := true
}

def stateOff : ChoiceDeclaration := {
  id := stateOffId
  source
  binding := some Umpire.Examples.Switch.offState
}

def faultBaseline : ChoiceDeclaration := {
  id := faultBaselineId
  source
  baseline := true
}

def faultDelay : ChoiceDeclaration := {
  id := faultDelayId
  source
  faults := [delayFaultId]
}

def stateAxis : VariationAxisDeclaration := {
  id := stateAxisId
  source
  role := some Umpire.Examples.Switch.switchRoleId
  choices := [stateOff, stateBaseline]
}

def faultAxis : VariationAxisDeclaration := {
  id := faultAxisId
  source
  choices := [faultDelay, faultBaseline]
}

def delayFault : FaultIntentDeclaration := {
  id := delayFaultId
  source
  occurrence := id "switch.occurrence.flip"
  action := Umpire.Examples.Switch.flipActionId
  capability := Umpire.Examples.Switch.switchCapabilityId
  incompatibleWith := [failureFaultId]
}

def failureFault : FaultIntentDeclaration := {
  id := failureFaultId
  source
  occurrence := id "switch.occurrence.flip"
  action := Umpire.Examples.Switch.flipActionId
  capability := Umpire.Examples.Switch.switchCapabilityId
  incompatibleWith := [delayFaultId]
}

def stateGoal : CoverageGoalDeclaration := {
  id := stateGoalId
  source
  subject := .axisChoice stateAxisId stateOffId
  minimum := 2
}

def delayGoal : CoverageGoalDeclaration := {
  id := delayGoalId
  source
  subject := .fault delayFaultId
  minimum := 2
}

def semanticGoal : CoverageGoalDeclaration := {
  id := semanticGoalId
  source
  subject := .state Umpire.Examples.Switch.powerStateId
  minimum := 4
}

def propertyGoal : CoverageGoalDeclaration := {
  id := propertyGoalId
  source
  subject := .property Umpire.Examples.Switch.flipPropertyId
  minimum := 4
}

def declaration : ExperimentSpaceDeclaration := {
  id := spaceId
  source
  baseQuery := Umpire.Examples.Switch.exactActionQuery.id
  axes := [stateAxis, faultAxis]
  faults := [failureFault, delayFault]
  coverageGoals := [propertyGoal, delayGoal, stateGoal, semanticGoal]
  documentation := "A reusable two-by-two checked Space fixture."
}

def context : SpaceCheckContext Umpire.Examples.Switch.LawStatement :=
  .ofQuery Umpire.Examples.Switch.exactActionQuery

def checkedResult : Except SpaceError
    (CheckedExperimentSpace Umpire.Examples.Switch.LawStatement) :=
  checkExperimentSpace context declaration

def checked : CheckedExperimentSpace Umpire.Examples.Switch.LawStatement :=
  checkedResult.toOption.get (by native_decide)

def errorKindOf
    (result : Except SpaceError (CheckedExperimentSpace LawStatement)) : Option SpaceErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

def canonicalErrorOf
    (result : Except SpaceError (CheckedExperimentSpace LawStatement)) : Option String :=
  match result with
  | .ok _ => none
  | .error error => some (canonicalSpaceErrorJson error)

end Umpire.SpaceTests
