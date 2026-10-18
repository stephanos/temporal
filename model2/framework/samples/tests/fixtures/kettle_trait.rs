// Option D (custom, no macros): a trait and plain Rust, stateright's shape with Umpire's claims.
// Hypothetical framework "umpire"; does not compile.
//
// Zero macros beyond std derives. The Model is a trait implementation; the action catalog is an
// explicit enum and an explicit list; claims are plain functions registered by name in one place.
// This is the floor: what Rust gives with no DSL at all. Everything else is this plus sugar.
// Lifting cannot be done from the compiler here; it would be a `syn` pass over the source in
// build.rs, like go/ast.

use umpire::{Checker, Claim, Machine, Outcome, Progress, Scope};

const BOILING_TEMPERATURE: i32 = 100;
const COLDEST_TEMPERATURE: i32 = -40;

/// A kettle: a heating element the user switches on and off, which warms the water by one degree
/// a step and fails over, beeping, once the water boils.
#[derive(Clone, PartialEq, Eq, Hash, Debug)]
pub struct Kettle {
    temperature: i32,
    heating_on: bool,
    beeping: bool,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug)]
pub enum Act {
    UserPressButton,
    UserDepressButton,
    HeaterHeat,
    HeaterFailover,
}

impl Machine for Kettle {
    type Action = Act;

    fn starts() -> Vec<Self> {
        (COLDEST_TEMPERATURE..=40)
            .map(|temperature| Kettle { temperature, heating_on: false, beeping: false })
            .collect()
    }

    fn actions() -> &'static [Act] {
        &[Act::UserPressButton, Act::UserDepressButton, Act::HeaterHeat, Act::HeaterFailover]
    }

    fn internal(action: Act) -> bool {
        matches!(action, Act::HeaterHeat | Act::HeaterFailover)
    }

    /// One step of the state machine: any enabled action. `None` refuses.
    fn step(&self, action: Act) -> Option<Self> {
        let mut next = self.clone();
        match action {
            Act::UserPressButton if !self.heating_on => {
                next.heating_on = true;
                next.beeping = false;
            }
            Act::HeaterFailover if self.heating_on && self.temperature >= BOILING_TEMPERATURE => {
                next.heating_on = false;
                next.beeping = true;
            }
            Act::HeaterHeat if self.heating_on && self.temperature < BOILING_TEMPERATURE => {
                next.temperature += 1;
            }
            Act::UserDepressButton if self.heating_on => {
                next.heating_on = false;
                next.beeping = false;
            }
            _ => return None,
        }
        Some(next)
    }

    fn invariants() -> Vec<Claim<Self>> {
        vec![Claim::new("beeps_only_when_off", |k| !k.beeping || !k.heating_on)]
    }

    fn postconditions() -> Vec<StepClaim<Self>> {
        vec![
            StepClaim::new("warms_only_when_on", |before, after, _| {
                before.heating_on || after.temperature <= before.temperature
            }),
            StepClaim::after("fails_over_at_boiling", Act::HeaterFailover, |_, after| {
                after.temperature == BOILING_TEMPERATURE && after.beeping
            }),
        ]
    }

    fn progress() -> Vec<Progress<Self>> {
        // The user switches the kettle on but never off: one who switches it off keeps it from
        // boiling, whatever the heater does.
        let never_switched_off = Scope::without(&[Act::UserDepressButton]);
        // Every claim above also holds for a kettle that never heats; this one says it boils.
        // Switched on and left on, the water boils and the kettle beeps.
        vec![Progress::<Self>::new("boils")
            .whenever(|k| k.heating_on)
            .eventually(|k| k.beeping)
            .within(never_switched_off)]
    }
}

// tests/kettle.rs
//   #[test]
//   fn kettle_claims_hold() {
//       let result = Checker::explore::<Kettle>(1 << 20);
//       result.verify_all().assert_verified();
//       Checker::refuted::<Kettle>(Claim::new("never_beeps", |k| !k.beeping));
//   }
