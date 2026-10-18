---
satisfies: [R5]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.5 #[machine] attribute: pass-through impl, generated Machine impl from #[on]/#[invariant]/#[postcondition]/#[scope]/#[progress]

## Description
Attribute macro on an `impl` block: emit the impl unchanged, then a `Machine` impl whose `step` dispatches by `#[on(Action::Variant)]` (a `None` return refuses), whose claim lists collect `#[invariant]` and `#[postcondition(after = …)]` functions by name, and whose progress collects `#[progress(within = …)]`. Compile errors naming the action for a missing or duplicate `#[on]` and for an unknown `after`. Confirm rust-analyzer navigates the impl (record in the write-up).

**Files:** model2/framework/dsl/macros/src/machine.rs, tests/ui/.

**Quick commands:** `mise exec -- cargo test -p framework-dsl`

## Acceptance
- Attribute output contains the original impl byte-for-byte (macrotest snapshot).
- trybuild: missing #[on], duplicate #[on], unknown after.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
