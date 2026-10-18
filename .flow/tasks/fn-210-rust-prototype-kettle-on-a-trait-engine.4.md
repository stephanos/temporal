---
satisfies: [R5]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.4 umpire-dsl-macros: derive(State) with #[starts], derive(Action) with #[internal]

## Description
**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model2-check` at 30 s or less, warm; paste `make model2-time` output into the done summary; a breach is fixed here or excepted by the owner.

**Shared fixtures 2026-10-18 (R29):** end-to-end fixtures live in `umpire/tests/fixtures/<name>/` after .13; their expected files select the consumers (checker, TLC, Apalache).

**Conventions 2026-10-18 (R25, R26):** `derive(State)` and `derive(Action)` also generate `Spell` in the R25 form, using the author's names (R26); replace the hand-written `Spell` impls in DSL-owned fixtures with the derive and require byte-identical goldens. Compile Kettle-shaped state/action declarations in this lane; canonical shared Kettle migration belongs to .6 so this lane does not modify .3's shared fixture surface.

**Touches:** model2/umpire/dsl/build.rs, model2/umpire/dsl/macros/build.rs, model2/umpire/dsl/src/**, model2/umpire/dsl/tests/**, model2/umpire/dsl/macros/src/**, model2/umpire/dsl/macros/tests/**

**Parallel boundary:** shared Cargo wiring and empty macro crate are prepared by .11; no manifests or lockfiles change in this lane.

**Layout (R22):** expansion cases go in `umpire/dsl/macros/tests/fixtures/<name>/`; refusals in `dsl/macros/tests/ui/`.

**Deps 2026-10-18 (R18):** proc-macro crate on `syn` 3, `quote`, `proc-macro2`, `darling` 0.24 (attribute parsing) and `manyhow` (multiple errors with spans). `derive(Action)` may delegate enumeration to `enum-iterator`'s `Sequence` (syn 2 duplicate accepted). Expansion goldens: insta snapshots of the derive's implementation function, formatted by `prettyplease`, not macrotest.

**Note 2026-10-18 (R17):** derive only what Kettle needs (unit variants, `#[starts]` ranges), but keep `derive(Action)` and `derive(State)` shaped so variants with data and a `Finite` domain trait slot in at Bank without breaking Kettle.

**Note 2026-10-18 (R8, golden files):** refusals are `trybuild` `.stderr` goldens and the derived output is an insta snapshot formatted by `prettyplease`; no unit tests of the derive.

New proc-macro crate `umpire/dsl/macros` re-exported from `umpire-dsl`: `#[derive(State)]` adds Clone, Eq, Hash and `fn starts() -> Vec<Self>` from `#[starts(expr)]` field attributes (cartesian over annotated fields, defaults for the rest); `#[derive(Action)]` yields the catalog, names and `internal()` from `#[internal]` variants. Use `syn`, `quote`, `darling`; pin messages with `trybuild`.

**Files:** model2/umpire/dsl/macros/{Cargo.toml,src/lib.rs}, model2/umpire/dsl/Cargo.toml, tests/ui/.

**Quick commands:** `mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros`
## Acceptance
- Kettle-shaped struct and enums compile with the derives in DSL-owned coverage; insta expansion snapshots for `derive(State)` and `derive(Action)` checked in (prettyplease-formatted).
- `ui/` goldens for a non-range `#[starts]` and a non-unit `#[internal]` variant.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
