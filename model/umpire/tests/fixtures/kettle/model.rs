//! Model authored with the attribute surface.

use umpire_dsl::{Claim, machine};

/// A kettle controlled by its user and heating element.
#[machine]
pub mod kettle {
    use umpire_dsl::{Action, State};

    const BOILING_TEMPERATURE: i32 = 100;
    const COLDEST_TEMPERATURE: i32 = -40;

    /// Model's dynamic state.
    #[derive(State)]
    pub struct Model {
        /// Current temperature.
        #[starts(COLDEST_TEMPERATURE..=40)]
        pub temperature: i32,
        /// Whether heating is enabled.
        pub heating_on: bool,
        /// Whether the kettle beeps.
        pub beeping: bool,
    }

    /// Actions controlled by the user.
    #[derive(Action)]
    pub enum User {
        /// Enable heating.
        PressButton,
        /// Disable heating.
        DepressButton,
    }

    /// Actions controlled by the heating element.
    #[derive(Action)]
    #[internal]
    pub enum Heater {
        /// Raise the temperature.
        Heat,
        /// Stop heating and beep at boiling.
        Failover,
    }

    #[transitions]
    impl Model {
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
    }

    #[invariants]
    impl Model {
        fn beeps_only_when_off(&self) -> bool {
            !self.beeping || !self.heating_on
        }
    }

    #[postconditions]
    impl Model {
        fn warms_only_when_on(&self, before: &Self) -> bool {
            before.heating_on || self.temperature <= before.temperature
        }

        #[after(Heater::Failover)]
        fn fails_over_at_boiling(&self) -> bool {
            self.temperature == BOILING_TEMPERATURE && self.beeping
        }
    }

    #[scopes]
    impl Model {
        /// The user switches the kettle on but never off.
        fn never_switched_off() -> Scope {
            without([User::DepressButton])
        }
    }

    #[progress]
    impl Model {
        /// Switched on and left on, the water boils and the kettle beeps.
        #[within(Self::never_switched_off)]
        fn boils(&self) -> Progress {
            whenever(self.heating_on).eventually(self.beeping)
        }
    }
}

umpire_testing::fixture!(kettle::world(), 1 << 20, Claim::new("never_beeps", |_, state: &kettle::Model| (!state.beeping).into()));
