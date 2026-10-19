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
R1 delivered: every shared fixture is `{model.rs, tests.rs, expected/}`.

- `testing/src/discovery.rs`: roots declare their inputs (`MODEL` for synthetic control roots in checker/dsl, `MODEL_AND_TESTS` for `umpire/tests/fixtures` and `temporal/features`). `model.rs` compiles as `<name>`, `tests.rs` as `<name>_tests`; `run` comes from `tests.rs`. A fixture missing `tests.rs` fails the build naming its folder (control in `testing/tests/discovery.rs`). Plain lowercase folder names map to themselves; others (and names starting `fixture_`, ending `_tests`, or keywords) keep the escaped `fixture_` form, so names stay collision-free.
- The golden hooks (`fixture!` and the hand-written `run`s of deep_chain, kettle_progress, sample_kettle_trait, activity) moved into `tests.rs`; `model.rs` holds only the Model. State struct fields (and the world/action types tests.rs constructs) are `pub`. Dependent fixtures re-export their Model with `pub use crate::<dep>::...`.
- Lifter: `Lift::source` reads hooks from a sibling `tests.rs` (supplemental claims, selected mode, constructor check), with diagnostics located in `tests.rs`; dependencies resolve plain `crate::<name>` modules; world paths resolve in a generated `fixture_lifted::<name>` module.
- README/DESIGN updated (testing layout, Kettle size table).

Deviations:
- "Goldens unchanged": all behaviour and TLA+/cfg goldens are byte-identical. Two lift refusal goldens changed location only: `deep_chain` model.rs:32:5 -> 28:5 (test-only imports left model.rs) and `progress_dead_end` model.rs:46:1 -> tests.rs:3:1 (the refused constructor now lives in tests.rs).
- Also applied to the Temporal feature root `temporal/features/activity` (a shared Model with goldens); its unit tests moved with its hook into `tests.rs`. Synthetic checker/DSL control roots keep the hook in `model.rs` (their model.rs is often only a hook), with their escaped module names unchanged.

Gate: `make model-check` green.

`make model-time` (warm):
```
timing-controls 0.015s, capability-controls 0.081s, source-boundary 0.088s, protobuf-freshness 0.054s,
temporal-protobuf-freshness 0.059s, fmt 0.212s, clippy 0.091s, docs 0.130s, tests 2.037s,
wasm32-wasip1 0.048s, core-wasm-clippy 0.065s, wasm-guest 0.059s, go-host 1.143s, temporal-go-host 5.823s
inner loop (edit+test) 2.222s
gate total 10.111s (soft target 30s)
inner loop 2.222s (soft target 10s)
```
No budget warnings.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ee3c5ee85a898e0b9925620ef8015edb88589ecc
- Tests: make model-check, make model-time
- PRs: