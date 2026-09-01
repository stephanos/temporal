---
satisfies: [R2]
---
# fn-5-umpire-discovery-promotion-and-artifact.5 Bind the duplicate-delivery promotion candidate

## Description
Create the one static Temporal binding that fn-22 may invoke only after its reproduction,
minimization, and Exact Replay gates pass.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotion.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotionTests.lean`, `model/Temporal/Tool/PromotionEligibility.lean`, `model/Temporal/Tool/PromotionBinding.lean`, `model/Temporal/Tool/PromotionBindingTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotion.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotionTests.lean, model/Temporal/Tool/PromotionEligibility.lean, model/Temporal/Tool/PromotionBinding.lean, model/Temporal/Tool/PromotionBindingTests.lean]

### Approach

- Register exactly `temporal.nexus.caller-closure.promotion.cancel-unique-regression` against the
  existing duplicate-delivery negative-control Query, target, kernel, and expected count-one plan.
- Fix fresh promoted identities `workflow-nexus.behavior.regression.cancel-is-unique` and
  `workflow-nexus.query.regression.cancel-is-unique`; reject collisions with every identity exposed
  by the closed Nexus discovery inventory.
- Bind the exact required imports and compile through task `.4` into one sealed
  `CompiledPromotionSource`.
- Define the exact `umpire-reviewed-promotion-eligibility/v1` handoff for this candidate. It carries
  canonical reproduced-result, complete `minimized|irreducible`, and Exact Replay receipt bytes plus
  their identities/digests and cross-binds them to the same original result, Violation Signature,
  minimized candidate, checked Query lineage, and fixed promotion binding.
- Keep `CheckedPromotionEligibility`'s constructor private. Recompute every receipt identity and
  admit the token only when all gates are successful, complete, canonical, and cross-bound. Resolve
  the static binding only from that token; a candidate identity alone has no resolution API.
- Keep the static source binding independent of fn-22 implementation types. Fn-22 produces the
  canonical handoff after its runtime gates and consumes this fn-5 checker, preserving dependency direction.

### Non-goals

- No candidate registry extension point, generic graph lookup, unchecked identity/source override, observed-trace promotion, or automatic installation.

### Investigation targets

**Required:**
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean` — original checked expected lineage.
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureFault.lean` — duplicate-delivery fault intent.
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureFaultTests.lean` — count-two negative-control evidence.
- `model/Umpire/Promotion.lean` — task `.4` sealed-source boundary.
- `model/Temporal/Tool/NexusDiscovery.lean` — task `.1` closed identity inventory.
- `.flow/specs/fn-22-deterministic-replay-semantic.md` — downstream dependency direction and caller gate.

### Quick command

`cd model && mise exec -- lake build Temporal.Feature.Nexus.Experimental.CallerClosurePromotionTests Temporal.Tool.PromotionBindingTests`

## Acceptance
- [ ] Exactly one fixed candidate resolves, and only from private-constructor checked eligibility; a bare identity has no production resolution path.
- [ ] The binding uses the existing fault-bearing Query lineage and its target-owned expected count-one trace, never the observed count-two result.
- [ ] The two promoted identities are fixed, fresh, and collision-checked against every retained Nexus discovery identity.
- [ ] Missing, incomplete, non-success, noncanonical, digest-mismatched, or crossed reproduction/reduction/Exact-Replay receipts fail before binding resolution.
- [ ] Changed candidate/query/artifact/target/kernel/import/promoted identity or source digest fails before a sealed source is returned.
- [ ] Runtime/reduction/replay lineage remains outside `CompiledPromotionSource`, and the fn22-to-fn5 dependency direction is preserved.
- [ ] Existing comments in touched files are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
