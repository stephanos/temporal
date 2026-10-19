use umpire_dsl::machine;

/// Every reached state triggers, and a cycle avoids the target.
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
                User::Enter if self.value == 0 => self.value = 1,
                User::Left if self.value == 1 => self.value = 2,
                User::Right if self.value == 2 => self.value = 1,
                _ => {}
            }
        }

        #[progress]
        fn reaches_target(&self) -> LeadsTo {
            path![true, .., self.value == 9]
        }
    }
}
