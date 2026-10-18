---
satisfies: [R5, R13, R14, R26]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.5 #[machine] attribute: pass-through impl, generated Machine impl from #[on]/#[invariant]/#[postcondition]/#[scope]/#[progress]

## Description
**Integration after .3/.4:** the existing engine is world-shaped (R9/R10). Generated dispatch must clone the state, call the original transition, and turn an enabled `Option<()>` into the ordered unit-outcome/fact-free edge; `None` means disabled, not an API Refusal or rejecting outcome. Preserve the original impl and method bodies in the expansion; helper attributes must remain valid pass-through Rust attributes. Record any deliberate authoring changes from the hypothetical sketch in the .6 DESIGN comparison, not by silently rewriting bodies.

**R14 cross-kind order:** source-order metadata must retain interleaved invariant/postcondition/progress names, not merely each kind's order. A defaulted Machine metadata hook can be added within this DSL lane; .6 will connect it to checker reporting and prove actual cross-kind order after the lanes join. Do not edit checker source while .3 is active.

**R19 handoff:** keep the original machine impl tokens available for the later lift. State/action declarations and start expressions live outside that impl; preserve/expose their derive input at a minimal implementation-only seam here or document the exact seam .8 must add. Do not substitute handwritten Kettle TLA or create a stable IR/public protobuf metadata protocol. Read backend-readiness.md, fn2104-derive-seam.md and claim-ordering.md in the shared notes directory.

**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model2-check` at 30 s or less, warm; paste `make model2-time` output into the done summary; a breach is fixed here or excepted by the owner.

**Shared fixtures 2026-10-18 (R29):** end-to-end fixtures live in `umpire/tests/fixtures/<name>/` after .13; their expected files select the consumers (checker, TLC, Apalache).

**Conventions 2026-10-18 (R26):** `#[machine]` registers claims, scopes and progress under the author's Rust names; outputs use them verbatim.

**Touches:** model2/umpire/dsl/build.rs, model2/umpire/dsl/macros/build.rs, model2/umpire/dsl/src/**, model2/umpire/dsl/tests/**, model2/umpire/dsl/macros/src/**, model2/umpire/dsl/macros/tests/**

**Parallel boundary:** shared Cargo wiring is prepared by .11; no manifests or lockfiles change in this lane. This lane may follow .4 while .3 remains active.

**Layout (R22):** expansion cases go in `umpire/dsl/macros/tests/fixtures/<name>/`; refusals in `dsl/macros/tests/ui/`.

**Deps 2026-10-18 (R18):** same macro stack as .4; the pass-through golden is an insta snapshot of `machine_impl(tokens)` formatted by `prettyplease` (replaces macrotest).

**Recut 2026-10-18 (R13):** `#[machine]` warns on expressions outside the liftable subset (arithmetic, comparisons, boolean logic, field access, enum `match`, `imbl` operations, helpers in the same impl), naming the expression and the reason; pin one warning with a trybuild golden.

**Note 2026-10-18 (R8, golden files):** the pass-through property is an insta snapshot (`expand/kettle.expanded.rs` contains the original impl unchanged); refusals are `trybuild` goldens.

Attribute macro on an `impl` block: emit the impl unchanged, then a `Machine` impl whose `step` dispatches by `#[on(Action::Variant)]` (a `None` return refuses), whose claim lists collect `#[invariant]` and `#[postcondition(after = …)]` functions by name, and whose progress collects `#[progress(within = …)]`. Compile errors naming the action for a missing or duplicate `#[on]` and for an unknown `after`. Confirm rust-analyzer navigates the impl (record in the write-up).

**Files:** model2/umpire/dsl/macros/src/machine.rs, tests/ui/.

**Quick commands:** `mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros`
## Acceptance
- Attribute output contains the original impl unchanged (insta snapshot of the prettyplease-formatted expansion).
- trybuild: missing #[on], duplicate #[on], unknown after.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
