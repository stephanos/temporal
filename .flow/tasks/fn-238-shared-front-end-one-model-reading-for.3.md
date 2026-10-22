---
satisfies: [R2, R3]
---
# fn-238-shared-front-end-one-model-reading-for.3 Macros read through umpire-front; one diagnostic per unsupported form

## Description
Design: `model/design/shared-front-end.md`, Migration row .3.

Replace the macro readers listed in the design's duplication table (about 1,024 lines) with `umpire-front`'s structure; macros keep generation only. Settle the two drifts (State struct name; action inputs without a declared World): keep the form native accepts and widen front end and lifter where emission already supports it, otherwise a located refusal. Unify each differing message pair into one front-end text.

From fn-238.2's review: `umpire-front` still holds TLA+/TLC policy moved verbatim in .2 — TLA+ operator naming and symbol fields (`front/src/shape.rs:543`), `reserved()` `__umpire_*` names (`shape.rs:557`) and refusals worded around TLC's 32-bit limit (`body/typer.rs:21`, `body/vector.rs:173`, `body/progress.rs:187`). Before the macros consume front, move emission-specific naming back to the lifter (or behind an emission pass) and word front refusals in Model terms, so native-only Models never see TLC-worded errors.

Also fix fn-227.8's review follow-ups, which live in this admission code: m1 fairness-name collisions (`lift/src/names.rs:182,198` — prefix check must include generated names); m2 a single-edit fairness readmission control in `action_inputs/tests.rs`; m3 record that the bounded no-fairness control is pinned by the golden (or make it discriminate); m4 refusal tests in `lift/tests/inputs.rs` assert message substrings so each mutation hits its intended refusal.

Follow the preservation plan in `model/design/shared-front-end.md` (Migration): baseline `.flow/tmp/fn238-baseline.sha256` over every golden, expansion snapshot, trybuild `.stderr`, `lift.snap.txt` and guest bytes before .2; only listed differences allowed; scratch IR dump compared across .2-.4; TLC agreement in every Full receipt; typed replay stays on compiled `checked_step`; never a global `model-update`. Gate order per the conductor brief: Quick before review, Full after.

## Acceptance
- [ ] Macros consume `umpire-front`; no independent attribute/schema/domain parsing remains in `dsl/macros`.
- [ ] One diagnostic per unsupported form, identical for both consumers; only the listed trybuild `.stderr`/`lift.snap.txt` messages change, each justified.
- [ ] Expansion snapshots, checker, TLA+/cfg and TLC goldens byte-identical.
- [ ] fn-227.8 follow-ups m1-m4 resolved.

## Done summary
Macros read Models only through `umpire-front`: new `umpire_front::declare` (attributes and order, registration, State/World/actor schema, starts, input domains, Finite, integer widths, `&World`) and `umpire_front::path` (`path!` grammar, progress validation); `machine/registration.rs` and the darling dependency deleted; the lifter calls the same readers and its duplicate checks are gone. TLA+ naming moved back to `umpire-lift` (`names.rs`); front refusals worded in Model terms. Drifts: any State struct name is accepted (TLA+/cfg identical after a rename, tested); inputs without a declared World stay a located lift refusal. fn-227.8 follow-ups m1-m4 done (fairness-name collision refused, fairness-only readmission control, bounded no-fairness pin, input refusals assert messages). Only new golden: `machine_progress_gap.stderr`; all other goldens and the IR dump unchanged except the newly admitted `progress_bounds` fixture.

Lines: dsl/macros 3,306 -> 2,153; front 3,103 -> 4,253; lift +246 (net about +240): the design's lifter duplication column mostly counted lowering that rustc does natively, so .3 had little to merge. Guest refreshed (dbd9221ef2, 3,332,298 bytes) because the macro crate's lockfile closure changed. Review: SHIP (.flow/tmp/fn2383-review.md); follow-ups m1-m3 and nits carried into .4. Gates: Quick default/required (207); `UMPIRE_BACKENDS=require make model-check` 282s; `make model-time` 51.307s gate / 8.822s inner loop; slice 21s; gaps 358s.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1dc089a8ce, dbd9221ef2
- Tests: UMPIRE_BACKENDS=require make model-check, make model-time, make umpire-check-model-slice, make umpire-check-model-gaps
- PRs: