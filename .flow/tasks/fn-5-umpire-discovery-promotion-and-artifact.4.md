---
satisfies: [R2]
---
# fn-5-umpire-discovery-promotion-and-artifact.4 Seal review-only Lean promotion sources

## Description
Define the narrow checked source boundary used by the one retained duplicate-delivery promotion.

**Size:** M
**Files:** `model/Umpire/Promotion.lean`, `model/Umpire/PromotionTests.lean`, `model/Umpire.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Promotion.lean, model/Umpire/PromotionTests.lean, model/Umpire.lean, model/UmpireTests.lean]

### Approach

- Define private-constructor `CompiledPromotionSource` and the smallest compiler input needed to
  bind an unchanged base checked Query, its target/kernel-owned `.found` trace and base planned
  `ExperimentSpec`, fresh promoted Behavior/Query identities, fixed imports, and deterministic bytes.
- Recompute the base plan and require whole-value equality with the base `ExperimentSpec` before
  rendering; fault intent and the observed runtime trace are never accepted as expected model behavior.
- Seal a source only after deterministic rendering, SHA-256 identity computation, and successful
  elaboration through a clean focused Lake test.
- Keep runtime reproduction, minimization, Exact Replay receipts, Nexus identities, and proposal
  publication outside the reusable Umpire module.

### Non-goals

- No general artifact evolution, automatic source installation, runtime replay engine, reducer, or dynamic source template surface.

### Investigation targets

**Required:**
- `model/Umpire/Query.lean` — checked Query and target-indexed planning contracts.
- `model/Umpire/Plan.lean` — `.found` result and expected trace ownership.
- `model/Umpire/Artifact.lean` — current checked Query/plan lineage conventions.
- `model/Umpire/Tests/Artifact.lean` — exact whole-value mutation-test style.
- `.plans/LEAN_GUIDELINES.md` — total checked construction and compile-time test rules.

### Quick command

`cd model && mise exec -- lake build Umpire.PromotionTests UmpireTests`

## Acceptance
- [ ] Only an unchanged base checked Query with its recomputed target-owned `.found` trace and matching base ExperimentSpec can produce a sealed source.
- [ ] Non-found results, base target/kernel/query/ExperimentSpec drift, trace/reason/count drift, reused promoted identities, missing imports, nondeterministic rendering, and digest drift fail without a partial source.
- [ ] Substituting the observed duplicate-delivery count-two trace for the expected count-one trace is rejected by a focused test.
- [ ] Exact source bytes elaborate in a clean focused Lake invocation before `CompiledPromotionSource` is exposed.
- [ ] The reusable module imports no Temporal, Nexus, runtime, replay, minimization, filesystem, or command package.
- [ ] Existing comments in touched files are preserved.

## Done summary
Implemented a sealed review-only Lean promotion-source compiler that replans an unchanged checked Query, validates the complete target-owned PlannerRun and ExperimentSpec lineage, rejects observed count-two trace substitution, and exposes no partial source on drift. The renderer is closed over one clean-elaborated syntax shape, accepts only quoted identity/location data, fixes imports and declarations, and seals exact bytes plus SHA-256 behind a private constructor.

Focused compile-time tests cover base/query/target/kernel/artifact/trace/reason drift, non-found planning, reused identities, invalid source data, deterministic bytes and digest, clean fixture elaboration through a typed base Query, and constructor/record-update/syntax-input sealing. The exact Quick build and full model/import-graph lint pass.

stage: impl-review - ran [2026-09-02T15:44:27Z..2026-09-02T15:46:39Z] (model: codex:gpt-5.6-sol:high)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ac7fdbbfc5e860d867875a4451789f64514d9773, c89737339c0a9f09a81ec686b72abd836efeeda7, 1c54abcfbafcef9ffa35291dbbe559742506b935
- Tests: cd model && mise exec -- lake build Umpire.PromotionTests UmpireTests, make lint-model, git diff --check, /Users/stephan/.codex/plugins/cache/flow-next-marketplace/flow-next/4.5.1/scripts/flowctl codex impl-review fn-5-umpire-discovery-promotion-and-artifact.4 --base 3868bc28a34a0c658064af9d74ecb0adc6e95e42 --receipt /tmp/impl-review-receipt-fn-5-umpire-discovery-promotion-and-artifact.4.json (SHIP)
- PRs: