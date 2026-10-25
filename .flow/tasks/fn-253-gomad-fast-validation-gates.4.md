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

## Acceptance
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
