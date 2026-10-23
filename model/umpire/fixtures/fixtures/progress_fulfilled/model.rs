// No user action is ever enabled, so its handler never reads the state.
#![allow(clippy::unused_self)]

use umpire_dsl::machine;

/// The target already holds where the trigger does, with no enabled action.
#[machine]
pub mod graph {
    use umpire_dsl::{Action, State, path};

    #[derive(State)]
    pub struct Model {
        pub value: u8,
    }

    #[derive(Action, Copy)]
    pub enum User {
        Enter,
        Left,
        Right,
        Escape,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Enter | User::Left | User::Right | User::Escape => {}
            }
        }

        #[progress]
        fn reaches_target(&self) -> LeadsTo {
            path![self.value == 0, .., true]
        }
    }
}
