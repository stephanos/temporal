import Umpire.Model

/-! Focused import contract for target authoring and checked composition. -/

#check Umpire.ModelSpec
#check Umpire.DefinitionFamily
#check Umpire.DefinitionFamily.id
#check Umpire.FiniteMachine
#check Umpire.FiniteMachine.initialStateCoverage
#check Umpire.FiniteMachine.actionExecutable
#check Umpire.FiniteMachine.kernel
#check Umpire.FiniteMachine.machineAvailability
#check Umpire.FiniteMachine.planning
#check Umpire.FiniteMachine.authoredPlanning
#check Umpire.FiniteMachine.modelSpec
#check Umpire.FiniteMachine.draftModel
#check Umpire.TableModelSpec
#check Umpire.TableAdmissionError
#check Umpire.CheckedTable.machine
#check Umpire.CheckedTable.draftModel
#check Umpire.FiniteTable.checkModel
#check Umpire.FiniteModelIdentity
#check Umpire.FiniteModelSetupBinding
#check Umpire.CheckedTableModel
#check Umpire.FiniteTable.checkIdentity
#check Umpire.CheckedTableModel.stateValue
#check Umpire.CheckedTableModel.setupValue
#check Umpire.Providers
#check Umpire.Providers.empty
#check Umpire.Providers.provide
#check Umpire.Providers.connect
#check Umpire.DraftModel
#check Umpire.DraftModel.make
#check Umpire.CheckedModel
#check Umpire.checkModel
#check Umpire.model
#check Umpire.elabModel

/-- error: Unknown constant `Umpire.CheckedModel.mk` -/
#guard_msgs (error) in
#check Umpire.CheckedModel.mk

/-- error: Unknown constant `Umpire.DraftModel.mk` -/
#guard_msgs (error) in
#check Umpire.DraftModel.mk
