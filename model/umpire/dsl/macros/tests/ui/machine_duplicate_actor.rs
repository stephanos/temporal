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
        fn first(&mut self, action: User) { match action { User::Press => {} } }
        fn second(&mut self, action: User) { match action { User::Press => {} } }
    }
}

fn main() {}

