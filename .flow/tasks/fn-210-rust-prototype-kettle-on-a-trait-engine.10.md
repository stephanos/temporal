---
satisfies: [R21]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.10 Rename model2/framework/ to model2/umpire/ and framework-* crates to umpire-*

## Description
Mechanical rename after .1 and .2 land (R21). `git mv`-equivalent move of `model2/framework/` to `model2/umpire/`; crate names `framework-dsl`, `framework-dsl-macros` (if present), `framework-checker` to `umpire-dsl`, `umpire-dsl-macros`, `umpire-checker` in every `Cargo.toml`, the workspace members and `[workspace.dependencies]`; Rust paths `framework_dsl`/`framework_checker` to `umpire_dsl`/`umpire_checker`; the Makefile's model2 targets (`-p` flags in the wasm step); `model2/README.md`; insta snapshot names if they embed crate names (re-run `make model2-update` and require an empty diff apart from renames). Do not touch `model/framework/` (Scala).

**Files:** model2/** (move), model2/Cargo.toml, model2/umpire/*/Cargo.toml, Makefile, model2/README.md.

**Quick commands:** `make model2-check`; `grep -rn -E 'framework[-_](dsl|checker|lift|tlc|wasm)' model2` returns nothing

## Acceptance
- `model2/umpire/{dsl,checker,samples}` exist and `model2/framework/` does not.
- `make model2-check` passes; goldens unchanged except for renamed paths.
- No `framework-*`/`framework_*` crate name remains in model2.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
