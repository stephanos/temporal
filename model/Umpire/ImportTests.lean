import Umpire

/-! Umbrella import contract for the reusable Umpire module family. -/

#check Umpire.DefinitionId
#check Umpire.DefinitionKind
#check Umpire.DefinitionMetadata
#check Umpire.DefinitionError
#check Umpire.SourceLocation
#check Umpire.ModelValue
#check Umpire.ModelValue.named
#check Umpire.ModelTraceStep
#check Umpire.ModelTrace
#check Umpire.ModelSpec
#check Umpire.Providers
#check Umpire.DraftModel
#check Umpire.DraftModel.make
#check Umpire.checkModel
#check Umpire.model
#check Umpire.PropertyDeclaration
#check Umpire.BehaviorDeclaration
#check Umpire.BehaviorCheckContext.ofTarget
#check Umpire.QueryDeclaration
#check Umpire.IncrementalPlannerKernel
#check Umpire.ExperimentSpec
#check Umpire.ExecutionHandoffDeclaration
#check Umpire.ExecutionHandoff
#check Umpire.ExperimentSpaceDeclaration
#check Umpire.CheckedExperimentSpace
#check Umpire.checkExperimentSpace
#check Umpire.CheckedSpaceMetadata
#check Umpire.projectCheckedSpaceMetadata
#check Umpire.LoweredSpacePoint
#check Umpire.lowerSpacePoint
#check Umpire.compileBatch
#check Umpire.ObservationMappingDeclaration
#check Umpire.CheckedObservationPlan
#check Umpire.Case
#check Umpire.Case.Program
#check Umpire.Case.Contract
#check Umpire.Case.Run
#check Umpire.Case.ValueType

/-! Neutral outcome-classification constructor contracts stay usable without inventory imports. -/
#check Umpire.OutcomeConstructorDescriptor
#check Umpire.OutcomeConstructorClassifier.ofValue
#check Umpire.OutcomeConstructorClassifiers.descriptors
#check Umpire.OutcomeConstructorClassifiers.names
#check Umpire.OutcomeConstructorClassifiers.HasUniqueNames
#check Umpire.ProjectionSentinelDescriptor

example (Outcome : Type) (descriptor : Umpire.OutcomeConstructorDescriptor) :
    Umpire.OutcomeConstructorClassifiers.ExactlyOne
      ([{ descriptor, accepts := fun (_ : Outcome) => true }] :
        List (Umpire.OutcomeConstructorClassifier Outcome)) := by
  intro outcome
  rfl
