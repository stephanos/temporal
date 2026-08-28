---
satisfies: [R2, R4, R7]
---
# fn-41-finitemachine-target-authoring.2 Migrate Feature Nexus Lifecycle

## Description
Move the ordinary Feature Lifecycle target onto the proven adapter (R2, R4, R7). Keep the family semantic vocabulary and public proof surface intact while deleting only the duplicated assembly that `FiniteMachine` now owns.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Lifecycle.lean`, `model/Temporal/Feature/Nexus/LifecycleTests.lean`, `model/Temporal/Feature/Nexus/OperationsTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Lifecycle.lean, model/Temporal/Feature/Nexus/LifecycleTests.lean, model/Temporal/Feature/Nexus/OperationsTests.lean]

### Approach
- Declare one family-owned `finiteMachine` from the existing ordered domains, encoders, `initialStates`, `stepResults`, residual closure proofs, and per-action executable witnesses.
- Make the existing `authoritativeInitial`, `authoritativeStep`, sound/complete theorem names, `transitionKernel`, and `finitePlanning` declarations delegate to the adapter so qualified names and consumer types remain compatible.
- Preserve the typed lifecycle `step`, ModelValue conversions, exposed-result/case theorems needed for semantic closure, provider/composition/definition values, source provenance, and named target transition lemmas.
- Remove the handwritten domain-membership equivalences and kernel/planning record wiring now derived by the adapter; do not remove semantic case analysis that establishes genuine coverage or action executability.
- Strengthen focused compatibility assertions around the descriptor-to-kernel relation, planning action order, canonical checked target, fingerprint, and operation Query/plan/Artifact fixtures without accepting fixture regeneration.
- Preserve every existing declaration comment and keep later fn-39 physical splitting viable behind the same public Lifecycle facade.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Lifecycle.lean:175-217` — repeated membership relations and identity proofs.
- `model/Temporal/Feature/Nexus/Lifecycle.lean:227-484` — genuine coverage proofs, handwritten kernel, planning, and Target assembly.
- `model/Temporal/Feature/Nexus/LifecycleTests.lean:11-82` — current semantic, kernel, canonical-target, and planning compatibility checks.
- `model/Temporal/Feature/Nexus/OperationsTests.lean:1-190` — downstream Query, planner, and Artifact compatibility coverage.
- `model/Temporal/System/Nexus/ImplementationLink.lean:321-360` — public authority proof seam that must continue to elaborate unchanged.

**Optional** (reference as needed):
- `.flow/tasks/fn-39-make-the-temporal-nexus-feature-model.1.md:12-27` — later facade split that must consume the simplified target.

## Acceptance
- [ ] Lifecycle constructs its kernel and planning through `FiniteMachine`, with routine membership/record boilerplate removed and genuine semantic evidence still explicit.
- [ ] Existing public declarations, theorem types, imports, source paths, IDs, canonical metadata, Behavior Fingerprint, planning action order, Queries, plans, Artifacts, and comments remain unchanged.
- [ ] The three supported and all unsupported lifecycle transitions retain their current results.
- [ ] Existing System Implementation Link consumers elaborate through the stable named authority seam without adapter-internal unfolding.
- [ ] Generated fixtures are unchanged and unrelated worktree files are untouched.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
