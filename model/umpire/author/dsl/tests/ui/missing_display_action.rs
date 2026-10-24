use umpire_dsl::{Machine, Steps};

#[derive(Clone, PartialEq, Eq, Hash, Debug)]
struct Action;

struct World;

impl Machine for World {
    type State = u8;
    type Action = Action;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<u8> {
        vec![0]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, _: &u8, _: Action) -> Steps<u8, (), ()> {
        Steps::new()
    }
}

fn main() {}
