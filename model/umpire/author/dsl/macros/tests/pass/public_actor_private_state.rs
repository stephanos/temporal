#![allow(dead_code, private_interfaces)]

#[umpire_dsl::machine]
mod private_model {
    use umpire_dsl::{Action, Path, State, path};

    #[derive(State)]
    struct Model {
        done: bool,
    }

    #[derive(Action)]
    pub enum User {
        Toggle,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            self.done = !self.done;
        }
    }

    pub fn check() {
        let states: Path<Model> = path![Model { done: false }, .., |state| state.done];
        assert_eq!(states.alternative_count(), Ok(1));
        let actions = path![User::Toggle, |state| state.done];
        assert_eq!(actions.alternative_count(), Ok(1));
    }
}

fn main() {
    private_model::check();
}
