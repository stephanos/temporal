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

### Reviewer context: task 2 evidence

These local evidence pointers supply review context, without changing acceptance, requirement assignments or source scope. The normalized source range is `2756d40478d37414949c7da2328df2673dc7f4dc..cb0f13cccf1aa41e510ca64027e0c926a555b259`; later review-context commits contain Flow metadata only. The implementation is limited to the three Touches paths above. Other tasks' source work is outside this review.

- Worker summary: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task2/.flow/tmp/fn-155/task2/handover-summary.md`.
- Worker evidence: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task2/.flow/tmp/fn-155/task2/handover-evidence.json`.
- Effect mapping: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task2/.flow/tmp/fn-155/task2/mapping.md` and `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task2/.flow/tmp/fn-155/task2/identity-map.json`.
- Structural predicate proof: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task2/.flow/tmp/fn-155/task2/structural-review.json` and its executable `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task2/.flow/tmp/fn-155/task2/structural.py`. The original projection's nine `resetSettles` difference paths remain retained; specialization over all 13 before-state phases leaves remaining fields and ordered facts arbitrary. The five other complete IR/lint projections compare equal. This is recorded structural evidence for the expression collapse, not a silently normalized Property change.
- Complete verification index: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task2/.flow/tmp/fn-155/task2/final-verification.json`.
- Proof seal: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task2/.flow/tmp/fn-155/task2/proof-manifest.json`, SHA256 `9b3b4b366b15d19e7c1905f205df9c3de556ffecc2ac16b924dff3141b97ee93`, covers 537 proof/source files.
- Original ordered comparator: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task2/.flow/tmp/fn-155/task2/table-comparison-attempt2.json`; complete 46-table/13,891,948-record comparison passed on the second natural attempt. The first timeout remains in `table-comparison.json` at that directory. Cumulative comparison time was 1,313.655 seconds, with no reduced domains or third attempt. The original foundation is `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn-155-name-the-standalone-activitys-repeated/.flow/tmp/fn-155/before/`; the unchanged comparator is `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn-155-name-the-standalone-activitys-repeated/.flow/tmp/fn-155/task1/tables/compare.py`.
- Root integrated evidence: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn-155-name-the-standalone-activitys-repeated/.flow/tmp/fn155-integration/task2-integrated.json`, `task2-focused.json`, `task2-focused.log`, `task2-lint.json` and `task2-lint.log` in that directory. The focused runs passed 56 Activity tests and nine capability tests, and lint passed. Root rechecked all source pins unchanged after the Flow metadata integration.

The evidence preserves exact declaration and rule-arm order, outcomes, landing states, ordered facts and `because` variants; `backOff` and `backOffPaused` retain their distinct reasons. Task 2 adds no Case-coordinate exception: its five Case Query source paths retain literal line 1/column 1, while 43 Query IR lines shift. Fresh canonical Case equality remains task 6's gate. Canonical Model/Case/fixture/Go gates retain inherited RED with `canonical_gate_credit: false`, deferred to task 6, fn157, fn154 and Batch5. These evidence pointers grant no full-generator, full-suite, live-execution or replay credit.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
