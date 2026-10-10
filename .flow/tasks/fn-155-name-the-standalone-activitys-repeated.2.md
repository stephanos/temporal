---
satisfies: [R1, R3, R4]
---
# fn-155-name-the-standalone-activitys-repeated.2 Collapse the System machine's held-attempt ending

## Description
Implements spec sections A and C, plus the System-machine parts of D: the landing function, the collapsed retry/restart effects, named guards, deadline guards that read `states.*Armed`, the `resetSettles` collapse, one shared reset reason, the initial-state derivations and the `resetDispatch` call. This is the core of the spec.

**Size:** M
**Files:** the System-machine owner from fn-151 (today `system/System.scala`) and the fn-151 subject files that hold `resetSettles`, `resetResumes`/`resetKeepsPaused`, `completedOnRetry`, `restarted` and `keptPaused`
**Touches:** [model/temporal/features/activity/standalone/system/System.scala, model/temporal/features/activity/standalone/system/RetryFailures.scala, model/temporal/features/activity/standalone/system/Reset.scala]

Scope: (never Dispatch* or Realization.scala; any additional concrete subject requires conductor authorization after task 1's re-anchor)

### Approach
- **Landing function.** Add it with its status companion in `states`. Collapse only effects whose `because` text and fact list match after the landing phase is factored out. Keep the surviving names `backOff`, `applyReset` and `heartbeatTimeOut`.
- **Effect-name guard.** `tools/umpire/lower/withholding_test.go:47-50` hard-codes `effects$.backOff`. Keep that name. Go-test updates belong to task 6.
- **Named guards.** Add `exhausted` and `endsTerminally` in `states`. Replace the lambdas in the failure, By-ID failure, start-to-close and heartbeat rules.
- **Rule arms.** Keep `respondFailed` a Match. `tools/umpire/export/quint_test.go:258` and `export/open_test.go:154` read it.
- **Deadline guards.** Every deadline rule reads `states.scheduleToCloseArmed`, `states.scheduleToStartArmed`, `states.startToCloseArmed` or `states.heartbeatArmed`.
- **`resetSettles`.** Collapse its two branches through the landing function. If task 1's probe refused a `states` def inside a property, keep both branches and record that.
- **Shared reset reason.** One `val` for the six `overriding` `because` strings. Follow the `because = <val>` precedent at `system/Dispatch.scala:69`.
- **Initial-state derivations.** `completedOnRetry`, `restarted` and `keptPaused` become `init.copy(...)`. The direct-reset Properties call `states.resetDispatch`.
- **Mapping.** Update `.flow/tmp/fn-155/mapping.md` with every merged or renamed effect.

### Investigation targets
**Required:**
- System machine effects, rules and properties (pre-split `system/System.scala:171-325`, `:407-534`, `:638-695`, `:862-894`) — re-anchored in task 1's `mapping.md`
- `model/temporal/capabilities/Deadline.scala` — `armed` usage
- `tools/umpire/lower/withholding_test.go:40-60`

### Key context
- **`.because` text.** It cannot be passed as a parameter. Effects with distinct reasons stay distinct.
- **No `effect {}` blocks here.** They would reorder facts.

### Acceptance
- [ ] The Pins step-table test passes unchanged
- [ ] `project.py` on a scratch lift shows only positions, mapped identities and recorded structural-review entries
- [ ] Interpreter-built step tables for the System machine and its subject models match the baseline
- [ ] No inline `Timeout.expires` comparison remains in a deadline rule guard
- [ ] Focused Scala tests pass; Go tests are not run here (they read the checked-in IR, which task 6 regenerates)
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
