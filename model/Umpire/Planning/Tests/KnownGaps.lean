import Umpire.Planning.Types

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

end Umpire.PlanningTests.KnownGaps
