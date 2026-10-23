use umpire_dsl::machine;

/// The only enabled step is an accepted self-loop away from the target.
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
                User::Left if self.value == 0 => stay!(),
                _ => {}
            }
        }

        #[progress]
        fn reaches_target(&self) -> LeadsTo {
            path![true, .., self.value == 9]
        }
    }
}
