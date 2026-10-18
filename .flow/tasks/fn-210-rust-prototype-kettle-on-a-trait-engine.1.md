---
satisfies: [R1]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.1 framework-dsl: Machine trait, Claim, StepClaim, Scope, Progress, Outcome; Kettle.trait.rs compiles as a fixture

## Description
Define the vocabulary in `framework/dsl/src`: `Machine { type Action; starts(); actions(); internal(a); step(&self, a) -> Option<Self> }` with `Self: Clone + Eq + Hash`; `Claim<S>` and `StepClaim<S>` as named predicates with `StepClaim::after(action, f)`; `Scope<A>::without(&[A])`; `Progress<S>` builder (`whenever`, `eventually`, `within`); `Outcome`. Copy `options/Kettle.trait.rs` into `framework/samples/tests/fixtures/kettle_trait.rs` (or a test module) and make it compile with the sole owner-approved `Progress::<Self>::new` inference annotation.

**Files:** model2/framework/dsl/src/lib.rs (+ machine.rs, claim.rs, scope.rs, progress.rs), model2/framework/samples/tests/.

**Quick commands:** `mise exec -- cargo build --workspace`; `mise exec -- cargo test -p samples`

## Acceptance
- Kettle.trait.rs compiles against the crate as an executable fixture with only `Progress::<Self>::new` added; the owner approved this inference annotation on 2026-10-18, and the original option sketch remains unchanged.
- `cargo doc` renders every public item with a one-line doc.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
