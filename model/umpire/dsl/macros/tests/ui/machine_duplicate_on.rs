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
        #[on(User::Press)]
        fn first(&mut self) {}
        #[on(User::Press)]
        fn second(&mut self) {}
    }
}

fn main() {}
