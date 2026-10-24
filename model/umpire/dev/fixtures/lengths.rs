//! Lengths are `usize` in Rust and 32-bit integers in TLC. The front refuses a sum or product
//! of a length, which can exceed TLC's range where Rust's holds, and admits a difference,
//! quotient or remainder, which every back end computes alike.
use umpire::dsl::machine;

/// A sequence's length multiplied past `i32::MAX`: compiled, the claim holds; the front refuses
/// the product at its span, so no other back end reads it.
#[machine]
pub mod lengths {
    use imbl::Vector;
    use umpire::dsl::{Action, State, check};

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

/// Admitted length arithmetic: compiled, interpreted and TLC agree on every verdict.
#[machine]
pub mod differences {
    use imbl::Vector;
    use umpire::dsl::{Action, State, check};

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
        fn halves(&self) {
            check!(self.values.len() - self.values.len() / 2 <= 2);
            check!(self.values.len() % 2 < 2);
        }

        #[invariant]
        fn below_two(&self) {
            check!(self.values.is_empty() || self.values.len() - 1 < 2);
        }
    }
}

/// A length difference that underflows once the sequence holds two values: every back end
/// fails there.
#[machine]
pub mod underflow {
    use imbl::Vector;
    use umpire::dsl::{Action, State, check};

    /// A growing sequence.
    #[derive(State)]
    pub struct Model {
        /// Up to two values.
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
                User::Push if self.values.len() < 2 => {
                    self.values.push_back(1);
                }
                User::Push => {}
            }
        }

        #[invariant]
        fn remaining(&self) {
            check!(1 - self.values.len() < 2);
        }
    }
}
