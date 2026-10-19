//! A present element still evaluates the eager `unwrap_or` default.
use umpire_dsl::machine;

#[machine]
pub(crate) mod fallback {
    use imbl::Vector;
    use umpire_dsl::{Action, State, check};

    #[derive(State)]
    pub(crate) struct Model {
        pub values: Vector<u8>,
        #[starts(255..=255)]
        pub ceiling: u8,
    }
    #[derive(Action, Copy)]
    pub(crate) enum User {
        Push,
    }
    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Push if self.values.is_empty() => {
                    self.values.push_back(7);
                }
                User::Push => {}
            }
        }
        #[invariant]
        fn eager_default(&self) {
            if !self.values.is_empty() {
                check!(self.values.get(0).copied().unwrap_or(self.ceiling + 1) == 7);
            }
        }
    }
}
