---
satisfies: [R29, R30]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.13 Shared end-to-end fixtures: umpire-tests crate, consumers selected by expected files

## Description
**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model2-check` at 30 s or less, warm; paste `make model2-time` output into the done summary; a breach is fixed here or excepted by the owner.

Runs alone, after .12 (R29).

- Create crate `umpire-tests` at `model2/umpire/tests/` (workspace member, `publish = false`, `autotests = false`, one `[[test]] name = "fixtures" path = "fixtures.rs"`), with a `build.rs` that generates the `#[path]` modules from `fixtures/*/model.rs`, and fixtures directly under `model2/umpire/tests/fixtures/<name>/`.
- Move `kettle`, `kettle_trait` and `kettle_cut` from `samples/tests/fixtures/`, and `choices`, `deep_chain`, `edge_checks`, `shared_starts`, `vacuity` from `checker/tests/fixtures/`, with their `expected/` byte-identical. Move `samples/src/kettle.rs` into `umpire/tests/fixtures/kettle/model.rs` (the fixture's own Model, not a re-export) and remove the `samples` crate and folder from the workspace.
- Give `umpire-testing` a consumer registry: each consumer declares the expected file names it owns and renders them; the runner runs a fixture through every consumer whose files it has, fails on an expected file no consumer owns, and on a fixture with none. Register the checker consumer now; TLC (.8) and Apalache (fn-211) register theirs later.
- Update the README's testing and layout sections.

**Files:** model2/Cargo.toml (members), model2/umpire/tests/{Cargo.toml,build.rs,fixtures.rs,fixtures/}, model2/umpire/samples/ (removed), model2/umpire/checker/tests/fixtures/, model2/umpire/testing/src/, model2/README.md.

**Quick commands:** `make model2-check`
## Acceptance
- `model2/umpire/tests/fixtures/` holds the eight shared fixtures; checker-only fixtures remain in `checker/tests/fixtures/`; the `samples` crate and folder no longer exist; Kettle's source is its fixture's `model.rs`.
- Every moved golden is byte-identical.
- An unrecognised expected file, and a fixture with none, each fail naming the folder.
- `make model2-check` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
