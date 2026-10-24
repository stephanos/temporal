//! Narrow signed remainder has a checked MIN/-1 failure.
use umpire_dsl::machine;

#[machine]
pub(crate) mod remainder {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(crate) struct Model {
        #[starts(-128..=-128)]
        pub value: i8,
        #[starts(-1..=-1)]
        pub divisor: i8,
        pub done: bool,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Calculate,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Calculate if !self.done => {
                    let remainder = self.value % self.divisor;
                    self.value = remainder;
                    self.done = true;
                }
                User::Calculate => {}
            }
        }
    }
}
