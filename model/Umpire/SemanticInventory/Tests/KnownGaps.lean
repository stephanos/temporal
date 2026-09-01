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
    | .namespacedPrefix _ _ => none

private def sourceAt (index : Nat) : KnownGapSourceDescriptor :=
  productionKnownGapSources[index]?.getD observationKnownGapSource

example : exactGaps plannerKnownGapSources = canonicalPlannerKnownGaps := by
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
      (DefinitionId.of "umpire.observation") ∧
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
      source := .namespacedPrefix .interpretation (DefinitionId.of "observation") }
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
    let requestGap : KnownGap := {
      kind := .input
      code := DefinitionId.of "temporal.request.raw-unknown"
    }
    productionKnownGapSources.all (fun descriptor => !descriptor.source.covers requestGap) := by
  native_decide

end Umpire.SemanticInventoryTests.KnownGaps
