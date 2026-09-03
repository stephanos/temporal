import Umpire.SemanticInventory.KnownGaps

/-! Fixed and synthesized production Known Gap sources stay closed and canonical. -/

namespace Umpire.SemanticInventoryTests.KnownGaps

open Umpire
open Umpire.SemanticInventory

private def validationErrorKind?
    (sources : List KnownGapSourceDescriptor) : Option KnownGapSourceErrorKind :=
  match validateProductionKnownGapSources sources with
  | .ok _ => none
  | .error error => some error.kind

private def exactGaps (sources : List KnownGapSourceDescriptor) : List KnownGap :=
  sources.filterMap fun descriptor => match descriptor.source with
    | .exact gap => some gap
    | .namespacedPrefix _ _ _ => none

/-- A reusable-boundary fixture for the owner-published admitted input catalog row. -/
private def requestRawKnownGapInputCatalogRow : KnownGapCatalogDescriptor := {
  id := "umpire.semantic-inventory.known-gap-source.17-request-raw-known-gap-input"
  owner := "Umpire.SemanticInventoryTests.KnownGaps"
  lineage := .carried
  scope := .production
  shape := .admittedKnownGapInput
  source := "umpire.run-evaluation.request-and-raw-known-gap-input"
  fieldMapping := none
  description := "Validated request and Raw Evidence Known Gaps before stage-specific projection."
}

private def knownGapCatalog : List KnownGapCatalogDescriptor :=
  Umpire.SemanticInventory.knownGapCatalog requestRawKnownGapInputCatalogRow

private def sourceAt (index : Nat) : KnownGapSourceDescriptor :=
  productionKnownGapSources[index]?.getD observationKnownGapSource

private def catalogValidationErrorKindFor?
    (requestRawKnownGapInputCatalogRow : KnownGapCatalogDescriptor)
    (catalog : List KnownGapCatalogDescriptor) : Option KnownGapCatalogErrorKind :=
  match validateKnownGapCatalog requestRawKnownGapInputCatalogRow catalog with
  | .ok _ => none
  | .error error => some error.kind

private def catalogValidationErrorKind?
    (catalog : List KnownGapCatalogDescriptor) : Option KnownGapCatalogErrorKind :=
  catalogValidationErrorKindFor? requestRawKnownGapInputCatalogRow catalog

private def catalogAt (index : Nat) : KnownGapCatalogDescriptor :=
  knownGapCatalog[index]?.getD {
    id := "umpire.semantic-inventory.known-gap-source.unknown"
    owner := "Umpire.SemanticInventory"
    lineage := .carried
    scope := .production
    shape := .carriedCatalogEntry
    source := "umpire.semantic-inventory.known-gap-source.unknown"
    fieldMapping := some .exact
    description := "Unknown catalog row."
  }

example : exactGaps plannerKnownGapSources = canonicalPlannerKnownGaps.toList := by
  native_decide

example : plannerKnownGapSources.length = 8 ∧
    productionKnownGapSources.length = 9 ∧
    validationErrorKind? productionKnownGapSources = none := by
  native_decide

example : productionKnownGapSources.map (DefinitionId.value ∘ KnownGapSourceDescriptor.id) = [
    "umpire.semantic-inventory.known-gap-source.01-execution-evidence",
    "umpire.semantic-inventory.known-gap-source.02-artifact-migrations",
    "umpire.semantic-inventory.known-gap-source.03-artifact-reading",
    "umpire.semantic-inventory.known-gap-source.04-evidence-evaluation",
    "umpire.semantic-inventory.known-gap-source.05-runtime-scheduler-order",
    "umpire.semantic-inventory.known-gap-source.06-runtime-storage-order",
    "umpire.semantic-inventory.known-gap-source.07-runtime-transport-order",
    "umpire.semantic-inventory.known-gap-source.08-promotion",
    "umpire.semantic-inventory.known-gap-source.09-observation-diagnostic"
    ] := by
  native_decide

example : observationKnownGapSource.source = .namespacedPrefix .interpretation
      (DefinitionId.of "umpire.observation") observationKnownGapSuffixes ∧
    observationKnownGap .missingClosure (DefinitionId.of "temporal.observation.fixture") = {
      kind := .interpretation
      code := DefinitionId.of "umpire.observation.missing-closure"
      subject := some (DefinitionId.of "temporal.observation.fixture")
    } := by
  native_decide

example :
    let first := sourceAt 0
    validationErrorKind? (first :: productionKnownGapSources) = some .duplicateId := by
  native_decide

example :
    let first := sourceAt 0
    let second := sourceAt 1
    let duplicate := { second with source := first.source }
    validationErrorKind? (first :: duplicate :: productionKnownGapSources.drop 2) =
      some .duplicateCode := by
  native_decide

example :
    let first := sourceAt 0
    let invalid := { first with id := DefinitionId.of "invalid" }
    validationErrorKind? (invalid :: productionKnownGapSources.drop 1) = some .invalidId := by
  native_decide

example :
    let first := sourceAt 0
    let invalidGap := { plannerExecutionEvidenceKnownGap with code := DefinitionId.of "invalid" }
    let invalid := { first with source := .exact invalidGap }
    validationErrorKind? (invalid :: productionKnownGapSources.drop 1) = some .invalidCode := by
  native_decide

example :
    let last := productionKnownGapSources.getLast?.getD observationKnownGapSource
    let invalid := { last with
      source := .namespacedPrefix .interpretation (DefinitionId.of "observation")
        observationKnownGapSuffixes }
    validationErrorKind? (productionKnownGapSources.dropLast ++ [invalid]) =
      some .invalidPrefix := by
  native_decide

example :
    let first := sourceAt 0
    let invalid := { first with lineage := .synthesized }
    validationErrorKind? (invalid :: productionKnownGapSources.drop 1) =
      some .invalidLineage := by
  native_decide

example :
    let first := sourceAt 0
    let invalid := { first with scope := .testOnly }
    validationErrorKind? (invalid :: productionKnownGapSources.drop 1) =
      some .invalidScope := by
  native_decide

example : validationErrorKind? productionKnownGapSources.reverse = some .noncanonicalOrder := by
  native_decide

example :
    let changedKind : KnownGapSource := .namespacedPrefix .claim
      (DefinitionId.of "umpire.observation") ["missing-closure"]
    changedKind.materialize "missing-closure" (some (DefinitionId.of "umpire.fixture")) = {
      kind := .claim
      code := DefinitionId.of "umpire.observation.missing-closure"
      subject := some (DefinitionId.of "umpire.fixture")
    } := by
  native_decide

example :
    let requestGap : KnownGap := {
      kind := .interpretation
      code := DefinitionId.of "umpire.observation.raw-unknown"
    }
    productionKnownGapSources.all (fun descriptor => !descriptor.source.covers requestGap) := by
  native_decide

example : knownGapCatalog.length = 24 ∧
    catalogValidationErrorKind? knownGapCatalog = none ∧
    (knownGapCatalog.drop 9).map KnownGapCatalogDescriptor.id = [
      "umpire.semantic-inventory.known-gap-source.10-implementation-link-setup",
      "umpire.semantic-inventory.known-gap-source.11-implementation-link-state",
      "umpire.semantic-inventory.known-gap-source.12-implementation-link-action",
      "umpire.semantic-inventory.known-gap-source.13-implementation-link-outcome",
      "umpire.semantic-inventory.known-gap-source.14-implementation-link-observation",
      "umpire.semantic-inventory.known-gap-source.15-implementation-link-relation",
      "umpire.semantic-inventory.known-gap-source.16-implementation-link-capability",
      "umpire.semantic-inventory.known-gap-source.17-request-raw-known-gap-input",
      "umpire.semantic-inventory.known-gap-source.18-observation-known-gap-admission",
      "umpire.semantic-inventory.known-gap-source.19-result-request-raw-known-gap-carry",
      "umpire.semantic-inventory.known-gap-source.20-result-observation-known-gap-carry",
      "umpire.semantic-inventory.known-gap-source.21-test-capability",
      "umpire.semantic-inventory.known-gap-source.22-test-input",
      "umpire.semantic-inventory.known-gap-source.23-test-interpretation",
      "umpire.semantic-inventory.known-gap-source.24-test-claim-reference"
  ] := by
  native_decide

example :
    let invalid := { requestRawKnownGapInputCatalogRow with owner := "" }
    catalogValidationErrorKindFor? invalid
      (Umpire.SemanticInventory.knownGapCatalog invalid) = some .invalidSource := by
  native_decide

example :
    let admitted := catalogAt 16
    admitted.shape = .admittedKnownGapInput ∧
      admitted.lineage = .carried ∧
      admitted.scope = .production ∧
      admitted.fieldMapping = none ∧
    let projection := catalogAt 17
    projection.shape = .evidenceGapAdmissionProjection ∧
      projection.lineage = .carried ∧
      projection.scope = .production ∧
      projection.source = admitted.id ∧
      projection.fieldMapping = some .observationAdmission ∧
    let requestCarry := catalogAt 18
    requestCarry.shape = .carriedCatalogEntry ∧
      requestCarry.lineage = .carried ∧
      requestCarry.scope = .production ∧
      requestCarry.source = admitted.id ∧
      requestCarry.source != projection.id ∧
      requestCarry.fieldMapping = some .exact ∧
    let observationCarry := catalogAt 19
    observationCarry.shape = .carriedCatalogEntry ∧
      observationCarry.lineage = .carried ∧
      observationCarry.scope = .production ∧
      observationCarry.source = (catalogAt 8).id ∧
      observationCarry.source != projection.id ∧
      observationCarry.fieldMapping = some .exact := by
  native_decide

example :
    (knownGapCatalog.drop 20).all (fun row => row.scope == .testOnly) ∧
      let claimReference := catalogAt 23
      claimReference.shape = .carriedCatalogEntry ∧
        claimReference.lineage = .carried ∧
        claimReference.source = (catalogAt 7).id ∧
        claimReference.fieldMapping = some .exact := by
  native_decide

example :
    let projection := catalogAt 17
    let invalid := { projection with fieldMapping := some .exact }
    catalogValidationErrorKind? (knownGapCatalog.take 17 ++ invalid :: knownGapCatalog.drop 18) =
      some .invalidProjectionMapping := by
  native_decide

example :
    let projection := catalogAt 17
    let carry := catalogAt 18
    let missing := "umpire.semantic-inventory.known-gap-source.missing"
    [
      catalogValidationErrorKind?
        (knownGapCatalog.take 17 ++ { projection with source := missing } ::
          knownGapCatalog.drop 18),
      catalogValidationErrorKind?
        (knownGapCatalog.take 18 ++ { carry with source := missing } ::
          knownGapCatalog.drop 19)
    ] = [some .unresolvedCarry, some .unresolvedCarry] := by
  native_decide

example :
    let family := catalogAt 9
    let duplicate := {
      family with id := "umpire.semantic-inventory.known-gap-source.25-duplicate-setup"
    }
    catalogValidationErrorKind? (knownGapCatalog ++ [duplicate]) = some .duplicateOwnership := by
  native_decide

example :
    let family := catalogAt 9
    let wrongLineage := { family with lineage := .carried }
    let wrongScope := { family with scope := .testOnly }
    catalogValidationErrorKind? (knownGapCatalog.take 9 ++ wrongLineage :: knownGapCatalog.drop 10) =
        some .invalidLineage ∧
      catalogValidationErrorKind? (knownGapCatalog.take 9 ++ wrongScope :: knownGapCatalog.drop 10) =
        some .invalidScope := by
  native_decide

example :
    let fixture := catalogAt 20
    let wrongScope := { fixture with scope := .production }
    let duplicateCode := {
      (catalogAt 0) with id := "umpire.semantic-inventory.known-gap-source.25-duplicate-code"
    }
    catalogValidationErrorKind?
        (knownGapCatalog.take 20 ++ wrongScope :: knownGapCatalog.drop 21) =
        some .invalidScope ∧
      catalogValidationErrorKind? (knownGapCatalog ++ [duplicateCode]) =
        some .duplicateCode := by
  native_decide

end Umpire.SemanticInventoryTests.KnownGaps
