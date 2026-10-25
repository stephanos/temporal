---
satisfies: [R4]
---
# fn-253-reusable-deadline-countdowns-for-models.1 Activity deadline clarity: Option equality, named expiry effects, split state_bounds

Touches: [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/expected/**]

## Description
Quick readability wins that need no new framework: the lifter already admits Option equality and local `&self` helpers. A reviewed patch (2026-10-25) passed all 33 activity fixture tests, including TLC, before fn-246.15 started. Lands before the countdown work so .7 starts from cleaner code.

**Cross-spec gate:** start only after fn-246.15's implementation and Flow closure are committed in `umpire`. It rewrites the same deadline cells. Independent of .2.

**Size:** M
**Files:** activity `model.rs`, `proof_checks.rs`, `policy_checks.rs`, `expected/`.
**Touches:** [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/expected/**]

### Approach
- In `deadline`: replace each `x.is_some() && x.unwrap_or(Remaining { ticks: 0 }).ticks == 0` with `x == Some(Remaining { ticks: 0 })`. Factor the repeated expiry bodies into `#[effect] fn time_out` (TimedOut and clear) and `#[effect] fn time_out_attempt` (Canceled if a cancel is pending, else TimedOut, then clear). Move the four-way armed guard into an unmarked `fn armed(&self) -> bool` placed before the invariants. Reapply this over fn-246.15's retry branch, which may replace `time_out_attempt` for nonpending retries below the cap.
- Split `state_bounds` into `policy_in_range`, `attempt_within_policy` and `countdowns_within_budget`. Update the claim-name lookups at `policy_checks.rs:797`, `proof_checks.rs:467` and `proof_checks.rs:608` (`Invariant_state_bounds`).
- `tick_priority`: replace each `before.x.is_some() && before.x.unwrap_or(Remaining { ticks: 0 }).ticks == 1` with `before.x == Some(Remaining { ticks: 1 })`. Apply the same rewrite to any other `is_some() && unwrap_or(..) == k` pair in the claims.
- Progress claims: name the shared preconditions with unmarked `&self` helpers (e.g. `pending_cancel()` for `world.asynchronous_cancel && self.phase == Phase::Started && self.cancel_requested`, which `pending_is_cooperative` reduces to `cancel_requested` on reachable states, and `running_without_cancel()` for the four `ordinary_*` claims). Keep the meaning explicit in the helper name, not dropped. If `path!` waypoints refuse helper calls, record the refusal and keep the inline form.
- Remove overlapping Tick/Heartbeat postcondition checks and give each property one owner: `tick_descends_and_expires`'s expiry check duplicates `tick_priority`'s first check; its descent check duplicates `countdowns_decrement`'s global line (given `deadline_arming`); `heartbeat_preserves_attempt_and_global` re-checks `deadline_arming`'s heartbeat arming. Re-point the mutant expectations naming those claims (`proof_checks.rs:533`, `:611`; `policy_checks.rs:652`, `:657`) to the owning claim; each mutant must still be refuted by a named claim under required TLC.
- Reanchor the `global_mutants` anchors in `proof_checks.rs` (`expiry`, `tick`) to the new text; every mutant must still discriminate.
- Reference patch (pre-fn-246.15 base): `/home/agent/.cache/codex-build/tmp/claude-1000/-Users-stephan-Workspace-skunkworks-umpire/d60fd246-d4a3-4007-b584-644d9d8b66df/scratchpad/deadline-clarity.patch` (scratch; may be gone, and the approach above is sufficient).
- **Equivalence pin:** `expected/states.txt.snap` and `rows.txt.snap` stay byte-identical; `tlc.txt.snap`, `Activity.tla.snap` and `lift.txt.snap` change only in invariant names and lifted expression text, and are reviewed.

### Acceptance
- [ ] No `is_some() && unwrap_or(..)` pair left in `deadline`; repeated expiry bodies are named effects.
- [ ] `state_bounds` replaced by three named invariants; name lookups updated.
- [ ] Equivalence pin holds; every mutant still discriminates.
- [ ] `mise exec -- cargo test -p umpire-temporal-tests --test fixtures` passes, also with `UMPIRE_BACKENDS=require`.
## Acceptance
- [ ] No `is_some() && unwrap_or(..)` pair left in `deadline`; repeated expiry bodies are named effects.
- [ ] `state_bounds` replaced by three named invariants; name lookups updated.
- [ ] Equivalence pin holds; every mutant still discriminates.
- [ ] `mise exec -- cargo test -p umpire-temporal-tests --test fixtures` passes, also with `UMPIRE_BACKENDS=require`.

### Acceptance
- [ ] TBD
- [ ] `tick_priority` uses Option equality throughout; progress preconditions are named helpers (or a recorded refusal).
- [ ] No Tick/Heartbeat property is checked by two postconditions; re-pointed mutant expectations still refute under `UMPIRE_BACKENDS=require`.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
