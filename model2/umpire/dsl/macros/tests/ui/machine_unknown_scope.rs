use umpire_dsl::{Action, State, machine};

#[derive(State)]
struct Model {}
#[derive(Action)]
enum Nobody {}

#[machine(actions = (Nobody,))]
impl Model {
    fn unregistered() -> umpire_dsl::Scope<ModelAction> { umpire_dsl::Scope::default() }

    #[umpire_dsl::progress(within = Self::unregistered())]
    fn eventual() -> umpire_dsl::Progress<ModelWorld> { umpire_dsl::Progress::new("eventual") }
}

fn main() {}
