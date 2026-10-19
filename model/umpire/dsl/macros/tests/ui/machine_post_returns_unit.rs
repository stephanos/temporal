use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum Nobody {}

    #[transitions]
    impl Model {
        fn nobody(&mut self, action: Nobody) { match action {} }
    }

    #[postconditions]
    impl Model {
        fn claim(&self) -> () {}
    }
}

fn main() {}
