//! Model authored with the attribute surface.

use umpire_dsl::machine;

/// A kettle controlled by its user and heating element.
#[machine]
pub mod kettle {
    use umpire_dsl::{Action, State, check, path};

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
    #[derive(Action, Copy)]
    pub enum User {
        /// Enable heating.
        PressButton,
        /// Disable heating.
        DepressButton,
    }

    /// Actions controlled by the heating element.
    #[derive(Action, Copy)]
    #[internal]
    pub enum Heater {
        /// Raise the temperature.
        Heat,
        /// Stop heating and beep at boiling.
        Failover,
    }

    impl Model {
        #[on(User)]
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

        #[on(Heater)]
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

        #[invariant]
        fn beeps_only_when_off(&self) {
            if self.beeping {
                check!(!self.heating_on);
            }
        }

        #[postcondition]
        fn warms_only_when_on(&self, before: &Self) {
            if !before.heating_on {
                check!(self.temperature <= before.temperature);
            }
        }

        #[postcondition(after = Heater::Failover)]
        fn fails_over_at_boiling(&self) {
            check!(self.temperature == BOILING_TEMPERATURE);
            check!(self.beeping);
        }

        /// The user switches the kettle on but never off.
        #[assumption]
        fn never_switched_off() -> Assumption {
            without([User::DepressButton])
        }

        /// Switched on and left on, the water boils and the kettle beeps.
        #[progress]
        fn boils(&self) -> LeadsTo {
            path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off())
        }
    }
}
