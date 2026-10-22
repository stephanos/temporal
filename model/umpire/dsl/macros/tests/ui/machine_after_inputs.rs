use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State, check};
    #[derive(State)]
    struct Model { value: u8 }
    #[derive(Action)]
    enum User { Set(#[inputs(0..=1)] u8) }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) { match action { User::Set(value) => self.value = value } }

        #[postcondition(after = User::Set)]
        fn set(&self) { check!(self.value <= 1); }
    }
}

fn main() {}
