---
satisfies: [R1]
---
# fn-213-modelsuite-in-rust-examples-and.1 Fixture layout: tests.rs beside model.rs, golden hook moves, pub fields

## Description
Every shared fixture becomes `umpire/tests/fixtures/<name>/{model.rs, tests.rs, expected/}`. `build.rs` declares both files as modules (`<name>` and `<name>_tests`); the `umpire_testing::fixture!` golden hook moves from `model.rs` into `tests.rs`; `model.rs` holds only the Model; the state struct's fields are `pub` (a sibling module cannot see private fields). Goldens do not change. A fixture without `tests.rs` fails the run naming it.

**Prerequisite:** fn-210 done (the attribute Kettle of fn-210.6 is the `model.rs` this moves around). Runs alone: it moves paths across every shared fixture.

**Touches:** `model/umpire/tests/**`, `model/umpire/wasm/**` (the `#[path]` include if it changes), `model/README.md`.

**Feedback loop (R30):** paste `make model-time` into the done summary.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
