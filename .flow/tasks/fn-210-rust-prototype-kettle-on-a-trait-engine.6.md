---
satisfies: [R6, R14, R44]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.6 Kettle in surface C with its test; parity with the Scala suite

## Description
**R14 joined integration:** consume the .5 source-order metadata in checker reporting, if the separate registry hooks otherwise group claims by kind. Add a shared Model with interleaved invariant/postcondition/progress declarations and a report golden proving actual order, while keeping manual-world defaults and dynamic effect failures deterministic. This checker integration belongs here after .3/.5, not in the parallel DSL lane.

**Touches:** `model/umpire/tests/**`, `model/umpire/checker/src/**`, `model/umpire/checker/tests/**`, `model/umpire/dsl/**`, `model/DESIGN.md`

**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model-check` at 30 s or less, warm; paste `make model-time` output into the done summary; a breach is fixed here or excepted by the owner.

**Location (R29 amended):** there is no `samples` crate; write Kettle in surface C as `umpire/tests/fixtures/kettle/model.rs`, replacing the trait-shaped source there. Shared end-to-end fixtures live in `umpire/tests/fixtures/<name>/` and their expected files select the consumers.

**Owner-approved authoring decision (2026-10-18):** keep the sketch's separate `User` and `Heater` enums. This supersedes this task's earlier byte-identical canonical-goldens requirement only for the four author action spellings: `Act::UserPressButton` becomes `User::PressButton`, `Act::UserDepressButton` becomes `User::DepressButton`, `Act::HeaterHeat` becomes `Heater::Heat`, and `Act::HeaterFailover` becomes `Heater::Failover`. Counts, state spelling and discovery order, action order, claims, verdicts, vacuity and counterexample structure must not change. Record the exception in DESIGN.md. Preserve the separate trait Kettle and all its goldens unchanged as the WASM/native reference. Names produced by surface C must remain the actual author Rust names (R25/R26), not aliases chosen to fit old goldens.

Write `umpire/tests/fixtures/kettle/model.rs` in surface C from `options/Kettle.attrs.rs`; the shared fixture runner asks the Scala suite's questions: explore counts, verify_all verified, `boils` verified, and `!beeping` refuted. Prove parity against the existing canonical goldens with only the four permitted name substitutions, and retain executable coverage of the unchanged trait reference. Record every deliberate deviation from the hypothetical sketch, including static scope/progress builders and generated world/action adapters.

**Files:** model/umpire/tests/fixtures/kettle/model.rs and shared fixture runner; checker ordering integration and shared interleaved-claim fixture; model/DESIGN.md for deliberate sketch differences. Do not change workspace manifests, the ABI, archived Scala, or the separate trait fixture and its goldens.

**Quick commands:** `cd model && mise exec -- cargo test --workspace`

**Structural lint (R44, owner 2026-10-18):** `#[machine]` rejects a Model whose registered items are not grouped by kind in the order `#[on]`, `#[invariant]`, `#[postcondition]`, `#[scope]`, `#[progress]`; one ordering table in `dsl/macros/src/machine.rs`, error at the first out-of-place item, helper methods without an attribute ignored, pinned by a trybuild `.stderr` fixture (`machine_out_of_order`). Reorder the existing `machine` macro fixture to comply. The R14 interleaved-claim fixture must be a manual trait world, since the attribute surface can no longer interleave kinds.

**Target surface (owner approved 2026-10-18; supersedes the sketch where they differ):** three cuts to `options/Kettle.attrs.rs`, each keeping every item a plain Rust function with full editor support. `#[scope]` stays. The `Machine` trait, checker, goldens and wasm host do not move; `Scope` and `Progress` become plain non-generic values that `#[machine]` wraps into the trait's `Scope<KettleAction>` and fn-pointer predicates.

1. Actor-level `#[internal]` only (as Scala's `extends Internal`); drop per-variant `#[internal]`.

```rust
#[derive(Action)]
#[internal]
pub enum Heater {
    Heat,
    Failover,
}
```

2. `before` is optional: the macro reads the arity, so `fails_over_at_boiling(&self) -> bool` compiles; `warms_only_when_on(&self, before: &Self) -> bool` is unchanged.

3. Scope and progress as the Scala line, no closures, no generated type names, doc comments on the items:

```rust
/// The user switches the kettle on but never off.
#[scope]
fn never_switched_off() -> Scope {
    without([User::DepressButton])
}

/// Switched on and left on, the water boils and the kettle beeps.
#[progress(within = Self::never_switched_off)]
fn boils(&self) -> Progress {
    whenever(self.heating_on).eventually(self.beeping)
}
```

`Progress` is `{ trigger: bool, target: bool }` evaluated per state; the expansion is `|_, s| s.boils().trigger` and `.target`. `Scope` holds actor-level action values; the expansion maps them into `KettleAction`. Rejected today's `fn never_reset() -> Scope<CounterAction>` / `Progress::<CounterWorld>::new("..").whenever(|_, s| ..)` spelling; rewrite the `machine` macro fixture to the target surface.

Unchanged: the `State` struct with `#[starts]`, `#[machine(actions = (User, Heater))]` (the only line fixing catalog order), the `#[on]` handlers returning `Option<()>`, `#[invariant]`. Record any remaining difference from the sketch in DESIGN.md, one sentence each.
## Acceptance
- The canonical Kettle uses the attribute surface with separate `User` and `Heater` enums; its fixture asks all Scala-suite questions.
- Canonical expected payloads differ from the previous trait-shaped Kettle only by the four owner-approved action-name substitutions; report counts/verdicts, states, catalog/discovery order and counterexample structure remain identical.
- The separate `kettle_trait` source and all its expected files remain unchanged and its native/WASM consumers still pass.
- `#[machine]` rejects an impl whose registered parts are not grouped in the order on, invariant, postcondition, scope, progress, with an error at the first out-of-place item; a trybuild `.stderr` fixture pins the message and every attribute fixture complies (R44).
- A shared interleaved invariant/postcondition/progress fixture on a manual trait world proves actual checker report declaration order; manual-world defaults and dynamic effect-failure ordering remain deterministic.
- Kettle is written in the approved target surface: actor-level `#[internal]`, optional `before`, `fn never_switched_off() -> Scope { without([User::DepressButton]) }` and `fn boils(&self) -> Progress { whenever(self.heating_on).eventually(self.beeping) }`, with no generated world or action type named by the author; remaining differences from the sketch are recorded in DESIGN.md.
- The workspace test command and full local gate pass; actual warm model-time output satisfies R30 or records an owner-approved exception.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
