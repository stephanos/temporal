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

    #[transitions]
    impl Model {
        fn nobody(&mut self, action: Nobody) { match action {} }
    }

    #[assumptions]
    impl Model {
        fn unknown_actor() -> Assumption { without([super::Other::Unknown]) }
    }
}

fn main() {}
