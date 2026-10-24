use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model { value: bool }
    #[derive(Action)]
    enum User { Press }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action { User::Press => self.rogue() }
        }
    }

    mod helper {
        use super::Model as Imported;
        type Alias = Imported;
        impl Alias {
            pub fn rogue(&mut self) { self.value = true; }
        }
    }
}

fn main() {}
