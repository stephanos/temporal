import Umpire.Artifact.Types
import Umpire.Artifact.Result
import Umpire.ImplementationLink.Language
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
  | namespacedPrefix
      (kind : KnownGapKind)
      (namespaceId : DefinitionId)
      (suffixes : List String)
  deriving BEq, DecidableEq, Repr

namespace KnownGapSource

/-- The exact code or family prefix that identifies a source. -/
def codeNamespace : KnownGapSource → DefinitionId
  | .exact gap => gap.code
  | .namespacedPrefix _ namespaceId _ => namespaceId

/-- Whether a Known Gap belongs to this closed source. -/
def covers : KnownGapSource → KnownGap → Bool
  | .exact expected, candidate => expected == candidate
  | .namespacedPrefix kind namespaceId suffixes, candidate =>
      candidate.kind == kind && suffixes.any fun suffix =>
        candidate.code == DefinitionId.of (namespaceId.value ++ "." ++ suffix)

/-- Materialize a gap using this source's catalog-owned kind and code namespace. -/
def materialize
    (source : KnownGapSource)
    (suffix : String)
    (subject : Option DefinitionId) : KnownGap :=
  match source with
  | .exact gap => gap
  | .namespacedPrefix kind namespaceId _ => {
      kind
      code := DefinitionId.of (namespaceId.value ++ "." ++ suffix)
      subject
    }

private def label : KnownGapSource → String
  | .exact gap => gap.code.value
  | .namespacedPrefix _ namespaceId _ => namespaceId.value ++ ".*"

private def shape : KnownGapSource → KnownGapSourceShape
  | .exact _ => .exactKnownGap
  | .namespacedPrefix _ _ _ => .generatedKnownGapFamily

private def expectedLineage : KnownGapSource → KnownGapLineage
  | .exact _ => .authored
  | .namespacedPrefix _ _ _ => .synthesized

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
    | .namespacedPrefix _ namespaceId suffixes =>
        if !namespaceId.isNamespaced || suffixes.isEmpty || !suffixes.Nodup ||
            suffixes.any (fun suffix => suffix.toList.contains '.' ||
              !(DefinitionId.of ("source." ++ suffix)).isNamespaced) then
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

/-- The catalog-owned fixed planner promotion source. -/
def plannerPromotionKnownGapSource : KnownGapSourceDescriptor :=
  plannerSource "08" "promotion" plannerPromotionKnownGap
    "Promotion is not established by pure planning."

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
  plannerPromotionKnownGapSource
]

/-- Closed suffixes emitted by the Observation diagnostic family. -/
def observationKnownGapSuffixes : List String := [
  "empty-evidence",
  "evidence-bound-exhausted",
  "known-gap",
  "missing-initial-state",
  "missing-closure",
  "sequence-gap",
  "missing-causal-parent",
  "normalization-failure",
  "unresolved-binding",
  "incomparable-ordering",
  "profile-mismatch",
  "profile-version-mismatch",
  "kind-mismatch",
  "field-mismatch",
  "duplicate-evidence-identity",
  "contradictory-fact",
  "contradictory-binding",
  "contradictory-order",
  "misdirected-fault-receipt",
  "compatible-alternatives",
  "zero-usable-interpretations",
  "absent-model-coordinate",
  "duplicate-model-coordinate",
  "extra-model-coordinate",
  "inconsistent-evidence-link",
  "unconsumed-reference",
  "missing-closure-support",
  "missing-order-support",
  "raw-value-leakage",
  "redacted-value-leakage",
  "rejected-value-leakage",
  "rejected-field-present",
  "digest-policy-mismatch",
  "digest-collision",
  "disallowed-raw-material"
]

/-- The generated Observation diagnostic family emitted by Run Evaluation. -/
def observationKnownGapSource : KnownGapSourceDescriptor := {
  id := DefinitionId.of "umpire.semantic-inventory.known-gap-source.09-observation-diagnostic"
  owner := "Umpire.Observation"
  lineage := .synthesized
  scope := .production
  source := .namespacedPrefix .interpretation (DefinitionId.of "umpire.observation")
    observationKnownGapSuffixes
  description := "A closed Observation diagnostic synthesized during Run Evaluation."
}

/-- All fixed and synthesized production sources in canonical inventory order. -/
def productionKnownGapSources : List KnownGapSourceDescriptor :=
  plannerKnownGapSources ++ [observationKnownGapSource]

/-- Erased production rows consumed by the semantic-inventory renderer. -/
def productionKnownGapCatalog : List KnownGapCatalogDescriptor :=
  productionKnownGapSources.map KnownGapSourceDescriptor.catalog

private def implementationLinkKnownGapCatalogRow
    (ordinal : String)
    (family : ImplementationLinkKnownGapFamily) : KnownGapCatalogDescriptor := {
  id := "umpire.semantic-inventory.known-gap-source." ++ ordinal ++
    "-implementation-link-" ++ family.name
  owner := "Umpire.ImplementationLink"
  lineage := .authored
  scope := .production
  shape := .authoredImplementationLinkKnownGapFamily
  source := family.name
  fieldMapping := none
  description := "Polymorphic authored " ++ family.name ++
    " Known Gaps retained by an Implementation Link declaration."
}

/-- The seven polymorphic declaration fields that may author Implementation Link Known Gaps. -/
def implementationLinkKnownGapCatalog : List KnownGapCatalogDescriptor := [
  implementationLinkKnownGapCatalogRow "10" .setup,
  implementationLinkKnownGapCatalogRow "11" .state,
  implementationLinkKnownGapCatalogRow "12" .action,
  implementationLinkKnownGapCatalogRow "13" .outcome,
  implementationLinkKnownGapCatalogRow "14" .observation,
  implementationLinkKnownGapCatalogRow "15" .relation,
  implementationLinkKnownGapCatalogRow "16" .capability
]

/-- The non-semantic request and Raw Evidence input carrying arbitrary validated Known Gaps. -/
def requestRawKnownGapInputCatalogRow : KnownGapCatalogDescriptor := {
  id := "umpire.semantic-inventory.known-gap-source.17-request-raw-known-gap-input"
  owner := "Temporal.Tool.RunEvaluation"
  lineage := .carried
  scope := .production
  shape := .admittedKnownGapInput
  source := "umpire.run-evaluation.request-and-raw-known-gap-input"
  fieldMapping := none
  description := "Validated request and Raw Evidence Known Gaps before stage-specific projection."
}

/-- The lossy request and Raw Evidence Known Gap admission into Observation Evaluation. -/
def observationKnownGapAdmissionCatalogRow : KnownGapCatalogDescriptor := {
  id := "umpire.semantic-inventory.known-gap-source.18-observation-known-gap-admission"
  owner := "Umpire.Observation"
  lineage := .carried
  scope := .production
  shape := .evidenceGapAdmissionProjection
  source := requestRawKnownGapInputCatalogRow.id
  fieldMapping := some EvidenceGap.knownGapAdmissionMapping
  description := "Request and Raw Evidence Known Gaps admitted as lossy Evidence Gaps."
}

/-- The exact request and Raw Evidence Known Gap aggregation retained by the Result Artifact. -/
def resultRequestRawKnownGapCarryCatalogRow : KnownGapCatalogDescriptor := {
  id := "umpire.semantic-inventory.known-gap-source.19-result-request-raw-known-gap-carry"
  owner := "Umpire.Artifact.Result"
  lineage := .carried
  scope := .production
  shape := .carriedCatalogEntry
  source := requestRawKnownGapInputCatalogRow.id
  fieldMapping := some ResultArtifact.knownGapCarryMapping
  description := "Request and Raw Evidence Known Gaps carried exactly into Result."
}

/-- The exact synthesized Observation Known Gap aggregation retained by the Result Artifact. -/
def resultObservationKnownGapCarryCatalogRow : KnownGapCatalogDescriptor := {
  id := "umpire.semantic-inventory.known-gap-source.20-result-observation-known-gap-carry"
  owner := "Umpire.Artifact.Result"
  lineage := .carried
  scope := .production
  shape := .carriedCatalogEntry
  source := observationKnownGapSource.id.value
  fieldMapping := some ResultArtifact.knownGapCarryMapping
  description := "Synthesized Observation Known Gaps carried exactly into Result."
}

private def testKnownGapCatalogRow
    (ordinal slug source description : String) : KnownGapCatalogDescriptor := {
  id := "umpire.semantic-inventory.known-gap-source." ++ ordinal ++ "-test-" ++ slug
  owner := "Umpire.PlanningTests.KnownGaps"
  lineage := .authored
  scope := .testOnly
  shape := .exactKnownGap
  source
  fieldMapping := none
  description
}

/-- Test-only Known Gap fixtures, separated from every production source. -/
def testKnownGapCatalog : List KnownGapCatalogDescriptor := [
  testKnownGapCatalogRow "21" "capability" "umpire.known-gap.capability-contract"
    "Test-only capability-contract Known Gap fixture.",
  testKnownGapCatalogRow "22" "input" "umpire.known-gap.runtime-evidence"
    "Test-only input Known Gap fixture.",
  testKnownGapCatalogRow "23" "interpretation" "umpire.known-gap.runtime-order"
    "Test-only interpretation Known Gap fixture.",
  {
    id := "umpire.semantic-inventory.known-gap-source.24-test-claim-reference"
    owner := "Umpire.PlanningTests.KnownGaps"
    lineage := .carried
    scope := .testOnly
    shape := .carriedCatalogEntry
    source := plannerPromotionKnownGapSource.id.value
    fieldMapping := some .exact
    description := "Test-only use of the production planner promotion Known Gap."
  }
]

/-- The complete canonical Known Gap catalog, including explicitly scoped test fixtures. -/
def knownGapCatalog : List KnownGapCatalogDescriptor :=
  productionKnownGapCatalog ++ implementationLinkKnownGapCatalog ++
    [requestRawKnownGapInputCatalogRow, observationKnownGapAdmissionCatalogRow,
      resultRequestRawKnownGapCarryCatalogRow, resultObservationKnownGapCarryCatalogRow] ++
    testKnownGapCatalog

/-- Atomic validation failures for the complete Known Gap catalog. -/
inductive KnownGapCatalogErrorKind where
  | duplicateId
  | duplicateCode
  | invalidId
  | invalidSource
  | invalidLineage
  | invalidScope
  | invalidProjectionMapping
  | unresolvedCarry
  | duplicateOwnership
  | noncanonicalOrder
  deriving BEq, DecidableEq, Ord, Repr

/-- The first complete-catalog row that violates a Known Gap invariant. -/
structure KnownGapCatalogError where
  kind : KnownGapCatalogErrorKind
  id : String
  source : String
  deriving BEq, DecidableEq, Repr

private def catalogError
    (kind : KnownGapCatalogErrorKind)
    (row : KnownGapCatalogDescriptor) : KnownGapCatalogError := {
  kind
  id := row.id
  source := row.source
}

private def catalogExpectedLineage : KnownGapSourceShape → KnownGapLineage
  | .exactKnownGap => .authored
  | .generatedKnownGapFamily => .synthesized
  | .authoredImplementationLinkKnownGapFamily => .authored
  | .admittedKnownGapInput => .carried
  | .evidenceGapAdmissionProjection => .carried
  | .carriedCatalogEntry => .carried

private def productionExactCatalogSources : List String :=
  productionKnownGapCatalog.filterMap fun row =>
    if row.shape == .exactKnownGap then some row.source else none

private def testExactCatalogSources : List String :=
  testKnownGapCatalog.filterMap fun row =>
    if row.shape == .exactKnownGap then some row.source else none

private def catalogScopeIsValid (row : KnownGapCatalogDescriptor) : Bool :=
  let expected := if row.shape == .exactKnownGap &&
      testExactCatalogSources.contains row.source then .testOnly
    else if row.shape == .carriedCatalogEntry &&
      row.owner == "Umpire.PlanningTests.KnownGaps" &&
      row.source == plannerPromotionKnownGapSource.id.value then .testOnly
    else KnownGapScope.production
  row.scope == expected

private def catalogSourceIsValid (row : KnownGapCatalogDescriptor) : Bool :=
  match row.shape with
  | .exactKnownGap =>
      (productionExactCatalogSources ++ testExactCatalogSources).contains row.source
  | .generatedKnownGapFamily => row.source == observationKnownGapSource.source.label
  | .authoredImplementationLinkKnownGapFamily =>
      row.owner == "Umpire.ImplementationLink" &&
        ImplementationLinkKnownGapFamily.all.any fun family => family.name == row.source
  | .admittedKnownGapInput =>
      row.owner == requestRawKnownGapInputCatalogRow.owner &&
        row.source == requestRawKnownGapInputCatalogRow.source
  | .evidenceGapAdmissionProjection =>
      row.owner == observationKnownGapAdmissionCatalogRow.owner &&
        row.source == requestRawKnownGapInputCatalogRow.id
  | .carriedCatalogEntry =>
      (row.owner == "Umpire.Artifact.Result" && [
        requestRawKnownGapInputCatalogRow.id,
        observationKnownGapSource.id.value
      ].contains row.source) ||
      (row.owner == "Umpire.PlanningTests.KnownGaps" &&
        row.source == plannerPromotionKnownGapSource.id.value)

private def catalogMappingIsValid (row : KnownGapCatalogDescriptor) : Bool :=
  match row.shape with
  | .exactKnownGap | .generatedKnownGapFamily |
      .authoredImplementationLinkKnownGapFamily | .admittedKnownGapInput =>
      row.fieldMapping.isNone
  | .evidenceGapAdmissionProjection =>
      row.fieldMapping == some EvidenceGap.knownGapAdmissionMapping
  | .carriedCatalogEntry =>
      row.fieldMapping == some ResultArtifact.knownGapCarryMapping

private def catalogOwnershipKey (row : KnownGapCatalogDescriptor) : String :=
  row.owner ++ ":" ++ row.shape.name ++ ":" ++ row.source

private def validateKnownGapCatalogRows
    (seenIds exactCodes ownershipKeys : List String)
    (previousId : Option String) :
    List KnownGapCatalogDescriptor → Except KnownGapCatalogError Unit
  | [] => pure ()
  | row :: rest => do
      if seenIds.contains row.id then
        throw (catalogError .duplicateId row)
      if !(DefinitionId.of row.id).isNamespaced then
        throw (catalogError .invalidId row)
      if row.lineage != catalogExpectedLineage row.shape then
        throw (catalogError .invalidLineage row)
      if !catalogScopeIsValid row then
        throw (catalogError .invalidScope row)
      if (row.shape == .evidenceGapAdmissionProjection ||
          row.shape == .carriedCatalogEntry) && !seenIds.contains row.source then
        throw (catalogError .unresolvedCarry row)
      if !catalogSourceIsValid row then
        throw (catalogError .invalidSource row)
      if !catalogMappingIsValid row then
        throw (catalogError .invalidProjectionMapping row)
      if row.shape == .exactKnownGap && exactCodes.contains row.source then
        throw (catalogError .duplicateCode row)
      let ownershipKey := catalogOwnershipKey row
      if ownershipKeys.contains ownershipKey then
        throw (catalogError .duplicateOwnership row)
      if previousId.any fun previous => decide (row.id < previous) then
        throw (catalogError .noncanonicalOrder row)
      let exactCodes := if row.shape == .exactKnownGap then row.source :: exactCodes
        else exactCodes
      validateKnownGapCatalogRows (row.id :: seenIds) exactCodes
        (ownershipKey :: ownershipKeys) (some row.id) rest

/-- Validate the complete Known Gap catalog without returning a partially accepted prefix. -/
def validateKnownGapCatalog
    (catalog : List KnownGapCatalogDescriptor) : Except KnownGapCatalogError Unit :=
  validateKnownGapCatalogRows [] [] [] none catalog

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
    (subject : DefinitionId) : KnownGap :=
  observationKnownGapSource.source.materialize (observationFailureKnownGapSuffix kind)
    (some subject)

end SemanticInventory
end Umpire
