use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum User { Press }

    #[transitions]
    impl Model {
        fn user(&mut self, action: User) { match action { User::Press => {} } }
    }
    #[postconditions]
    impl Model {
        #[after(User::Missing)]
        fn claim(&self) {}
    }
}

fn main() {}
