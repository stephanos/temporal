//! A failed check does not skip the next authored statement.
use umpire_dsl::machine;

#[machine]
pub(crate) mod sequential {
    use umpire_dsl::{Action, State, check};

    #[derive(State)]
    pub(crate) struct Model {
        pub divisor: i32,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Idle,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Idle if self.divisor == 0 => {}
                User::Idle => {}
            }
        }

        #[invariant]
        fn eager_checks(&self) {
            check!(false);
            check!(1 / self.divisor == 0);
        }
    }
}
