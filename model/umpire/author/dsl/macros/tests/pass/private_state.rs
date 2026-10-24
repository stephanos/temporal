#![allow(dead_code, private_interfaces)]

#[umpire_dsl::machine]
mod private {
    use umpire_dsl::{Action, State};

    #[derive(Action)]
    enum User {
        Toggle,
    }

    #[derive(State)]
    struct Model {
        enabled: bool,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            self.enabled = !self.enabled;
        }
    }
    pub fn check() {
        use umpire_dsl::Machine;
        let world = world();
        assert_eq!(world.actions().len(), 1);
    }
}

fn main() {
    private::check();
}
