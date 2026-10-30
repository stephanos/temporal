use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    struct Model {}

    #[derive(Action)]
    enum User { Step }

    #[derive(Debug)]
    enum Fact { A(bool) }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) -> Option<Fact> {
            match action { User::Step => Some(Fact::A(true)) }
        }
    }
}

fn main() {}
