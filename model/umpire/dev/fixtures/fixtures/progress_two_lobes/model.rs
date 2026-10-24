// No user action is ever enabled, so its handler never reads the state.
#![allow(clippy::unused_self)]

use umpire::dsl::machine;

/// Two fair internal actions alternate between lobes that each reach the target.
#[machine]
pub mod graph {
    use umpire::dsl::{Action, State, path};

    #[derive(State)]
    pub struct Model {
        pub value: u8,
    }

    #[derive(Action, Copy)]
    pub enum User {
        Enter,
        Escape,
    }

    #[derive(Action, Copy)]
    #[internal]
    pub enum Environment {
        Left,
        Right,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Enter | User::Escape => {}
            }
        }

        #[on(Environment)]
        fn environment(&mut self, action: Environment) {
            match action {
                Environment::Left if self.value == 0 => self.value = 1,
                Environment::Left if self.value == 1 => self.value = 0,
                Environment::Left if self.value == 2 => self.value = 9,
                Environment::Right if self.value == 0 => self.value = 2,
                Environment::Right if self.value == 1 => self.value = 9,
                Environment::Right if self.value == 2 => self.value = 0,
                _ => {}
            }
        }

        #[assumption]
        fn alternates() -> Assumption {
            fair([Environment::Left, Environment::Right])
        }

        #[progress]
        fn reaches_target(&self) -> LeadsTo {
            path![true, .., self.value == 9].assuming(Self::alternates())
        }
    }
}
