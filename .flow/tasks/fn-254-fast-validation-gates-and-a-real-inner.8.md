---
satisfies: [R6, R1]
---
# fn-254-fast-validation-gates-and-a-real-inner.8 Final measurement, targets and docs


## Description
Re-measure under .1's conditions, compare with the baseline, and document the two tiers (spec R6).

**Size:** S
**Files:** `model/design/gate-speed.md`, `model/README.md` (verification commands), `model/scripts/time-gate.sh` (targets), `MILESTONES.md`.
**Touches:** [model/design/gate-speed.md, model/README.md, model/GUEST_BUILDS.md, model/umpire/README.md, model/umpire/dev/**/README.md, model/scripts/time-gate.sh, AGENTS.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

### Approach
- Re-run .1's probe on a quiet host. Table: before, after and saving per stage and for the largest binaries; TLC launches and TLC wall time.
- Set `time-gate.sh` soft targets to the owner-confirmed values, defaulting to the proposals in spec R6.
- README: when to use `model-quick`, `model-check`, `model-validate`; what the TLC cache does and does not prove.
- Validation-cost review for spec close, reusing this measurement.
- Include actual reversible activity edits for both cache-hit and fresh-TLC paths, standalone-copy validation, stage/count budget controls and both full gates. Preserve the baseline's inventory/stage identity mappings and account for bounded budget retries in totals. Root alone edits MILESTONES; update moved tool/workspace guidance in AGENTS, module map and guest/tooling docs using writing-for-agents where applicable.

## Acceptance
- [ ] Before/after table recorded; every missed target is reported with its cause.
- [ ] README documents the tiers and the cache limits.
- [ ] `make model-check` and TLC-required `make model-validate` pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
