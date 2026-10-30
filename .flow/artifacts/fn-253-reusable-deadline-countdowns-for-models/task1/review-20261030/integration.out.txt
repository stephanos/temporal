## Review: fn-253.1 activity deadline clarity (`36a03b6..b7347d8`)

Scope read: task spec, the full diff, current `model.rs`, `proof_checks.rs`, `policy_checks.rs`, the snapshot hunks, `umpire_testing::source_edit`, the TLA `Some` lowering in `backends/tla/src/emit.rs:692`, and the before/after `Deadline_Tick` TLA definitions. No shell in this session, so no commands were run; verification evidence is the committed preflight logs plus my own trace of the lowering.

### Source semantics (integration with unchanged code)

- **Expiry guards are equivalent.** Old `x.is_some() && x.unwrap_or(0).ticks == 0` lowers to `x ≠ 0 ∧ (IF x ≠ 0 THEN x−1 ELSE 0) = 0`; new `x == Some(Remaining { ticks: 0 })` lowers to `x = (0)+1`. Both are `x = 1` on every integer, forged values included. Confirmed by reading the `Deadline_Tick` before/after lines in `Activity.tla.snap`; the rest of the definition is byte-identical.
- **`armed()` and the progress helpers inline identically.** `Deadline_Tick` carries the same four-way disjunction; no `Progress_*` definition changed, so `pending_cancel(world)` and `running_without_cancel()` preserved all conjuncts. `states.txt.snap` and `rows.txt.snap` hashes are unchanged in both `baseline.json` and `frozen-source-expected.json`.
- **`time_out_attempt` preserves the fn-246.13/fn-246.15 rule** (pending → TimedOut; else below cap → `retry`; else TimedOut). `time_out` is exactly the old inline pair.
- **Ten bounds clauses retained** across `policy_in_range` (5), `attempt_within_policy` (1), `countdowns_within_budget` (4). `deadline_arming`, `pending_is_cooperative`, and all Tick/Heartbeat postconditions untouched.
- **Mutant anchors re-point correctly.** `replace_in` regions `fn time_out_attempt( .. fn clear_terminal(` (`model.rs:286–297`) and `fn retry( .. fn clear_terminal(` each contain their `from` text exactly once; the enlarged retry region now spans `time_out`, which contains none of `self.attempt += 1;` / `ticks: world.schedule_to_start…`. `replace_code` compact anchors `ifself.schedule_to_close_remaining==Some(Remaining{ticks:0}){` and `Deadline::Tickif!self.closed()&&self.armed()` each match once. The corrected `global_loses` mutant (`Canceled; clear_terminal()`) is semantically the same mutation as before.
- **Module-level `#[expect(clippy::unused_self)]`** matches the `optional_facts` fixture precedent; clippy log shows it is fulfilled (by `policy_in_range`).
- No references to `state_bounds`/`expire_attempt`/`Invariant_state_bounds` remain in code, goldens, or tooling outside `.flow/` and the activity README.

### Findings

**P3 · Confidence 100 · introduced · `.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task1/review-preflight-20261030/equivalence-and-owners.json:378` · R-IDs: []**
- **Problem:** `hashes_after` records `tlc.txt.snap` as `1900ad…`, the *baseline* hash, i.e. claims the TLC snapshot is unchanged. The same commit changes `tlc.txt.snap` (+3/−1) and `frozen-source-expected.json:404` records it as `88aa6e…`. The equivalence record is internally inconsistent on the one snapshot the spec says to review explicitly.
- **Suggestion:** Regenerate `hashes_after` from the frozen tree (or note that it was captured before the `INSTA_UPDATE` pass) so the equivalence pin and frozen-source records agree.

**P3 · Confidence 100 · introduced · `model/temporal/feature/activity/model.rs:596–661` · R-IDs: []**
- **Problem:** Task acceptance bullet "`tick_priority` uses Option equality throughout" is not literally met: every `tick_priority` comparison is still `is_some() && unwrap_or(..).ticks <= 1`. The artifact records this as deliberate (no `== 1` pair exists; `<= 1` is the forged-zero boundary control the same task says to retain), and I agree the retention is correct. But the checkbox and the code disagree.
- **Suggestion:** Either rewrite as `x == Some(Remaining{ticks:0}) || x == Some(Remaining{ticks:1})` (same lowering shape, no coverage change) or amend the acceptance text to record the retention. Owner's call; not a blocker.

**FYI (not findings):**
- `model/design/deadline-countdowns.md:98` still lists `expire_attempt` as a current consumer. It is outside this task's `Touches` and the sentence itself defers reconciliation to the .7 refactor; flagging only so .7 picks it up.
- The full `UMPIRE_BACKENDS=require` fixture run is recorded as *in progress* (`status.json`). I am not treating it as passed. The focused complement for the one re-anchored policy mutant passed; the remaining required controls are a closure gate, not a review gate here.
- Clause split keeps `policy_in_range` with an unused `&self` behind a module-wide `expect`. Fine today; if a later change makes every method use `self`, the expectation becomes unfulfilled and `-D warnings` fails. No action needed.

No overengineering: every addition (two effects, three helpers, three invariants) is what the task asked for. No security surface touched.

### Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R4 | partial | .1 slice met: no `unwrap_or(Remaining { ticks: 0 })` in `deadline` (`model.rs:238–265`); `state_bounds` split into the three named invariants; expiry chain reads as one priority list. Remaining R4 bullets (countdown library adoption, `.9` `Attempts` wrapper) are assigned to .7/.9 by the spec's coverage table. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 0 pre_existing.
Protected-path filter: 0

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>