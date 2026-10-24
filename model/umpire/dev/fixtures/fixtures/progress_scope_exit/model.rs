use umpire::dsl::machine;

/// A fair escape the scope excludes cannot rescue a self-loop.
#[machine]
pub mod graph {
    use umpire::dsl::{Action, State, path, stay};

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
                User::Left if self.value == 0 => stay!(),
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
        fn escape_excluded() -> Assumption {
            fair([Environment::Escape]).without([Environment::Escape])
        }

        #[progress]
        fn reaches_target(&self) -> LeadsTo {
            path![true, .., self.value == 9].assuming(Self::escape_excluded())
        }
    }
}
