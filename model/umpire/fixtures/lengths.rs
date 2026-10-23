//! Lengths are `usize` arithmetic, wider than any admitted integer: compiled, the claim
//! holds, since no intermediate product overflows `usize`.
use umpire_dsl::machine;

/// A sequence's length multiplied past `i32::MAX`.
#[machine]
pub mod lengths {
    use imbl::Vector;
    use umpire_dsl::{Action, State, check};

    /// A growing sequence.
    #[derive(State)]
    pub struct Model {
        /// Up to three values.
        pub values: Vector<u8>,
    }

    /// Growth.
    #[derive(Action, Copy)]
    pub enum User {
        /// Appends one value.
        Push,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Push if self.values.len() < 3 => {
                    self.values.push_back(1);
                }
                User::Push => {}
            }
        }

        #[invariant]
        fn wide(&self) {
            check!(self.values.len() * 1_000_000_000 != 7);
        }
    }
}
