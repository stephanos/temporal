---
satisfies: [R8]
---
# fn-254-fast-validation-gates-and-a-real-inner.12 Per-stage budgets: committed table, hard work counts, wall time with tolerance

Touches: [model/umpire/**, model/scripts/time-gate.sh, model/scripts/test-time-gate.sh, Makefile]

## Description
Make the fast-gate goal enforceable for every current and future stage (spec R8): a committed budget table, hard work-count checks, and wall-time budgets with a tolerance. fn-255's stages (and anything added later) must declare entries here.

**Size:** M
**Files:** budget table (framework side in `model/umpire`, Temporal side beside the root gate; format per fn-254.9), the framework entry point's timing (successor of `model/scripts/time-gate.sh`), `test-time-gate.sh` controls.
**Touches:** [model/umpire/**, model/scripts/time-gate.sh, model/scripts/test-time-gate.sh, Makefile]

### Approach
- Seed the table from .1's baseline: per stage, the wall-time budget (warm, baseline host) and, where the stage has them, work counts (test count from the nextest list, TLC launches per fixture once .2 lands).
- Enforcement: a work count above budget fails the stage. A wall-time overrun above the tolerance (proposal 25%) is reported; the stage fails only if a repeat run also overruns. A stage with no table entry fails the gate.
- Output: one line per stage with actual vs budget, plus tier totals against fn-254 R6's targets.
- Changing a budget is a reviewed table edit with a reason; record that rule in the README.
- Controls in `test-time-gate.sh`: a planted over-budget count, a planted unbudgeted stage, and two consecutive planted slow runs each fail with the stage name; one slow run alone passes with a warning.

### Investigation targets
**Required:**
- `model/scripts/time-gate.sh`, `test-time-gate.sh`
- `model/design/gate-speed.md` (.1 baseline)
- `model/design/framework-gate.md` (.9 layout)

### Acceptance
- [ ] A committed budget table covers every current stage, framework and Temporal.
- [ ] Work-count overruns and unbudgeted stages fail; wall-time overruns fail only on repeat; all four controls pass.
- [ ] README documents how to add a stage and how to change a budget.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
