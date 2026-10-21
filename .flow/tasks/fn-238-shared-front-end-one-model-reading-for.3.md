---
satisfies: [R2, R3]
---
# fn-238-shared-front-end-one-model-reading-for.3 Macros read through umpire-front; one diagnostic per unsupported form

## Description
Design: `model/design/shared-front-end.md`, Migration row .3.

Replace the macro readers listed in the design's duplication table (about 1,024 lines) with `umpire-front`'s structure; macros keep generation only. Settle the two drifts (State struct name; action inputs without a declared World): keep the form native accepts and widen front end and lifter where emission already supports it, otherwise a located refusal. Unify each differing message pair into one front-end text.

Also fix fn-227.8's review follow-ups, which live in this admission code: m1 fairness-name collisions (`lift/src/names.rs:182,198` — prefix check must include generated names); m2 a single-edit fairness readmission control in `action_inputs/tests.rs`; m3 record that the bounded no-fairness control is pinned by the golden (or make it discriminate); m4 refusal tests in `lift/tests/inputs.rs` assert message substrings so each mutation hits its intended refusal.

Follow the preservation plan in `model/design/shared-front-end.md` (Migration): baseline `.flow/tmp/fn238-baseline.sha256` over every golden, expansion snapshot, trybuild `.stderr`, `lift.snap.txt` and guest bytes before .2; only listed differences allowed; scratch IR dump compared across .2-.4; TLC agreement in every Full receipt; typed replay stays on compiled `checked_step`; never a global `model-update`. Gate order per the conductor brief: Quick before review, Full after.

## Acceptance
- [ ] Macros consume `umpire-front`; no independent attribute/schema/domain parsing remains in `dsl/macros`.
- [ ] One diagnostic per unsupported form, identical for both consumers; only the listed trybuild `.stderr`/`lift.snap.txt` messages change, each justified.
- [ ] Expansion snapshots, checker, TLA+/cfg and TLC goldens byte-identical.
- [ ] fn-227.8 follow-ups m1-m4 resolved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
