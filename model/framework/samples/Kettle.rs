// Kettle in Rust, pretty but native.
// Hypothetical framework "umpire"; this file does not compile.
//
// No custom grammar and no proc macro that rewrites code. The framework ships four `macro_rules!`
// helpers (`transitions!`, `invariants!`, `postconditions!`, `progress!`) that expand to the
// obvious `match`, `Vec<Claim>` and builder calls, and bind the vars by destructuring `self`
// first, so guards and claims read bare names. `macro_rules!` is part of the language, costs no
// crate of its own, and rust-analyzer expands it: go-to-definition on `heating_on` lands on the
// field, rename works, errors point into this file. The only proc macros are the two derives, and
// `#[umpire::machine]`, which passes the impl through unchanged and records its tokens as the
// lifted tree for the emitters.

use umpire::prelude::*;

// the boiling point of water, in Celsius
const BOILING_TEMPERATURE: i32 = 100;
// the coldest water the kettle starts with, in Celsius
const COLDEST_TEMPERATURE: i32 = -40;

/// A kettle: a heating element the user switches on and off, which warms the water by one degree
/// a step and fails over, beeping, once the water boils.
#[derive(State)]
pub struct Kettle {
    #[starts(COLDEST_TEMPERATURE..=40)]
    temperature: i32,
    heating_on: bool,
    beeping: bool,
}

#[derive(Action)]
pub enum User {
    PressButton,
    DepressButton,
}

#[derive(Action, Internal)]
pub enum Heater {
    Heat,
    Failover,
}

#[umpire::machine]
impl Machine for Kettle {
    type Actions = (User, Heater);
    type Outcome = Outcome;

    // one step of the state machine: any enabled action; an arm with no effect refuses
    transitions! {
        User::PressButton    if !heating_on                                      => press_button,
        Heater::Failover     if heating_on && temperature >= BOILING_TEMPERATURE => failover,
        Heater::Heat         if heating_on && temperature < BOILING_TEMPERATURE  => heat,
        User::DepressButton  if heating_on                                       => depress_button,
    }

    invariants! {
        beeps_only_when_off:
            if beeping { check!(!heating_on) }
    }

    postconditions! {
        warms_only_when_on:
            if !before.heating_on { check!(temperature <= before.temperature) }
        fails_over_at_boiling:
            after!(Heater::Failover) {
                check!(temperature == BOILING_TEMPERATURE);
                check!(beeping);
            }
    }

    progress! {
        // Every claim above also holds for a kettle that never heats; this one says it boils.
        // Switched on and left on, the water boils and the kettle beeps.
        boils:
            whenever!(heating_on) { eventually!(beeping) } in NEVER_SWITCHED_OFF
    }
}

/// The user switches the kettle on but never off: one who switches it off keeps it from boiling,
/// whatever the heater does.
const NEVER_SWITCHED_OFF: Scope = Scope::without(&[User::DepressButton]);

impl Kettle {
    /// turn on the heating element
    fn press_button(&mut self) {
        self.heating_on = true;
        self.beeping = false;
    }
    /// turn off the heating element and beep
    fn failover(&mut self) {
        self.heating_on = false;
        self.beeping = true;
    }
    /// heat up the water by 1C; `heating_on` holds and `beeping` doesn't, since heat only fires
    /// while heating
    fn heat(&mut self) {
        self.temperature += 1;
    }
    /// turn off the heating element
    fn depress_button(&mut self) {
        self.heating_on = false;
        self.beeping = false;
    }
}

// What the helpers expand to, so nothing is hidden:
//
//   transitions! { A if g => e, ... }
//     fn step(&mut self, action: Self::Actions) -> Outcome {
//         let Self { temperature, heating_on, beeping } = *self;   // bare names for the guards
//         match action { A if g => { self.e(); Outcome::Accepted } ... _ => Outcome::Refused }
//     }
//
//   invariants! { name: body ... }
//     fn invariants(&self) -> Vec<Claim> {
//         let Self { temperature, heating_on, beeping } = *self;
//         vec![Claim::new("name", |check| body), ...]            // check! records a conjunct
//     }
//
//   postconditions! { ... }        same, with `before: &Self` in scope and `after!(A) { .. }`
//                                  expanding to `if action == A { .. }`
//   progress! { name: whenever!(p) { eventually!(q) } in scope }
//     Progress::new("name").whenever(|s| p).eventually(|s| q).within(scope)
//
// Test (would live in tests/kettle.rs), the same questions as the Scala suite:
//
//   #[test]
//   fn kettle_claims_hold() {
//       let result = Checker::explore::<Kettle>(1 << 20);
//       result.verify_all().assert_verified();
//       Checker::refuted::<Kettle>(invariant!(check!(!beeping)));
//   }
