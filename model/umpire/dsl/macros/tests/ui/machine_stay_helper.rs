use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum User { Press }

    fn idle() {
        umpire_dsl::stay!();
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) { match action { User::Press => idle() } }
    }
}

fn main() {}
