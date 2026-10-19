//! Small authored Model used with both compiled Rust and independent source lifting.

use umpire_dsl::machine;

/// A finite counter whose complete exploration exceeds the reference test's budget.
#[machine]
pub(crate) mod counter {
    use umpire_dsl::{Action, State, check};

    /// The counter's dynamic state.
    #[derive(State)]
    pub(crate) struct Model {
        /// Current counter value.
        #[starts(0..=0)]
        pub value: i32,
    }

    /// The counter's action catalog.
    #[derive(Action, Copy)]
    pub(crate) enum User {
        /// Advance until completion.
        Advance,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Advance if self.value < 3 => self.value += 1,
                User::Advance => {}
            }
        }

        #[invariant]
        fn within_range(&self) {
            check!(self.value <= 3);
        }

        #[progress]
        fn completes(&self) -> LeadsTo {
            whenever(self.value == 0).eventually(self.value == 3)
        }
    }
}
