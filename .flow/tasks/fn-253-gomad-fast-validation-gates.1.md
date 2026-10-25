---
satisfies: [R1, R2]
---
# fn-253-gomad-fast-validation-gates.1 Inner loop: make check-fast and dev profile with optimised dependencies

## Description
Build the inner loop: R1 (`make check-fast`) and R2 (dev profile with optimised dependencies).

1. Record the baseline first: incremental wall time after touching one file in `gomad/src` and one in `wasmhost/src`, for `cargo test --release -p gomad --no-run` (3 min 11 s on 2026-10-25), `make lint-rust` and `cargo check`.
2. Add `[profile.dev.package."*"] opt-level = 3` and keep the workspace crates at the dev default.
3. Run the `gomad` and `gomad-wasmhost` test suites under the dev profile. Any test that depends on the profile (timing, overflow, fuel counts) is fixed or explicitly marked release-only with its reason.
4. Add `make check-fast BASE=<rev>` as a transition shim to the standalone mise entrypoint. It maps changed paths to workspace packages via `cargo metadata` and runs `cargo check`, a lint phase covering the complete supported feature matrix with `clippy --all-targets -D warnings`, `cargo fmt --check`, nextest, and separate doctests for those packages. Reverse dependents are tested too; when the mapping can't decide, it falls back to the whole workspace. Preserve the intentional invalid-feature `compile_error!` guards; the spec's implementation clarification replaces the impossible literal `--all-features` invocation with all 9 host and 7 engine modes from one shared Rust table. Include engine `reference-helper` golden coverage when the engine is selected.
5. Record the after numbers.

There is no dependency on fn-242: this touches only the workspace `Cargo.toml` and the Makefile. Coordinate with open fn-242 worktrees on the Makefile.

## Acceptance
- `make check-fast` exists. After a one-file edit it finishes in about 30 s or less of incremental time on the 12-core host, with before and after numbers in `.flow/artifacts/fn-253/inner-loop.md`.
- A mutation in each workspace crate (a failing assertion, a clippy violation, a formatting break) makes `check-fast` fail, and a change in a dependency crate also runs its dependents' tests.
- The dev-profile test suites pass; every release-only test lists its reason.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
