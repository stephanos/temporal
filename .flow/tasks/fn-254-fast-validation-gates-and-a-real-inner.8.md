---
satisfies: [R6, R1]
---
# fn-254-fast-validation-gates-and-a-real-inner.8 Final measurement, targets and docs

Touches: [model/design/gate-speed.md, model/README.md, model/scripts/time-gate.sh, MILESTONES.md]

## Description
Re-measure under .1's conditions, compare with the baseline, and document the two tiers (spec R6).

**Size:** S
**Files:** `model/design/gate-speed.md`, `model/README.md` (verification commands), `model/scripts/time-gate.sh` (targets), `MILESTONES.md`.
**Touches:** [model/design/gate-speed.md, model/README.md, model/scripts/time-gate.sh, MILESTONES.md]

### Approach
- Re-run .1's probe on a quiet host. Table: before, after and saving per stage and for the largest binaries; TLC launches and TLC wall time.
- Set `time-gate.sh` soft targets to the owner-confirmed values, defaulting to the proposals in spec R6.
- README: when to use `model-quick`, `model-check`, `model-validate`; what the TLC cache does and does not prove.
- Validation-cost review for spec close, reusing this measurement.

### Acceptance
- [ ] Before/after table recorded; every missed target is reported with its cause.
- [ ] README documents the tiers and the cache limits.
- [ ] `make model-check` and TLC-required `make model-validate` pass.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
