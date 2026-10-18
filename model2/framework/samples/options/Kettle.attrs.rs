// Option C (custom): attribute macros on plain functions, statig's mechanics with Umpire's shape.
// Hypothetical framework "umpire"; does not compile.
//
// Every item is an ordinary Rust item. The attributes (`#[on]`, `#[invariant]`,
// `#[postcondition]`, `#[progress]`) are pass-through: they leave the function untouched and
// register it (name, kind, which action it answers) in a table the `#[machine]` attribute on the
// impl collects. No macro_rules, no grammar, no destructuring tricks. rust-analyzer sees plain
// functions. Lifting reads the same impl tokens.
//
// Trade: `self.` everywhere and `-> bool` claims instead of `check` statements; in exchange the
// only "DSL" is the attribute names, and a transition is one function you can call in a unit test.

use umpire::prelude::*;

const BOILING_TEMPERATURE: i32 = 100;
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

#[machine(actions = (User, Heater))]
impl Kettle {
    // Transitions: one function per action; returning `None` refuses. The attribute binds the
    // function to the action, and the compiler checks every action has exactly one `#[on]`.
    #[on(User::PressButton)]
    fn press_button(&mut self) -> Option<()> {
        (!self.heating_on).then(|| {
            self.heating_on = true;
            self.beeping = false;
        })
    }

    #[on(Heater::Failover)]
    fn failover(&mut self) -> Option<()> {
        (self.heating_on && self.temperature >= BOILING_TEMPERATURE).then(|| {
            self.heating_on = false;
            self.beeping = true;
        })
    }

    #[on(Heater::Heat)]
    fn heat(&mut self) -> Option<()> {
        (self.heating_on && self.temperature < BOILING_TEMPERATURE).then(|| {
            self.temperature += 1;
        })
    }

    #[on(User::DepressButton)]
    fn depress_button(&mut self) -> Option<()> {
        self.heating_on.then(|| {
            self.heating_on = false;
            self.beeping = false;
        })
    }

    // Invariants: a predicate over the state.
    #[invariant]
    fn beeps_only_when_off(&self) -> bool {
        !self.beeping || !self.heating_on
    }

    // Postconditions: `before` is an ordinary parameter; `after = ...` scopes the claim.
    #[postcondition]
    fn warms_only_when_on(&self, before: &Self) -> bool {
        before.heating_on || self.temperature <= before.temperature
    }

    #[postcondition(after = Heater::Failover)]
    fn fails_over_at_boiling(&self, _before: &Self) -> bool {
        self.temperature == BOILING_TEMPERATURE && self.beeping
    }

    // Scopes and progress: values and predicates.
    /// The user switches the kettle on but never off: one who switches it off keeps it from
    /// boiling, whatever the heater does.
    #[scope]
    const NEVER_SWITCHED_OFF: Scope = Scope::without(&[User::DepressButton]);

    /// Every claim above also holds for a kettle that never heats; this one says it boils.
    /// Switched on and left on, the water boils and the kettle beeps.
    #[progress(within = Self::NEVER_SWITCHED_OFF)]
    fn boils(&self) -> Progress {
        Progress::whenever(self.heating_on).eventually(|k: &Self| k.beeping)
    }
}

// tests/kettle.rs
//   #[test]
//   fn kettle_claims_hold() {
//       let result = Checker::explore::<Kettle>(1 << 20);
//       result.verify_all().assert_verified();
//       Checker::refuted::<Kettle>(|k| !k.beeping);
//   }
