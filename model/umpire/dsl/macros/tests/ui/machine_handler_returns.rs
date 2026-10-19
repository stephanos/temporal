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
        fn press(&mut self) -> Option<()> { Some(()) }
    }
}

fn main() {}
