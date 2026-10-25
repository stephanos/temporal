---
satisfies: [R8]
---
# fn-254-fast-validation-gates-and-a-real-inner.12 Per-stage budgets: committed table, hard work counts, wall time with tolerance


## Description
Make the fast-gate goal enforceable for every current and future stage (spec R8): a committed budget table, hard work-count checks, and wall-time budgets with a tolerance. fn-255's stages (and anything added later) must declare entries here.

**Size:** M
**Files:** budget table (framework side in `model/umpire`, Temporal side beside the root gate; format per fn-254.9), the framework entry point's timing (successor of `model/scripts/time-gate.sh`), `test-time-gate.sh` controls.
**Touches:** [model/umpire/**, model/scripts/time-gate.sh, model/scripts/test-time-gate.sh, Makefile]

### Approach
- Use .1's historical timing references as initial estimates, then calibrate against the optimized stages and actual counts produced by .2/.3/.4/.5/.6/.11. Historical host/workload differences remain explicit; no fresh pre-optimization timing campaign is required. Count budgets use actual discovered/executed inventories and launch counts, not historical estimates.
- Enforcement: count overruns and unknown/omitted stage registrations fail immediately. Use a 25% warm-time tolerance by default. After an overrun, drain competing owned work and retry that stage once immediately with identical source/tool/budget/tier inputs. A second overrun fails; second-fast passes with the first overrun reported. Interrupted/incomparable retries are incomplete evidence. Include both attempts in tier totals; use invocation-local records so concurrent gates cannot reset one another's state.
- Counts distinguish discovered/executed/ignored/skipped tests and actual TLC launches/configurations. Consume .2/.3/.4/.5/.6/.11's final framework stage/count inventory. Later .7 and fn-255 stage changes update these entries as part of their implementation; .12 does not wait for deferred Temporal consolidation.
- Output: one line per stage with actual vs budget, plus tier totals against fn-254 R6's targets.
- Changing a budget is a reviewed table edit with a reason; record that rule in the README.
- Controls cover over-budget counts, unbudgeted/omitted stages, second-slow failure, first-slow/second-fast success, interrupted retry and independent concurrent invocation records. Each failure names the stage, and no backend refusal is converted to a timing-only warning.

### Investigation targets
**Required:**
- `model/scripts/time-gate.sh`, `test-time-gate.sh`
- `model/design/gate-speed.md` (.1 baseline)
- `model/design/framework-gate.md` (.9 layout)

## Owner validation cadence (2026-10-25)

Keep stage registration/count/timing/retry controls now. Derive actual count bounds and preserve deferred full-gate/final timing obligations for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] A committed budget table covers every current stage, framework and Temporal.
- [ ] Work-count overruns and unbudgeted stages fail; wall-time overruns fail only on repeat; all four controls pass.
- [ ] README documents how to add a stage and how to change a budget.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
