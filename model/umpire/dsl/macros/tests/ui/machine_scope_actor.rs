use umpire_dsl::{Action, machine};

#[derive(Action)]
enum Other { Unknown }

#[machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {}
    #[derive(Action)]
    enum Nobody {}

    #[assumptions]
    impl Model {
        fn unknown_actor() -> Assumption { without([super::Other::Unknown]) }
    }
}

fn main() {}
