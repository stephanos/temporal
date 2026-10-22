---
satisfies: [R2]
---
# fn-238-shared-front-end-one-model-reading-for.2 umpire-front: move the lifter's admission into a shared crate, no behaviour change

## Description
Design: `model/design/shared-front-end.md`, Migration row .2 (Option A, owner-accepted via the active milestone goal 2026-10-21).

New crate `model/umpire/front/`. Move `shape.rs`, `body/*`, `expr.rs`, `syntax.rs` and the declaration parts of `tree.rs` out of `umpire-lift` without behaviour change. `Diagnostic` carries line, column and message; consumers format the file. The lifter calls `umpire_front::read`. Write the fn-238 baseline manifest first.

Follow the preservation plan in `model/design/shared-front-end.md` (Migration): baseline `.flow/tmp/fn238-baseline.sha256` over every golden, expansion snapshot, trybuild `.stderr`, `lift.snap.txt` and guest bytes before .2; only listed differences allowed; scratch IR dump compared across .2-.4; TLC agreement in every Full receipt; typed replay stays on compiled `checked_step`; never a global `model-update`. Gate order per the conductor brief: Quick before review, Full after.

## Acceptance
- [ ] `umpire-front` owns admission; `umpire-lift` consumes it.
- [ ] Every golden byte-identical, including all `lift.snap.txt`; lift tests pass unchanged or move with their code.
- [ ] Baseline manifest and IR dump recorded; line counts reported.

## Done summary
New crate `umpire-front` owns admission (shape, body, expr, syntax, declaration tree, symbol allocation), moved mechanically from `umpire-lift`; the lifter calls `umpire_front::read` and keeps file reading, hook discovery, name validation, binding and emission. Lift 4,884 -> 1,871 lines, front 3,085 (+72 glue). All 719 baselined goldens identical except the refreshed guest; IR dump identical before/after. Review fixes: no `DerefMut` on `umpire_lift::Lifted`, hook-refusal file test, documented `front::Lifted` fields.

Guest refreshed (9ba8166c55, 3,332,185 bytes) because the guest identity hashes the whole Cargo.lock; follow-up filed as fn-238.8. Review: NEEDS_WORK only for the stale guest, then fixes applied (.flow/tmp/fn2382-review.md); front still holds TLA+/TLC-worded policy, carried into .3. Gates: Quick default/required; `UMPIRE_BACKENDS=require make model-check` 509s; `make model-time` 119.6s gate / 19.6s inner loop, measured while the fn-227.12 lane ran concurrently (fn-227.14's idle receipt was 50.5s / 7.9s; re-measure idle); slice 95s; gaps 374s.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fc2acec3cd, 9fceee9eca, 9ba8166c55, ba14ec238a
- Tests: UMPIRE_BACKENDS=require make model-check, make model-time, make umpire-check-model-slice, make umpire-check-model-gaps
- PRs: