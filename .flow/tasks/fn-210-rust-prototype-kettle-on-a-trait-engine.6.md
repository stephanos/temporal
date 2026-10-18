---
satisfies: [R6]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.6 Kettle in surface C with its test; parity with the Scala suite

## Description
**Recut 2026-10-18 (R8, golden files):** Kettle in surface C must produce exactly the same `tests/fixtures/kettle/expected/` goldens as the trait fixture from .2; that equality is the parity test.

Write `umpire/samples/src/kettle.rs` in surface C (from `options/Kettle.attrs.rs`), and `tests/kettle.rs` asking the Scala suite's questions: explore counts, verify_all verified, `boils` verified, `!beeping` refuted. Record any deliberate deviation from the sketch.

**Files:** model2/umpire/samples/src/{lib.rs,kettle.rs}, model2/umpire/samples/tests/kettle.rs.

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
