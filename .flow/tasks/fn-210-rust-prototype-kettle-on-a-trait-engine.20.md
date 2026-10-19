---
satisfies: [R6]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.20 Transitions as one match per actor; no #[on]

## Description
**Owner decision (2026-10-19):** the `#[transitions]` block holds one method per actor, `fn user(&mut self, action: User)` and `fn heater(&mut self, action: Heater)`, each a `match` whose arms are the Scala `on(x): if guard then effect` lines (match guards are the guards); no `#[on]` attribute. `#[machine]` binds a method to an actor by its parameter type, which it knows from the module's `#[derive(Action)]` enums, and dispatches `step` through it; "enabled iff the state changed" (.18) is unchanged, so `_ => {}` is "disabled". rustc's exhaustiveness replaces the macro's missing-handler error when no `_` arm is written. Not one `step` for the whole Model: that would make the author write the generated `kettle::Action`.

```rust
#[transitions]
impl Model {
    fn user(&mut self, action: User) {
        match action {
            User::PressButton if !self.heating_on => {
                self.heating_on = true;
                self.beeping = false;
            }
            User::DepressButton if self.heating_on => {
                self.heating_on = false;
                self.beeping = false;
            }
            _ => {}
        }
    }

    fn heater(&mut self, action: Heater) {
        match action {
            Heater::Heat if self.heating_on && self.temperature < BOILING_TEMPERATURE => {
                self.temperature += 1;
            }
            Heater::Failover if self.heating_on && self.temperature >= BOILING_TEMPERATURE => {
                self.heating_on = false;
                self.beeping = true;
            }
            _ => {}
        }
    }
}
```

Why: closest to the Scala block; inputs later become destructuring arms (`User::Send { from, to, denom, amount } if .. =>`) with no handler-signature rule; each arm is one TLA+ action `guard /\ effect` for fn-212; the macro shrinks. Effects shared between arms are ordinary helper methods.

Do: replace `#[on]` registration in `dsl/macros/src/machine.rs` with per-actor binding by parameter type; remove the `on` attribute from `umpire-dsl`'s exports; refuse a `#[transitions]` method whose single parameter is not one of the module's actor enums (`a transitions method takes one actor enum: fn user(&mut self, action: User)`), two methods for one actor (`actor User already has a transitions method`), and an actor with no method (`actor Heater has no transitions method`), each pinned by a trybuild `.stderr` fixture (retire `machine_duplicate_on` / `machine_missing_on`); rewrite Kettle's `#[transitions]` block and the `machine` macro fixture; all canonical goldens byte-identical (same enabled set); `kettle_trait` untouched; DESIGN.md's comparison shows the Scala `transitions` block beside the Rust one; README updated.

**Order:** after .18 (done); before or after .19, both touch `machine.rs` in different functions, the second to land rebases; .7 waits for both. **Touches:** `model/umpire/dsl/**`, `model/umpire/tests/fixtures/kettle/model.rs`, `model/umpire/tests/fixtures/changed_state/model.rs`, `model/README.md`, `model/DESIGN.md`. **Feedback loop (R30):** paste `make model-time` into the done summary.

**Read-only outside transitions (owner, 2026-10-19):** `&mut self` is allowed only in `#[transitions]` and in an optional `#[effects]` block, a sixth group placed right after transitions in R44's order (Scala's `object effects`, for effects shared between arms, e.g. `fn switch_off(&mut self, beep: bool)`). Every other method of `Model` in the module, in marked groups or in unmarked helper impls, must take `&self` or no `self`; `#[machine]` checks every impl's signatures and refuses `<method> takes &mut self; only transitions and effects change the state` at the method's span (clearer than rustc's borrow error inside the expansion). `#[machine]` also emits `#[forbid(unsafe_code)]` on the module. Two more trybuild fixtures: a `&mut self` invariant and a `&mut self` helper in an unmarked impl. Kettle needs no `#[effects]` block (its effects are inline); the `machine` macro fixture gains one to pin the group. DESIGN.md notes that `State: Hash` already excludes interior mutability and that thread-locals/atomics in statics fall under the capability Clippy policy, not this check.

Implementation preflight: migrate the shared changed_state attribute fixture as a directly required consumer, preserving every behavior payload. Keep a separate actual actor-variant catalog when validating #[after(..)] after transitioning handlers to actor-method registrations. Insert effects in all group-rank calculations without regressing .19 claim/assumption behavior. Migrate UI scaffolding to valid actor methods so its intended errors are not masked. An empty actor still gets a transition method using an empty exhaustive match. Keep macro fixture source and expansion artifacts under one lane; root build coordination remains serial. Add refusal controls for mutable receivers in invariants and unmarked Model helpers, misplaced/duplicate effects, and unsafe code under an attempted allowance. No model naming changes here; .22 owns those.
## Acceptance
- `#[transitions]` holds one `fn <actor>(&mut self, action: <Actor>)` per actor; `#[machine]` dispatches by parameter type; no `#[on]` attribute exists.
- A non-actor parameter, a second method for an actor, and an actor without a method are refused with the messages above, pinned by trybuild `.stderr` fixtures; omitting `_ => {}` yields rustc's exhaustiveness error.
- Kettle and the `machine` macro fixture use the per-actor match form; all canonical goldens are byte-identical; `kettle_trait` unchanged.
- DESIGN.md and README show the form; the full local gate passes within R30.
- `&mut self` outside `#[transitions]`/`#[effects]` is refused with the message above (invariant and helper fixtures); `#[effects]` is accepted only directly after `#[transitions]`; the module carries `#[forbid(unsafe_code)]`.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
