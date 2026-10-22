use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State, check};
    #[derive(State)]
    struct Model { value: u8 }
    #[derive(Action)]
    enum Nobody {}

    impl Model {
        #[on(Nobody)]
        fn nobody(&mut self, action: Nobody) { match (self, action) {} }

        #[invariant]
        fn bounded(&self, limit: u8) { check!(self.value <= limit); }
    }
}

fn main() {}
