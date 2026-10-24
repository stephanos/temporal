use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum User { Press }

    #[derive(Action)]
    enum Clock { Tick }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: Clock) { match action { Clock::Tick => {} } }
    }
}

fn main() {}
