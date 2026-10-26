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

        #[progress]
        #[within(Self::unregistered)]
        fn eventual(&self) -> LeadsTo { whenever(true).eventually(true) }
    }
}

fn main() {}
