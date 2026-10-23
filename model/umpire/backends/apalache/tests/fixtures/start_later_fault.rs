//! A later domain faults after a concrete nonzero outer binding.
#![allow(arithmetic_overflow, clippy::unused_self)]
use umpire_dsl::machine;
#[machine]
pub(crate) mod later_fault {
    use umpire_dsl::{Action, State};
    const LIMIT: i8 = 127;
    #[derive(State)]
    pub(crate) struct Model {
        #[starts(7..=8)]
        earlier: i8,
        #[starts(1..=(LIMIT+1))]
        later: i8,
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
