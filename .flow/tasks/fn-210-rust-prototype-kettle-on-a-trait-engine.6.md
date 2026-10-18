---
satisfies: [R6, R14]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.6 Kettle in surface C with its test; parity with the Scala suite

## Description
**R14 joined integration:** consume the .5 source-order metadata in checker reporting, if the separate registry hooks otherwise group claims by kind. Add a shared Model with interleaved invariant/postcondition/progress declarations and a report golden proving actual order, while keeping manual-world defaults and dynamic effect failures deterministic. This checker integration belongs here after .3/.5, not in the parallel DSL lane.

**Touches:** `model2/umpire/tests/**`, `model2/umpire/checker/src/**`, `model2/umpire/checker/tests/**`, `model2/DESIGN.md`

**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model2-check` at 30 s or less, warm; paste `make model2-time` output into the done summary; a breach is fixed here or excepted by the owner.

**Location (R29 amended):** there is no `samples` crate; write Kettle in surface C as `umpire/tests/fixtures/kettle/model.rs`, replacing the trait-shaped source there.

**Shared fixtures 2026-10-18 (R29):** end-to-end fixtures live in `umpire/tests/fixtures/<name>/` after .13; their expected files select the consumers (checker, TLC, Apalache).

**Conventions 2026-10-18 (R25):** Kettle in surface C spells through the derived `Spell`; its goldens must stay byte-identical to the trait Kettle's.

**Recut 2026-10-18 (R8, golden files):** Kettle in surface C must produce exactly the same `tests/fixtures/kettle/expected/` goldens as the trait fixture from .2; that equality is the parity test.

Write `umpire/tests/fixtures/kettle/model.rs` in surface C (from `options/Kettle.attrs.rs`), and `tests/kettle.rs` asking the Scala suite's questions: explore counts, verify_all verified, `boils` verified, `!beeping` refuted. Record any deliberate deviation from the sketch.

**Files:** model2/umpire/tests/fixtures/kettle/model.rs and shared fixture runner; checker ordering integration; model2/DESIGN.md for deliberate sketch differences. Keep the separate trait Kettle as the WASM/native reference and preserve its goldens.

**Quick commands:** `mise exec -- cargo test --workspace`
## Acceptance
- `tests/fixtures/kettle/expected/` unchanged when the trait fixture is replaced by the surface-C Kettle.
- Sketch and source differ only where DESIGN.md says.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
