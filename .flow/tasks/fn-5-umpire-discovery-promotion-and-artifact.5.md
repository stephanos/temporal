---
satisfies: [R2]
---
# fn-5-umpire-discovery-promotion-and-artifact.5 Bind the duplicate-delivery promotion candidate

## Description
Create the one static Temporal binding whose inert checked proposal fn-22 may cross-bind only after
its reproduction, minimization, and Exact Replay gates pass.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotion.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotionTests.lean`, `model/Temporal/Tool/PromotionBinding.lean`, `model/Temporal/Tool/PromotionBindingTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotion.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotionTests.lean, model/Temporal/Tool/PromotionBinding.lean, model/Temporal/Tool/PromotionBindingTests.lean]

### Approach

- Register exactly `temporal.nexus.caller-closure.promotion.cancel-unique-regression` against two
  explicit lineages: the unchanged `exactActionQuery`, target, kernel, PlannerRun, and base
  `ExperimentSpec` that supply the expected count-one trace; and the selected duplicate-delivery
  Space-point `ExperimentSpec` whose distinct identity/checksum is reproduced and minimized by fn-22.
- Fix fresh promoted identities `workflow-nexus.behavior.regression.cancel-is-unique` and
  `workflow-nexus.query.regression.cancel-is-unique`; reject collisions with every identity exposed
  by the closed Nexus discovery inventory.
- Bind the exact required imports and compile through task `.4` into one sealed
  `CompiledPromotionSource`.
- Emit only inert checked source lineage. Runtime Result, Violation Signature, reproduction,
  minimization, and Exact Replay receipts never enter the binding or `CompiledPromotionSource`.
- Keep the static binding independent of fn-22 implementation types. Fn-22 consumes the fixed
  candidate output, separately validates and cross-binds its runtime lineage, and owns the eligibility
  claim and review-artifact write, preserving dependency direction.

### Non-goals

- No candidate registry extension point, generic graph lookup, dynamic identity/source override, observed-trace promotion, runtime eligibility claim, or automatic installation.

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
- [ ] Exactly one fixed candidate identity resolves, with no path, import, identity, trace, or source override.
- [ ] The unchanged base Query/PlannerRun/base ExperimentSpec supplies the target-owned expected count-one trace, while the selected fault-bearing ExperimentSpec retains its separate identity/checksum; neither is conflated with the observed count-two result.
- [ ] The two promoted identities are fixed, fresh, and collision-checked against every retained Nexus discovery identity.
- [ ] Changed candidate, base Query/plan/ExperimentSpec, fault-bearing ExperimentSpec, target/kernel/import/promoted identity, or source digest fails before a sealed source is returned.
- [ ] Direct candidate resolution makes no runtime eligibility claim; runtime/reduction/replay lineage remains outside `CompiledPromotionSource`, and the fn22-to-fn5 dependency direction is preserved.
- [ ] Existing comments in touched files are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
