use umpire_dsl::{Action, State, machine, on};

#[derive(State)]
struct Model {}
#[derive(Action)]
enum User { Press, Release }

#[machine(actions = (User,))]
impl Model {
    #[on(User::Press)]
    fn press(&mut self) -> Option<()> { Some(()) }
}

fn main() {}
