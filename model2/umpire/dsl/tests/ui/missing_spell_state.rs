use umpire_dsl::{Machine, Steps};

#[derive(Clone, PartialEq, Eq, Hash, Debug)]
struct State;

struct World;

impl Machine for World {
    type State = State;
    type Action = ();
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<State> {
        vec![State]
    }
    fn actions(&self) -> Vec<()> {
        vec![()]
    }
    fn internal(&self, _: &()) -> bool {
        false
    }
    fn step(&self, _: &State, _: ()) -> Steps<State, (), ()> {
        Steps::new()
    }
}

fn main() {}
