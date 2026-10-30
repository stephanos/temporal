---
satisfies: [R4]
---
# fn-253-reusable-deadline-countdowns-for-models.1 Activity deadline clarity: Option equality, named expiry effects, split state_bounds

Touches: [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/expected/**]

## Description
Quick readability wins that need no new framework: the lifter already admits Option equality and local `&self` helpers. A reviewed patch (2026-10-25) passed all 33 activity fixture tests, including TLC, before fn-246.15 started. Lands before the countdown work so .7 starts from cleaner code.

**Cross-spec gate:** start only after fn-246.15 and fn-246.13 implementation and Flow closures are committed in `umpire`. The latter owns the approved pending-cancellation outcome correction in the same file. Independent of .2; the conductor verifies these cross-spec gates because task dependencies cannot encode them.

**Size:** M
**Files:** activity `model.rs`, `proof_checks.rs`, `policy_checks.rs`, `expected/`.
**Touches:** [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/expected/**]

### Approach
- In `deadline`: replace each `x.is_some() && x.unwrap_or(Remaining { ticks: 0 }).ticks == 0` with `x == Some(Remaining { ticks: 0 })`. Factor the repeated expiry bodies into `#[effect] fn time_out` (TimedOut and clear) and `#[effect] fn time_out_attempt` (the corrected fn-246.13 pending outcome, otherwise the fn-246.15 retry/terminal rule). Move the four-way armed guard into an unmarked `fn armed(&self) -> bool` placed before the invariants. Reapply this over fn-246.15's retry branch, which may replace `time_out_attempt` for nonpending retries below the cap.
- Split `state_bounds` into `policy_in_range`, `attempt_within_policy` and `countdowns_within_budget`. Update the claim-name lookups for `Invariant_state_bounds` in `policy_checks.rs` and `proof_checks.rs`.
- `tick_priority`: replace each `before.x.is_some() && before.x.unwrap_or(Remaining { ticks: 0 }).ticks == 1` with `before.x == Some(Remaining { ticks: 1 })`. Apply the same rewrite to any other `is_some() && unwrap_or(..) == k` pair in the claims.
- Progress claims: name the shared preconditions with unmarked `&self` helpers (e.g. `pending_cancel()` for `world.asynchronous_cancel && self.phase == Phase::Started && self.cancel_requested`, which `pending_is_cooperative` reduces to `cancel_requested` on reachable states, and `running_without_cancel()` for the four `ordinary_*` claims). Keep the meaning explicit in the helper name, not dropped. If `path!` waypoints refuse helper calls, record the refusal and keep the inline form.
- Audit overlapping Tick/Heartbeat postconditions before deduplicating them. The expiry guards using `<= 1` and `== 1` differ on forged-zero states, so reachable-state agreement alone does not establish duplicate coverage. Retain the required zero-boundary controls and mutants; remove a check only when the remaining named owner preserves its intended boundary coverage. Check the descent overlap against `countdowns_decrement` plus `deadline_arming`, and heartbeat arming against `deadline_arming`. Re-point affected mutant expectations in `proof_checks.rs` and `policy_checks.rs`; each retained mutant must still discriminate under required TLC.
- Reanchor the `global_mutants` anchors in `proof_checks.rs` (`expiry`, `tick`) to the new text; every mutant must still discriminate.
- Reference patch (pre-fn-246.15 base): `/home/agent/.cache/codex-build/tmp/claude-1000/-Users-stephan-Workspace-skunkworks-umpire/d60fd246-d4a3-4007-b584-644d9d8b66df/scratchpad/deadline-clarity.patch` (scratch; may be gone, and the approach above is sufficient).
- **Equivalence pin:** compare with the committed fn-246.15/fn-246.13 corrected baseline. `expected/states.txt.snap` and `rows.txt.snap` stay byte-identical. Review `tlc.txt.snap`, `Activity.tla.snap`, `lift.txt.snap`, report/coverage snapshots and `Activity.cfg.snap` for the explicit invariant renames, lifted expressions and justified claim removals. Record the before/after claim inventory and preserved mutant/boundary owners. Complete and integrate this cleanup before .7; .9 and .10 follow .7.

## Acceptance
- [ ] No `is_some() && unwrap_or(..)` pair left in `deadline`; repeated expiry bodies are named effects.
- [ ] `state_bounds` replaced by three named invariants; name lookups updated.
- [ ] Equivalence pin holds; every mutant still discriminates.
- [ ] `mise exec -- cargo test -p umpire-temporal-tests --test fixtures` passes, also with `UMPIRE_BACKENDS=require`.
- [ ] `tick_priority` uses Option equality throughout; progress preconditions are named helpers (or a recorded refusal).
- [ ] Each removed Tick/Heartbeat check has a recorded remaining owner with the required forged-zero/boundary and mutant coverage; any checks with distinct coverage remain. Re-pointed mutant expectations still discriminate under `UMPIRE_BACKENDS=require`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
