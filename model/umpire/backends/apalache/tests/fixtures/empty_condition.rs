//! A real accepted self-loop still evaluates conditions when the State has no fields.
#![allow(clippy::unused_self, clippy::collapsible_if)]
use umpire_dsl::machine;
#[machine]
pub(crate) mod empty_condition {
    use umpire_dsl::{Action, State, World, stay};
    #[derive(World)]
    pub(crate) struct World {
        pub divisor: u8,
        pub enabled: bool,
    }
    #[derive(State)]
    pub(crate) struct Model {}
    #[derive(Action, Copy)]
    pub(crate) enum User {
        Check,
    }
    impl Model {
        #[on(User)]
        fn step(&mut self, a: User, w: &World) {
            match a {
                User::Check => {
                    stay!();
                    if w.enabled {
                        if 1 / w.divisor > 0 {}
                    }
                }
            }
        }
    }
}
