use umpire_dsl::machine;

/// The trigger stops holding after its first step; the target is still owed.
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
                User::Left if self.value == 1 => self.value = 9,
                _ => {}
            }
        }

        #[progress]
        fn reaches_target(&self) -> LeadsTo {
            path![self.value == 0, .., self.value == 9]
        }
    }
}
