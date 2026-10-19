//! Local scopes must not change the caller's saved value.
use umpire_dsl::machine;

#[machine]
pub(crate) mod lexical {
    use umpire_dsl::{Action, State, check};

    #[derive(State)]
    pub(crate) struct Model {
        #[starts(2..=2)]
        pub n: u8,
        pub done: bool,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Helper,
        Block,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Helper if !self.done => {
                    let x = self.n;
                    self.helper();
                    self.n = x;
                    self.done = true;
                }
                User::Block if !self.done => {
                    let x = self.n;
                    {
                        let x = 0u8;
                        self.n = x + 1;
                    }
                    self.n = x;
                    self.done = true;
                }
                _ => {}
            }
        }

        #[effect]
        fn helper(&mut self) {
            let x = 0u8;
            self.n = x + 1;
        }

        #[invariant]
        fn saved_value(&self) {
            check!(self.n == 2);
        }
    }
}
