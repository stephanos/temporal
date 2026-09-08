import Umpire.Target

/-! Focused import contract for target authoring and checked composition. -/

#check Umpire.TargetDeclaration
#check Umpire.DefinitionFamily
#check Umpire.DefinitionFamily.id
#check Umpire.TargetDefinition
#check Umpire.FiniteMachine
#check Umpire.FiniteMachine.initialStateCoverage
#check Umpire.FiniteMachine.actionExecutable
#check Umpire.FiniteMachine.kernel
#check Umpire.FiniteMachine.kernelAvailability
#check Umpire.FiniteMachine.planning
#check Umpire.FiniteMachine.authoredPlanning
#check Umpire.FiniteMachine.targetDefinition
#check Umpire.FiniteMachine.authoredTarget
#check Umpire.FiniteTargetDefinition
#check Umpire.FiniteTargetAdmissionError
#check Umpire.ValidatedFiniteTable.machine
#check Umpire.ValidatedFiniteTable.authoredTarget
#check Umpire.FiniteTable.checkTarget
#check Umpire.FiniteModelIdentity
#check Umpire.FiniteModelSetupBinding
#check Umpire.ValidatedFiniteModel
#check Umpire.FiniteTable.validateModel
#check Umpire.ValidatedFiniteModel.stateValue
#check Umpire.ValidatedFiniteModel.setupValue
#check Umpire.FiniteTable.checkModelTarget
#check Umpire.TargetComposition
#check Umpire.TargetComposition.empty
#check Umpire.TargetComposition.provide
#check Umpire.TargetComposition.connect
#check Umpire.AuthoredTarget
#check Umpire.AuthoredTarget.make
#check Umpire.CheckedTarget
#check Umpire.checkTarget
#check Umpire.checkedTarget
#check Umpire.composeTarget
#check Umpire.elaborateTarget

/-- error: Unknown constant `Umpire.CheckedTarget.mk` -/
#guard_msgs (error) in
#check Umpire.CheckedTarget.mk

/-- error: Unknown constant `Umpire.AuthoredTarget.mk` -/
#guard_msgs (error) in
#check Umpire.AuthoredTarget.mk
