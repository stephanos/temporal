import Umpire.SemanticInventory.KnownGaps

/-! Closed Known Gap validation and canonical encoding. -/

namespace Umpire.PlanningTests.KnownGaps

open Umpire

private def id (value : String) : DefinitionId := DefinitionId.of value

private def capabilityGap : KnownGap := {
  kind := .capabilityContract
  code := id "umpire.known-gap.capability-contract"
}

private def inputGap : KnownGap := {
  kind := .input
  code := id "umpire.known-gap.runtime-evidence"
  subject := some (id "umpire.target.fixture")
  detail := some "Runtime Evidence is not available to model-only Planning."
}

private def interpretationGap : KnownGap := {
  kind := .interpretation
  code := id "umpire.known-gap.runtime-order"
}

private def claimGap : KnownGap := {
  kind := .claim
  code := id "umpire.known-gap.promotion"
}

example : ["capability-contract", "input", "interpretation", "claim"].map
    KnownGapKind.parse? =
    [some .capabilityContract, some .input, some .interpretation, some .claim] ∧
    KnownGapKind.parse? "other" = none := by
  native_decide

example : (validateKnownGaps [capabilityGap, inputGap, interpretationGap, claimGap]).isOk = true := by
  native_decide

example : [
    validateKnownGaps [{ capabilityGap with code := id "gap" }],
    validateKnownGaps [{ inputGap with subject := some (id "subject") }],
    validateKnownGaps [capabilityGap, capabilityGap],
    validateKnownGaps [
      { inputGap with detail := some "first" },
      { inputGap with detail := some "second" }
    ],
    validateKnownGaps [inputGap, capabilityGap]
  ].map (fun result => result.toOption) = [none, none, none, none, none] := by
  native_decide

example : canonicalKnownGapJson inputGap =
    "{\"kind\":\"input\",\"code\":\"umpire.known-gap.runtime-evidence\"," ++
      "\"subject\":\"umpire.target.fixture\"," ++
      "\"detail\":\"Runtime Evidence is not available to model-only Planning.\"}" := by
  native_decide

private def catalogRow (catalogId : String) : KnownGapCatalogDescriptor :=
  SemanticInventory.knownGapCatalog.find? (fun row => row.id == catalogId) |>.getD {
    id := "umpire.semantic-inventory.known-gap-source.unknown"
    owner := "Umpire.SemanticInventory"
    lineage := .authored
    scope := .testOnly
    shape := .exactKnownGap
    source := "umpire.known-gap.unknown"
    fieldMapping := none
    description := "Unknown test catalog row."
  }

/-! Private fixtures remain one-way bound to their public catalog source or reference. -/
example : [
    (catalogRow "umpire.semantic-inventory.known-gap-source.21-test-capability").source,
    (catalogRow "umpire.semantic-inventory.known-gap-source.22-test-input").source,
    (catalogRow "umpire.semantic-inventory.known-gap-source.23-test-interpretation").source
  ] = [capabilityGap.code.value, inputGap.code.value, interpretationGap.code.value] ∧
    let claimReference :=
      catalogRow "umpire.semantic-inventory.known-gap-source.24-test-claim-reference"
    claimGap = plannerPromotionKnownGap ∧
      claimReference.scope = .testOnly ∧
      claimReference.shape = .carriedCatalogEntry ∧
      claimReference.source =
        "umpire.semantic-inventory.known-gap-source.08-promotion" ∧
      claimReference.fieldMapping = some .exact := by
  native_decide

end Umpire.PlanningTests.KnownGaps
