---
satisfies: [R2, R4, R7]
---
# fn-10-temporal-semantic-model-layout-and.3 Move the Nexus auto-close feature model

## Description
Relocate the auto-close semantic model into `Temporal.Feature.Nexus.AutoClose` and update its namespace mechanically (R2, R4, R7). Keep a temporary root import module solely to preserve intermediate Lake builds; task 7 removes it.

**Size:** S
**Files:** `model/Temporal/Feature/Nexus/AutoClose.lean`, `model/NexusAutoClose.lean`
**Touches:** [model/Temporal/Feature/Nexus/AutoClose.lean, model/NexusAutoClose.lean]

### Approach
- Move the model before changing references so the extensive tutorial and proof comments remain intact.
- Rename the namespace and qualified uses to the approved Feature identity.
- Preserve state transitions, invariants, proof witnesses, and public declarations without redesign.

### Investigation targets
**Required** (read before coding):
- `model/NexusAutoClose.lean:77-427` — explanatory model and proof comments to preserve
- `model/NexusAutoClose.lean:181-1101` — current semantic model and proofs
- `model/Temporal/Umpire/NexusCallerClosure.lean:1-7` — current import/open sites

**Optional** (reference as needed):
- `.plans/UMPIRE_DSL.md:1135-1204` — approved Feature ownership and dependency direction

### Acceptance
- [ ] The model compiles as `Temporal.Feature.Nexus.AutoClose`.
- [ ] Existing state-machine semantics, examples, invariants, and proofs remain intact.
- [ ] Existing comments remain attached and materially unchanged.
- [ ] The temporary root import contains no compatibility declarations or aliases.

## Acceptance
- [ ] Auto-close compiles in the Feature/Nexus namespace.
- [ ] Proofs, semantics, and comments are preserved.
- [ ] The transition root is import-only and scheduled for removal.

## Done summary
Relocated the Nexus AutoClose model to `Temporal.Feature.Nexus.AutoClose`, retained an import-only `NexusAutoClose` transition root, and preserved its checked semantics, proofs, examples, and tutorial comments. Rebased moved source references and stabilized caller-closure Config serialization so deterministic artifacts and semantic digests remain unchanged; the baseline handoff and final regression gate were green.

stage: impl-review - ran [2026-08-25T22:52:34Z..2026-08-25T23:01:43Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8475d9a4be9b9b0cb91d72e5f256506c601ff711, 3a7063f8d66e789f9aa0f6b2423b754f8a41577a, eb893b1003493e13b224b7238e7878504921f79a
- Tests: baseline: green via handoff (green (verified at 8b4a744d by fn-10-temporal-semantic-model-layout-and.2)), GATE_SKIPPED:smoke:green-receipt 8b4a744d - baseline reused from prior post-gate pass, mise exec -- lake build NexusAutoClose Temporal.Umpire.NexusCallerClosure TemporalUmpireTests, mise exec -- lake build NexusAutoClose Temporal.Umpire.NexusCallerClosure TemporalUmpireTests temporal-umpire-inspect, make umpire-check-regression
- PRs: