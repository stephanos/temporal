**Scope reviewed.** Source semantics in `model.rs`, the DSL-admitted helper behavior, mutant anchor ownership in the two check files, the snapshot renames, and the preflight artifacts, all against task .1 and epic R4. The full required-TLC run is still in flight per `status.json`, so this review certifies the native evidence and the focused required complement only, not the full required pass.

**Semantics verified.**
- `deadline` is behavior-preserving. Each `x.is_some() && x.unwrap_or(..).ticks == 0` became `x == Some(Remaining { ticks: 0 })`. The lifter encodes the new form as `x = 0 + 1` and the old as `x ≠ 0 ∧ x − 1 = 0`, which coincide under the None-is-zero integer encoding. States and rows snapshot hashes are byte-identical to the baseline and `tlc.txt.snap` still reports 18 states with native agreement.
- `time_out` and `time_out_attempt` reproduce the prior bodies exactly, including the fn-246.13 pending-cancellation outcome (TimedOut, not Canceled) and the fn-246.15 retry-below-cap rule.
- `armed`, `pending_cancel`, and `running_without_cancel` are inlined by the lifter. Progress definitions in the TLA snapshot keep all conjuncts.
- `state_bounds` split keeps all ten clauses, now partitioned as four policy-range checks, one attempt check, and four countdown checks plus the attempts-range check. Nothing was dropped and no Tick or Heartbeat claim was removed.
- Mutant anchors: every `replace_in` region still contains its anchor exactly once in the current source. The `pending_expiry_cancels` and `global_loses` replacements add an explicit `clear_terminal()` so they are the same mutation as before, and `Post_tick_priority` remains their owner. `global_mutants` `expiry` and `tick` anchors match the new text; `broken_expiry` and `missing_tick` still differ from healthy. `forged_countdowns_refute_bounds_and_armed_zero_expires` now looks up `countdowns_within_budget`, the clause owner for forged zero and out-of-range values.

## Findings

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task1/review-preflight-20261030/equivalence-and-owners.json:378`
- **R-IDs**: []
- **Problem**: `hashes_after` records `tlc.txt.snap` at the baseline hash `1900ad…`, but the diff changes that snapshot (two claim lines added, one renamed) and `frozen-source-expected.json` records it as `88aa6e…`. The equivalence record contradicts the frozen-source record for the same commit.
- **Suggestion**: Regenerate `hashes_after` from the frozen tree so both artifacts agree before closure.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/design/deadline-countdowns.md:98`
- **R-IDs**: [R4]
- **Problem**: The design note still lists `expire_attempt` as a current consumer and `time_out`/`time_out_attempt` as later-refactor names. After this change those effects exist and `expire_attempt` does not. The README was corrected; the design doc was not.
- **Suggestion**: Update the sentence to name `time_out_attempt` as the current consumer, or leave for .7 if the conductor prefers the design doc to move with the library adoption.

**FYI (non-blocking).**
- The task acceptance lines for `UMPIRE_BACKENDS=require` and "every mutant still discriminates" are evidenced only by the native run and the one focused required complement. The full required run must land before closure, as the status artifact already states.
- `tick_priority` keeps `is_some() && unwrap_or(..).ticks <= 1`. Option equality cannot express `<= 1`, and the artifact records that no `== 1` pair existed to rewrite. The acceptance clause's recorded-refusal path covers this.
- The module-level `#[expect(clippy::unused_self)]` carries a reason and is forced by the source-admission rule against function-local attributes. It is broad, so a future genuinely unused `&self` elsewhere in the module will not be flagged.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R4 | partial | No `unwrap_or(Remaining { ticks: 0 })` in `deadline`; `state_bounds` split into the three named invariants; expiry chain reads as one priority list. Library adoption (R2/R3 use) and the `.9` wrapper are owned by later tasks. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>