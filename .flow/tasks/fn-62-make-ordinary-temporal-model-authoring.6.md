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
Moved the existing Known Gap vocabulary, errors, validation, canonical encoding, and set operations unchanged from `Umpire.Planning.Types` to the lower `Umpire.KnownGap` owner. Added explicit default-empty `authoredKnownGaps` fields to `QueryDeclaration`, `CheckedQuery`, and `QuerySpec`, copying the checked set unchanged through `QuerySpec.declaration`, `checkQuery`, Space-derived Query declarations, Promotion rechecking, and record-update materialization while keeping it outside Query semantic JSON and fingerprints.

Nonempty full-set regressions cover constructor lowering/checking, Space lowering, Promotion rechecking, and record updates; established Nexus operation Queries remain empty and retain the existing exact artifact compatibility. Existing Known Gap tests retain exact invalid-code, invalid-subject, duplicate, conflicting-detail, and noncanonical-order payloads. Public Query and aggregate imports expose the moved types without a Planning cycle.

Structural complexity audit: the complete moved implementation block from `KnownGapKind` through `canonicalKnownGapJson` is byte-identical to task-start staged tree `734d38c18c7b1ede52a6797543ba894d9d9d091d`. The attachment adds only record fields and direct field copies, with no list traversal, validation, normalization, or checker call. Existing validation and set-operation complexity remains unchanged: identifier validation traverses rows, canonical validation scans adjacent rows, and canonicalization/union retain their existing `mergeSort` and `eraseDups` work; no linear-complexity claim is made. Task `.8` still owns authored/phase union, planner `Except` APIs, artifact publication, Case conversion, and higher-level error mapping.

Verification: baseline parent Quick passed 128/128; final task Quick passed 78/78; focused Space/Promotion and public-import builds passed; `make lint-model` passed 260/260. `make lint-code GOLANGCI_LINT_FIX=false` retained the inherited exit 2 with exactly 1,316 normalized diagnostic headers, byte-identical to the approved baseline at SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`. The official staged-overlay review returned SHIP with no findings, and reviewed/final staged source tree `3327e488ccba6b7b285f2be85803ffb9e2078dad` is unchanged.

No commit was created under the user's standing commit policy; HEAD remains `8f40772ebb8e56fac4d068c9990b3c1827f8a05f` and cumulative/unrelated staged work remains preserved.

stage: impl-review - ran [2026-09-06T04:43:31Z..2026-09-06T04:47:19Z] (SHIP; codex:gpt-5.6-sol:medium; receipt `/tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.6.json`)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: tracker-sync - skipped(config: tracker inactive)
## Evidence
- Commits:
- Tests: baseline: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Umpire.Observation.Tests Umpire.Planning.Tests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.ObservationTests (128/128; /tmp/fn62-task6-baseline.log), TDD red: cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Space.Tests.Compilation Umpire.PromotionTests (failed on missing authoredKnownGaps as expected; /tmp/fn62-task6-red.log), cd model && mise exec -- lake build Umpire.Query.Tests.AuthoredKnownGaps (27/27), cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Planning.Tests Umpire.Space.Tests.Compilation Umpire.PromotionTests Temporal.Feature.Nexus.OperationsTests (89/89; /tmp/fn62-task6-focused-final.log), cd model && mise exec -- lake build Umpire.ImportTests Umpire.Query.Tests.Visibility Umpire.Planning.VisibilityTests (88 jobs; /tmp/fn62-task6-public.log), cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests (47 jobs), cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Planning.Tests Temporal.Feature.Nexus.OperationsTests (78/78; /tmp/fn62-task6-final-quick.log), structural extraction diff against staged task-start tree 734d38c18c7b1ede52a6797543ba894d9d9d091d (zero diff), make lint-model (260/260; /tmp/fn62-task6-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited rc2; exactly 1316 normalized headers; SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; byte-identical to approved baseline; /tmp/fn62-task6-lint-code.log), impl-review SHIP codex:gpt-5.6-sol:medium (/tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.6.json)
- PRs: