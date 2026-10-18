---
satisfies: [R5, R26]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.5 #[machine] attribute: pass-through impl, generated Machine impl from #[on]/#[invariant]/#[postcondition]/#[scope]/#[progress]

## Description
**Conventions 2026-10-18 (R26):** `#[machine]` registers claims, scopes and progress under the author's Rust names; outputs use them verbatim.

**Touches:** model2/umpire/dsl/build.rs, model2/umpire/dsl/macros/build.rs, model2/umpire/dsl/src/**, model2/umpire/dsl/tests/**, model2/umpire/dsl/macros/src/**, model2/umpire/dsl/macros/tests/**

**Parallel boundary:** shared Cargo wiring is prepared by .11; no manifests or lockfiles change in this lane. This lane may follow .4 while .3 remains active.

**Layout (R22):** expansion cases go in `umpire/dsl/macros/tests/fixtures/<name>/`; refusals in `dsl/macros/tests/ui/`.

**Deps 2026-10-18 (R18):** same macro stack as .4; the pass-through golden is an insta snapshot of `machine_impl(tokens)` formatted by `prettyplease` (replaces macrotest).

**Recut 2026-10-18 (R13):** `#[machine]` warns on expressions outside the liftable subset (arithmetic, comparisons, boolean logic, field access, enum `match`, `imbl` operations, helpers in the same impl), naming the expression and the reason; pin one warning with a trybuild golden.

**Note 2026-10-18 (R8, golden files):** the pass-through property is an insta snapshot (`expand/kettle.expanded.rs` contains the original impl unchanged); refusals are `trybuild` goldens.

Attribute macro on an `impl` block: emit the impl unchanged, then a `Machine` impl whose `step` dispatches by `#[on(Action::Variant)]` (a `None` return refuses), whose claim lists collect `#[invariant]` and `#[postcondition(after = …)]` functions by name, and whose progress collects `#[progress(within = …)]`. Compile errors naming the action for a missing or duplicate `#[on]` and for an unknown `after`. Confirm rust-analyzer navigates the impl (record in the write-up).

**Files:** model2/umpire/dsl/macros/src/machine.rs, tests/ui/.

**Quick commands:** `mise exec -- cargo test -p umpire-dsl`
## Acceptance
- Attribute output contains the original impl unchanged (insta snapshot of the prettyplease-formatted expansion).
- trybuild: missing #[on], duplicate #[on], unknown after.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
