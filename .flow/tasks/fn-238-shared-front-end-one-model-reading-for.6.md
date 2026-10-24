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
Gate not met; compiled stays the default, per the task. Landed (merge b22ae08cd1): `#[machine]` generates one callback per claim (no claim cap; expansion snapshots byte-identical), interpreted fields are read directly (no per-evaluation cache), and each match guard is evaluated once (first-match semantics preserved). With `Interpreted::new` counted as every run pays it, on a loaded host: Kettle per-pair median 1.94x (p75 2.01x), wide shape 1.77x (p90 3.3x). The sets half of the gate (at most 25% above Option A) cannot be measured: no set construct is admitted; the owner has not accepted deferring it. The reviewed switch and its follow-ups are preserved on local branch `work/fn2386-interpreted-switch` (be59562661, 6deb7dcb9d). Stop record in DESIGN.md (a5d25c1e17).

Review: fresh-context round (Opus, same family) NEEDS_WORK on the switch (gate not met, wide ratio unstable, benchmark omitted setup cost, switch coverage gaps); commits 1-3 judged correct and landable. Follow-ups addressed on the preserved branch.

Gates: worktree default `make model-check` rc 0 (831 s) after the follow-ups; all 800 protected files byte-identical. Not run, owed: `UMPIRE_BACKENDS=require make model-time` on the merged tip (wrap-up under the owner's usage limit, 2026-10-24); the guard-fallback and claim-count tests from 6deb7dcb9d are on the preserved branch, not on `umpire`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: db9df02bb0, e9f71b00ae, 39cdae2691, b22ae08cd1, a5d25c1e17
- Tests: make model-check (worktree)
- PRs: