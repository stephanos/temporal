use umpire_dsl::machine;

/// A fair escape is enabled only at one state of the cycle.
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
    }

    #[derive(Action, Copy)]
    #[internal]
    pub enum Environment {
        Escape,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Left if self.value == 0 => self.value = 1,
                User::Right if self.value == 1 => self.value = 0,
                _ => {}
            }
        }

        #[on(Environment)]
        fn environment(&mut self, action: Environment) {
            match action {
                Environment::Escape if self.value == 0 => self.value = 9,
                Environment::Escape => {}
            }
        }

        #[assumption]
        fn escapes() -> Assumption {
            fair([Environment::Escape])
        }

        #[progress]
        fn reaches_target(&self) -> LeadsTo {
            path![true, .., self.value == 9].assuming(Self::escapes())
        }
    }
}
