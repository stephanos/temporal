use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum Nobody {}

    #[invariants]
    impl Model {
        async fn claim(&self) {}
    }
}

fn main() {}
