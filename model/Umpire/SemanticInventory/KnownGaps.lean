import Umpire.Artifact.Types
import Umpire.Observation.Evaluation
import Umpire.SemanticInventory.Types

/-!
The production Known Gap source registry names fixed planner gaps and the closed Observation
diagnostic family. Typed exact and prefix sources let producers reuse the registry declarations
without turning request-provided codes into semantic definitions.
-/

namespace Umpire
namespace SemanticInventory

/-- A closed production source is either one exact Known Gap or one generated namespaced family. -/
inductive KnownGapSource where
  | exact (gap : KnownGap)
  | namespacedPrefix (kind : KnownGapKind) (namespaceId : DefinitionId)
  deriving BEq, DecidableEq, Repr

namespace KnownGapSource

/-- The exact code or family prefix that identifies a source. -/
def codeNamespace : KnownGapSource → DefinitionId
  | .exact gap => gap.code
  | .namespacedPrefix _ namespaceId => namespaceId

/-- Whether a Known Gap belongs to this closed source. -/
def covers : KnownGapSource → KnownGap → Bool
  | .exact expected, candidate => expected == candidate
  | .namespacedPrefix kind namespaceId, candidate =>
      candidate.kind == kind && candidate.code.value.startsWith (namespaceId.value ++ ".")

private def label : KnownGapSource → String
  | .exact gap => gap.code.value
  | .namespacedPrefix _ namespaceId => namespaceId.value ++ ".*"

private def shape : KnownGapSource → KnownGapSourceShape
  | .exact _ => .exactKnownGap
  | .namespacedPrefix _ _ => .generatedKnownGapFamily

private def expectedLineage : KnownGapSource → KnownGapLineage
  | .exact _ => .authored
  | .namespacedPrefix _ _ => .synthesized

end KnownGapSource

/-- One typed production Known Gap source and its stable inventory metadata. -/
structure KnownGapSourceDescriptor where
  id : DefinitionId
  owner : String
  lineage : KnownGapLineage
  scope : KnownGapScope
  source : KnownGapSource
  description : String
  deriving BEq, DecidableEq, Repr

namespace KnownGapSourceDescriptor

/-- Erase a typed production source to the shared semantic-inventory row. -/
def catalog (descriptor : KnownGapSourceDescriptor) : KnownGapCatalogDescriptor := {
  id := descriptor.id.value
  owner := descriptor.owner
  lineage := descriptor.lineage
  scope := descriptor.scope
  shape := descriptor.source.shape
  source := descriptor.source.label
  fieldMapping := none
  description := descriptor.description
}

end KnownGapSourceDescriptor

/-- Atomic validation failures for a production Known Gap source list. -/
inductive KnownGapSourceErrorKind where
  | duplicateId
  | duplicateCode
  | invalidId
  | invalidCode
  | invalidPrefix
  | invalidLineage
  | invalidScope
  | noncanonicalOrder
  deriving BEq, DecidableEq, Ord, Repr

/-- The first source that violates a production Known Gap catalog invariant. -/
structure KnownGapSourceError where
  kind : KnownGapSourceErrorKind
  id : DefinitionId
  codeNamespace : DefinitionId
  deriving BEq, DecidableEq, Repr

private def sourceError
    (kind : KnownGapSourceErrorKind)
    (descriptor : KnownGapSourceDescriptor) : KnownGapSourceError := {
  kind
  id := descriptor.id
  codeNamespace := descriptor.source.codeNamespace
}

private def firstDuplicateId
    (seen : List DefinitionId) : List KnownGapSourceDescriptor → Option KnownGapSourceDescriptor
  | [] => none
  | descriptor :: rest =>
      if seen.contains descriptor.id then some descriptor
      else firstDuplicateId (descriptor.id :: seen) rest

private def firstDuplicateCode
    (seen : List DefinitionId) : List KnownGapSourceDescriptor → Option KnownGapSourceDescriptor
  | [] => none
  | descriptor :: rest =>
      if seen.contains descriptor.source.codeNamespace then some descriptor
      else firstDuplicateCode (descriptor.source.codeNamespace :: seen) rest

private def sourceLe
    (left right : KnownGapSourceDescriptor) : Bool :=
  decide (left.id.value ≤ right.id.value)

/-- Validate a complete production source list without returning a partial catalog. -/
def validateProductionKnownGapSources
    (sources : List KnownGapSourceDescriptor) : Except KnownGapSourceError Unit := do
  for descriptor in sources do
    if !descriptor.id.isNamespaced then
      throw (sourceError .invalidId descriptor)
    if descriptor.scope != .production then
      throw (sourceError .invalidScope descriptor)
    if descriptor.lineage != descriptor.source.expectedLineage then
      throw (sourceError .invalidLineage descriptor)
    match descriptor.source with
    | .exact gap =>
        if !gap.code.isNamespaced || gap.subject.any (fun subject => !subject.isNamespaced) then
          throw (sourceError .invalidCode descriptor)
    | .namespacedPrefix _ namespaceId =>
        if !namespaceId.isNamespaced then
          throw (sourceError .invalidPrefix descriptor)
  match firstDuplicateId [] sources with
  | some descriptor => throw (sourceError .duplicateId descriptor)
  | none => pure ()
  match firstDuplicateCode [] sources with
  | some descriptor => throw (sourceError .duplicateCode descriptor)
  | none => pure ()
  if sources.mergeSort sourceLe != sources then
    let descriptor := sources.getD 0 {
      id := DefinitionId.of "umpire.semantic-inventory.known-gap-source.unknown"
      owner := "Umpire.SemanticInventory"
      lineage := .authored
      scope := .production
      source := .exact plannerExecutionEvidenceKnownGap
      description := "Unknown production source."
    }
    throw (sourceError .noncanonicalOrder descriptor)

private def plannerSource
    (ordinal slug : String)
    (gap : KnownGap)
    (description : String) : KnownGapSourceDescriptor := {
  id := DefinitionId.of ("umpire.semantic-inventory.known-gap-source." ++ ordinal ++ "-" ++ slug)
  owner := "Umpire.Artifact"
  lineage := .authored
  scope := .production
  source := .exact gap
  description
}

/-- The eight fixed planner sources in their unchanged canonical Known Gap order. -/
def plannerKnownGapSources : List KnownGapSourceDescriptor := [
  plannerSource "01" "execution-evidence" plannerExecutionEvidenceKnownGap
    "Execution Evidence is unavailable during pure planning.",
  plannerSource "02" "artifact-migrations" plannerArtifactMigrationsKnownGap
    "Artifact migration interpretation is outside pure planning.",
  plannerSource "03" "artifact-reading" plannerArtifactReadingKnownGap
    "Persisted Artifact reading is outside pure planning.",
  plannerSource "04" "evidence-evaluation" plannerEvidenceEvaluationKnownGap
    "Runtime Evidence evaluation is outside pure planning.",
  plannerSource "05" "runtime-scheduler-order" plannerRuntimeSchedulerOrderKnownGap
    "Runtime scheduler ordering is unavailable during pure planning.",
  plannerSource "06" "runtime-storage-order" plannerRuntimeStorageOrderKnownGap
    "Runtime storage ordering is unavailable during pure planning.",
  plannerSource "07" "runtime-transport-order" plannerRuntimeTransportOrderKnownGap
    "Runtime transport ordering is unavailable during pure planning.",
  plannerSource "08" "promotion" plannerPromotionKnownGap
    "Promotion is not established by pure planning."
]

/-- The generated Observation diagnostic family emitted by Run Evaluation. -/
def observationKnownGapSource : KnownGapSourceDescriptor := {
  id := DefinitionId.of "umpire.semantic-inventory.known-gap-source.09-observation-diagnostic"
  owner := "Umpire.Observation"
  lineage := .synthesized
  scope := .production
  source := .namespacedPrefix .interpretation (DefinitionId.of "umpire.observation")
  description := "A closed Observation diagnostic synthesized during Run Evaluation."
}

/-- All fixed and synthesized production sources in canonical inventory order. -/
def productionKnownGapSources : List KnownGapSourceDescriptor :=
  plannerKnownGapSources ++ [observationKnownGapSource]

/-- Erased production rows consumed by the semantic-inventory renderer. -/
def productionKnownGapCatalog : List KnownGapCatalogDescriptor :=
  productionKnownGapSources.map KnownGapSourceDescriptor.catalog

/-- Stable suffixes for the closed Observation diagnostic family. -/
def observationFailureKnownGapSuffix : ObservationFailureKind → String
  | .emptyEvidence => "empty-evidence"
  | .evidenceBoundExhausted => "evidence-bound-exhausted"
  | .knownGap => "known-gap"
  | .missingInitialState => "missing-initial-state"
  | .missingClosure => "missing-closure"
  | .sequenceGap => "sequence-gap"
  | .missingCausalParent => "missing-causal-parent"
  | .normalizationFailure => "normalization-failure"
  | .unresolvedBinding => "unresolved-binding"
  | .incomparableOrdering => "incomparable-ordering"
  | .profileMismatch => "profile-mismatch"
  | .profileVersionMismatch => "profile-version-mismatch"
  | .kindMismatch => "kind-mismatch"
  | .fieldMismatch => "field-mismatch"
  | .duplicateEvidenceIdentity => "duplicate-evidence-identity"
  | .contradictoryFact => "contradictory-fact"
  | .contradictoryBinding => "contradictory-binding"
  | .contradictoryOrder => "contradictory-order"
  | .misdirectedFaultReceipt => "misdirected-fault-receipt"
  | .compatibleAlternatives => "compatible-alternatives"
  | .zeroUsableInterpretations => "zero-usable-interpretations"
  | .absentModelCoordinate => "absent-model-coordinate"
  | .duplicateModelCoordinate => "duplicate-model-coordinate"
  | .extraModelCoordinate => "extra-model-coordinate"
  | .inconsistentEvidenceLink => "inconsistent-evidence-link"
  | .unconsumedReference => "unconsumed-reference"
  | .missingClosureSupport => "missing-closure-support"
  | .missingOrderSupport => "missing-order-support"
  | .rawValueLeakage => "raw-value-leakage"
  | .redactedValueLeakage => "redacted-value-leakage"
  | .rejectedValueLeakage => "rejected-value-leakage"
  | .rejectedFieldPresent => "rejected-field-present"
  | .digestPolicyMismatch => "digest-policy-mismatch"
  | .digestCollision => "digest-collision"
  | .disallowedRawMaterial => "disallowed-raw-material"

/-- Materialize one Known Gap from the catalog-owned Observation family. -/
def observationKnownGap
    (kind : ObservationFailureKind)
    (subject : DefinitionId) : KnownGap := {
  kind := .interpretation
  code := DefinitionId.of (observationKnownGapSource.source.codeNamespace.value ++ "." ++
    observationFailureKnownGapSuffix kind)
  subject := some subject
}

end SemanticInventory
end Umpire
