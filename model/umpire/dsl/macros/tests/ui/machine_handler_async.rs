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
        async fn user(&mut self, action: User) { match action { User::Press => { self.value = true; } } }
    }
}

fn main() {}
