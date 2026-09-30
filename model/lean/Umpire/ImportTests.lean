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
#check Umpire.Property
#check Umpire.Scenario
#check Umpire.ScenarioCheckContext.ofTarget
#check Umpire.Query
#check Umpire.SearchView
#check Umpire.Plan
#check Umpire.VariationSpace
#check Umpire.CheckedVariationSpace
#check Umpire.checkVariationSpace
#check Umpire.CheckedSpaceMetadata
#check Umpire.projectCheckedSpaceMetadata
#check Umpire.PlannedVariant
#check Umpire.lowerSpacePoint
#check Umpire.compileBatch
#check Umpire.Evidence.Reading
#check Umpire.Evidence.CheckedReading
#check Umpire.Provenance.Metadata
#check Umpire.Provenance.DefinitionBinding
#check Umpire.Provenance.DefinitionKind
#check Umpire.Provenance.KnownGap
#check Umpire.Provenance.make

/-! Neutral outcome-classification constructor contracts stay usable without inventory imports. -/
#check Umpire.OutcomeConstructorDescriptor
#check Umpire.OutcomeConstructorClassifier.ofValue
#check Umpire.OutcomeConstructorClassifiers.descriptors
#check Umpire.OutcomeConstructorClassifiers.names
#check Umpire.OutcomeConstructorClassifiers.HasUniqueNames
#check Umpire.NotRunMarker

example (Outcome : Type) (descriptor : Umpire.OutcomeConstructorDescriptor) :
    Umpire.OutcomeConstructorClassifiers.ExactlyOne
      ([{ descriptor, accepts := fun (_ : Outcome) => true }] :
        List (Umpire.OutcomeConstructorClassifier Outcome)) := by
  intro outcome
  rfl
