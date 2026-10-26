---
satisfies: [R6, R1]
---
# fn-254-fast-validation-gates-and-a-real-inner.8 Final measurement, targets and docs


## Description
Measure the final gates warm, compare honestly with .1's historical references, and document the tiers (spec R6). The owner waived fresh baseline runs, not these final measurements.

**Size:** S
**Files:** `model/design/gate-speed.md`, maintained verification documentation, .15's framework/Temporal native timer and controls, .12's existing stage/tier budget tables, `MILESTONES.md`.
**Touches:** [model/design/gate-speed.md, model/README.md, model/GUEST_BUILDS.md, model/umpire/README.md, model/umpire/dev/**/README.md, model/dev/xtask/**, model/umpire/dev/xtask/**, model/dev/gate/budgets.toml, model/umpire/dev/gate/budgets.toml, AGENTS.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

### Approach
- Use the retained optional probe on a quiet host for final gate/inner-loop measurements. Table: current costs/counts beside historical references, with backend/source/workload differences explicit. Attribute savings only where inputs and coverage are comparable; otherwise report the comparison limit rather than inventing a matched before/after run.
- Use .15's native timer and .12's committed tables at `model/umpire/dev/gate/budgets.toml` and `model/dev/gate/budgets.toml`. Calibrate those entries with the final measurements and owner-confirmed R6 targets, recording reasons for changes under .12's review rule. Preserve its count enforcement, tolerance and bounded retry behavior; do not recreate retired shell timers or a second budget mechanism.
- README: when to use `model-quick`, `model-check`, `model-validate`; what the TLC cache does and does not prove.
- Validation-cost review for spec close, reusing this measurement.
- Include actual reversible activity edits for both cache-hit and fresh-TLC paths, standalone-copy validation, stage/count budget controls and both full gates. Preserve the baseline's inventory/stage identity mappings and account for bounded budget retries in totals. Root alone edits MILESTONES; update moved tool/workspace guidance in AGENTS, module map and guest/tooling docs using writing-for-agents where applicable.

## Acceptance
- [ ] Before/after table recorded; every missed target is reported with its cause.
- [ ] README documents the tiers and the cache limits.
- [ ] `make model-check` and TLC-required `make model-validate` pass.

## Throughput proof renewal

Include .17's final stage graph and uniform nextest paths, .5's 24-case mapping/compiler controls and .6's full source-consumer selection. Renew affected standalone/current-input checks after these producers. Current 444.027 s complete-coverage quick evidence is a reported target miss; old approximately 20 s incomplete-consumer runs cannot establish the final SLO or an attributed speedup.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
