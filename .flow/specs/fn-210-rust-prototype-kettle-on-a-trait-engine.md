# Rust prototype: Kettle on a trait engine with an attribute surface (model2)

## Conversation Evidence

> user (2026-10-18): "let's plan a prototype for temporal/model2 that uses Rust. the focus is to support only what kettle.scala does today. we want to follow the roughly same package layout and naming."
> user (2026-10-18): "we want to keep the effort for the macro to a minimum. and leverage as much Rust natively as we can."
> user (2026-10-18): yes to "build D's engine first because every surface needs it, then C as the surface".
> user (2026-10-18): "mark all others as deferred for now."

## Goal & Context

Before any more of the Scala framework is built out, the owner wants to know whether Rust is the better base: it would make lifting a macro's by-product, make wasm and native hosting first-class, run the checker fast by default, and has more mind share at the company. The way to know is a bounded prototype, not an argument. `temporal/model2` is a Cargo workspace mirroring `model/framework`'s layout (`framework/dsl`, `framework/checker`, `framework/samples`). Its scope is exactly what `model/framework/samples/Kettle.scala` uses today: vars with a `oneOf` start, actors with actions and internal steps, transitions with guards and effects, invariants, postconditions with `before` and `after(action)`, one `without` scope, one `whenever … eventually … in` progress claim, and the questions `explore`, `verifyAll` and `refuted`. The authoring-surface options were written out in `framework/samples/options/`; the owner chose to build the plain-trait engine first (option D, which every surface expands to) and then the attribute surface (option C, plain functions with pass-through attributes). Every other open spec is deferred until this prototype's write-up gives a verdict.

## Architecture & Data Models

- `framework-dsl`: the `Machine` trait (`type Action`, `starts()`, `actions()`, `internal(action)`, `step(&self, action) -> Option<Self>`), `Claim<S>` and `StepClaim<S>` (named predicates, `StepClaim::after(action)`), `Scope<A>` (`without`), `Progress<S>` (`whenever`, `eventually`, `within`), `Outcome`. States are `Clone + Eq + Hash`.
- `framework-checker`: BFS exploration from the starts with a shared visited set and a state budget; per-state invariant evaluation; per-edge postcondition evaluation with `before`, `after` and the action; progress as a search for a fair lasso inside the scope's action set with weak fairness on internal steps; `Result` with verdict (verified, refuted with a path, unknown on budget cut), reached-state and emitted-step counts; `Report` from `verify_all` with per-claim results and vacuity warnings (an action never fired, an `after` claim never triggered); `refuted(claim)` for negative controls.
- Surface C in `framework-dsl` (a proc-macro crate `framework-dsl-macros`): `#[derive(State)]` (Clone, Eq, Hash, `starts()` from `#[starts(range)]` field attributes), `#[derive(Action)]` with `#[internal]` variants, and `#[machine(actions = (A, B))]` on an `impl` that passes the impl through unchanged and generates the `Machine` impl from `#[on(Action::Variant)]`, `#[invariant]`, `#[postcondition(after = …)]`, `#[scope]` and `#[progress(within = …)]` items. The attribute must refuse an action with no `#[on]` or with two.
- Samples: `Kettle` written in surface C in `framework/samples/src`, with `tests/kettle.rs` asking the Scala suite's questions; the option sketches stay in `options/` as documentation.
- Parity reference: the measurement note's Kettle row (81 starts, 283 reached states, 424 emitted steps, BFS depth 62, 4 claims verified) and the Scala test's negative control (`!beeping` refuted).

## API Contracts

- `Checker::explore::<M>(budget) -> Result<M>`; `Result::verify_all() -> Report<M>`; `Checker::refuted::<M>(claim) -> Result<M>`.
- `Report` lists every claim with verified or refuted-with-path and carries vacuity warnings; `assert_verified()` panics with the first refutation's path.
- Counts on `Result`: starts, reached states, emitted steps, depth, cut (bool).
- The `#[machine]` attribute emits a compile error naming the action for a missing or duplicate `#[on]`, and for `after = X` naming an action the impl does not handle.
- `mise exec -- cargo test --workspace` is the gate; `rustfmt` clean.

## Edge Cases & Constraints

- A `step` that returns `None` is a refusal and emits no edge; the explorer still counts the evaluation so emitted-step parity with the Scala note (which counts refusals) can be reasoned about explicitly in the write-up.
- Progress within a scope excludes the scoped-out actions from the search entirely; fairness applies to internal steps only, as in the Scala checker's default.
- Budgets are positive; a cut yields unknown unless a refutation was already found.
- No compositions, channels, inputs, facets, realizations, lifting, emitters or Go in scope.
- The Scala framework is not modified by this spec.

## Acceptance Criteria

- **R1:** `framework-dsl` defines `Machine`, `Claim`, `StepClaim`, `Scope`, `Progress` and `Outcome` as contracted, and `Kettle.trait.rs` from `options/` compiles against them as a test fixture, with the sole source change `Progress::<Self>::new` explicitly approved by the owner on 2026-10-18 because Rust cannot infer the state type through the builder chain; the original option sketch stays unchanged. Errors: none beyond compilation.
- **R2:** `Checker::explore` reaches exactly the Scala note's Kettle counts (81 starts, 283 states, depth 62) and `verify_all` reports the two invariant and postcondition claims verified with no vacuity warnings. Errors: a budget below one refuses; a cut yields unknown with the counts so far.
- **R3:** Progress `boils` within `without(DepressButton)` is verified with weak fairness on internal steps, and a control Model with no heater step is refuted with a lasso path. Errors: a scope naming an unknown action refuses.
- **R4:** `Checker::refuted` reports `!beeping` refuted with a path from a start to a beeping state. No error surface beyond R2.
- **R5:** `#[derive(State)]`, `#[derive(Action)]` and `#[machine]` implement surface C: the attribute passes the impl through unchanged, generates the `Machine` impl, and refuses missing or duplicate `#[on]` and an unknown `after` action with the action's name. Errors: as stated; `trybuild` tests pin the messages.
- **R6:** `Kettle` in surface C lives in `framework/samples/src`, its test asks the Scala suite's questions and passes, and the Kettle sketch in `options/Kettle.attrs.rs` matches it line for line except where the write-up records a deliberate change. No error surface.
- **R7:** A design note, `model2/DESIGN.md`, records: authoring comparison against `Kettle.scala` (line count, constructs that got worse or better), framework line counts per crate against the Scala counterparts, `verify_all` wall time for Kettle against the measurement note's 0.37 s, build time, rust-analyzer behaviour inside the impl, and a recommendation on whether to extend the prototype to Bank and Voting or stop. No error surface.
- **R8:** (2026-10-18) Tests are cases with golden files, not unit assertions. Behaviour: one test runs every registered sample Model and snapshots `cases/<model>/expected/report.txt` (starts, reached states, emitted steps, depth, cut, every claim with its verdict, vacuity warnings) and `cases/<model>/expected/paths/<claim>.txt` for each counterexample, with `insta`. Refusals: `trybuild` `.rs` + `.stderr`. Expansion: `macrotest` `.expanded.rs`. Goldens are deterministic (no hash-order leakage; states sorted, paths in BFS order over a fixed action order, a stable `Display` for states), human-readable (one claim or one step per line), contain no timings, and are updated by one flag wrapped as `make model2-update`; a golden without a case or a case without a golden fails. One property test (proptest) checks the checker against a brute-force reference explorer on small random Models; it is the only check that goldens are right rather than merely stable. No other unit tests unless a defect cannot be expressed as a case. Errors: a nondeterministic golden (differs across two runs) is a defect of the producer, not of the case.

## Boundaries

- Only Kettle's constructs; Bank, Voting and the rest are the next decision, not this spec.
- No lifting, no TLA emission, no wasm build, no Testpilot integration.
- No changes under `model/`; the Scala framework and its specs are deferred, not altered.
- Surface A (`model!` grammar) is not built; B's `macro_rules!` helpers only if R6 finds a specific construct clearly better that way, recorded in the write-up.

## Decision Context

Seven surface options were written for the same Model (`framework/samples/options/README.md`). The libraries (statig, smlang, stateright) model control flow with named states or offer only state predicates, so they bend a var-shaped Model with claims out of shape; stateright's `next_state` is the reference for the explorer loop but the crate is single-maintainer and unreleased since July 2025. Among the custom surfaces, the grammar block is a second compiler with no editor support, the `macro_rules!` helpers look better than they will compile (destructuring breaks on collections), and the no-macro trait is right for the engine but too noisy for authors. The attribute surface keeps every item a plain function with Rust's own errors and editor support; its honest cost is `self.` on every line. The engine is built first because every surface expands to it and it is what the parity numbers test. All other open specs are deferred on 2026-10-18 until R7's recommendation; the Scala framework keeps working as it is. Filed 2026-10-18.

2026-10-18, testing philosophy (owner): heavy emphasis on end-to-end tests through fixtures and golden files instead of unit tests, matching fn-198's case folders on the Scala side. Decisions: `insta` runs and reviews behaviour goldens (`cargo insta review`); the strategy is R8 and applies to every task from now on, including the in-progress fn-210.1's fixture; the brute-force oracle property test lands with fn-210.2 while the checker is small. Kettle's parity numbers become lines of its `report.txt` rather than assertions.
