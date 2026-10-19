//! A finite collection authored with the existing machine surface.
use umpire_dsl::machine;

#[machine]
pub(crate) mod collection {
    use imbl::Vector;
    use umpire_dsl::{Action, State, check};

    #[derive(State)]
    pub(crate) struct Model {
        pub values: Vector<u8>,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Push,
        Clear,
        Reset,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Push if self.values.len() < 2 => {
                    self.values.push_back(7);
                }
                User::Clear if !self.values.is_empty() => {
                    self.values = Vector::new();
                }
                User::Reset if !self.values.is_empty() => {
                    self.values = Vector::default();
                }
                _ => {}
            }
        }

        #[invariant]
        fn contents(&self) {
            check!(self.values.len() <= 2);
            check!(self.values.get(0).copied().unwrap_or(7) == 7);
            check!(self.values.get(1).copied().unwrap_or(7) == 7);
            check!(self.values.get(2usize).copied().unwrap_or(9) == 9);
            check!(self.values.get(0).is_some() != self.values.is_empty());
            check!(self.values.get(2).is_none());
        }

        #[invariant]
        fn never_two(&self) {
            check!(self.values.len() != 2);
        }
    }
}
