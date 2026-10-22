---
satisfies: [R2, R3, R4]
---
# fn-238-shared-front-end-one-model-reading-for.4 Embed the admitted Model from the macro; lifter emits from the compiled Model

## Description
Design: `model/design/shared-front-end.md`, Migration row .4. Starts with a spike: located diagnostics inside the macro on Rust 1.99 (fallback `quote_spanned!(span=> line!())`).

The macro emits `__umpire_lifted()` (not on `wasm32`) and the source identifier set. The lifter emits from the compiled Model and same-process values. Delete the `model.rs` re-read, the canonical-import checks, the tautological parts of `tree.rs:65-468` and the build-script source path; keep the `.cfg` range and value-type checks and the `tests.rs` hook read. If guest bytes change, refresh in a separate commit with slice and gap gates.

Follow the preservation plan in `model/design/shared-front-end.md` (Migration): baseline `.flow/tmp/fn238-baseline.sha256` over every golden, expansion snapshot, trybuild `.stderr`, `lift.snap.txt` and guest bytes before .2; only listed differences allowed; scratch IR dump compared across .2-.4; TLC agreement in every Full receipt; typed replay stays on compiled `checked_step`; never a global `model-update`. Gate order per the conductor brief: Quick before review, Full after.

**From fn-238.3's review (fold into this task's `shape.rs` rework):** m1 refuse in `declare::registrations` the forms that still get two messages (extra parameters on invariant/assumption/progress methods, `after =` naming an input variant, `path![..].within(n)`; see `body/claim.rs:11-14`, `body/progress.rs:18-20,30-32,112-114,153-155`); m2 remove the ~95 lines of remaining overlap inside `front` (`shape.rs:116-144` local enums, `:410-423` `#[domain]` re-parse, unreachable `body/progress.rs:26-28,198-211`, `typer.rs:55-68` width table, `shape.rs:185-198,208` method-kind re-detection); m3 replace the hard-coded `__umpire_outcome` in `front/src/body/claim.rs:51` with a neutral name renamed in lift's `names::assign`; nits: the unreachable error in `dsl/macros/src/state.rs:21-28`.

## Acceptance
- [ ] Lifter no longer reads `model.rs`; emission uses the embedded admitted Model.
- [ ] TLA+/cfg, TLC agreement, replay, `lift.snap.txt` and guest bytes byte-identical (or guest refreshed separately with slice/gaps).
- [ ] Measured deletion against the -850 to -1,600 estimate; `make model-time` recorded.

## Done summary
The `#[machine]` macro embeds the admitted Model (new `umpire-ir` crate; `Lifts` trait, off wasm32, never checked in) and the lifter emits TLA+ from the compiled Model with same-process World, starts and catalog. Deleted: `lift/src/build.rs` and `Lift::source`, the `model.rs` re-read (kept only to locate refusals when no `#[machine]` is embedded: kettle_cut, kettle_progress, progress_dead_end), tautological binding checks, the dead action-input domain admission (Models with computed domains now lift), and the naming source walk. Kept: `.cfg` range/value-type checks and the `tests.rs` hook read (hook claims are lowered against the tree; documented that they cannot call Model helpers). Shadowing imports of the Umpire macros are refused; the tree printer uses one `record!` macro (277 lines); declarations are read once per `#[machine]`; binding outputs live in the lifter. Spike: located spans work inside the macro on Rust 1.99.1.

Goldens: TLA+/cfg, `lift.snap.txt`, expansion, checker and trybuild unchanged; three new trybuild controls. Guest refreshed (0d3179714f) for the umpire-ir lockfile closure. Lines: core crates 8,297 against 8,190 before fn-238.2 (design estimated -850 to -1,600; .3 moved rather than merged, body lowering and emission stay, embedding costs about 420). Review: NEEDS_WORK (dead domain admission, import shadowing), then SHIP (.flow/tmp/fn2384-review.md, fn2384-review2.md); the qualified `path!` match (`front/src/body/progress.rs:132`) is carried into fn-227.11. Gates: Quick default/required (221); `UMPIRE_BACKENDS=require make model-check` 247s; `make model-time` 47.6s gate / 7.3s inner loop; slice 7s; gaps 359s.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: df29486f10, 0d3179714f, 0bbeb1ff35
- Tests: UMPIRE_BACKENDS=require make model-check, make model-time, make umpire-check-model-slice, make umpire-check-model-gaps
- PRs: