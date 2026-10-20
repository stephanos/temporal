use umpire_dsl::machine;

/// The target is reached before the walk settles into a self-loop beyond it.
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
                User::Enter if self.value == 0 => self.value = 9,
                User::Left if self.value == 9 => self.value = 1,
                User::Right if self.value == 1 => stay!(),
                _ => {}
            }
        }

        #[progress]
        fn reaches_target(&self) -> LeadsTo {
            path![self.value == 0, .., self.value == 9]
        }
    }
}
