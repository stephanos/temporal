---
satisfies: [R5]
---
# fn-253-gomad-fast-validation-gates.4 Standalone tools/gomad: mise tasks and cargo entrypoints replace the Makefiles

## Description
R5: make `tools/gomad` standalone. Move gomad's targets out of `temporal/` and replace `tools/gomad/Makefile` with mise tasks and cargo entrypoints.

Current coupling, 2026-10-25:
- `tools/gomad/Makefile` `lint-code` runs `$(MAKE) -C ../.. lint-code-gomad`. That is temporal's `cmd/tools/lintcode` with temporal's `.github/.golangci.yml`.
- `temporal/Makefile` defines `lint-code-gomad` and `lint-code-gomad-wasm`, identical recipes, both gomad-only.
- `validate-wasm-determinism` runs `cd ../.. && python3 -m unittest discover -s tools/gomad/rust/driver/tests`. That goes away with fn-242's Python deletion; until then, run it from `tools/gomad` with a relative `-s`.
- `test-wasm-slim-refusals` runs `cd ../.. && go test ./common/archiver/provider ./temporal ./tests/testcore`. This is gomad exercising its subject, so it stays, but it reaches the target tree through an explicit `GOMAD_TARGET_REPO` (defaulting to `../..`), not an implicit `cd`.
- `tools/gomad/internal/backends/wasm/Makefile` is a second, nested Makefile with `lint-rust`, `fmt-check` and test targets; CI calls it (`make -C tools/gomad/internal/backends/wasm lint-rust`). Fold it into the same entrypoints and delete it.
- `.github/workflows/gomad.yml` calls `make -C tools/gomad …`, plus `make … lint-code-gomad` at the temporal root. It must stay under `temporal/.github` (GitHub requires that), but it calls only `tools/gomad` entrypoints.
- The cycle stages `lint-module` and `lint-root` run `make lint-code-gomad` and `make lint-code-fast` at the temporal root (`static_stages.py:247`; the Rust port binds `make` in `cycle/preparation/composition/static_inputs.rs`). Repoint `lint-module` at the new gomad lint. Whether gomad's cycle should still run temporal's own `lint-code-fast` (`lint-root`) is an owner question: keep it only if gomad changes code under `temporal/`, such as the `gomad_wasm_slim` build tags.

Shape:
- **`tools/gomad/mise.toml`** pins every tool a gate uses: rust 1.99.1 (already pinned), go 1.27.1, golangci-lint, cargo-deny, cargo-nextest, actionlint if a gomad stage needs it.
- **`[tasks]`** cover generate, validate, build, lint (go + rust + deny), test, check-fast, qualify and parity. Each task is a one-line command or a `gomad` verb; new logic goes in Rust, not shell or Python (house rule).
- **Gomad's own `.golangci.yml`**, in `tools/gomad`, starts as a copy of the rules temporal's lintcode applies to `-module tools/gomad`. The base-rev filtering that `lintcode` does is replaced by golangci's `--new-from-rev`, or by a small `gomad` lint verb if the package pre-filter matters for speed.
- **`rust/.cargo/config.toml`** aliases cover pure-cargo commands (this subsumes fn-240.13's alias item).
- **Delete** `lint-code-gomad` and `lint-code-gomad-wasm` from `temporal/Makefile`. temporal's prune paths that exclude `tools/gomad` from its own lint and format stay; they are how temporal ignores gomad.

Ordering:
- The cycle's leaf argv is pinned by fn-242 R2. So either land this after the fn-242 cutover, or have the owner approve the argv change alongside the G1–G3 draft.
- fn-240.5, fn-240.13 and fn-253.1 name `make lint-rust` / `check-fast`; they use the new entrypoints once this lands.
- Coordinate with open worktrees that edit either Makefile.

### Owner simplification amendment, 2026-10-25

The owner approved these simplifications from a review of the integrated fn-253 branch at `30e19a85be`. Line references are to that commit. Gates get faster, never weaker: drop a command only when another command in the same gate already covers it.

1. **Makefiles.** Delete `tools/gomad/Makefile`, `tools/gomad/internal/backends/wasm/Makefile` and the root `lint-code-gomad-wasm` now. No code calls them: CI uses mise, and the cycle calls only root `lint-code-gomad`/`lint-code-fast` (`cycle/plan/leaf/static_commands.rs:337-339`, `static_stages.py:246`). Move `lint-code-gomad-wasm`'s users in `cmd/tools/lintcode/main_test.go` to `lint-code-gomad`. Root `lint-code-gomad` stays until the cycle argv is repointed. This task owns the removal, not fn-253.2; fix the "fn-253.2 removes the shims" text in `docs/GATES.md` and the Makefile headers.
2. **Aggregates.** Remove from `premerge`/`test-wasm` `depends` any task that is a subset of another task in the same aggregate. The tasks stay runnable on their own:
   - `check-rust` is a subset of `lint-rust`.
   - archcheck runs three times (`test-architecture`, `test-reference-harness`, `test-reference`).
   - `./internal/support/...` and `./internal/model/...` rerun inside `model-corpora`'s `-count=2`.

   Trim `test-reference` to what `model-corpora` does not cover. Confirm once that feature unification does not change what the remaining run compiles. The Rust test subsets are in fn-253.6.
3. **CI** (`.github/workflows/gomad.yml`):
   - Add the slim-seam paths (`common/archiver/provider/provider_cloud*`, `temporal/membership*`, `tests/testcore/flag_sql*`) to both `on.paths` lists. Today changes there never trigger the workflow. This is a coverage fix.
   - Delete the `changes` job (`:35-75`). Its `tools/gomad/*` case matches every triggering path, so `wasm` is always true.
   - Delete `TestCIBackendGatesRoutePaths` (`internal/archcheck/integration_test.go:12+`) with it.
   - Run `validate` in the `wasm-linux` job, so it does not rebuild model corpora and the helper on the uncached `reference-tools-linux` runner (`:154`).
4. **One `lint-rust` task (owner decision).** The mode table in `rust/gate-tool/src/lib.rs:4-25` is the only list of Rust lint modes. `lint-rust` becomes one task, `gomad-gates -- lint-rust`, with a `--plan` printout like check-fast's. Delete:
   - the 16 `lint-rust-*` mise tasks and their `depends` list
   - `graph_tests.rs:61-117`

   check-fast keeps reading the same table. This replaces the visibility clause's "real mise graph" for the lint modes; the `--plan` output is their tree.
5. **gomad-gates subcommands.**
   - Delete the unused `test-engine` and `lint-go`.
   - Delete `oracle-plan` together with the tests that are its only callers (item 6).
   - Replace the `nextest` and `go` argv passthroughs with direct `cargo nextest`/`go` calls in mise tasks.
   - Keep env such as GOWORK, GOENV and GOTOOLCHAIN in `mise.toml` `[env]` only, not also in `main.rs:60-63`.
   - Keep the subcommands that carry logic: `slim-refusals`, `replay-plan`, `model-corpora`, `build`/`helper` publish, `lint-mutants`, `lint-errors`, `lint-golangci` and `lint-rust`.
6. **Tests of build config.** Delete:
   - `TestMiseTasksMatchTheirOwnership` (`architecture_test.go:354-369`)
   - `TestDefaultMiseTasksKeepSharedValidation` and its `mise --dry-run`/`cargo run` helpers (`defaultset_test.go:128-170`)
   - `TestNativeWASMTaskRetainsStockGoOracleSelection` (`wasm_preservation_test.go:99-110`)
   - the description/graph-membership test in `graph_tests.rs`

   mise's own `tasks ls`/`tasks deps` gives that view.
7. **Pin each version once.**
   - Rust: `rust-toolchain.toml` is the single pin. Have mise read it, or keep `mise.toml` as the pin if mise 2026.10.11 cannot.
   - Replace the parity workflow's `cargo +1.99.1` lines (`gomad-behavioral-parity.yml:53,55,67`) with mise tasks.
   - Drop the literal `rust1.99.1-` from the cache keys (`gomad.yml:107,110`); the key already hashes the toolchain file.
   - mise: one pin in `mise.toml`, read by CI.
   - Remove copied tool versions from `docs/GATES.md:43,94-100` and `docs/README.md:23-24`. Link `mise.toml` instead.
   - The README's no-mise fallback is already deleted on `gomad` (c1064a8cc8); keep it deleted when merging.
8. **Cargo aliases.** Delete the `rust/.cargo/config.toml` aliases that repeat mise tasks, including the weaker `lint-all`. This overrides the Shape bullet above.
9. **Root lintcode.** Remove the gomad module mode and archcheck-inventory reading from `cmd/tools/lintcode/main.go` (about `:22-320`). Remove the `make lint-code-gomad` use in `rename_policy_test.go:75` along with the root targets, once the cycle argv no longer needs them.

Only the parked fn-242 driver uses the Python pin, `validate-driver`, `qualify`, `parity` and the parity workflow's Python steps. They leave with the fn-242 cutover, not here. Keep the 16 lint modes; none is provably covered by another, and cargo-hack would add combinations.

## Acceptance

Owner visibility extension, 2026-10-25: the real mise dependency graph and task descriptions provide the simple gate tree. Aggregate groups use actual dependency edges, not a separate hand-maintained list. Nextest supplies Rust test inventory and live results; check-fast supplies its actual changed-package command plan. Each public gate is represented and runnable; ignored/unrun checks cannot appear as passed. Keep this within existing entrypoints, without adding another orchestration or receipt framework.
- Simplification amendment items 1–9 are carried out. Each removed aggregate member has a named surviving command in the same gate that covers it, and the slim-seam paths trigger the workflow.
- A clone with `temporal/Makefile`, `temporal/cmd/tools/lintcode` and `temporal/.github/.golangci.yml` removed, or with the `tools/gomad` subtree copied out alongside a target tree passed through `GOMAD_TARGET_REPO`, runs every gomad mise task green from `tools/gomad`.
- `temporal/Makefile` has no target or variable that names gomad other than the prune paths. `tools/gomad/Makefile` and `tools/gomad/internal/backends/wasm/Makefile` are deleted, or reduced to a documented one-line shim per target if CI or the cycle still needs them during the transition, with a removal task named.
- `.github/workflows/gomad.yml` invokes only `mise run …` (or cargo) inside `tools/gomad`.
- The new gomad golangci config fails on a seeded violation of each rule the old `lint-code-gomad` enforced (mutation fixtures).


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

### Focused entrypoint checkpoint — 2026-10-25

Entrypoint source `2c879eb904` is integrated on `5585b11ccb89464e6ee66f7042115b371a0a1629`.
On that integration, all eight gate-tool tests and 16 check-fast tests passed;
scoped tooling Clippy with `-D warnings`, formatting and both workflow actionlint
checks exited 0. The stock-Go PATH and workspace-relative JUnit rotation fixes
have focused regression coverage.

The production Go lint configuration rejected all ten active golangci rule seeds
and the errortype seed; restored controls passed. Revive's inherited Gomad-wide
exemption was preserved and checked.

A separate Gomad copy with an explicit private target repository ran the real
protocol-generation check and both changed architecture contracts successfully.
The target's Makefile, lintcode and root golangci config were physically absent;
all 68 Mise task definitions validated. The private fixtures were removed after
retaining logs under `/home/agent/.cache/codex-build/fn253-entrypoints-logs/`.

The complete standalone task sweep, broad suites and final spec acceptance remain
pending. These checks supply no qualification result and do not resume fn-242.
