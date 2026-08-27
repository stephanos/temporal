---
satisfies: [R7]
---
# fn-22-deterministic-replay-semantic.5 Compile the review-only expected-trace promotion proposal

## Description
Add the exact Temporal promotion bridge for the caller-closure result lineage. Statically register `temporal.nexus.caller-closure.promotion.cancel-unique-regression` in fn-5's closed proposal registry with the spec's fixed fresh Behavior/Query IDs, typed fn-21 fault-bearing Query/run/target/kernel constants, validated imports, and a CompiledPromotionSource sealed only after exact source elaboration. In Go, resolve only the verified `temporal-model-promote` sibling, invoke it with that one candidate identity under the exact time/stream Limits, strictly decode fn-5's canonical `umpire-promotion-proposal/v2`, and validate candidate/binding, original Query/artifact/target/kernel, promoted IDs, source identity/SHA, and source bytes against the minimized candidate. Return a separate in-memory cross-binding of the source identity/SHA to minimized candidate, Result, ViolationSignature, and binding identities; none of that runtime lineage enters CompiledPromotionSource. Add mutation tests for command failure/timeout/output Limits, stale/crossed lineage, observed-trace substitution, reused IDs, target/kernel drift, bad imports/constants, unsealed source, elaboration failure, and cross-binding drift.

**Size:** M
**Files:** `tools/umpire/replay/promotion.go`, `tools/umpire/replay/promotion_test.go`, `model/Temporal/Feature/Nexus/CallerClosurePromotion.lean`, `model/Temporal/Feature/Nexus/CallerClosurePromotionTests.lean`, `model/Temporal/Tool/PromotionBinding.lean`, `model/Temporal/Tool/PromoteTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [tools/umpire/replay/promotion.go, tools/umpire/replay/promotion_test.go, model/Temporal/Feature/Nexus/CallerClosurePromotion.lean, model/Temporal/Feature/Nexus/CallerClosurePromotionTests.lean, model/Temporal/Tool/PromotionBinding.lean, model/Temporal/Tool/PromoteTests.lean, model/TemporalModelTests.lean]

## Acceptance
The exact static candidate builds into fn-5's proposal registry only when its expected count-one source elaborates and yields one deterministic unchanged sealed CompiledPromotionSource. The fixed Go invocation accepts only the canonical matching fn-5 envelope and the separate orchestration cross-binding contains complete minimized/result/signature/binding/source identity and digest lineage. Substituting the observed count-two trace, putting runtime lineage into the reusable source type, changing any checked/cross-binding identity, or command/stream drift fails before output. The emitted proposal bytes compile in a clean focused Lake test; runtime never installs them into source, either catalog, stable-regression fixtures, glossary, or generated projections.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
