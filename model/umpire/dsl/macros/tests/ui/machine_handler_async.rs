use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model { value: bool }
    #[derive(Action)]
    enum User { Press }

    #[transitions]
    impl Model {
        #[on(User::Press)]
        async fn press(&mut self) { self.value = true; }
    }
}

fn main() {}
