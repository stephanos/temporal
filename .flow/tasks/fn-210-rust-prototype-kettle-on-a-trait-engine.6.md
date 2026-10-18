---
satisfies: [R6]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.6 Kettle in surface C with its test; parity with the Scala suite

## Description
Write `framework/samples/src/kettle.rs` in surface C (from `options/Kettle.attrs.rs`), and `tests/kettle.rs` asking the Scala suite's questions: explore counts, verify_all verified, `boils` verified, `!beeping` refuted. Record any deliberate deviation from the sketch.

**Files:** model2/framework/samples/src/{lib.rs,kettle.rs}, model2/framework/samples/tests/kettle.rs.

**Quick commands:** `mise exec -- cargo test --workspace`

## Acceptance
- All questions pass; counts match the Scala note.
- Sketch and source differ only where the write-up says.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
