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
Umpire's Rust crates now live under model2/umpire, and each owning crate discovers its fixtures from tests/fixtures/<name>/{model.rs,expected/}. The dev-only umpire-testing crate owns rendering, comparison, updates and discovery; samples::kettle exports the existing trait world and its budget.

R21 renames workspace paths, crate names, imports, documentation and the Makefile's WASI package flags. R22 and R23 place 15 synthetic fixtures in checker and three Kettle fixtures in samples, including the retained adapted trait compilation sketch. Dependency-free discovery source is shared by the two build scripts to avoid a checker build-dependency cycle. The compile-only fixture reserves expected/.gitkeep for its empty output; arbitrary orphan files still fail.

baseline: green from make model2-check before implementation. The first wrapper invocation observed the parent directory and was inconclusive; the corrected invocation ran in temporal and passed before edits. The pre-edit base is 05b0bbd34d5a394eb18306d70b14eeed98182349. Owner checkpoint a5040fa23b and authorized concurrent Flow planning changes are included in the range.

Verification passed make model2-check, targeted checker/samples/testing tests, the unchanged 256-world oracle, release-mode checker fixtures including caught panic and overflow, and cargo deny check bans. The final full-gate log is .flow/tmp/fn210-10-closure-gate.log; its receipt is .flow/tmp/green-receipts/6138dd4b-unittest.json. Other logs use .flow/tmp/fn210-10-*.

The discovery regression failed before validation and passed after it. Real cargo check --tests controls removed model.rs and expected/ independently in both checker and samples; all four refused with exit 101 and named the missing fixture. The orphan regression failed when stale detection was disabled, then passed after restoration. A deliberate golden mismatch generated an insta pending snapshot; cargo insta accept restored the original payload through the supported workflow. Byte-manifest comparison accounts for all 56 original payloads exactly once with matching SHA-256 hashes. The 56 metadata files change only their source path. Scala files are unchanged.

stage: impl-review - ran [2026-10-18T20:33:35Z..2026-10-18T20:35:29Z] (SHIP; three first-round lenses; codex:gpt-6.1-sol:high; same-family review)
Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)

Review receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.10.json
The reviewers' fresh Cargo gate attempts were inconclusive because their read-only sandbox could not open Cargo's build lock. The worker's fresh full gate passed afterward. Memory capture was skipped because all three first-round reviews returned SHIP without findings.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a5040fa23b30e8f3464e13e60e2dcad64c3cf663, 6138dd4b0d6996951111b4a8e9ccff37b8f2e2cb
- Tests: make model2-check, cd model2 && mise exec -- cargo test -p umpire-checker -p samples -p umpire-testing, cd model2 && mise exec -- cargo test --release -p umpire-checker --test fixtures, cd model2 && mise exec -- cargo deny check bans, node .flow/tmp/fn210-10-verify-payloads.js, cd model2 && mise exec -- cargo insta accept --workspace --snapshot umpire/checker/tests/fixtures/shared_starts/expected/report.snap, Real cargo check --tests controls: samples/kettle and checker/choices missing model.rs and expected/ each refused with exit 101; logs .flow/tmp/fn210-10-missing-*, Discovery red-to-green and orphan-control mutation red-to-green; logs .flow/tmp/fn210-10-{discovery-red,discovery-green,orphan-red,focused}.log, Retired framework crate-name search has no matches; Scala model diff is empty
- PRs: