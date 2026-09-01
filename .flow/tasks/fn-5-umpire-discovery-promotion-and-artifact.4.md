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
  bind a checked original Query, its target/kernel-owned `.found` trace, fresh promoted
  Behavior/Query identities, fixed imports, and deterministic source bytes.
- Recompute the original plan and require whole-value equality before rendering; the observed runtime
  trace is never accepted as expected model behavior.
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
- [ ] Only a checked original Query with its recomputed target-owned `.found` trace can produce a sealed source.
- [ ] Non-found results, target/kernel/query drift, trace/reason/count drift, reused promoted identities, missing imports, nondeterministic rendering, and digest drift fail without a partial source.
- [ ] Substituting the observed duplicate-delivery count-two trace for the expected count-one trace is rejected by a focused test.
- [ ] Exact source bytes elaborate in a clean focused Lake invocation before `CompiledPromotionSource` is exposed.
- [ ] The reusable module imports no Temporal, Nexus, runtime, replay, minimization, filesystem, or command package.
- [ ] Existing comments in touched files are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
