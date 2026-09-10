import Umpire.Artifact.RunRecord
import Umpire.ImplementationLink.Application
import Umpire.Evidence.PropertyStatus
import Umpire.Search
import Umpire.Inventory.KnownGaps

/-!
Validation and canonical Markdown rendering for Umpire's semantic inventory.

The inventory aggregates only typed descriptors published by their owning modules. Validation
normalizes outer catalog order while retaining each owner's constructor order, and rendering starts
only after the complete aggregate has passed validation.
-/

namespace Temporal.Tool.Inventory

open Umpire

/-- The typed catalogs required to render one semantic inventory. -/
structure Inventory where
  outcomeFamilies : List OutcomeFamilyDescriptor
  notRunMarkers : List NotRunMarker
  knownGaps : List KnownGapCatalogDescriptor
  deriving BEq, DecidableEq, Repr

private def outcomeFamily
    (id owner description : String)
    (classifiers : List (OutcomeConstructorClassifier Outcome)) : OutcomeFamilyDescriptor := {
  id
  owner
  description
  constructors := OutcomeConstructorClassifiers.descriptors classifiers
}

/-- Canonical owner-published outcome families, independent of declaration order. -/
def outcomeFamilies : List OutcomeFamilyDescriptor := [
  outcomeFamily "umpire.semantic-inventory.outcome-family.01-planning" "Umpire.PlanningOutcome"
    "Bounded planning outcomes." PlanningOutcome.constructorClassifiers,
  outcomeFamily "umpire.semantic-inventory.outcome-family.02-execution-phase"
    "Umpire.Artifact.PhaseOutcomeStatus" "Execution phase outcomes."
    PhaseOutcomeStatus.constructorClassifiers,
  outcomeFamily "umpire.semantic-inventory.outcome-family.03-control-attempt"
    "Umpire.Artifact.ControlAttemptStatus" "Requested control-attempt outcomes."
    ControlAttemptStatus.constructorClassifiers,
  outcomeFamily "umpire.semantic-inventory.outcome-family.04-source-closure"
    "Umpire.Artifact.SourceClosureStatus" "Raw Evidence source-closure outcomes."
    SourceClosureStatus.constructorClassifiers,
  outcomeFamily "umpire.semantic-inventory.outcome-family.05-cleanup"
    "Umpire.Artifact.CleanupStatus" "Run cleanup outcomes."
    CleanupStatus.constructorClassifiers,
  outcomeFamily "umpire.semantic-inventory.outcome-family.06-operational"
    "Umpire.Artifact.OperationalStatus" "Overall operational outcomes."
    OperationalStatus.constructorClassifiers,
  outcomeFamily "umpire.semantic-inventory.outcome-family.07-observation"
    "Umpire.ObservationStatus" "Observation Evaluation outcomes."
    ObservationStatus.constructorClassifiers,
  outcomeFamily "umpire.semantic-inventory.outcome-family.08-implementation-link"
    "Umpire.ImplementationLinkStatus" "Implementation Link application outcomes."
    ImplementationLinkStatus.constructorClassifiers,
  outcomeFamily "umpire.semantic-inventory.outcome-family.09-semantic-property"
    "Umpire.Evidence.PropertyStatus" "Semantic Property evaluation outcomes."
    Evidence.PropertyStatus.constructorClassifiers,
  outcomeFamily "umpire.semantic-inventory.outcome-family.10-strict-query"
    "Umpire.QueryStatus" "Strict Query projection outcomes."
    QueryStatus.constructorClassifiers
]

/-- Canonical projection-only values that are not constructors of their owning outcome type. -/
def notRunMarkers : List NotRunMarker := [
  ImplementationLinkStatus.stageNotRunMarker
]

/-- The non-semantic Case input carrying arbitrary validated Known Gaps. -/
def caseKnownGapInputCatalogRow : KnownGapCatalogDescriptor := {
  id := "umpire.semantic-inventory.known-gap-source.17-request-raw-known-gap-input"
  owner := "Umpire.Case"
  lineage := .carried
  scope := .production
  shape := .admittedKnownGapInput
  source := "umpire.case.known-gap-input"
  fieldMapping := none
  description := "Validated Case Known Gaps before stage-specific projection."
}

/-- The complete canonical Known Gap catalog assembled from its owner-published descriptors. -/
def knownGapCatalog : List KnownGapCatalogDescriptor :=
  Umpire.Inventory.knownGapCatalog caseKnownGapInputCatalogRow

/-- The repository's complete typed semantic inventory. -/
def currentInventory : Inventory := {
  outcomeFamilies
  notRunMarkers
  knownGaps := knownGapCatalog
}

/-- Atomic aggregate-validation failures. -/
inductive InventoryErrorKind where
  | invalidOutcomeFamily
  | invalidNotRunMarker
  | invalidKnownGapCatalog
  deriving BEq, DecidableEq, Repr

/-- One stable diagnostic emitted only after aggregate validation fails. -/
structure InventoryError where
  kind : InventoryErrorKind
  detail : String
  deriving BEq, DecidableEq, Repr

def InventoryError.message (failure : InventoryError) : String :=
  match failure.kind with
  | .invalidOutcomeFamily => "invalid outcome family: " ++ failure.detail
  | .invalidNotRunMarker => "invalid stage not-run marker: " ++ failure.detail
  | .invalidKnownGapCatalog => "invalid Known Gap catalog: " ++ failure.detail

instance : ToString InventoryError where
  toString := InventoryError.message

private def familyLe (left right : OutcomeFamilyDescriptor) : Bool :=
  decide (left.id ≤ right.id)

private def notRunMarkerLe (left right : NotRunMarker) : Bool :=
  decide (left.id ≤ right.id)

private def knownGapLe (left right : KnownGapCatalogDescriptor) : Bool :=
  decide (left.id ≤ right.id)

private def normalize (inventory : Inventory) : Inventory := {
  outcomeFamilies := inventory.outcomeFamilies.mergeSort familyLe
  notRunMarkers := inventory.notRunMarkers.mergeSort notRunMarkerLe
  knownGaps := inventory.knownGaps.mergeSort knownGapLe
}

private def outcomeFamiliesAreValid (families : List OutcomeFamilyDescriptor) : Bool :=
  families == outcomeFamilies &&
    (families.map OutcomeFamilyDescriptor.id).Nodup &&
    families.all fun family =>
      (DefinitionId.of family.id).isNamespaced && !family.owner.trimAscii.isEmpty &&
        !family.description.trimAscii.isEmpty && !family.constructors.isEmpty &&
        (family.constructors.map OutcomeConstructorDescriptor.name).Nodup &&
        family.constructors.all fun constructor =>
          !constructor.name.trimAscii.isEmpty && !constructor.description.trimAscii.isEmpty

private def notRunMarkersAreValid
    (markers : List NotRunMarker)
    (families : List OutcomeFamilyDescriptor) : Bool :=
  markers == notRunMarkers &&
    (markers.map NotRunMarker.id).Nodup &&
    markers.all fun marker =>
      (DefinitionId.of marker.id).isNamespaced && !marker.owner.trimAscii.isEmpty &&
        !marker.name.trimAscii.isEmpty && !marker.description.trimAscii.isEmpty &&
        families.all fun family =>
          family.owner != marker.owner ||
            !(family.constructors.map OutcomeConstructorDescriptor.name).contains marker.name

/-- Validate the complete aggregate and return its canonical outer order. -/
def validate (inventory : Inventory) : Except InventoryError Inventory := do
  let canonical := normalize inventory
  unless outcomeFamiliesAreValid canonical.outcomeFamilies do
    throw { kind := .invalidOutcomeFamily, detail := "catalog or owner-local order drift" }
  unless notRunMarkersAreValid canonical.notRunMarkers canonical.outcomeFamilies do
    throw { kind := .invalidNotRunMarker, detail := "catalog or owning family drift" }
  match Umpire.Inventory.validateKnownGapCatalog
      caseKnownGapInputCatalogRow canonical.knownGaps with
  | .error failure =>
      throw {
        kind := .invalidKnownGapCatalog
        detail := s!"{repr failure.kind} at {failure.id}"
      }
  | .ok () => pure ()
  unless canonical.knownGaps == knownGapCatalog do
    throw { kind := .invalidKnownGapCatalog, detail := "catalog membership drift" }
  pure canonical

private def markdownCell (value : String) : String :=
  value.replace "\\" "\\\\" |>.replace "|" "\\|" |>.replace "\n" " "

private def outcomeFamilyLines (family : OutcomeFamilyDescriptor) : List String :=
  [
    s!"### `{family.id}`",
    "",
    s!"Owner: `{family.owner}`",
    "",
    family.description,
    "",
    "| Outcome | Meaning |",
    "| --- | --- |"
  ] ++ family.constructors.map fun constructor =>
    s!"| `{markdownCell constructor.name}` | {markdownCell constructor.description} |"

private def notRunMarkerLines (markers : List NotRunMarker) : List String :=
  [
    "## Stage not-run markers",
    "",
    "These rendered values represent an unevaluated projection; they are not outcome constructors.",
    "",
    "| ID | Owner | Value | Meaning |",
    "| --- | --- | --- | --- |"
  ] ++ markers.map fun marker =>
    s!"| `{markdownCell marker.id}` | `{markdownCell marker.owner}` | " ++
      s!"`{markdownCell marker.name}` | {markdownCell marker.description} |"

private def knownGapLines (catalog : List KnownGapCatalogDescriptor) : List String :=
  [
    "## Known Gap flows",
    "",
    "Each row identifies one authored source, synthesized family, projection, exact carry, or test-only reference.",
    "",
    "| Catalog ID | Owner | Lineage | Scope | Shape | Source/reference | Field mapping | Description |",
    "| --- | --- | --- | --- | --- | --- | --- | --- |"
  ] ++ catalog.map fun row =>
    let mapping := row.fieldMapping.map KnownGapCarryMapping.name |>.getD "—"
    s!"| `{markdownCell row.id}` | `{markdownCell row.owner}` | {row.lineage.name} | " ++
      s!"{row.scope.name} | {row.shape.name} | `{markdownCell row.source}` | " ++
      s!"{markdownCell mapping} | {markdownCell row.description} |"

/-- Render a previously validated inventory as canonical Markdown with one terminal line feed. -/
def render (inventory : Inventory) : String :=
  let familyLines := inventory.outcomeFamilies.flatMap fun family =>
    outcomeFamilyLines family ++ [""]
  String.intercalate "\n" <| [
    "# Umpire semantic inventory",
    "",
    "> Generated from typed Lean catalogs. Do not edit semantic meaning in this file.",
    "",
    "## Outcome families",
    ""
  ] ++ familyLines ++ notRunMarkerLines inventory.notRunMarkers ++ [""] ++
    knownGapLines inventory.knownGaps ++ [""]

/-- Validate the full aggregate before exposing any Markdown bytes. -/
def validateAndRender (inventory : Inventory) : Except InventoryError String := do
  pure <| render (← validate inventory)

/-- Validate and buffer the complete document before performing the final standard-output write. -/
def run
    (inventory : Inventory)
    (writeOutput writeError : String → IO Unit) : IO UInt32 := do
  match validateAndRender inventory with
  | .error failure =>
      writeError (failure.message ++ "\n")
      pure 1
  | .ok document =>
      try
        writeOutput document
        pure 0
      catch failure =>
        try writeError ("semantic inventory write failed: " ++ failure.toString ++ "\n")
        catch _ => pure ()
        pure 1

end Temporal.Tool.Inventory
