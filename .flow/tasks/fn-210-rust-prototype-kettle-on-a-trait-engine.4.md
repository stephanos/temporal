---
satisfies: [R5]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.4 framework-dsl-macros: derive(State) with #[starts], derive(Action) with #[internal]

## Description
New proc-macro crate `framework/dsl/macros` re-exported from `framework-dsl`: `#[derive(State)]` adds Clone, Eq, Hash and `fn starts() -> Vec<Self>` from `#[starts(expr)]` field attributes (cartesian over annotated fields, defaults for the rest); `#[derive(Action)]` yields the catalog, names and `internal()` from `#[internal]` variants. Use `syn`, `quote`, `darling`; pin messages with `trybuild`.

**Files:** model2/framework/dsl/macros/{Cargo.toml,src/lib.rs}, model2/framework/dsl/Cargo.toml, tests/ui/.

**Quick commands:** `mise exec -- cargo test -p framework-dsl`

## Acceptance
- Kettle's struct and enums derive and produce 81 starts and 4 actions (2 internal).
- trybuild cases for a non-range `#[starts]` and a non-unit `#[internal]` variant.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
