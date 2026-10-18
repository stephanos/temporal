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

**Target surface (owner approved 2026-10-18, nested form; R44, supersedes the sketch where they differ):** the Model is one inline `#[machine] pub mod kettle { .. }` with one `impl Kettle` per part kind, in this order and each at most once: `#[transitions]`, `#[invariants]`, `#[postconditions]`, `#[scopes]`, `#[progress]`. The block is the group: no `#[invariant]`/`#[postcondition]`/`#[progress]` per item; only the per-claim qualifiers `#[on(..)]`, `#[after(..)]`, `#[within(..)]`. `#[machine]` takes no arguments; catalog order is the `#[derive(Action)]` enums' declaration order. `#[machine]` registers from the module's items (R36), errors at the first out-of-place or duplicated group, ignores unmarked impls and helpers; trybuild `.stderr` fixtures `machine_out_of_order` and `machine_duplicate_group` pin the messages. Rewrite the `machine` macro fixture to this form. The R14 interleaved-claim fixture must be a manual trait world, since the attribute surface cannot interleave kinds.

```rust
#[machine]
pub mod kettle {
    #[derive(State)]
    pub struct Model {
        #[starts(COLDEST_TEMPERATURE..=40)]
        temperature: i32,
        heating_on: bool,
        beeping: bool,
    }

    #[derive(Action)]
    pub enum User { PressButton, DepressButton }

    #[derive(Action)]
    #[internal]
    pub enum Heater { Heat, Failover }

    #[transitions]
    impl Model {
        #[on(User::PressButton)]
        fn press_button(&mut self) -> Option<()> { (!self.heating_on).then(|| { self.heating_on = true; self.beeping = false; }) }
        #[on(Heater::Failover)]
        fn failover(&mut self) -> Option<()> { .. }
        #[on(Heater::Heat)]
        fn heat(&mut self) -> Option<()> { .. }
        #[on(User::DepressButton)]
        fn depress_button(&mut self) -> Option<()> { .. }
    }

    #[invariants]
    impl Model {
        fn beeps_only_when_off(&self) -> bool { !self.beeping || !self.heating_on }
    }

    #[postconditions]
    impl Model {
        fn warms_only_when_on(&self, before: &Self) -> bool { before.heating_on || self.temperature <= before.temperature }
        #[after(Heater::Failover)]
        fn fails_over_at_boiling(&self) -> bool { self.temperature == BOILING_TEMPERATURE && self.beeping }
    }

    #[scopes]
    impl Model {
        /// The user switches the kettle on but never off.
        fn never_switched_off() -> Scope { without([User::DepressButton]) }
    }

    #[progress]
    impl Model {
        /// Switched on and left on, the water boils and the kettle beeps.
        #[within(Self::never_switched_off)]
        fn boils(&self) -> Progress { whenever(self.heating_on).eventually(self.beeping) }
    }
}
```

Details fixed with it: actor-level `#[internal]` only (as Scala's `extends Internal`), no per-variant form; `before` optional in postconditions (macro reads the arity); `Scope` and `Progress` are plain non-generic values, `Progress` = `{ trigger: bool, target: bool }` evaluated per state, expanded as `|_, s| s.boils().trigger` / `.target`, `Scope` holds actor-level action values mapped into `kettle::Action` by the expansion; the author never names the generated `kettle::World`/`kettle::Action` (module-scoped, no prefix). Kept: `#[on]` handlers returning `Option<()>` (empty-vs-enabled-no-op matters for liveness) and named `#[scopes]`. The `Machine` trait, checker, goldens and wasm host do not move; consumers (shared fixture runner, `umpire-wasm`'s `#[path]` include) refer to `kettle::Model` and the `pub` items. **Naming (owner 2026-10-18):** the state struct is `Model` (Scala's `object Model`), so the canonical goldens carry a fifth approved substitution, state spelling `Kettle { .. }` to `Model { .. }` in `states.txt`, `rows.txt` and `paths/`; record it next to the four action substitutions in DESIGN.md. `kettle_trait` stays unchanged. Record any remaining difference from the sketch in DESIGN.md, one sentence each.
## Acceptance
- The canonical Kettle uses the attribute surface with separate `User` and `Heater` enums; its fixture asks all Scala-suite questions.
- Canonical expected payloads differ from the previous trait-shaped Kettle only by the four owner-approved action-name substitutions and the state spelling `Kettle { .. }` to `Model { .. }`; report counts/verdicts, states, catalog/discovery order and counterexample structure remain identical.
- The separate `kettle_trait` source and all its expected files remain unchanged and its native/WASM consumers still pass.
- `#[machine]` on the Model module rejects a duplicated or out-of-order group impl (`#[transitions]`, `#[invariants]`, `#[postconditions]`, `#[scopes]`, `#[progress]`) with an error at the first offending block; trybuild `.stderr` fixtures pin both messages and every attribute fixture complies (R44).
- A shared interleaved invariant/postcondition/progress fixture on a manual trait world proves actual checker report declaration order; manual-world defaults and dynamic effect-failure ordering remain deterministic.
- Kettle is written in the approved nested surface: a `#[machine] pub mod kettle` whose state struct is `Model`, with one impl per part kind, actor-level `#[internal]`, optional `before`, `#[after(..)]`/`#[within(..)]` qualifiers, `fn never_switched_off() -> Scope { without([User::DepressButton]) }` and `fn boils(&self) -> Progress { whenever(self.heating_on).eventually(self.beeping) }`, with no generated world or action type named by the author; remaining differences from the sketch are recorded in DESIGN.md.
- The workspace test command and full local gate pass; actual warm model-time output satisfies R30 or records an owner-approved exception.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
