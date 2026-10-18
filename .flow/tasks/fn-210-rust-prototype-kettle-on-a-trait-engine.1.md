---
satisfies: [R1]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.1 framework-dsl: Machine trait, Claim, StepClaim, Scope, Progress, Outcome; Kettle.trait.rs compiles as a fixture

## Description
**Deps 2026-10-18 (R18):** `Steps` is `SmallVec<[Step; 1]>`; library errors use `thiserror`. No other dependency is needed for the vocabulary.

**Recut 2026-10-18 (R9, R10, R12, in progress):** the trait shape changed. A world value implements `Machine` with associated `State` and `Action`; `starts(&self) -> Vec<State>`; `step(&self, &State, Action) -> Steps<State, Outcome, Fact>` with `Steps` an ordered list of `(outcome, facts, next)`, empty = disabled; claims return `Verdict { Holds, Fails, Unknown(Hole) }` with `From<bool>`. Adapt `tests/trait_fixture.rs` to this shape (an empty `KettleWorld` struct); R1's 'compiles unchanged' is superseded by R9.

**Note 2026-10-18 (R8, golden files):** keep the fixture minimal: it proves the vocabulary compiles. Behaviour assertions belong in goldens from .2 on; do not add unit tests here beyond what compilation needs.

Define the vocabulary in `framework/dsl/src`: `Machine { type Action; starts(); actions(); internal(a); step(&self, a) -> Option<Self> }` with `Self: Clone + Eq + Hash`; `Claim<S>` and `StepClaim<S>` as named predicates with `StepClaim::after(action, f)`; `Scope<A>::without(&[A])`; `Progress<S>` builder (`whenever`, `eventually`, `within`); `Outcome`. Copy `options/Kettle.trait.rs` into `framework/samples/tests/fixtures/kettle_trait.rs` (or a test module) and make it compile with the sole owner-approved `Progress::<Self>::new` inference annotation.

**Files:** model2/framework/dsl/src/lib.rs (+ machine.rs, claim.rs, scope.rs, progress.rs), model2/framework/samples/tests/.

**Quick commands:** `mise exec -- cargo build --workspace`; `mise exec -- cargo test -p samples`
## Acceptance
- Kettle.trait.rs compiles against the crate as an executable fixture with only `Progress::<Self>::new` added; the owner approved this inference annotation on 2026-10-18, and the original option sketch remains unchanged.
- `cargo doc` renders every public item with a one-line doc.

## Done summary
Defined Machine, Claim, StepClaim, Scope, Progress and Outcome with documented public APIs. The executable trait fixture type-checks against the DSL; its only source change is the owner-approved `Progress::<Self>::new` annotation, with namespace compatibility imports outside the fixture. The original option sketch is untouched.

R8's compilation-only policy superseded the initial behavioral assertions during review. The final fixture only requires Kettle: Machine; behavior coverage belongs to task .2's goldens and oracle property test. Focused workspace build and samples tests passed after the fix, and make model2-check passed formatting, strict Clippy, warning-free docs and workspace tests. Reviewers could not rebuild in their read-only sandbox; conductor verification supplies execution evidence.

Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key).
stage: impl-review - ran (model: gpt-6.1-sol at high; codex backend, same family as writer; SHIP after one R8 policy fix)
stage: plan-sync - skipped(config: planSync.enabled=false)
Review receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.1.json
Memory capture: skipped; review fix mechanically relocated coverage under the newly added testing policy.
## Evidence
- Commits:
- Tests:
- PRs:
