use umpire_dsl::machine;

/// An excluded prefix enters a self-loop that the scope keeps, refuting the claim there.
#[machine]
pub mod graph {
    use umpire_dsl::{Action, State, path, stay};

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
                User::Enter if self.value == 0 => self.value = 1,
                User::Left if self.value == 1 => stay!(),
                _ => {}
            }
        }

        #[assumption]
        fn no_entry() -> Assumption {
            without([User::Enter])
        }

        #[progress]
        fn reaches_target(&self) -> LeadsTo {
            path![self.value == 1, .., self.value == 9].assuming(Self::no_entry())
        }
    }
}
