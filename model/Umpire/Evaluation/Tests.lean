import Umpire.Evaluation

/-! Declaration checks, rendering and identity of Evaluation Profiles, over a Temporal-free
Profile. -/

namespace Umpire.Evaluation.Tests

open Umpire Umpire.Evaluation

private def valid : Declaration := {
  name := "example-2"
  claim := "the example holds"
  trust := "example-trust"
  blockingGaps := [.interpretation, .capability]
  reasons := [
    { name := "violated", condition := .verdictViolated, decision := .rejected },
    { name := "gap", condition := .knownGapBlocking, decision := .incomplete }
  ]
}

private def errorOf (declaration : Declaration) : Option ProfileError :=
  match Profile.declare declaration with
  | .ok _ => none
  | .error error => some error

private def renderOf (declaration : Declaration) : Option String :=
  (Profile.declare declaration).toOption.map Profile.render

#guard errorOf valid == none

-- Each check rejects by name.
#guard errorOf { valid with name := "" } == some (.invalidName "")
#guard errorOf { valid with name := "Local" } == some (.invalidName "Local")
#guard errorOf { valid with name := "-local" } == some (.invalidName "-local")
#guard errorOf { valid with name := "local-" } == some (.invalidName "local-")
#guard errorOf { valid with name := "a/b" } == some (.invalidName "a/b")
#guard errorOf { valid with claim := "" } == some .emptyClaim
#guard errorOf { valid with trust := "" } == some .emptyTrust
#guard errorOf { valid with blockingGaps := [], reasons := [] } == some .emptyTable
#guard errorOf { valid with reasons := valid.reasons ++
    [{ name := "", condition := .cleanupUnclosed, decision := .incomplete }] } ==
  some .emptyReasonName
#guard errorOf { valid with reasons := valid.reasons ++
    [{ name := "violated", condition := .cleanupUnclosed, decision := .incomplete }] } ==
  some (.duplicateReason "violated")
#guard errorOf { valid with reasons := valid.reasons ++
    [{ name := "again", condition := .verdictViolated, decision := .incomplete }] } ==
  some (.repeatedCondition .verdictViolated)
#guard errorOf { valid with blockingGaps := [.capability, .capability] } ==
  some (.duplicateBlockingKind .capability)
#guard errorOf { valid with blockingGaps := [] } == some .blockingWithoutKinds
#guard errorOf { valid with reasons := valid.reasons.take 1 } == some .kindsWithoutBlocking
#guard errorOf { valid with blockingGaps := [], reasons := valid.reasons.take 1 } == none

#guard (ProfileError.repeatedCondition .unsupportedRule).render =
  "condition 'unsupported-rule' is named by two reasons"

-- The canonical bytes: fields in a fixed order, blocking kinds in the kind order whatever order
-- they were declared in, one trailing newline.
#guard renderOf valid == some
  ("{\"version\":1,\"name\":\"example-2\",\"claim\":\"the example holds\"," ++
    "\"trust\":\"example-trust\",\"blockingKnownGaps\":[\"capability\",\"interpretation\"]," ++
    "\"reasons\":[{\"name\":\"violated\",\"condition\":\"verdict-violated\"," ++
    "\"decision\":\"rejected\"},{\"name\":\"gap\",\"condition\":\"known-gap-blocking\"," ++
    "\"decision\":\"incomplete\"}]}\n")
#guard renderOf { valid with blockingGaps := [.capability, .interpretation] } == renderOf valid

-- The identity is the bytes' digest: the same bytes, the same identity; other bytes, another.
#guard (Profile.declare valid).toOption.map Profile.identity ==
  (renderOf valid).map fun bytes => "sha256:" ++ Fingerprint.sha256Hex bytes
#guard (Profile.declare valid).toOption.map Profile.identity !=
  (Profile.declare { valid with claim := "another claim" }).toOption.map Profile.identity

end Umpire.Evaluation.Tests
