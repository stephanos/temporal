//! An empty outer range prevents evaluation of the later overflowing endpoint.
#![allow(arithmetic_overflow, clippy::reversed_empty_ranges, clippy::unused_self)]
use umpire_dsl::machine;
#[machine]
pub(crate) mod empty_first {
    use umpire_dsl::{Action, State};
    const LIMIT: i8 = 127;
    #[derive(State)]
    pub(crate) struct Model {
        #[starts(1..0)]
        spare: i8,
        #[starts(1..=(LIMIT+1))]
        value: i8,
    }
    #[derive(Action, Copy)]
    pub(crate) enum User {
        Idle,
    }
    impl Model {
        #[on(User)]
        fn step(&mut self, a: User) {
            match a {
                User::Idle => {}
            }
        }
    }
}
