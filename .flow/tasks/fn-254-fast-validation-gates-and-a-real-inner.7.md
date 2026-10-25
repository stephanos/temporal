---
satisfies: [R4]
---
# fn-254-fast-validation-gates-and-a-real-inner.7 Temporal crates: one test binary per crate after fn-246

Touches: [model/temporal/**/tests/**]

## Description
Apply .4's consolidation to `model/temporal/**`.

**Cross-spec gate:** start after fn-246's committed closure; its lanes edit `temporal/realization` and activity tests.

**Size:** S
**Files:** `model/temporal/**/tests/*.rs`.
**Touches:** [model/temporal/**/tests/**]

### Approach
- Same rules and exceptions as .4. `temporal/testing`'s `fixtures`, `boundary` and `realization` binaries are the main candidates; keep `boundary` separate if the gate's `source-boundary` stage runs it alone.
- **Equivalence pin:** full test list identical modulo the recorded path mapping.

### Acceptance
- [ ] Binary count and compile/link time recorded before and after.
- [ ] Test list identical modulo mapping; snapshots unchanged; `make model-check` passes.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
