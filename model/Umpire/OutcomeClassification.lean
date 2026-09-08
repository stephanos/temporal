import Init.Data.List.Basic

/-! Owner-typed outcome classifiers and projection sentinels shared by semantic stages. -/

namespace Umpire

/-- Documentation retained for one constructor of an owner-defined outcome family. -/
structure OutcomeConstructorDescriptor where
  name : String
  description : String
  deriving BEq, DecidableEq, Repr

/-- A documentation descriptor paired with an owner-typed constructor matcher. -/
structure OutcomeConstructorClassifier (Outcome : Type) where
  descriptor : OutcomeConstructorDescriptor
  accepts : Outcome → Bool

namespace OutcomeConstructorClassifier

/-- Build a constructor classifier for a payload-free outcome value. -/
def ofValue [BEq Outcome]
    (value : Outcome)
    (descriptor : OutcomeConstructorDescriptor) : OutcomeConstructorClassifier Outcome := {
  descriptor
  accepts := fun candidate => candidate == value
}

end OutcomeConstructorClassifier

namespace OutcomeConstructorClassifiers

/-- Erase owner-typed matchers while retaining their documentation order. -/
def descriptors
    (classifiers : List (OutcomeConstructorClassifier Outcome)) :
    List OutcomeConstructorDescriptor :=
  classifiers.map OutcomeConstructorClassifier.descriptor

/-- Return the rendered constructor names in owner-defined order. -/
def names (classifiers : List (OutcomeConstructorClassifier Outcome)) : List String :=
  (descriptors classifiers).map OutcomeConstructorDescriptor.name

/-- Count the owner descriptors that classify one outcome value. -/
def matchCount
    (classifiers : List (OutcomeConstructorClassifier Outcome))
    (outcome : Outcome) : Nat :=
  (classifiers.filter fun classifier => classifier.accepts outcome).length

/-- Every value of an owner-defined outcome family matches exactly one descriptor. -/
def ExactlyOne (classifiers : List (OutcomeConstructorClassifier Outcome)) : Prop :=
  ∀ outcome, matchCount classifiers outcome = 1

/-- Constructor names are unique inside one owner-defined outcome family. -/
def HasUniqueNames (classifiers : List (OutcomeConstructorClassifier Outcome)) : Prop :=
  (names classifiers).Nodup

end OutcomeConstructorClassifiers

/-- A rendered projection value that is not a constructor of its owning outcome type. -/
structure ProjectionSentinelDescriptor where
  id : String
  owner : String
  name : String
  description : String
  deriving BEq, DecidableEq, Repr

end Umpire
