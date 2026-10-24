//! The sample Kettle without its claims, for the suite's message fixtures.

use umpire_dsl::machine;

/// A kettle controlled by its user and heating element.
#[machine]
pub mod kettle {
    use umpire_dsl::{Action, State};

    const BOILING_TEMPERATURE: i32 = 100;

    /// Model's dynamic state.
    #[derive(State)]
    pub struct Model {
        /// Current temperature.
        #[starts(20..=20)]
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
    }
}

/// A Kettle state by its fields.
pub(crate) const fn at(temperature: i32, heating_on: bool, beeping: bool) -> kettle::Model {
    kettle::Model { temperature, heating_on, beeping }
}
