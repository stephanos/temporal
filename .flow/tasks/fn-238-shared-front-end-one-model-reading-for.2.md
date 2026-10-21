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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
