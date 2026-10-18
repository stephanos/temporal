use umpire_dsl::{Action, State, machine};

#[derive(State)]
struct Model {}
#[derive(Action)]
enum User { Press }

#[machine(actions = (User,))]
impl Model {
    #[umpire_dsl::on(User::Press)]
    fn first(&mut self) -> Option<()> { Some(()) }
    #[umpire_dsl::on(User::Press)]
    fn second(&mut self) -> Option<()> { Some(()) }
}

fn main() {}
