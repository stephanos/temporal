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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
