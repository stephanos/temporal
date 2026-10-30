use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State, path};
    #[derive(State)]
    struct Model { value: u8 }
    #[derive(Action)]
    enum Nobody {}

    impl Model {
        #[on(Nobody)]
        fn nobody(&mut self, action: Nobody) { match (self, action) {} }

        #[progress]
        fn claim(&self) -> LeadsTo { path![self.value == 0, 1..3, self.value == 1] }
    }
}

fn main() {}
