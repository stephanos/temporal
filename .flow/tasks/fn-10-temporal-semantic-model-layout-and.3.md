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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
