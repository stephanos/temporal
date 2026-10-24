use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State, stay};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum User { Press }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) { match action { User::Press => self.pause() } }

        #[effect]
        fn pause(&mut self) {
            stay!();
        }
    }
}

fn main() {}
