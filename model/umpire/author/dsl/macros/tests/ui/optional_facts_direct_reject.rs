use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, Rejection, State, reject};

    #[derive(State)]
    struct Model {}

    #[derive(Action)]
    enum User { Step }

    #[derive(Debug)]
    enum Fact { A }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) -> Option<Fact> {
            match action {
                User::Step => {
                    reject!(Rejection::FailedPrecondition);
                    Some(Fact::A)
                }
            }
        }
    }
}

fn main() {}
