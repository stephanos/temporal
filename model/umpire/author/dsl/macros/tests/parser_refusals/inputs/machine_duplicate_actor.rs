use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum User { Press }

    impl Model {
        #[on(User)]
        fn first(&mut self, action: User) { match action { User::Press => {} } }
        #[on(User)]
        fn second(&mut self, action: User) { match action { User::Press => {} } }
    }
}

fn main() {}

