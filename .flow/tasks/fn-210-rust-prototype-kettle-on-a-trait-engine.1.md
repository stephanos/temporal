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
The DSL separates a world's constants from its associated dynamic State and returns ordered edges carrying domain outcomes and facts. R18 now stores the first edge inline through smallvec, while the adapted KettleWorld fixture continues to prove compilation of the trait vocabulary and tri-state predicates.

Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key).
stage: impl-review - skipped(policy: parallel-wave - conductor owns the final combined review)

The world API and R18 SmallVec update are integrated into umpire. The conductor re-ran both exact Quick commands, make model2-check (including WASI), and cargo deny check bans on the integrated tree; all passed. The ban check required explicit 0.0.1 versions on internal workspace path dependencies.

R18 changes Steps<S, O, F> to smallvec::SmallVec<[Step<S, O, F>; 1]>, with Step<S, O, F> = (O, Vec<F>, S). The DSL opts into smallvec.workspace and Cargo.lock resolves 1.16.3. The fixture uses Steps::new() for disabled actions and Steps::new() plus push for the single emitted result. This adds no direct smallvec dependency to samples. The vocabulary has no fallible API or error type, so thiserror remains available in workspace dependencies for the checker's actual error surface.

Baseline was green for both exact task Quick commands. After the Steps alias change, the existing fixture failed compilation with mismatched Vec and SmallVec return types. Adapting the fixture constructors made both Quick commands pass. make model2-check passed formatting, strict Clippy, warning-free documentation, workspace tests, and the wasm32-wasip2 build. The samples suite ran the trait compilation fixture and the inherited placeholder. R8 behavior goldens remain owned by .2. Cumulative gate classification was FULL. Logs are in the workspace's .flow/tmp/fn2101-r18-{baseline-build,baseline-samples,red,verify-build,verify-samples,model2-check}.log files.

The remainder of the world API is unchanged from the previous handover. Machine is implemented by a Sized world with State: Clone + Eq + Hash, Action: Clone + Eq + Hash + Debug, Outcome, and Fact. starts, actions, internal, and claim registration take the world by reference; step(&self, &State, Action) returns Steps<State, Self::Outcome, Self::Fact>. The finite action catalog is ordered and admits future data variants. Empty Steps disables an action; a rejecting domain outcome remains an edge.

Claim predicates take (&M, &M::State). StepPredicate takes (&M, &before, &after, &Action, &M::Outcome, &[M::Fact]). StepClaim::holds takes (&self, world, before, action, &Step<M::State, M::Outcome, M::Fact>). Progress trigger and target also receive the world and state. All predicates return Verdict::{Holds, Fails, Unknown(Hole { reason })}, with bool conversion. Aggregate Outcome retains Verified, Refuted, and Unknown separately from Machine::Outcome. Scope supports Clone actions and borrowed allows(&Action). The Kettle fixture uses an empty public KettleWorld with separate public Kettle state and unit outcomes/facts. Original option sketches stay unchanged.

The conductor reported a completed review of the world API before R18. The new R18 delta and final combined artifact require conductor review. R15 here proves target compilation; the conductor's observed wasm32-wasip2 panic=abort limitation remains for the later R11 hosting write-up.

Review receipt: /tmp/impl-review-receipt-be5707dcaf67-fn-210-rust-prototype-kettle-on-a-trait-engine.1.json
stage: plan-sync - skipped(config: planSync.enabled=false)
Memory capture: skipped; both world/R18 review cycles were clean first-pass SHIP.
## Evidence
- Commits: 05787789742ee7872c43d464360c54de72578d06, 4147e723e750878952f2760ef3152df6218e921c, 841867500274cd1a34ad5d0b5f83822648c2a097, 9e22629e26b2dc1bbd00e7a003e9ec89507f8d01, a6661dcf7601f08e69ada3c6592049e7929f0f36, a8cf087dbc61cd2c9544472d88f6875c13787883
- Tests: mise exec -- cargo build --workspace, mise exec -- cargo test -p samples, make model2-check, mise exec -- cargo deny check bans, git diff --check
- PRs: