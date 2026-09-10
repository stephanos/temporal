import Umpire.KnownGap
import Umpire.OutcomeClassification

/-!
Typed documentation vocabulary for the semantic inventory.

Outcome owners retain their own status types and publish matchers over those types. The inventory
uses only their erased descriptors, so this module does not introduce a shared outcome enum.
-/

namespace Umpire

/-- Erased documentation for one owner-defined outcome family. -/
structure OutcomeFamilyDescriptor where
  id : String
  owner : String
  description : String
  constructors : List OutcomeConstructorDescriptor
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
  | authoredUnmappedSourceFamily
  | admittedKnownGapInput
  | evidenceGapAdmissionProjection
  | carriedCatalogEntry
  deriving BEq, DecidableEq, Ord, Repr

def KnownGapSourceShape.name : KnownGapSourceShape → String
  | .exactKnownGap => "exact-known-gap"
  | .generatedKnownGapFamily => "generated-known-gap-family"
  | .authoredUnmappedSourceFamily =>
      "authored-implementation-link-known-gap-family"
  | .admittedKnownGapInput => "admitted-known-gap-input"
  | .evidenceGapAdmissionProjection => "evidence-gap-admission-projection"
  | .carriedCatalogEntry => "carried-catalog-entry"

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
