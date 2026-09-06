---
satisfies: [R7]
---
# fn-62-make-ordinary-temporal-model-authoring.6 Attach checked authored Known Gaps below the planning boundary

## Description
Separate existing Known Gap vocabulary from its Query-dependent owner and attach an explicit default-empty checked set to authored/checked Queries. `.8` owns phase composition and publication.

**Size:** M
**Files:** existing Planning types/KnownGap tests, lower KnownGap owner, Query language/authoring/facades and affected literal consumers
**Touches:** [model/Umpire/KnownGap.lean, model/Umpire/Planning/Types.lean, model/Umpire/Planning/Tests/KnownGaps.lean, model/Umpire/Query.lean, model/Umpire/Query/**, model/Umpire/Planning/VisibilityTests.lean, model/Umpire/Space/Compiler.lean, model/Umpire/Space/Tests/**, model/Umpire/Promotion.lean, model/Umpire/PromotionTests.lean, model/Umpire/Tests/MigrationCompatibility.lean, model/Temporal/Feature/Nexus/Operations/**]

## Approach
- Move the unchanged `KnownGap`, `KnownGapSet`, errors, and set operations below Query, breaking `Planning.Types -> Query` ownership without adding dependencies back into Planning.
- Preserve existing public namespaces/re-exports and validation semantics. Audit all importers and direct Query constructions before attaching data.
- Add checked authored gaps with explicit empty default; keep them out of behavioral semantic JSON/fingerprints. Wire the existing Query constructor path, including migrated operations, without creating required-gap inference.
- Name the field `authoredKnownGaps : KnownGapSet := KnownGapSet.empty` on QueryDeclaration, CheckedQuery and QuerySpec. QuerySpec.declaration and checkQuery copy it unchanged. Explicitly copy it in Space.Compiler.queryDeclaration from space.baseQuery and Promotion.checkPromotedQuery from baseQuery; inventory every other reconstruction/recheck path. Preserve record-update materializers and prove they retain the set.
- Keep malformed/duplicate/conflicting/noncanonical row checks in the existing set checker. Exact extraction regression precedes new Query attachment tests.

## Investigation targets
**Required:**
- `model/Umpire/Planning/Types.lean:8` — current vocabulary and cycle.
- `model/Umpire/Planning/Tests/KnownGaps.lean:64` — set contracts.
- `model/Umpire/Query/Language.lean:257` — declaration and checked values.
- `model/Umpire/Query/Authoring.lean` — existing constructor.
- `model/Umpire/Planning/VisibilityTests.lean` — compatibility imports.

## Acceptance
- [ ] Existing Known Gap type/error/set contracts and public imports survive the extraction unchanged; no import cycle or extra validation pass appears.
- [ ] Both authored and checked Query values retain the checked authored set; empty remains explicit and valid; all literal/constructor consumers compile.
- [ ] Nonempty authored gaps survive Space-derived Queries, promoted Queries, constructor lowering/checkQuery and target materialization exactly. Tests must compare the full set; compilation with a default-empty field cannot detect attachment loss.
- [ ] Malformed code/subject, duplicates, conflicting detail and noncanonical order retain exact errors; empty and nonempty Query examples preserve behavior/fingerprint and default-empty canonical bytes.
- [ ] `cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Planning.Tests Temporal.Feature.Nexus.OperationsTests` passes; import/axiom/lint checks show no new issues. `.8` is required before publication support is complete.
- [ ] Structural complexity audit compares moved set algorithms byte-for-byte/semantically to baseline and verifies Query attachment copies the checked set without revalidation or traversal; preserve the existing set complexity rather than claiming it is linear.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
