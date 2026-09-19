import Umpire.Query.Tests.Fixtures

/-! Checked authored Known Gaps remain attached to Queries without affecting semantic identity. -/

namespace Umpire.QueryTests

open Umpire

private def authoredGap : KnownGap := {
  kind := .input
  code := id "query.known-gap.runtime-input"
  subject := some (id "query.declaration.fixture")
  detail := some "Runtime input is outside model-only Query evaluation."
}

private def authoredGaps : KnownGapSet :=
  (KnownGapSet.checkCanonical [authoredGap]).toOption.get (by native_decide)

private def declarationWithAuthoredGaps : Query := {
  declaration (.find Property.checked) with authoredKnownGaps := authoredGaps
}

private def checkedWithAuthoredGaps : CheckedQuery (fun _ => True) :=
  (Query.check context declarationWithAuthoredGaps).toOption.get (by native_decide)

private def keyedWithAuthoredGaps : Query := {
  id := (DefinitionFamily.mk (id "query.fixture")).id "query" "authored-gaps"
  source
  target := target.id
  form := .find Property.checked
  behavior := Scenario.checked
  limits := Limits.bounded 1 1 8
  policy := searchPolicy
  authoredKnownGaps := authoredGaps
}

example : declarationWithAuthoredGaps.authoredKnownGaps = authoredGaps ∧
    checkedWithAuthoredGaps.authoredKnownGaps = authoredGaps ∧
    keyedWithAuthoredGaps.authoredKnownGaps = authoredGaps ∧
    (Query.check (.ofTarget target) keyedWithAuthoredGaps).toOption.map
        CheckedQuery.authoredKnownGaps = some authoredGaps := by
  native_decide

example :
    let emptyDeclaration := declaration (.find Property.checked)
    let explicitEmpty := { emptyDeclaration with authoredKnownGaps := KnownGapSet.empty }
    let checkedMetadata (owner : Query) :=
      (Query.check context owner).toOption.map CheckedQuery.canonicalMetadata
    let checkedFingerprint (owner : Query) :=
      (Query.check context owner).toOption.map CheckedQuery.behaviorFingerprint
    checkedMetadata emptyDeclaration = checkedMetadata explicitEmpty ∧
      checkedFingerprint emptyDeclaration = checkedFingerprint explicitEmpty ∧
      checkedMetadata declarationWithAuthoredGaps = checkedMetadata emptyDeclaration ∧
      checkedFingerprint declarationWithAuthoredGaps = checkedFingerprint emptyDeclaration := by
  native_decide

example :
    ({ checkedWithAuthoredGaps with documentation := "updated documentation" }).authoredKnownGaps =
      authoredGaps := by
  native_decide

end Umpire.QueryTests
