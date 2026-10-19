use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum Nobody {}

    impl Model {
        #[on(Nobody)]
        fn nobody(&mut self, action: Nobody) { match (self, action) {} }

        #[invariant]
        fn claim(&self) -> bool { true }
    }
}

fn main() {}
