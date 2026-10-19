use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum Nobody {}

    #[progress]
    impl Model {
        #[within]
        fn claim(&self) -> Progress { whenever(true) }
    }
}

fn main() {}
