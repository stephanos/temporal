---
satisfies: [R2, R3, R4]
---
# fn-238-shared-front-end-one-model-reading-for.4 Embed the admitted Model from the macro; lifter emits from the compiled Model

## Description
Design: `model/design/shared-front-end.md`, Migration row .4. Starts with a spike: located diagnostics inside the macro on Rust 1.99 (fallback `quote_spanned!(span=> line!())`).

The macro emits `__umpire_lifted()` (not on `wasm32`) and the source identifier set. The lifter emits from the compiled Model and same-process values. Delete the `model.rs` re-read, the canonical-import checks, the tautological parts of `tree.rs:65-468` and the build-script source path; keep the `.cfg` range and value-type checks and the `tests.rs` hook read. If guest bytes change, refresh in a separate commit with slice and gap gates.

Follow the preservation plan in `model/design/shared-front-end.md` (Migration): baseline `.flow/tmp/fn238-baseline.sha256` over every golden, expansion snapshot, trybuild `.stderr`, `lift.snap.txt` and guest bytes before .2; only listed differences allowed; scratch IR dump compared across .2-.4; TLC agreement in every Full receipt; typed replay stays on compiled `checked_step`; never a global `model-update`. Gate order per the conductor brief: Quick before review, Full after.

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
