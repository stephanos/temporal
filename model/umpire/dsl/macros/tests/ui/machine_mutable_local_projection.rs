use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model { value: bool }
    #[derive(Action)]
    enum User { Press }

    trait Change {
        fn rogue(&mut self);
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action { User::Press => self.rogue() }
        }
    }

    fn declare() {
        struct Carrier;
        trait Resolve { type Target; }
        impl Resolve for Carrier { type Target = Model; }
        impl Change for <Carrier as Resolve>::Target {
            fn rogue(&mut self) { self.value = true; }
        }
    }
}

fn main() {}
