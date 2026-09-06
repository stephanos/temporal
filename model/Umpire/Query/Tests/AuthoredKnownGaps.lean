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

private def declarationWithAuthoredGaps : QueryDeclaration := {
  declaration (.witness checkedProperty) with authoredKnownGaps := authoredGaps
}

private def checkedWithAuthoredGaps : CheckedQuery (fun _ => True) :=
  (checkQuery context declarationWithAuthoredGaps).toOption.get (by native_decide)

private def specWithAuthoredGaps : QuerySpec := {
  family := { root := id "query.fixture" }
  key := "authored-gaps"
  source
  target := target.id
  form := .witness checkedProperty
  behavior := checkedBehavior
  limits := { transitions := 1, selectedActions := 1, candidateEvaluations := 8 }
  policy := searchPolicy
  authoredKnownGaps := authoredGaps
}

example : declarationWithAuthoredGaps.authoredKnownGaps = authoredGaps ∧
    checkedWithAuthoredGaps.authoredKnownGaps = authoredGaps ∧
    specWithAuthoredGaps.declaration.authoredKnownGaps = authoredGaps ∧
    (specWithAuthoredGaps.check target).toOption.map CheckedQuery.authoredKnownGaps =
      some authoredGaps := by
  native_decide

example :
    let emptyDeclaration := declaration (.witness checkedProperty)
    let explicitEmpty := { emptyDeclaration with authoredKnownGaps := KnownGapSet.empty }
    let checkedMetadata (owner : QueryDeclaration) :=
      (checkQuery context owner).toOption.map CheckedQuery.canonicalMetadata
    let checkedFingerprint (owner : QueryDeclaration) :=
      (checkQuery context owner).toOption.map CheckedQuery.behaviorFingerprint
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
