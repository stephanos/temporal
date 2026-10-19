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
        fn user(&mut self, action: User) {
            match action { User::Press => self.rogue() }
        }
    }

    mod helper {
        impl super::Model {
            pub fn rogue(&mut self) { self.value = true; }
        }
    }
}

fn main() {}
