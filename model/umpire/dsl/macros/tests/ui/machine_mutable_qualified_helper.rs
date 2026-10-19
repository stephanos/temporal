use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub struct Model { value: bool }
    #[derive(Action)]
    enum User { Press }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action { User::Press => self.rogue() }
        }
    }

    impl crate::model::Model {
        fn rogue(&mut self) { self.value = true; }
    }
}

fn main() {}
