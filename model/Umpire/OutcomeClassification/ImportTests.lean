import Umpire.OutcomeClassification

/-! Neutral constructor contracts remain usable without inventory or concrete stage imports. -/

namespace Umpire.OutcomeClassification.ImportTests

#check OutcomeConstructorDescriptor
#check OutcomeConstructorClassifier.ofValue
#check OutcomeConstructorClassifiers.descriptors
#check OutcomeConstructorClassifiers.names
#check OutcomeConstructorClassifiers.HasUniqueNames
#check ProjectionSentinelDescriptor

example (Outcome : Type) (descriptor : OutcomeConstructorDescriptor) :
    OutcomeConstructorClassifiers.ExactlyOne
      ([{ descriptor, accepts := fun (_ : Outcome) => true }] :
        List (OutcomeConstructorClassifier Outcome)) := by
  intro outcome
  rfl

end Umpire.OutcomeClassification.ImportTests
