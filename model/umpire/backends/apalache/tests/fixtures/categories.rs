//! Equal authored names retain their distinct property categories.
use umpire_dsl::machine;
#[machine]
pub(crate) mod categories {
    use umpire_dsl::{Action, State, check};
    #[derive(State)]
    pub(crate) struct Model {
        #[starts(0..=1)]
        pub(crate) value: u8,
    }
    #[derive(Action, Copy)]
    pub(crate) enum User {
        Reset,
    }
    impl Model {
        #[on(User)]
        fn step(&mut self, a: User) {
            match a {
                User::Reset => self.value = 0,
            }
        }
        #[invariant]
        fn arithmetic(&self) {
            check!(self.value != 0);
        }
        #[invariant]
        fn valid_step(&self) {
            check!(self.value != 0);
        }
        #[invariant]
        fn shared(&self) {
            check!(self.value == 0);
        }
    }
}
