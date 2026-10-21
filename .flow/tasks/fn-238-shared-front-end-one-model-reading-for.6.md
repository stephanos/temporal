---
satisfies: [R2, R3]
---
# fn-238-shared-front-end-one-model-reading-for.6 Interpreted checker by default, behind a measured gate (Option B, step 2)

## Description
Design: `model/design/shared-front-end.md`, Migration row .6.

Switch `Checker` to `Interpreted<M>` for admitted Models; refused Models stay compiled; guest and replay stay compiled; the .5 differential run stays in the gate. Gate (conductor decision 2026-10-21, accepting the design's proposal): .5 green, whole-checker slowdown at most 2x on Kettle and the wide shape, and per-construct cost on the first Quint-parity construct (sets) at most 25% above Option A. If the gate fails, stop, record why, and leave the compiled checker as default.

Follow the preservation plan in `model/design/shared-front-end.md` (Migration): baseline `.flow/tmp/fn238-baseline.sha256` over every golden, expansion snapshot, trybuild `.stderr`, `lift.snap.txt` and guest bytes before .2; only listed differences allowed; scratch IR dump compared across .2-.4; TLC agreement in every Full receipt; typed replay stays on compiled `checked_step`; never a global `model-update`. Gate order per the conductor brief: Quick before review, Full after.

## Acceptance
- [ ] Gate measured and recorded; switch made only if it passes.
- [ ] Every golden byte-identical; typed replay unchanged.
- [ ] `make model-time` within soft targets.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
