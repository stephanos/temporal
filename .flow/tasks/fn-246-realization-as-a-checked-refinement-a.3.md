---
satisfies: [R4, R7]
---
# fn-246-realization-as-a-checked-refinement-a.3 Activity binding rewritten onto the shared layer: action registry, entity and read declarations, snapshot projection table, claims, fillers; gap list emptied; live slice identical; ratio measured

## Description
After .2. `feature/activity/realization.rs` becomes: `#[rpc(...)]`/`#[environmental(...)]` on the `Action` enum, `#[entity(...)]`, `#[read(...)]`, the `(status, run_state) -> Phase` table derive replacing `phase.rs`, field rules on claimed fields, one filler per action, per-action evidence predicates (completed result, failure content, cancel outcome). `api.rs`, `api_transport.rs`, `probe_gaps.rs` and the hand-written projection go; the probe inventory agrees with the previous one except the cases the opaque rule removes, listed. The live slice (complete, retry-then-complete, fail, cancel-before-start under a hold, cancel-after-completion, host refusal) passes through `umpire explore` with identical judgments; guest rebuilt. Record per-feature non-Model non-test lines before and after and the shared layer separately; target at most twice the Model.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
