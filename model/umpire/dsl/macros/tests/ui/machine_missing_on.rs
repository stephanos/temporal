use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum User { Press, Release }

    #[transitions]
    impl Model {
        #[on(User::Press)]
        fn press(&mut self) {}
    }
}

fn main() {}
