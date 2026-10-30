**Scope read:** task `.1` of fn-253, diff over the activity model, its two check files, README, five snapshots, and the new review-preflight artifacts. Source semantics, mutant anchors, and snapshot renames were verified against the current files; the full required TLC run is recorded as still in progress and is not treated as passed here.

## Findings

**P2 · confidence 100 · introduced**
`.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task1/review-preflight-20261030/equivalence-and-owners.json:378` · R-IDs `[R4]`
**Problem:** The equivalence record's `hashes_after` lists `tlc.txt.snap` with the same hash as `baseline.json` (`1900ad9a…`), claiming it unchanged. The diff does change `tlc.txt.snap` (two claim lines added, one renamed), and `frozen-source-expected.json:404` in the same directory records a different hash (`88aa6e00…`). The artifact that is supposed to pin the equivalence review therefore misstates which snapshots moved. The states and rows hashes are consistent across both files and the pin for them holds.
**Suggestion:** Regenerate `hashes_after` from the frozen source, or mark the entry as captured before the TLC snapshot refresh so the two artifacts agree.

**P3 · confidence 100 · introduced**
`model/design/deadline-countdowns.md:98` · R-IDs `[]`
**Problem:** The design doc still says the per-Model effects `time_out`/`time_out_attempt` belong to "the later refactor" and that current consumers include `expire_attempt`. After this change the effects exist and `expire_attempt` is gone. The README was corrected for the same statement but this sentence was not. The path is outside `<changed_files>`, so it is noted for consistency only.
**Suggestion:** Update the sentence to name `deadline`, `retry`, `time_out`, `time_out_attempt`, `clear_terminal` as current owners.

## FYI (non-blocking)

- **Acceptance "tick_priority uses Option equality throughout" is not literally achievable.** The corrected baseline compares with `<= 1`, not `== 1`, so there is no `is_some() && unwrap_or(..) == k` pair to rewrite (`model.rs:596-661`). The preflight artifact records this; the task's Done summary should record the same deviation so the checkbox is not read as met.
- **Required-backend acceptance is unverified at this commit.** `status.json` says the full `UMPIRE_BACKENDS=require` run is still executing; only the corrected `global_loses` complement has a logged pass. Closure still needs the full run, as the artifacts themselves say.
- **Module-wide lint expectation.** `#[expect(clippy::unused_self, …)]` on the whole machine module (`model.rs:7`) suppresses the lint for every future method, not just `policy_in_range`. This follows the `optional_facts` fixture precedent and is a licensed suppression with reason; just note the widened scope.
- **Mutant anchors verified.** `replace_in` regions `fn time_out_attempt(`..`fn clear_terminal(` and `self.decrement();`..`Some(ExpiredDeadline::ScheduleToClose)` are unique and contain exactly one target each; `replace_code` anchors for `expiry` and `tick` match once in whitespace-compacted source. The Canceled mutations now add an explicit `clear_terminal()` to preserve the pre-rename mutation semantics. Correct.
- **Clause accounting verified.** The three new invariants carry 5 + 1 + 4 = 10 checks, matching the old `state_bounds`; `countdowns_within_budget` retains `> 0` so the forged-zero test still refutes it. Snapshot ordering (cfg, coverage, report, tlc) matches source order.
- **Code smell, pre-existing:** `tick_priority` repeats the `is_some() && unwrap_or(..).ticks <= 1` triple across four checks (Duplicated Code). Untouched by this diff; a named `about_to_expire(x)` helper would be the natural .7 follow-up.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R4 | partial | `state_bounds` split into three named invariants and all lookups updated; expiry chain in `deadline` reads as one priority list with no `unwrap_or(Remaining { ticks: 0 })` left in that function. Remaining `unwrap_or` sites in `decrement`, `tick_priority`, `countdowns_within_budget` and the `Attempts` wrapper are assigned by the spec to tasks .7 and .9. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>