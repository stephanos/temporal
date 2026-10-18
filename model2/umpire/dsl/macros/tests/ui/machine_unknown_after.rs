use umpire_dsl::{Action, State, machine};

#[derive(State)]
struct Model {}
#[derive(Action)]
enum User { Press }

#[machine(actions = (User,))]
impl Model {
    #[umpire_dsl::on(User::Press)]
    fn press(&mut self) -> Option<()> { Some(()) }
    #[umpire_dsl::postcondition(after = User::Missing)]
    fn claim(&self, _before: &Self) -> bool { true }
}

fn main() {}
