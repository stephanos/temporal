import Umpire.Core

/-!
Typed documentation vocabulary for the semantic inventory.

Outcome owners retain their own status types and publish matchers over those types. The inventory
uses only their erased descriptors, so this module does not introduce a shared outcome enum.
-/

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

/-- Erased documentation for one owner-defined outcome family. -/
structure OutcomeFamilyDescriptor where
  id : String
  owner : String
  description : String
  constructors : List OutcomeConstructorDescriptor
  deriving BEq, DecidableEq, Repr

/-- A rendered projection value that is not a constructor of its owning outcome type. -/
structure ProjectionSentinelDescriptor where
  id : String
  owner : String
  name : String
  description : String
  deriving BEq, DecidableEq, Repr

/-- How a Known Gap enters or crosses the documented pipeline. -/
inductive KnownGapLineage where
  | authored
  | synthesized
  | carried
  deriving BEq, DecidableEq, Ord, Repr

def KnownGapLineage.name : KnownGapLineage → String
  | .authored => "authored"
  | .synthesized => "synthesized"
  | .carried => "carried"

/-- Whether a Known Gap source participates in production or exists only in tests. -/
inductive KnownGapScope where
  | production
  | testOnly
  deriving BEq, DecidableEq, Ord, Repr

def KnownGapScope.name : KnownGapScope → String
  | .production => "production"
  | .testOnly => "test-only"

/-- The six closed source shapes represented by the Known Gap inventory. -/
inductive KnownGapSourceShape where
  | exactKnownGap
  | generatedKnownGapFamily
  | authoredImplementationLinkKnownGapFamily
  | admittedKnownGapInput
  | evidenceGapAdmissionProjection
  | carriedCatalogEntry
  deriving BEq, DecidableEq, Ord, Repr

def KnownGapSourceShape.name : KnownGapSourceShape → String
  | .exactKnownGap => "exact-known-gap"
  | .generatedKnownGapFamily => "generated-known-gap-family"
  | .authoredImplementationLinkKnownGapFamily =>
      "authored-implementation-link-known-gap-family"
  | .admittedKnownGapInput => "admitted-known-gap-input"
  | .evidenceGapAdmissionProjection => "evidence-gap-admission-projection"
  | .carriedCatalogEntry => "carried-catalog-entry"

/-- Closed field mappings for exact Known Gap carry and lossy Observation admission. -/
inductive KnownGapCarryMapping where
  | exact
  | observationAdmission
  deriving BEq, DecidableEq, Ord, Repr

def KnownGapCarryMapping.name : KnownGapCarryMapping → String
  | .exact => "kind -> kind; code -> code; subject -> subject; detail -> detail"
  | .observationAdmission =>
      "code -> code; subject.toList -> relatedDefinitionIds; kind -> absent; detail -> absent"

/-- One typed documentation row for a Known Gap source, projection, or carry boundary. -/
structure KnownGapCatalogDescriptor where
  id : String
  owner : String
  lineage : KnownGapLineage
  scope : KnownGapScope
  shape : KnownGapSourceShape
  source : String
  fieldMapping : Option KnownGapCarryMapping
  description : String
  deriving BEq, DecidableEq, Repr

/-- Catalog identifiers are unique within one assembled descriptor list. -/
def KnownGapCatalogDescriptor.HasUniqueIds
    (catalog : List KnownGapCatalogDescriptor) : Prop :=
  (catalog.map KnownGapCatalogDescriptor.id).Nodup

end Umpire
