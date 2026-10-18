---
satisfies: [R21, R22, R23]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.10 Rename framework to umpire; move fixtures into each owning crate's tests/fixtures; extract umpire-testing

## Description
Mechanical rename after .1 and .2 land (R21). `git mv`-equivalent move of `model2/framework/` to `model2/umpire/`; crate names `framework-dsl`, `framework-dsl-macros` (if present), `framework-checker` to `umpire-dsl`, `umpire-dsl-macros`, `umpire-checker` in every `Cargo.toml`, the workspace members and `[workspace.dependencies]`; Rust paths `framework_dsl`/`framework_checker` to `umpire_dsl`/`umpire_checker`; the Makefile's model2 targets (`-p` flags in the wasm step); `model2/README.md`; insta snapshot names if they embed crate names (re-run `make model2-update` and require an empty diff apart from renames). Do not touch `model/framework/` (Scala).

**Files:** model2/** (move), model2/Cargo.toml, model2/umpire/*/Cargo.toml, Makefile, model2/README.md.

**Quick commands:** `make model2-check`; `grep -rn -E 'framework[-_](dsl|checker|lift|tlc|wasm)' model2` returns nothing

**Also (R22, owner 2026-10-18):** in the same step, restructure tests. Create `umpire/testing/` (crate `umpire-testing`, dev-only) holding the runner now in `samples/tests/cases/mod.rs` and `samples/tests/kettle.rs` (render, compare, update flag, stale detection, discovery). Move every synthetic case to `umpire/checker/tests/fixtures/<name>/` with its Model extracted into `model.rs` beside its `expected/`; add `checker/build.rs` generating the `#[path]` modules and `checker/tests/fixtures.rs`. Keep only Kettle (and `kettle_cut`, the trait fixture) in `samples`. Move `orphan_control` to whichever crate it pins. Goldens move byte-identically.

**Also (R23, owner 2026-10-18):** the target folder is `tests/fixtures/<name>/` in each crate (not `cases/`), the runner target `tests/fixtures.rs`; Kettle's fixture gets a one-line `model.rs` naming `samples::kettle` and its budget. Sources to move today: `samples/cases/*` and the Models inline in `samples/tests/cases/mod.rs`; `samples/tests/fixtures/kettle_trait.rs` becomes `samples/tests/fixtures/kettle_trait/model.rs`.
## Acceptance
- `model2/umpire/{dsl,checker,samples}` exist and `model2/framework/` does not.
- `make model2-check` passes; goldens unchanged except for renamed paths.
- No `framework-*`/`framework_*` crate name remains in model2.
- `umpire/checker/tests/fixtures/` holds the synthetic cases, each with `model.rs` and `expected/`; `samples/tests/fixtures/` holds only Kettle's.
- `umpire-testing` owns the runner; `samples/tests/fixtures.rs` and `checker/tests/fixtures.rs` are a few lines each.
- A case folder missing `model.rs` or `expected/` fails the build naming it.
- Golden contents unchanged (byte-identical after the move).
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
