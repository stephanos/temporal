use umpire_dsl::{Machine, Steps};

#[derive(Clone, PartialEq, Eq, Hash, Debug)]
struct Action;

struct World;

impl Machine for World {
    type State = ();
    type Action = Action;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<()> {
        vec![()]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, _: &(), _: Action) -> Steps<(), (), ()> {
        Steps::new()
    }
}

fn main() {}
