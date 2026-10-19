---
satisfies: [R6]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.20 Flat form: per-item attributes on one impl, match per actor, grouping enforced

## Description
**Owner decisions (2026-10-19, final form after reverting the nested groups and marker traits as "too much ceremony"):** the Model is `#[machine] pub mod kettle { .. }` with the state struct, the action enums and **one `impl Model`** whose methods carry per-item kind attributes, the `options/Kettle.attrs.rs` style: `#[on(User)]` / `#[on(Heater)]` (one `match` method per actor; match guards are the guards; `_ => {}` is "disabled" under .18's enabled-iff-changed rule; rustc's exhaustiveness replaces the missing-handler error when no `_` arm is written), optional `#[effect]` helpers (the only other methods allowed `&mut self`), `#[invariant]`, `#[postcondition]` and `#[postcondition(after = Heater::Failover)]`, `#[assumption]`, `#[progress]`. `#[machine]` enforces that items of one kind are contiguous and the kinds appear in the order on, effect, invariant, postcondition, assumption, progress, with a hard error at the first out-of-place item naming the kind it must follow and precede; unattributed helpers are ignored by the order check but refused if they take `&mut self` (`<method> takes &mut self; only #[on] and #[effect] methods change the state`); `#[machine]` emits `#[forbid(unsafe_code)]` on the module and takes no arguments (catalog order = the `Action` enums' declaration order). Every attribute is pass-through; nothing is rewritten.

```rust
#[machine]
pub mod kettle {
    #[derive(State)]
    pub struct Model {
        #[starts(COLDEST_TEMPERATURE..=40)]
        pub temperature: i32,
        pub heating_on: bool,
        pub beeping: bool,
    }

    #[derive(Action)]
    pub enum User { PressButton, DepressButton }

    #[derive(Action)]
    #[internal]
    pub enum Heater { Heat, Failover }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::PressButton if !self.heating_on => { self.heating_on = true; self.beeping = false; }
                User::DepressButton if self.heating_on => { self.heating_on = false; self.beeping = false; }
                _ => {}
            }
        }

        #[on(Heater)]
        fn heater(&mut self, action: Heater) {
            match action {
                Heater::Heat if self.heating_on && self.temperature < BOILING_TEMPERATURE => self.temperature += 1,
                Heater::Failover if self.heating_on && self.temperature >= BOILING_TEMPERATURE => { self.heating_on = false; self.beeping = true; }
                _ => {}
            }
        }

        #[invariant]
        fn beeps_only_when_off(&self) { if self.beeping { check!(!self.heating_on) } }

        #[postcondition]
        fn warms_only_when_on(&self, before: &Self) { if !before.heating_on { check!(self.temperature <= before.temperature) } }

        #[postcondition(after = Heater::Failover)]
        fn fails_over_at_boiling(&self) { check!(self.temperature == BOILING_TEMPERATURE); check!(self.beeping) }

        /// The user switches the kettle on but never off.
        #[assumption]
        fn never_switched_off() -> Assumption { without([User::DepressButton]) }

        /// Switched on and left on, the water boils and the kettle beeps.
        #[progress]
        fn boils(&self) -> LeadsTo { path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off()) }
    }
}
```

Do: convert `#[machine]`'s registration from the group impls (.6) to per-item attributes on one impl, binding `#[on(Actor)]` by the attribute's actor and checking the parameter type matches; remove the group attributes and any marker traits; implement the order/grouping check, the `&mut self` rule and `forbid(unsafe_code)`; refusals, each pinned by a trybuild `.stderr` fixture: out-of-order item, `#[on]` parameter not the named actor, two `#[on]` for one actor, an actor without `#[on]`, `&mut self` on a helper or a claim, a leftover group attribute (`#[transitions]`, `#[invariants]`, ...) naming the per-item form; rewrite Kettle and the `machine` macro fixture; all canonical goldens byte-identical (same enabled set); `kettle_trait` untouched; DESIGN.md's comparison shows the Scala `transitions` block beside the Rust `#[on(Heater)]` match, and notes that `State: Hash` excludes interior mutability and that thread-locals/static atomics fall under the capability Clippy policy; README updated. The claim value type is `LeadsTo`; `Progress` is free again and unused as a type.

**Order:** after .18 (done); before or after .19, both touch `machine.rs` in different functions, the second to land rebases; .7 waits for both. **Touches:** `model/umpire/dsl/**`, `model/umpire/tests/fixtures/kettle/model.rs`, `model/umpire/tests/fixtures/changed_state/model.rs`, `model/README.md`, `model/DESIGN.md`. **Feedback loop (R30):** paste `make model-time` into the done summary.

Implementation continuation: the existing shared changed_state attribute fixture must migrate too, without behavior/payload changes. The open .20 review finding still applies to the flat form: qualified, nested, imported and aliased Model impls must not bypass the read-only signature rule, while unrelated helper types stay usable. Compiler-resolved declaration checks may preserve ordinary method bodies; no runtime guards or body rewriting. Keep a real actor-variant catalog for postcondition(after=..) validation. All existing .19 recorder, fixed-assumption and manual-trait compatibility requirements remain; the unchanged manual trait Kettle may retain its legacy core Progress builder, while the attribute-facing claim value is LeadsTo.

Open owner question (asked asynchronously, one decision at a time): the example uses path!, previously deferred to fn215. The settled flat annotations, grouping, actor binding and read-only validation may proceed independently; retain the existing whenever/eventually/assuming expression as an interim compilable body until this boundary is resolved. Do not claim task acceptance or implement the deferred path notation spec without the answer. Preserve repro commit25841ad439 and the compiler identity proof during continuation; .22 still owns fixture directory renames.

**Owner-approved pass-through exception (2026-10-19):** the owner selected “Allow those compile-time declarations (recommended).” The machine macro may insert compile-time-only checking declarations inside function/const lexical blocks so Rust resolves local aliases, local trait projections and impl identity. Every original authored statement and method/impl header remains unchanged; enclosing blocks may gain declarations. No runtime guards, impl-header predicates, or new authoring restrictions are authorized by this exception. Cover Model refusals and unrelated helper acceptance with regression fixtures. This narrow permission supersedes the no-body-augmentation interpretation above, not the ordinary pass-through contract. The separate path! scope question remains unresolved.
## Acceptance
- One `impl Model` with per-item attributes `#[on(Actor)]`, `#[effect]`, `#[invariant]`, `#[postcondition]`/`#[postcondition(after = ..)]`, `#[assumption]`, `#[progress]`; no group attributes, no marker traits; every attribute pass-through.
- `#[machine]` refuses an out-of-place item, a non-grouped kind, an `#[on]` whose parameter is not the named actor, two `#[on]` for one actor, an actor without `#[on]`, `&mut self` outside `#[on]`/`#[effect]`, and a leftover group attribute, each with the message above and a trybuild `.stderr` fixture; omitting `_ => {}` yields rustc's exhaustiveness error; the module carries `#[forbid(unsafe_code)]`.
- Kettle and the `machine` macro fixture use the flat form; all canonical goldens are byte-identical; `kettle_trait` unchanged.
- DESIGN.md and README show the form; the full local gate passes within R30.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
